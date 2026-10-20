package cmd

import (
	"fmt"

	"burrow/protocol/telemetry"
	"github.com/spf13/cobra"
)

func telemetryCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "telemetry [status|disable|enable]",
		Short:             "Control anonymous analytics and diagnostics",
		Long:              "Official releases send anonymous command/tunnel outcomes and sanitized diagnostics. No arguments, ports, URLs, credentials or tunnel contents are sent. Set DO_NOT_TRACK=1 or BURROW_TELEMETRY=0 to opt out. See https://useburrow.app/privacy.",
		Args:              cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
		ValidArgs:         []string{"status", "disable", "enable"},
		PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
		RunE: func(command *cobra.Command, args []string) error {
			dir := telemetry.StateDir()
			if len(args) != 0 && args[0] != "status" {
				if err := telemetry.SetEnabled(dir, args[0] == "enable"); err != nil {
					return err
				}
			}
			if telemetry.Disabled(dir) {
				fmt.Fprintln(command.OutOrStdout(), "Anonymous telemetry is disabled.")
			} else {
				fmt.Fprintln(command.OutOrStdout(), "Anonymous telemetry is enabled for official releases. Re-enable takes effect on the next invocation.")
			}
			return nil
		},
	}
}
