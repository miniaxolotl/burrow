package protocol

import (
	"bufio"
	"strings"
	"testing"
)

func TestLifecycleFramesAreBoundedAndGenerationScoped(t *testing.T) {
	for _, tc := range []struct {
		data, kind string
		valid      bool
	}{
		{`{"version":1,"type":"hello","tunnel_id":"a","generation":"g"}`, "hello", true},
		{`{"version":1,"type":"draining","tunnel_id":"a","generation":"g","reason":"server_shutdown","remaining_ms":15,"notice_id":"one"}`, "draining", true},
		{`{"version":1,"type":"hello","tunnel_id":"a","generation":"old"}`, "hello", false},
		{`{"version":1,"type":"hello","tunnel_id":"a","generation":"g"}`, "draining", false},
		{`{"version":1,"type":"draining","tunnel_id":"a","generation":"g","reason":"deployment","remaining_ms":15,"notice_id":"one"}`, "draining", false},
		{`{"version":1,"type":"draining","tunnel_id":"a","generation":"g","reason":"server_shutdown","remaining_ms":15001,"notice_id":"one"}`, "draining", false},
		{strings.Repeat("x", MaxLifecycleSize+1), "hello", false},
		{"not-json", "hello", false},
	} {
		_, err := ReadLifecycle(bufio.NewReaderSize(strings.NewReader(tc.data+"\n"), MaxLifecycleSize), "a", "g", tc.kind)
		if (err == nil) != tc.valid {
			t.Fatalf("valid=%v err=%v data=%q", tc.valid, err, tc.data)
		}
	}
	if _, err := ReadLifecycle(bufio.NewReader(strings.NewReader(`{"version":1}`)), "a", "g", "hello"); err == nil {
		t.Fatal("unterminated frame accepted")
	}
}
