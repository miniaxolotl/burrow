package tunnel

import (
	"context"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var TunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Tunnel management commands",
	Long:  `Manage tunnels on the burrow server.`,
}

func init() {
	TunnelCmd.PersistentFlags().String("server", "localhost:25701", "Burrow server address")
	TunnelCmd.PersistentFlags().String("token", "", "Authentication token")
	TunnelCmd.PersistentFlags().Bool("tls", false, "Use TLS (https://) when connecting to the server")
	viper.BindPFlag("server", TunnelCmd.PersistentFlags().Lookup("server"))
	viper.BindPFlag("token", TunnelCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("tls", TunnelCmd.PersistentFlags().Lookup("tls"))

	TunnelCmd.AddCommand(listCmd)
	TunnelCmd.AddCommand(revokeCmd)
}

func scheme() string {
	if viper.GetBool("tls") {
		return "https"
	}
	return "http"
}

func apiDo(method, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("X-Tunnel-Token", viper.GetString("token"))
	return http.DefaultClient.Do(req)
}
