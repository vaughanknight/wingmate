# S4: Module Dependencies and Cross-Cutting Concerns

**Research Area**: Dependency Mapper - Architectural Boundaries
**Date**: 2026-01-21
**Architecture Reference**: `docs/project-rules/architecture.md`

---

## Overview

This document maps module dependencies, package interfaces, and cross-cutting concerns for the Wingmate implementation. The analysis ensures architectural boundaries are respected and identifies patterns for consistent implementation.

**Dependency Rules (from architecture.md)**:
- `pilot/` and `wingmate/` MAY import `agent/`, `protocol/`, `flightlog/`
- `pilot/` MUST NOT import `wingmate/` (and vice versa)
- `protocol/` MUST NOT import `pilot/` or `wingmate/`
- `flightlog/` MUST NOT import `pilot/` or `wingmate/`

---

## Discoveries

### Discovery S4-01: Protocol Package Interface Design
**Category**: Boundary
**Impact**: Critical
**What**: The `internal/protocol/` package must expose interfaces that both `pilot/` and `wingmate/` can consume without coupling to their specific implementations. This requires careful interface design that abstracts A2A operations.

**Architectural Context**: The protocol package is the communication layer that implements A2A JSON-RPC 2.0. It must be usable by both the initiating (Pilot) and responding (Wingmate) agents without knowing their specific business logic.

**Design Constraint**: Protocol package must define interfaces in terms of A2A primitives (Task, Message, Artifact) and never reference Pilot/Wingmate domain concepts (Mission, Sortie).

**Go Example**:
```go
// internal/protocol/interfaces.go

package protocol

import (
    "context"
    "github.com/<org>/wingmate/pkg/types"
)

// MessageHandler processes incoming A2A messages.
// Implemented by both pilot and wingmate packages.
type MessageHandler interface {
    HandleMessage(ctx context.Context, msg *types.A2AMessage) (*types.A2AResponse, error)
}

// TaskSender sends tasks to remote agents.
// Used by pilot for outbound requests.
type TaskSender interface {
    SendTask(ctx context.Context, url string, task *types.A2ATask) (*types.A2ATaskResult, error)
}

// Server exposes A2A endpoint for incoming requests.
// Used by wingmate to listen for tasks.
type Server interface {
    RegisterHandler(handler MessageHandler)
    ListenAndServe(ctx context.Context, addr string) error
    Shutdown(ctx context.Context) error
}

// Client sends requests to remote A2A agents.
// Used by pilot to communicate with wingmate.
type Client interface {
    TaskSender
    GetAgentCard(ctx context.Context, url string) (*types.AgentCard, error)
    Close() error
}
```

**Action Required**: Define protocol interfaces before implementing pilot/wingmate. Use interface segregation - callers should only depend on the methods they need.

---

### Discovery S4-02: Agent Base Abstraction Pattern
**Category**: Dependency
**Impact**: Critical
**What**: The `internal/agent/` package should provide shared agent functionality that both Pilot and Wingmate can embed or compose. This includes configuration loading, lifecycle management, and common utilities.

**Architectural Context**: Both Pilot and Wingmate are A2A agents with shared concerns: they load configuration, manage lifecycles, interact with Claude API, and log to FlightLog. The agent package abstracts this commonality.

**Design Constraint**: The agent package must not contain Pilot-specific or Wingmate-specific logic. It provides building blocks, not complete implementations. Use composition over inheritance.

