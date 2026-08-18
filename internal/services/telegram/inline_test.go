package telegram

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/sxwebdev/downloaderbot/internal/media"
	"github.com/sxwebdev/downloaderbot/internal/models"
	"gopkg.in/telebot.v3"
)

// fakeLoader is a media.Loader that reports a fixed size without touching the
// network, so the inline size decision can be exercised deterministically.
type fakeLoader struct {
	size    int64
	sizeErr error
	payload string
	openErr error
	// headCalls counts ContentLength lookups so tests can assert the size is
	// probed only for the item types that need it.
	headCalls atomic.Int64
}

func (f *fakeLoader) DirectURL(item *models.MediaItem) (string, bool) {
	if item == nil || item.Url == "" || len(item.DownloadHeaders) > 0 {
		return "", false
	}
	return item.Url, true
}

func (f *fakeLoader) Open(context.Context, *models.MediaItem) (*media.Content, error) {
	if f.openErr != nil {
		return nil, f.openErr
	}
	return &media.Content{Body: io.NopCloser(strings.NewReader(f.payload)), ContentLength: f.size}, nil
}

func (f *fakeLoader) ContentLength(context.Context, *models.MediaItem) (int64, error) {
	f.headCalls.Add(1)
	if f.sizeErr != nil {
		return 0, f.sizeErr
	}
	return f.size, nil
}

// TestInlineResultFor_URLSizeLimit guards the reported bug: an Instagram reel
// that the bot delivers fine in a direct message never appears at all in inline
// mode. Telegram fetches an inline result from the URL itself and caps that at
// 20MB — not the 50MB that applies to files the bot uploads — so a 22MB reel
// offered as a VideoResult is one Telegram silently fails to fetch.
func TestInlineResultFor_URLSizeLimit(t *testing.T) {
	const (
		mb       = 1024 * 1024
		videoURL = "https://cdn.example/reel.mp4"
	)

	tests := []struct {
		name     string
		size     int64
		sizeErr  error
		wantLink bool // true: a download-link article instead of a playable video
	}{
		{name: "well under the limit", size: 5 * mb, wantLink: false},
		{name: "exactly at the limit", size: 20 * mb, wantLink: false},
		{name: "one byte over the limit", size: 20*mb + 1, wantLink: true},
		{
			// The real reel DbTK_BOssyb: 23,486,848 bytes. Under the 50MB upload cap
			// (so direct messages work) but over the 20MB URL cap, which is exactly
			// why it worked in chat and vanished inline.
			name: "the reported reel at 22.4MB", size: 23_486_848, wantLink: true,
		},
		{name: "over the upload cap too", size: 60 * mb, wantLink: true},
		{
			// The CDN refused a HEAD, so the size is unknown. Offering the video is
			// the better guess — assuming "too large" would downgrade every result
			// from a source that does not answer HEAD.
			name: "unknown size still offers the video", sizeErr: errors.New("no HEAD"), wantLink: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			loader := &fakeLoader{size: tc.size, sizeErr: tc.sizeErr}
			h := &handler{loader: loader}
			item := &models.MediaItem{
				Type:         models.MediaTypeVideo,
				Url:          videoURL,
				Width:        1080,
				Height:       1920,
				Duration:     120,
				ThumbnailUrl: "https://cdn.example/cover.jpg",
			}

			result, ok := h.inlineResultFor(t.Context(), item, 0, "a caption")
			if !ok {
				t.Fatal("inlineResultFor reported the item as un-offerable")
			}
			if got := loader.headCalls.Load(); got != 1 {
				t.Fatalf("ContentLength called %d times, want exactly 1", got)
			}

			article, isArticle := result.(*telebot.ArticleResult)
			if tc.wantLink {
				if !isArticle {
					t.Fatalf("result type = %T, want a download-link ArticleResult", result)
				}
				content, isText := article.Content.(*telebot.InputTextMessageContent)
				if !isText {
					t.Fatalf("article content type = %T, want InputTextMessageContent", article.Content)
				}
				if !strings.Contains(content.Text, videoURL) {
					t.Fatalf("download link text %q does not carry the source URL", content.Text)
				}
				// The wording must quote the limit that actually applied, not the
				// 50MB upload cap that does not bound inline results.
				if !strings.Contains(content.Text, "20MB") || strings.Contains(content.Text, "50") {
					t.Fatalf("download link text %q should cite the 20MB URL limit", content.Text)
				}
				return
			}

			if isArticle {
				t.Fatalf("got a download-link article for a %d-byte video, want a playable VideoResult", tc.size)
			}
			video, isVideo := result.(*telebot.VideoResult)
			if !isVideo {
				t.Fatalf("result type = %T, want *telebot.VideoResult", result)
			}
			if video.URL != videoURL {
				t.Fatalf("VideoResult.URL = %q, want %q", video.URL, videoURL)
			}
		})
	}
}

