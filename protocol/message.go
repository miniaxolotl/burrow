package protocol

import "time"

type TunnelInfo struct {
	TunnelID   string `json:"tunnel_id"`
	Port       uint16 `json:"port"`
	URL        string `json:"url"`
	Status     string `json:"status"`
	Latency    string `json:"latency"`
	Reconnects int    `json:"reconnects"`
}

type TunnelLog struct {
	Timestamp time.Time `json:"timestamp"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	Duration  string    `json:"duration"`
}