**Go Example**:
```go
// internal/agent/base.go

package agent

import (
    "context"
    "github.com/<org>/wingmate/internal/flightlog"
    "github.com/<org>/wingmate/internal/protocol"
    "github.com/<org>/wingmate/pkg/types"
)

// Config holds common agent configuration.
type Config struct {
    Name           string            `json:"name"`
    Port           int               `json:"port"`
    TLS            TLSConfig         `json:"tls"`
    Authentication AuthConfig        `json:"authentication"`
    Claude         ClaudeConfig      `json:"claude"`
    FlightLog      FlightLogConfig   `json:"flight_log"`
    Capabilities   []string          `json:"capabilities"`
}

// Base provides common functionality for all agents.
// Embed this in PilotAgent and WingmateAgent.
type Base struct {
    config    *Config
    flightLog *flightlog.FlightLog
    card      *types.AgentCard
}

// NewBase creates a new base agent with common setup.
func NewBase(cfg *Config) (*Base, error) {
    fl, err := flightlog.New(cfg.FlightLog.Path, cfg.FlightLog.Level)
    if err != nil {
        return nil, fmt.Errorf("creating flight log: %w", err)
    }

    card := &types.AgentCard{
        Name:           cfg.Name,
        Capabilities:   cfg.Capabilities,
        Authentication: cfg.Authentication.ToAgentCardAuth(),
    }

    return &Base{
        config:    cfg,
        flightLog: fl,
        card:      card,
    }, nil
}

// FlightLog returns the agent's flight log for recording.
func (b *Base) FlightLog() *flightlog.FlightLog {
    return b.flightLog
}

// Card returns the agent's A2A card.
func (b *Base) Card() *types.AgentCard {
    return b.card
}

// Shutdown performs graceful cleanup.
func (b *Base) Shutdown(ctx context.Context) error {
    return b.flightLog.Close()
}
```

**Action Required**: Implement agent.Base first. Pilot and Wingmate should embed Base and add role-specific behavior.

---

### Discovery S4-03: Public vs Internal Type Placement
**Category**: Boundary
**Impact**: High
**What**: Types in `pkg/types/` are for external consumption (Agent Card schema, A2A message types). Types in internal packages are implementation details. Clear separation prevents accidental API exposure.

**Architectural Context**: Go's `internal/` convention enforces that these packages cannot be imported by external code. `pkg/types/` is explicitly for types that external tools might need (e.g., a CLI tool parsing Agent Cards, integration tests in separate repos).

**Design Constraint**:
- `pkg/types/`: A2A standard types (AgentCard, Task, Message, Artifact), serialization-focused
- `internal/*/`: Implementation types (FlightLogEntry, Mission, Config structs)
- Never put internal domain concepts in pkg/types

**Go Example**:
```go
// pkg/types/agentcard.go - PUBLIC: external tools may parse these

package types

// AgentCard represents an A2A agent's capabilities and endpoint.
// This follows the A2A specification for agent discovery.
type AgentCard struct {
    Name           string          `json:"name"`
    Description    string          `json:"description,omitempty"`
    URL            string          `json:"url"`
    Version        string          `json:"version"`
    Capabilities   []Capability    `json:"capabilities"`
    Authentication AuthSchemes     `json:"authentication"`
}

// A2ATask represents a task sent via A2A protocol.
type A2ATask struct {
    TaskID  string      `json:"task_id"`
    Message A2AMessage  `json:"message"`
}

// A2AMessage represents a message in A2A protocol.
type A2AMessage struct {
    Role  string        `json:"role"`
    Parts []MessagePart `json:"parts"`
}

// --- INTERNAL types stay in internal/ ---
// internal/flightlog/entry.go

package flightlog

import "time"

// Entry is an internal type for flight log records.
// Not exported to pkg/ - implementation detail.
type Entry struct {
    Timestamp   time.Time   `json:"timestamp"`
    TraceID     string      `json:"trace_id"`
    PilotID     string      `json:"pilot_id"`
    WingmateID  string      `json:"wingmate_id"`
    Direction   Direction   `json:"direction"`
    Summary     string      `json:"summary"`
    PayloadType string      `json:"payload_type"`
    Payload     interface{} `json:"payload"`
    MissionID   string      `json:"mission_id,omitempty"`
}
```

**Action Required**: Create `pkg/types/` with A2A standard types only. All Wingmate-specific types (FlightLogEntry, Mission, etc.) remain internal.

---

### Discovery S4-04: Configuration Propagation Pattern
**Category**: Cross-Cutting
**Impact**: High
**What**: Configuration should flow from `cmd/wingmate/main.go` through dependency injection, not through global state or environment variable reads scattered across packages.

