package auth

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored authentication token",
	RunE:  runAuthLogout,
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	tokenFile := getConfigDir() + "/token"
	if _, err := os.Stat(tokenFile); os.IsNotExist(err) {
		fmt.Println("No token stored")
		return nil
	}
	if err := os.Remove(tokenFile); err != nil {
		return fmt.Errorf("failed to remove token: %w", err)
	}
	fmt.Println("Logged out successfully")
	return nil
}
