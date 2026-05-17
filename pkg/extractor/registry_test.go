package extractor

import (
	"context"
	"strings"
	"testing"

	"github.com/sxwebdev/downloaderbot/internal/models"
)

type fakeExtractor struct {
	name  string
	hosts []string
}

func (f *fakeExtractor) Name() string                                           { return f.name }
func (f *fakeExtractor) Hosts() []string                                        { return f.hosts }
func (f *fakeExtractor) Extract(context.Context, string) (*models.Media, error) { return nil, nil }

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()

	// `www.` prefix would normalize to the same host — registry rejects duplicates,
	// so only the canonical form is listed here.
	a := &fakeExtractor{name: "site-a", hosts: []string{"a.test"}}
	b := &fakeExtractor{name: "site-b", hosts: []string{"b.test"}}

	if err := r.Register(a); err != nil {
		t.Fatalf("register a: %v", err)
	}
	if err := r.Register(b); err != nil {
		t.Fatalf("register b: %v", err)
	}

	if _, ok := r.GetByName("site-a"); !ok {
		t.Fatal("expected site-a by name")
	}
	if _, ok := r.GetByName("missing"); ok {
		t.Fatal("did not expect missing")
	}

	ext, err := r.GetByURL("https://m.a.test/foo")
	if err != nil {
		t.Fatalf("GetByURL: %v", err)
	}
	if ext.Name() != "site-a" {
		t.Fatalf("got %q, want site-a", ext.Name())
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(&fakeExtractor{name: "x", hosts: []string{"x.test"}}); err != nil {
		t.Fatalf("first register: %v", err)
	}
	err := r.Register(&fakeExtractor{name: "x", hosts: []string{"y.test"}})
	if err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("want duplicate error, got %v", err)
	}

	err = r.Register(&fakeExtractor{name: "y", hosts: []string{"x.test"}})
	if err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("want host duplicate error, got %v", err)
	}
}

func TestRegistry_GetByURL_Unsupported(t *testing.T) {
	r := NewRegistry()
	if _, err := r.GetByURL("https://nowhere.test"); err == nil {
		t.Fatal("expected error for unsupported host")
	}
}

func TestNormalizeHost(t *testing.T) {
	cases := map[string]string{
		"WWW.Example.com":    "example.com",
		"m.youtube.com":      "youtube.com",
		"mobile.twitter.com": "twitter.com",
		"example.com":        "example.com",
	}
	for in, want := range cases {
		if got := normalizeHost(in); got != want {
			t.Errorf("normalizeHost(%q) = %q, want %q", in, got, want)
		}
	}
}
