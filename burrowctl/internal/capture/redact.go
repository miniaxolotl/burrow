package capture

import (
	"net/http"
	"net/url"
	"strings"
)

func secret(name string) bool {
	name = strings.ToLower(strings.ReplaceAll(name, "_", "-"))
	return name == "authorization" || name == "cookie" || name == "set-cookie" || name == "key" || name == "code" || name == "auth" || strings.Contains(name, "token") || strings.Contains(name, "secret") || strings.Contains(name, "password") || strings.Contains(name, "signature") || strings.Contains(name, "api-key") || name == "proxy-authorization"
}
func stripHeaders(headers http.Header) bool {
	changed := false
	for _, value := range headers.Values("Connection") {
		for _, key := range strings.Split(value, ",") {
			key = strings.TrimSpace(key)
			if headers.Get(key) != "" {
				changed = true
			}
			headers.Del(key)
		}
	}
	for key := range headers {
		lower := strings.ToLower(key)
		if secret(lower) || lower == "connection" || lower == "upgrade" || lower == "transfer-encoding" || lower == "content-length" || lower == "trailer" || lower == "te" || lower == "keep-alive" || lower == "proxy-connection" || lower == "forwarded" || lower == "x-real-ip" || strings.HasPrefix(lower, "x-forwarded-") || strings.HasPrefix(lower, "x-burrow-") {
			headers.Del(key)
			changed = true
		}
	}
	return changed
}
func redactQuery(raw string) (string, bool, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "", false, err
	}
	changed := false
	for key := range values {
		if secret(key) {
			values[key] = []string{"[REDACTED]"}
			changed = true
		}
	}
	if !changed {
		return raw, false, nil
	}
	return values.Encode(), true, nil
}
