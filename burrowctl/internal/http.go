package internal

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Custom authentication headers are not stripped by Go's default redirect
// policy. Keep management credentials on the original origin and transport.
func NewManagementHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			first := via[0].URL
			if req.URL.Scheme != first.Scheme || !strings.EqualFold(req.URL.Host, first.Host) {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}
