package tunnel

import (
	"net"
	"os"
	"strings"

	"burrow/burrowctl/internal"
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
	TunnelCmd.PersistentFlags().String("log-level", "info", "Log level: off, info, debug")
	viper.BindPFlag("server", TunnelCmd.PersistentFlags().Lookup("server"))
	viper.BindPFlag("token", TunnelCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("secret", TunnelCmd.PersistentFlags().Lookup("secret"))
	viper.BindPFlag("domain", TunnelCmd.PersistentFlags().Lookup("domain"))
	viper.BindPFlag("tls", TunnelCmd.PersistentFlags().Lookup("tls"))
	viper.BindPFlag("log-level", TunnelCmd.PersistentFlags().Lookup("log-level"))

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

// wsScheme returns "wss" or "ws" based on secureTLS.
func wsScheme() string {
	if secureTLS() {
		return "wss"
	}
	return "ws"
}

// resolveToken returns the best available auth token. Priority:
//  1. --token flag / BURROW_TOKEN env var
//  2. ~/.config/burrow/client.json token field (written by `burrowctl auth login`)
//  3. ~/.burrow/token file (legacy fallback, written by `burrowd auth login`)
//  4. Generate from --secret / BURROW_SECRET env var
func resolveToken() string {
	if t := viper.GetString("token"); t != "" {
		return t
	}
	if home, err := os.UserHomeDir(); err == nil {
		if data, err := os.ReadFile(home + "/.burrow/token"); err == nil {
			if t := strings.TrimSpace(string(data)); t != "" {
				return t
			}
		}
	}
	if secret := viper.GetString("secret"); secret != "" {
		return protocol.GenerateToken(secret)
	}
	return ""
}

func parseLogLevel(s string) internal.LogLevel {
	switch s {
	case "off":
		return internal.LogLevelOff
	case "debug":
		return internal.LogLevelDebug
	default:
		return internal.LogLevelInfo
	}
}
