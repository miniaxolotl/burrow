package tunnel

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"burrow/burrowctl/internal"
	"burrow/burrowctl/internal/clioutput"
	"burrow/burrowctl/internal/routing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var doctorCmd = &cobra.Command{
	Use: "doctor <port> [port...]", Short: "Check apex reachability and explicit loopback TCP targets (no app requests or tunnel)",
	RunE: runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	ports := []uint16{}
	if cmd.Flags().Changed("config") {
		if len(args) > 0 {
			return clioutput.Invalid(errors.New("use ports or --config, not both"))
		}
		path, _ := cmd.Flags().GetString("config")
		file, err := os.Open(path)
		if err != nil {
			return clioutput.Invalid(errors.New("route config unavailable"))
		}
		data, err := io.ReadAll(io.LimitReader(file, routing.MaxConfig+1))
		file.Close()
		if err != nil {
			return clioutput.Invalid(err)
		}
		table, err := routing.Parse(data)
		if err != nil {
			return clioutput.Invalid(err)
		}
		for _, route := range table.Routes {
			ports = append(ports, route.Port)
		}
	} else {
		if len(args) == 0 || len(args) > 32 {
			return clioutput.Invalid(errors.New("doctor requires 1–32 explicit ports or --config"))
		}
		for _, arg := range args {
			port, err := strconv.ParseUint(arg, 10, 16)
			if err != nil || port == 0 {
				return clioutput.Invalid(errors.New("ports must be between 1 and 65535"))
			}
			ports = append(ports, uint16(port))
		}
	}
	seen := map[uint16]bool{}
	unique := ports[:0]
	for _, port := range ports {
		if !seen[port] {
			seen[port] = true
			unique = append(unique, port)
		}
	}
	server := viper.GetString("server")
	if _, err := internal.ServerURL(server); err != nil {
		return clioutput.Invalid(err)
	}
	report, err := internal.Diagnose(cmd.Context(), server, unique)
	if cmd.Context().Err() != nil {
		return cmd.Context().Err()
	}
	out := clioutput.FromCommand(cmd)
	if out.Structured() {
		if writeErr := out.Emit("result", "", report); writeErr != nil {
			return writeErr
		}
	} else {
		if _, writeErr := fmt.Fprintf(out, "Server: %s (%s)\n", report.Server.Outcome, report.Server.Reason); writeErr != nil {
			return writeErr
		}
		for _, target := range report.Targets {
			if _, writeErr := fmt.Fprintf(out, "Loopback %d: %s (%s)\n", target.Port, target.Outcome, target.Reason); writeErr != nil {
				return writeErr
			}
		}
		if _, writeErr := fmt.Fprintf(out, "Registration: unavailable in this process; application: not probed.\n%s\n", report.NextSteps); writeErr != nil {
			return writeErr
		}
	}
	return err
}
