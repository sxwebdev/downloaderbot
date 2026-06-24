package tiktok_test

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/sxwebdev/downloaderbot/pkg/browser"
	"github.com/sxwebdev/downloaderbot/pkg/tiktok"
)

func TestGetVideo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	links := []string{
		"https://vt.tiktok.com/ZSCNjNQFC/",
	}
	if v := os.Getenv("TT_URL"); v != "" {
		links = []string{v}
	}

	t.Cleanup(func() { _ = browser.Default().Close() })

	for _, link := range links {
		t.Run(link, func(t *testing.T) {
			media, err := tiktok.GetVideo(t.Context(), link)
			if err != nil {
				t.Fatalf("GetVideo: %v", err)
			}
			if len(media.Items) == 0 || media.Items[0].Url == "" {
				t.Fatalf("no media: %+v", media)
			}
			item := media.Items[0]
			t.Logf("caption=%.40q url=%.90s", media.Caption, item.Url)

			req, _ := http.NewRequest(http.MethodGet, item.Url, nil)
			for k, v := range item.DownloadHeaders {
				req.Header.Set(k, v)
			}
			req.Header.Set("Range", "bytes=0-")
			client := &http.Client{Timeout: 15 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("download: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode/100 != 2 {
				t.Fatalf("download status = %d", resp.StatusCode)
			}
			t.Logf("downloadable: status=%d content-length=%d type=%s",
				resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"))
		})
	}
}
