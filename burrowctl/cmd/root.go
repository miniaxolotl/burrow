package cmd

import (
	"strings"

	"burrow/burrowctl/cmd/tunnel"
	"burrow/burrowctl/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:               "burrowctl",
	Short:             "burrowctl - Tunnel client",
	Long:              `burrowctl is the client CLI for creating tunnels to the burrow server.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return configErr },
}

var configErr error

func Execute() error {
	return rootCmd.Execute()
}

func SetVersion(v string) {
	rootCmd.Version = v
}

func init() {
	cobra.OnInitialize(loadClientConfig)

	viper.SetEnvPrefix("BURROW")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	tunnel.AddCommands(rootCmd)
}

func loadClientConfig() {
	var cfgFile string
	cfgFile, configErr = config.EnsureConfig()
	if configErr != nil {
		return
	}

	viper.SetConfigFile(cfgFile)
	viper.SetConfigType("json")
	configErr = viper.ReadInConfig()
}
