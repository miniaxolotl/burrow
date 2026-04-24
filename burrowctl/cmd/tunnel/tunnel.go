package tunnel

import (
	"os"
	"strings"

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
	TunnelCmd.PersistentFlags().String("domain", "inkspire.one", "Domain for tunnel URLs")
	viper.BindPFlag("server", TunnelCmd.PersistentFlags().Lookup("server"))
	viper.BindPFlag("token", TunnelCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("secret", TunnelCmd.PersistentFlags().Lookup("secret"))
	viper.BindPFlag("domain", TunnelCmd.PersistentFlags().Lookup("domain"))

	TunnelCmd.AddCommand(createCmd)
	TunnelCmd.AddCommand(listCmd)
	TunnelCmd.AddCommand(statusCmd)
	TunnelCmd.AddCommand(inspectCmd)
	TunnelCmd.AddCommand(closeCmd)
}

// resolveToken returns the best available auth token. Priority:
//  1. --token flag / BURROW_TOKEN env var
//  2. ~/.burrow/token file (written by `burrowd auth login`)
//  3. Generate from --secret / BURROW_SECRET env var
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
