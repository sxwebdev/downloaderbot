package parser

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/sxwebdev/downloaderbot/internal/metrics"
	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/internal/util"
	"github.com/sxwebdev/downloaderbot/pkg/extractor"
	// Import extractor packages to register them
	_ "github.com/sxwebdev/downloaderbot/pkg/extractor/instagram"
	_ "github.com/sxwebdev/downloaderbot/pkg/extractor/lux"
	_ "github.com/sxwebdev/downloaderbot/pkg/extractor/youtube"
)

type GetLinkInfoResponse struct {
	RequestLink string
	MediaSource models.MediaSource
	Url         *url.URL
	Extractor   extractor.Extractor
}

func (s *Service) GetLinkInfo(ctx context.Context, link string) (GetLinkInfoResponse, error) {
	if err := ctx.Err(); err != nil {
		return GetLinkInfoResponse{}, err
	}

	if !util.IsValidUrl(link) {
		return GetLinkInfoResponse{}, fmt.Errorf("received invalid link")
	}

	// Convert link to URL object
	uri, err := url.ParseRequestURI(link)
	if err != nil {
		return GetLinkInfoResponse{}, fmt.Errorf("failed to parse link %s: %w", link, err)
	}

	// Get extractor from registry
	registry := extractor.GetRegistry()
	ext, err := registry.GetByURL(link)
	if err != nil {
		return GetLinkInfoResponse{}, fmt.Errorf("unsupported source: %w", err)
	}

	return GetLinkInfoResponse{
		RequestLink: link,
		MediaSource: models.MediaSource(ext.Name()),
		Url:         uri,
		Extractor:   ext,
	}, nil
}

func (s *Service) GetMedia(ctx context.Context, linkInfo GetLinkInfoResponse) (*models.Media, error) {
	if linkInfo.Extractor == nil {
		return nil, fmt.Errorf("no extractor available for this source")
	}

	source := string(linkInfo.MediaSource)
	start := time.Now()
	media, err := linkInfo.Extractor.Extract(ctx, linkInfo.RequestLink)
	metrics.ExtractDuration.WithLabelValues(source).Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ExtractErrors.WithLabelValues(source, classifyExtractError(err)).Inc()
		return nil, fmt.Errorf("failed to get media from source: %w", err)
	}

	return media, nil
}

func classifyExtractError(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "other"
	}
}
