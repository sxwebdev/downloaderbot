package instagram

import (
	"context"

	"github.com/sxwebdev/downloaderbot/internal/models"
)

// Fetcher retrieves an Instagram post by its shortcode. Different
// implementations use different transports (anonymous HTTP API, headless
// browser, ...).
type Fetcher interface {
	GetPost(ctx context.Context, code string) (*models.Media, error)
}

// APIFetcher is the legacy implementation that talks to Instagram's GraphQL /
// embed endpoints over plain HTTP. It is kept behind the Fetcher interface as a
// fallback; note that Instagram increasingly serves anti-bot challenges
// (error 1357054) to these anonymous requests, especially from datacenter IPs.
type APIFetcher struct{}

// NewAPIFetcher creates the legacy HTTP-based fetcher.
func NewAPIFetcher() *APIFetcher { return &APIFetcher{} }

// GetPost implements Fetcher using the GraphQL/embed HTTP path.
func (f *APIFetcher) GetPost(ctx context.Context, code string) (*models.Media, error) {
	return GetPostWithCode(ctx, code)
}
