package tunnel

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"burrow/protocol"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var TunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Tunnel management commands",
	Long:  `Create and manage tunnels.`,
}

func init() {
	TunnelCmd.PersistentFlags().String("server", "localhost:25701", "Burrow server address")
	TunnelCmd.PersistentFlags().String("token", "", "Authentication token")
	TunnelCmd.PersistentFlags().String("secret", "", "Shared secret (generates a token)")
	TunnelCmd.PersistentFlags().String("domain", "burrow.mawa.dev", "Domain for tunnel URLs")
	TunnelCmd.PersistentFlags().Bool("tls", false, "Use TLS (wss:// and https://) when connecting to the server")
	viper.BindPFlag("server", TunnelCmd.PersistentFlags().Lookup("server"))
	viper.BindPFlag("token", TunnelCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("secret", TunnelCmd.PersistentFlags().Lookup("secret"))
	viper.BindPFlag("domain", TunnelCmd.PersistentFlags().Lookup("domain"))
	viper.BindPFlag("tls", TunnelCmd.PersistentFlags().Lookup("tls"))

	TunnelCmd.AddCommand(createCmd)
	TunnelCmd.AddCommand(listCmd)
	TunnelCmd.AddCommand(statusCmd)
	TunnelCmd.AddCommand(inspectCmd)
	TunnelCmd.AddCommand(closeCmd)
}

// secureTLS returns true when TLS should be used. Explicit --tls flag or
// BURROW_TLS=true takes priority; otherwise TLS is auto-enabled for any
// server that is not localhost / 127.0.0.1 / ::1.
func secureTLS() bool {
	if f := TunnelCmd.PersistentFlags().Lookup("tls"); f != nil && f.Changed {
		return viper.GetBool("tls")
	}
	if viper.GetBool("tls") {
		return true
	}
	host := viper.GetString("server")
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return false
	}
	return true
}

// httpScheme returns "https" or "http" based on secureTLS.
func httpScheme() string {
	if secureTLS() {
		return "https"
	}
	return "http"
}

// resolveToken returns the best available auth token.
// Priority follows viper: --token flag > BURROW_TOKEN env > client.json token field.
// Falls back to generating a token from --secret / BURROW_SECRET if no token is set.
func resolveToken() string {
	if t := viper.GetString("token"); t != "" {
		return t
	}
	if secret := viper.GetString("secret"); secret != "" {
		return protocol.GenerateToken(secret)
	}
	return ""
}

// apiDo performs an authenticated HTTP request to the burrow server.
func apiDo(method, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("X-Tunnel-Token", resolveToken())
	return http.DefaultClient.Do(req)
}
