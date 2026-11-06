package tunnel

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"burrow/burrowctl/internal"
	"burrow/burrowctl/internal/capture"
	"burrow/burrowctl/internal/clioutput"
	"burrow/burrowctl/internal/config"
	"burrow/burrowctl/internal/localstate"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var localID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func captureStore(cmd *cobra.Command) (*capture.Store, string, error) {
	server, err := internal.ServerURL(viper.GetString("server"))
	if err != nil {
		return nil, "", err
	}
	owner, err := resolveToken(cmd, server.Host)
	if err != nil {
		return nil, "", err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, "", err
	}
	base := filepath.Join(home, ".config", "burrow")
	if err := localstate.Directory(base); err != nil {
		return nil, "", err
	}
	base = filepath.Join(base, "captures")
	if err := localstate.Directory(base); err != nil {
		return nil, "", err
	}
	scope := capture.Hash(server.String() + "\x00" + owner)
	store, err := capture.Open(filepath.Join(base, scope), scope, capture.OSKeys{})
	if err != nil {
		return nil, "", err
	}
	if err := store.Cleanup(); err != nil {
		store.Close()
		return nil, "", err
	}
	return store, owner, nil
}

func selector(cmd *cobra.Command, owner string) (string, string, error) {
	name, _ := cmd.Flags().GetString("name")
	tunnel, _ := cmd.Flags().GetString("tunnel")
	if (name == "") == (tunnel == "") {
		return "", "", clioutput.Invalid(errors.New("select exactly one --name or --tunnel"))
	}
	if name != "" {
		alias, err := config.FindSaved(viper.GetString("server"), owner, name)
		if err != nil {
			return "", "", err
		}
		return alias.TunnelID, alias.TargetDigest, nil
	}
	if !localID.MatchString(tunnel) {
		return "", "", clioutput.Invalid(errors.New("invalid tunnel selector"))
	}
	return tunnel, "", nil
}
func selectors(cmd *cobra.Command) {
	cmd.Flags().String("name", "", "Private saved alias")
	cmd.Flags().String("tunnel", "", "Owned local capture tunnel ID")
	previous := cmd.Args
	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if previous != nil {
			if err := previous(cmd, args); err != nil {
				return err
			}
		}
		name, _ := cmd.Flags().GetString("name")
		tunnel, _ := cmd.Flags().GetString("tunnel")
		if name != "" && tunnel != "" || (name == "" && tunnel == "" && cmd.Name() != "retry" && cmd.Name() != "jobs") {
			return clioutput.Invalid(errors.New("select exactly one --name or --tunnel"))
		}
		if name != "" {
			if err := config.ValidateAlias(name); err != nil {
				return clioutput.Invalid(err)
			}
		}
		if tunnel != "" && !localID.MatchString(tunnel) {
			return clioutput.Invalid(errors.New("invalid tunnel selector"))
		}
		return nil
	}
}
func arguments(check cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := check(cmd, args); err != nil {
			return clioutput.Invalid(err)
		}
		for _, arg := range args {
			if !localID.MatchString(arg) {
				return clioutput.Invalid(errors.New("invalid local record ID"))
			}
		}
		return nil
	}
}

func approve(cmd *cobra.Command, warning string) error {
	yes, _ := cmd.Flags().GetBool("yes")
	if yes {
		return nil
	}
	input, ok := cmd.InOrStdin().(*os.File)
	if !ok || !term.IsTerminal(input.Fd()) || clioutput.FromCommand(cmd).Structured() {
		return clioutput.Invalid(errors.New("explicit --yes is required without an interactive terminal"))
	}
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "%s\nType yes to approve: ", warning); err != nil {
		return err
	}
	answer, err := bufio.NewReader(io.LimitReader(input, 16)).ReadString('\n')
	if err != nil {
		return err
	}
	if strings.TrimSpace(answer) != "yes" {
		return errors.New("not approved")
	}
	return nil
}

