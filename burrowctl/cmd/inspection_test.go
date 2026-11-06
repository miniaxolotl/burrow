package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"burrow/protocol"
)

func TestLogsFiltersBeforeTailAndRetainsArrayShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]*protocol.TunnelLog{
			{ID: "a", Method: "POST", Path: "/api/hook", StatusCode: 500, Duration: "300ms"},
			{ID: "b", Method: "GET", Path: "/apiculture", StatusCode: 200, Duration: "1ms"},
			{ID: "c", Method: "POST", Path: "/api/latest", StatusCode: 503, Duration: "400ms"},
			{ID: "d", Method: "GET", Path: "/other", StatusCode: 200, Duration: "1ms"},
		})
	}))
	defer server.Close()
	command, _ := workflowCommand(t, "logs", "probe", "--method", "POST", "--status", "5xx", "--path", "/api", "--min-duration", "250ms", "--tail", "1", "--output", "json", "--server", strings.TrimPrefix(server.URL, "http://"))
	out, err := command.CombinedOutput()
	if err != nil || !bytes.Contains(out, []byte(`"id":"c"`)) || bytes.Contains(out, []byte(`"id":"a"`)) || !bytes.Contains(out, []byte(`"data":[`)) {
		t.Fatalf("filtered tail: %v %s", err, out)
	}
}
