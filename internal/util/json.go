package util

import "encoding/json"

// JSONUnescape decodes a JSON string body (handling \/, \uXXXX, etc.) extracted
// from a larger document via regex. It returns the input unchanged if it is not
// a valid JSON string body.
func JSONUnescape(s string) string {
	var out string
	if err := json.Unmarshal([]byte(`"`+s+`"`), &out); err == nil {
		return out
	}
	return s
}
