package tunnel

import (
	"fmt"
	"os"

	"burrow/burrowctl/internal"
	"burrow/burrowctl/internal/clioutput"
	"burrow/burrowctl/internal/config"
	"burrow/protocol/telemetry"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var TunnelCmd = &cobra.Command{
	Use:   "tunnel <port> [port...]",
	Short: "Share local ports and open the tunnel manager",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && !cmd.Flags().Changed("config") {
			return fmt.Errorf("a port or --config is required")
		}
		return nil
	},
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
	for _, command := range []*cobra.Command{TunnelCmd, listCmd, logsCmd, closeCmd} {
		validate := command.Args
		command.Args = func(cmd *cobra.Command, args []string) error {
			if err := validate(cmd, args); err != nil {
				return clioutput.Invalid(err)
			}
			return nil
		}
	}
	root.AddCommand(TunnelCmd, listCmd, logsCmd, closeCmd, savedCmd, doctorCmd)
	doctorCmd.Flags().String("config", "", "Explicit route configuration; TCP connect/close only (visible to the app)")
	addCaptureCommands(root)
	TunnelCmd.Flags().Bool("headless", false, "Run without the interactive manager")
	TunnelCmd.Flags().String("name", "", "Private saved alias (not a public subdomain)")
	TunnelCmd.Flags().String("config", "", "Explicit versioned route configuration file")
	TunnelCmd.Flags().Bool("capture", false, "Explicitly capture bounded requests in encrypted local storage")
	TunnelCmd.Flags().Bool("inspect", false, "Keep bounded in-memory HTTP metadata (strict dispatcher; no capture/replay)")
	TunnelCmd.Flags().StringSlice("capture-exclude", nil, "Do not capture these explicit path prefixes")
	TunnelCmd.Flags().Bool("replace", false, "Replace a confirmed closed alias with a new URL")
	TunnelCmd.Flags().Bool("yes", false, "Confirm the selected replacement")
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
	if cmd.Name() == "saved" || cmd.Name() == "replay" || cmd.Parent() != nil && (cmd.Parent().Name() == "requests" || cmd.Parent().Name() == "replay") {
		return config.ExistingOwnerToken(server)
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
	return internal.NewClientWithContext(cmd.Context(), server.Host, token, viper.GetString("domain"), server.Scheme == "https", telemetry.FromContext(cmd.Context())), nil
}
