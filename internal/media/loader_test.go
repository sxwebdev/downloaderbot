package media_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sxwebdev/downloaderbot/internal/media"
	"github.com/sxwebdev/downloaderbot/internal/models"
)

func TestDirectURL(t *testing.T) {
	loader := media.NewHTTPLoader()

	tests := []struct {
		name    string
		item    *models.MediaItem
		wantURL string
		wantOK  bool
	}{
		{
			name:    "public url is direct",
			item:    &models.MediaItem{Url: "https://cdn.example/x.mp4"},
			wantURL: "https://cdn.example/x.mp4",
			wantOK:  true,
		},
		{
			name:   "url with download headers is not direct",
			item:   &models.MediaItem{Url: "https://cdn.tiktok/x.mp4", DownloadHeaders: map[string]string{"Cookie": "a=b"}},
			wantOK: false,
		},
		{
			name:   "empty url",
			item:   &models.MediaItem{},
			wantOK: false,
		},
		{
			name:   "nil item",
			item:   nil,
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			url, ok := loader.DirectURL(tc.item)
			if ok != tc.wantOK || url != tc.wantURL {
				t.Fatalf("DirectURL = (%q, %v), want (%q, %v)", url, ok, tc.wantURL, tc.wantOK)
			}
		})
	}
}

func TestContentLength(t *testing.T) {
	const size = 60 * 1024 * 1024 // over Telegram's 50MB cap

	var gotMethod, gotCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotCookie = r.Header.Get("Cookie")
		if r.URL.Path == "/missing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Length", "62914560")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	loader := media.NewHTTPLoader()

	t.Run("reports size via HEAD and applies download headers", func(t *testing.T) {
		item := &models.MediaItem{
			Url:             srv.URL + "/video.mp4",
			DownloadHeaders: map[string]string{"Cookie": "a=b"},
		}

		got, err := loader.ContentLength(t.Context(), item)
		if err != nil {
			t.Fatalf("ContentLength: %v", err)
		}
		if got != size {
			t.Fatalf("size = %d, want %d", got, size)
		}
		if gotMethod != http.MethodHead {
			t.Fatalf("method = %q, want HEAD", gotMethod)
		}
		if gotCookie != "a=b" {
			t.Fatalf("cookie = %q, want a=b", gotCookie)
		}
	})

	t.Run("non-2xx is an error", func(t *testing.T) {
		item := &models.MediaItem{Url: srv.URL + "/missing"}
		if _, err := loader.ContentLength(t.Context(), item); err == nil {
			t.Fatal("expected error for 404 response")
		}
	})

	t.Run("empty url is an error", func(t *testing.T) {
		if _, err := loader.ContentLength(t.Context(), &models.MediaItem{}); err == nil {
			t.Fatal("expected error for empty url")
		}
	})
}
