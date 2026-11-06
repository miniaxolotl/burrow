package inspection

import (
	"testing"
	"time"

	"burrow/protocol"
)

func TestQueryANDOrUnknownSegmentAndOrdering(t *testing.T) {
	rows := []*protocol.TunnelLog{{ID: "a", Method: "POST", Path: "/api/hook", StatusCode: 503, Duration: "300ms"}, {ID: "b", Method: "GET", Path: "/apiculture", StatusCode: 500, Duration: "400ms"}, {ID: "c", Method: "POST", Path: "/api/v2", StatusCode: 0, Duration: ""}, {ID: "d", Method: "POST", Path: "/api/v2", StatusCode: 404, Duration: "200ms"}}
	minimum := 250 * time.Millisecond
	q := Query{Methods: []string{"POST", "PATCH"}, Statuses: []string{"5xx", "404"}, Path: "/api", Minimum: &minimum, Search: "HOOK"}
	if err := q.Validate("server"); err != nil {
		t.Fatal(err)
	}
	result := q.Apply(rows)
	if len(result) != 1 || result[0].ID != "a" {
		t.Fatalf("wrong scoped filters: %+v", result)
	}
	result = (Query{Statuses: []string{"unknown"}}).Apply(rows)
	if len(result) != 1 || result[0].ID != "c" {
		t.Fatal("unknown matched a known code")
	}
	result = (Query{Sort: "status", Descending: true}).Apply(rows)
	if result[len(result)-1].ID != "c" {
		t.Fatal("descending moved unknown first")
	}
	result = (Query{Sort: "duration"}).Apply(rows)
	if result[0].ID != "d" || result[len(result)-1].ID != "c" {
		t.Fatal("duration ordering incorrect")
	}
	if Safe("\x1b]0;private\a/api\x1b[31m界\n") != "/api界" {
		t.Fatal("terminal sequences not sanitized")
	}
}

func TestQueryRejectsUnsupportedAndUnboundedInput(t *testing.T) {
	for _, tc := range []struct {
		q      Query
		source string
	}{{Query{Route: "/api"}, "server"}, {Query{Statuses: []string{"2xx"}}, "capture"}, {Query{Methods: []string{"GET\n"}}, "server"}, {Query{Statuses: []string{"100"}}, "server"}, {Query{Search: string(make([]byte, 257))}, "local"}, {Query{Descending: true}, "local"}} {
		if tc.q.Validate(tc.source) == nil {
			t.Fatalf("invalid query accepted: %+v", tc)
		}
	}
}
