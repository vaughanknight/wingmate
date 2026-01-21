package flightlog

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const traceIDKey contextKey = "wingmate-trace-id"

// WithTraceID returns a new context with the given trace ID.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceIDFromContext extracts the trace ID from context.
// Returns empty string if no trace ID is set.
func TraceIDFromContext(ctx context.Context) string {
	v := ctx.Value(traceIDKey)
	if v == nil {
		return ""
	}
	traceID, ok := v.(string)
	if !ok {
		return ""
	}
	return traceID
}

// GenerateTraceID creates a new random trace ID.
// Returns a 16-character hex string (8 random bytes).
func GenerateTraceID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return "fallback-trace"
	}
	return hex.EncodeToString(b)
}

// EnsureTraceID returns context with a trace ID, generating one if missing.
func EnsureTraceID(ctx context.Context) context.Context {
	if TraceIDFromContext(ctx) == "" {
		return WithTraceID(ctx, GenerateTraceID())
	}
	return ctx
}