func addCaptureCommands(root *cobra.Command) {
	requests := &cobra.Command{Use: "requests", Short: "Manage opt-in encrypted local request captures"}
	list := &cobra.Command{Use: "list", Args: arguments(cobra.NoArgs), RunE: func(cmd *cobra.Command, _ []string) error { return requestsCommand(cmd, "list", "") }}
	del := &cobra.Command{Use: "delete <request-id>", Args: arguments(cobra.ExactArgs(1)), RunE: func(cmd *cobra.Command, args []string) error { return requestsCommand(cmd, "delete", args[0]) }}
	clear := &cobra.Command{Use: "clear", Args: arguments(cobra.NoArgs), RunE: func(cmd *cobra.Command, _ []string) error { return requestsCommand(cmd, "clear", "") }}
	for _, cmd := range []*cobra.Command{list, del, clear} {
		selectors(cmd)
		if cmd != list {
			cmd.Flags().Bool("yes", false, "Approve the selected local erasure")
		}
		requests.AddCommand(cmd)
	}
	replay := &cobra.Command{Use: "replay <request-id>", Short: "Approve and run one durable local replay", Args: arguments(cobra.ExactArgs(1)), RunE: func(cmd *cobra.Command, args []string) error { return runReplay(cmd, args[0], false) }}
	selectors(replay)
	replay.Flags().Bool("yes", false, "Approve this selected record/target and possible duplicate effects")
	status := &cobra.Command{Use: "status <job-id>", Args: arguments(cobra.ExactArgs(1)), RunE: func(cmd *cobra.Command, args []string) error {
		store, _, err := captureStore(cmd)
		if err != nil {
			return err
		}
		defer store.Close()
		job, err := store.Status(args[0])
		if err != nil {
			return err
		}
		out := clioutput.FromCommand(cmd)
		if out.Structured() {
			return out.Emit("result", "", job)
		}
		status := "unknown"
		if job.HTTPStatus != nil {
			status = strconv.Itoa(*job.HTTPStatus)
		}
		_, err = fmt.Fprintf(out, "%s %s · HTTP %s · %s\n", job.ID, job.State, status, job.Code)
		return err
	}}
	retry := &cobra.Command{Use: "retry <job-id>", Args: arguments(cobra.ExactArgs(1)), RunE: func(cmd *cobra.Command, args []string) error { return runReplay(cmd, args[0], true) }}
	selectors(retry)
	retry.Flags().Bool("yes", false, "Approve a linked resend after inspecting duplicate-effect risk")
	jobs := &cobra.Command{Use: "jobs", Short: "List durable local replay outcomes for recovery", Args: arguments(cobra.NoArgs), RunE: func(cmd *cobra.Command, _ []string) error {
		store, owner, err := captureStore(cmd)
		if err != nil {
			return err
		}
		defer store.Close()
		name, _ := cmd.Flags().GetString("name")
		tunnel, _ := cmd.Flags().GetString("tunnel")
		if name != "" || tunnel != "" {
			tunnel, _, err = selector(cmd, owner)
			if err != nil {
				return err
			}
		}
		items, err := store.Jobs(tunnel)
		if err != nil {
			return err
		}
		out := clioutput.FromCommand(cmd)
		if out.Structured() {
			return out.Emit("result", tunnel, items)
		}
		for _, job := range items {
			if _, err := fmt.Fprintf(out, "%s %s · capture %s · %s\n", job.ID, job.State, job.CaptureID, job.Code); err != nil {
				return err
			}
		}
		return nil
	}}
	selectors(jobs)
	replay.AddCommand(status, retry, jobs)
	root.AddCommand(requests, replay)
}

