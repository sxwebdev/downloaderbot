package util

import "testing"

func TestIsValidUrl(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", false},
		{"no scheme", "example.com", false},
		{"only scheme", "https://", false},
		{"valid http", "http://example.com", true},
		{"valid https with path", "https://www.instagram.com/p/abc/", true},
		{"valid with query", "https://x.test/q?a=1", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsValidUrl(tc.in); got != tc.want {
				t.Fatalf("IsValidUrl(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
