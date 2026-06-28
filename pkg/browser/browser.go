// Package browser provides a small wrapper around a shared, reusable headless
// Chromium instance (via go-rod). Loading pages through a real browser lets the
// media extractors bypass anti-bot challenges that block plain HTTP clients.
package browser

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// ErrClosed is returned by Load/Warmup after the Manager has been closed.
var ErrClosed = errors.New("browser manager is closed")

// UserAgent is the desktop Chrome UA used for both navigation and any follow-up
// download requests that must look like they came from the same browser.
const UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// navTimeout bounds a single page navigation + load.
const navTimeout = 40 * time.Second

// settleDelay is the upper bound spent waiting for client-side hydration and
// short-link redirects (e.g. vt.tiktok.com) to finish before the page HTML is
// snapshotted. With a WithReady predicate, Load returns as soon as the wanted
// markup appears and only waits this long as a fallback.
const settleDelay = 1500 * time.Millisecond

// pollInterval is how often Load re-checks the rendered HTML against a ready
// predicate while waiting out the settle window.
const pollInterval = 150 * time.Millisecond

// LoadOption configures a single Load call.
type LoadOption func(*loadConfig)

type loadConfig struct {
	ready func(html string) bool
}

// WithReady makes Load snapshot and return as soon as pred matches the rendered
// HTML, instead of always waiting out settleDelay. Callers pass a predicate that
// detects their source's embedded JSON (e.g. Instagram's "video_versions"); this
// is what trims the per-request latency once resources are blocked.
func WithReady(pred func(html string) bool) LoadOption {
	return func(c *loadConfig) { c.ready = pred }
}

// Result is the outcome of loading a page.
type Result struct {
	FinalURL string                 // URL after redirects
	HTML     string                 // full rendered HTML
	Cookies  []*proto.NetworkCookie // cookies set during the visit
}

// CookieHeader renders the visit cookies into a Cookie request header value.
func (r *Result) CookieHeader() string {
	var b []byte
	for i, c := range r.Cookies {
		if i > 0 {
			b = append(b, ';', ' ')
		}
		b = append(b, c.Name...)
		b = append(b, '=')
		b = append(b, c.Value...)
	}
	return string(b)
}

// Manager owns a lazily-launched, reused browser instance.
type Manager struct {
	binPath  string
	headless bool

	mu       sync.Mutex
	browser  *rod.Browser
	closed   bool
	inflight sync.WaitGroup // tracks active Load calls so Close can drain them
}

// NewManager creates a browser manager. The Chromium binary path can be set via
// BROWSER_BIN (recommended in containers); otherwise go-rod locates or
// downloads a browser.
func NewManager() *Manager {
	return &Manager{
		binPath:  os.Getenv("BROWSER_BIN"),
		headless: true,
	}
}

var (
	defaultOnce sync.Once
	defaultMgr  *Manager
)

// Default returns the process-wide shared Manager.
func Default() *Manager {
	defaultOnce.Do(func() { defaultMgr = NewManager() })
	return defaultMgr
}

// instance lazily launches the browser and reuses it across calls, relaunching
// if a previous instance died.
func (m *Manager) instance() (*rod.Browser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil, ErrClosed
	}

	if m.browser != nil {
		if _, err := m.browser.Version(); err == nil {
			return m.browser, nil
		}
		_ = m.browser.Close()
		m.browser = nil
	}

	l := launcher.New().
		Headless(m.headless).
		Leakless(false).              // the leakless helper is unreliable on musl/alpine
		Set("no-sandbox").            // required when running as non-root in containers
		Set("disable-dev-shm-usage"). // avoid crashes on small /dev/shm
		Set("disable-gpu").
		Set("disable-blink-features", "AutomationControlled")
	if m.binPath != "" {
		l = l.Bin(m.binPath)
	}

	controlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch browser: %w", err)
	}

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("connect browser: %w", err)
	}

	m.browser = browser
	return browser, nil
}

// Warmup launches the browser ahead of time so the first real request does not
// pay the cold-start cost (important for the time-boxed inline-query path).
func (m *Manager) Warmup() error {
	_, err := m.instance()
	return err
}

