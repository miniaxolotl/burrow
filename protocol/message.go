package protocol

import "time"

type TunnelInfo struct {
	TunnelID   string `json:"tunnel_id"`
	Port       uint16 `json:"port"`
	URL        string `json:"url"`
	Status     string `json:"status"`
	Latency    string `json:"latency"`
	Reconnects int    `json:"reconnects"`
	TotalSize  int64  `json:"total_size"`
}

type TunnelLog struct {
	Timestamp   time.Time `json:"timestamp"`
	Method      string    `json:"method"`
	Path        string    `json:"path"`
	StatusCode  int       `json:"status_code"`
	Size        int64     `json:"size"`
	Duration    string    `json:"duration"`
	IP          string    `json:"ip"`
}
