# Research Dossier: MCP Server Integration

**Feature**: Add MCP Server to Wingmate CLI
**Date**: 2026-01-22
**Status**: Research Complete

---

## Executive Summary

This research explores adding Model Context Protocol (MCP) server capability to Wingmate, enabling Claude Code and other MCP-compatible clients to invoke Wingmate's LLM capabilities without understanding the underlying A2A protocol, ports, or configurations.

**Key Findings**:
1. MCP is architecturally sanctioned (Constitution P4: "MCP may be used for local tool integration")
2. Architecture docs already show MCP as a planned component
3. Zero external dependencies currently - clean integration opportunity
4. LLMExecutor interface is the primary capability to expose as an MCP tool
5. Existing patterns (server lifecycle, error codes, logging) provide clear templates
6. Go MCP libraries exist (e.g., `mark3labs/mcp-go`) but may require evaluation

**Recommended Approach**: Add `wingmate mcp` subcommand that runs an MCP server over stdio, exposing `wingmate_chat` as the primary tool with optional discovery/status tools.

---

## How It Currently Works

### Architecture Overview

Wingmate implements the A2A (Agent-to-Agent) protocol for inter-agent communication:

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Current Architecture                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   Claude Code                    Wingmate Agent                      │
│   ┌─────────┐                   ┌──────────────────────────────┐    │
│   │         │   Manual HTTP     │  cmd/wingmate/main.go        │    │
│   │ Cannot  │ ──────────────────│    └── runServer()           │    │
│   │ invoke  │   (user must      │                              │    │
│   │ easily  │   configure)      │  internal/agent/             │    │
│   │         │                   │    └── agent.go              │    │
│   └─────────┘                   │        ├── llmExecutor       │    │
│                                 │        ├── sessionManager    │    │
│                                 │        └── flightLog         │    │
│                                 │                              │    │
│                                 │  internal/protocol/          │    │
│                                 │    └── A2AServer (HTTP)      │    │
│                                 │                              │    │
│                                 │  internal/llm/               │    │
│                                 │    ├── LLMExecutor interface │    │
│                                 │    └── CLIExecutor impl      │    │
│                                 └──────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### CLI Structure (cmd/wingmate/main.go)

The CLI uses a simple switch-based subcommand pattern:

```go
switch command {
case "":
    return runServer(cfg)      // Default: run A2A server
case "ping":
    return runPing(cfg, args)  // Ping another agent
case "status":
    return runStatus(cfg, args) // Check agent status
default:
    fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
}
```

**Extension Point**: Adding `case "mcp":` would follow established pattern.

### Core Interfaces

**LLMExecutor** (`internal/llm/client.go`) - Primary tool to expose:
```go
type LLMExecutor interface {
    Execute(ctx context.Context, prompt string, sessionID string) (*CLIResponse, error)
    IsInstalled() bool
}
```

**CLIResponse** - Response structure:
```go
type CLIResponse struct {
    Result    string `json:"result"`
    SessionID string `json:"session_id"`
    Model     string `json:"model"`
    Duration  int64  `json:"duration_ms"`
    CostUSD   string `json:"cost_usd,omitempty"`
}
```

### Error Code Ranges

| Range | Domain | Examples |
|-------|--------|----------|
| -32xxx | JSON-RPC reserved | -32600 Invalid Request |
| 1xxx | Application | 1001 Invalid configuration |
| 2xxx | LLM | 2001 CLI not available, 2002 Execution failed |
| **3xxx** | **MCP (proposed)** | 3001 Tool not found, 3002 Invalid arguments |

### Observability (Flight Log)

Constitution P1 requires all operations logged. Current pattern:

```go
logger := ctx.Value(flightlog.LoggerKey).(flightlog.Logger)
logger.LogToolInvocation(ctx, flightlog.ToolEntry{
    Tool:     "llm_execute",
    Input:    prompt,
    Output:   response.Result,
    Duration: time.Since(start),
})
```

---

## Proposed MCP Architecture

