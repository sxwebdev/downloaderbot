// Package media provides a single entry point for obtaining media content.
// All consumers — Telegram direct messages, Telegram inline queries and the
// gRPC API — go through Loader so that source-specific download requirements
// (e.g. TikTok's mandatory cookies + referer headers) live in one place.
package media

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/internal/util"
)

// Content is a readable media stream with metadata.
type Content struct {
	Body          io.ReadCloser
	ContentLength int64
}

// Loader is the unified media-loading interface.
type Loader interface {
	// DirectURL reports a URL that any HTTP client can fetch without special
	// headers — suitable to hand to Telegram inline results or API clients.
	// ok is false when the item can only be obtained by downloading it via Open
	// (e.g. TikTok, whose CDN requires cookies + referer).
	DirectURL(item *models.MediaItem) (url string, ok bool)

	// Open streams the item's content, applying any required download headers.
	// The caller must close Content.Body.
	Open(ctx context.Context, item *models.MediaItem) (*Content, error)
}

type httpLoader struct {
	client *http.Client
}

// NewHTTPLoader creates the default HTTP-based loader.
func NewHTTPLoader() Loader {
	return &httpLoader{client: util.DefaultHttpClient()}
}

var defaultLoader = NewHTTPLoader()

// Default returns the process-wide loader.
func Default() Loader { return defaultLoader }

func (l *httpLoader) DirectURL(item *models.MediaItem) (string, bool) {
	if item == nil || item.Url == "" {
		return "", false
	}
	// Items that need custom headers cannot be fetched by a third party
	// (Telegram, API clients) — they must be downloaded via Open.
	if len(item.DownloadHeaders) > 0 {
		return "", false
	}
	return item.Url, true
}

func (l *httpLoader) Open(ctx context.Context, item *models.MediaItem) (*Content, error) {
	if item == nil || item.Url == "" {
		return nil, fmt.Errorf("empty url")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.Url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range item.DownloadHeaders {
		req.Header.Set(k, v)
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		resp.Body.Close()
		return nil, fmt.Errorf("source returned %s", resp.Status)
	}

	return &Content{Body: resp.Body, ContentLength: resp.ContentLength}, nil
}
