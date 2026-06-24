package util

import (
	"net"
	"net/http"
	"time"
)

var defaultTransport = &http.Transport{
	DialContext: (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	TLSHandshakeTimeout:   5 * time.Second,
	IdleConnTimeout:       90 * time.Second,
	MaxIdleConns:          100,
	MaxIdleConnsPerHost:   20,
	MaxConnsPerHost:       50,
	ForceAttemptHTTP2:     true,
	ResponseHeaderTimeout: 15 * time.Second,
}

var defaultClient = &http.Client{
	Timeout:   30 * time.Second,
	Transport: defaultTransport,
}

func DefaultHttpClient() *http.Client { return defaultClient }

// DefaultTransport returns the shared HTTP transport used by the default client,
// so callers can build their own *http.Client (e.g. with a cookie jar) while
// reusing the same connection pool and timeouts.
func DefaultTransport() *http.Transport { return defaultTransport }
