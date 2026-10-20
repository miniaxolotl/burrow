package main

import (
	"fmt"
	"os"

	"burrow/burrowctl/cmd"
	"burrow/protocol/telemetry"
)

// Public ingestion configuration is linked only into admitted release builds.
var Revision, TelemetryBuild, PostHogKey, SentryDSN string

func main() {
	cmd.SetVersion(Version)
	if err := cmd.Execute(telemetry.Config{Enabled: TelemetryBuild == "production", Surface: "cli", Version: Version, Revision: Revision, Key: PostHogKey, DSN: SentryDSN, StateDir: telemetry.StateDir()}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
