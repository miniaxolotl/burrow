package auth

import (
	"fmt"
	"os"
	"strings"

	"burrow/burrowd/internal"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	RunE:  runAuthStatus,
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	tokenFile := internal.ConfigDir() + "/token"
	data, err := os.ReadFile(tokenFile)
	if os.IsNotExist(err) {
		fmt.Println("Not authenticated")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}

	token := strings.TrimSpace(string(data))
	maskedLen := len(token) - 4
	if maskedLen > 0 {
		masked := strings.Repeat("*", maskedLen) + token[4:]
		fmt.Printf("Authenticated\nToken: %s\n", masked)
	} else {
		fmt.Printf("Authenticated\nToken: %s\n", token)
	}
	return nil
}
