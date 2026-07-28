package instagram

import "testing"

// TestParseMediaFromHTML_Thumbnail covers the JPEG cover frame that Telegram
// requires for inline video results (thumbnail_url is documented "JPEG only",
// so the .mp4 URL is not a usable stand-in).
func TestParseMediaFromHTML_Thumbnail(t *testing.T) {
	const video = `"video_versions":[{"type":101,"url":"https:\/\/cdn.example\/reel.mp4"}]`

	tests := []struct {
		name string
		html string
		want string
	}{
		{
			// The shape a real reel serves: image_versions2 spells
			// additional_candidates *before* candidates, so anchoring the cover on
			// `"image_versions2":{"candidates"` finds nothing here.
			name: "first_frame ahead of candidates",
			html: `{"image_versions2":{"additional_candidates":{"first_frame":` +
				`{"url":"https:\/\/cdn.example\/cover.jpg"}},"candidates":` +
				`[{"url":"https:\/\/cdn.example\/big.jpg","height":2096,"width":1179}]},` + video + `}`,
			want: "https://cdn.example/cover.jpg",
		},
		{
			name: "candidates only",
			html: `{"image_versions2":{"candidates":[{"url":"https:\/\/cdn.example\/big.jpg"}]},` + video + `}`,
			want: "https://cdn.example/big.jpg",
		},
		{
			// Nothing to offer: the caller falls back to the media URL rather than
			// dropping the result.
			name: "no cover yields empty",
			html: `{` + video + `}`,
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			item := firstItem(t, tc.html)
			if item.ThumbnailUrl != tc.want {
				t.Fatalf("ThumbnailUrl = %q, want %q", item.ThumbnailUrl, tc.want)
			}
		})
	}
}

// TestParseMediaFromHTML_CarouselPerItemThumbnail checks each carousel child
// keeps its own cover instead of sharing the first child's.
func TestParseMediaFromHTML_CarouselPerItemThumbnail(t *testing.T) {
	child := func(name string) string {
		return `{"image_versions2":{"additional_candidates":{"first_frame":` +
			`{"url":"https:\/\/cdn.example\/` + name + `.jpg"}}},` +
			`"video_versions":[{"type":101,"url":"https:\/\/cdn.example\/` + name + `.mp4"}]}`
	}

	media, err := parseMediaFromHTML(`{"carousel_media":[`+child("a")+`,`+child("b")+`]}`, "ABC123")
	if err != nil {
		t.Fatalf("parseMediaFromHTML: %v", err)
	}
	if len(media.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(media.Items))
	}

	want := []string{"https://cdn.example/a.jpg", "https://cdn.example/b.jpg"}
	for i, item := range media.Items {
		if item.ThumbnailUrl != want[i] {
			t.Fatalf("item %d ThumbnailUrl = %q, want %q", i, item.ThumbnailUrl, want[i])
		}
	}
}
