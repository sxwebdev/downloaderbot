package instagram

import (
	"context"
	"fmt"

	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/pkg/instagram"
)

// Extractor implements the extractor.Extractor interface for Instagram
type Extractor struct {
	fetcher instagram.Fetcher
}

// New creates a new Instagram extractor. It uses the browser-based fetcher by
// default, which drives a real headless Chromium and so bypasses Instagram's
// anti-bot challenges that break the legacy HTTP fetcher. The legacy fetcher
// remains available via instagram.NewAPIFetcher().
func New() *Extractor {
	return &Extractor{
		fetcher: instagram.NewBrowserFetcher(),
	}
}

// Name returns the extractor name
func (e *Extractor) Name() string {
	return string(models.MediaSourceInstagram)
}

// Hosts returns the supported hosts.
// `www.` / `m.` / `mobile.` prefixes are normalized by the registry, so we only
// list the canonical form here.
func (e *Extractor) Hosts() []string {
	return []string{"instagram.com"}
}

// Extract extracts media from Instagram URL
func (e *Extractor) Extract(ctx context.Context, url string) (*models.Media, error) {
	// Extract shortcode from URL
	code, err := instagram.ExtractShortcodeFromLink(url)
	if err != nil {
		return nil, fmt.Errorf("failed to extract shortcode: %w", err)
	}

	// Get media data
	media, err := e.fetcher.GetPost(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	media.RequestUrl = url
	media.Source = models.MediaSourceInstagram

	return media, nil
}
