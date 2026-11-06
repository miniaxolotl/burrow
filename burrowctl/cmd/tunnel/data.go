package tunnel

import (
	"time"

	"burrow/protocol"
)

// CLI v1 uses explicit numeric units without changing the existing server API.
// Unknown measurements/statuses remain null rather than fabricating a zero/200.
type tunnelData struct {
	TunnelID       string `json:"tunnel_id"`
	Port           uint16 `json:"port"`
	URL            string `json:"url"`
	Status         string `json:"status"`
	LatencyMS      *int64 `json:"latency_ms"`
	Reconnects     int    `json:"reconnects"`
	TotalSizeBytes int64  `json:"total_size_bytes"`
}

func listData(result *protocol.TunnelList) any {
	items := make([]tunnelData, 0, len(result.Tunnels))
	for _, entry := range result.Tunnels {
		items = append(items, tunnelData{entry.TunnelID, entry.Port, entry.URL, entry.Status, milliseconds(entry.Latency), entry.Reconnects, entry.TotalSize})
	}
	return struct {
		Tunnels []tunnelData `json:"tunnels"`
		Count   int          `json:"count"`
	}{items, result.Count}
}

type logData struct {
	ID         string    `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	StatusCode *int      `json:"status_code"`
	SizeBytes  int64     `json:"size_bytes"`
	DurationMS *int64    `json:"duration_ms"`
	IP         string    `json:"ip"`
}

func requestData(entry *protocol.TunnelLog) logData {
	var status *int
	if entry.StatusCode >= 100 && entry.StatusCode <= 599 {
		status = &entry.StatusCode
	}
	return logData{entry.ID, entry.Timestamp.UTC(), entry.Method, entry.Path, status, entry.Size, milliseconds(entry.Duration), entry.IP}
}

func requestsData(entries []*protocol.TunnelLog) []logData {
	items := make([]logData, 0, len(entries))
	for _, entry := range entries {
		items = append(items, requestData(entry))
	}
	return items
}

func milliseconds(value string) *int64 {
	duration, err := time.ParseDuration(value)
	if err != nil || duration < 0 {
		return nil
	}
	valueMS := duration.Milliseconds()
	return &valueMS
}
