package auth

import (
	"fmt"

	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored authentication token",
	RunE:  runLogout,
}

func runLogout(cmd *cobra.Command, args []string) error {
	cfg, cfgFile, err := readConfig()
	if err != nil {
		fmt.Println("No token stored")
		return nil
	}

	if cfg.Token == "" {
		fmt.Println("No token stored")
		return nil
	}

	cfg.Token = ""
	if err := writeConfig(cfg, cfgFile); err != nil {
		return err
	}

	fmt.Println("Logged out successfully")
	return nil
}
