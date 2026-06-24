package instagram_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/sxwebdev/downloaderbot/pkg/browser"
	"github.com/sxwebdev/downloaderbot/pkg/instagram"
)

func TestBrowserFetcher(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name string
		link string
	}{
		{name: "reel-DZ7PxDJIcz9", link: "https://www.instagram.com/reel/DZ7PxDJIcz9/?igsh=MXNkbmlocmp3Z3JzeQ=="},
		{name: "reel-DZuZ4SvtrnP", link: "https://www.instagram.com/reel/DZuZ4SvtrnP/?igsh=MXdyaXozd2VnN2k5OQ=="},
	}

	f := instagram.NewBrowserFetcher()
	t.Cleanup(func() { _ = browser.Default().Close() })

	t.Run("carousel-multi-item", func(t *testing.T) {
		// A multi-photo carousel post: every child item must be extracted.
		code, err := instagram.ExtractShortcodeFromLink("https://www.instagram.com/p/C0FBSN8Re1y/")
		if err != nil {
			t.Fatal(err)
		}
		media, err := f.GetPost(t.Context(), code)
		if err != nil {
			t.Fatalf("GetPost: %v", err)
		}
		if len(media.Items) < 2 {
			t.Fatalf("expected multiple carousel items, got %d", len(media.Items))
		}
		for i, item := range media.Items {
			if item.Url == "" {
				t.Fatalf("carousel item %d has empty url", i)
			}
		}
		t.Logf("carousel items: %d", len(media.Items))
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, err := instagram.ExtractShortcodeFromLink(tc.link)
			if err != nil {
				t.Fatal(err)
			}

			media, err := f.GetPost(t.Context(), code)
			if err != nil {
				t.Fatalf("GetPost: %v", err)
			}
			if len(media.Items) == 0 || media.Items[0].Url == "" {
				t.Fatalf("no media items extracted: %+v", media)
			}

			t.Logf("type=%s caption=%.40q url=%.80s", media.Type, media.Caption, media.Items[0].Url)

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Head(media.Items[0].Url)
			if err != nil {
				t.Fatalf("HEAD: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("HEAD status = %d", resp.StatusCode)
			}
			t.Logf("downloadable: %d bytes, %s", resp.ContentLength, resp.Header.Get("Content-Type"))
		})
	}
}
