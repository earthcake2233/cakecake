// Package traceid carries a request-scoped trace ID through the upload ->
// outbox -> RabbitMQ -> worker pipeline so every log line of one transcode
// job can be correlated.
package traceid

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey struct{}

// FromContext returns the trace ID stored in ctx, or empty.
func FromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKey{}).(string)
	return v
}

// WithContext returns a ctx carrying traceID. Empty IDs are ignored.
func WithContext(ctx context.Context, traceID string) context.Context {
	if traceID == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, traceID)
}

// New returns a fresh random trace ID.
func New() string {
	return uuid.NewString()
}
