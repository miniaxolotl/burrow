package telemetry

import (
	"context"
	"time"
)

// Observer is deliberately content-free and injectable for lifecycle tests.
type Observer interface {
	Track(Event)
	Operation(Operation, time.Time, Outcome)
}

type observerKey struct{}

func WithObserver(ctx context.Context, observer Observer) context.Context {
	return context.WithValue(ctx, observerKey{}, observer)
}
func FromContext(ctx context.Context) Observer {
	if observer, ok := ctx.Value(observerKey{}).(Observer); ok {
		return observer
	}
	return (*Reporter)(nil)
}
