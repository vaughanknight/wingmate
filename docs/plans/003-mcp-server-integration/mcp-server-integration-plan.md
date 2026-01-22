# MCP Server Integration - Implementation Plan

**Plan ID**: 003
**Feature**: MCP Server Integration
**Mode**: Full
**Complexity**: CS-3 (Medium)
**Status**: Draft
**Created**: 2026-01-22

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Technical Context](#2-technical-context)
3. [Critical Findings](#3-critical-findings)
4. [Testing Philosophy](#4-testing-philosophy)
5. [Project Structure](#5-project-structure)
6. [Implementation Phases](#6-implementation-phases)
7. [Cross-Cutting Concerns](#7-cross-cutting-concerns)
8. [Complexity Tracking](#8-complexity-tracking)
9. [Progress Tracking](#9-progress-tracking)
10. [Change Footnotes Ledger](#10-change-footnotes-ledger)

---

## 1. Executive Summary

### What We're Building

An MCP (Model Context Protocol) server integrated into the Wingmate CLI that enables Claude Code and other MCP-compatible clients to invoke Wingmate's LLM capabilities through a standardized interface.

**User Command**: `wingmate mcp`

**Primary Tool**: `wingmate_chat` - Send prompts to Claude CLI with session continuity

**Additional Tools** (P1/P2):
- `wingmate_status` - Health monitoring
- `wingmate_discover` - List peer agents
- `wingmate_send` - A2A message dispatch

### Why We're Building It

Currently, using Wingmate from Claude Code requires understanding A2A protocol, HTTP ports, and manual configuration. MCP provides a standardized way for LLM tools to communicate, allowing users to simply add Wingmate as an MCP server in their `~/.claude.json` and immediately access its capabilities.

### Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| SDK | Official `modelcontextprotocol/go-sdk` v1.1.0 | Zero dependencies, stable, Google-backed |
| Transport | stdio (NDJSON) | Standard for local MCP servers |
| Error Range | 3001-3099 | Separates from app (1xxx) and LLM (2xxx) |
| Testing | Hybrid TDD | TDD for protocol, lightweight for CLI |
| Config | Env vars + file with precedence | Reuses existing WINGMATE_LLM_* pattern |

### Success Criteria

After implementation, users can:
1. Run `wingmate mcp` to start the MCP server
2. Add Wingmate to Claude Code via `~/.claude.json`
3. Invoke `wingmate_chat` tool directly from Claude Code
4. Maintain conversation continuity via session IDs

---

## 2. Technical Context

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    Wingmate + MCP Architecture                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Claude Code                     Wingmate Binary                │
│  ┌─────────┐                    ┌─────────────────┐            │
│  │ MCP     │  stdio (JSON-RPC)  │ cmd/wingmate    │            │
│  │ Client  │◄──────────────────►│ ├─ main.go      │            │
│  └─────────┘                    │ │  case "mcp":  │            │
│                                 │ │  runMCP()     │            │
│                                 │ └───────┬───────┘            │
│                                 │         │                    │
│                                 │         ▼                    │
│                                 │ ┌───────────────┐            │
│                                 │ │ internal/mcp/ │            │
│                                 │ │ ├─ server.go  │            │
│                                 │ │ ├─ tools.go   │            │
│                                 │ │ ├─ handlers.go│            │
│                                 │ │ └─ transport.go            │
│                                 │ └───────┬───────┘            │
│                                 │         │                    │
│                                 │         ▼                    │
│                                 │ ┌───────────────┐            │
│                                 │ │ internal/llm/ │            │
│                                 │ │ ├─ client.go  │◄──┐        │
│                                 │ │ ├─ session.go │   │        │
│                                 │ │ └─ cli.go     │   │        │
│                                 │ └───────────────┘   │        │
│                                 │                     │        │
│                                 │ ┌───────────────────┘        │
│                                 │ │ Claude CLI (external)      │
│                                 │ │ `claude` binary            │
│                                 │ └─────────────────────────── │
│                                 │                              │
│                                 │ ┌───────────────┐            │
│                                 │ │internal/flightlog          │
│                                 │ │ (Observability - P1)       │
│                                 │ └───────────────┘            │
│                                 └─────────────────┘            │
└─────────────────────────────────────────────────────────────────┘
```

### Integration Points

| Component | Role | Changes Required |
|-----------|------|------------------|
| `cmd/wingmate/main.go` | CLI entry point | Add `case "mcp":` handler |
| `internal/llm/client.go` | LLMExecutor interface | No changes (reuse) |
| `internal/llm/session.go` | SessionManager | No changes (reuse) |
| `internal/flightlog/` | Observability | No changes (reuse) |
| `internal/mcp/` (NEW) | MCP server package | Create new package |

### Dependencies

**New External Dependency**:
```
require github.com/modelcontextprotocol/go-sdk v1.1.0
```

**Note**: The official SDK has zero external dependencies in its core package, maintaining Wingmate's minimal-dependency philosophy (ADR-001).

### Protocol Overview

MCP uses JSON-RPC 2.0 over stdio with newline-delimited messages:

```
Client → Server: {"jsonrpc":"2.0","id":1,"method":"initialize","params":{...}}\n
Server → Client: {"jsonrpc":"2.0","id":1,"result":{...}}\n
Client → Server: {"jsonrpc":"2.0","method":"notifications/initialized"}\n
Client → Server: {"jsonrpc":"2.0","id":2,"method":"tools/list"}\n
Server → Client: {"jsonrpc":"2.0","id":2,"result":{"tools":[...]}}\n
Client → Server: {"jsonrpc":"2.0","id":3,"method":"tools/call","params":{...}}\n
Server → Client: {"jsonrpc":"2.0","id":3,"result":{"content":[...]}}\n
```

---

## 3. Critical Findings

### From Implementation Strategy Research

1. **Phase Boundaries**: Natural splits at Protocol (P1) → Tools (P2) → CLI (P3) → Docs (P4)
2. **ADR Candidates**: ADR-004 (MCP SDK Decision), ADR-005 (Error Code Allocation) - optional
3. **io.Pipe Pattern**: Use `io.Pipe` for stdio testing, not subprocess spawning
4. **Adapter Pattern**: Wrap SDK to isolate from API changes

### From Risk Analysis

| Priority | Risk | Mitigation |
|----------|------|------------|
| P0 | stdio blocking/buffering | Use goroutines + context cancellation |
| P0 | Flight Log observability | Define MCPToolEntry log type |
| P0 | Binary path resolution | Document absolute paths in setup |
| P0 | Claude CLI in CI | Mock LLMExecutor for unit tests |
| P1 | SDK API changes | Version pin + adapter pattern |
| P1 | Session thread safety | Verify mutex locks + race detector |

### Key Constraints

1. **Constitution P1**: All MCP tool invocations MUST log to Flight Log
2. **Constitution P6**: Audit logging with truncated sensitive data
3. **ADR-001**: Single binary, minimal dependencies
4. **ADR-002**: Flight Log at `./flight.jsonl`
5. **ADR-003**: Unified peer architecture (MCP is independent of A2A)

---

## 4. Testing Philosophy

### Approach: Hybrid TDD

| Component | Approach | Rationale |
|-----------|----------|-----------|
| MCP protocol handshake | TDD | Protocol compliance is brittle |
| Tool invocation | TDD | Core functionality, many edge cases |
| Error handling | TDD | Critical paths need validation |
| Graceful shutdown | TDD | Complex lifecycle management |
| CLI subcommand | Lightweight | Simple command parsing |
| Configuration | Lightweight | Straightforward validation |

### Test Pyramid

```
                /\
               /  \         E2E (5-10%)
              /    \        - Manual Claude Code testing
             /------\
            /        \      Integration (15-20%)
           /          \     - MCP + mock LLMExecutor
          /------------\
         /              \   Unit (70-75%)
        /                \ - Protocol, tools, handlers
       /------------------\
```

### Mock Boundaries

**Mock (External)**:
- Claude CLI (`os/exec`)
- stdio transport (use `io.Pipe`)

**Real (Internal)**:
- LLMExecutor interface
- SessionManager
- FlightLog

### Test Utilities to Create

```go
// tests/helpers/mcp.go
type TestTransport struct {
    Reader *io.PipeReader
    Writer *io.PipeWriter
}

func NewTestTransport() *TestTransport

func SendRequest(t *TestTransport, method string, params interface{}) error

func ReadResponse(t *TestTransport) (*Response, error)
```

---

## 5. Project Structure

### New Files

```
wingmate/
├── internal/mcp/                    # NEW PACKAGE
│   ├── server.go                    # MCPServer struct, lifecycle
│   ├── server_test.go               # TDD: Protocol compliance
│   ├── transport.go                 # stdio transport, NDJSON framing
│   ├── transport_test.go            # TDD: Message framing
│   ├── tools.go                     # Tool definitions, registration
│   ├── tools_test.go                # TDD: Tool schemas
│   ├── handlers.go                  # Tool handlers (chat, status, discover)
│   ├── handlers_test.go             # TDD: Tool invocation
│   ├── types.go                     # MCPRequest, MCPResponse, ToolInput/Output
│   ├── errors.go                    # Error codes 3001-3099
│   └── errors_test.go               # Error mapping tests
│
├── tests/helpers/                   # Test utilities
│   └── mcp.go                       # TestTransport, SendRequest, etc.
│
├── tests/integration/               # Integration tests
│   └── mcp_test.go                  # E2E MCP server tests with io.Pipe
│
├── docs/adr/                        # ADRs
│   └── 004-mcp-server-implementation.md  # NEW
│
└── docs/how/                        # User documentation
    └── mcp-setup.md                 # NEW
```

### Modified Files

```
cmd/wingmate/main.go                 # Add `case "mcp":` handler
go.mod                               # Add SDK dependency
go.sum                               # Generated
README.md                            # Add MCP quick-start
CLAUDE.md                            # Add MCP context
```

### Import Graph

```
cmd/wingmate/main.go
└── runMCP() creates:
    ├── llm.CLIExecutor (existing)
    ├── llm.SessionManager (existing)
    ├── flightlog.Logger (existing)
    └── mcp.Server (NEW)
        ├── internal/mcp/server.go
        ├── internal/mcp/tools.go
        ├── internal/mcp/handlers.go
        └── github.com/modelcontextprotocol/go-sdk/mcp
```

---

## 6. Implementation Phases

### Phase 1: Foundation & Protocol Layer

**Complexity**: CS-2
**Goal**: Working MCP server that handles initialize/tools/list handshake

#### Prerequisites
- [ ] Create ADR-004: MCP Server Implementation Decision
- [ ] Add SDK to go.mod: `go get github.com/modelcontextprotocol/go-sdk@v1.1.0`

#### Tasks

| ID | Task | TDD | File | Acceptance |
|----|------|-----|------|------------|
| 1.1 | Define MCP types | No | `internal/mcp/types.go` | Compiles, matches spec |
| 1.2 | Define error codes | No | `internal/mcp/errors.go` | 3001-3099 range |
| 1.3 | Implement transport | **Yes** | `internal/mcp/transport.go` | NDJSON read/write works |
| 1.4 | Implement server lifecycle | **Yes** | `internal/mcp/server.go` | Start/Stop/Run methods |
| 1.5 | Handle initialize request | **Yes** | `internal/mcp/server.go` | Returns capabilities |
| 1.6 | Handle tools/list request | **Yes** | `internal/mcp/server.go` | Returns empty tools list |
| 1.7 | Add Flight Log integration | **Yes** | `internal/mcp/server.go` | Startup logged |

#### Test Cases (TDD)

```go
// transport_test.go
func TestTransportReadNDJSON(t *testing.T)
func TestTransportWriteNDJSON(t *testing.T)
func TestTransportHandlesMultipleMessages(t *testing.T)
func TestTransportRejectsInvalidJSON(t *testing.T)

// server_test.go
func TestServerInitializeHandshake(t *testing.T)
func TestServerRejectsPreInitializeRequests(t *testing.T)
func TestServerToolsListEmpty(t *testing.T)
func TestServerLogsStartup(t *testing.T)
func TestServerGracefulShutdownOnEOF(t *testing.T)
```

#### Test Commands (Phase 1)

```bash
# Run all Phase 1 tests with race detection
go test ./internal/mcp/... -v -race -count=3

# Run with coverage
go test ./internal/mcp/... -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out | grep -E "(transport|server)\.go"

# Verify minimum 80% coverage on new files
go tool cover -func=coverage.out | grep "total:" | awk '{print $3}'
```

#### Acceptance Criteria (Phase 1)

- [ ] `wingmate mcp` runs (placeholder, exits immediately OK for now)
- [ ] MCP initialize request returns valid capabilities (includes `protocol_version`, `capabilities.tools`)
- [ ] tools/list returns empty tools array `{"tools": []}`
- [ ] Flight Log contains startup entry with `event: "mcp_server_started"`
- [ ] All TDD tests pass with `go test -race -count=3`
- [ ] Code coverage ≥80% on transport.go and server.go

---

### Phase 2: Tool Implementation

**Complexity**: CS-2
**Goal**: `wingmate_chat` tool works with LLMExecutor integration

#### Prerequisites
- Phase 1 complete
- [ ] Create test helpers in `tests/helpers/mcp.go`

#### Tasks

| ID | Task | TDD | File | Acceptance |
|----|------|-----|------|------------|
| 2.1 | Define tool schemas | No | `internal/mcp/tools.go` | JSON Schema valid |
| 2.2 | Register wingmate_chat | **Yes** | `internal/mcp/tools.go` | Appears in tools/list |
| 2.3 | Implement chat handler | **Yes** | `internal/mcp/handlers.go` | Calls LLMExecutor |
| 2.4 | Session continuity | **Yes** | `internal/mcp/handlers.go` | session_id works |
| 2.5 | Error handling | **Yes** | `internal/mcp/handlers.go` | Returns isError: true |
| 2.6 | Flight Log for tools | **Yes** | `internal/mcp/handlers.go` | Tool invocations logged (see Flight Log Integration below) |
| 2.7 | Implement wingmate_status | **Yes** | `internal/mcp/handlers.go` | Returns `{uptime_ms, sessions_active, cli_available}` |
| 2.8 | Implement wingmate_discover | **Yes** | `internal/mcp/handlers.go` | Returns `{peers: [], agent_id}` |

#### Test Cases (TDD)

```go
// tools_test.go
func TestToolsListIncludesWingmateChat(t *testing.T)
func TestWingmateChatSchema(t *testing.T)

// handlers_test.go
func TestWingmateChatSuccess(t *testing.T)
func TestWingmateChatMissingPrompt(t *testing.T)
func TestWingmateChatSessionContinuity(t *testing.T)
func TestWingmateChatCLINotAvailable(t *testing.T)
func TestWingmateChatTimeout(t *testing.T)
func TestWingmateChatLogsToFlightLog(t *testing.T)
```

#### Tool Schemas

**wingmate_chat**:
```json
{
  "name": "wingmate_chat",
  "description": "Send a prompt to Claude CLI and get a response",
  "inputSchema": {
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
}
```

**wingmate_status**:
```json
{
  "name": "wingmate_status",
  "description": "Get Wingmate server health status",
  "inputSchema": {
    "type": "object",
    "properties": {}
  }
}
```

**wingmate_discover**:
```json
{
  "name": "wingmate_discover",
  "description": "List known peer agents",
  "inputSchema": {
    "type": "object",
    "properties": {}
  }
}
```

#### Flight Log Integration (Task 2.6 Details)

**Trace ID Generation**:
- Generate UUID v4 per MCP connection (on `initialize` request)
- Store in server context: `ctx = context.WithValue(ctx, TraceIDKey, uuid.New().String())`
- All tool invocations inherit trace_id from connection context

**Session ID Propagation**:
- If tool input includes `session_id`: use provided value
- If no `session_id` provided: auto-generate via `SessionManager.Create()`
- Return `session_id` in tool output for client to reuse

**Flight Log Entry Mapping**:
```go
// Maps to existing flightlog.Entry format
entry := flightlog.Entry{
    Timestamp: time.Now(),
    TraceID:   ctx.Value(TraceIDKey).(string),  // From MCP connection
    Event:     "mcp_tool_invocation",
    Tool:      toolName,                         // "wingmate_chat", etc.
    Input:     truncate(input, 200),             // Security: truncate
    Output:    truncate(output, 500),            // Security: truncate
    Duration:  duration.Milliseconds(),
    Status:    status,                           // "success" or "error"
    ErrorCode: errorCode,                        // 0 on success, 3xxx on error
    SessionID: sessionID,                        // For conversation continuity
}
```

**Backwards Compatibility**: MCPToolEntry fields map directly to existing `flightlog.Entry` struct. No schema changes required. New `Event` value `"mcp_tool_invocation"` distinguishes from A2A entries.

#### Test Commands (Phase 2)

```bash
# Run all Phase 2 tests with race detection
go test ./internal/mcp/... -v -race -count=3

# Run specific handler tests
go test ./internal/mcp/... -v -run "TestWingmateChat"

# Verify Flight Log entries in test output
go test ./internal/mcp/... -v -run "TestWingmateChatLogsToFlightLog"
```

#### Acceptance Criteria (Phase 2)

- [ ] tools/list returns wingmate_chat, wingmate_status, wingmate_discover
- [ ] wingmate_chat invocation returns LLM response with `session_id` in output
- [ ] Session continuity works (second call with session_id continues conversation)
- [ ] Missing Claude CLI returns error code 3001 with message "Claude CLI not available"
- [ ] All tool invocations logged to Flight Log with trace_id, session_id, duration_ms
- [ ] wingmate_status returns `{uptime_ms: N, sessions_active: N, cli_available: bool}`
- [ ] wingmate_discover returns `{peers: [], agent_id: "..."}`

---

### Phase 3: CLI Integration & Configuration

**Complexity**: CS-1
**Goal**: `wingmate mcp` subcommand fully functional

#### Prerequisites
- Phase 2 complete

#### Tasks

| ID | Task | TDD | File | Acceptance |
|----|------|-----|------|------------|
| 3.1 | Add mcp subcommand | No | `cmd/wingmate/main.go` | `wingmate mcp` runs |
| 3.2 | Wire up dependencies | No | `cmd/wingmate/main.go` | Executor, Session, FlightLog |
| 3.3 | Signal handling | No | `cmd/wingmate/main.go` | SIGINT/SIGTERM graceful |
| 3.4 | Config from env vars | No | `cmd/wingmate/main.go` | WINGMATE_LLM_* applied |
| 3.5 | Verbose flag | No | `cmd/wingmate/main.go` | `--verbose` mirrors logs |

#### Implementation Pattern

```go
// cmd/wingmate/main.go
case "mcp":
    return runMCP(cfg)

func runMCP(cfg *agent.Config) int {
    // Create LLM executor
    executor := llm.NewCLIExecutor(
        llm.WithModel(cfg.Claude.Model),
        llm.WithTimeout(cfg.Claude.Timeout),
    )

    // Check CLI availability
    if !executor.IsInstalled() {
        fmt.Fprintln(os.Stderr, "Warning: Claude CLI not installed")
    }

    // Create session manager
    sessionMgr := llm.NewSessionManager()

    // Create flight log
    flightLog, err := flightlog.New(cfg.FlightLog.Path, cfg.Verbose)
    if err != nil {
        return ExitInternal
    }
    defer flightLog.Close()

    // Create and run MCP server
    server := mcp.NewServer(
        mcp.WithExecutor(executor),
        mcp.WithSessionManager(sessionMgr),
        mcp.WithFlightLog(flightLog),
    )

    // Handle signals
    ctx, cancel := signal.NotifyContext(context.Background(),
        syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    if err := server.Run(ctx); err != nil {
        flightLog.Log("mcp_server_error", err.Error())
        return ExitInternal
    }

    return ExitSuccess
}
```

#### Environment Variables (Task 3.4 Details)

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `WINGMATE_LLM_MODEL` | string | `claude-sonnet-4-20250514` | Model for LLM invocations |
| `WINGMATE_LLM_TIMEOUT` | int (seconds) | `120` | Timeout for LLM calls |
| `WINGMATE_FLIGHT_LOG_PATH` | string | `./flight.jsonl` | Flight Log output path |
| `WINGMATE_MCP_DEBUG` | bool | `false` | Enable debug logging to stderr |

**Precedence**: Environment variables override config file defaults.

#### Test Commands (Phase 3)

```bash
# Test CLI subcommand parsing
go test ./cmd/wingmate/... -v -run "TestMCP"

# Test signal handling (manual)
/usr/local/bin/wingmate mcp &
PID=$!
sleep 2
kill -SIGINT $PID  # Should exit gracefully

# Verify env var handling
WINGMATE_LLM_MODEL=test-model wingmate mcp &
# Check Flight Log for model setting
```

#### Acceptance Criteria (Phase 3)

- [ ] `wingmate mcp` starts MCP server and listens on stdio
- [ ] SIGINT/SIGTERM triggers graceful shutdown within 5 seconds
- [ ] In-flight requests complete before shutdown (verified via test)
- [ ] `WINGMATE_LLM_MODEL` environment variable sets model in LLMExecutor
- [ ] `WINGMATE_LLM_TIMEOUT` environment variable sets timeout in seconds
- [ ] `--verbose` flag outputs Flight Log entries to stdout in addition to file

---

### Phase 4: Documentation & Integration Testing

**Complexity**: CS-1
**Goal**: Users can successfully configure Claude Code

#### Prerequisites
- Phase 3 complete

#### Tasks

| ID | Task | File | Acceptance |
|----|------|------|------------|
| 4.1 | Write mcp-setup.md | `docs/how/mcp-setup.md` | Complete guide |
| 4.2 | Update README.md | `README.md` | Quick-start section |
| 4.3 | Update CLAUDE.md | `CLAUDE.md` | MCP context |
| 4.4 | Integration tests | `tests/integration/mcp_test.go` | E2E with io.Pipe |
| 4.5 | Manual verification | N/A | Claude Code works |

#### Documentation Content

**README.md Quick-Start**:
```markdown
## Using with Claude Code

1. Build and install:
   ```bash
   go build -o /usr/local/bin/wingmate ./cmd/wingmate
   ```

2. Add to `~/.claude.json`:
   ```json
   {
     "mcpServers": {
       "wingmate": {
         "command": "/usr/local/bin/wingmate",
         "args": ["mcp"],
         "type": "stdio"
       }
     }
   }
   ```

3. Restart Claude Code and use `wingmate_chat` tool.
```

**docs/how/mcp-setup.md** (full guide):
- Prerequisites
- Installation options (build, download)
- Configuration (all platforms)
- Environment variables
- Troubleshooting
- Verification steps

#### Test Commands (Phase 4)

```bash
# Run integration tests with race detection
go test ./tests/integration/... -v -race -count=1

# Run specific MCP integration test
go test ./tests/integration/... -v -run "TestMCPIntegration"

# Verify documentation links (markdown lint)
# Optional: npx markdownlint-cli docs/how/mcp-setup.md README.md

# Manual verification script
./scripts/verify-mcp.sh  # Creates this in Phase 4
```

#### Acceptance Criteria (Phase 4)

- [ ] README has MCP quick-start section with working code snippets
- [ ] docs/how/mcp-setup.md covers macOS, Linux, and Windows setup
- [ ] CLAUDE.md includes MCP server context and error codes
- [ ] Integration tests pass: `go test ./tests/integration/... -v -race`
- [ ] Manual verification: Claude Code shows wingmate_chat tool connected
- [ ] All documentation examples verified working

---

### Phase 5: Optional Extensions (Future)

**Complexity**: CS-1 each
**Goal**: Additional tools and features

#### Optional Tasks

| ID | Task | Priority | Notes |
|----|------|----------|-------|
| 5.1 | wingmate_send tool | P2 | A2A message dispatch |
| 5.2 | Config file support | P2 | ~/.wingmate/config.yaml |
| 5.3 | Health endpoint | P2 | For monitoring |
| 5.4 | Rate limiting | P3 | Per-session limits |

These are deferred to post-MVP based on user feedback.

---

## 7. Cross-Cutting Concerns

### Error Code Allocation

| Range | Domain | Examples |
|-------|--------|----------|
| 3001-3009 | Tool execution | 3001 CLI unavailable, 3002 execution failed |
| 3010-3019 | Server lifecycle | 3010 initialization failed |
| 3020-3029 | Transport | 3020 invalid JSON, 3021 message too large |
| 3030-3099 | Reserved | Future use |

### Flight Log Entry Format

```go
type MCPToolEntry struct {
    Timestamp     time.Time `json:"ts"`
    TraceID       string    `json:"trace_id"`
    Tool          string    `json:"tool"`
    InputSummary  string    `json:"input_summary"`  // First 200 chars
    OutputSummary string    `json:"output_summary"` // First 500 chars
    Duration      int64     `json:"duration_ms"`
    Status        string    `json:"status"` // "success" or "error"
    ErrorCode     int       `json:"error_code,omitempty"`
    SessionID     string    `json:"session_id,omitempty"`
}
```

### Security Considerations

1. **Prompt Truncation**: Log only first 200 chars of prompts (P6)
2. **Response Truncation**: Log only first 500 chars of responses
3. **No Secrets**: Never log API keys, tokens, credentials
4. **Local Only**: stdio transport = same-machine, no network auth needed

### Observability Requirements (P1)

Every MCP operation MUST be logged:
- Server startup/shutdown
- Tool invocations (start, end, duration)
- Errors (with codes)
- Session creation/lookup

---

## 8. Complexity Tracking

### Initial Assessment

| Factor | Score | Rationale |
|--------|-------|-----------|
| Surface Area (S) | 1 | New package + CLI addition |
| Integration (I) | 1 | One external: MCP SDK |
| Data/State (D) | 0 | Reuses existing SessionManager |
| Novelty (N) | 2 | MCP protocol new to codebase |
| Non-Functional (F) | 1 | Observability required |
| Testing/Rollout (T) | 1 | Integration tests for stdio |

**Total**: 6 points → **CS-3**

### Phase Complexity

| Phase | Tasks | Estimated CS |
|-------|-------|--------------|
| Phase 1 | 7 | CS-2 |
| Phase 2 | 8 | CS-2 |
| Phase 3 | 5 | CS-1 |
| Phase 4 | 5 | CS-1 |
| Phase 5 | 4 | CS-1 (deferred) |

---

## 9. Progress Tracking

### Phase 1: Foundation & Protocol Layer

| Task | Status | Notes |
|------|--------|-------|
| 1.1 Define MCP types | ✅ Complete | internal/mcp/types.go |
| 1.2 Define error codes | ✅ Complete | internal/mcp/errors.go (3001-3022) |
| 1.3 Implement transport | ✅ Complete | internal/mcp/transport.go (NDJSON) |
| 1.4 Implement server lifecycle | ✅ Complete | internal/mcp/server.go |
| 1.5 Handle initialize request | ✅ Complete | Server handshake implemented |
| 1.6 Handle tools/list request | ✅ Complete | Returns registered tools |
| 1.7 Add Flight Log integration | ✅ Complete | Via Logger interface |

### Phase 2: Tool Implementation

| Task | Status | Notes |
|------|--------|-------|
| 2.1 Define tool schemas | ✅ Complete | internal/mcp/tools.go |
| 2.2 Register wingmate_chat | ✅ Complete | DefaultTools() |
| 2.3 Implement chat handler | ✅ Complete | internal/mcp/handlers.go |
| 2.4 Session continuity | ✅ Complete | session_id parameter |
| 2.5 Error handling | ✅ Complete | MCP error codes |
| 2.6 Flight Log for tools | ✅ Complete | Logged via handlers |
| 2.7 Implement wingmate_status | ✅ Complete | NewStatusHandler |
| 2.8 Implement wingmate_discover | ✅ Complete | NewDiscoverHandler |

### Phase 3: CLI Integration

| Task | Status | Notes |
|------|--------|-------|
| 3.1 Add mcp subcommand | ✅ Complete | cmd/wingmate/main.go |
| 3.2 Wire up dependencies | ✅ Complete | runMCP function |
| 3.3 Signal handling | ✅ Complete | Context cancellation |
| 3.4 Config from env vars | ✅ Complete | WINGMATE_LLM_* |
| 3.5 Verbose flag | ✅ Complete | --verbose for stderr logging |

### Phase 4: Documentation & Testing

| Task | Status | Notes |
|------|--------|-------|
| 4.1 Write mcp-setup.md | ✅ Complete | docs/how/mcp-setup.md created |
| 4.2 Update README.md | ✅ Complete | MCP quick-start section added |
| 4.3 Update CLAUDE.md | ✅ Complete | MCP context for AI assistants |
| 4.4 Integration tests | ✅ Complete | 6 tests with io.Pipe, all passing |
| 4.5 Manual verification | ✅ Complete | All tools respond correctly |

---

## 10. Change Footnotes Ledger

| ID | Date | Change | Rationale |
|----|------|--------|-----------|
| [^1] | 2026-01-22 | Initial plan created | Based on clarified spec |
| [^2] | 2026-01-22 | Added Flight Log integration details | Compliance with Constitution P1 |
| [^3] | 2026-01-22 | Added test commands per phase | Plan completeness validation |

### Deviation Ledger

| Item | Doctrine Reference | Deviation | Justification |
|------|-------------------|-----------|---------------|
| Package location: `internal/mcp/` | Architecture: `src/` for internal code | Using `internal/` instead of `src/` | ADR-003 established `internal/` pattern; Go idiom for non-exported packages; existing codebase uses `internal/` exclusively |
| New dependency: go-sdk | ADR-001: Minimal dependencies | Adding `modelcontextprotocol/go-sdk` | SDK has zero transitive deps; ADR-004 documents decision; 50-100x effort reduction vs custom impl |

---

## Appendix A: ADR-004 Summary

> **Note**: ADR-004 has been created and is now **DECIDED**. See `docs/adr/004-mcp-server-implementation.md` for the full decision record.

### Title: MCP Server Implementation Approach

### Status: DECIDED (2026-01-22)

### Context

Wingmate needs to expose LLM capabilities via MCP for Claude Code integration. This requires choosing an implementation approach that balances development effort, dependency management, and maintainability.

### Decision Drivers

- Constitution P1: All operations must be observable
- Constitution P4: MCP sanctioned for local tool integration
- ADR-001: Single Go binary with minimal dependencies
- Need for stable, well-tested MCP protocol implementation

### Considered Options

1. **Official SDK (modelcontextprotocol/go-sdk)**: Zero deps, v1.1.0, Google-backed
2. **Community SDK (mark3labs/mcp-go)**: More examples, 7.5k stars, pre-v1.0
3. **Custom implementation**: Full control, ~5-10k LOC, months of work

### Decision

Use the **official modelcontextprotocol/go-sdk v1.1.0**.

### Rationale

- Zero external dependencies in core package (matches Wingmate philosophy)
- Stable v1.x release (not beta)
- Backed by MCP team and Google
- Type-safe API with automatic JSON Schema generation
- If SDK issues arise, custom implementation from spec is tractable

### Consequences

- Single new dependency in go.mod
- Tied to SDK API (mitigate with adapter pattern)
- Protocol compliance guaranteed by official implementation

---

## Appendix B: Claude Code Configuration Examples

### Basic (~/.claude.json)

```json
{
  "mcpServers": {
    "wingmate": {
      "command": "/usr/local/bin/wingmate",
      "args": ["mcp"],
      "type": "stdio"
    }
  }
}
```

### With Environment Variables

```json
{
  "mcpServers": {
    "wingmate": {
      "command": "/usr/local/bin/wingmate",
      "args": ["mcp"],
      "env": {
        "WINGMATE_LLM_MODEL": "claude-sonnet-4-20250514",
        "WINGMATE_LLM_TIMEOUT": "120"
      },
      "type": "stdio"
    }
  }
}
```

### Project-Scoped (.mcp.json)

```json
{
  "mcpServers": {
    "wingmate": {
      "command": "/usr/local/bin/wingmate",
      "args": ["mcp"],
      "env": {
        "WINGMATE_PROJECT_PATH": "${PWD}"
      }
    }
  }
}
```

---

## Appendix C: Verification Commands

```bash
# Build wingmate
go build -o /usr/local/bin/wingmate ./cmd/wingmate

# Verify binary
/usr/local/bin/wingmate --version

# Test MCP server starts
/usr/local/bin/wingmate mcp
# (Should start, Ctrl+C to exit)

# Verify Claude Code config
claude mcp list
claude mcp get wingmate

# Test in Claude Code session
claude
# Then: /mcp (should show wingmate connected)

# Run tests
go test ./internal/mcp/... -v
go test ./internal/mcp/... -race
```

---

*Plan generated by /plan-3-architect based on clarified specification.*