func requestsCommand(cmd *cobra.Command, operation, id string) error {
	store, owner, err := captureStore(cmd)
	if err != nil {
		return err
	}
	defer store.Close()
	tunnel, _, err := selector(cmd, owner)
	if err != nil {
		return err
	}
	out := clioutput.FromCommand(cmd)
	if operation == "list" {
		items, err := store.List(tunnel)
		if err != nil {
			return err
		}
		if out.Structured() {
			return out.Emit("result", tunnel, items)
		}
		for _, item := range items {
			job := ""
			if item.LastJob != nil {
				job = " · last job " + item.LastJob.ID + " " + item.LastJob.State
			}
			if _, err := fmt.Fprintf(out, "%s %s %s · %d bytes · redacted=%t%s\n", item.ID, item.Method, item.Route, item.SizeBytes, item.Redacted, job); err != nil {
				return err
			}
		}
		return nil
	}
	if err := approve(cmd, "Erase selected local captures? This prevents future replay but cannot undo requests already dispatched."); err != nil {
		return err
	}
	if operation == "clear" {
		err = store.Clear(tunnel)
	} else {
		err = store.Delete(id, tunnel)
	}
	if err != nil {
		return err
	}
	if out.Structured() {
		return out.Emit("result", tunnel, map[string]bool{"erased": true})
	}
	_, err = fmt.Fprintln(out, "Local capture erasure committed")
	return err
}

func runReplay(cmd *cobra.Command, id string, retry bool) error {
	store, owner, err := captureStore(cmd)
	if err != nil {
		return err
	}
	defer store.Close()
	var record capture.Record
	var digest, tunnel, parent string
	if retry {
		record, err = store.RecordForJob(id)
		if err != nil {
			return err
		}
		parent = id
		tunnel = record.TunnelID
		digest = record.ConfigDigest
		name, _ := cmd.Flags().GetString("name")
		selected, _ := cmd.Flags().GetString("tunnel")
		if name != "" || selected != "" {
			var selectedDigest string
			selected, selectedDigest, err = selector(cmd, owner)
			if err != nil {
				return err
			}
			if selected != tunnel {
				return capture.ErrUnavailable
			}
			if selectedDigest != "" {
				digest = selectedDigest
			}
		}
	} else {
		tunnel, digest, err = selector(cmd, owner)
		if err != nil {
			return err
		}
		record, err = store.Read(id, tunnel)
		if err != nil {
			return err
		}
		if digest == "" {
			digest = record.ConfigDigest
		}
	}
	if digest != record.ConfigDigest {
		return errors.New("replay configuration drift")
	}
	// A retry without flags still rechecks any current alias for this owned ID.
	// It must not bypass a target change simply because the job ID is durable.
	aliases, err := config.ListSaved(viper.GetString("server"), owner)
	if err != nil {
		return err
	}
	for _, view := range aliases {
		if view.TunnelID == tunnel {
			alias, err := config.FindSaved(viper.GetString("server"), owner, view.Name)
			if err != nil {
				return err
			}
			if alias.TargetDigest != "" && alias.TargetDigest != record.ConfigDigest {
				return errors.New("replay configuration drift")
			}
		}
	}
	warning := fmt.Sprintf("Replay %s %s to loopback:%d, captured %s ago, redacted=%t, configuration %s. The port may now serve another application. Every replay, including GET, can duplicate side effects. No secrets or signatures are restored.", record.Method, record.Path, record.Route.Port, time.Since(record.CreatedAt).Round(time.Second), record.Redacted, digest)
	if err := approve(cmd, warning); err != nil {
		return err
	}
	out := clioutput.FromCommand(cmd)
	if err := out.Emit("approval", tunnel, map[string]any{"method": record.Method, "path": record.Path, "port": record.Route.Port, "redacted": record.Redacted, "config_digest": digest, "warning": "Approved local replay may duplicate effects; verify the application currently using this port"}); err != nil {
		return err
	}
	job, err := store.Queue(record.ID, tunnel, digest, parent, true)
	if err != nil {
		return err
	}
	if err := out.Emit("queued", tunnel, job); err != nil {
		return err
	}
	_, err = store.Run(cmd.Context(), job.ID, func(state capture.Job) error { return out.Emit(state.State, tunnel, state) })
	return err
}
