package parser

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/internal/util"
	"github.com/sxwebdev/downloaderbot/pkg/instagram"
)

func (s *Service) GetMedia(ctx context.Context, link string) (*models.Media, error) {
	if !util.IsValidUrl(link) {
		return nil, fmt.Errorf("received invalid link")
	}

	// Convert link to URL object
	uri, err := url.ParseRequestURI(link)
	if err != nil {
		return nil, fmt.Errorf("failed to parse link %s: %w", link, err)
	}

	var mediaSource models.MediaSource
	switch uri.Host {
	case "instagram.com", "www.instagram.com":
		mediaSource = models.MediaSourceInstagram
	case "youtube.com", "www.youtube.com", "m.youtube.com":
		mediaSource = models.MediaSourceInstagram
	default:
		return nil, fmt.Errorf("can only process links from instagram and youtube not [%s]", uri.Host)
	}

	var media *models.Media
	switch mediaSource {
	case models.MediaSourceInstagram:
		media, err = s.processInstagram(ctx, uri.Path)
	case models.MediaSourceYoutube:
		media, err = s.processYoutube(ctx, uri.Path)
	default:
		return nil, fmt.Errorf("unsupported media source %s", string(mediaSource))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get media from source: %w", err)
	}

	if err := s.getMediaData(ctx, media); err != nil {
		return nil, fmt.Errorf("failed to get media data: %w", err)
	}

	return media, nil
}

func (s *Service) processInstagram(ctx context.Context, path string) (*models.Media, error) {
	// extract media code from url
	code, err := instagram.ExtractShortcodeFromLink(path)
	if err != nil {
		return nil, err
	}

	// get media data from instagram
	data, err := instagram.GetPostWithCode(ctx, code)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *Service) processYoutube(ctx context.Context, path string) (*models.Media, error) {
	return nil, fmt.Errorf("source is %s not supported yet", string(models.MediaSourceYoutube))
}

func (s *Service) getMediaData(ctx context.Context, media *models.Media) error {
	if len(media.Items) == 0 {
		return nil
	}

	for _, item := range media.Items {
		resp, err := http.Get(item.Url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		item.Data = data
	}

	return nil
}