**Architectural Context**: Per Constitution P3 (Engineer Autonomy), configuration is explicit. The main entry point loads config, validates it, and injects it into constructed components. No package should read environment variables directly except through the config layer.

**Design Constraint**:
- Config loading happens once in main.go or a dedicated config package
- Components receive their config subset via constructor injection
- No global config singletons
- Environment variable expansion happens at load time, not use time

**Go Example**:
```go
// cmd/wingmate/main.go

package main

import (
    "github.com/<org>/wingmate/internal/agent"
    "github.com/<org>/wingmate/internal/pilot"
    "github.com/<org>/wingmate/internal/wingmate"
)

func main() {
    // Load config once at startup
    cfg, err := agent.LoadConfig(*configPath)
    if err != nil {
        log.Fatalf("loading config: %v", err)
    }

    // Validate and expand environment variables
    if err := cfg.Validate(); err != nil {
        log.Fatalf("invalid config: %v", err)
    }

    // Inject config into component constructors
    switch cfg.Mode {
    case "pilot":
        p, err := pilot.New(pilot.Config{
            Base:     cfg.Base(),
            PeerURL:  cfg.Pilot.PeerURL,
        })
        if err != nil {
            log.Fatalf("creating pilot: %v", err)
        }
        runPilot(ctx, p)

    case "wingmate":
        w, err := wingmate.New(wingmate.Config{
            Base:     cfg.Base(),
            ListenAddr: cfg.Wingmate.ListenAddr,
        })
        if err != nil {
            log.Fatalf("creating wingmate: %v", err)
        }
        runWingmate(ctx, w)
    }
}

// internal/agent/config.go

package agent

import (
    "encoding/json"
    "os"
)

// LoadConfig reads and parses configuration file.
// Environment variables in values are expanded.
func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("reading config: %w", err)
    }

    // Expand environment variables in JSON
    expanded := os.ExpandEnv(string(data))

    var cfg Config
    if err := json.Unmarshal([]byte(expanded), &cfg); err != nil {
        return nil, fmt.Errorf("parsing config: %w", err)
    }

    return &cfg, nil
}
```

**Action Required**: Implement config loading in agent package. Create typed config structs for each component. Wire via constructors in main.go.

---

### Discovery S4-05: FlightLog as Cross-Cutting Concern
**Category**: Cross-Cutting
**Impact**: Critical
**What**: FlightLog (observability) must be injectable into all components that participate in agent communication. It's a cross-cutting concern that every package uses but none should tightly couple to.

**Architectural Context**: Per Constitution P1 (Observability First), every agent conversation must be traceable. FlightLog is used by protocol (to log raw messages), pilot (to log missions), wingmate (to log handled requests), and agent (to log lifecycle events).

**Design Constraint**:
- FlightLog interface defined in flightlog package
- All other packages depend on the interface, not concrete implementation
- FlightLog must not import pilot or wingmate packages (enforced by architecture)
- Logging methods should accept structured data, not format strings

**Go Example**:
```go
// internal/flightlog/flightlog.go

package flightlog

import (
    "context"
    "encoding/json"
    "os"
    "sync"
    "time"
)

// Direction indicates message flow direction.
type Direction string

const (
    Outbound Direction = "outbound"
    Inbound  Direction = "inbound"
    Error    Direction = "error"
)

// Logger is the interface for flight log operations.
// Components depend on this interface, not the concrete FlightLog.
type Logger interface {
    Record(ctx context.Context, entry Entry)
    RecordError(ctx context.Context, err error, summary string)
    Close() error
}

// FlightLog implements structured observability logging.
type FlightLog struct {
    mu     sync.Mutex
    file   *os.File
    enc    *json.Encoder
    level  Level
}

// New creates a new FlightLog writing to the specified path.
func New(path string, level Level) (*FlightLog, error) {
    f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        return nil, fmt.Errorf("opening flight log: %w", err)
    }

    return &FlightLog{
        file:  f,
        enc:   json.NewEncoder(f),
        level: level,
    }, nil
}

// Record writes a structured entry to the flight log.
func (fl *FlightLog) Record(ctx context.Context, entry Entry) {
    entry.Timestamp = time.Now().UTC()

    // Extract trace ID from context if present
    if traceID := TraceIDFromContext(ctx); traceID != "" {
        entry.TraceID = traceID
    }

    fl.mu.Lock()
    defer fl.mu.Unlock()
    fl.enc.Encode(entry)
}

// RecordError logs an error event.
func (fl *FlightLog) RecordError(ctx context.Context, err error, summary string) {
    fl.Record(ctx, Entry{
        Direction:   Error,
        Summary:     summary,
        PayloadType: "error",
        Payload:     map[string]string{"error": err.Error()},
    })
}
```

