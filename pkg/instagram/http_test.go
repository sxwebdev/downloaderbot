package instagram

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sxwebdev/downloaderbot/internal/models"
)

const reelPage = `<html><script>{"code":"ABC123","caption":{"text":"hello"},` +
	`"original_height":1920,"original_width":1080,` +
	`"video_versions":[{"type":101,"url":"https:\/\/cdn.example\/reel.mp4"}]}</script></html>`

func newTestHTTPFetcher(t *testing.T, h http.HandlerFunc) *HTTPFetcher {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	f := NewHTTPFetcher()
	f.baseURL = srv.URL
	return f
}

func TestHTTPFetcher(t *testing.T) {
	t.Parallel()

	t.Run("parses the post page and sends navigation headers", func(t *testing.T) {
		t.Parallel()
		var gotPath, gotAccept, gotMode string
		f := newTestHTTPFetcher(t, func(w http.ResponseWriter, r *http.Request) {
			gotPath, gotAccept, gotMode = r.URL.Path, r.Header.Get("Accept"), r.Header.Get("Sec-Fetch-Mode")
			_, _ = w.Write([]byte(reelPage))
		})

		media, err := f.GetPost(t.Context(), "ABC123")
		if err != nil {
			t.Fatalf("GetPost: %v", err)
		}
		if gotPath != "/p/ABC123/" {
			t.Errorf("path = %q, want /p/ABC123/", gotPath)
		}
		// The full browser Accept is what Instagram gates the media JSON on.
		if gotAccept != navigationHeaders["Accept"] || gotMode != "navigate" {
			t.Errorf("Accept = %q, Sec-Fetch-Mode = %q; want browser navigation headers", gotAccept, gotMode)
		}
		if len(media.Items) != 1 {
			t.Fatalf("items = %d, want 1", len(media.Items))
		}
		item := media.Items[0]
		if item.Type != models.MediaTypeVideo || item.Url != "https://cdn.example/reel.mp4" {
			t.Errorf("item = %+v, want the reel video", item)
		}
		if item.Width != 1080 || item.Height != 1920 {
			t.Errorf("dims = %dx%d, want 1080x1920", item.Width, item.Height)
		}
		if media.Caption != "hello" {
			t.Errorf("caption = %q, want hello", media.Caption)
		}
	})

	t.Run("does not follow a redirect to the login page", func(t *testing.T) {
		t.Parallel()
		var loginHit bool
		f := newTestHTTPFetcher(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/accounts/login/" {
				loginHit = true
				_, _ = w.Write([]byte(reelPage))
				return
			}
			http.Redirect(w, r, "/accounts/login/", http.StatusFound)
		})

		if _, err := f.GetPost(t.Context(), "ABC123"); err == nil {
			t.Fatal("GetPost succeeded on a redirect, want an error")
		}
		if loginHit {
			t.Error("redirect was followed")
		}
	})

	t.Run("page without media is an error", func(t *testing.T) {
		t.Parallel()
		f := newTestHTTPFetcher(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`<html>log in to continue</html>`))
		})
		if _, err := f.GetPost(t.Context(), "ABC123"); err == nil {
			t.Fatal("GetPost succeeded without media, want an error")
		}
	})

	t.Run("non-200 is an error", func(t *testing.T) {
		t.Parallel()
		f := newTestHTTPFetcher(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(reelPage))
		})
		if _, err := f.GetPost(t.Context(), "ABC123"); err == nil {
			t.Fatal("GetPost succeeded on 429, want an error")
		}
	})
}

type fakeFetcher struct {
	media *models.Media
	err   error
	calls int
}

func (f *fakeFetcher) GetPost(context.Context, string) (*models.Media, error) {
	f.calls++
	return f.media, f.err
}

func TestFallbackFetcher(t *testing.T) {
	t.Parallel()

	httpMedia := &models.Media{Shortcode: "http"}
	browserMedia := &models.Media{Shortcode: "browser"}

	t.Run("primary success skips the browser", func(t *testing.T) {
		t.Parallel()
		primary, fallback := &fakeFetcher{media: httpMedia}, &fakeFetcher{media: browserMedia}
		got, err := NewFallbackFetcher(primary, fallback).GetPost(t.Context(), "x")
		if err != nil || got != httpMedia {
			t.Fatalf("got %v, %v; want the primary media", got, err)
		}
		if fallback.calls != 0 {
			t.Errorf("fallback called %d times, want 0", fallback.calls)
		}
	})

	t.Run("primary failure falls back", func(t *testing.T) {
		t.Parallel()
		primary, fallback := &fakeFetcher{err: errors.New("challenge")}, &fakeFetcher{media: browserMedia}
		got, err := NewFallbackFetcher(primary, fallback).GetPost(t.Context(), "x")
		if err != nil || got != browserMedia {
			t.Fatalf("got %v, %v; want the fallback media", got, err)
		}
	})

	t.Run("both fail: the browser error is the wrapped cause", func(t *testing.T) {
		t.Parallel()
		primary := &fakeFetcher{err: errors.New("challenge")}
		fallback := &fakeFetcher{err: context.DeadlineExceeded}
		_, err := NewFallbackFetcher(primary, fallback).GetPost(t.Context(), "x")
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v, want it to wrap the browser's DeadlineExceeded", err)
		}
	})

	t.Run("no fallback once the caller's context is done", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		primary, fallback := &fakeFetcher{err: context.Canceled}, &fakeFetcher{media: browserMedia}
		if _, err := NewFallbackFetcher(primary, fallback).GetPost(ctx, "x"); err == nil {
			t.Fatal("want an error")
		}
		if fallback.calls != 0 {
			t.Errorf("fallback called %d times, want 0", fallback.calls)
		}
	})
}