// Close marks the Manager closed, waits for in-flight Load calls to finish, then
// shuts the shared browser down. Safe to call multiple times. After Close, Load
// and Warmup return ErrClosed instead of relaunching a new browser.
func (m *Manager) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	m.mu.Unlock()

	// Drain active loads before tearing the browser down (they hold the *rod.Browser
	// pointer after releasing the lock).
	m.inflight.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.browser == nil {
		return nil
	}
	err := m.browser.Close()
	m.browser = nil
	return err
}

// Load opens url in the shared browser, waits for it to load, and returns the
// rendered HTML, the final URL (after redirects) and the visit cookies.
func (m *Manager) Load(ctx context.Context, url string, opts ...LoadOption) (*Result, error) {
	var cfg loadConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	// Register as in-flight before acquiring the browser so Close waits for us.
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil, ErrClosed
	}
	m.inflight.Add(1)
	m.mu.Unlock()
	defer m.inflight.Done()

	browser, err := m.instance()
	if err != nil {
		return nil, err
	}

	page, err := browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("open page: %w", err)
	}
	defer func() { _ = page.Close() }()

	page = page.Context(ctx).Timeout(navTimeout)

	// Block the heavy resource types the extractors never use (images, video,
	// fonts, stylesheets). Everything we need lives in the server-rendered JSON,
	// so dropping these makes the page reach a usable state far sooner.
	router := page.HijackRequests()
	if err := router.Add("*", "", blockHeavyResources); err != nil {
		return nil, fmt.Errorf("set up request hijack: %w", err)
	}
	go router.Run()
	defer func() { _ = router.Stop() }()

	if err := page.SetUserAgent(&proto.NetworkSetUserAgentOverride{UserAgent: UserAgent}); err != nil {
		return nil, fmt.Errorf("set user agent: %w", err)
	}
	if err := page.Navigate(url); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}
	// Short-link redirects (e.g. vt.tiktok.com) re-navigate the target, which can
	// make WaitLoad return a transient "navigated or closed" error. Retry once,
	// then proceed best-effort rather than failing the whole load.
	if err := page.WaitLoad(); err != nil {
		_ = page.WaitLoad()
	}

	html, err := settle(ctx, page, cfg.ready)
	if err != nil {
		return nil, err
	}

	cookies, err := page.Cookies(nil)
	if err != nil {
		return nil, fmt.Errorf("read cookies: %w", err)
	}

	res := &Result{HTML: html, Cookies: cookies}
	if info, err := page.Info(); err == nil {
		res.FinalURL = info.URL
	}
	return res, nil
}

// blockHeavyResources fails image/media/font/stylesheet requests and lets the
// rest through, so navigation does not stall on assets the extractors discard.
func blockHeavyResources(h *rod.Hijack) {
	switch h.Request.Type() {
	case proto.NetworkResourceTypeImage,
		proto.NetworkResourceTypeMedia,
		proto.NetworkResourceTypeFont,
		proto.NetworkResourceTypeStylesheet:
		h.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
	default:
		h.ContinueRequest(&proto.FetchContinueRequest{})
	}
}

// settle returns the rendered HTML once the page is ready. With a ready
// predicate it polls and returns as soon as the wanted markup appears, capping
// the wait at settleDelay; without one it simply waits out settleDelay to let
// hydration / short-link redirects finish (the legacy behavior).
func settle(ctx context.Context, page *rod.Page, ready func(string) bool) (string, error) {
	deadline := time.After(settleDelay)

	if ready != nil {
		for {
			if html, err := page.HTML(); err == nil && ready(html) {
				return html, nil
			}
			select {
			case <-deadline:
				return readHTML(page)
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(pollInterval):
			}
		}
	}

	select {
	case <-deadline:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	return readHTML(page)
}

func readHTML(page *rod.Page) (string, error) {
	html, err := page.HTML()
	if err != nil {
		return "", fmt.Errorf("read page html: %w", err)
	}
	return html, nil
}
