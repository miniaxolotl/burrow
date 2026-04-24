package tunnel

import (
	"context"
	"fmt"

	"burrow/burrowd/internal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active tunnels",
	RunE:  runTunnelList,
}

func runTunnelList(cmd *cobra.Command, args []string) error {
	redisURL := viper.GetString("redis-url")
	domain := viper.GetString("domain")

	client, err := internal.NewRedisClient(redisURL)
	if err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}
	defer client.Close()

	registry := internal.NewTunnelRegistry(client, domain)
	tunnels, err := registry.List(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list tunnels: %w", err)
	}

	if len(tunnels) == 0 {
		fmt.Println("No active tunnels")
		return nil
	}

	fmt.Printf("%-30s %-8s %s\n", "TUNNEL ID", "PORT", "URL")
	for _, t := range tunnels {
		fmt.Printf("%-30s %-8d %s\n", t.TunnelID, t.Port, t.URL)
	}
	return nil
}
