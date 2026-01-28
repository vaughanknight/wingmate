# Research Report: Wingmate Peer Delegation, Capability Exposure & Agent Specialization

**Generated**: 2026-01-28
**Research Query**: "MCP tools should delegate to peers (not run locally), peers should expose capabilities/purpose, optional system prompts, container-based agent specialization"
**Mode**: Pre-Plan
**Location**: `docs/plans/005-peer-delegation/research-dossier.md`
**FlowSpace**: Available
**Findings**: 65+ across 7 subagents

---

## Executive Summary

### What It Does Today
Wingmate runs as a unified peer agent serving both A2A protocol (inter-agent JSON-RPC) and MCP tools (Claude Code integration) from a single HTTP process. MCP tools currently execute **locally** - `wingmate_chat` invokes Claude CLI on the local machine, `wingmate_discover` reads a local peer cache, and `wingmate_status` reports local health.

### Business Purpose
The user's vision requires a fundamental shift: MCP tools should **delegate work to remote peers** rather than running locally. The local agent exists so peers can ask questions back. Peers should advertise their **purpose** (e.g., "Build native iOS applications") and optionally enforce **system prompts** to specialize behavior. This enables dedicated dev containers with loose security controls while the local machine maintains strict security but gains access to specialized remote capabilities.

### Key Insights
1. **The architecture already supports delegation** - Agent owns both `protocol.Client` (for A2A outbound) and implements `PeerProvider`, but MCP handlers don't use the client for remote calls
2. **AgentCard has a `Skills` field** with name/description/schema, but no `Purpose` or `SystemPrompt` fields - these need to be added
3. **`knownPeers` map is never populated** - `GetPeers()` always returns empty because no code path writes to `knownPeers` during normal operation (only `GetPeerCard` writes to it, and it's only called by CLI commands like `wingmate status`)
4. **The PeerProvider interface pattern** (defined in `mcp/`, implemented by `agent/`) already solves circular dependency - extending it for richer peer data is straightforward

### Quick Stats
- **Components**: ~60 Go files across 6 packages
- **Key Interfaces**: 6 (MessageHandler, Server, Client, PeerProvider, LLMExecutor, ReadyNotifier)
- **Test Coverage**: ~42% file coverage, strong in MCP/protocol, gaps in validation/config
- **Prior Learnings**: 15 relevant discoveries from previous phases

---

## How It Currently Works

### Entry Points

| Entry Point | Type | Location | Purpose |
|------------|------|----------|---------|
| `POST /mcp` | HTTP | `mcp/http_transport.go:63` | MCP tool invocations from Claude Code |
| `POST /` | HTTP | `protocol/server.go:194` | A2A JSON-RPC messages from peers |
| `GET /.well-known/agent.json` | HTTP | `protocol/server.go:174` | Agent Card discovery |
| `wingmate --port N --name X` | CLI | `cmd/wingmate/main.go:175` | Start agent server |
| `wingmate ping <url>` | CLI | `cmd/wingmate/main.go:242` | Test peer connectivity |

### Core Execution Flow: MCP Tool Call (current - local only)

```
Claude Code --> POST /mcp --> LocalhostMiddleware --> HTTPHandler.ServeHTTP()
  --> handleToolsCall() --> lookup handler["wingmate_chat"]
  --> NewChatHandler.Execute() --> llmExecutor.Execute(prompt, sessionID)
  --> Claude CLI subprocess --> response --> JSON-RPC response --> Claude Code
```

**Key Code Path** (`internal/mcp/http_transport.go:95-101`):
```go
switch method {
case "initialize":
    return h.handleInitialize(ctx, msg), http.StatusOK
case "tools/list":
    return h.handleToolsList(ctx, msg), http.StatusOK
case "tools/call":
    return h.handleToolsCall(ctx, msg), http.StatusOK
}
```

### Core Execution Flow: A2A Message (peer-to-peer)

```
Peer Agent --> POST / --> A2AServer.handleJSONRPC()
  --> Agent.HandleMessage(msg)
  --> isPing? --> pongResponse()
  --> else --> handleLLMMessage() --> llmExecutor.Execute()
  --> A2AResponse --> JSON-RPC response --> Peer Agent
```

**Key Code Path** (`internal/agent/agent.go:436-487`):
```go
func (a *Agent) HandleMessage(ctx context.Context, msg *types.Message) (*types.A2AResponse, error) {
    // Log inbound
    if isPing(msg) {
        return pongResponse(), nil
    }
    if a.llmExecutor != nil {
        return a.handleLLMMessage(ctx, msg)
    }
    return nil, &protocol.ErrorObject{Code: llm.CodeLLMUnavailable, ...}
}
```

### Core Execution Flow: Peer Discovery (current - incomplete)

```
Config --peers --> stored in Config.Peers []string
  --> Agent.New() validates URLs
  --> But NEVER calls GetPeerCard() to populate knownPeers
  --> wingmate_discover always returns empty peers list
```

**Key Code Path** (`internal/agent/agent.go:491-500`):
```go
func (a *Agent) GetPeers() []string {
    a.peersMu.RLock()
    defer a.peersMu.RUnlock()
    peers := make([]string, 0, len(a.knownPeers))
    for url := range a.knownPeers {
        peers = append(peers, url)
    }
    return peers  // ALWAYS EMPTY - knownPeers never written during startup
}
```

### Data Flow (Current State)

```
+----------------+                              +------------------------+
|  Claude Code   |--POST /mcp------------------->|   Local Agent          |
|  (MCP Client)  |<--JSON-RPC response-----------|   (alpha)              |
+----------------+                              |                        |
                                                |  MCP: local only       |
                                                |  A2A: can send/recv    |
                                                |                        |
                                                |  +------------------+  |
                                                |  | knownPeers: {}   |  |  <-- EMPTY
                                                |  +------------------+  |
                                                +------------------------+
                                                         | A2A (unused by MCP)
                                                +--------v---------------+
                                                |   Remote Agent         |
                                                |   (bravo)              |
                                                |   Purpose: ???         |  <-- NOT EXPOSED
                                                |   SystemPrompt: ???    |  <-- NOT SUPPORTED
                                                +------------------------+
```

### State Management

- **MCP Sessions**: `MCPSessionManager` with 30-min TTL, `Mcp-Session-Id` header
- **LLM Sessions**: `SessionManager` keyed by FlightLog trace ID for conversation continuity
- **Peer Cache**: `knownPeers map[string]*types.AgentCard` protected by `sync.RWMutex` (but never populated)
- **Server State**: `ServerState` enum (Uninitialized -> Ready -> ShuttingDown -> Stopped)

---

## Architecture & Design

### Component Map

#### Core Components

- **Agent** (`internal/agent/agent.go`): Unified peer - owns server, client, flight log, peer cache, LLM executor
- **A2AServer** (`internal/protocol/server.go`): HTTP server routing to Agent Card, A2A JSON-RPC, and MCP
- **A2AClient** (`internal/protocol/client.go`): HTTP client for outbound A2A messages to peers
- **HTTPHandler** (`internal/mcp/http_transport.go`): MCP JSON-RPC handler with session management
- **CLIExecutor** (`internal/llm/cli.go`): Invokes Claude CLI subprocess for LLM inference
- **FlightLog** (`internal/flightlog/`): JSONL observability log with role/direction context

### Current Interface Map

| Interface | Package | Purpose | Implementor |
|-----------|---------|---------|-------------|
| `MessageHandler` | `protocol` | Handle incoming A2A messages | `Agent` |
| `Server` | `protocol` | HTTP server for A2A | `A2AServer` |
| `Client` | `protocol` | HTTP client for A2A | `A2AClient` |
| `PeerProvider` | `mcp` | Supply peer URLs to discover tool | `Agent` |
| `LLMExecutor` | `llm` | Abstract LLM provider | `CLIExecutor` |
| `ReadyNotifier` | `protocol` | Signal server ready | `A2AServer` |

### Design Patterns Identified

1. **PS-01: Dependency Injection via Interfaces** - PeerProvider, LLMExecutor, MessageHandler injected at construction
   - Example: `NewChatHandler(executor llm.LLMExecutor, ...)` in `handlers.go:34`
   - Enables: testability, swappable implementations, peer delegation
2. **PS-02: HTTP Router Multiplexing** - ServeHTTP switches on URL path
   - Example: `protocol/server.go:147-156` routes /, /.well-known/agent.json, /mcp
3. **PS-03: Config Composition** - WithEnv -> WithDefaults -> Merge chain
   - Example: `config.go:80-128` immutable config layering
4. **PS-06: JSON-RPC Factories** - `NewRequest()`, `NewSuccessResponse()` centralize marshaling
5. **PS-07: Contextual Error Wrapping** - ProtocolError, MCPError, LLMError with Unwrap()
6. **PS-08: Agent Card Declaration** - Dynamic skill building based on availability
   - Example: `agent.go:137-167` adds "chat" skill only if CLI installed
7. **PS-09: Middleware Pattern** - LocalhostMiddleware wraps MCP handler
8. **PS-10: Content-Based Routing** - isPing() checks message text for dispatch

### System Boundaries

- **Internal**: Agent process boundary (single binary)
- **External**: A2A protocol (HTTP JSON-RPC to peers), Claude CLI (subprocess), Flight Log (file I/O)
- **Security**: MCP localhost-only (403 for non-localhost), A2A open (all interfaces)

---

## Dependencies & Integration

### What MCP Tools Depend On

#### Internal Dependencies

| Dependency | Type | Purpose | Risk if Changed |
|------------|------|---------|-----------------|
| `llm.LLMExecutor` | Required | Chat tool execution | High - core functionality |
| `mcp.PeerProvider` | Required | Discover tool | Medium - interface stable |
| `mcp.MCPSessionManager` | Required | Session continuity | Medium - TTL behavior |
| `mcp.ServerConfig` | Required | Server metadata | Low - static config |

#### External Dependencies

| Service/Library | Version | Purpose | Criticality |
|-----------------|---------|---------|-------------|
| Claude CLI | latest | LLM inference | High |
| `modelcontextprotocol/go-sdk` | v1.2.0 | MCP types | Medium |
| `a2aproject/a2a-go` | - | Not yet used | Low |

### What Depends on MCP Tools

- **Claude Code** - Invokes tools via HTTP MCP protocol
- **Flight Log** - Records all tool invocations

### Key Dependency Gap: MCP -> A2A Client

MCP `NewChatHandler` signature:
```go
func NewChatHandler(executor llm.LLMExecutor, sessions SessionManager, logger Logger) ToolHandler
```

It receives `LLMExecutor` (local CLI) but **no** `protocol.Client` (for A2A outbound) and **no** `PeerProvider` (for peer resolution). The handler literally cannot send messages to remote agents.

### Agent Card vs MCP Tools (Parallel Channels)

```
Agent Card (A2A Protocol)          MCP Tools (MCP Protocol)
/.well-known/agent.json            POST /mcp tools/list
+---------------------------+      +---------------------------+
| Skills:                   |      | Tools:                    |
|   - ping                  |      |   - wingmate_chat         |
|   - chat (if CLI avail)   |      |   - wingmate_status       |
+---------------------------+      |   - wingmate_discover     |
                                   +---------------------------+
```

These are independent - Agent Card advertises A2A capabilities, MCP tools advertise MCP capabilities. They can diverge (and should for the delegation model).

---

## Quality & Testing

### Current Test Coverage

- **Unit Tests**: 25 test files (~42% of Go files), strong in mcp/ and protocol/
- **Integration Tests**: `tests/integration/ping_pong_test.go` (4 tests), `internal/agent/mcp_integration_test.go` (4 tests)
- **Race Detection**: All tests designed for `go test -race`, 7 sync.RWMutex locations
- **Gaps**: `internal/agent/validate.go` (zero coverage), `internal/agent/conversation.go` (zero coverage), `cmd/wingmate/main.go` (zero coverage)

### Test Strategy Analysis

- Mock infrastructure exists: `MockLLMExecutor`, `mockPeerProvider`, `mockHandler`
- Test docs follow TDD format: Why / Contract / Usage Notes / Quality Contribution
- Concurrent access tests: 20 goroutines x 50 ops in session manager

### Known Issues & Technical Debt

| Issue | Severity | Location | Impact |
|-------|----------|----------|---------|
| `knownPeers` never populated | High | `agent.go:103` | Discover tool useless |
| No validation tests | Medium | `validate.go` | Config errors caught at runtime |
| No conversation.go tests | Low | `conversation.go` | Flight Log helpers untested |
| Empty `internal/pilot/` and `internal/wingmate/` dirs | Low | ADR-003 artifact | Confusion |

---

## Modification Considerations

### Safe to Modify

1. **AgentCard/Skill types** (`pkg/types/agentcard.go`) - Add `Purpose`, `SystemPrompt` fields
   - Well-isolated; JSON serialization; no complex logic
2. **Config struct** (`internal/agent/config.go`) - Add `Purpose`, `SystemPrompt` fields
   - Follows established WithEnv/WithDefaults/Merge pattern
3. **MCP tool handlers** (`internal/mcp/handlers.go`) - Add new tools or extend existing ones
   - RegisterTool/RegisterHandler pattern well-tested
4. **PeerProvider interface** (`internal/mcp/handlers.go`) - Extend to return richer peer data
   - Interface change, but only one implementor (Agent)

### Modify with Caution

1. **Agent.HandleMessage** (`agent.go:436-487`) - Core message routing, well-tested but central
   - Risk: Breaking A2A compatibility
   - Mitigation: Add new message types rather than modifying existing routing
2. **HTTPHandler.ServeHTTP** (`http_transport.go:63`) - MCP request pipeline
   - Risk: Breaking Claude Code compatibility
   - Mitigation: Only add new methods, don't change existing ones

### Danger Zones

1. **JSON-RPC ID handling** - IDs arrive as `float64` from `json.Unmarshal`, must preserve as `any` (PL-03)
2. **Session manager cleanup** - Background goroutine with copy-on-write pattern (PL-05)
3. **Port zero auto-assign** - Go zero-value ambiguity, use negative sentinels (PL-08)

### Extension Points

1. **New MCP Tools**: `RegisterTool()` + `RegisterHandler()` in `agent.New()`
2. **New A2A Message Types**: Extend `HandleMessage()` switch
3. **AgentCard Skills**: Add to `buildAgentCard()` slice
4. **Config Fields**: Add to struct + WithEnv + WithDefaults + Merge + Validate

---

## Prior Learnings (From Previous Implementations)

### PL-03: JSON Number Type Assertions in JSON-RPC
**Source**: `docs/plans/003-mcp-server-integration/tasks/phase-2/execution.log.md`
**Original Type**: gotcha
**What They Found**: JSON-RPC message IDs from `json.Unmarshal` arrive as `float64` type, not `int`. Must preserve as `any` in responses.
**Why This Matters Now**: When building A2A forwarding for peer delegation, must maintain precise JSON-RPC ID matching across parse/unmarshal cycles.
**Action**: Never cast message IDs when forwarding between MCP and A2A.

### PL-04: Localhost Validation Security Pattern
**Source**: `docs/plans/004-mcp-http-transport/tasks/phase-1/execution.log.md`
**Original Type**: decision
**What They Found**: Must use `net.SplitHostPort` + `net.ParseIP` + `ip.IsLoopback()`. Intentionally ignore proxy headers to prevent spoofing.
**Why This Matters Now**: MCP remains localhost-only even when delegating to remote peers. Security boundary is at the MCP entry point, not at delegation.
**Action**: Don't relax localhost restriction for delegation - delegate via A2A (which is open), not by opening MCP.

### PL-06: PeerProvider Interface for Circular Dependency Avoidance
**Source**: `docs/plans/004-mcp-http-transport/tasks/phase-2/execution.log.md`
**Original Type**: decision
**What They Found**: Define interface in consumer package (mcp), implement in provider (agent). Agent imports MCP but MCP never imports Agent.
**Why This Matters Now**: Extending PeerProvider to return richer data (name, purpose, skills) follows the same pattern. Define extended interface in mcp/, implement in agent/.
**Action**: Extend PeerProvider rather than creating new interfaces.

### PL-07: HTTPHandler Self-Contained Message Dispatch
**Source**: `docs/plans/004-mcp-http-transport/tasks/phase-1/execution.log.md`
**Original Type**: decision
**What They Found**: Rather than coupling transports, duplicate simple dispatch logic. Self-contained handlers are easier to test.
**Why This Matters Now**: New delegation tools should be self-contained - don't couple MCP tool logic with A2A client internals.
**Action**: Delegation handler should own its own A2A call logic, not share with Agent's HandleMessage.

### PL-08: Port Zero Auto-Assign Preservation
**Source**: `docs/plans/004-mcp-http-transport/tasks/phase-2/execution.log.md`
**Original Type**: gotcha
**What They Found**: Go doesn't distinguish "zero value not set" from "explicitly zero" for ints. Fixed by checking `< 0` instead of `== 0`.
**Why This Matters Now**: Same pattern needed for any new config fields where zero/empty is a valid value (e.g., empty SystemPrompt means "no system prompt").
**Action**: Use pointer types or sentinel values for optional config fields.

### PL-09: Flight Log Entry Pattern for Tool Invocation
**Source**: `docs/plans/003-mcp-server-integration/tasks/phase-2/execution.log.md`
**Original Type**: decision
**What They Found**: Log trace_id, tool name, session_id, duration, truncated input (200 chars), truncated output (500 chars), error code.
**Why This Matters Now**: Delegated peer calls need similar logging with both the MCP invocation AND the A2A forwarding logged under the same trace ID.
**Action**: Log delegation as two entries: MCP inbound + A2A outbound, correlated by trace ID.

### Prior Learnings Summary

| ID | Type | Source Plan | Key Insight | Action |
|----|------|-------------|-------------|--------|
| PL-03 | gotcha | 003 | JSON-RPC IDs are float64 | Preserve as `any` in forwarding |
| PL-04 | decision | 004 | Localhost check ignores proxy headers | Keep MCP localhost-only |
| PL-06 | decision | 004 | Interface in consumer, impl in provider | Extend PeerProvider |
| PL-07 | decision | 004 | Self-contained handlers | Delegation handler owns A2A calls |
| PL-08 | gotcha | 004 | Zero-value ambiguity | Use pointers for optional config |
| PL-09 | decision | 003 | Flight Log truncation pattern | Dual-entry logging for delegation |

---

## Critical Discoveries

### Discovery 01: knownPeers Never Populated During Normal Operation
**Impact**: Critical
**Source**: IA-05, DC-03, DC-09
**What**: `Agent.knownPeers` map is initialized empty in `New()` (agent.go:103) and only written by `GetPeerCard()` (client.go:121-133). `GetPeerCard()` is only called by CLI commands (`wingmate status <url>`), never during agent startup or MCP tool invocation. Despite `Config.Peers` containing valid peer URLs, they are never probed.
**Why It Matters**: `wingmate_discover` always returns `{"agent_id":"alpha","peers":[]}`. Peer delegation is impossible without a populated peer registry.
**Required Action**: Add peer probing on startup - iterate `Config.Peers`, call `GetPeerCard()` for each, populate `knownPeers`. Consider background re-probing for liveness.

### Discovery 02: No Purpose/SystemPrompt in AgentCard or Config
**Impact**: Critical for vision
**Source**: IA-06, IC-05, DC-07
**What**: `AgentCard` has `Skills` (name + description) but no `Purpose` field for high-level agent description beyond the generic "Wingmate A2A agent". `Config` has `Claude.SystemPrompt` string but it's not exposed in AgentCard and not used when handling incoming A2A messages - it's only passed to the CLI executor.
**Why It Matters**: Claude Code cannot know what a peer specializes in. Without purpose/role metadata, the user must manually specify which peer to delegate to.
**Required Action**: Add `Purpose` to both Config and AgentCard. Use SystemPrompt in A2A message handling to contextualize LLM responses.

### Discovery 03: MCP Tools Cannot Delegate to Peers
**Impact**: Critical for vision
**Source**: IA-01, IA-02, DC-01, DC-10
**What**: `NewChatHandler` signature is `func NewChatHandler(executor llm.LLMExecutor, sessions SessionManager, logger Logger)`. It receives an `LLMExecutor` (local CLI) but no `protocol.Client` for A2A forwarding and no `PeerProvider` for peer resolution. The handler can only execute locally.
**Why It Matters**: This is the fundamental gap. MCP tool calls from Claude Code cannot reach remote agents.
**Required Action**: New MCP tool (e.g., `wingmate_ask`) that accepts peer target + message, delegates via A2A `SendMessage()`. Or extend existing chat handler with optional peer parameter.

### Discovery 04: Agent Card and MCP Tools Are Independent Channels
**Impact**: Medium - architectural insight
**Source**: DC-07, IC-05, IC-08
**What**: Agent Card skills (A2A) and MCP tools are separate advertisement channels. Agent Card lists "ping" and "chat" as A2A skills. MCP lists "wingmate_chat", "wingmate_status", "wingmate_discover" as MCP tools. They don't reference each other.
**Why It Matters**: For delegation, Claude Code needs to see what remote peers can do. This should come from the peer's AgentCard (fetched via A2A), not from MCP tools/list. The discover tool should synthesize peer AgentCards into a useful response.
**Required Action**: Enhanced discover tool that returns peer AgentCards (name, purpose, skills) not just URLs.

---

## Supporting Documentation

### Related ADRs
- **ADR-001** (`docs/adr/001-language-choice.md`): Go, single binary - constrains to Go implementation
- **ADR-002** (`docs/adr/002-flight-log-storage.md`): JSONL + stdout - all delegation must be logged
- **ADR-003** (`docs/adr/003-unified-peer-architecture.md`): Unified peers - every agent can initiate or respond
- **ADR-004** (`docs/adr/004-mcp-server-implementation.md`): MCP via HTTP at /mcp - amended for HTTP transport

### Key Code Comments
- `agent.go:103`: `knownPeers: make(map[string]*types.AgentCard)` - empty map, never populated on startup
- `handlers.go:36-40`: `PeerProvider interface` - designed for extension
- `config.go:43`: `SystemPrompt string` field exists in ClaudeConfig but unused in A2A handling

### A2A Protocol Research
- `docs/research/a2a-research.md`: Comprehensive A2A spec analysis (v0.3.0)
- `docs/research/a2a-technical-constraints.md`: Go-specific gotchas for A2A implementation

---

## Recommendations

### Architecture for Peer Delegation

```
+----------------+                              +------------------------+
|  Claude Code   |--wingmate_ask(peer,msg)------>|   Local Agent          |
|                |<--response from peer----------|   (your machine)       |
+----------------+                              |                        |
                                                |  MCP -> A2A bridge     |
                                                |  Localhost only         |
                                                |  Strict security       |
                                                +----------+-------------+
                                                           | A2A protocol
                                    +-----------+----------+-----------+
                                    v                      v           v
                           +----------------+  +----------------+  +----------------+
                           | iOS Builder    |  | Backend Dev    |  | Data Pipeline  |
                           | (container)    |  | (container)    |  | (container)    |
                           |                |  |                |  |                |
                           | Purpose:       |  | Purpose:       |  | Purpose:       |
                           | "Build native  |  | "Develop Go    |  | "Build data    |
                           |  iOS apps"     |  |  microservices"|  |  pipelines"    |
                           |                |  |                |  |                |
                           | SystemPrompt:  |  | SystemPrompt:  |  | SystemPrompt:  |
                           | "You are an    |  | "You are a     |  | "You are a     |
                           |  iOS expert.."|  |  Go backend.."|  |  data eng.."   |
                           |                |  |                |  |                |
                           | Loose security |  | Loose security |  | Loose security |
                           | (container)    |  | (container)    |  | (container)    |
                           +----------------+  +----------------+  +----------------+
```

### What Needs to Be Built

1. **Peer Probing on Startup** - Agent iterates `Config.Peers`, fetches AgentCards, populates `knownPeers`
2. **Extended AgentCard** - Add `Purpose` string field for high-level agent description
3. **Config.Purpose** - Maps to AgentCard.Purpose, exposed via `--purpose` flag
4. **SystemPrompt in A2A** - Pass to Claude CLI when handling incoming A2A messages
5. **New MCP Tool: `wingmate_ask`** - Accepts `peer` (name or URL) + `message`, delegates via A2A `SendMessage()`
6. **Enhanced `wingmate_discover`** - Return peer names, purposes, and skills (not just URLs)
7. **PeerProvider extension** - Return `[]PeerInfo{Name, URL, Purpose, Skills}` instead of `[]string`
8. **Background peer health checks** - Periodic re-probe to detect peer availability changes

### If Modifying This System
1. Follow PeerProvider interface pattern (define in mcp, implement in agent) per PL-06
2. Use existing `protocol.Client.SendMessage()` for A2A delegation
3. Add new MCP tools via `RegisterTool`/`RegisterHandler` pattern
4. Log all delegated calls to Flight Log with trace ID correlation per PL-09
5. Keep MCP localhost-only even when delegating per PL-04

### If Extending This System
1. New tools follow `NewXxxHandler(deps...) ToolHandler` pattern
2. New config fields follow WithEnv/WithDefaults/Merge chain
3. New AgentCard fields use `omitempty` JSON tags
4. Use pointer types for optional fields per PL-08

---

## External Research Opportunities

### Research Opportunity 1: A2A Agent Card Extensions for Purpose/Role

**Why Needed**: The A2A spec defines AgentCard with Skills but it's unclear if adding custom fields (Purpose, SystemPrompt) is spec-compliant or if there's a standard way to advertise agent specialization.
**Impact on Plan**: Determines whether to use standard A2A fields or custom extensions.
**Source Findings**: IC-05, DE-08

**Ready-to-use prompt:**
```
/deepresearch "A2A protocol Agent Card extensibility: Does the A2A specification
(v0.3.0 or later) support custom fields in Agent Cards? Is there a standard way
to advertise agent purpose/role beyond Skills? How do other A2A implementations
handle agent specialization and capability advertisement? Context: Go codebase
using AgentCard struct with Name, Description, URL, Version, Capabilities, Skills.
Want to add Purpose (string) and SystemPrompt (string) fields."
```

**Results location**: Save results to `docs/plans/005-peer-delegation/external-research/a2a-agentcard-extensions.md`

### Research Opportunity 2: Container Security Models for Delegated Agent Execution

**Why Needed**: The vision involves containers with "looser" security controls running specialized agents. Need to understand best practices for sandboxed Claude Code execution in containers, and how to configure different security policies per container.
**Impact on Plan**: Affects how system prompts and security boundaries are configured per container.
**Source Findings**: PS-09, DE-05 (P6 Security by Design)

**Ready-to-use prompt:**
```
/deepresearch "Best practices for running AI coding agents (like Claude Code) in
Docker/Podman containers with relaxed security controls vs host machine with strict
controls. How to configure Claude Code permissions per environment. Container
networking for agent-to-agent communication (A2A protocol over HTTP). Security
model: containers can execute code freely, host machine restricts execution but
can delegate to containers."
```

**Results location**: Save results to `docs/plans/005-peer-delegation/external-research/container-security-models.md`

---

**After External Research:**
- To conduct external research: Run the `/deepresearch` commands above
- To skip and proceed: Run `/plan-1b-specify "peer-delegation"` (unresolved opportunities will be noted as a soft warning)

---

## Appendix: File Inventory

### Core Files

| File | Purpose | Lines |
|------|---------|-------|
| `internal/agent/agent.go` | Unified agent struct, lifecycle, message handling | ~500 |
| `internal/agent/config.go` | Config struct, WithEnv, WithDefaults, Merge | ~230 |
| `internal/agent/client.go` | Outbound A2A methods (Ping, SendMessage, GetPeerCard) | ~150 |
| `internal/mcp/http_transport.go` | MCP HTTP handler, JSON-RPC dispatch | ~290 |
| `internal/mcp/handlers.go` | MCP tool handlers (chat, status, discover) | ~190 |
| `internal/mcp/tools.go` | MCP tool definitions and schemas | ~75 |
| `internal/mcp/session.go` | MCP session manager with TTL | ~185 |
| `internal/mcp/localhost.go` | Localhost-only middleware | ~47 |
| `internal/protocol/server.go` | A2A HTTP server, routing | ~310 |
| `internal/protocol/client.go` | A2A HTTP client | ~160 |
| `internal/protocol/jsonrpc.go` | JSON-RPC 2.0 types and factories | ~115 |
| `internal/llm/cli.go` | Claude CLI executor | ~170 |
| `internal/llm/client.go` | LLMExecutor interface | ~28 |
| `pkg/types/agentcard.go` | AgentCard, Skill, Capabilities types | ~68 |
| `pkg/types/message.go` | Message, Part types | ~35 |

### Test Files

| File | Tests | Coverage Area |
|------|-------|---------------|
| `internal/mcp/http_transport_test.go` | 13 | MCP HTTP handler |
| `internal/mcp/session_test.go` | 13 | Session management |
| `internal/mcp/localhost_test.go` | 20 | Localhost validation |
| `internal/mcp/handlers_test.go` | ~15 | Tool handlers |
| `internal/protocol/server_test.go` | 8 | A2A server |
| `internal/protocol/client_test.go` | 8 | A2A client |
| `internal/agent/agent_test.go` | ~30 | Agent lifecycle |
| `internal/agent/mcp_integration_test.go` | 4 | MCP integration |
| `tests/integration/ping_pong_test.go` | 4 | A2A integration |

---

## Next Steps

1. **Optional**: Run `/deepresearch` prompts above for A2A extensions and container security
2. **Proceed**: Run `/plan-1b-specify "peer-delegation"` to create the feature specification
3. The specification should address all 4 critical discoveries and leverage the 6 prior learnings

---

**Research Complete**: 2026-01-28
**Report Location**: `docs/plans/005-peer-delegation/research-dossier.md`