**Action Required**: Define Logger interface first. Implement FlightLog. All packages that log should accept Logger interface in their constructors.

---

### Discovery S4-06: Context Propagation for Tracing
**Category**: Cross-Cutting
**Impact**: High
**What**: Go's `context.Context` should propagate trace IDs, cancellation, and deadlines across all package boundaries. This enables distributed tracing and graceful shutdown.

**Architectural Context**: A2A supports OpenTelemetry trace propagation. Context flows from incoming HTTP request through protocol handling, agent processing, Claude API calls, and back. Every function that does I/O or crosses package boundaries should accept context.

**Design Constraint**:
- All public functions that do I/O must accept `context.Context` as first parameter
- Trace IDs are stored in context, extracted by FlightLog
- Cancellation propagates from HTTP request through entire call chain
- Timeouts are set at appropriate layers (HTTP handler, Claude API call)

**Go Example**:
```go
// internal/flightlog/context.go

package flightlog

import "context"

type contextKey string

const traceIDKey contextKey = "wingmate-trace-id"

// WithTraceID returns a context with the trace ID set.
func WithTraceID(ctx context.Context, traceID string) context.Context {
    return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceIDFromContext extracts the trace ID from context.
func TraceIDFromContext(ctx context.Context) string {
    if v := ctx.Value(traceIDKey); v != nil {
        return v.(string)
    }
    return ""
}

// internal/protocol/server.go

package protocol

import (
    "context"
    "net/http"

    "github.com/<org>/wingmate/internal/flightlog"
)

// handleRequest wraps incoming requests with trace context.
func (s *server) handleRequest(w http.ResponseWriter, r *http.Request) {
    // Extract or generate trace ID
    traceID := r.Header.Get("X-Trace-ID")
    if traceID == "" {
        traceID = generateTraceID()
    }

    // Propagate through context
    ctx := flightlog.WithTraceID(r.Context(), traceID)

    // Set response header for correlation
    w.Header().Set("X-Trace-ID", traceID)

    // Pass context to handler (pilot or wingmate)
    resp, err := s.handler.HandleMessage(ctx, parseMessage(r))
    // ... handle response/error
}

// internal/pilot/mission.go

package pilot

import (
    "context"
    "time"
)

// ExecuteMission runs a debugging mission with proper context.
func (p *PilotAgent) ExecuteMission(ctx context.Context, req MissionRequest) (*MissionResult, error) {
    // Respect cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    // Add mission-specific timeout
    ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
    defer cancel()

    // Context flows to all downstream calls
    p.flightLog.Record(ctx, flightlog.Entry{
        Direction: flightlog.Outbound,
        Summary:   fmt.Sprintf("Starting mission: %s", req.Description),
    })

    // Send task to wingmate - context propagates
    result, err := p.client.SendTask(ctx, p.peerURL, req.ToTask())
    // ...
}
```

**Action Required**: Establish context propagation pattern early. Define trace ID context helpers in flightlog. All package interfaces must include context.Context.

---

