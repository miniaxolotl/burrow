package tunnel

import (
	"context"
	"fmt"

	"burrow/burrowd/internal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var revokeCmd = &cobra.Command{
	Use:   "revoke [tunnel_id]",
	Short: "Force-close a specific tunnel",
	Args:  cobra.ExactArgs(1),
	RunE:  runTunnelRevoke,
}

func runTunnelRevoke(cmd *cobra.Command, args []string) error {
	tunnelID := args[0]
	redisURL := viper.GetString("redis-url")

	client, err := internal.NewRedisClient(redisURL)
	if err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}
	defer client.Close()

	if err := client.DeleteTunnel(context.Background(), tunnelID); err != nil {
		return fmt.Errorf("failed to revoke tunnel %s: %w", tunnelID, err)
	}

	fmt.Printf("Tunnel %s revoked\n", tunnelID)
	return nil
}
