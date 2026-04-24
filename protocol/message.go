package protocol

type TunnelInfo struct {
	TunnelID string `json:"tunnel_id"`
	Port     uint16 `json:"port"`
	URL      string `json:"url"`
	Status   string `json:"status"`
}
