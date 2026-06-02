package internal

import (
	"context"
	"fmt"
)

func CloseTunnel(ctx context.Context, pg *PostgresClient, registry *TunnelRegistry, tunnelID string) error {
	if err := registry.Remove(tunnelID); err != nil {
		return fmt.Errorf("remove tunnel: %w", err)
	}
	return pg.UpdateTunnelStatus(ctx, tunnelID, "stopped")
}

func RestartTunnel(ctx context.Context, pg *PostgresClient, registry *TunnelRegistry, tunnelID string) error {
	tunnel, err := pg.GetTunnelByTunnelID(ctx, tunnelID)
	if err != nil {
		return fmt.Errorf("get tunnel: %w", err)
	}
	if tunnel.Status == "active" {
		return nil
	}
	_, err = pg.CreateTunnel(ctx, tunnel.TunnelID, tunnel.Port, tunnel.ClientIP, tunnel.UserID)
	if err != nil {
		return fmt.Errorf("create tunnel: %w", err)
	}
	return nil
}