### Target Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Proposed Architecture                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   Claude Code                    Wingmate Agent                      │
│   ┌─────────┐                   ┌──────────────────────────────┐    │
│   │         │   MCP over        │  cmd/wingmate/main.go        │    │
│   │ Claude  │   stdio           │    ├── runServer()  (A2A)    │    │
│   │  Code   │ ◄─────────────────│    └── runMCP()     (NEW)    │    │
│   │         │                   │                              │    │
│   └─────────┘                   │  internal/mcp/      (NEW)    │    │
│       │                         │    ├── server.go             │    │
│       │ MCP Tools:              │    ├── tools.go              │    │
│       │  • wingmate_chat        │    └── types.go              │    │
│       │  • wingmate_status      │                              │    │
│       │  • wingmate_discover    │  internal/agent/             │    │
│       ▼                         │    └── (unchanged)           │    │
│   ┌─────────┐                   │                              │    │
│   │ Other   │                   │  internal/llm/               │    │
│   │  MCP    │                   │    └── LLMExecutor           │    │
│   │ Clients │                   │        (exposed via MCP)     │    │
│   └─────────┘                   └──────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### MCP Tools to Expose

| Tool | Description | Priority |
|------|-------------|----------|
| `wingmate_chat` | Send prompt to LLM via Claude CLI | P0 (Core) |
| `wingmate_status` | Check agent health and CLI availability | P1 |
| `wingmate_discover` | List available peer agents | P2 |
| `wingmate_send` | Send A2A message to peer | P2 |

### Primary Tool: wingmate_chat

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "prompt": {
      "type": "string",
      "description": "The prompt to send to Claude CLI"
    },
    "session_id": {
      "type": "string",
      "description": "Optional session ID for conversation continuity"
    }
  },
  "required": ["prompt"]
}
```

**Output**:
```json
{
  "result": "Claude's response text",
  "session_id": "sess-abc123",
  "model": "claude-sonnet-4-20250514",
  "duration_ms": 1234
}
```

---

## Integration Points

### 1. CLI Entry Point

**File**: `cmd/wingmate/main.go`
**Change**: Add `case "mcp":` to command switch

```go
case "mcp":
    return runMCP(cfg)
```

### 2. New Package: internal/mcp/

**Files to create**:
- `server.go` - MCP server lifecycle (stdio transport)
- `tools.go` - Tool definitions and handlers
- `types.go` - MCP-specific types
- `errors.go` - Error codes 3001-3010

### 3. Dependency Injection

The MCP server needs access to:
- `LLMExecutor` - for `wingmate_chat` tool
- `SessionManager` - for conversation continuity
- `FlightLog Logger` - for observability (P1)
- `Config` - for model/timeout settings

### 4. Configuration

Extend existing config pattern:

```go
type MCPConfig struct {
    Enabled  bool   `json:"enabled"`
    LogLevel string `json:"log_level"`
}
```

### 5. go.mod Dependencies

Options for MCP implementation:
1. **Use existing SDK**: `mark3labs/mcp-go` or similar
2. **Implement from spec**: Zero-dependency approach (matches current style)

**Recommendation**: Evaluate `mark3labs/mcp-go` first; implement custom only if SDK has issues.

---

## Prior Learnings Applied

### From Phase 1-4 Implementation

| ID | Learning | Application to MCP |
|----|----------|-------------------|
| PL-01 | JSON-RPC -32xxx reserved | Use 3xxx range for MCP errors |
| PL-02 | Agent Card at /.well-known/ | MCP uses different discovery |
| PL-04 | time.Time omitempty never omits | Use pointer or custom type |
| PL-06 | HTTP WriteTimeout no context cancel | Use context.WithTimeout |
| PL-07 | Env vars fail silently | Log configuration source |
| PL-08 | CLI exit code handling | Check exec.ExitError |
| PL-10 | SDK incompatibility → custom types | May need custom MCP types |
| PL-11 | Graceful shutdown needs WaitGroup | Track in-flight MCP requests |

### Testing Patterns to Reuse

1. **Mock LLMExecutor** (`internal/llm/mock.go`)
2. **TestHelperProcess** for CLI subprocess mocking
3. **httptest** for protocol testing (adapt for stdio)
4. **Interface satisfaction tests** at compile time

---

## Key Constraints

### From Constitution

| Principle | Constraint | Impact |
|-----------|------------|--------|
| P1 Observability | All MCP tool invocations must be logged | Add Flight Log entries |
| P4 Protocol | MCP sanctioned for local tools | Green light for implementation |
| P6 Security | Audit logging required | Log all requests/responses |

### From ADRs

| ADR | Constraint |
|-----|------------|
| ADR-001 | Must be Go, single binary |
| ADR-002 | Flight Log at ./flight.jsonl |
| ADR-003 | Unified peer architecture (no mode separation) |

### From Architecture

- MCP already shown in architecture diagrams as planned component
- `internal/mcp/` is an appropriate module location
- Dependency rules: mcp/ MAY import llm/, flightlog/, pkg/types/

---

## External Research Opportunities

The following topics require external research before specification:

### 1. MCP Protocol Specification

**Goal**: Understand MCP wire protocol, tool registration, and stdio transport.

```
/deepresearch

