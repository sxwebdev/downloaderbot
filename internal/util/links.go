package util

import "regexp"

var linkRe = regexp.MustCompile(`(http|ftp|https)://([\w_-]+(?:(?:\.[\w_-]+)+))([\w.,@?^=%&:/~+#-]*[\w@?^=%&/~+#-])?`)

func ExtractLinksFromString(input string) []string {
	return linkRe.FindAllString(input, -1)
}
