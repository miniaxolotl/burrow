package capture

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"burrow/burrowctl/internal/routing"
	"modernc.org/sqlite"
)

func Digest(table *routing.Table) string { data, _ := json.Marshal(table); return Hash(string(data)) }
func readJobPayload(tx *sql.Tx, n int64, j Job) ([]byte, error) {
	var data []byte
	err := tx.QueryRow("SELECT payload FROM captures WHERE id=? AND tunnel=? AND deleted=0 AND expires_ms>?", j.CaptureID, j.tunnel, n).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) || err == nil && data == nil {
		return nil, ErrUnavailable
	}
	return data, err
}
func validateReplay(r Record) error {
	if r.Route.Port == 0 || r.Method == "CONNECT" || r.Method == "" || strings.ContainsAny(r.Method, " \r\n\t") {
		return ErrUnavailable
	}
	if r.ConfigPath != "" {
		file, err := os.Open(r.ConfigPath)
		if err != nil {
			return errors.New("captured route config unavailable")
		}
		data, err := io.ReadAll(io.LimitReader(file, routing.MaxConfig+1))
		file.Close()
		if err != nil {
			return err
		}
		table, err := routing.Parse(data)
		if err != nil || Digest(table) != r.ConfigDigest {
			return errors.New("replay configuration drift")
		}
	}
	return nil
}

type Preview struct {
	PreviousJob                              *Job
	ID, TunnelID, Method, Path, ConfigDigest string
	Port                                     uint16
	Redacted                                 bool
	CreatedAt                                time.Time
}

func (s *Store) Preview(id, tunnel string) (Preview, error) {
	r, err := s.Read(id, tunnel)
	if err != nil {
		return Preview{}, err
	}
	if err := validateReplay(r); err != nil {
		return Preview{}, err
	}
	var previous *Job
	err = s.transaction(func(tx *sql.Tx, n int64) error {
		if err := invalidateJobs(tx, n); err != nil {
			return err
		}
		if err := expireJobs(tx, n); err != nil {
			return err
		}
		var err error
		previous, err = latestJob(tx, r.ID, Hash(tunnel))
		return err
	})
	if err != nil {
		return Preview{}, err
	}
	return Preview{PreviousJob: previous, ID: r.ID, TunnelID: r.TunnelID, Method: r.Method, Path: r.Path, ConfigDigest: r.ConfigDigest, Port: r.Route.Port, Redacted: r.Redacted, CreatedAt: r.CreatedAt}, nil
}

// Run is the only dispatch path. Status, tunnel startup and cleanup never call
// it. Every HTTP client is created after a durable intent under the current lease.
func (s *Store) Run(ctx context.Context, id string, emit func(Job) error) (Job, error) {
	j, err := s.Claim(id)
	if err != nil {
		status, _ := s.Status(id)
		return status, err
	}
	fail := func(code string) (Job, error) {
		result, err := s.finish(j, nil, code)
		if err != nil {
			return result, err
		}
		return result, errors.New("replay preparation failed")
	}
	if emit != nil {
		if err := emit(j); err != nil {
			return fail("output_unavailable")
		}
	}
	var payload []byte
	for attempt := 1; attempt <= 3; attempt++ {
		err = s.transaction(func(tx *sql.Tx, n int64) error { var err error; payload, err = readJobPayload(tx, n, j); return err })
		if err == nil {
			break
		}
		var sqliteError *sqlite.Error
		if !errors.As(err, &sqliteError) || (sqliteError.Code()&255 != 5 && sqliteError.Code()&255 != 6 && sqliteError.Code()&255 != 10) || attempt == 3 {
			break
		}
		state, stateErr := s.preparation(j, true, attempt)
		if stateErr != nil {
			return fail("storage_unavailable")
		}
		if emit != nil {
			if err := emit(state); err != nil {
				return fail("output_unavailable")
			}
		}
		timer := time.NewTimer(time.Until(*state.RetryAt))
		select {
		case <-ctx.Done():
			timer.Stop()
			return fail("cancelled_before_dispatch")
		case <-timer.C:
		}
		if _, err := s.preparation(j, false, attempt+1); err != nil {
			return fail("preparation_fenced")
		}
	}
	if err != nil {
		return fail("capture_unavailable")
	}
	r, err := s.unseal(j.CaptureID, payload)
	if err != nil || Hash(r.TunnelID) != j.tunnel {
		return fail("capture_corrupt")
	}
	if err := validateReplay(r); err != nil {
		return fail("config_drift")
	}
	deadline := j.DeadlineAt
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	if ctx.Err() != nil {
		return fail("cancelled_before_dispatch")
	}
	if err := s.Intent(j.ID, j.Generation); err != nil {
		return fail("dispatch_fenced")
	}
	// Durable cancellation/erasure can originate in a different CLI process.
	// This foreground worker rechecks it; acceptance and execution stay in SQLite.
	watchDone := make(chan struct{})
	watchStopped := make(chan struct{})
	go func() {
		defer close(watchStopped)
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-watchDone:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				status, err := s.Status(j.ID)
				if err != nil || status.State != "processing" || status.Generation != j.Generation {
					cancel()
					return
				}
			}
		}
	}()
	defer func() { close(watchDone); <-watchStopped }()
	path := r.Path
	if r.Route.StripPrefix {
		path = strings.TrimPrefix(path, r.Route.Path)
		if path == "" {
			path = "/"
		}
	}
	decoded, err := url.PathUnescape(path)
	if err != nil {
		return fail("invalid_path")
	}
	u := url.URL{Scheme: "http", Host: "loopback.invalid", Path: decoded, RawPath: path, RawQuery: r.RawQuery}
	req, err := http.NewRequestWithContext(ctx, r.Method, u.String(), bytes.NewReader(r.Body))
	if err != nil {
		return fail("invalid_request")
	}
	req.GetBody = nil
	req.Host = r.Host
	if r.Route.HostHeader != "" {
		req.Host = r.Route.HostHeader
	}
	req.Header = r.Headers.Clone()
	stripHeaders(req.Header)
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return routing.DialLoopback(ctx, r.Route.Port)
	}, DisableKeepAlives: true, DisableCompression: true, MaxResponseHeaderBytes: routing.MaxHeaders}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		result, finishErr := s.finish(j, nil, "transport_outcome_uncertain")
		if finishErr != nil {
			return result, finishErr
		}
		return result, errors.New("replay outcome uncertain; inspect before explicitly resending")
	}
	status := resp.StatusCode
	result, err := s.finish(j, &status, "")
	// Response bodies are never archived; headers/status have already committed.
	io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
	resp.Body.Close()
	if err != nil {
		return result, err
	}
	if emit != nil {
		if err := emit(result); err != nil {
			return result, err
		}
	}
	if status < 200 || status >= 300 {
		return result, errors.New("replay observed a non-2xx HTTP response")
	}
	return result, nil
}
