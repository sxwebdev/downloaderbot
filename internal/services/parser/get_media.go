package parser

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"

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
		uri, err := url.ParseRequestURI(item.Url)
		if err != nil {
			return err
		}

		fileName := filepath.Base(uri.Path)

		// check if file already exists in storage
		exists, err := s.filesService.Exists(ctx, s.config.S3.BucketName, fileName)
		if err != nil {
			return fmt.Errorf("failed to check exists file with name %s error: %w", fileName, err)
		}

		// use file from storage if it esists
		if exists {
			// get public file url
			fileUrl, err := url.JoinPath(s.config.S3BaseUrl, s.config.S3.BucketName, fileName)
			if err != nil {
				return err
			}

			item.Url = fileUrl
			continue
		}

		// download media file
		resp, err := http.Get(item.Url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// upload file to storage
		filePath, err := s.filesService.UploadStream(ctx, s.config.S3.BucketName, fileName, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to upload file with name %s error: %w", fileName, err)
		}

		// get public file url
		fileUrl, err := url.JoinPath(s.config.S3BaseUrl, s.config.S3.BucketName, filePath)
		if err != nil {
			return err
		}

		item.Url = fileUrl
	}

	return nil
}