### Discovery S4-07: Error Handling and Wrapping Strategy
**Category**: Cross-Cutting
**Impact**: High
**What**: Errors should be wrapped with context as they propagate up the call stack, using Go's error wrapping (`fmt.Errorf` with `%w`). Domain-specific error types enable proper handling at boundaries.

**Architectural Context**: Errors cross package boundaries frequently. Protocol errors need to become A2A error responses. FlightLog must capture errors without exposing sensitive details. Callers need to distinguish auth errors from protocol errors from connection errors.

**Design Constraint**:
- Use `fmt.Errorf("context: %w", err)` to wrap errors with context
- Define sentinel errors or error types for each domain
- Protocol package defines A2A error codes
- Never log raw errors that might contain credentials

**Go Example**:
```go
// internal/protocol/errors.go

package protocol

import "errors"

// Sentinel errors for protocol layer.
var (
    ErrAuthentication = errors.New("authentication failed")
    ErrConnection     = errors.New("connection failed")
    ErrProtocol       = errors.New("protocol error")
    ErrTimeout        = errors.New("request timeout")
)

// A2AError represents an error with A2A error code.
type A2AError struct {
    Code    int
    Message string
    Cause   error
}

func (e *A2AError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("A2A error %d: %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("A2A error %d: %s", e.Code, e.Message)
}

func (e *A2AError) Unwrap() error {
    return e.Cause
}

// NewAuthError creates an authentication error.
func NewAuthError(msg string, cause error) error {
    return &A2AError{
        Code:    401,
        Message: msg,
        Cause:   fmt.Errorf("%w: %v", ErrAuthentication, cause),
    }
}

// internal/pilot/mission.go

package pilot

import (
    "errors"
    "github.com/<org>/wingmate/internal/protocol"
)

func (p *PilotAgent) ExecuteMission(ctx context.Context, req MissionRequest) (*MissionResult, error) {
    result, err := p.client.SendTask(ctx, p.peerURL, req.ToTask())
    if err != nil {
        // Log error with context (summary only, not full error)
        p.flightLog.RecordError(ctx, err, "Mission failed during task send")

        // Wrap with mission context for caller
        if errors.Is(err, protocol.ErrAuthentication) {
            return nil, fmt.Errorf("mission %s: authentication with wingmate failed: %w",
                req.ID, err)
        }
        if errors.Is(err, protocol.ErrConnection) {
            return nil, fmt.Errorf("mission %s: could not reach wingmate at %s: %w",
                req.ID, p.peerURL, err)
        }
        return nil, fmt.Errorf("mission %s: unexpected error: %w", req.ID, err)
    }
    // ...
}

// cmd/wingmate/main.go - top level error handling

func runPilot(ctx context.Context, p *pilot.PilotAgent) {
    err := p.ExecuteMission(ctx, mission)
    if err != nil {
        // Check error type for appropriate response
        var a2aErr *protocol.A2AError
        if errors.As(err, &a2aErr) {
            log.Printf("A2A Error [%d]: %s", a2aErr.Code, a2aErr.Message)
            os.Exit(1)
        }
        if errors.Is(err, context.Canceled) {
            log.Printf("Mission canceled")
            os.Exit(0)
        }
        log.Printf("Mission failed: %v", err)
        os.Exit(1)
    }
}
```

**Action Required**: Define error types in protocol package. Establish wrapping convention. Create error handling helpers for common patterns.

---

### Discovery S4-08: Build Order and Dependency Graph
**Category**: Dependency
**Impact**: Medium
**What**: The package dependency graph determines build order and highlights coupling. A clean acyclic graph ensures packages can be tested independently.

**Architectural Context**: Go compiles packages in dependency order. Circular dependencies are compile errors. The architecture enforces this through module boundaries. Understanding the graph helps identify where interfaces break cycles.

**Design Constraint**:
- `pkg/types` has no internal dependencies (leaf node)
- `internal/flightlog` depends only on `pkg/types`
- `internal/protocol` depends on `pkg/types` and `internal/flightlog`
- `internal/agent` depends on all above
- `internal/pilot` and `internal/wingmate` depend on `agent`, `protocol`, `flightlog`
- `cmd/wingmate` depends on everything (root node)

