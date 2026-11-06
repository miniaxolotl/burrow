package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"burrow/protocol"
)

// Attachment callbacks commit private local state before admission and after the
// matching acknowledgement. They must fail closed on disk failure/corruption.
type Attachment struct {
	TunnelID    string
	Prepare     func() (id, expected, nonce string, err error)
	Acknowledge func(generation, url string) error
	Reject      func(status int) error
}

func (c *Client) SetAttachment(a *Attachment) { c.attachment = a }

func (c *Client) Capabilities(ctx context.Context) (protocol.Capabilities, error) {
	var caps protocol.Capabilities
	scheme := "http"
	if c.secure {
		scheme = "https"
	}
	req, err := http.NewRequestWithContext(ctx, "GET", scheme+"://"+c.server+protocol.CapabilityPath, nil)
	if err != nil {
		return caps, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return caps, errors.New("capability discovery unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return caps, errors.New("server does not support required capabilities")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4097))
	if err != nil || len(data) > 4096 {
		return caps, errors.New("invalid capability response")
	}
	if json.Unmarshal(data, &caps) != nil || caps.Version != 1 {
		return caps, errors.New("invalid capability response")
	}
	return caps, nil
}

func (c *Client) Reservation(id string) (protocol.Reservation, error) {
	var result protocol.Reservation
	err := c.api("GET", fmt.Sprintf("/tunnels/%s/reservation", id), &result)
	return result, err
}
