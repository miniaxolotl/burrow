package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"burrow/burrowctl/cmd/tunnel"
	"burrow/burrowctl/internal/clioutput"
	"burrow/burrowctl/internal/config"
	"burrow/protocol/telemetry"

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

func Execute(config telemetry.Config) (err error) {
	return ExecuteContext(context.Background(), config)
}

func ExecuteContext(ctx context.Context, config telemetry.Config) (err error) {
	rootCmd.SetContext(ctx)
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	rootCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return clioutput.Invalid(err) })
	var reporter *telemetry.Reporter
	started := time.Now()
	operation := telemetry.Command
	rootCmd.PersistentPreRunE = func(command *cobra.Command, _ []string) error {
		if err := clioutput.Prepare(command); err != nil {
			return err
		}
		loadClientConfig()
		if configErr != nil {
			return clioutput.Invalid(configErr)
		}
		switch command.Name() {
		case "tunnel":
			operation = telemetry.Command
		case "list":
			operation = telemetry.List
		case "logs":
			operation = telemetry.Logs
		case "close":
			operation = telemetry.Close
		default:
			return nil
		}
		reporter = telemetry.New(config)
		command.SetContext(telemetry.WithObserver(command.Context(), reporter))
		started = time.Now()
		return nil
	}
	defer func() {
		defer reporter.Close()
		if failure := recover(); failure != nil {
			reporter.Operation(operation, started, telemetry.Failure)
			panic(failure)
		}
		outcome := telemetry.Success
		if err != nil {
			outcome = telemetry.Unavailable
		}
		reporter.Operation(operation, started, outcome)
		if err == nil {
			reporter.Track(telemetry.CommandCompleted)
		}
	}()
	command, err := rootCmd.ExecuteC()
	// Transports may wrap Cause(ctx), not Err(ctx). In particular, signal
	// contexts can carry a signal-specific cause instead of context.Canceled.
	if ctx.Err() != nil && (errors.Is(err, ctx.Err()) || errors.Is(err, context.Cause(ctx))) {
		if output := clioutput.FromCommand(command); output.Structured() {
			return output.Emit("stopped", "", map[string]string{"reason": "cancelled"})
		}
		return nil
	}
	if err != nil {
		if writeErr := clioutput.Report(command, err); writeErr != nil {
			fmt.Fprintln(rootCmd.ErrOrStderr(), "Error: command output unavailable")
			return writeErr
		}
	}
	return err
}

func SetVersion(v string) {
	rootCmd.Version = v
}

func init() {
	viper.SetEnvPrefix("BURROW")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	tunnel.AddCommands(rootCmd)
	rootCmd.PersistentFlags().String("output", "text", "Output format: text, json (finite commands), or jsonl")
	rootCmd.AddCommand(telemetryCommand())
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