// TestInlineResultFor_VideoMetadata locks in the fields Telegram needs to render
// an inline video without downloading it first: a duration, a JPEG thumbnail and
// the real dimensions.
func TestInlineResultFor_VideoMetadata(t *testing.T) {
	item := &models.MediaItem{
		Type:         models.MediaTypeVideo,
		Url:          "https://cdn.example/reel.mp4",
		Width:        1080,
		Height:       1920,
		Duration:     120,
		ThumbnailUrl: "https://cdn.example/cover.jpg",
	}

	h := &handler{loader: &fakeLoader{size: 5 * 1024 * 1024}}
	result, ok := h.inlineResultFor(t.Context(), item, 2, "a caption")
	if !ok {
		t.Fatal("inlineResultFor reported the item as un-offerable")
	}

	video, isVideo := result.(*telebot.VideoResult)
	if !isVideo {
		t.Fatalf("result type = %T, want *telebot.VideoResult", result)
	}

	if video.Duration != 120 {
		t.Errorf("Duration = %d, want 120 — without it Telegram shows no length", video.Duration)
	}
	// thumbnail_url is documented "JPEG only"; handing Telegram the .mp4 gives
	// the result no preview at all.
	if video.ThumbURL != item.ThumbnailUrl {
		t.Errorf("ThumbURL = %q, want the JPEG cover %q", video.ThumbURL, item.ThumbnailUrl)
	}
	if strings.HasSuffix(video.ThumbURL, ".mp4") {
		t.Errorf("ThumbURL = %q is a video, but thumbnail_url must be a JPEG", video.ThumbURL)
	}
	if video.Width != 1080 || video.Height != 1920 {
		t.Errorf("dimensions = %dx%d, want 1080x1920", video.Width, video.Height)
	}
	if video.MIME != "video/mp4" {
		t.Errorf("MIME = %q, want video/mp4", video.MIME)
	}
	if video.Title != "video-3" {
		t.Errorf("Title = %q, want video-3 (1-based index)", video.Title)
	}
	if video.Description != "a caption" {
		t.Errorf("Description = %q, want the caption", video.Description)
	}
}

// TestInlineResultFor_ThumbnailFallback covers a source that exposes no cover
// frame: the media URL is a poor thumbnail, but dropping the result entirely
// would be worse.
func TestInlineResultFor_ThumbnailFallback(t *testing.T) {
	item := &models.MediaItem{
		Type: models.MediaTypeVideo,
		Url:  "https://cdn.example/reel.mp4",
	}

	h := &handler{loader: &fakeLoader{size: 1024}}
	result, ok := h.inlineResultFor(t.Context(), item, 0, "")
	if !ok {
		t.Fatal("inlineResultFor reported the item as un-offerable")
	}

	video, isVideo := result.(*telebot.VideoResult)
	if !isVideo {
		t.Fatalf("result type = %T, want *telebot.VideoResult", result)
	}
	if video.ThumbURL != item.Url {
		t.Fatalf("ThumbURL = %q, want the media URL as the last-resort fallback", video.ThumbURL)
	}
	if video.Duration != 0 {
		t.Fatalf("Duration = %d, want 0 when unknown", video.Duration)
	}
}

