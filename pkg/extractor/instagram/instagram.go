package instagram

import (
	"context"
	"fmt"

	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/pkg/instagram"
)

// Extractor implements the extractor.Extractor interface for Instagram
type Extractor struct{}

// New creates a new Instagram extractor
func New() *Extractor {
	return &Extractor{}
}

// Name returns the extractor name
func (e *Extractor) Name() string {
	return string(models.MediaSourceInstagram)
}

// Hosts returns the supported hosts
func (e *Extractor) Hosts() []string {
	return []string{
		"instagram.com",
		"www.instagram.com",
	}
}

// Extract extracts media from Instagram URL
func (e *Extractor) Extract(ctx context.Context, url string) (*models.Media, error) {
	// Extract shortcode from URL
	code, err := instagram.ExtractShortcodeFromLink(url)
	if err != nil {
		return nil, fmt.Errorf("failed to extract shortcode: %w", err)
	}

	// Get media data
	media, err := instagram.GetPostWithCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	media.RequestUrl = url
	media.Source = models.MediaSourceInstagram

	return media, nil
}
