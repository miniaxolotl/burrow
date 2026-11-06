package protocol

import (
	"encoding/json"
	"errors"
	"io"
)

const RegistrationHeader = "X-Burrow-Registration"
const RegistrationVersion = "1"
const MaxRegistrationSize = 2048

// Registration is the sole message on the first server-opened yamux stream,
// only when explicitly negotiated. The stream is opened before publication and
// written after durable registration and installation. HTTP uses other streams.
type Registration struct {
	Version    int    `json:"version"`
	TunnelID   string `json:"tunnel_id"`
	Generation string `json:"generation"`
}

func ReadRegistration(r io.Reader, tunnelID string) (Registration, error) {
	var ack Registration
	data, err := io.ReadAll(io.LimitReader(r, MaxRegistrationSize+1))
	if err != nil {
		return ack, err
	}
	if len(data) > MaxRegistrationSize {
		return ack, errors.New("registration frame too large")
	}
	if err := json.Unmarshal(data, &ack); err != nil {
		return ack, errors.New("invalid registration frame")
	}
	if ack.Version != 1 || ack.TunnelID != tunnelID || ack.Generation == "" || len(ack.Generation) > 128 {
		return ack, errors.New("registration identity mismatch")
	}
	return ack, nil
}
