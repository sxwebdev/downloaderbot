package parser

import (
	"context"
	"fmt"
	"net/url"

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

func (s *Service) GetLinkInfo(link string) (GetLinkInfoResponse, error) {
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

	media, err := linkInfo.Extractor.Extract(ctx, linkInfo.RequestLink)
	if err != nil {
		return nil, fmt.Errorf("failed to get media from source: %w", err)
	}

	return media, nil
}
