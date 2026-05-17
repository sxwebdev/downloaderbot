package telegram

import "testing"

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"hello", "hello"},
		{"*bold*", "\\*bold\\*"},
		{"a_b_c", "a\\_b\\_c"},
		{"code: `x`", "code: \\`x\\`"},
		{"[link]", "\\[link]"},
		{"mix *a_b* `c` [d]", "mix \\*a\\_b\\* \\`c\\` \\[d]"},
	}

	for _, tc := range tests {
		if got := escapeMarkdown(tc.in); got != tc.want {
			t.Errorf("escapeMarkdown(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

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
