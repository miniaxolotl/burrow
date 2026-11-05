package tunnel

import (
	"context"
	"errors"
	"os"
	"time"

	"burrow/burrowctl/internal"
	"burrow/burrowctl/internal/clioutput"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

func headless(cmd *cobra.Command) bool {
	explicit, _ := cmd.Flags().GetBool("headless")
	if explicit || clioutput.FromCommand(cmd).Structured() {
		return true
	}
	input, inputOK := cmd.InOrStdin().(*os.File)
	output, outputOK := cmd.OutOrStdout().(*os.File)
	return !inputOK || !outputOK || !term.IsTerminal(input.Fd()) || !term.IsTerminal(output.Fd())
}

func runHeadless(cmd *cobra.Command, client *internal.Client, ports []int) (err error) {
	ctx, cancel := context.WithCancelCause(cmd.Context())
	defer func() {
		cancel(context.Canceled)
		drain, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if closeErr := client.CloseContext(drain); err == nil {
			err = closeErr
		}
	}()
	output := clioutput.FromCommand(cmd)
	events := make(chan internal.LifecycleEvent, 64)
	client.SetEventHandler(func(event internal.LifecycleEvent) {
		select {
		case events <- event:
		default:
			cancel(errors.New("lifecycle output backpressure"))
		}
	})
	started := make(chan error, 1)
	go func() {
		for _, port := range ports {
			if _, err := client.CreateTunnel(ctx, uint16(port)); err != nil {
				started <- err
				return
			}
		}
		started <- nil
	}()
	startup := time.NewTimer(30 * time.Second)
	defer startup.Stop()
	startupDeadline := startup.C
	readiness := newReadiness(len(ports))
	for {
		select {
		case event := <-events:
			transition, acknowledged := readiness.observe(event)
			// Invalidate the previous barrier before reporting reconnect details.
			if transition == "not_ready" {
				if err := output.Emit(transition, "", map[string]any{"observed_at": event.ObservedAt, "reason": event.Code}); err != nil {
					return err
				}
			}
			data := struct {
				ObservedAt            time.Time `json:"observed_at"`
				Port                  uint16    `json:"port"`
				URL                   string    `json:"url,omitempty"`
				Attempt               int       `json:"attempt,omitempty"`
				RetryDelayMS          int64     `json:"retry_delay_ms,omitempty"`
				Code                  string    `json:"code,omitempty"`
				RegistrationConfirmed bool      `json:"registration_confirmed"`
				Generation            string    `json:"generation,omitempty"`
			}{ObservedAt: event.ObservedAt, Port: event.Port, URL: event.URL, Attempt: event.Attempt, RetryDelayMS: event.RetryDelayMS, Code: event.Code, RegistrationConfirmed: event.RegistrationConfirmed, Generation: event.Generation}
			if err := output.Emit(event.Type, event.TunnelID, data); err != nil {
				return err
			}
			if event.Type == "failed" {
				return errors.New("tunnel connection failed")
			}
			if transition == "ready" {
				if err := output.Emit(transition, "", map[string]any{"observed_at": event.ObservedAt, "tunnels": acknowledged}); err != nil {
					return err
				}
			}
		case err := <-started:
			if err != nil {
				return err
			}
			started = nil
			startup.Stop()
			startupDeadline = nil
		case <-startupDeadline:
			return errors.New("tunnel startup deadline exceeded")
		case <-ctx.Done():
			if cmd.Context().Err() != nil {
				return output.Emit("stopped", "", map[string]string{"reason": "cancelled"})
			}
			return context.Cause(ctx)
		}
	}
}
