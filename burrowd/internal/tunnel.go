package internal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"burrow/protocol"
)

type TunnelRegistry struct {
	redis  *RedisClient
	domain string
	secure bool
}

func NewTunnelRegistry(redis *RedisClient, domain string, secure bool) *TunnelRegistry {
	return &TunnelRegistry{redis: redis, domain: domain, secure: secure}
}

func (t *TunnelRegistry) tunnelURL(tunnelID string) string {
	scheme := "http"
	if t.secure {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s.%s", scheme, tunnelID, t.domain)
}

func (t *TunnelRegistry) Register(ctx context.Context, tunnelID string, port uint16) (string, error) {
	if err := t.redis.SetTunnel(ctx, tunnelID, port, 24*time.Hour); err != nil {
		return "", fmt.Errorf("failed to register tunnel: %w", err)
	}
	return t.tunnelURL(tunnelID), nil
}

func (t *TunnelRegistry) Remove(tunnelID string) error {
	return t.redis.DeleteTunnel(context.Background(), tunnelID)
}

func (t *TunnelRegistry) List(ctx context.Context) ([]*protocol.TunnelInfo, error) {
	tunnels, err := t.redis.ListTunnels(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*protocol.TunnelInfo, 0, len(tunnels))
	for _, td := range tunnels {
		result = append(result, &protocol.TunnelInfo{
			TunnelID: td.ID,
			Port:     td.Port,
			URL:      t.tunnelURL(td.ID),
			Status:   "active",
		})
	}
	return result, nil
}

func ParseTunnelID(host, domain string) string {
	suffix := "." + domain
	if !strings.HasSuffix(host, suffix) {
		return ""
	}
	return strings.TrimSuffix(host, suffix)
}
