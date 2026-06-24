package tiktok

import (
	"context"
	"fmt"

	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/pkg/tiktok"
)

// Extractor implements the extractor.Extractor interface for TikTok using a
// real headless browser (pkg/tiktok), which bypasses TikTok's anti-bot where
// the lux-based extractor fails.
type Extractor struct{}

// New creates a new TikTok extractor.
func New() *Extractor {
	return &Extractor{}
}

// Name returns the extractor name.
func (e *Extractor) Name() string {
	return string(models.MediaSourceTikTok)
}

// Hosts returns the supported hosts, including the short-link domains
// (vt./vm.) which are not normalized away by the registry.
func (e *Extractor) Hosts() []string {
	return []string{"tiktok.com", "vt.tiktok.com", "vm.tiktok.com"}
}

// Extract extracts media from a TikTok URL.
func (e *Extractor) Extract(ctx context.Context, url string) (*models.Media, error) {
	media, err := tiktok.GetVideo(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to get tiktok video: %w", err)
	}

	media.RequestUrl = url
	media.Source = models.MediaSourceTikTok

	return media, nil
}