// TestInlineResultFor_NonVideo covers the item kinds that do not take the video
// path at all.
func TestInlineResultFor_NonVideo(t *testing.T) {
	t.Run("photo needs no size probe", func(t *testing.T) {
		loader := &fakeLoader{size: 60 * 1024 * 1024}
		h := &handler{loader: loader}

		result, ok := h.inlineResultFor(t.Context(),
			&models.MediaItem{Type: models.MediaTypePhoto, Url: "https://cdn.example/p.jpg"}, 0, "")
		if !ok {
			t.Fatal("inlineResultFor reported the photo as un-offerable")
		}
		photo, isPhoto := result.(*telebot.PhotoResult)
		if !isPhoto {
			t.Fatalf("result type = %T, want *telebot.PhotoResult", result)
		}
		if photo.URL != "https://cdn.example/p.jpg" || photo.ThumbURL != photo.URL {
			t.Fatalf("photo URLs = %q / %q, want both to be the image", photo.URL, photo.ThumbURL)
		}
		if got := loader.headCalls.Load(); got != 0 {
			t.Fatalf("ContentLength called %d times for a photo, want 0", got)
		}
	})

	t.Run("audio is not offerable", func(t *testing.T) {
		h := &handler{loader: &fakeLoader{}}
		if _, ok := h.inlineResultFor(t.Context(),
			&models.MediaItem{Type: models.MediaTypeAudio, Url: "https://cdn.example/a.m4a"}, 0, ""); ok {
			t.Fatal("audio was offered inline, want it skipped")
		}
	})

	t.Run("items needing download headers are not offerable", func(t *testing.T) {
		// TikTok: the CDN only serves the bytes with cookies + referer, which
		// Telegram cannot send on the bot's behalf.
		h := &handler{loader: &fakeLoader{}}
		if _, ok := h.inlineResultFor(t.Context(), &models.MediaItem{
			Type:            models.MediaTypeVideo,
			Url:             "https://cdn.tiktok/v.mp4",
			DownloadHeaders: map[string]string{"Referer": "https://tiktok.com"},
		}, 0, ""); ok {
			t.Fatal("a header-gated item was offered inline, want it skipped")
		}
	})

	t.Run("empty url is not offerable", func(t *testing.T) {
		h := &handler{loader: &fakeLoader{}}
		if _, ok := h.inlineResultFor(t.Context(), &models.MediaItem{Type: models.MediaTypeVideo}, 0, ""); ok {
			t.Fatal("an item with no URL was offered inline, want it skipped")
		}
	})
}

// TestTooLargeText checks the message quotes whichever limit actually applied —
// the chat path is bounded by the 50MB upload cap, inline by the 20MB URL cap.
func TestTooLargeText(t *testing.T) {
	const src = "https://cdn.example/reel.mp4"

	tests := []struct {
		name  string
		limit int64
		want  string
	}{
		{name: "upload cap", limit: maxFileSize, want: "50MB"},
		{name: "url cap", limit: maxURLFileSize, want: "20MB"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tooLargeText(src, tc.limit)
			if !strings.Contains(got, tc.want) {
				t.Errorf("tooLargeText = %q, want it to mention %s", got, tc.want)
			}
			if !strings.Contains(got, fmt.Sprintf("(%s)", src)) {
				t.Errorf("tooLargeText = %q, want a markdown link to the source", got)
			}
		})
	}

	t.Run("result title matches its limit", func(t *testing.T) {
		t.Parallel()

		result := tooLargeResult(src, maxURLFileSize)
		article, ok := result.(*telebot.ArticleResult)
		if !ok {
			t.Fatalf("tooLargeResult type = %T, want *telebot.ArticleResult", result)
		}
		if !strings.Contains(article.Title, "20MB") {
			t.Errorf("title = %q, want it to cite the 20MB URL limit", article.Title)
		}
	})
}

// TestMaxURLFileSize pins the constants to the values the Telegram Bot API
// documents, so a future edit cannot quietly reintroduce the 50MB assumption on
// the inline path: "5 MB max size for photos and 20 MB max for other types of
// content" by URL, versus "10 MB max size for photos, 50 MB for other files"
// when the bot uploads them itself.
func TestMaxURLFileSize(t *testing.T) {
	if maxURLFileSize != 20*1024*1024 {
		t.Errorf("maxURLFileSize = %d, want 20MB", maxURLFileSize)
	}
	if maxFileSize != 50*1024*1024 {
		t.Errorf("maxFileSize = %d, want 50MB", maxFileSize)
	}
	if maxURLFileSize >= maxFileSize {
		t.Error("the URL limit must be stricter than the upload limit")
	}
}
