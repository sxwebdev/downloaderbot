package telegram

import (
	"strings"
	"testing"

	"github.com/sxwebdev/downloaderbot/internal/models"
)

// TestVideoFromItem guards the second half of the reported bug: a reel the bot
// delivers to a direct message renders in Telegram with no length until the user
// downloads the whole file. Telegram never probes an uploaded file, so a
// sendVideo call that omits duration leaves the clients nothing to display.
func TestVideoFromItem(t *testing.T) {
	tests := []struct {
		name string
		item *models.MediaItem
	}{
		{
			// The reported reel: 1080x1920, ~120s.
			name: "reel with full metadata",
			item: &models.MediaItem{
				Type:     models.MediaTypeVideo,
				Width:    1080,
				Height:   1920,
				Duration: 120,
				MimeType: "video/mp4",
			},
		},
		{
			name: "duration unknown",
			item: &models.MediaItem{Type: models.MediaTypeVideo, Width: 1080, Height: 1920},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			video := videoFromItem(tc.item, strings.NewReader("payload"))

			if video.Duration != tc.item.Duration {
				t.Errorf("Duration = %d, want %d — Telegram shows no length without it",
					video.Duration, tc.item.Duration)
			}
			// A 2-minute reel that cannot stream must be downloaded in full before it
			// plays, which is the other half of what the user sees.
			if !video.Streaming {
				t.Error("Streaming = false, want true so the client can play before the download finishes")
			}
			if video.Width != tc.item.Width || video.Height != tc.item.Height {
				t.Errorf("dimensions = %dx%d, want %dx%d",
					video.Width, video.Height, tc.item.Width, tc.item.Height)
			}
			if video.MIME != tc.item.MimeType {
				t.Errorf("MIME = %q, want %q", video.MIME, tc.item.MimeType)
			}
			if video.File.FileReader == nil {
				t.Error("File carries no reader, so nothing would be uploaded")
			}
		})
	}
}

// TestVideoFromItem_MatchesAcrossPaths pins the single-video and album paths to
// the same description of a file. They used to build their own telebot.Video
// literals, which is how one could gain a field the other lacked.
func TestVideoFromItem_MatchesAcrossPaths(t *testing.T) {
	item := &models.MediaItem{
		Type:     models.MediaTypeVideo,
		Width:    1080,
		Height:   1920,
		Duration: 120,
		MimeType: "video/mp4",
	}

	single := videoFromItem(item, strings.NewReader("a"))
	album := videoFromItem(item, strings.NewReader("b"))

	if single.Duration != album.Duration ||
		single.Streaming != album.Streaming ||
		single.Width != album.Width ||
		single.Height != album.Height ||
		single.MIME != album.MIME {
		t.Fatalf("paths disagree: single=%+v album=%+v", single, album)
	}
}
