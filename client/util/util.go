package util

import (
	"crypto/tls"
	"net/http"
	"strings"
	"time"
)

// ConvertEndpointNameToKey converts a group and an endpoint to a key
func ConvertEndpointNameToKey(endpointName string) string {
	return sanitize(endpointName)
}

// GetHTTPClient return an HTTP client matching the Config's parameters.
func GetHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			Proxy:               http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Follow redirects
			return nil
		},
	}
}

func sanitize(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, ",", "-")
	s = strings.ReplaceAll(s, " ", "-")
	return s
}
