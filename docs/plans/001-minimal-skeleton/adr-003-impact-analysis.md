# ADR-003 Impact Analysis

**Date**: 2026-01-21
**ADR Reference**: [ADR-003: Unified Peer Architecture](/docs/adr/003-unified-peer-architecture.md)
**Plan Reference**: [001-minimal-skeleton/plan.md](/docs/plans/001-minimal-skeleton/plan.md)

---

## Executive Summary

ADR-003 introduces a **major architectural change**: removing the `--mode pilot` and `--mode wingmate` distinction. Every wingmate instance becomes a **full peer** that can both initiate and respond to conversations.

**Key Findings**:
- **Phase 1 (Foundation)**: ✅ **MINIMAL IMPACT** - One field addition to Entry struct
- **Phases 2-3**: ✅ **MINIMAL IMPACT** - Protocol and config changes
- **Phases 4-5**: ❌ **MAJOR IMPACT** - Must be **MERGED** into single unified agent
- **Phase 6**: 🔄 **MODERATE IMPACT** - CLI and main.go changes
- **Phase 7**: ✅ **NO IMPACT** - Cross-machine testing unchanged

---

## Phase 1 Impact Analysis

### Phase 1 Work Completed

All 16 files created:
- `go.mod`
- `pkg/types/agentcard.go`, `agentcard_test.go`
- `pkg/types/message.go`, `message_test.go`
- `pkg/types/task.go`
- `pkg/types/errors.go`
- `internal/flightlog/entry.go`, `writer.go`, `context.go`, `flightlog.go`, `testutil.go`
- `internal/flightlog/writer_test.go`, `context_test.go`, `flightlog_test.go`

### Impact Assessment

#### pkg/types/ - NO CHANGES NEEDED ✅

| File | Impact | Reason |
|------|--------|--------|
| `agentcard.go` | None | AgentCard declares capabilities regardless of peer model |
| `message.go` | None | Messages are protocol-level, not mode-dependent |
| `task.go` | None | Task states are conversation-level |
| `errors.go` | None | Error codes are protocol-level |

**Rationale**: These are A2A protocol types. The unified peer model doesn't change the protocol itself - it changes how instances are deployed and configured.

#### internal/flightlog/ - MINOR ADDITION NEEDED ⚠️

**Current Entry struct**:
```go
type Entry struct {
    Timestamp time.Time   `json:"ts"`
    TraceID   string      `json:"trace,omitempty"`
    Agent     string      `json:"agent"`
    Peer      string      `json:"peer,omitempty"`
    Direction Direction   `json:"dir"`
    Method    string      `json:"method,omitempty"`
    Summary   string      `json:"summary"`
    Payload   interface{} `json:"payload"`
}
```

**ADR-003 Flight Log Example**:
```json
{
  "timestamp": "2026-01-21T14:32:17.123Z",
  "direction": "outbound",
  "local_agent": "agent-device-01",
  "remote_agent": "agent-server-02",
  "role": "pilot",
  "summary": "Requested server logs for last 10 minutes"
}
```

**Required Change**: Add `Role` field to Entry struct

```go
// Role indicates this agent's role in the conversation.
// Per ADR-003, pilot/wingmate are conversation roles, not instance modes.
type Role string

const (
    RolePilot    Role = "pilot"     // Initiated this conversation
    RoleWingmate Role = "wingmate"  // Responding to this conversation
)

type Entry struct {
    // ... existing fields ...

    // Role indicates whether this agent initiated (pilot) or responded (wingmate)
    // in this specific conversation. Per ADR-003.
    Role Role `json:"role,omitempty"`
}
```

**Impact Level**: Minor - additive change, backwards compatible

---

## Phases 2-3 Impact Analysis

### Phase 2: Protocol Layer - MINOR CHANGES ⚠️

**Changes Needed**:
1. Server interface should support bidirectional operation
2. Client interface already supports this (can send to any peer)
3. No structural changes to JSON-RPC implementation

