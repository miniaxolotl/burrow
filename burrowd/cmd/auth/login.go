package auth

import (
	"fmt"
	"os"

	"burrow/burrowd/internal"
	"burrow/protocol"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var loginCmd = &cobra.Command{
	Use:   "login [token]",
	Short: "Authenticate with a token",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuthLogin,
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	token := args[0]
	secret := viper.GetString("secret")

	if secret == "" {
		return fmt.Errorf("no secret configured: set --secret flag or BURROW_SECRET env var")
	}

	if !protocol.ValidateToken(token, secret) {
		return fmt.Errorf("invalid token")
	}

	configDir := internal.ConfigDir()
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	tokenFile := configDir + "/token"
	if err := os.WriteFile(tokenFile, []byte(token), 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	fmt.Println("Authentication successful")
	return nil
}
