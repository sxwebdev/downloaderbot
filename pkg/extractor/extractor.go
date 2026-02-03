package extractor

import (
	"context"

	"github.com/sxwebdev/downloaderbot/internal/models"
)

// Extractor defines the interface that all media extractors must implement
type Extractor interface {
	// Name returns the unique name of the extractor
	Name() string

	// Hosts returns the list of hosts that this extractor supports
	Hosts() []string

	// Extract extracts media from the given URL
	Extract(ctx context.Context, url string) (*models.Media, error)
}