**Specific Adjustments**:
- None to `internal/protocol/interfaces.go` - already designed for bidirectional
- None to `internal/protocol/server.go` - serves any incoming request
- None to `internal/protocol/client.go` - sends to any peer

### Phase 3: Agent Core - MODERATE CHANGES 🔄

**Current Config (from plan)**:
```go
type Config struct {
    Mode    string `json:"mode"`     // "pilot" or "wingmate" - REMOVE
    Name    string `json:"name"`
    Port    int    `json:"port"`
    Peer    string `json:"peer"`     // CHANGE to Peers []string
    LogFile string `json:"logFile"`
    Verbose bool   `json:"verbose"`
}
```

**Updated Config (per ADR-003)**:
```go
type Config struct {
    Name         string   `json:"name"`
    Port         int      `json:"port"`
    Peers        []string `json:"peers,omitempty"`  // Known peers to connect to
    LogFile      string   `json:"logFile"`
    Verbose      bool     `json:"verbose"`
    Capabilities []string `json:"capabilities,omitempty"` // What this agent offers
}
```

**Impact**:
- Remove `Mode` field entirely
- Change `Peer` (singular) to `Peers` (plural array)
- Add `Capabilities` field
- Update environment variables: remove `WINGMATE_MODE`, change `WINGMATE_PEER` to `WINGMATE_PEERS`
- Update validation logic

---

## Phases 4-5 Impact Analysis - MAJOR RESTRUCTURE ❌

This is the **critical impact area**.

### Current Plan Structure (OBSOLETE)

```
Phase 4: Wingmate Agent - internal/wingmate/
├── agent.go          - WingmateAgent struct
├── agent_test.go
├── handlers.go       - Handle incoming requests
└── handlers_test.go

Phase 5: Pilot Agent - internal/pilot/
├── agent.go          - PilotAgent struct
├── agent_test.go
├── commands.go       - Initiate outgoing requests
└── commands_test.go
```

### New Structure (per ADR-003)

```
Unified Phase 4: Agent Implementation - internal/agent/
├── agent.go          - Agent struct (both client + server)
├── agent_test.go
├── server.go         - Handle incoming A2A requests (was handlers.go)
├── server_test.go
├── client.go         - Make outgoing A2A requests (was commands.go)
├── client_test.go
├── conversation.go   - Conversation/mission management
├── conversation_test.go
├── config.go         - Configuration (moved from agent core)
└── config_test.go
```

**Key Changes**:
1. **DO NOT CREATE** `internal/pilot/` directory
2. **DO NOT CREATE** `internal/wingmate/` directory
3. **MERGE** both into unified `internal/agent/`
4. Single `Agent` struct that can:
   - Listen for incoming requests (server capability)
   - Make outgoing requests (client capability)
   - Track conversation roles dynamically

### Merged Agent Design

```go
// Agent is a unified A2A peer that can both initiate and respond to conversations.
// Per ADR-003: "pilot" and "wingmate" are conversation roles, not instance modes.
type Agent struct {
    config    *Config
    server    *protocol.Server  // Handle incoming
    client    *protocol.Client  // Make outgoing
    flightLog *flightlog.FlightLog
    ready     chan struct{}     // For WaitUntilReady()

    // Peer management
    knownPeers map[string]*types.AgentCard
    peersMu    sync.RWMutex
}

// Start begins listening for incoming requests and connects to known peers.
func (a *Agent) Start(ctx context.Context) error { ... }

// SendMessage sends a message to a peer (acts as "pilot" in this conversation).
func (a *Agent) SendMessage(ctx context.Context, peerURL string, msg *types.Message) (*types.A2AResponse, error) { ... }

// HandleMessage processes an incoming message (acts as "wingmate" in this conversation).
func (a *Agent) HandleMessage(ctx context.Context, msg *types.A2AMessage) (*types.A2AResponse, error) { ... }
```

---

## Phase 6 Impact Analysis - CLI CHANGES 🔄

