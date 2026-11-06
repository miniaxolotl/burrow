package protocol

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestMetadataBoundsEntriesBytesAndUTF8(t *testing.T) {
	var rows []*TunnelLog
	for range 300 {
		rows, _ = AppendMetadata(rows, &TunnelLog{ID: "opaque", Method: "POST", Path: "/" + strings.Repeat("界", 3000) + "?token=private"})
	}
	if len(rows) > MaxMetadataEntries {
		t.Fatal("entry cap exceeded")
	}
	size := 0
	for _, row := range rows {
		data, _ := json.Marshal(row)
		size += len(data)
		if len(row.Path) > 2048 || !utf8.ValidString(row.Path) || strings.Contains(row.Path, "private") || !row.PathTruncated {
			t.Fatal("unsafe path retention")
		}
	}
	if size > MaxMetadataBytes {
		t.Fatal("byte cap exceeded")
	}
	rows, _ = AppendMetadata(nil, &TunnelLog{Path: "/already-clipped", PathTruncated: true})
	if !rows[0].PathTruncated {
		t.Fatal("existing truncation evidence was erased")
	}
}
