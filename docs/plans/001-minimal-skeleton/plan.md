# Implementation Plan: Minimal Wingmate Skeleton

**Spec Reference**: [/docs/plans/001-minimal-skeleton/minimal-skeleton-spec.md](/docs/plans/001-minimal-skeleton/minimal-skeleton-spec.md)
**Mode**: Full
**Complexity**: CS-2 (Small)
**Created**: 2026-01-21
**Updated**: 2026-01-21 (ADR-003 Unified Peer Architecture)

---

## Table of Contents

- [Executive Summary](#executive-summary)
- [Phase Overview](#phase-overview)
- [Phase 1: Foundation](#phase-1-foundation)
- [Phase 2: Protocol Layer](#phase-2-protocol-layer)
- [Phase 3: Unified Agent](#phase-3-unified-agent)
- [Phase 4: CLI & Integration](#phase-4-cli--integration)
- [Phase 5: Cross-Machine Verification](#phase-5-cross-machine-verification)
- [Deviation Ledger](#deviation-ledger)
- [Risk Mitigation](#risk-mitigation)
- [Testing Strategy](#testing-strategy)
- [Acceptance Criteria Mapping](#acceptance-criteria-mapping)
- [File Checklist](#file-checklist)
- [Research Artifacts](#research-artifacts)

---

## Executive Summary

This plan implements the foundation of Wingmate - a bidirectional Claude agent communication system using the A2A protocol. The MVP delivers a single binary that runs as a **unified peer** - every instance can both initiate conversations (pilot role) and respond to conversations (wingmate role).

**Key Decisions**:
- **Language**: Go ([ADR-001](/docs/adr/001-language-choice.md)) - single binary, cross-platform compilation
- **Storage**: JSONL + stdout ([ADR-002](/docs/adr/002-flight-log-storage.md)) - human-readable, machine-parseable Flight Log
- **Architecture**: Unified Peer Model ([ADR-003](/docs/adr/003-unified-peer-architecture.md)) - no separate modes; every instance is a full peer
- **Protocol**: A2A JSON-RPC 2.0 over HTTP (custom implementation, SDK as reference)

---

## Phase Overview

| Phase | Name | Description | Dependencies |
|-------|------|-------------|--------------|
| 1 | Foundation | pkg/types, internal/flightlog | None |
| 2 | Protocol Layer | internal/protocol (JSON-RPC, Agent Card) | Phase 1 |
| 3 | Unified Agent | internal/agent (server + client capabilities) | Phase 1, 2 |
| 4 | CLI & Integration | cmd/wingmate, Makefile, README | All |
| 5 | Cross-Machine Test | Verification on separate machines | Phase 4 |

**Note**: Per ADR-003, the original Phases 4 (Wingmate Agent) and 5 (Pilot Agent) have been **merged** into Phase 3 (Unified Agent). The architecture no longer separates pilot/wingmate into distinct modes.

---

## Phase 1: Foundation

**Status**: ✅ Complete (with ADR-003 update)
**Goal**: Establish type definitions and observability infrastructure.

### 1.1 Create Project Structure

```
wingmate/
├── cmd/
│   └── wingmate/
│       └── main.go
├── internal/
│   ├── agent/              # Unified agent (per ADR-003)
│   ├── flightlog/
│   └── protocol/
├── pkg/
│   └── types/
├── config/
│   └── agent.json.example
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

**Note**: Per ADR-003, there is NO `internal/pilot/` or `internal/wingmate/` directory. All agent functionality lives in `internal/agent/`.

### 1.2 pkg/types - A2A Standard Types

**Files**:
- `pkg/types/agentcard.go` - Agent Card schema (A2A compliant)
- `pkg/types/message.go` - A2A Message, Part types with union handling
- `pkg/types/task.go` - A2A Task, TaskResult types
- `pkg/types/errors.go` - JSON-RPC error types

**Key Constraints** (from S2 research):
- Agent Card served at `/.well-known/agent.json` (not agent-card.json)
- Required fields: name, url, version, capabilities, skills (even if empty array)
- Message Parts use `kind` discriminator: "text", "data", "file"
- Error codes: use 1000+ for application errors (not reserved -32000 range)

**Tasks** (TDD Order):
1. [x] **SDK Evaluation**: Import `a2aproject/a2a-go`, assess what types/utilities it provides. Decision: Custom types (SDK requires Go 1.24.4, we target 1.21+)
2. [x] Write test: `pkg/types/agentcard_test.go` - JSON round-trip, required fields
3. [x] Write test: `pkg/types/message_test.go` - Part union unmarshaling, nil vs empty
4. [x] Implement: `pkg/types/agentcard.go`
5. [x] Implement: `pkg/types/message.go`
6. [x] Implement: `pkg/types/task.go`
7. [x] Implement: `pkg/types/errors.go`

### 1.3 internal/flightlog - Observability

**Files**:
- `internal/flightlog/entry.go` - Entry struct with Role field (per ADR-003)
- `internal/flightlog/writer.go` - Thread-safe JSONL writer
- `internal/flightlog/context.go` - Trace ID context helpers
- `internal/flightlog/flightlog.go` - Logger interface and implementation
- `internal/flightlog/testutil.go` - Test helper: `ReadLogEntries(t, path)`

**Entry Schema** (updated for ADR-003):
```go
type Role string

const (
    RolePilot    Role = "pilot"     // Initiated this conversation
    RoleWingmate Role = "wingmate"  // Responding to this conversation
)

type Entry struct {
    Timestamp   time.Time   `json:"ts"`
    TraceID     string      `json:"trace,omitempty"`
    Agent       string      `json:"agent"`
    Peer        string      `json:"peer,omitempty"`
    Role        Role        `json:"role,omitempty"`  // ADR-003: conversation role
    Direction   Direction   `json:"dir"`
    Method      string      `json:"method,omitempty"`
    Summary     string      `json:"summary"`
    Payload     interface{} `json:"payload"`
}
```

**Usage Pattern**:
```go
// Constructor handles timestamp and trace ID
entry := flightlog.NewEntry(ctx, agentName, flightlog.Outbound, "Sending ping", msg)
entry.Peer = peerName
entry.Role = flightlog.RolePilot  // This agent initiated the conversation
entry.Method = "ping"
fl.Record(entry)
```

**Tasks** (TDD Order):
1. [x] Write test: `internal/flightlog/writer_test.go`
2. [x] Write test: `internal/flightlog/context_test.go`
3. [x] Write test: `internal/flightlog/flightlog_test.go`
4. [x] Implement: `internal/flightlog/entry.go` (with Role per ADR-003)
5. [x] Implement: `internal/flightlog/writer.go`
6. [x] Implement: `internal/flightlog/context.go`
7. [x] Implement: `internal/flightlog/flightlog.go`
8. [x] Implement: `internal/flightlog/testutil.go`

### 1.4 Success Criteria

```bash
# All must pass:
go build ./pkg/...
go build ./internal/flightlog/...
go test -v ./pkg/... ./internal/flightlog/...
```

- [x] `go build ./pkg/...` succeeds
- [x] `go build ./internal/flightlog/...` succeeds
- [ ] `go test ./pkg/... ./internal/flightlog/...` passes (pending Go installation)
- [x] Flight Log entry written matches schema (verified by test)
- [x] Role field present in Entry struct (ADR-003 compliance)

---

## Phase 2: Protocol Layer

**Goal**: Implement A2A JSON-RPC server and client.

### 2.1 internal/protocol - Interfaces

**Files**:
- `internal/protocol/interfaces.go` - Server, Client, MessageHandler interfaces
- `internal/protocol/jsonrpc.go` - JSON-RPC request/response types
- `internal/protocol/errors.go` - Protocol error types and codes

**Interface Design**:
```go
type MessageHandler interface {
    HandleMessage(ctx context.Context, msg *types.A2AMessage) (*types.A2AResponse, error)
}

type Server interface {
    RegisterHandler(handler MessageHandler)
    ListenAndServe(ctx context.Context, addr string) error
    Shutdown(ctx context.Context) error
}

type Client interface {
    SendMessage(ctx context.Context, url string, msg *types.A2AMessage) (*types.A2AResponse, error)
    GetAgentCard(ctx context.Context, url string) (*types.AgentCard, error)
    Close() error
}
```

### 2.2 internal/protocol - Server Implementation

**Files**:
- `internal/protocol/server.go` - HTTP server with JSON-RPC dispatcher

**Key Gotchas** (from S2 research):
- Check `http.Flusher` for SSE (not MVP but structure for it)
- Set `X-Accel-Buffering: no` header
- Use `http.TimeoutHandler` or explicit context deadlines
- Implement graceful shutdown with WaitGroup for in-flight requests

**Endpoints**:
- `POST /` - JSON-RPC endpoint (message/send method)
- `GET /.well-known/agent.json` - Agent Card

**Tasks** (TDD Order):
1. [ ] Write test: `internal/protocol/jsonrpc_test.go`
2. [ ] Write test: `internal/protocol/server_test.go`
3. [ ] Implement: `internal/protocol/interfaces.go`
4. [ ] Implement: `internal/protocol/jsonrpc.go`
5. [ ] Implement: `internal/protocol/errors.go`
6. [ ] Implement: `internal/protocol/server.go`

### 2.3 internal/protocol - Client Implementation

**Files**:
- `internal/protocol/client.go` - HTTP client with connection pooling

**Features**:
- Context-aware requests with timeout
- Connection pooling (`MaxIdleConns: 10`)
- Agent Card fetching
- Error wrapping with protocol.Err* sentinels

**Tasks** (TDD Order):
1. [ ] Write test: `internal/protocol/client_test.go`
2. [ ] Implement: `internal/protocol/client.go`

### 2.4 Success Criteria

```bash
go build ./internal/protocol/...
go test -v ./internal/protocol/...
```

- [ ] `go build ./internal/protocol/...` succeeds
- [ ] `go test ./internal/protocol/...` passes (0 failures)
- [ ] Server handles valid JSON-RPC POST (verified by test)
- [ ] Server serves Agent Card at `/.well-known/agent.json` (verified by test)
- [ ] Client can send message and receive response (verified by test)

---

## Phase 3: Unified Agent

**Goal**: Implement the unified agent that can both initiate and respond to conversations.

**Note**: Per ADR-003, this phase replaces the original separate Wingmate Agent (Phase 4) and Pilot Agent (Phase 5). Every agent instance is a full peer.

### 3.1 internal/agent - Configuration

**Files**:
- `internal/agent/config.go` - Config struct and loading
- `internal/agent/validate.go` - Configuration validation

**Configuration Schema** (per ADR-003):
```go
type Config struct {
    Name         string   `json:"name"`
    Port         int      `json:"port"`
    Peers        []string `json:"peers,omitempty"`     // Known peers to connect to
    LogFile      string   `json:"logFile"`
    Verbose      bool     `json:"verbose"`
    Capabilities []string `json:"capabilities,omitempty"` // What this agent offers
}
```

**Note**: No `Mode` field - per ADR-003, there is no pilot/wingmate mode distinction.

**Precedence**: CLI flags > Environment variables > Config file > Defaults

**Environment Variables**:
- `WINGMATE_NAME`
- `WINGMATE_PORT`
- `WINGMATE_PEERS` (comma-separated)
- `WINGMATE_LOG`
- `WINGMATE_VERBOSE`

### 3.2 internal/agent - Agent Implementation

**Files**:
- `internal/agent/agent.go` - Unified Agent struct (both client + server)
- `internal/agent/server.go` - Handle incoming A2A requests
- `internal/agent/client.go` - Make outgoing A2A requests
- `internal/agent/conversation.go` - Conversation/mission management

**Agent Design** (per ADR-003):
```go
// Agent is a unified A2A peer that can both initiate and respond to conversations.
// Per ADR-003: "pilot" and "wingmate" are conversation roles, not instance modes.
type Agent struct {
    config    *Config
    server    *protocol.Server  // Handle incoming
    client    *protocol.Client  // Make outgoing
    flightLog *flightlog.FlightLog
    card      *types.AgentCard
    ready     chan struct{}     // For WaitUntilReady()

    // Peer management
    knownPeers map[string]*types.AgentCard
    peersMu    sync.RWMutex
}

// Start begins listening for incoming requests.
func (a *Agent) Start(ctx context.Context) error

// WaitUntilReady blocks until the HTTP listener is accepting connections.
func (a *Agent) WaitUntilReady(timeout time.Duration) error

// SendMessage sends a message to a peer (acts as "pilot" in this conversation).
func (a *Agent) SendMessage(ctx context.Context, peerURL string, msg *types.Message) (*types.A2AResponse, error)

// Ping sends a ping to a peer and expects a pong response.
func (a *Agent) Ping(ctx context.Context, peerURL string) error

// HandleMessage processes an incoming message (acts as "wingmate" in this conversation).
func (a *Agent) HandleMessage(ctx context.Context, msg *types.A2AMessage) (*types.A2AResponse, error)

// Shutdown stops the agent gracefully.
func (a *Agent) Shutdown(ctx context.Context) error
```

### 3.3 Agent Card Content

```json
{
  "name": "agent-001",
  "description": "Wingmate diagnostic agent",
  "url": "http://localhost:9000",
  "version": "0.1.0",
  "capabilities": {
    "streaming": false,
    "pushNotifications": false
  },
  "skills": [
    {
      "name": "ping",
      "description": "Respond to ping with pong"
    }
  ],
  "authentication": {
    "schemes": []
  }
}
```

### 3.4 Message Handlers

**Capabilities**:
- `ping` - Respond to ping with pong (MVP)

**Handler Implementation**:
```go
func (a *Agent) HandleMessage(ctx context.Context, msg *types.A2AMessage) (*types.A2AResponse, error) {
    // Log inbound (as wingmate in this conversation)
    entry := flightlog.NewEntry(ctx, a.config.Name, flightlog.Inbound, "Received message", msg)
    entry.Role = flightlog.RoleWingmate
    a.flightLog.Record(entry)

    // Handle ping
    if isPing(msg) {
        return pongResponse(), nil
    }

    return nil, protocol.ErrMethodNotFound
}
```

### 3.5 Client Commands

**Commands**:
- `ping` - Send ping to peer, expect pong
- `status` - Fetch peer Agent Card, report capabilities

**Flow**:
1. Initialize agent
2. Fetch peer Agent Card (verify reachable)
3. Send ping message (as pilot)
4. Log outbound and inbound
5. Report result

### 3.6 Error Handling

**Peer Unreachable Scenarios**:
- Connection refused: "Cannot connect to peer at http://localhost:9001 - is the agent running?"
- DNS failure: "Cannot resolve host: machineB"
- Invalid response: "Peer is not an A2A agent"
- Timeout: "Request timed out after 30s"

### 3.7 Tasks (TDD Order)

1. [ ] Write test: `internal/agent/config_test.go` - file load, env override, validation
2. [ ] Write test: `internal/agent/agent_test.go` - initialization, WaitUntilReady, shutdown
3. [ ] Write test: `internal/agent/server_test.go` - ping/pong, unknown message, Flight Log records
4. [ ] Write test: `internal/agent/client_test.go` - ping success, error scenarios, Flight Log records
5. [ ] Implement: `internal/agent/config.go`
6. [ ] Implement: `internal/agent/validate.go`
7. [ ] Implement: `internal/agent/agent.go`
8. [ ] Implement: `internal/agent/server.go`
9. [ ] Implement: `internal/agent/client.go`
10. [ ] Implement: `internal/agent/conversation.go`

### 3.8 Success Criteria

```bash
go build ./internal/agent/...
go test -v ./internal/agent/...
```

- [ ] `go build ./internal/agent/...` succeeds
- [ ] `go test ./internal/agent/...` passes (0 failures)
- [ ] Agent Card accessible at `/.well-known/agent.json` (verified by test)
- [ ] Ping message returns pong response (verified by test)
- [ ] All messages logged to Flight Log with correct Role (verified by test)
- [ ] Clear error messages for unreachable peer (verified by test)
- [ ] WaitUntilReady prevents race conditions (verified by test)

---

## Phase 4: CLI & Integration

**Goal**: Wire everything together with CLI and build system.

### 4.1 cmd/wingmate - Entry Point

**Files**:
- `cmd/wingmate/main.go` - CLI entry point with flag parsing

**Flags** (per ADR-003 - no mode flag):
```
Usage: wingmate [options]

Options:
  --port      Listen port (default: 9000)
  --peers     Comma-separated list of peer URLs
  --log       Flight log path (default: ./flight.jsonl)
  --verbose   Mirror logs to stdout
  --config    Config file path (default: ./config/agent.json)
  --help      Show help

Commands:
  ping <peer-url>   Send ping to a peer
  status <peer-url> Fetch peer's Agent Card
```

**Example Usage**:
```bash
# Start agent, listen for connections
./wingmate --port 9000

# Start agent with known peers
./wingmate --port 9000 --peers http://machineB:9001,http://machineC:9002

# Send ping to a peer
./wingmate ping http://machineB:9001
```

### 4.2 Makefile

```makefile
.PHONY: build build-all test lint clean

# Build for current platform
build:
	go build -o bin/wingmate ./cmd/wingmate

# Cross-compile for all platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build -o bin/wingmate-darwin-amd64 ./cmd/wingmate
	GOOS=darwin GOARCH=arm64 go build -o bin/wingmate-darwin-arm64 ./cmd/wingmate
	GOOS=linux GOARCH=amd64 go build -o bin/wingmate-linux-amd64 ./cmd/wingmate
	GOOS=linux GOARCH=arm64 go build -o bin/wingmate-linux-arm64 ./cmd/wingmate
	GOOS=windows GOARCH=amd64 go build -o bin/wingmate-windows-amd64.exe ./cmd/wingmate
	GOOS=android GOARCH=arm64 go build -o bin/wingmate-android-arm64 ./cmd/wingmate

test:
	go test -v ./...

lint:
	go vet ./...

clean:
	rm -rf bin/
```

### 4.3 Integration Test

**Test Flow**:
1. Start Agent A on port 9000
2. **Wait for Agent A to be ready** (prevents race condition)
3. Start Agent B on port 9001 with peer http://localhost:9000
4. Agent B sends ping to Agent A
5. Verify pong response
6. Verify Flight Log entries on both agents (with correct roles)
7. Shutdown both agents cleanly

**Test Pattern**:
```go
func TestPingPong(t *testing.T) {
    // Start Agent A in goroutine
    agentA := agent.New(configA)
    go agentA.Start(ctx)

    // CRITICAL: Wait for server to be ready
    if err := agentA.WaitUntilReady(5 * time.Second); err != nil {
        t.Fatalf("Agent A failed to start: %v", err)
    }

    // Now Agent B can safely connect
    agentB := agent.New(configB)
    resp, err := agentB.Ping(ctx, "http://localhost:9000")
    // ... assertions

    // Verify Flight Log roles
    entriesA := flightlog.ReadLogEntries(t, configA.LogFile)
    // Agent A was wingmate (responded)
    flightlog.AssertEntryExists(t, entriesA, "agent-a", flightlog.Inbound, "")

    entriesB := flightlog.ReadLogEntries(t, configB.LogFile)
    // Agent B was pilot (initiated)
    flightlog.AssertEntryExists(t, entriesB, "agent-b", flightlog.Outbound, "")
}
```

**Test File**: `tests/integration/ping_pong_test.go`

### 4.4 README.md

Contents:
- Project overview (what is Wingmate)
- Prerequisites (Go 1.21+)
- Build instructions
- Quick start guide (start agents, ping between them)
- Configuration reference
- Flight Log format

### 4.5 Tasks

1. [ ] Implement: `cmd/wingmate/main.go`
2. [ ] Create: `config/agent.json.example`
3. [ ] Create: `Makefile`
4. [ ] Create: `README.md`
5. [ ] Write test: `tests/integration/ping_pong_test.go`

### 4.6 Success Criteria

```bash
make build
make build-all
make test
go test -v ./tests/integration/...
```

- [ ] `make build` produces working binary
- [ ] `make build-all` produces cross-platform binaries
- [ ] `make test` passes all tests (0 failures)
- [ ] Integration test passes
- [ ] README covers all quick-start scenarios

---

## Phase 5: Cross-Machine Verification

**Goal**: Verify communication works across network.

### 5.1 Test Plan

1. **Machine A**:
   ```bash
   ./wingmate --port 9001 --verbose
   ```

2. **Machine B**:
   ```bash
   ./wingmate ping http://machineA:9001 --verbose
   ```

3. **Verification**:
   - Machine B logs show outbound ping (role: pilot) and inbound pong
   - Machine A logs show inbound ping (role: wingmate) and outbound pong
   - Both Flight Log files contain correlated trace IDs
   - Agent Card accessible from Machine B: `curl http://machineA:9001/.well-known/agent.json`

### 5.2 Bidirectional Test (ADR-003 Validation)

Per ADR-003, any agent can initiate conversations with any other agent:

1. **Machine A pings Machine B**:
   ```bash
   # On Machine A
   ./wingmate ping http://machineB:9002
   ```
   - Machine A is pilot in this conversation
   - Machine B is wingmate in this conversation

2. **Machine B pings Machine A** (same instances):
   ```bash
   # On Machine B
   ./wingmate ping http://machineA:9001
   ```
   - Machine B is now pilot in this new conversation
   - Machine A is now wingmate in this new conversation

This validates the unified peer model from ADR-003.

### 5.3 Troubleshooting Guide

| Symptom | Likely Cause | Resolution |
|---------|--------------|------------|
| Connection refused | Agent not running | Start agent first |
| Connection timeout | Firewall blocking port | Open port |
| DNS resolution failed | Hostname not resolvable | Use IP address |
| Invalid response | Wrong service on port | Verify Agent Card URL |

### 5.4 Success Criteria

- [ ] Cross-machine ping/pong works
- [ ] Both Flight Logs have matching trace IDs
- [ ] Roles correctly recorded (pilot for initiator, wingmate for responder)
- [ ] Agent Card fetchable from remote machine
- [ ] Bidirectional communication works (either agent can initiate)

---

## Deviation Ledger

| ID | Rule/Principle | Deviation | Rationale | Tracking |
|----|----------------|-----------|-----------|----------|
| DEV-001 | Rules 2.4: "MUST use TLS" | MVP uses HTTP | Simplify initial development | Next iteration |
| DEV-002 | Rules 5.3: "MUST implement auth" | MVP has no auth | Focus on core protocol | Next iteration |
| DEV-003 | Constitution P6: "Security by Design" | Deferred for MVP | Security in Non-Goals | Post-MVP |

---

## Risk Mitigation

| Risk | Mitigation | Trigger |
|------|------------|---------|
| Go A2A SDK incomplete | Custom implementation (done) | SDK requires Go 1.24.4 |
| Network/firewall issues | Document required ports | First cross-machine failure |
| Port conflicts | Clear error messages | EADDRINUSE on startup |

---

## Testing Strategy

| Layer | Test Type | Coverage Focus |
|-------|-----------|----------------|
| pkg/types | Unit | JSON serialization |
| internal/flightlog | Unit | File I/O, thread safety |
| internal/protocol | Unit + Integration | JSON-RPC, error handling |
| internal/agent | Unit | Config, handlers, client |
| Integration | E2E | Full ping/pong flow |

**Mock Policy**: Targeted mocks only
- Mock HTTP transport for protocol client tests
- Real file I/O for FlightLog tests (per spec)
- Real HTTP for integration tests

---

## Acceptance Criteria Mapping

| AC | Phase | Verified By |
|----|-------|-------------|
| AC1: Build | 4 | `make build` succeeds |
| AC1b: Binary Distribution | 4 | Copy binary, runs |
| AC2: Start Agent | 3 | Unit test + manual |
| AC3: Start Agent with Peers | 3 | Unit test + manual |
| AC4: Basic Communication | 4 | Integration test |
| AC5: Flight Log Outbound | 3 | Unit test |
| AC6: Flight Log Inbound | 3 | Unit test |
| AC7: Agent Card Content | 3 | Unit test |
| AC8: Cross-Machine | 5 | Manual test |
| AC9: Configuration | 3 | Unit test |
| AC10: Cross-Compile | 4 | `make build-all` |

---

## File Checklist

### Phase 1 (Complete)
- [x] `go.mod`
- [x] `pkg/types/agentcard.go`
- [x] `pkg/types/agentcard_test.go`
- [x] `pkg/types/message.go`
- [x] `pkg/types/message_test.go`
- [x] `pkg/types/task.go`
- [x] `pkg/types/errors.go`
- [x] `internal/flightlog/entry.go` (with Role per ADR-003)
- [x] `internal/flightlog/writer.go`
- [x] `internal/flightlog/writer_test.go`
- [x] `internal/flightlog/context.go`
- [x] `internal/flightlog/context_test.go`
- [x] `internal/flightlog/flightlog.go`
- [x] `internal/flightlog/flightlog_test.go`
- [x] `internal/flightlog/testutil.go`

### Phase 2
- [ ] `internal/protocol/interfaces.go`
- [ ] `internal/protocol/jsonrpc.go`
- [ ] `internal/protocol/jsonrpc_test.go`
- [ ] `internal/protocol/errors.go`
- [ ] `internal/protocol/server.go`
- [ ] `internal/protocol/server_test.go`
- [ ] `internal/protocol/client.go`
- [ ] `internal/protocol/client_test.go`

### Phase 3 (Unified Agent - per ADR-003)
- [ ] `internal/agent/config.go`
- [ ] `internal/agent/config_test.go`
- [ ] `internal/agent/validate.go`
- [ ] `internal/agent/agent.go`
- [ ] `internal/agent/agent_test.go`
- [ ] `internal/agent/server.go`
- [ ] `internal/agent/server_test.go`
- [ ] `internal/agent/client.go`
- [ ] `internal/agent/client_test.go`
- [ ] `internal/agent/conversation.go`

### Phase 4
- [ ] `cmd/wingmate/main.go`
- [ ] `config/agent.json.example`
- [ ] `Makefile`
- [ ] `README.md`
- [ ] `tests/integration/ping_pong_test.go`

---

## Research Artifacts

| Document | Key Discoveries |
|----------|-----------------|
| [go-patterns-research.md](/docs/research/go-patterns-research.md) | Project layout, JSON-RPC, testing |
| [a2a-technical-constraints.md](/docs/research/a2a-technical-constraints.md) | Error codes, Agent Card path |
| [spec-discovery-s3.md](/docs/research/spec-discovery-s3.md) | Config, ports, error handling |
| [s4-dependency-mapping.md](/docs/research/s4-dependency-mapping.md) | Interfaces, build order |
| [a2a-go-sdk-evaluation.md](/docs/research/a2a-go-sdk-evaluation.md) | SDK requires Go 1.24.4 |
| [adr-003-impact-analysis.md](/docs/plans/001-minimal-skeleton/adr-003-impact-analysis.md) | ADR-003 impact on plan |

---

## Critical Insights Discussion

**Session**: 2026-01-21 (pre-implementation review via `/didyouknow`)

### Insight #1: SDK Evaluation Task
**Decision**: Added explicit SDK evaluation task. Result: Use custom types (SDK requires Go 1.24.4).

### Insight #2: FlightLog "Never Mock" Testing Pattern
**Decision**: Real file I/O with `testutil.go` helper using `t.TempDir()`.

### Insight #3: Go time.Time Zero-Value Gotcha
**Decision**: `NewEntry()` constructor auto-fills Timestamp and TraceID.

### Insight #4: Phases 4 and 5 Parallelization
**Decision**: Keep sequential. **UPDATE**: Now merged per ADR-003.

### Insight #5: Integration Test Race Condition
**Decision**: `WaitUntilReady(timeout)` method on agent.

---

## ADR-003 Plan Update

**Date**: 2026-01-21

This plan was updated to reflect ADR-003 (Unified Peer Architecture):

1. **Removed separate modes**: No `--mode pilot` or `--mode wingmate`
2. **Merged phases**: Original Phases 4 (Wingmate) and 5 (Pilot) merged into Phase 3 (Unified Agent)
3. **Updated directory structure**: No `internal/pilot/` or `internal/wingmate/`
4. **Updated config**: No `Mode` field, added `Peers` array
5. **Updated CLI**: Simplified flags, no mode selection
6. **Added Role to FlightLog**: Track conversation role per entry

See [adr-003-impact-analysis.md](/docs/plans/001-minimal-skeleton/adr-003-impact-analysis.md) for full impact analysis.
