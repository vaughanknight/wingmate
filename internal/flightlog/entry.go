// Package flightlog provides observability infrastructure for Wingmate.
//
// Flight Log captures all agent conversations with human-readable summaries
// and raw payloads for full traceability per Constitution P1 (Observability First).
package flightlog

import (
	"context"
	"time"
)

// Direction indicates the flow direction of a log entry.
type Direction string

const (
	// Outbound indicates a message sent by this agent.
	Outbound Direction = "out"

	// Inbound indicates a message received by this agent.
	Inbound Direction = "in"

	// ErrorDir indicates an error condition.
	ErrorDir Direction = "error"
)

// Role indicates this agent's role in a specific conversation.
// Per ADR-003, "pilot" and "wingmate" are conversation roles, not instance modes.
// Any agent can be either role depending on who initiated the conversation.
type Role string

const (
	// RolePilot indicates this agent initiated the conversation.
	RolePilot Role = "pilot"

	// RoleWingmate indicates this agent is responding to a conversation.
	RoleWingmate Role = "wingmate"
)

// Entry represents a single Flight Log entry.
// Per ADR-002, entries are written as JSONL (one JSON object per line).
type Entry struct {
	// Timestamp is the UTC time of the event. Auto-filled by NewEntry.
	Timestamp time.Time `json:"ts"`

	// TraceID enables correlation across agents. Auto-filled from context.
	TraceID string `json:"trace,omitempty"`

	// Agent is this agent's identifier.
	Agent string `json:"agent"`

	// Peer is the other agent's identifier (if known).
	Peer string `json:"peer,omitempty"`

	// Role indicates whether this agent was pilot (initiator) or wingmate (responder)
	// in this specific conversation. Per ADR-003, this is a conversation-level role,
	// not an instance configuration.
	Role Role `json:"role,omitempty"`

	// Direction indicates message flow (out/in/error).
	Direction Direction `json:"dir"`

	// Method is the JSON-RPC method name (if applicable).
	Method string `json:"method,omitempty"`

	// Summary is a human-readable one-line description.
	Summary string `json:"summary"`

	// Payload is the raw message content.
	Payload interface{} `json:"payload"`
}

// NewEntry creates a new Entry with Timestamp and TraceID auto-filled.
//
// Per Critical Insight #3, this constructor handles Go's time.Time zero-value
// gotcha by always setting Timestamp to time.Now().UTC(). It also extracts
// TraceID from the context for log correlation.
//
// Usage:
//
//	entry := flightlog.NewEntry(ctx, "agent-001", flightlog.Inbound, "Received ping", msg)
//	entry.Peer = peerName  // Optional fields set after construction
//	entry.Role = flightlog.RoleWingmate  // Set conversation role per ADR-003
//	entry.Method = "ping"
//	fl.Record(entry)
func NewEntry(ctx context.Context, agent string, dir Direction, summary string, payload interface{}) Entry {
	return Entry{
		Timestamp: time.Now().UTC(),
		TraceID:   TraceIDFromContext(ctx),
		Agent:     agent,
		Direction: dir,
		Summary:   summary,
		Payload:   payload,
	}
}

// NewEntryWithRole creates a new Entry with Role, Timestamp and TraceID auto-filled.
// This is a convenience constructor when the conversation role is known at entry creation.
//
// Per ADR-003, Role indicates whether this agent initiated (pilot) or responded (wingmate)
// in this specific conversation.
//
// Usage:
//
//	entry := flightlog.NewEntryWithRole(ctx, "agent-001", flightlog.RolePilot, flightlog.Outbound, "Sending ping", msg)
//	entry.Peer = peerName
//	fl.Record(entry)
func NewEntryWithRole(ctx context.Context, agent string, role Role, dir Direction, summary string, payload interface{}) Entry {
	return Entry{
		Timestamp: time.Now().UTC(),
		TraceID:   TraceIDFromContext(ctx),
		Agent:     agent,
		Role:      role,
		Direction: dir,
		Summary:   summary,
		Payload:   payload,
	}
}
