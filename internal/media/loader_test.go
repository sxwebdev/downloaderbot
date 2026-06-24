package media_test

import (
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
