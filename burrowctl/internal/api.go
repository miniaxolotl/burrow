package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"burrow/protocol"
)

// Both the interactive manager and standalone commands use this owner-scoped
// API, including the same credential and redirect policy.
func (c *Client) api(method, path string, result any) error {
	scheme := "http"
	if c.secure {
		scheme = "https"
	}
	req, err := http.NewRequest(method, fmt.Sprintf("%s://%s%s", scheme, c.server, path), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Tunnel-Token", c.token)
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to contact server: %w", err)
	}
	defer resp.Body.Close()
	want := http.StatusOK
	if method == http.MethodDelete {
		want = http.StatusNoContent
	}
	if resp.StatusCode != want {
		return fmt.Errorf("server returned %s", resp.Status)
	}
	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (c *Client) GetLogs(id string) ([]*protocol.TunnelLog, error) {
	var logs []*protocol.TunnelLog
	err := c.api(http.MethodGet, "/tunnels/"+url.PathEscape(id)+"/logs", &logs)
	return logs, err
}

func (c *Client) RemoteTunnels() (*protocol.TunnelList, error) {
	var result protocol.TunnelList
	err := c.api(http.MethodGet, "/tunnels", &result)
	return &result, err
}

func (c *Client) DeleteTunnel(id string) error {
	return c.api(http.MethodDelete, "/tunnels/"+url.PathEscape(id), nil)
}
