package auth

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, _, err := readConfig()
	if err != nil {
		fmt.Println("Not authenticated")
		return nil
	}

	if cfg.Token == "" {
		fmt.Println("Not authenticated")
		return nil
	}

	masked := maskToken(cfg.Token)
	fmt.Printf("Authenticated\nToken: %s\n", masked)
	fmt.Printf("Server: %s\n", cfg.Server)
	fmt.Printf("Domain: %s\n", cfg.Domain)
	return nil
}

func maskToken(token string) string {
	if len(token) > 8 {
		return strings.Repeat("*", len(token)-8) + token[len(token)-8:]
	}
	return token
}
