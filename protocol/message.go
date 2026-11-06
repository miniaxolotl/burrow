package protocol

import "time"

type TunnelInfo struct {
	TunnelID              string `json:"tunnel_id"`
	Port                  uint16 `json:"port"`
	URL                   string `json:"url"`
	Status                string `json:"status"`
	Latency               string `json:"latency"`
	Reconnects            int    `json:"reconnects"`
	TotalSize             int64  `json:"total_size"`
	Generation            string `json:"generation,omitempty"`
	RegistrationConfirmed bool   `json:"registration_confirmed,omitempty"`
	LifecycleConfirmed    bool   `json:"lifecycle_confirmed,omitempty"`
}

type TunnelList struct {
	Tunnels []*TunnelInfo `json:"tunnels"`
	Count   int           `json:"count"`
}

type TunnelLog struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	Method        string    `json:"method"`
	Path          string    `json:"path"`
	StatusCode    int       `json:"status_code"`
	Size          int64     `json:"size"`
	Duration      string    `json:"duration"`
	IP            string    `json:"ip"`
	Generation    string    `json:"generation,omitempty"`
	CompletedAt   time.Time `json:"completed_at,omitempty"`
	DurationNS    *int64    `json:"duration_ns,omitempty"`
	StatusOrigin  string    `json:"status_origin,omitempty"`
	Outcome       string    `json:"outcome,omitempty"`
	Reason        string    `json:"reason,omitempty"`
	Route         string    `json:"route,omitempty"`
	PathTruncated bool      `json:"path_truncated,omitempty"`
}
