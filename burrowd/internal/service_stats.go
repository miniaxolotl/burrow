package internal

import (
	"context"
	"time"
)

func GetDashboardStats(ctx context.Context, pg *PostgresClient) (*DashboardStats, error) {
	tunnels, err := pg.ListActiveTunnels(ctx)
	if err != nil {
		return nil, err
	}
	stats := &DashboardStats{
		ActiveTunnels: len(tunnels),
	}
	return stats, nil
}

func GetActiveTunnels(ctx context.Context, pg *PostgresClient) ([]*Tunnel, error) {
	return pg.ListActiveTunnels(ctx)
}

func GetTunnelDetail(ctx context.Context, pg *PostgresClient, tunnelID string) (*Tunnel, error) {
	return pg.GetTunnelByTunnelID(ctx, tunnelID)
}

func GetTunnelStats(ctx context.Context, pg *PostgresClient, tunnelID string, from, to time.Time) ([]*TunnelStat, error) {
	return pg.GetTunnelStats(ctx, tunnelID, from, to)
}

func GetTunnelLogs(ctx context.Context, pg *PostgresClient, tunnelID string, from, to time.Time, limit, offset int) ([]*TunnelLog, error) {
	return pg.GetTunnelLogs(ctx, tunnelID, from, to, limit, offset)
}