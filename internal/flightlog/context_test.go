package flightlog

import (
	"context"
	"testing"
)

// TestContext_TraceID verifies trace ID round-trip through context.
//
// Test Doc:
// - Why: Trace IDs enable log correlation across Pilot/Wingmate
// - Contract: WithTraceID → TraceIDFromContext returns same value
// - Usage Notes: Trace ID propagates through context chain
// - Quality Contribution: Catches context key collisions, type assertions
// - Worked Example: WithTraceID(ctx, "abc") → TraceIDFromContext(ctx) == "abc"
func TestContext_TraceID(t *testing.T) {
	traceID := "test-trace-12345"

	ctx := context.Background()
	ctx = WithTraceID(ctx, traceID)

	got := TraceIDFromContext(ctx)
	if got != traceID {
		t.Errorf("TraceIDFromContext = %q, want %q", got, traceID)
	}
}

// TestContext_NoTraceID verifies behavior when no trace ID is set.
//
// Test Doc:
// - Why: Handlers must work even without trace ID
// - Contract: Missing trace ID returns empty string
// - Usage Notes: Callers should handle empty trace gracefully
// - Quality Contribution: Catches nil pointer panics
// - Worked Example: TraceIDFromContext(emptyCtx) == ""
func TestContext_NoTraceID(t *testing.T) {
	ctx := context.Background()

	got := TraceIDFromContext(ctx)
	if got != "" {
		t.Errorf("TraceIDFromContext = %q, want empty string", got)
	}
}

// TestContext_TraceIDOverwrite verifies that trace ID can be overwritten.
//
// Test Doc:
// - Why: Request handlers may need to set their own trace ID
// - Contract: Later WithTraceID overwrites earlier value
// - Usage Notes: Context is immutable; WithTraceID returns new context
// - Quality Contribution: Catches context layering bugs
func TestContext_TraceIDOverwrite(t *testing.T) {
	ctx := context.Background()
	ctx = WithTraceID(ctx, "original")
	ctx = WithTraceID(ctx, "overwritten")

	got := TraceIDFromContext(ctx)
	if got != "overwritten" {
		t.Errorf("TraceIDFromContext = %q, want %q", got, "overwritten")
	}
}

// TestGenerateTraceID verifies trace ID generation.
//
// Test Doc:
// - Why: Need unique trace IDs for correlation
// - Contract: GenerateTraceID returns non-empty unique strings
// - Quality Contribution: Catches empty or duplicate ID generation
func TestGenerateTraceID(t *testing.T) {
	id1 := GenerateTraceID()
	id2 := GenerateTraceID()

	if id1 == "" {
		t.Error("GenerateTraceID returned empty string")
	}
	if id1 == id2 {
		t.Error("GenerateTraceID returned duplicate IDs")
	}
}
