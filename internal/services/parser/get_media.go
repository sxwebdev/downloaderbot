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
	"time"

	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/internal/util"
	"github.com/sxwebdev/downloaderbot/pkg/extractor"
	// Import extractor packages to register them
	_ "github.com/sxwebdev/downloaderbot/pkg/extractor/instagram"
	_ "github.com/sxwebdev/downloaderbot/pkg/extractor/lux"
	_ "github.com/sxwebdev/downloaderbot/pkg/extractor/youtube"
	"golang.org/x/sync/errgroup"
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

func (s *Service) saveMediaData(ctx context.Context, media *models.Media) error {
	if len(media.Items) == 0 {
		return nil
	}

	eg, egCtx := errgroup.WithContext(ctx)

	for _, item := range media.Items {
		eg.Go(func() error {
			return s.saveMediaItem(egCtx, item)
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func (s *Service) saveMediaItem(ctx context.Context, item *models.MediaItem) error {
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
		return nil
	}

	// download media file
	resp, err := http.Get(item.Url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// upload file to storage
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

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

	return nil
}
