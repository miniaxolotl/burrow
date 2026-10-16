package internal

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

const DefaultServer = "brmbl.dev"

// ServerURL defines both the connection origin and its credential scope.
func ServerURL(server string) (*url.URL, error) {
	u, err := url.Parse("https://" + strings.TrimSpace(server))
	if err != nil || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return nil, fmt.Errorf("invalid server address %q: use hostname[:port]", server)
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "" || (strings.Contains(host, ":") && (net.ParseIP(host) == nil || !strings.HasPrefix(u.Host, "["))) {
		return nil, fmt.Errorf("invalid server host %q", host)
	}
	port := 443
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		u.Scheme, port = "http", 80
	}
	if u.Port() != "" {
		port, err = strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("invalid server port %q", u.Port())
		}
	}
	u.Host = net.JoinHostPort(host, strconv.Itoa(port))
	return u, nil
}
