package telemetry

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/attribute"
	"github.com/posthog/posthog-go"
)

type Event string

const (
	TunnelOpened      Event = "tunnel_opened"
	TunnelForwarded   Event = "tunnel_request_completed"
	TunnelReconnected Event = "tunnel_reconnected"
	CommandCompleted  Event = "command_completed"
)

type Operation string

const (
	Connect   Operation = "tunnel_connect"
	Reconnect Operation = "tunnel_reconnect"
	Forward   Operation = "tunnel_forward"
	List      Operation = "tunnel_list"
	Logs      Operation = "tunnel_logs"
	Close     Operation = "tunnel_close"
	Serve     Operation = "server_run"
	Command   Operation = "command_run"
)

type Outcome string

const (
	Success     Outcome = "success"
	Unavailable Outcome = "unavailable"
	Failure     Outcome = "error"
)

type Config struct {
	Enabled                                        bool
	Surface, Version, Revision, Key, DSN, StateDir string
}

type Reporter struct {
	config    Config
	id        string
	posthog   posthog.Client
	hub       *sentry.Hub
	closeOnce sync.Once
	stopped   atomic.Bool
	accepting atomic.Bool
	queue     chan record
	closing   chan struct{}
	done      chan struct{}
}

type record struct {
	event              Event
	operation          Operation
	outcome            Outcome
	started, completed time.Time
	stack              *sentry.Stacktrace
}

type gatedTransport struct {
	base     http.RoundTripper
	disabled func() bool
}

func (t gatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.disabled() {
		return nil, errors.New("telemetry disabled")
	}
	return t.base.RoundTrip(req)
}

var revisionPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)

// New is inert for local builds, CI, explicit opt-out, or unavailable private
// storage. Both SDKs have bounded queues, upload deadlines and shutdown waits.
func New(config Config) *Reporter { return newReporter(config, "https://us.i.posthog.com", nil, 0.1) }
func newReporter(config Config, endpoint string, transport http.RoundTripper, sampleRate float64) *Reporter {
	r := &Reporter{config: config}
	if !config.Enabled || !revisionPattern.MatchString(config.Revision) || (config.Surface != "cli" && config.Surface != "server") || Disabled(config.StateDir) {
		return r
	}
	r.id = installationID(config.StateDir)
	if r.id == "" {
		return r
	}
	if transport == nil {
		transport = http.DefaultTransport.(*http.Transport).Clone()
	}
	transport = gatedTransport{base: transport, disabled: r.disabled}
	if config.Key != "" {
		r.posthog, _ = posthog.NewWithConfig(config.Key, posthog.Config{
			Endpoint: endpoint, Transport: transport, Logger: posthog.StdLogger(log.New(io.Discard, "", 0), false),
			Interval: 2 * time.Second, BatchSize: 20, MaxQueueSize: 128, MaxEnqueuedRequests: 2,
			MaxRetries: posthog.Ptr(0), BatchUploadTimeout: 500 * time.Millisecond,
			BatchSubmitTimeout: -1, ShutdownTimeout: 500 * time.Millisecond,
			DisableGeoIP: posthog.Ptr(true), IsServer: posthog.Ptr(config.Surface == "server"),
			BeforeSend: r.sanitizeCapture,
		})
	}
	if config.DSN != "" {
		off := &sentry.KeyValueCollectionBehavior{Mode: sentry.CollectionOff}
		client, err := sentry.NewClient(sentry.ClientOptions{
			Dsn: config.DSN, Environment: "production", Release: r.release(), ServerName: "burrow-" + config.Surface,
			HTTPTransport: transport, HTTPClient: &http.Client{Transport: transport, Timeout: 500 * time.Millisecond},
			AttachStacktrace: true, MaxBreadcrumbs: -1, MaxSpans: 20, DisableClientReports: true,
			EnableTracing: true, TracesSampleRate: sampleRate, TracePropagationTargets: []string{},
			DataCollection: &sentry.DataCollection{UserInfo: sentry.Set(false), Cookies: off, QueryParams: off, HTTPHeaders: &sentry.HeaderCollectionConfig{Request: off, Response: off}, HTTPBodies: []sentry.BodyType{}},
			BeforeSend:     r.sanitizeDiagnostic, BeforeSendTransaction: r.sanitizeDiagnostic,
			BeforeSendLog: r.sanitizeLog, BeforeSendMetric: r.sanitizeMetric,
		})
		if err == nil {
			r.hub = sentry.NewHub(client, sentry.NewScope())
		}
	}
	r.queue = make(chan record, 128)
	r.closing = make(chan struct{})
	r.done = make(chan struct{})
	r.accepting.Store(true)
	go r.run()
	return r
}
func (r *Reporter) disabled() bool {
	if r == nil || !r.config.Enabled || r.id == "" || r.stopped.Load() {
		return true
	}
	// An old process may never resume a cleared identity after another process
	// opts back in. The new identity is used on its next invocation instead.
	id, err := os.ReadFile(filepath.Join(r.config.StateDir, "telemetry-id"))
	if Disabled(r.config.StateDir) || err != nil || strings.TrimSpace(string(id)) != r.id {
		r.stopped.Store(true)
		return true
	}
	return false
}
func (r *Reporter) release() string { return "burrow-" + r.config.Surface + "@" + r.config.Revision }
func validOperation(op Operation) bool {
	switch op {
	case Connect, Reconnect, Forward, List, Logs, Close, Serve, Command:
		return true
	}
	return false
}
func validOutcome(outcome Outcome) bool {
	return outcome == Success || outcome == Unavailable || outcome == Failure
}

func (r *Reporter) Track(event Event) {
	switch event {
	case TunnelOpened, TunnelForwarded, TunnelReconnected, CommandCompleted:
		r.enqueue(record{event: event})
	}
}

