package util

import "net/url"

func IsValidUrl(link string) bool {
	if link == "" {
		return false
	}

	_, err := url.ParseRequestURI(link)
	if err != nil {
		return false
	}

	u, err := url.Parse(link)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}
