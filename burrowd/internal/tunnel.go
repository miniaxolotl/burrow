package internal

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
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

func (t *TunnelRegistry) Register(ctx context.Context, tunnelID string, port uint16, ownerHash, sessionID string) (string, error) {
	if err := t.redis.SetTunnel(ctx, tunnelID, port, ownerHash, sessionID, 24*time.Hour); err != nil {
		return "", fmt.Errorf("failed to register tunnel: %w", err)
	}
	return t.tunnelURL(tunnelID), nil
}

func (t *TunnelRegistry) Get(ctx context.Context, tunnelID string) (*TunnelData, error) {
	return t.redis.GetTunnel(ctx, tunnelID)
}

func (t *TunnelRegistry) Remove(ctx context.Context, tunnelID, sessionID string) error {
	return t.redis.DeleteTunnel(ctx, tunnelID, sessionID)
}

func (t *TunnelRegistry) Refresh(ctx context.Context, tunnelID, sessionID string) error {
	return t.redis.RefreshTunnel(ctx, tunnelID, sessionID, 24*time.Hour)
}

func (t *TunnelRegistry) Revoke(ctx context.Context, tunnelID, ownerHash string) error {
	return t.redis.RevokeTunnel(ctx, tunnelID, ownerHash)
}

func ParseTunnelID(host, domain string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	suffix := "." + domain
	if !strings.HasSuffix(host, suffix) {
		return ""
	}
	return strings.TrimSuffix(host, suffix)
}
