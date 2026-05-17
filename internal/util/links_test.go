package util

import (
	"reflect"
	"testing"
)

func TestExtractLinksFromString(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty", in: "", want: nil},
		{name: "plain text", in: "hello world", want: nil},
		{name: "single http", in: "go to http://example.com please", want: []string{"http://example.com"}},
		{name: "single https with path", in: "https://www.instagram.com/p/CzBjgFiISfF/", want: []string{"https://www.instagram.com/p/CzBjgFiISfF/"}},
		{name: "two links", in: "first https://a.test/x and second http://b.test/y", want: []string{"https://a.test/x", "http://b.test/y"}},
		{name: "ftp", in: "ftp://files.example.org/path", want: []string{"ftp://files.example.org/path"}},
		{name: "query string preserved", in: "see https://x.test/q?a=1&b=2#frag end", want: []string{"https://x.test/q?a=1&b=2#frag"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractLinksFromString(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}
