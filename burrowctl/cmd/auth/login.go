package auth

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login [token]",
	Short: "Store an authentication token",
	Args:  cobra.ExactArgs(1),
	RunE:  runLogin,
}

func init() {
	loginCmd.Flags().String("server", "", "Server address to validate token against")
}

func runLogin(cmd *cobra.Command, args []string) error {
	token := args[0]

	cfg, cfgFile, err := readConfig()
	if err != nil {
		cfg = &clientConfig{
			Server: "burrow.mawa.dev",
			Domain: "burrow.mawa.dev",
		}
	}

	serverOverride, _ := cmd.Flags().GetString("server")
	if serverOverride != "" {
		cfg.Server = serverOverride
	}

	cfg.Token = token

	if err := writeConfig(cfg, cfgFile); err != nil {
		return err
	}

	fmt.Println("Authentication successful")
	fmt.Printf("Token stored in %s\n", cfgFile)

	if cfg.Server != "" {
		fmt.Printf("Verifying against %s...", cfg.Server)
		scheme := "https"
		req, err := http.NewRequest("GET", fmt.Sprintf("%s://%s/tunnels", scheme, cfg.Server), nil)
		if err != nil {
			fmt.Println()
			return fmt.Errorf("failed to build request: %w", err)
		}
		req.Header.Set("X-Tunnel-Token", token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println()
			fmt.Printf("Warning: could not reach server: %v\n", err)
			return nil
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
			fmt.Println(" ok")
			if resp.StatusCode == http.StatusUnauthorized {
				fmt.Println("Warning: server rejected the token")
			}
		} else {
			fmt.Printf(" unexpected status: %s\n", resp.Status)
		}
	}

	return nil
}
