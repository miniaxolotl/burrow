package auth

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	RunE:  runAuthStatus,
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	tokenFile := getConfigDir() + "/token"
	data, err := os.ReadFile(tokenFile)
	if os.IsNotExist(err) {
		fmt.Println("Not authenticated")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}

	token := strings.TrimSpace(string(data))
	if len(token) > 8 {
		masked := strings.Repeat("*", len(token)-8) + token[len(token)-8:]
		fmt.Printf("Authenticated\nToken: %s\n", masked)
	} else {
		fmt.Printf("Authenticated\nToken: %s\n", token)
	}
	return nil
}
