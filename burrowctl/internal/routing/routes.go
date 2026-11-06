package routing

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"burrow/burrowctl/internal/strictjson"
)

const MaxConfig = 64 * 1024
const MaxRoutes = 32

type Route struct {
	Path        string `json:"path"`
	Port        uint16 `json:"port"`
	StripPrefix bool   `json:"strip_prefix,omitempty"`
	HostHeader  string `json:"host_header,omitempty"`
}
type Table struct {
	Version int                                            `json:"version"`
	Routes  []Route                                        `json:"routes"`
	Observe func(*http.Request, Route, string) Observation `json:"-"`
}

func Parse(data []byte) (*Table, error) {
	if len(data) > MaxConfig {
		return nil, errors.New("route config exceeds 64 KiB")
	}
	var table Table
	if err := strictjson.Decode(data, &table); err != nil {
		return nil, err
	}
	if err := table.Validate(); err != nil {
		return nil, err
	}
	return &table, nil
}
func (t *Table) Validate() error {
	if t.Version != 1 || len(t.Routes) == 0 || len(t.Routes) > MaxRoutes {
		return errors.New("version 1 requires 1-32 routes")
	}
	seen := map[string]bool{}
	for _, r := range t.Routes {
		if r.Port == 0 || r.Path == "" || !strings.HasPrefix(r.Path, "/") || strings.ContainsAny(r.Path, "%?#\\") || seen[r.Path] || r.Path != "/" && strings.HasSuffix(r.Path, "/") || r.Path == "/" && r.StripPrefix {
			return errors.New("invalid or duplicate route")
		}
		for _, c := range r.Path {
			if c < 33 || c > 126 {
				return errors.New("route prefix must be ASCII")
			}
		}
		if err := validPath(r.Path); err != nil {
			return err
		}
		if r.HostHeader != "" {
			if err := validHost(r.HostHeader); err != nil {
				return err
			}
		}
		seen[r.Path] = true
	}
	return nil
}
func validHost(host string) error {
	if len(host) > 255 || strings.ContainsAny(host, "@/?#\\ \t\r\n") {
		return errors.New("invalid host_header authority")
	}
	for _, c := range host {
		if c < 33 || c > 126 {
			return errors.New("invalid host_header")
		}
	}
	u, err := url.Parse("http://" + host)
	if err != nil || u.Host != host || u.Hostname() == "" || u.User != nil {
		return errors.New("invalid host_header authority")
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") || strings.Contains(host, "]:") {
		_, port, err := net.SplitHostPort(host)
		if err != nil {
			return errors.New("invalid host_header port")
		}
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return errors.New("invalid host_header port")
		}
	}
	return nil
}
func validPath(path string) error {
	if !strings.HasPrefix(path, "/") || strings.Contains(path, "//") || strings.ContainsAny(path, "\\\x00?#") {
		return errors.New("unsafe request path")
	}
	lower := strings.ToLower(path)
	if strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c") {
		return errors.New("encoded path separator")
	}
	decoded, err := url.PathUnescape(path)
	if err != nil || strings.ContainsAny(decoded, "\\\x00") {
		return errors.New("invalid request path escape")
	}
	for _, segment := range strings.Split(decoded, "/") {
		if segment == "." || segment == ".." {
			return errors.New("dot path segment")
		}
	}
	return nil
}
func (t *Table) Match(path string) (Route, string, error) {
	if err := validPath(path); err != nil {
		return Route{}, "", err
	}
	var selected Route
	for _, r := range t.Routes {
		if r.Path == "/" || path == r.Path || strings.HasPrefix(path, r.Path+"/") {
			if len(r.Path) > len(selected.Path) {
				selected = r
			}
		}
	}
	if selected.Path == "" {
		return Route{}, path, nil
	}
	if selected.StripPrefix {
		path = strings.TrimPrefix(path, selected.Path)
		if path == "" {
			path = "/"
		}
	}
	return selected, path, nil
}