**Dependency Graph**:
```
                    +-----------------+
                    | cmd/wingmate    |
                    +--------+--------+
                             |
              +--------------+--------------+
              |                             |
     +--------v--------+          +---------v-------+
     | internal/pilot  |          | internal/wingmate|
     +--------+--------+          +---------+-------+
              |                             |
              +--------------+--------------+
                             |
                    +--------v--------+
                    | internal/agent  |
                    +--------+--------+
                             |
              +--------------+--------------+
              |                             |
     +--------v--------+          +---------v---------+
     |internal/protocol|          | internal/flightlog|
     +--------+--------+          +---------+---------+
              |                             |
              +--------------+--------------+
                             |
                    +--------v--------+
                    |   pkg/types     |
                    +-----------------+
```

**Build Order** (bottom to top):
1. `pkg/types` - no dependencies
2. `internal/flightlog` - depends on types
3. `internal/protocol` - depends on types, flightlog
4. `internal/agent` - depends on types, flightlog, protocol
5. `internal/pilot` - depends on agent, protocol, flightlog
6. `internal/wingmate` - depends on agent, protocol, flightlog (parallel with pilot)
7. `cmd/wingmate` - depends on pilot, wingmate

**Go Example**:
```go
// go.mod at project root

module github.com/<org>/wingmate

go 1.21

// No external dependencies initially - add as needed:
// require (
//     github.com/a2aproject/a2a-go v0.x.x
//     go.opentelemetry.io/otel v1.x.x
// )

// Makefile

.PHONY: build test lint

# Build order respects dependency graph
build:
	go build -v ./pkg/...
	go build -v ./internal/flightlog/...
	go build -v ./internal/protocol/...
	go build -v ./internal/agent/...
	go build -v ./internal/pilot/...
	go build -v ./internal/wingmate/...
	go build -v -o bin/wingmate ./cmd/wingmate

# Test packages in isolation
test:
	go test -v ./pkg/types/...
	go test -v ./internal/flightlog/...
	go test -v ./internal/protocol/...
	go test -v ./internal/agent/...
	go test -v ./internal/pilot/...
	go test -v ./internal/wingmate/...

# Verify no import cycles
lint:
	go vet ./...
	@echo "Checking for import violations..."
	@! grep -r "internal/pilot" internal/protocol internal/flightlog internal/wingmate
	@! grep -r "internal/wingmate" internal/protocol internal/flightlog internal/pilot
	@echo "Import boundaries verified."
```

**Action Required**: Implement packages in dependency order. Create Makefile targets for building and testing. Add CI checks to verify import boundaries are not violated.

---

## Summary

| Discovery | Category | Impact | Key Insight |
|-----------|----------|--------|-------------|
| S4-01 | Boundary | Critical | Protocol exposes interfaces, not concrete types |
| S4-02 | Dependency | Critical | Agent provides composable base, not inheritance |
| S4-03 | Boundary | High | pkg/types for A2A standards, internal for domain |
| S4-04 | Cross-Cutting | High | Config flows via constructor injection from main |
| S4-05 | Cross-Cutting | Critical | FlightLog interface enables observability everywhere |
| S4-06 | Cross-Cutting | High | Context propagates traces, cancellation, deadlines |
| S4-07 | Cross-Cutting | High | Errors wrap with context, typed for handling |
| S4-08 | Dependency | Medium | Acyclic graph enables independent testing |

---

## Implementation Sequence Recommendation

Based on dependency analysis, implement in this order:

1. **pkg/types** - A2A type definitions (no dependencies)
2. **internal/flightlog** - Logger interface and implementation
3. **internal/protocol** - A2A client/server interfaces and JSON-RPC
4. **internal/agent** - Base agent, config loading
5. **internal/pilot** and **internal/wingmate** - Role-specific agents (parallel)
6. **cmd/wingmate** - CLI entry point and wiring

This order ensures each package can be tested independently before its dependents are implemented.
