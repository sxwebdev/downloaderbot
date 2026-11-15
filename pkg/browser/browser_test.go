package browser

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/launcher/flags"
)

// These tests launch a real Chromium against a local server (no external
// network), so they are skipped with -short like the other browser tests.

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hang" {
			// Never answer: the caller's context has to end the load.
			select {
			case <-r.Context().Done():
			case <-time.After(10 * time.Second):
			}
			return
		}
		_, _ = w.Write([]byte(`<html><body><script>var data = {"ready":"yes"};</script></body></html>`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func isReady(html string) bool { return strings.Contains(html, `"ready":"yes"`) }

func openTabs(t *testing.T, m *Manager) int {
	t.Helper()
	m.mu.Lock()
	in := m.current
	m.mu.Unlock()
	if in == nil {
		t.Fatal("no current browser instance")
	}
	pages, err := in.browser.Pages()
	if err != nil {
		t.Fatalf("list pages: %v", err)
	}
	return len(pages)
}

func TestLoadClosesTabWhenContextEnds(t *testing.T) {
	if testing.Short() {
		t.Skip("launches a browser")
	}
	srv := newTestServer(t)
	m := NewManager()
	t.Cleanup(func() { _ = m.Close() })

	if _, err := m.Load(t.Context(), srv.URL+"/ok", WithReady(isReady)); err != nil {
		t.Fatalf("Load: %v", err)
	}

	// The inline path cancels after 10s; any load that ends on its context used
	// to leave its tab open in Chromium for good.
	for range 3 {
		ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
		_, err := m.Load(ctx, srv.URL+"/hang", WithReady(isReady))
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Load err = %v, want DeadlineExceeded", err)
		}
	}

	if n := openTabs(t, m); n != 0 {
		t.Fatalf("%d tabs left open after loads finished, want 0", n)
	}
}

func TestBrowserIsReplacedAfterMaxLoads(t *testing.T) {
	if testing.Short() {
		t.Skip("launches a browser")
	}
	srv := newTestServer(t)
	m := NewManager()
	m.maxLoads = 2
	t.Cleanup(func() { _ = m.Close() })

	load := func() *instance {
		t.Helper()
		if _, err := m.Load(t.Context(), srv.URL+"/ok", WithReady(isReady)); err != nil {
			t.Fatalf("Load: %v", err)
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		return m.current
	}

	if err := m.Warmup(); err != nil {
		t.Fatalf("Warmup: %v", err)
	}
	m.mu.Lock()
	first := m.current
	m.mu.Unlock()
	firstDir := first.launcher.Get(flags.UserDataDir)

	load()
	load() // second load hits the quota: the instance is retired

	// The replacement is launched in the background; a third load must run on it.
	load()
	m.mu.Lock()
	second := m.current
	m.mu.Unlock()
	if second == nil || second == first {
		t.Fatal("browser was not replaced after reaching its load quota")
	}

	// Retirement deletes the old profile directory (HTTP + code cache).
	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(firstDir); errors.Is(err, os.ErrNotExist) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("profile dir %s of the retired browser still exists", firstDir)
		}
		time.Sleep(100 * time.Millisecond)
	}

	secondDir := second.launcher.Get(flags.UserDataDir)
	if err := m.Close(); err != nil {
		t.Logf("Close: %v", err)
	}
	if _, err := os.Stat(secondDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("profile dir %s still exists after Close", secondDir)
	}
	if _, err := m.Load(t.Context(), srv.URL+"/ok"); !errors.Is(err, ErrClosed) {
		t.Errorf("Load after Close err = %v, want ErrClosed", err)
	}
}
