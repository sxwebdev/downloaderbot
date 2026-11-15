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

// maxLoadsPerBrowser is how many page loads one Chromium process serves before
// it is retired and replaced. A long-lived Chromium only grows: its browser
// process, renderer processes and on-disk profile (HTTP + code cache — page
// cache that counts against the container's memory limit) all accumulate. In
// production a 45-day-old instance held ~1.1GB, with a 2GB profile directory.
// Relaunching costs well under a second and wipes all of it.
const maxLoadsPerBrowser = 100

// closeTimeout bounds tab/browser teardown calls, which run on a context that
// is independent of the (possibly already cancelled) request context.
const closeTimeout = 5 * time.Second

// instance is one launched Chromium process.
type instance struct {
	browser  *rod.Browser
	launcher *launcher.Launcher
	loads    int // page loads started on this instance
	active   int // page loads currently running on this instance
}

// shutdown closes the browser, kills the process tree and removes the profile
// directory. It may block for a second or more (rod's Kill waits for child
// processes to settle).
func (in *instance) shutdown() error {
	err := in.browser.Timeout(closeTimeout).Close()
	if in.launcher != nil {
		in.launcher.Kill()
		in.launcher.Cleanup()
	}
	return err
}

// Manager owns a lazily-launched, reused browser instance, replaced after
// maxLoadsPerBrowser loads.
type Manager struct {
	binPath  string
	headless bool
	maxLoads int

	mu       sync.Mutex
	current  *instance
	closed   bool
	inflight sync.WaitGroup // tracks active Load calls and retirements so Close can drain them
}

// NewManager creates a browser manager. The Chromium binary path can be set via
// BROWSER_BIN (recommended in containers); otherwise go-rod locates or
// downloads a browser.
func NewManager() *Manager {
	return &Manager{
		binPath:  os.Getenv("BROWSER_BIN"),
		headless: true,
		maxLoads: maxLoadsPerBrowser,
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

// currentLocked returns a live instance, launching one when there is none, the
// previous one died, or it has served its quota. m.mu must be held.
func (m *Manager) currentLocked() (*instance, error) {
	if m.closed {
		return nil, ErrClosed
	}

	if in := m.current; in != nil {
		_, err := in.browser.Version()
		if err == nil && in.loads < m.maxLoads {
			return in, nil
		}
		// Dead or used up: detach it. Loads still running on it keep their
		// reference and the last one to finish shuts it down.
		m.current = nil
		if in.active == 0 {
			m.retire(in)
		}
	}

	in, err := m.launch()
	if err != nil {
		return nil, err
	}
	m.current = in
	return in, nil
}

func (m *Manager) launch() (*instance, error) {
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
		l.Cleanup()
		return nil, fmt.Errorf("launch browser: %w", err)
	}

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		l.Kill()
		l.Cleanup()
		return nil, fmt.Errorf("connect browser: %w", err)
	}

	return &instance{browser: browser, launcher: l}, nil
}

// retire shuts a detached instance down in the background. m.mu must be held;
// Close waits for retirements through inflight.
func (m *Manager) retire(in *instance) {
	m.inflight.Go(func() {
		_ = in.shutdown()
	})
}

// acquire reserves a slot on the current instance for one page load.
func (m *Manager) acquire() (*instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	in, err := m.currentLocked()
	if err != nil {
		return nil, err
	}
	in.loads++
	in.active++
	return in, nil
}

// release returns a slot taken by acquire. The last load on a detached
// instance shuts it down; when the current instance has just used up its quota
// it is detached and a replacement is launched in the background, so the next
// request does not pay the cold start.
func (m *Manager) release(in *instance) {
	m.mu.Lock()
	defer m.mu.Unlock()

	in.active--
	if in.active > 0 {
		return
	}
	if in != m.current {
		m.retire(in)
		return
	}
	if in.loads >= m.maxLoads && !m.closed {
		m.current = nil
		m.retire(in)
		m.inflight.Go(func() {
			_ = m.Warmup()
		})
	}
}

// Warmup launches the browser ahead of time so the first real request does not
// pay the cold-start cost (important for the time-boxed inline-query path).
func (m *Manager) Warmup() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, err := m.currentLocked()
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

	// Drain active loads and background retirements before tearing the browser
	// down.
	m.inflight.Wait()

	m.mu.Lock()
	in := m.current
	m.current = nil
	m.mu.Unlock()
	if in == nil {
		return nil
	}
	return in.shutdown()
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

	in, err := m.acquire()
	if err != nil {
		return nil, err
	}
	defer m.release(in)

	tab, err := in.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("open page: %w", err)
	}
	// Close through the tab's own (browser-scoped) context, not the request
	// one: when the request times out or is cancelled — routine for inline
	// queries with their 10s budget — a close bound to that context is never
	// sent, and the tab stays open in Chromium forever, rendering Instagram's
	// JS. That is how production accumulated a 488MB renderer.
	defer func() { _ = tab.Timeout(closeTimeout).Close() }()

	navCtx, cancel := context.WithTimeout(ctx, navTimeout)
	defer cancel()
	page := tab.Context(navCtx)

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

	html, err := settle(navCtx, page, cfg.ready)
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
// predicate it polls from the moment navigation starts and returns as soon as
// the wanted markup appears — the media JSON is in the server-rendered document,
// so that is usually well before the load event, which is why we do not block on
// WaitLoad first. WaitLoad runs concurrently only to bound the case where the
// markup never appears (an error/login/interstitial page): once the page has
// finished loading we allow a short settleDelay grace for late hydration or a
// short-link redirect, then snapshot best-effort. Without a predicate it simply
// waits out settleDelay after the load event (the legacy behavior).
func settle(ctx context.Context, page *rod.Page, ready func(string) bool) (string, error) {
	if ready == nil {
		// Short-link redirects (e.g. vt.tiktok.com) re-navigate the target, which
		// can make WaitLoad return a transient "navigated or closed" error. Retry
		// once, then proceed best-effort rather than failing the whole load.
		if err := page.WaitLoad(); err != nil {
			_ = page.WaitLoad()
		}
		select {
		case <-time.After(settleDelay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
		return readHTML(page)
	}

	loaded := make(chan struct{})
	go func() {
		_ = page.WaitLoad()
		close(loaded)
	}()

	var grace <-chan time.Time
	for {
		if html, err := page.HTML(); err == nil && ready(html) {
			return html, nil
		}
		select {
		case <-loaded:
			// Page finished loading without the markup yet; give late hydration /
			// redirects a brief window, then fall back to a best-effort snapshot.
			loaded = nil
			grace = time.After(settleDelay)
		case <-grace:
			return readHTML(page)
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

func readHTML(page *rod.Page) (string, error) {
	html, err := page.HTML()
	if err != nil {
		return "", fmt.Errorf("read page html: %w", err)
	}
	return html, nil
}
