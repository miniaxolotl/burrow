package tunnel

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func resolveRedisURL(cmd *cobra.Command) string {
	url := viper.GetString("redis-url")
	if !cmd.Flags().Changed("redis-url") {
		if _, ok := os.LookupEnv("BURROW_REDIS_URL"); !ok {
			if u := os.Getenv("REDIS_URL"); u != "" {
				return u
			}
		}
	}
	return url
}

var TunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Tunnel management commands",
	Long:  `Manage tunnels on the burrow server.`,
}

func init() {
	TunnelCmd.PersistentFlags().String("redis-url", "localhost:6379", "Redis connection URL")
	TunnelCmd.PersistentFlags().String("domain", "inkspire.one", "Domain for tunnel URLs")
	viper.BindPFlag("redis-url", TunnelCmd.PersistentFlags().Lookup("redis-url"))
	viper.BindPFlag("domain", TunnelCmd.PersistentFlags().Lookup("domain"))

	TunnelCmd.AddCommand(listCmd)
	TunnelCmd.AddCommand(revokeCmd)
}
