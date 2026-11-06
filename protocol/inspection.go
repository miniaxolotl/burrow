package protocol

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

const MaxMetadataEntries = 200
const MaxMetadataBytes = 512 * 1024
const MaxMetadataPath = 2048

// ClipPath never includes query values and keeps valid UTF-8 within the byte cap.
func ClipPath(path string) (string, bool) {
	path, _, _ = strings.Cut(path, "?")
	path = strings.ToValidUTF8(path, "�")
	if len(path) <= MaxMetadataPath {
		return path, false
	}
	n := MaxMetadataPath
	for n > 0 && !utf8.RuneStart(path[n]) {
		n--
	}
	return path[:n], true
}

// AppendMetadata operates on completed immutable records, evicting oldest first.
// Its bounded input makes accounting independent of request/response body size.
func AppendMetadata(rows []*TunnelLog, entry *TunnelLog) ([]*TunnelLog, int) {
	path, truncated := ClipPath(entry.Path)
	entry.Path, entry.PathTruncated = path, entry.PathTruncated || truncated
	rows = append(rows, entry)
	bytes := 0
	for _, row := range rows {
		encoded, _ := json.Marshal(row)
		bytes += len(encoded)
	}
	evicted := 0
	for len(rows) > MaxMetadataEntries || bytes > MaxMetadataBytes {
		encoded, _ := json.Marshal(rows[0])
		bytes -= len(encoded)
		rows = rows[1:]
		evicted++
	}
	return append([]*TunnelLog(nil), rows...), evicted
}
