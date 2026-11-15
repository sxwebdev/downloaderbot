package instagram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sxwebdev/downloaderbot/internal/metrics"
	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/internal/util"
)

// httpAttemptTimeout bounds the plain-HTTP attempt so a stalled Instagram
// response still leaves the browser fallback time inside the 10s inline budget.
// From the production pod the page arrives in ~0.6s (p50), ~0.85s (p90).
const httpAttemptTimeout = 5 * time.Second

// maxPageSize caps how much of a post page is read. Real pages are 0.5=1MB.
const maxPageSize = 8 << 20

// navigationHeaders are what Chrome sends for a top-level navigation. Instagram
// serves the logged-out post page with the embedded media JSON only to requests
// that look like a browser navigation — the full browser Accept value is the
// part it actually gates on (a bare "text/html" gets a page without the media)
// — so the whole set is sent to stay close to a real browser.
var navigationHeaders = map[string]string{
	"User-Agent":                igUserAgent,
	"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
	"Accept-Language":           "en-US,en;q=0.9",
	"Sec-Fetch-Dest":            "document",
	"Sec-Fetch-Mode":            "navigate",
	"Sec-Fetch-Site":            "none",
	"Sec-Fetch-User":            "?1",
	"Upgrade-Insecure-Requests": "1",
}

// HTTPFetcher retrieves a post page with a single plain HTTP request that
// mimics a browser navigation and parses the same embedded media JSON as
// BrowserFetcher. It is roughly 6x faster than rendering the page in Chromium
// on the production pod (which is CPU-limited) and uses no browser memory, but
// it has no way past a challenge or login wall — pair it with BrowserFetcher
// through NewFallbackFetcher.
type HTTPFetcher struct {
	client  *http.Client
	baseURL string
}

// NewHTTPFetcher creates a plain-HTTP fetcher on the shared transport.
func NewHTTPFetcher() *HTTPFetcher {
	return &HTTPFetcher{baseURL: igBaseURL, client: &http.Client{
		Transport: util.DefaultTransport(),
		Timeout:   httpAttemptTimeout,
		// A redirect means Instagram is sending us to the login page or a
		// challenge instead of the post; following it would only download a page
		// without media.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// GetPost implements Fetcher.
func (f *HTTPFetcher) GetPost(ctx context.Context, code string) (*models.Media, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.baseURL+"/p/"+code+"/", nil)
	if err != nil {
		return nil, err
	}
	for k, v := range navigationHeaders {
		req.Header.Set(k, v)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("instagram returned %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPageSize))
	if err != nil {
		return nil, fmt.Errorf("read post page: %w", err)
	}

	return parseMediaFromHTML(string(body), code)
}

// FallbackFetcher tries the fast plain-HTTP fetcher first and loads the page in
// the headless browser only when that attempt fails.
type FallbackFetcher struct {
	primary  Fetcher
	fallback Fetcher
}

// NewFallbackFetcher returns a Fetcher that uses primary and falls back to
// fallback on any primary error.
func NewFallbackFetcher(primary, fallback Fetcher) *FallbackFetcher {
	return &FallbackFetcher{primary: primary, fallback: fallback}
}

// GetPost implements Fetcher.
func (f *FallbackFetcher) GetPost(ctx context.Context, code string) (*models.Media, error) {
	media, err := f.primary.GetPost(ctx, code)
	metrics.ObserveInstagramFetch(metrics.InstagramMethodHTTP, err)
	if err == nil {
		return media, nil
	}
	// The caller's budget is gone; the browser would fail the same way.
	if ctx.Err() != nil {
		return nil, err
	}

	media, fbErr := f.fallback.GetPost(ctx, code)
	metrics.ObserveInstagramFetch(metrics.InstagramMethodBrowser, fbErr)
	if fbErr != nil {
		// Wrap the browser error: it is the final word, and its timeout/cancel
		// cause is what the extraction metrics classify.
		return nil, fmt.Errorf("%w (http attempt: %v)", fbErr, err)
	}
	return media, nil
}