Research the Model Context Protocol (MCP) specification for implementing an MCP server in Go:

1. What is the exact JSON-RPC message format for MCP tool registration?
2. How does stdio transport work for MCP servers?
3. What are the required MCP server capabilities to declare?
4. How do MCP clients discover available tools?
5. What is the lifecycle of an MCP server (initialize, list tools, invoke, shutdown)?

Focus on server implementation details, not client usage.
```

### 2. Go MCP Libraries

**Goal**: Evaluate existing Go libraries for MCP server implementation.

```
/deepresearch

Research Go libraries for implementing MCP (Model Context Protocol) servers:

1. What Go MCP libraries exist? (e.g., mark3labs/mcp-go, others)
2. How mature are they? (stars, recent commits, documentation quality)
3. What is the API for defining tools and handling requests?
4. Do they support stdio transport?
5. Are there any known issues or limitations?
6. What is the minimum Go version required?

Provide code examples for defining a simple tool with input validation.
```

### 3. Claude Code MCP Configuration

**Goal**: Understand how to configure Claude Code to use a local MCP server.

```
/deepresearch

Research how to configure Claude Code (Anthropic's CLI tool) to use a local MCP server:

1. What is the configuration file format for adding MCP servers?
2. How do you specify a stdio-based MCP server command?
3. What environment variables or settings affect MCP server discovery?
4. How does Claude Code handle MCP server errors or timeouts?
5. Are there any authentication requirements for local MCP servers?

Provide a complete example configuration for adding a local Go binary as an MCP server.
```

---

## Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| MCP SDK immaturity | Medium | Evaluate SDK; fallback to custom impl |
| Protocol versioning | Low | Target current MCP spec version |
| Stdio blocking issues | Medium | Use goroutines, context cancellation |
| Session state in stdio | Medium | SessionManager handles; document limitation |
| Testing stdio transport | Medium | Adapt httptest patterns; use io.Pipe |

---

## Recommended Next Steps

1. **Run /deepresearch** for MCP protocol specification
2. **Run /deepresearch** for Go MCP libraries evaluation
3. **Create ADR-004** for MCP integration decision
4. **Run /plan-1b-specify** to create feature specification
5. **Run /plan-3-architect** to create implementation plan

---

## Appendix: Research Findings by Subagent

### IA: Implementation Archaeologist (10 findings)

| ID | Finding | Relevance |
|----|---------|-----------|
| IA-01 | CLI uses switch-based subcommand pattern | Add `case "mcp":` |
| IA-02 | A2AServer is HTTP-based with handler pattern | MCP uses stdio instead |
| IA-03 | MCP mentioned in architecture docs | Already planned |
| IA-04 | Constitution P4 endorses MCP | Green light |
| IA-05 | No existing MCP code | Clean slate |
| IA-06 | Go language constraint | Affects SDK selection |
| IA-07 | Single binary requirement | MCP must compile in |
| IA-08 | Subcommand functions return error | Follow pattern |
| IA-09 | Config loaded before command dispatch | MCP config available |
| IA-10 | Verbose flag for stdout logging | Apply to MCP |

### DC: Dependency Cartographer (10 findings)

| ID | Finding | Relevance |
|----|---------|-----------|
| DC-01 | Zero external dependencies | Clean integration |
| DC-02 | HTTP server in protocol.A2AServer | Template for MCP server |
| DC-03 | LLMExecutor is primary tool | Expose via MCP |
| DC-04 | SessionManager for stateful | MCP session support |
| DC-05 | FlightLog required | P1 observability |
| DC-06 | Config extension pattern | Add MCPConfig |
| DC-07 | Agent struct central | Integration point |
| DC-08 | pkg/types for public types | MCP types here? |
| DC-09 | context.Context throughout | Pass to MCP handlers |
| DC-10 | sync primitives for safety | Apply to MCP |

### PS: Pattern Scout (10 findings)

| ID | Finding | Relevance |
|----|---------|-----------|
| PS-01 | Interface-first server design | Define MCPServer interface |
| PS-02 | Error code ranges by domain | Use 3xxx for MCP |
| PS-03 | Typed error wrappers | Create MCPError type |
| PS-04 | Config Clone + Merge | Apply to MCPConfig |
| PS-05 | Flight Log with trace IDs | Include in MCP logging |
| PS-06 | Server ready signaling | Channel for MCP ready |
| PS-07 | Graceful shutdown pattern | WaitGroup for MCP |
| PS-08 | Compile-time interface check | Verify MCPServer impl |
| PS-09 | Options pattern for config | MCPOption functions |
| PS-10 | Handler registration | Tool registration pattern |

### QT: Quality/Testing (10 findings)

| ID | Finding | Relevance |
|----|---------|-----------|
| QT-01 | Mock LLMExecutor pattern | Reuse for MCP tests |
| QT-02 | TestHelperProcess for CLI | Adapt for stdio |
| QT-03 | httptest for protocol | Use io.Pipe for stdio |
| QT-04 | Interface satisfaction tests | Add for MCPServer |
| QT-05 | TAD testing approach | Apply to MCP tests |
| QT-06 | Thread safety testing | Test concurrent tools |
| QT-07 | Flight Log verification | Assert MCP entries |
| QT-08 | Table-driven tests | Use for tool dispatch |
| QT-09 | Context cancellation tests | Test tool timeouts |
| QT-10 | Error unwrapping tests | Test MCPError chain |

### IC: Interface/Contract (10 findings)

| ID | Finding | Relevance |
|----|---------|-----------|
| IC-01 | LLMExecutor contract | Primary MCP tool |
| IC-02 | CLIResponse type | Tool response format |
| IC-03 | MessageHandler interface | Adapt for MCP tools |
| IC-04 | AgentCard structure | MCP has own discovery |
| IC-05 | Message/Part types | Map to MCP format |
| IC-06 | Error code contract | Extend for MCP |
| IC-07 | SessionManager contract | Session continuity |
| IC-08 | FlightLog Logger | Tool invocation logging |
| IC-09 | Config interface | MCPConfig addition |
| IC-10 | Server lifecycle | Start/Stop/Shutdown |

### DE: Documentation Examiner (10 findings)

| ID | Finding | Relevance |
|----|---------|-----------|
| DE-01 | MCP for local tools (P4) | Sanctioned protocol |
| DE-02 | Architecture shows MCP | Already planned |
| DE-03 | MCP complements A2A | Different layers |
| DE-04 | MCP out of MVP scope | Now adding it |
| DE-05 | Flight Log for MCP | Required logging |
| DE-06 | Security requirements | Apply to MCP |
| DE-07 | ADR required | Create ADR-004 |
| DE-08 | CLAUDE.md update | Document MCP |
| DE-09 | README update | Add MCP section |
| DE-10 | How-to guide needed | MCP setup guide |

### PL: Prior Learnings (15 findings)

| ID | Finding | Relevance |
|----|---------|-----------|
| PL-01 | JSON-RPC -32xxx reserved | Use 3xxx range |
| PL-02 | Agent Card path | MCP different discovery |
| PL-03 | SSE streaming flush | stdio is different |
| PL-04 | time.Time omitempty | Use pointers |
| PL-05 | JSON null vs empty | Handle in MCP |
| PL-06 | WriteTimeout no cancel | Use context |
| PL-07 | Env vars silent | Log config source |
| PL-08 | CLI exit codes | Handle in tool |
| PL-09 | Mock patterns | Reuse for MCP |
| PL-10 | Custom types needed | May need for MCP |
| PL-11 | Graceful shutdown | Track MCP requests |
| PL-12 | Config validation | Validate MCPConfig |
| PL-13 | Test documentation | Document MCP tests |
| PL-14 | Context trace IDs | Pass to MCP |
| PL-15 | Flight Log format | Add MCP entries |

---

## Document Metadata

| Field | Value |
|-------|-------|
| Created | 2026-01-22 |
| Author | Claude (plan-1a-explore) |
| Plan | 003-mcp-server-integration |
| Status | Research Complete |
| Next | /deepresearch or /plan-1b-specify |
