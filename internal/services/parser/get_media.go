package parser

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/internal/util"
	"github.com/sxwebdev/downloaderbot/pkg/instagram"
	"github.com/sxwebdev/downloaderbot/pkg/youtube"
)

type GetLinkInfoResponse struct {
	RequestLink string
	MediaSource models.MediaSource
	Url         *url.URL
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

	var mediaSource models.MediaSource
	switch uri.Host {
	case "instagram.com", "www.instagram.com":
		mediaSource = models.MediaSourceInstagram
	case "youtube.com", "www.youtube.com", "m.youtube.com":
		mediaSource = models.MediaSourceYoutube
	default:
		return GetLinkInfoResponse{}, fmt.Errorf("can only process links from instagram and youtube not [%s]", uri.Host)
	}

	return GetLinkInfoResponse{
		RequestLink: link,
		MediaSource: mediaSource,
		Url:         uri,
	}, nil
}

func (s *Service) GetMedia(ctx context.Context, linkInfo GetLinkInfoResponse) (*models.Media, error) {
	var media *models.Media
	var err error
	switch linkInfo.MediaSource {
	case models.MediaSourceInstagram:
		media, err = s.processInstagram(ctx, linkInfo.Url.Path)
	case models.MediaSourceYoutube:
		media, err = s.processYoutube(ctx, linkInfo.RequestLink)
	default:
		return nil, fmt.Errorf("unsupported media source %s", string(linkInfo.MediaSource))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get media from source: %w", err)
	}

	media.RequestUrl = linkInfo.RequestLink
	media.Source = linkInfo.MediaSource

	// save data for instagram source
	if linkInfo.MediaSource == models.MediaSourceInstagram {
		if err := s.saveMediaData(ctx, media); err != nil {
			return nil, fmt.Errorf("failed to get media data: %w", err)
		}
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
	// extract media code from url
	code, err := youtube.ExtractShortcodeFromLink(path)
	if err != nil {
		return nil, err
	}

	data, err := youtube.GetVideoByID(ctx, code)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *Service) saveMediaData(ctx context.Context, media *models.Media) error {
	if len(media.Items) == 0 {
		return nil
	}

	for _, item := range media.Items {
		uri, err := url.ParseRequestURI(item.Url)
		if err != nil {
			return err
		}

		ext := filepath.Ext(uri.Path)
		fileNameWithoutExt := strings.TrimSuffix(filepath.Base(uri.Path), ext)

		h := md5.New()
		if _, err := io.WriteString(h, fileNameWithoutExt); err != nil {
			return err
		}

		fileName := fmt.Sprintf("%x", h.Sum(nil)) + ext

		// check if file already exists in storage
		exists, err := s.filesService.Exists(ctx, s.config.S3.BucketName, fileName)
		if err != nil {
			return fmt.Errorf("failed to check exists file with name %s error: %w", fileName, err)
		}

		// use file from storage if it esists
		if exists {
			// get public file url
			fileUrl, err := url.JoinPath(s.config.S3BaseUrl, fileName)
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
		fileUrl, err := url.JoinPath(s.config.S3BaseUrl, filePath)
		if err != nil {
			return err
		}

		item.Url = fileUrl
	}

	return nil
}
