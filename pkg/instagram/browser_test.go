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
		// The reel that showed the bug: ~2 minutes and 22.4MB, so it fits the 50MB
		// upload cap the chat path uses but not the 20MB cap Telegram applies when
		// it fetches an inline result from a URL.
		{name: "reel-DbTK_BOssyb", link: "https://www.instagram.com/reel/DbTK_BOssyb/?igsh=MWFwdWN4cXV6cGY0aA=="},
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

			start := time.Now()
			media, err := f.GetPost(t.Context(), code)
			elapsed := time.Since(start)
			if err != nil {
				t.Fatalf("GetPost: %v", err)
			}
			if len(media.Items) == 0 || media.Items[0].Url == "" {
				t.Fatalf("no media items extracted: %+v", media)
			}

			item := media.Items[0]
			t.Logf("type=%s %dx%d in %s caption=%.40q url=%.80s",
				media.Type, item.Width, item.Height, elapsed.Round(time.Millisecond), media.Caption, item.Url)

			// Reels are vertical: the dimensions sent to Telegram must be portrait,
			// not the squeezed square the first-match parser used to produce.
			if item.Width <= 0 || item.Height <= 0 {
				t.Fatalf("missing dimensions: %dx%d", item.Width, item.Height)
			}
			if item.Width >= item.Height {
				t.Fatalf("expected portrait reel, got %dx%d", item.Width, item.Height)
			}

			// Telegram never probes the file it is handed, so a reel sent without a
			// duration shows no length in the client until the user has downloaded
			// all of it. The post page carries this only inside the DASH manifest.
			if item.Duration <= 0 {
				t.Fatalf("missing duration: %d", item.Duration)
			}

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Head(media.Items[0].Url)
			if err != nil {
				t.Fatalf("HEAD: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("HEAD status = %d", resp.StatusCode)
			}
			t.Logf("downloadable: %d bytes (%.1fMB), %s, %ds",
				resp.ContentLength, float64(resp.ContentLength)/1024/1024,
				resp.Header.Get("Content-Type"), item.Duration)

			// Inline video results carry a mandatory thumbnail_url that Telegram
			// documents as "JPEG only" — the .mp4 URL is not a valid value for it.
			if item.ThumbnailUrl == "" {
				t.Fatal("missing thumbnail url")
			}
			thumbResp, err := client.Head(item.ThumbnailUrl)
			if err != nil {
				t.Fatalf("thumbnail HEAD: %v", err)
			}
			defer thumbResp.Body.Close()
			if thumbResp.StatusCode != http.StatusOK {
				t.Fatalf("thumbnail HEAD status = %d", thumbResp.StatusCode)
			}
			if ct := thumbResp.Header.Get("Content-Type"); ct != "image/jpeg" {
				t.Fatalf("thumbnail Content-Type = %q, want image/jpeg", ct)
			}
			t.Logf("thumbnail: %d bytes", thumbResp.ContentLength)
		})
	}
}
