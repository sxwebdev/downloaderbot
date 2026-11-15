package lux

import (
	"strings"
	"testing"

	"github.com/iawia002/lux/extractors"
)

type fakeLux struct {
	got   string
	data  []*extractors.Data
	panic bool
}

func (f *fakeLux) Extract(url string, _ extractors.Options) ([]*extractors.Data, error) {
	f.got = url
	if f.panic {
		var m map[string]*extractors.Stream
		m["x"].ID = "boom" // nil pointer dereference, like a lux scraper on unexpected markup
	}
	return f.data, nil
}

func videoData(url string) []*extractors.Data {
	return []*extractors.Data{{
		Type: extractors.DataTypeVideo,
		Streams: map[string]*extractors.Stream{
			"720p": {Parts: []*extractors.Part{{URL: url, Ext: "mp4"}}},
		},
	}}
}

// Every site the registry can route a link to must have a lux extractor:
// otherwise the link reaches lux's host-based dispatch, which panics on hosts
// it does not know.
func TestEverySiteHasLuxExtractor(t *testing.T) {
	t.Parallel()
	for host, site := range hostToSite {
		if _, ok := siteExtractors[site]; !ok {
			t.Errorf("host %s maps to site %q, which has no entry in siteExtractors", host, site)
		}
	}
	for _, e := range GetAllExtractors() {
		if e.lux == nil {
			t.Errorf("extractor %q was built without a lux extractor", e.name)
		}
	}
}

// Short-link hosts used to panic in production: lux derived "pin" / "x" / "fb"
// from the host, found nothing registered and called a nil extractor. The link
// must go to the extractor of the site our registry resolved instead.
func TestExtractDispatchesBySiteNotHost(t *testing.T) {
	t.Parallel()
	tests := []struct {
		site, link string
	}{
		{"pinterest", "https://pin.it/6jOeQfO6C"},
		{"twitter", "https://x.com/someone/status/2121052987594281423/video/1?s=52"},
		{"facebook", "https://fb.watch/abc123/"},
		{"reddit", "https://redd.it/abc123"},
		{"vk", "https://vkvideo.ru/video-1_2"},
	}
	for _, tc := range tests {
		t.Run(tc.site, func(t *testing.T) {
			t.Parallel()
			fake := &fakeLux{data: videoData("https://cdn.example/v.mp4")}
			e := GetExtractorBySite(tc.site)
			e.lux = fake

			media, err := e.Extract(t.Context(), "  "+tc.link+" ")
			if err != nil {
				t.Fatalf("Extract: %v", err)
			}
			if fake.got != tc.link {
				t.Errorf("lux extractor got %q, want the trimmed link %q", fake.got, tc.link)
			}
			if len(media.Items) != 1 || media.Items[0].Url != "https://cdn.example/v.mp4" {
				t.Errorf("items = %+v, want the one video", media.Items)
			}
			if media.Source != siteToSource[tc.site] {
				t.Errorf("source = %q, want %q", media.Source, siteToSource[tc.site])
			}
		})
	}
}

func TestExtractTurnsLuxFailuresIntoErrors(t *testing.T) {
	t.Parallel()

	t.Run("panic in the lux extractor", func(t *testing.T) {
		t.Parallel()
		e := GetExtractorBySite("pinterest")
		e.lux = &fakeLux{panic: true}
		_, err := e.Extract(t.Context(), "https://pin.it/abc")
		if err == nil || !strings.Contains(err.Error(), "panicked") {
			t.Fatalf("err = %v, want a panic turned into an error", err)
		}
	})

	t.Run("nil data item", func(t *testing.T) {
		t.Parallel()
		e := GetExtractorBySite("pinterest")
		e.lux = &fakeLux{data: []*extractors.Data{nil}}
		if _, err := e.Extract(t.Context(), "https://pin.it/abc"); err == nil {
			t.Fatal("want an error for a nil data item")
		}
	})

	t.Run("only segmented streams", func(t *testing.T) {
		t.Parallel()
		e := GetExtractorBySite("twitter")
		e.lux = &fakeLux{data: []*extractors.Data{{
			Type: extractors.DataTypeVideo,
			Streams: map[string]*extractors.Stream{"720p": {Parts: []*extractors.Part{
				{URL: "https://video.example/1.ts"}, {URL: "https://video.example/2.ts"},
			}}},
		}}}
		if _, err := e.Extract(t.Context(), "https://x.com/a/status/1"); err == nil {
			t.Fatal("want an error when nothing is deliverable")
		}
	})

	t.Run("site without a lux extractor", func(t *testing.T) {
		t.Parallel()
		e := NewExtractor("nosuchsite", []string{"nosuchsite.example"})
		if _, err := e.Extract(t.Context(), "https://nosuchsite.example/v/1"); err == nil {
			t.Fatal("want an error for a site lux does not support")
		}
	})
}

// lux returns HLS (Twitter/X) and DASH (Bilibili) streams as many parts that
// lux merges with ffmpeg; sending each part would deliver an album of
// fragments, so such streams are skipped.
func TestConvertSkipsMultiPartStreams(t *testing.T) {
	t.Parallel()
	segments := make([]*extractors.Part, 17)
	for i := range segments {
		segments[i] = &extractors.Part{URL: "https://video.example/seg.ts", Ext: "ts"}
	}

	t.Run("only segmented streams", func(t *testing.T) {
		t.Parallel()
		data := &extractors.Data{
			Type:    extractors.DataTypeVideo,
			Streams: map[string]*extractors.Stream{"720p": {Parts: segments}, "360p": {Parts: segments}},
		}
		if items := convertLuxDataToMedia(data, "https://x.com/a/status/1").Items; len(items) != 0 {
			t.Fatalf("items = %d, want 0 for segmented streams", len(items))
		}
	})

	t.Run("single-part stream kept next to a segmented one", func(t *testing.T) {
		t.Parallel()
		data := &extractors.Data{
			Type: extractors.DataTypeVideo,
			Streams: map[string]*extractors.Stream{
				"hls": {Parts: segments},
				"hd":  {Parts: []*extractors.Part{{URL: "https://video.example/hd.mp4", Ext: "mp4", Size: 42}}},
			},
		}
		items := convertLuxDataToMedia(data, "https://example.com/v").Items
		if len(items) != 1 {
			t.Fatalf("items = %d, want 1", len(items))
		}
		if it := items[0]; it.Url != "https://video.example/hd.mp4" || it.Quality != "hd" || it.ContentLength != 42 || it.MimeType != "video/mp4" {
			t.Errorf("item = %+v, want the single-part hd stream", it)
		}
	})
}
