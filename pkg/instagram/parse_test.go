package instagram

import (
	"strings"
	"testing"

	"github.com/sxwebdev/downloaderbot/internal/models"
)

// TestParseMediaFromHTML_Dimensions guards the reel "squeezed square" regression.
// Instagram serves the real video size as a single original_width/original_height
// pair in either field order, while image_versions2 candidates carry only bare,
// often-square "width"/"height". The old regex assumed width-first and so missed
// reels entirely (sending 0x0, which Telegram squeezed into a square); these
// tests lock in order-independent extraction that ignores the square thumbnails.
func TestParseMediaFromHTML_Dimensions(t *testing.T) {
	// thumb is an image_versions2 block whose candidate carries a square
	// width/height — these must never be picked as the media dimensions.
	const thumb = `"image_versions2":{"candidates":[{"url":"https:\/\/cdn.example\/thumb.jpg","width":768,"height":768}]},`

	t.Run("video width-first order", func(t *testing.T) {
		html := `{` +
			`"caption":{"text":"a vertical reel"},` + thumb +
			`"original_width":1080,"original_height":1920,` +
			`"video_versions":[{"type":101,"url":"https:\/\/cdn.example\/reel.mp4?sig=abc"}]` +
			`}`

		item := firstItem(t, html)
		if item.Type != models.MediaTypeVideo {
			t.Fatalf("type = %q, want video", item.Type)
		}
		assertDims(t, item, 1080, 1920)
		if !strings.Contains(item.Url, "reel.mp4") || strings.Contains(item.Url, `\`) {
			t.Fatalf("url = %q, want unescaped reel url", item.Url)
		}
	})

	t.Run("video height-first order (real reel format)", func(t *testing.T) {
		// Instagram emits reels height-then-width; the old width-first regex used to
		// miss this entirely and send 0x0.
		html := `{` + thumb +
			`"original_height":1920,"original_width":1080,` +
			`"video_versions":[{"type":101,"url":"https:\/\/cdn.example\/reel.mp4"}]` +
			`}`

		assertDims(t, firstItem(t, html), 1080, 1920)
	})

	t.Run("photo", func(t *testing.T) {
		html := `{` + thumb + `"original_width":1080,"original_height":1350}`

		item := firstItem(t, html)
		if item.Type != models.MediaTypePhoto {
			t.Fatalf("type = %q, want photo", item.Type)
		}
		assertDims(t, item, 1080, 1350)
	})

	t.Run("no original dimensions falls back to zero", func(t *testing.T) {
		// Only square candidate width/height present; nothing extractable -> 0x0,
		// leaving Telegram to detect the size from the file rather than squeezing.
		html := `{` + thumb +
			`"video_versions":[{"type":101,"url":"https:\/\/cdn.example\/reel.mp4"}]}`

		assertDims(t, firstItem(t, html), 0, 0)
	})
}

func firstItem(t *testing.T, html string) *models.MediaItem {
	t.Helper()
	media, err := parseMediaFromHTML(html, "ABC123")
	if err != nil {
		t.Fatalf("parseMediaFromHTML: %v", err)
	}
	if len(media.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(media.Items))
	}
	return media.Items[0]
}

func assertDims(t *testing.T, item *models.MediaItem, wantW, wantH int) {
	t.Helper()
	if item.Width != wantW || item.Height != wantH {
		t.Fatalf("dimensions = %dx%d, want %dx%d", item.Width, item.Height, wantW, wantH)
	}
}
