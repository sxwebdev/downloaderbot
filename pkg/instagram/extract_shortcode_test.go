package instagram

import "testing"

func TestExtractShortcodeFromLink(t *testing.T) {
	tests := []struct {
		name    string
		link    string
		want    string
		wantErr bool
	}{
		{"post", "https://www.instagram.com/p/CzBjgFiISfF/?igshid=x", "CzBjgFiISfF", false},
		{"reel", "https://www.instagram.com/reel/C0tV4iMvlS_/", "C0tV4iMvlS_", false},
		{"tv", "https://www.instagram.com/tv/AbCdEf_-123/", "AbCdEf_-123", false},
		{"reels-videos", "https://www.instagram.com/reels/videos/XyZ12-_AbC/", "XyZ12-_AbC", false},
		{"trailing-slash-removed", "https://instagram.com/p/abcDEF/", "abcDEF", false},
		{"no-match", "https://example.com/page", "", true},
		{"empty", "", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExtractShortcodeFromLink(tc.link)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
