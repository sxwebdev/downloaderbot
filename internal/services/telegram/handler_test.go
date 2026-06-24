package telegram

import "testing"

func TestTruncateRunes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"shorter", "abc", 10, "abc"},
		{"exact", "abcde", 5, "abcde"},
		{"longer ascii", "abcdef", 4, "abc…"},
		{"unicode", "привет, мир!", 6, "приве…"},
		{"zero", "abc", 0, ""},
		{"one", "abc", 1, "a"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateRunes(tc.in, tc.max)
			if got != tc.want {
				t.Fatalf("truncateRunes(%q, %d) = %q, want %q", tc.in, tc.max, got, tc.want)
			}
			if got != tc.in {
				if len([]rune(got)) > tc.max {
					t.Fatalf("result %q exceeds max %d runes", got, tc.max)
				}
			}
		})
	}
}
