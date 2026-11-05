package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"burrow/burrowctl/cmd/tui"
	"burrow/burrowctl/internal"
	"burrow/burrowctl/internal/clioutput"
	"burrow/burrowctl/internal/config"
	"burrow/burrowctl/internal/routing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func runTunnelCreate(cmd *cobra.Command, args []string) error {
	ports := make([]int, 0, len(args))
	for _, arg := range args {
		port, err := strconv.Atoi(arg)
		if err != nil {
			return clioutput.Invalid(fmt.Errorf("invalid port %q", arg))
		}
		ports = append(ports, port)
	}
	seen := make(map[int]bool)
	for _, port := range ports {
		if port < 1 || port > 65535 {
			return clioutput.Invalid(fmt.Errorf("port must be between 1 and 65535: %d", port))
		}
		if seen[port] {
			return clioutput.Invalid(fmt.Errorf("port specified more than once: %d", port))
		}
		seen[port] = true
	}
	var routes *routing.Table
	if cmd.Flags().Changed("config") {
		if len(ports) != 0 {
			return clioutput.Invalid(errors.New("positional ports cannot be combined with route config"))
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
		routes, err = routing.Parse(data)
		if err != nil {
			return clioutput.Invalid(err)
		}
		ports = []int{int(routes.Routes[0].Port)}
	}
	name, _ := cmd.Flags().GetString("name")
	replace, _ := cmd.Flags().GetBool("replace")
	yes, _ := cmd.Flags().GetBool("yes")
	if cmd.Flags().Changed("name") {
		if err := config.ValidateAlias(name); err != nil {
			return clioutput.Invalid(err)
		}
	}
	if (name != "" && len(ports) != 1) || (replace && name == "") {
		return clioutput.Invalid(errors.New("named mode requires one tunnel; --replace requires --name"))
	}
	server := viper.GetString("server")
	client, err := configuredClient(cmd)
	if err != nil {
		return err
	}
	client.SetRoutes(routes)
	if routes != nil && !clioutput.FromCommand(cmd).Structured() {
		for _, route := range routes.Routes {
			if _, err := fmt.Fprintf(clioutput.FromCommand(cmd), "route %s -> loopback:%d (strip_prefix=%t, host_header=%q)\n", route.Path, route.Port, route.StripPrefix, route.HostHeader); err != nil {
				return err
			}
		}
	}
	if name != "" {
		ctx, stop := context.WithTimeout(cmd.Context(), 5*time.Second)
		caps, err := client.Capabilities(ctx)
		stop()
		if err != nil {
			return err
		}
		if caps.PersistentAttachment != 1 || caps.Registration != 1 {
			return errors.New("persistent attachments unavailable on server")
		}
		owner, err := resolveToken(cmd, server)
		if err != nil {
			return err
		}
		alias, err := config.OpenAlias(server, owner, name)
		if err != nil {
			return err
		}
		defer alias.Close()
		if alias.Record().State == "uncertain" {
			state, err := client.Reservation(alias.Record().TunnelID)
			if err != nil {
				return err
			}
			if state.State == "closed" {
				err = alias.SetState("closed")
			} else {
				err = alias.SetState("saved")
			}
			if err != nil {
				return err
			}
		}
		if replace {
			old := alias.Record()
			if err := alias.Replace(yes); err != nil {
				return err
			}
			if !clioutput.FromCommand(cmd).Structured() {
				if _, err := fmt.Fprintf(clioutput.FromCommand(cmd), "Replacing closed URL %s with a new random URL; target port %d\n", old.URL, ports[0]); err != nil {
					return err
				}
			}
		}
		client.SetAttachment(&internal.Attachment{TunnelID: alias.Record().TunnelID, Prepare: func() (string, string, string, error) {
			attempt, err := alias.Prepare()
			return attempt.ID, attempt.Expected, attempt.Nonce, err
		}, Acknowledge: alias.Acknowledge, Reject: func(status int) error {
			state := "conflict"
			if status == 410 {
				state = "closed"
			}
			return alias.SetState(state)
		}})
	}

	if headless(cmd) {
		return runHeadless(cmd, client, ports)
	}
	defer client.Close()

	failures := 0
	for _, port := range ports {
		if _, err := client.CreateTunnel(cmd.Context(), uint16(port)); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create tunnel for port %d: %v\n", port, err)
			failures++
		}
	}

	if failures == len(ports) {
		return fmt.Errorf("all %d tunnel(s) failed to create", len(ports))
	}

	return tui.RunWithClient(client, server)
}
