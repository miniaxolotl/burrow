package protocol

import (
	"bufio"
	"encoding/json"
	"errors"
)

const LifecycleHeader = "X-Burrow-Lifecycle"
const MaxLifecycleSize = 2048

// Lifecycle frames are newline-delimited on the separately negotiated second
// server-opened stream. Registration v1's object-plus-EOF is unchanged.
type Lifecycle struct {
	Version     int    `json:"version"`
	Type        string `json:"type"`
	TunnelID    string `json:"tunnel_id"`
	Generation  string `json:"generation"`
	Reason      string `json:"reason,omitempty"`
	RemainingMS int64  `json:"remaining_ms,omitempty"`
	NoticeID    string `json:"notice_id,omitempty"`
}

func ReadLifecycle(r *bufio.Reader, tunnelID, generation, kind string) (Lifecycle, error) {
	var frame Lifecycle
	data, err := r.ReadSlice('\n')
	if err != nil || len(data) > MaxLifecycleSize {
		return frame, errors.New("invalid or oversized lifecycle frame")
	}
	if err := json.Unmarshal(data, &frame); err != nil {
		return frame, errors.New("invalid lifecycle JSON")
	}
	if frame.Version != 1 || frame.TunnelID != tunnelID || frame.Generation != generation || frame.Type != kind {
		return frame, errors.New("lifecycle identity or ordering mismatch")
	}
	if kind == "hello" {
		if frame.Reason != "" || frame.NoticeID != "" || frame.RemainingMS != 0 {
			return frame, errors.New("invalid lifecycle hello")
		}
	} else if kind != "draining" || frame.Reason != "server_shutdown" || frame.NoticeID == "" || len(frame.NoticeID) > 128 || frame.RemainingMS < 0 || frame.RemainingMS > 15000 {
		return frame, errors.New("invalid lifecycle notice")
	}
	return frame, nil
}
