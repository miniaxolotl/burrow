package auth

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var setServerCmd = &cobra.Command{
	Use:   "set-server <address>",
	Short: "Set the burrow server address",
	Args:  cobra.ExactArgs(1),
	RunE:  runSetServer,
}

func runSetServer(cmd *cobra.Command, args []string) error {
	server := args[0]

	cfgFile := filepath.Join(ConfigDir(), "client.json")
	cfg, _, err := readConfig()
	if err != nil {
		cfg = &clientConfig{
			Domain: "burrow.mawa.dev",
		}
	}

	cfg.Server = server
	if err := writeConfig(cfg, cfgFile); err != nil {
		return err
	}

	fmt.Printf("Server set to %s\n", server)
	return nil
}