### Current CLI (OBSOLETE)

```bash
./wingmate --mode wingmate --port 9001
./wingmate --mode pilot --port 9000 --peer http://localhost:9001
```

### Updated CLI (per ADR-003)

```bash
# Single unified mode - just specify port and optional peers
./wingmate --port 9000 --peers http://machineB:9001,http://machineC:9002

# Minimal (no peers, just accepts incoming)
./wingmate --port 9000
```

**Changes to cmd/wingmate/main.go**:
1. Remove `--mode` flag entirely
2. Change `--peer` to `--peers` (comma-separated list)
3. Remove mode dispatch switch statement
4. Single code path: start unified agent

---

## Plan Restructure Required

### Current Phases (7 phases)

| Phase | Name | Status |
|-------|------|--------|
| 1 | Foundation | ✅ Complete (minor update needed) |
| 2 | Protocol Layer | Not started |
| 3 | Agent Core | Not started |
| 4 | Wingmate Agent | ❌ **OBSOLETE** |
| 5 | Pilot Agent | ❌ **OBSOLETE** |
| 6 | CLI & Integration | Not started |
| 7 | Cross-Machine Verification | Not started |

### Proposed Phases (6 phases)

| Phase | Name | Status | Notes |
|-------|------|--------|-------|
| 1 | Foundation | ✅ Complete | Add Role to Entry struct |
| 2 | Protocol Layer | Not started | Unchanged |
| 3 | Unified Agent | Not started | **MERGED** from old 3+4+5 |
| 4 | CLI & Integration | Not started | Was Phase 6 |
| 5 | Cross-Machine Verification | Not started | Was Phase 7 |

### Recommended Approach

Given the scope of changes, I recommend:

1. **Update Phase 1**: Add `Role` field to Entry struct (minor, backwards compatible)
2. **Update plan.md**: Restructure to reflect unified agent architecture
3. **Proceed to Phase 2**: Protocol layer is largely unaffected
4. **Merge Phases 3-5**: Implement unified agent in `internal/agent/`

---

## Immediate Action Items

### Phase 1 Update (Minor)

Add to `internal/flightlog/entry.go`:

```go
// Role indicates this agent's role in the conversation.
type Role string

const (
    RolePilot    Role = "pilot"
    RoleWingmate Role = "wingmate"
)

// Add to Entry struct:
Role Role `json:"role,omitempty"`

// Update NewEntry signature to accept role:
func NewEntry(ctx context.Context, agent string, role Role, dir Direction, summary string, payload interface{}) Entry
```

### Plan Update (Major)

Update `docs/plans/001-minimal-skeleton/plan.md`:
1. Remove Phase 4 (Wingmate Agent)
2. Remove Phase 5 (Pilot Agent)
3. Merge into new Phase 3 (Unified Agent)
4. Renumber Phase 6 → Phase 4
5. Renumber Phase 7 → Phase 5
6. Update directory structure to remove `internal/pilot/` and `internal/wingmate/`
7. Update CLI flags documentation
8. Update config schema

---

## Decision Required

The plan needs to be updated to reflect ADR-003. Options:

**Option A**: Update plan.md now, before continuing implementation
- Pros: Clean forward path, no wasted work
- Cons: Delays Phase 2 start

**Option B**: Continue to Phase 2 (protocol layer), update plan during Phase 3
- Pros: Protocol is unaffected, keeps momentum
- Cons: Plan document out of sync temporarily

**Recommendation**: **Option A** - Update plan now. The Phase 4/5 restructure is significant enough that having an accurate plan prevents confusion.

---

## Conclusion

ADR-003 has **minimal impact on Phase 1 completed work**. The pkg/types code is fully compatible. The flightlog code needs only a minor additive change (Role field).

The **major impact** is on the plan structure itself - Phases 4 and 5 must be merged into a unified agent implementation. This should be done before implementing Phase 3 to avoid creating obsolete code.

**Completed Phase 1 work is safe and compatible with ADR-003.**
