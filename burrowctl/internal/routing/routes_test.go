package routing

import (
	"strings"
	"testing"
)

func TestStrictRouteConfigAndEscapedDecisions(t *testing.T) {
	table, err := Parse([]byte(`{"version":1,"routes":[{"path":"/api","port":8080,"strip_prefix":true},{"path":"/","port":3000}]}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []struct {
		path    string
		port    uint16
		forward string
		bad     bool
	}{
		{"/api", 8080, "/", false}, {"/api/", 8080, "/", false}, {"/api/orders", 8080, "/orders", false},
		{"/apiculture", 3000, "/apiculture", false}, {"/API", 3000, "/API", false}, {"/%61pi/orders", 3000, "/%61pi/orders", false},
		{"/api%2forders", 0, "", true}, {"/api/../admin", 0, "", true}, {"/api/%2e%2e/admin", 0, "", true}, {"/api//orders", 0, "", true},
		{"/api/%00", 0, "", true}, {"/api/%5Cfoo", 0, "", true}, {"/api/%xy", 0, "", true},
	} {
		route, path, err := table.Match(s.path)
		if s.bad {
			if err == nil {
				t.Errorf("accepted %q", s.path)
			}
			continue
		}
		if err != nil || route.Port != s.port || path != s.forward {
			t.Errorf("%s => %+v %s %v", s.path, route, path, err)
		}
	}
}

func TestRouteConfigRejectsAmbiguityBeforeExposure(t *testing.T) {
	for _, input := range []string{
		`{"version":1,"version":1,"routes":[{"path":"/","port":3000}]}`,
		`{"version":1,"routes":[{"path":"/","port":3000,"port":3001}]}`,
		`{"version":1,"routes":[{"path":"/","port":3000,"Port":3001}]}`,
		`{"Version":1,"routes":[{"path":"/","port":3000}]}`,
		`{"version":2,"routes":[{"path":"/","port":3000}]}`,
		`{"version":1,"routes":[]}`,
		`{"version":1,"routes":[{"path":"/","port":0}]}`,
		`{"version":1,"routes":[{"path":"/","port":3000,"host":"remote"}]}`,
		`{"version":1,"routes":[{"path":"/","port":3000,"strip_prefix":true}]}`,
		`{"version":1,"routes":[{"path":"/api/","port":3000}]}`,
		`{"version":1,"routes":[{"path":"/%61pi","port":3000}]}`,
		`{"version":1,"routes":[{"path":"/api","port":3000},{"path":"/api","port":3001}]}`,
		`{"version":1,"routes":[{"path":"/","port":3000,"host_header":"user@remote"}]}`,
		strings.Repeat(" ", 65537),
	} {
		if _, err := Parse([]byte(input)); err == nil {
			t.Errorf("accepted %s", input)
		}
	}
}
