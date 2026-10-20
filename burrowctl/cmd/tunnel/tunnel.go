package tunnel

import (
	"fmt"
	"os"

	"burrow/burrowctl/internal"
	"burrow/burrowctl/internal/config"
	"burrow/protocol/telemetry"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var TunnelCmd = &cobra.Command{
	Use:     "tunnel <port> [port...]",
	Short:   "Share local ports and open the tunnel manager",
	Args:    cobra.MinimumNArgs(1),
	Example: "  burrowctl tunnel 3000\n  burrowctl tunnel 3000 8080",
	RunE:    runTunnelCreate,
}

func AddCommands(root *cobra.Command) {
	flags := root.PersistentFlags()
	flags.String("server", internal.DefaultServer, "Burrow server address")
	flags.String("token", "", "Explicit owner credential (generated automatically by default)")
	flags.String("domain", internal.DefaultServer, "Fallback domain for tunnel URLs")
	for _, name := range []string{"server", "token", "domain"} {
		viper.BindPFlag(name, flags.Lookup(name))
	}
	root.AddCommand(TunnelCmd, listCmd, logsCmd, closeCmd)
}

// resolveToken returns the best available auth token.
// Priority follows viper: --token flag > BURROW_TOKEN env > client.json token field.
// Otherwise load or generate the server-scoped private owner credential.
func resolveToken(cmd *cobra.Command, server string) (string, error) {
	// Explicit overrides are intentional; saved credentials are scoped to the
	// configured server, never automatically forwarded to a --server override.
	if cmd.Flags().Changed("token") || os.Getenv("BURROW_TOKEN") != "" {
		if token := viper.GetString("token"); token != "" {
			return token, nil
		}
	}
	token, err := config.StoredToken(server)
	if err != nil {
		return "", err
	}
	if token != "" {
		return token, nil
	}
	return config.OwnerToken(server)
}

func configuredClient(cmd *cobra.Command) (*internal.Client, error) {
	server, err := internal.ServerURL(viper.GetString("server"))
	if err != nil {
		return nil, err
	}
	token, err := resolveToken(cmd, server.Host)
	if err != nil {
		return nil, fmt.Errorf("load owner credential: %w", err)
	}
	return internal.NewClient(server.Host, token, viper.GetString("domain"), server.Scheme == "https", telemetry.FromContext(cmd.Context())), nil
}
