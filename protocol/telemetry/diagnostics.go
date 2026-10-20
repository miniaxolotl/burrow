package telemetry

import (
	"regexp"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/attribute"
)

var sourceFile = regexp.MustCompile(`(?:^|/)((?:burrowctl|burrowd|protocol)/[a-zA-Z0-9_/.-]+\.go)$`)

func (r *Reporter) safeFrames(stack *sentry.Stacktrace) *sentry.Stacktrace {
	if stack == nil {
		return nil
	}
	frames := []sentry.Frame{}
	for _, frame := range stack.Frames {
		match := sourceFile.FindStringSubmatch(frame.Filename)
		if len(match) != 2 {
			continue
		}
		function := frame.Function
		if len(function) > 120 || strings.ContainsAny(function, "\n\r") {
			function = "[redacted]"
		}
		frames = append(frames, sentry.Frame{Filename: match[1], AbsPath: "https://github.com/miniaxolotl/burrow/blob/" + r.config.Revision + "/" + match[1], Function: function, Lineno: frame.Lineno, InApp: true})
	}
	return &sentry.Stacktrace{Frames: frames}
}

func (r *Reporter) sanitizeDiagnostic(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	if r.disabled() {
		return nil
	}
	op := Operation(event.Tags["operation"])
	if event.Type == "transaction" {
		op = Operation(event.Transaction)
	}
	if !validOperation(op) {
		return nil
	}
	tags := map[string]string{"product": "burrow", "surface": r.config.Surface, "operation": string(op)}
	if outcome := Outcome(event.Tags["outcome"]); validOutcome(outcome) {
		tags["outcome"] = string(outcome)
	}
	safe := &sentry.Event{EventID: event.EventID, Timestamp: event.Timestamp, Level: event.Level, Platform: event.Platform, Sdk: event.Sdk, Release: r.release(), Environment: "production", Tags: tags}
	if trace, ok := event.Contexts["trace"]; ok {
		safe.Contexts = map[string]sentry.Context{"trace": {}}
		for _, key := range []string{"trace_id", "span_id", "parent_span_id", "status"} {
			if value, ok := trace[key]; ok {
				safe.Contexts["trace"][key] = value
			}
		}
		safe.Contexts["trace"]["op"] = "burrow.operation"
	}
	if event.Type == "transaction" {
		safe.Type = "transaction"
		safe.Transaction = string(op)
		safe.StartTime = event.StartTime
		safe.TransactionInfo = &sentry.TransactionInfo{Source: sentry.SourceCustom}
	} else {
		for _, exception := range event.Exception {
			safe.Exception = append(safe.Exception, sentry.Exception{Type: "OperationError", Value: string(op), Stacktrace: r.safeFrames(exception.Stacktrace)})
		}
	}
	return safe
}

func (r *Reporter) safeAttributes(input map[string]attribute.Value) map[string]attribute.Value {
	result := map[string]attribute.Value{
		"product": attribute.StringValue("burrow"), "surface": attribute.StringValue(r.config.Surface),
		"sentry.environment": attribute.StringValue("production"), "sentry.release": attribute.StringValue(r.release()),
	}
	if op := Operation(input["operation"].AsString()); validOperation(op) {
		result["operation"] = attribute.StringValue(string(op))
	}
	if outcome := Outcome(input["outcome"].AsString()); validOutcome(outcome) {
		result["outcome"] = attribute.StringValue(string(outcome))
	}
	return result
}
func (r *Reporter) sanitizeLog(event *sentry.Log) *sentry.Log {
	if r.disabled() || event.Body != "Burrow operation completed" {
		return nil
	}
	return &sentry.Log{Timestamp: event.Timestamp, TraceID: event.TraceID, SpanID: event.SpanID, Level: sentry.LogLevelInfo, Body: event.Body, Attributes: r.safeAttributes(event.Attributes)}
}
func (r *Reporter) sanitizeMetric(metric *sentry.Metric) *sentry.Metric {
	if r.disabled() || (metric.Name != "burrow.operation.count" && metric.Name != "burrow.operation.duration") {
		return nil
	}
	return &sentry.Metric{Timestamp: metric.Timestamp, TraceID: metric.TraceID, SpanID: metric.SpanID, Name: metric.Name, Type: metric.Type, Value: metric.Value, Unit: metric.Unit, Attributes: r.safeAttributes(metric.Attributes)}
}