func (r *Reporter) enqueue(item record) {
	if r == nil || !r.accepting.Load() || r.stopped.Load() {
		return
	}
	select {
	case r.queue <- item:
	default:
	}
}

func (r *Reporter) run() {
	defer func() {
		// This worker owns only telemetry. A provider SDK panic must never take
		// down a tunnel or the host process.
		if recover() != nil {
			r.stopped.Store(true)
			r.accepting.Store(false)
		}
		r.flush()
		close(r.done)
	}()
	process := func(item record) {
		if r.disabled() {
			return
		}
		if item.event != "" {
			if r.posthog != nil {
				_ = r.posthog.Enqueue(posthog.Capture{Event: string(item.event), DistinctId: r.id})
			}
		} else {
			r.recordOperation(item)
		}
	}
	for {
		select {
		case item := <-r.queue:
			process(item)
		case <-r.closing:
			for {
				select {
				case item := <-r.queue:
					process(item)
				default:
					return
				}
			}
		}
	}
}

// Operation accepts only closed operation/outcome enums and a monotonic duration.
// Diagnostic scopes are private to this reporter, never the global process hub.
func (r *Reporter) Operation(op Operation, started time.Time, outcome Outcome) {
	if r == nil || !r.accepting.Load() || !validOperation(op) || !validOutcome(outcome) {
		return
	}
	item := record{operation: op, outcome: outcome, started: started, completed: time.Now()}
	// Preserve the original call site, never locals or error text. This is only
	// needed for unexpected failures; normal tunnel forwarding is a channel send.
	if outcome == Failure {
		item.stack = sentry.NewStacktrace()
	}
	r.enqueue(item)
}

func (r *Reporter) recordOperation(item record) {
	op, outcome, started := item.operation, item.outcome, item.started
	duration := min(max(float64(item.completed.Sub(started))/float64(time.Millisecond), 0), 300_000)
	if r.posthog != nil {
		_ = r.posthog.Enqueue(posthog.Capture{Event: "operation_completed", DistinctId: r.id, Properties: posthog.Properties{"operation": string(op), "outcome": string(outcome), "duration_ms": duration}})
	}
	if r.hub == nil {
		return
	}
	hub := r.hub.Clone()
	ctx := sentry.SetHubOnContext(context.Background(), hub)
	span := sentry.StartTransaction(ctx, string(op), sentry.WithOpName("burrow.operation"), sentry.WithTransactionSource(sentry.SourceCustom))
	span.StartTime = started
	span.SetTag("operation", string(op))
	span.SetTag("outcome", string(outcome))
	if outcome == Success {
		span.Status = sentry.SpanStatusOK
	} else {
		span.Status = sentry.SpanStatusUnavailable
	}
	attributes := []attribute.Builder{attribute.String("operation", string(op)), attribute.String("outcome", string(outcome))}
	meter := sentry.NewMeter(span.Context())
	meter.Count("burrow.operation.count", 1, sentry.WithAttributes(attributes...))
	meter.Distribution("burrow.operation.duration", duration, sentry.WithUnit(sentry.UnitMillisecond), sentry.WithAttributes(attributes...))
	// Forwarding is high-volume; metrics/traces describe it without per-stream logs.
	if op != Forward {
		sentry.NewLogger(span.Context()).Info().String("operation", string(op)).String("outcome", string(outcome)).Emit("Burrow operation completed")
	}
	if outcome == Failure {
		hub.CaptureEvent(&sentry.Event{Level: sentry.LevelError, Tags: map[string]string{"operation": string(op), "outcome": string(outcome)}, Exception: []sentry.Exception{{Type: "OperationError", Value: string(op), Stacktrace: item.stack}}})
	}
	span.EndTime = item.completed
	span.Finish()
}

func (r *Reporter) Close() {
	if r == nil || r.queue == nil {
		return
	}
	r.closeOnce.Do(func() {
		r.accepting.Store(false)
		close(r.closing)
		timer := time.NewTimer(550 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-r.done:
		case <-timer.C:
		}
		r.stopped.Store(true)
	})
}

func (r *Reporter) flush() {
	var wg sync.WaitGroup
	if r.posthog != nil {
		wg.Go(func() { _ = r.posthog.Close() })
	}
	if r.hub != nil {
		wg.Go(func() { r.hub.Flush(500 * time.Millisecond); r.hub.Client().Close() })
	}
	wg.Wait()
}

func (r *Reporter) sanitizeCapture(message posthog.Message) posthog.Message {
	if r.disabled() {
		return nil
	}
	var capture posthog.Capture
	switch value := message.(type) {
	case posthog.Capture:
		capture = value
	case *posthog.Capture:
		capture = *value
	default:
		return nil
	}
	switch Event(capture.Event) {
	case TunnelOpened, TunnelForwarded, TunnelReconnected, CommandCompleted, "operation_completed":
	default:
		return nil
	}
	properties := posthog.Properties{"product": "burrow", "surface": r.config.Surface, "environment": "production", "version": r.config.Version, "release": r.release(), "$os": runtime.GOOS, "architecture": runtime.GOARCH, "$process_person_profile": false, "$geoip_disable": true}
	if op, ok := capture.Properties["operation"].(string); ok && validOperation(Operation(op)) {
		properties["operation"] = op
	}
	if outcome, ok := capture.Properties["outcome"].(string); ok && validOutcome(Outcome(outcome)) {
		properties["outcome"] = outcome
	}
	if duration, ok := capture.Properties["duration_ms"].(float64); ok && duration >= 0 && duration <= 300_000 {
		properties["duration_ms"] = duration
	}
	return posthog.Capture{Event: capture.Event, DistinctId: r.id, Uuid: capture.Uuid, Timestamp: capture.Timestamp, Properties: properties}
}
