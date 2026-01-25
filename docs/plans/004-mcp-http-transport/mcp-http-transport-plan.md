# MCP HTTP Transport Migration Implementation Plan

**Plan Version**: 1.0.1
**Created**: 2026-01-23
**Spec**: [./mcp-http-transport-spec.md](./mcp-http-transport-spec.md)
**Status**: COMPLETE

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Technical Context](#technical-context)
3. [Gate Validations](#gate-validations)
4. [Critical Research Findings](#critical-research-findings)
5. [Testing Philosophy](#testing-philosophy)
6. [Implementation Phases](#implementation-phases)
   - [Phase 1: HTTP Transport Layer](#phase-1-http-transport-layer)
   - [Phase 2: Agent Integration](#phase-2-agent-integration)
   - [Phase 3: Stdio Removal](#phase-3-stdio-removal)
   - [Phase 4: Documentation](#phase-4-documentation)
7. [Cross-Cutting Concerns](#cross-cutting-concerns)
8. [Complexity Tracking](#complexity-tracking)
9. [Progress Tracking](#progress-tracking)
10. [Change Footnotes Ledger](#change-footnotes-ledger)

---

## Executive Summary

**Problem**: Wingmate's MCP server currently runs as an isolated stdio process (`wingmate mcp`), unable to share state with the running A2A agent. This prevents `wingmate_discover` from returning actual peers and requires users to manage two separate processes.

**Solution approach**:
- Replace stdio transport with HTTP transport, exposing `/mcp` endpoint on the existing agent HTTP server
- Implement application-level localhost check for security (server still listens on all interfaces for A2A)
- Add session management with 30-minute idle TTL for conversation continuity
- Complete removal of stdio-based MCP code (no migration path)

**Expected outcomes**:
- Single `wingmate --port 9000` process serves both A2A and MCP
- Claude Code connects via `claude mcp add --transport http wingmate http://localhost:9000/mcp`
- `wingmate_discover` returns actual known peer agents
- Shared Flight Log captures both A2A and MCP activity

**Success metrics**:
- All 10 acceptance criteria from spec pass
- Integration test demonstrates Claude Code connectivity
- Concurrent request handling verified with `-race` flag

---

## Technical Context

### Current System State

| Component | Current State | After Migration |
|-----------|--------------|-----------------|
| `internal/mcp/transport.go` | stdio (bufio.Scanner, io.Writer) | Deleted |
| `internal/mcp/server.go` | Run() loop reads stdin | HTTP handler via ServeHTTP() |
| `internal/agent/agent.go` | No MCP awareness | Hosts /mcp route |
| `cmd/wingmate/main.go` | runMCP() creates isolated server | MCP integrated into runServer() |
| Session state | Per-process only | Shared across HTTP requests |
| `wingmate_discover` | Returns empty array | Returns Agent.knownPeers |

### Integration Requirements

1. **A2AServer HTTP infrastructure**: Add `/mcp` route to existing switch statement in `ServeHTTP()`
2. **State sharing**: MCP handlers need access to Agent's peer registry via thread-safe interface
3. **Session management**: New MCPSessionManager with TTL-based cleanup
4. **Localhost middleware**: Application-level check returning 403 for non-localhost

### Constraints and Limitations

- No backward compatibility with stdio mode (complete removal)
- Single port for both A2A and MCP
- Localhost-only MCP access enforced at application level, not network binding
- No SSE streaming in initial implementation (synchronous POST only)
- Session ID via `Mcp-Session-Id` header per MCP Streamable HTTP spec

### Assumptions

1. Claude Code's `--transport http` follows MCP Streamable HTTP spec
2. Existing handlers (`handleInitialize`, `handleToolsList`, `handleToolsCall`) are transport-agnostic
3. `sync.RWMutex` pattern from codebase sufficient for concurrent access
4. `Request.RemoteAddr` reliably indicates client IP for localhost check

---

## Gate Validations

### GATE - Constitution

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Observability First | ✅ Pass | AC-8 requires Flight Log integration; all MCP invocations logged |
| P2: Deploy Early, Deploy Often | ✅ Pass | Single binary remains; upgrade = pull + rebuild |
| P3: Engineer Autonomy | ✅ Pass | Explicit configuration via `claude mcp add --transport http` |
| P4: Protocol Compliance | ✅ Pass | Implements MCP Streamable HTTP spec (2025-03-26) |
| P5: Themed but Tasteful | ✅ Pass | Wingmate terminology in tool names preserved |
| P6: Security by Design | ✅ Pass | Localhost-only MCP access; A2A unchanged |

**Deviation Ledger**: None required - plan fully compliant with constitution.

### GATE - Architecture

| Rule | Status | Notes |
|------|--------|-------|
| `internal/mcp/` MAY import `internal/flightlog/` | ✅ Pass | Existing pattern preserved |
| `internal/agent/` MAY import `internal/mcp/` | ✅ Pass | New dependency for route handling |
| `internal/mcp/` MUST NOT import `internal/agent/` | ⚠️ Addressed | Use interface injection (PeerProvider) |
| Module boundaries respected | ✅ Pass | New code follows existing patterns |

**Architectural Exception**: MCP handlers need peer data from Agent. Resolved via `PeerProvider` interface - Agent implements interface, MCP handlers receive interface (dependency inversion).

### GATE - ADR

| ADR | Status | Affects Phases | Notes |
|-----|--------|----------------|-------|
| ADR-001 (Go Language) | ✅ Compliant | All | Single binary, no new external deps |
| ADR-002 (Flight Log) | ✅ Compliant | 1, 2 | MCP events logged to existing Flight Log |
| ADR-003 (Unified Peers) | ✅ Compliant | 2 | MCP shares state with unified agent |
| ADR-004 (MCP Implementation) | ⚠️ Superseded | 3, 4 | This plan supersedes ADR-004's "stdio transport only" constraint |

**ADR-004 Superseding Declaration**:

This plan formally supersedes ADR-004's constraint stating "stdio transport only (no HTTP for MCP)".

| Aspect | Original (ADR-004) | Superseded (This Plan) |
|--------|-------------------|------------------------|
| Transport | stdio only | HTTP only (at `/mcp` endpoint) |
| Process Model | Isolated `wingmate mcp` subprocess | Integrated into `wingmate --port` process |
| Rationale | Initial simplicity | Enable peer discovery, shared state, single process |

**Superseding Justification**:
1. User requirement: stdio isolation prevents `wingmate_discover` from returning actual peers
2. Architectural improvement: Single process serves both A2A and MCP
3. Operational simplicity: One process to manage, one port to configure
4. Protocol compliance: MCP Streamable HTTP (2025-03-26) is the current standard

**Required Action**: Task 4.5 amends ADR-004 to remove the superseded constraint and document the HTTP transport decision.

---

## Critical Research Findings

### 🚨 Critical Discovery 01: Server.Run Loop Incompatible with HTTP
**Impact**: Critical
**Sources**: [I1-01]
**Problem**: Current `Server.Run()` contains a blocking stdin read loop incompatible with HTTP request-response semantics.
**Root Cause**: stdio transport is inherently sequential; HTTP handlers must be stateless per-request.
**Solution**: Create new `HTTPHandler` implementing `http.Handler`, converting loop to `ServeHTTP()` method.
**Action Required**: Phase 1 must create `internal/mcp/http_transport.go` with new handler type.
**Affects Phases**: Phase 1

### 🚨 Critical Discovery 02: A2AServer Route Integration Pattern
**Impact**: Critical
**Sources**: [I1-02]
**Problem**: `/mcp` route must be added to existing A2AServer without breaking A2A functionality.
**Root Cause**: Single HTTP server handles both protocols.
**Solution**: Add `/mcp` case to `ServeHTTP()` switch statement; localhost check inside handler.
```go
switch r.URL.Path {
case "/mcp":
    s.handleMCP(w, r)  // New case
case types.AgentCardWellKnownPath:
    s.handleAgentCard(w, r)
// ...
}
```
**Action Required**: Phase 2 adds route; Phase 1 creates the handler it delegates to.
**Affects Phases**: Phase 1, Phase 2

### 🚨 Critical Discovery 03: Concurrent Access Race Conditions
**Impact**: Critical
**Sources**: [R1-01, R1-02, R1-03]
**Problem**: HTTP enables concurrent requests; shared state (sessions, handlers, peers) must be thread-safe.
**Root Cause**: stdio was sequential; HTTP handlers run in parallel goroutines.
**Solution**: Use existing `sync.RWMutex` patterns; test with `-race` flag; thread-safe accessors.
```go
// Pattern from existing codebase
s.mu.RLock()
handler := s.handlers[toolName]
s.mu.RUnlock()
```
**Action Required**: All Phase 1/2 code must use consistent locking; all tests run with `-race`.
**Affects Phases**: Phase 1, Phase 2

### 🚨 Critical Discovery 04: Localhost Validation Security-Critical
**Impact**: Critical
**Sources**: [R1-04]
**Problem**: Application-level localhost check must handle IPv4/IPv6, not trust proxy headers.
**Root Cause**: Server listens on 0.0.0.0 for A2A; MCP restricted via code check.
**Solution**: Parse `Request.RemoteAddr`, check against `127.0.0.1`, `::1`, `ip.IsLoopback()`.
```go
func isLocalhost(r *http.Request) bool {
    host, _, _ := net.SplitHostPort(r.RemoteAddr)
    ip := net.ParseIP(host)
    return ip != nil && ip.IsLoopback()
}
```
**Action Required**: Phase 1 TDD for localhost middleware with IPv4/IPv6 edge cases.
**Affects Phases**: Phase 1, Phase 2

### High Discovery 05: Session Management with TTL Cleanup
**Impact**: High
**Sources**: [I1-03, R1-02, R1-06]
**Problem**: 30-minute idle TTL requires background cleanup goroutine racing with active handlers.
**Root Cause**: Spec Q8 mandates TTL; cleanup goroutine is new concurrent access pattern.
**Solution**: Copy-on-write cleanup; re-verify under write lock; 5-minute cleanup interval.
**Action Required**: Phase 1 creates `internal/mcp/session.go` with full TDD for thread safety.
**Affects Phases**: Phase 1, Phase 2

### High Discovery 06: Discover Handler Needs Peer Access
**Impact**: High
**Sources**: [I1-05, R1-03]
**Problem**: `wingmate_discover` currently returns empty array; needs Agent's peer registry.
**Root Cause**: Spec Q7 says "returns peers only, not self" - requires shared state access.
**Solution**: Define `PeerProvider` interface; Agent implements it; inject into handler.
```go
type PeerProvider interface {
    ListKnownPeers() []string
}
```
**Action Required**: Phase 2 defines interface and wires Agent to MCP handlers.
**Affects Phases**: Phase 2

### High Discovery 07: Rollback Strategy - HTTP Before Stdio Deletion
**Impact**: High
**Sources**: [R1-07]
**Problem**: Complete stdio removal is breaking change with no fallback.
**Root Cause**: User accepted breaking change; must verify HTTP works first.
**Solution**: Phase 1/2 add HTTP alongside existing code; Phase 3 deletion only after verification.
**Verification checklist**:
- [ ] HTTP transport accepts MCP initialize
- [ ] tools/list returns all 3 tools
- [ ] tools/call works for wingmate_status
- [ ] Localhost check blocks non-local access
- [ ] Claude Code connects successfully
**Action Required**: Integration test gate before Phase 3.
**Affects Phases**: Phase 1, Phase 3

**Rollback Strategy** (per rules.md § 6.3):
1. **Pre-deletion checkpoint**: Create git tag `pre-stdio-removal` before Phase 3 begins
2. **Verification gate**: Phase 2 integration test MUST pass before any Phase 3 deletions
3. **Rollback procedure**: If HTTP transport fails in production:
   - `git checkout pre-stdio-removal` to restore stdio code
   - Build and deploy stdio version
   - Debug HTTP issues without production pressure
4. **stdio code preserved**: Until Phase 3 task 3.1, all stdio code remains functional

### Medium Discovery 08: JSON-RPC Error Code Consistency
**Impact**: Medium
**Sources**: [R1-08]
**Problem**: Multiple error code systems (JSON-RPC standard, MCP custom, LLM) need consistent mapping.
**Root Cause**: Error codes spread across `internal/protocol/errors.go`, `internal/mcp/errors.go`, `internal/llm/errors.go`.
**Solution**: Map internal errors to JSON-RPC wire format; use standard codes for protocol errors.
**Action Required**: Phase 1 error handling follows existing patterns in `sendError()`.
**Affects Phases**: Phase 1, Phase 2

---

## Testing Philosophy

### Testing Approach
**Selected Approach**: Hybrid (Full TDD where it makes sense)
**Rationale**: HTTP transport layer and localhost security check warrant full TDD. Protocol handlers are already tested; adapt existing tests. Integration tests for end-to-end Claude Code connectivity.

### Test-Driven Development (Full TDD Phases)

**Phase 1 components requiring TDD**:
- `http_transport.go` - new code, request/response handling
- `localhost.go` - security-critical, IPv4/IPv6 edge cases
- `session.go` - concurrent access, TTL cleanup races

**TDD cycle**:
- Write tests FIRST (RED)
- Implement minimal code (GREEN)
- Refactor for quality (REFACTOR)

### Test-Assisted Development (Lighter Phases)

**Phase 2 components with lighter testing**:
- Handler integration - existing handlers, minimal changes
- Route addition - simple switch case

**Phase 3 deletion**:
- No new tests (removing code)
- Verify existing tests still pass after deletion

### Mock Usage (Targeted)

Per spec clarification: "Targeted mocks - Mock external dependencies (LLM executor) and slow components; use real HTTP for integration tests."

| Dependency | Mock Usage | Rationale (WHY real not used - per rules.md § 3.7) |
|------------|------------|---------------------------------------------------|
| `LLMExecutor` | Mock | Real Claude CLI requires API key, adds 1-3s latency, creates external network dependency; unit tests must be fast and hermetic |
| `http.ResponseWriter` | Mock | Unit testing HTTP handler response logic in isolation without full HTTP stack; enables precise assertion on status codes and headers |
| Peer registry | Mock | Isolate session tests from Agent lifecycle; test session behavior independently of peer discovery |
| HTTP server | Real | Integration tests need actual TCP/HTTP to verify real client connectivity, headers, and network behavior |

### Test Documentation

Every promoted test must include:
```go
/*
Test Doc:
- Why: [business/bug/regression reason]
- Contract: [invariant this test asserts]
- Usage Notes: [how to use the code; gotchas]
- Quality Contribution: [what failure this catches]
- Worked Example: [inputs/outputs]
*/
```

---

## Implementation Phases

### Phase 1: HTTP Transport Layer

**Objective**: Create HTTP-based transport infrastructure replacing stdio, with session management and localhost security.

**Deliverables**:
- `internal/mcp/http_transport.go` - HTTP handler implementing `http.Handler`
- `internal/mcp/session.go` - MCPSessionManager with 30-min TTL cleanup
- `internal/mcp/localhost.go` - Localhost validation middleware
- Full test coverage with `-race` flag passing

**Dependencies**: None (foundational phase)

**Risks**:
| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Race conditions in session access | Medium | High | TDD with `-race` flag; copy-on-write cleanup |
| IPv6 localhost edge cases | Low | Medium | Comprehensive test cases for `::1`, brackets |
| Session memory leak | Low | Medium | Background cleanup + max session cap |

### Tasks (Full TDD Approach)

| # | Status | Task | CS | Success Criteria | Log | Notes |
|---|--------|------|----|------------------|-----|-------|
| 1.1 | [ ] | Write comprehensive tests for localhost validation | 2 | Tests cover: IPv4 `127.0.0.1`, IPv6 `::1`, non-localhost, malformed RemoteAddr | - | Create `internal/mcp/localhost_test.go` |
| 1.2 | [ ] | Implement localhost validation middleware | 2 | All tests from 1.1 pass | - | Create `internal/mcp/localhost.go` |
| 1.3 | [ ] | Write comprehensive tests for MCPSessionManager | 2 | Tests cover: create session, get session, TTL expiry, concurrent access | - | Create `internal/mcp/session_test.go` |
| 1.4 | [ ] | Implement MCPSessionManager with TTL cleanup | 3 | All tests from 1.3 pass; cleanup goroutine runs | - | Create `internal/mcp/session.go` |
| 1.5 | [ ] | Write comprehensive tests for HTTPHandler | 2 | Tests cover: POST /mcp initialize, tools/list, tools/call, GET rejected | - | Create `internal/mcp/http_transport_test.go` |
| 1.6 | [ ] | Implement HTTPHandler serving /mcp requests | 3 | All tests from 1.5 pass; JSON-RPC responses correct | - | Create `internal/mcp/http_transport.go` |
| 1.7 | [ ] | Wire HTTPHandler to reuse existing handleMessage | 2 | HTTP handler delegates to existing protocol handlers | - | Refactor `server.go` to expose handlers |
| 1.8 | [ ] | Run all tests with `-race` flag | 1 | No data races detected | - | `go test ./internal/mcp/... -race` |

### Test Examples (Write First!)

```go
// internal/mcp/localhost_test.go
func TestIsLocalhost(t *testing.T) {
    /*
    Test Doc:
    - Why: AC-5 requires localhost-only MCP access via application check
    - Contract: isLocalhost returns true only for loopback addresses
    - Usage Notes: Check Request.RemoteAddr, not proxy headers
    - Quality Contribution: Prevents remote MCP access security vulnerability
    - Worked Example: "127.0.0.1:54321" → true, "192.168.1.1:54321" → false
    */

    tests := []struct {
        name       string
        remoteAddr string
        want       bool
    }{
        {"IPv4 localhost", "127.0.0.1:54321", true},
        {"IPv6 localhost", "[::1]:54321", true},
        {"IPv4 remote", "192.168.1.1:54321", false},
        {"IPv6 remote", "[2001:db8::1]:54321", false},
        {"empty", "", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest("POST", "/mcp", nil)
            req.RemoteAddr = tt.remoteAddr

            if got := isLocalhost(req); got != tt.want {
                t.Errorf("isLocalhost() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

```go
// internal/mcp/session_test.go
func TestSessionManager_CleanupExpired(t *testing.T) {
    /*
    Test Doc:
    - Why: Spec Q8 requires 30-min idle TTL for session cleanup
    - Contract: Sessions inactive > maxAge are removed; active sessions preserved
    - Usage Notes: Cleanup races with Get/Create; uses copy-on-write pattern
    - Quality Contribution: Prevents unbounded memory growth in long-running server
    - Worked Example: Session inactive 31min → deleted; session accessed 5min ago → kept
    */

    sm := NewMCPSessionManager()

    // Create session and artificially age it
    sess := sm.Create()
    sess.lastAccess = time.Now().Add(-31 * time.Minute)

    // Create another session that's recent
    recent := sm.Create()

    // Run cleanup
    count := sm.CleanupExpired(30 * time.Minute)

    if count != 1 {
        t.Errorf("CleanupExpired() removed %d sessions, want 1", count)
    }

    if _, ok := sm.Get(sess.ID); ok {
        t.Error("Expired session should have been removed")
    }

    if _, ok := sm.Get(recent.ID); !ok {
        t.Error("Recent session should NOT have been removed")
    }
}
```

### Non-Happy-Path Coverage
- [ ] Malformed RemoteAddr (no port)
- [ ] Empty session ID header
- [ ] Initialize called twice in same session
- [ ] Concurrent Create/Get/Cleanup operations
- [ ] Invalid JSON in request body

### Test Commands

```bash
# Run all Phase 1 tests
go test -v ./internal/mcp/... -run "TestIsLocalhost|TestSessionManager|TestHTTPHandler"

# Run with race detector
go test -v ./internal/mcp/... -race

# Check coverage
go test ./internal/mcp/... -coverprofile=coverage.out && go tool cover -func=coverage.out | grep total
```

### Acceptance Criteria
- [ ] All tests passing (100% of phase tests)
- [ ] Test coverage > 80% for new code (`go tool cover -func=coverage.out | grep total` shows 80%+)
- [ ] `-race` flag passes with no warnings
- [ ] Mock usage limited to LLMExecutor

---

### Phase 2: Agent Integration

**Objective**: Integrate MCP HTTP handler into the Agent's HTTP server at `/mcp` route with real peer discovery.

**Deliverables**:
- `/mcp` route added to `A2AServer.ServeHTTP()`
- `PeerProvider` interface for handler access to peer registry
- Updated `wingmate_discover` returning actual peers
- Integration test proving end-to-end flow

**Dependencies**: Phase 1 must be complete (HTTP transport layer exists)

**Risks**:
| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Route conflicts with A2A | Low | Medium | Explicit path matching; `/mcp` is unique |
| Peer registry races | Medium | Medium | Thread-safe accessor via interface |
| Integration complexity | Medium | Medium | Incremental testing at each step |

### Tasks (Hybrid Approach - Integration Focus)

| # | Status | Task | CS | Success Criteria | Log | Notes |
|---|--------|------|----|------------------|-----|-------|
| 2.1 | [ ] | Define PeerProvider interface in internal/mcp | 1 | Interface with `ListKnownPeers() []string` method | - | `internal/mcp/interfaces.go` |
| 2.2 | [ ] | Implement PeerProvider on Agent struct | 2 | Agent.ListKnownPeers() returns peers with proper locking | - | Add to `internal/agent/agent.go` |
| 2.3 | [ ] | Update NewDiscoverHandler to accept PeerProvider | 2 | Handler returns real peers, excludes self | - | Modify `internal/mcp/handlers.go` |
| 2.4 | [ ] | Add MCP handler injection method to Agent | 2 | Agent.SetMCPHandler(http.Handler) stores reference | - | `internal/agent/agent.go` |
| 2.5 | [ ] | Add /mcp route to A2AServer.ServeHTTP | 2 | Route delegates to MCP handler; localhost check applied | - | `internal/protocol/server.go` |
| 2.6 | [ ] | Initialize MCP server in agent.New() | 3 | Agent creates HTTPHandler, registers tools, wires discover | - | `internal/agent/agent.go` |
| 2.7 | [ ] | Write integration test for end-to-end MCP over HTTP | 2 | Test starts agent, sends MCP requests to /mcp, verifies responses | - | Create `internal/agent/mcp_integration_test.go` |
| 2.8 | [ ] | Test concurrent MCP and A2A requests | 2 | Both protocols work simultaneously without interference | - | Add to integration test |

### Test Examples

```go
// internal/agent/mcp_integration_test.go
func TestMCPOverHTTP_Integration(t *testing.T) {
    /*
    Test Doc:
    - Why: AC-1 through AC-4 require MCP works over HTTP on agent server
    - Contract: /mcp endpoint accepts JSON-RPC, returns valid responses
    - Usage Notes: Start real agent server; use httptest client
    - Quality Contribution: Proves end-to-end before Phase 3 deletion
    - Worked Example: POST /mcp with initialize → 200, serverInfo returned
    */

    // Create and start agent
    cfg := agent.NewConfig()
    cfg.Name = "test-agent"
    cfg.Port = 0 // Random port
    a, err := agent.New(cfg)
    if err != nil {
        t.Fatal(err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go a.Start(ctx)
    a.WaitUntilReady(5 * time.Second)

    // Send MCP initialize request
    initReq := map[string]any{
        "jsonrpc": "2.0",
        "id":      1,
        "method":  "initialize",
        "params": map[string]any{
            "protocolVersion": "2024-11-05",
            "clientInfo": map[string]any{
                "name":    "test-client",
                "version": "1.0.0",
            },
        },
    }

    body, _ := json.Marshal(initReq)
    resp, err := http.Post(a.URL()+"/mcp", "application/json", bytes.NewReader(body))
    if err != nil {
        t.Fatal(err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        t.Errorf("Expected 200, got %d", resp.StatusCode)
    }

    var result map[string]any
    json.NewDecoder(resp.Body).Decode(&result)

    if result["error"] != nil {
        t.Errorf("Unexpected error: %v", result["error"])
    }
}
```

### Non-Happy-Path Coverage
- [ ] MCP request from non-localhost IP
- [ ] A2A request during MCP processing
- [ ] Discover with no known peers
- [ ] Invalid tool name in tools/call

### Test Commands

```bash
# Run Phase 2 integration tests
go test -v ./internal/agent/... -run "TestMCPOverHTTP"

# Run with race detector for concurrent access
go test -v ./internal/agent/... -race

# Full test suite (Phase 1 + Phase 2)
go test -v ./internal/mcp/... ./internal/agent/... -race
```

### Acceptance Criteria
- [ ] Integration test passes
- [ ] AC-1 (HTTP endpoint available) verified
- [ ] AC-4 (Discovery returns peers) verified
- [ ] AC-5 (Localhost-only) verified
- [ ] AC-9 (Concurrent requests) verified with `-race`

---

### Phase 3: Stdio Removal

**Objective**: Complete removal of all stdio-based MCP code, ensuring clean codebase with no dead code.

**Deliverables**:
- `internal/mcp/transport.go` deleted
- `runMCP()` function in `main.go` deleted
- `mcpLoggerAdapter` type deleted
- `wingmate mcp` command returns helpful error message
- Updated CLI help text

**Dependencies**:
- Phase 2 must be complete
- Integration test from Phase 2 must pass
- Manual verification that Claude Code connects via HTTP

**Risks**:
| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Users dependent on stdio | High | High | Breaking change accepted; documentation |
| Missed code references | Low | Low | Grep for removed identifiers |
| Test failures from missing code | Low | Medium | Update affected tests |

### Tasks (Lightweight Approach - Deletion)

| # | Status | Task | CS | Success Criteria | Log | Notes |
|---|--------|------|----|------------------|-----|-------|
| 3.1 | [ ] | Delete `internal/mcp/transport.go` | 1 | File removed; no compile errors | - | 56 lines removed |
| 3.2 | [ ] | Delete `runMCP()` function from main.go | 1 | Function removed; no compile errors | - | Delete function `runMCP()` |
| 3.3 | [ ] | Delete `mcpLoggerAdapter` type from main.go | 1 | Type removed; no compile errors | - | Delete type `mcpLoggerAdapter` and methods |
| 3.4 | [ ] | Update CLI help text (remove "mcp" command) | 1 | Help shows HTTP-only MCP usage | - | Update usage() |
| 3.5 | [ ] | Add "mcp" command stub with migration message | 1 | `wingmate mcp` prints helpful error | - | AC-6 |
| 3.6 | [ ] | Update `NewTransport()` references | 1 | No remaining references to deleted function | - | Grep verification |
| 3.7 | [ ] | Run full test suite | 1 | All tests pass; no new failures | - | `go test ./...` |
| 3.8 | [ ] | Verify Claude Code HTTP connection | 1 | Claude Code initialize succeeds, tools/list returns 3 tools, wingmate_status returns valid JSON | - | End-to-end validation with real Claude Code |

### Test Examples

```go
// cmd/wingmate/main_test.go
func TestMCPCommand_ShowsHTTPMigrationMessage(t *testing.T) {
    /*
    Test Doc:
    - Why: AC-6 requires helpful error when user runs deprecated "wingmate mcp"
    - Contract: Exit code non-zero; stderr contains HTTP endpoint guidance
    - Usage Notes: Run `wingmate mcp` and capture stderr
    - Quality Contribution: Eases user transition from stdio to HTTP
    - Worked Example: "wingmate mcp" → "MCP server now available at /mcp endpoint..."
    */

    // This is a manual verification task, not automated test
    // Verify: `./wingmate mcp` prints message like:
    // "The 'mcp' command has been removed. MCP is now available via HTTP.
    //  Start the agent: wingmate --port 9000
    //  Configure Claude Code: claude mcp add --transport http wingmate http://localhost:9000/mcp"
}
```

### Non-Happy-Path Coverage
- [ ] User runs `wingmate mcp --verbose` (still shows migration message)
- [ ] References to Transport type compile-fail after deletion

### Acceptance Criteria
- [ ] `internal/mcp/transport.go` does not exist
- [ ] `grep -r "NewTransport" internal/` returns no matches
- [ ] `grep -r "runMCP" cmd/` returns no matches
- [ ] AC-6 (stdio mode removed) verified
- [ ] All existing tests pass

---

### Phase 4: Documentation

**Objective**: Update all documentation to reflect HTTP transport, including README quick-start, detailed setup guide, and CLAUDE.md context.

**Deliverables**:
- Updated `README.md` with HTTP MCP setup
- Updated `docs/how/mcp-setup.md` with HTTP configuration
- Updated `CLAUDE.md` MCP section
- Migration guide from stdio (in mcp-setup.md)

**Dependencies**: Phases 1-3 complete (documentation reflects final state)

**Risks**:
| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Documentation drift | Medium | Medium | Include doc updates in phase acceptance |
| Unclear examples | Low | Medium | Use real code snippets from implementation |

### Discovery & Placement Decision

**Existing docs/how/ structure**:
```
docs/how/
├── mcp-setup.md  (exists - needs major update)
└── llm-setup.md  (exists - unchanged)
```

**Decision**: Update existing `docs/how/mcp-setup.md` (no new directory needed)

**File strategy**: Major update to existing file (replacing stdio with HTTP content)

### Tasks (Lightweight Approach for Documentation)

| # | Status | Task | CS | Success Criteria | Log | Notes |
|---|--------|------|----|------------------|-----|-------|
| 4.1 | [ ] | Update README.md MCP quick-start section | 2 | Shows `claude mcp add --transport http` command | - | `README.md` |
| 4.2 | [ ] | Rewrite docs/how/mcp-setup.md for HTTP transport | 3 | Complete guide with HTTP configuration, troubleshooting | - | `docs/how/mcp-setup.md` |
| 4.3 | [ ] | Add migration section to mcp-setup.md | 2 | Clear guidance for users upgrading from stdio | - | Section: "Migrating from Stdio" |
| 4.4 | [ ] | Update CLAUDE.md MCP section | 2 | Reflects HTTP transport, removes stdio references | - | `CLAUDE.md` |
| 4.5 | [ ] | Update ADR-004 to reflect HTTP decision | 2 | Remove "stdio transport only" constraint; add HTTP transport rationale | - | `docs/adr/004-mcp-server-implementation.md` |
| 4.6 | [ ] | Verify all documentation links | 1 | No broken internal links | - | `find docs -name "*.md" -exec grep -l "](.*)" {} \;` |
| 4.7 | [ ] | Update architecture.md with /mcp route pattern | 1 | Document HTTP transport and A2AServer routing | - | `docs/project-rules/architecture.md` |

### Content Outlines

**README.md section** (quick-start only):
```markdown
### MCP Integration (Claude Code)

Wingmate exposes MCP tools via HTTP for Claude Code integration:

​```bash
# Start Wingmate agent
wingmate --port 9000

# Configure Claude Code (run once)
claude mcp add --transport http wingmate http://localhost:9000/mcp
​```

Available tools: `wingmate_chat`, `wingmate_status`, `wingmate_discover`

For detailed setup, see [docs/how/mcp-setup.md](docs/how/mcp-setup.md).
```

**docs/how/mcp-setup.md** (major update):
- Overview: HTTP transport, single process architecture
- Prerequisites: Wingmate installed, Claude Code CLI
- Configuration: `claude mcp add --transport http` command
- Environment variables (unchanged)
- Troubleshooting: Updated for HTTP (connection refused, port in use)
- Migration from Stdio: Step-by-step upgrade guide
- Architecture: New diagram showing HTTP transport

**CLAUDE.md** (MCP section update):
- Remove "stdio transport" references
- Update running command to agent mode
- Update configuration examples

### Acceptance Criteria
- [ ] README quick-start works for new users
- [ ] mcp-setup.md complete for all user scenarios
- [ ] No references to stdio transport in docs
- [ ] Peer review passed for documentation clarity

---

## Cross-Cutting Concerns

### Security Considerations

| Concern | Approach |
|---------|----------|
| Localhost-only MCP | Application-level check on `/mcp` route returning 403 |
| No credential exposure | MCP tools don't return API keys or tokens |
| Audit logging | All MCP invocations logged to Flight Log (Constitution P1) |
| Session isolation | Sessions keyed by ID; one client can't access another's session |

### Observability

| Metric/Log | Location | Purpose |
|------------|----------|---------|
| MCP tool invocations | Flight Log | Track tool usage (AC-8) |
| Session lifecycle | Flight Log | Debug session issues |
| Protocol version negotiation | Flight Log | Compatibility debugging |
| Localhost rejections | Flight Log | Security monitoring |

### Documentation

Per spec Documentation Strategy (Hybrid):

| Content | Location | Reason |
|---------|----------|--------|
| Quick setup | README.md | First thing users need |
| Full configuration | docs/how/mcp-setup.md | Reference material |
| Troubleshooting | docs/how/mcp-setup.md | Edge cases |
| Migration guide | docs/how/mcp-setup.md | Breaking change |
| Project context | CLAUDE.md | Developer onboarding |

---

## Complexity Tracking

| Component | CS | Label | Breakdown (S,I,D,N,F,T) | Justification | Mitigation |
|-----------|-----|-------|------------------------|---------------|------------|
| HTTPHandler | 3 | Medium | S=1,I=1,D=1,N=1,F=1,T=1 | New file, HTTP semantics, session state | TDD; copy existing patterns |
| SessionManager | 3 | Medium | S=1,I=0,D=1,N=1,F=1,T=1 | Concurrent access, TTL cleanup | TDD with `-race`; copy-on-write |
| Agent Integration | 3 | Medium | S=1,I=1,D=1,N=0,F=1,T=1 | Route wiring, interface injection | Incremental integration |
| Localhost Middleware | 2 | Small | S=1,I=0,D=0,N=0,F=1,T=1 | Security critical but small scope | TDD for edge cases |

**Total Feature CS**: 3 (Medium) - as defined in spec

---

## Progress Tracking

### Phase Completion Checklist
- [x] Phase 1: HTTP Transport Layer - ✅ COMPLETE
- [x] Phase 2: Agent Integration - ✅ COMPLETE
- [x] Phase 3: Stdio Removal - ✅ COMPLETE
- [x] Phase 4: Documentation - ✅ COMPLETE

**Plan Status**: ✅ ALL PHASES COMPLETE (2026-01-24)

### STOP Rule

**IMPORTANT**: This plan must be validated before creating phase tasks.

**Next Steps**:
1. Run `/plan-4-complete-the-plan` to validate readiness
2. Only proceed to `/plan-5-phase-tasks-and-brief` after validation passes
3. Execute phases in order: 1 → 2 → 3 → 4

---

## Change Footnotes Ledger

**NOTE**: This section will be populated during implementation by plan-6a-update-progress.

**Footnote Numbering Authority**: plan-6a-update-progress is the single source of truth for footnote numbering.

**Initial State** (before implementation begins):

[^1]: [To be added during implementation via plan-6a]
[^2]: [To be added during implementation via plan-6a]
[^3]: [To be added during implementation via plan-6a]

---

## References

- [Feature Spec](./mcp-http-transport-spec.md)
- [MCP Streamable HTTP Specification](https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#streamable-http)
- [ADR-004: MCP Server Implementation](../../adr/004-mcp-server-implementation.md)
- [Constitution](../../project-rules/constitution.md)
- [Architecture](../../project-rules/architecture.md)
