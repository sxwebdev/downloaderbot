package instagram

import (
	"strings"
	"testing"
)

// dashManifest renders a "video_dash_manifest" value the way Instagram embeds it
// in a post page: an XML document serialized *into* a JSON string, so its own
// attribute quotes arrive backslash-escaped.
func dashManifest(iso string) string {
	return `"video_dash_manifest":"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n` +
		`<MPD xmlns=\"urn:mpeg:dash:schema:mpd:2011\" mediaPresentationDuration=\"` + iso + `\" minBufferTime=\"PT1.500S\">",`
}

// TestParseMediaFromHTML_Duration guards the "no length until downloaded"
// regression. Telegram never probes an uploaded or fetched video, so a post sent
// without an explicit duration renders in the client as a video of unknown
// length that must be downloaded in full before it shows anything. The post page
// Instagram serves carries no "video_duration" key at all — only the DASH
// manifest does — so the parser has to read the manifest.
func TestParseMediaFromHTML_Duration(t *testing.T) {
	const video = `"video_versions":[{"type":101,"url":"https:\/\/cdn.example\/reel.mp4"}]`

	tests := []struct {
		name string
		html string
		want int
	}{
		{
			// The shape the real reel DbTK_BOssyb serves: duration lives only in the
			// escaped DASH manifest, as fractional seconds.
			name: "dash manifest fractional seconds rounds up",
			html: `{` + dashManifest("PT119.575592S") + video + `}`,
			want: 120,
		},
		{
			name: "dash manifest rounds down",
			html: `{` + dashManifest("PT12.4S") + video + `}`,
			want: 12,
		},
		{
			name: "dash manifest whole seconds",
			html: `{` + dashManifest("PT30S") + video + `}`,
			want: 30,
		},
		{
			// The GraphQL-shaped payload spells the duration explicitly.
			name: "explicit video_duration key",
			html: `{"video_duration":45.2,` + video + `}`,
			want: 45,
		},
		{
			// Both present: the explicit key is authoritative, the manifest is only
			// the fallback for payloads that omit it.
			name: "explicit key wins over manifest",
			html: `{"video_duration":45.2,` + dashManifest("PT99S") + video + `}`,
			want: 45,
		},
		{
			// An unescaped manifest (a payload variant that is not double-encoded)
			// must parse just as well.
			name: "unescaped manifest quotes",
			html: `{"m":"<MPD mediaPresentationDuration="PT8.5S">",` + video + `}`,
			want: 9,
		},
		{
			name: "no duration anywhere yields zero",
			html: `{` + video + `}`,
			want: 0,
		},
		{
			// A malformed value must not be reported as a real duration.
			name: "zero duration is not reported",
			html: `{` + dashManifest("PT0S") + video + `}`,
			want: 0,
		},
		{
			// Photos carry no duration and must not inherit a stray one.
			name: "photo has no duration",
			html: `{"video_duration":45.2,"image_versions2":{"candidates":[{"url":"https:\/\/cdn.example\/p.jpg"}]}}`,
			want: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			item := firstItem(t, tc.html)
			if item.Duration != tc.want {
				t.Fatalf("Duration = %d, want %d", item.Duration, tc.want)
			}
		})
	}
}

// TestParseMediaFromHTML_DurationPicksNearestBlock guards against inheriting a
// suggested post's duration. A post page embeds unrelated reels, each with its
// own DASH manifest, rendered before the requested post's media — the same trap
// that once made the parser report a stranger's caption.
func TestParseMediaFromHTML_DurationPicksNearestBlock(t *testing.T) {
	const wantCode = "DbTK_BOssyb"

	decoy := `{"code":"OTHER1",` + dashManifest("PT7S") +
		`"video_versions":[{"type":101,"url":"https:\/\/cdn.example\/decoy.mp4"}]}`
	main := `{"code":"` + wantCode + `",` + dashManifest("PT119.575592S") +
		`"video_versions":[{"type":101,"url":"https:\/\/cdn.example\/reel.mp4"}]}`

	// The decoy is rendered first, so a first-match parser reports its 7s.
	media, err := parseMediaFromHTML(`{"items":[`+decoy+`,`+main+`]}`, wantCode)
	if err != nil {
		t.Fatalf("parseMediaFromHTML: %v", err)
	}
	if len(media.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(media.Items))
	}

	// The first video_versions on the page belongs to the decoy, so that is the
	// item extracted; what matters is that the duration paired with it is the
	// decoy's own 7s and never leaks across post boundaries.
	item := media.Items[0]
	if strings.Contains(item.Url, "decoy") {
		if item.Duration != 7 {
			t.Fatalf("decoy item Duration = %d, want its own 7", item.Duration)
		}
		return
	}
	if item.Duration != 120 {
		t.Fatalf("Duration = %d, want 120", item.Duration)
	}
}

// TestParseMediaFromHTML_CarouselPerItemDuration checks each carousel child gets
// its own duration rather than the first child's.
func TestParseMediaFromHTML_CarouselPerItemDuration(t *testing.T) {
	first := `{` + dashManifest("PT10S") +
		`"video_versions":[{"type":101,"url":"https:\/\/cdn.example\/a.mp4"}]}`
	second := `{` + dashManifest("PT25S") +
		`"video_versions":[{"type":101,"url":"https:\/\/cdn.example\/b.mp4"}]}`

	media, err := parseMediaFromHTML(`{"carousel_media":[`+first+`,`+second+`]}`, "ABC123")
	if err != nil {
		t.Fatalf("parseMediaFromHTML: %v", err)
	}
	if len(media.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(media.Items))
	}

	want := []int{10, 25}
	for i, item := range media.Items {
		if item.Duration != want[i] {
			t.Fatalf("item %d Duration = %d, want %d", i, item.Duration, want[i])
		}
	}
}
