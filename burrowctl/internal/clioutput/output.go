// Package clioutput owns the CLI's versioned machine-output contract. Request
// bodies, owner credentials and arbitrary provider error messages never enter it.
package clioutput

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

type Envelope struct {
	SchemaVersion int       `json:"schema_version"`
	Type          string    `json:"type"`
	Timestamp     time.Time `json:"timestamp"`
	Sequence      uint64    `json:"sequence"`
	TunnelID      *string   `json:"tunnel_id"`
	Data          any       `json:"data"`
}

type Writer struct {
	mu       sync.Mutex
	out      io.Writer
	format   string
	sequence uint64
	failure  error
}

type writerKey struct{}

func Format(cmd *cobra.Command) string {
	if flag := cmd.Flag("output"); flag != nil {
		return flag.Value.String()
	}
	return "text"
}

func FromCommand(cmd *cobra.Command) *Writer {
	if writer, ok := cmd.Context().Value(writerKey{}).(*Writer); ok {
		return writer
	}
	return &Writer{out: cmd.OutOrStdout(), format: Format(cmd)}
}

func Prepare(cmd *cobra.Command) error {
	format := Format(cmd)
	if format != "text" && format != "json" && format != "jsonl" {
		return Invalid(errors.New("--output must be text, json or jsonl"))
	}
	streaming := cmd.Name() == "tunnel"
	if cmd.Name() == "logs" {
		follow, _ := cmd.Flags().GetBool("follow")
		streaming = follow
		tail, _ := cmd.Flags().GetInt("tail")
		if tail < 0 {
			return Invalid(errors.New("--tail must not be negative"))
		}
	}
	if format == "json" && streaming {
		return Invalid(errors.New("streaming commands require --output jsonl or text"))
	}
	if format != "text" {
		switch cmd.Name() {
		case "tunnel", "list", "logs", "close":
		default:
			return Invalid(errors.New("structured output is supported for tunnel, list, logs and close"))
		}
	}
	writer := &Writer{out: cmd.OutOrStdout(), format: format}
	cmd.SetContext(context.WithValue(cmd.Context(), writerKey{}, writer))
	return nil
}

func (w *Writer) Structured() bool { return w.format == "json" || w.format == "jsonl" }

func (w *Writer) Emit(kind, tunnelID string, data any) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.failure != nil {
		return w.failure
	}
	w.sequence++
	var id *string
	if tunnelID != "" {
		id = &tunnelID
	}
	var frame []byte
	var err error
	if w.Structured() {
		frame, err = json.Marshal(Envelope{1, kind, time.Now().UTC(), w.sequence, id, data})
	} else {
		var detail []byte
		detail, err = json.Marshal(data)
		frame = []byte(fmt.Sprintf("%s %s %s", kind, tunnelID, detail))
	}
	if err != nil {
		w.failure = err
		return err
	}
	frame = append(frame, '\n')
	return w.write(frame)
}

// Write bounds ordinary text output using the same failure state as events.
func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.write(append([]byte(nil), p...)); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *Writer) write(frame []byte) error {
	if w.failure != nil {
		return w.failure
	}
	// One write at a time, and at most one abandoned blocked write per process:
	// after failure no later event can start another writer. Generic io.Writers
	// cannot be interrupted; the CLI stops forwarding and exits rather than keeping
	// tunnels alive behind a stuck output consumer. This is UI I/O, not a work queue.
	result := make(chan error, 1)
	go func() {
		n, err := w.out.Write(frame)
		if err == nil && n != len(frame) {
			err = io.ErrShortWrite
		}
		result <- err
	}()
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case w.failure = <-result:
	case <-timer.C:
		w.failure = errors.New("command output deadline exceeded")
	}
	return w.failure
}

type commandError struct {
	cause error
	code  string
	exit  int
}

func (e *commandError) Error() string { return e.cause.Error() }
func (e *commandError) Unwrap() error { return e.cause }

func Invalid(err error) error { return &commandError{err, "invalid_argument", 2} }

func ExitCode(err error) int {
	var classified *commandError
	if errors.As(err, &classified) {
		return classified.exit
	}
	return 1
}

// Report is called once by the root command, including for argument/flag errors
// that occur before a RunE or pre-run hook can initialize an output writer.
func Report(cmd *cobra.Command, err error) error {
	writer := FromCommand(cmd)
	if !writer.Structured() {
		_, writeErr := fmt.Fprintln(cmd.ErrOrStderr(), "Error:", err)
		return writeErr
	}
	code, message := "operation_failed", "Burrow operation failed"
	var classified *commandError
	if errors.As(err, &classified) {
		code = classified.code
		if classified.exit == 2 {
			message = "Invalid arguments or configuration; see command help"
		}
	}
	return writer.Emit("error", "", struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{code, message})
}
