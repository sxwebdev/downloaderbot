package youtube

import (
	"context"
	"fmt"

	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/pkg/youtube"
)

// Extractor implements the extractor.Extractor interface for YouTube
type Extractor struct{}

// New creates a new YouTube extractor
func New() *Extractor {
	return &Extractor{}
}

// Name returns the extractor name
func (e *Extractor) Name() string {
	return string(models.MediaSourceYoutube)
}

// Hosts returns the supported hosts
func (e *Extractor) Hosts() []string {
	return []string{
		"youtube.com",
		"www.youtube.com",
		"m.youtube.com",
		"youtu.be",
	}
}

// Extract extracts media from YouTube URL
func (e *Extractor) Extract(ctx context.Context, url string) (*models.Media, error) {
	// Extract video ID from URL
	videoID, err := youtube.ExtractShortcodeFromLink(url)
	if err != nil {
		return nil, fmt.Errorf("failed to extract video ID: %w", err)
	}

	// Get video data
	media, err := youtube.GetVideoByID(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get video: %w", err)
	}

	media.RequestUrl = url
	media.Source = models.MediaSourceYoutube

	return media, nil
}
