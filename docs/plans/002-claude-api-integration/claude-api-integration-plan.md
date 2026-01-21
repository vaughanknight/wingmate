# Claude CLI Integration - Implementation Plan

**Spec Reference**: [claude-api-integration-spec.md](/Users/vaughanknight/GitHub/wingmate/docs/plans/002-claude-api-integration/claude-api-integration-spec.md)
**Research Reference**: [research-dossier.md](/Users/vaughanknight/GitHub/wingmate/docs/plans/002-claude-api-integration/research-dossier.md) (initial), [research-cli-integration.md](/Users/vaughanknight/GitHub/wingmate/docs/plans/002-claude-api-integration/research-cli-integration.md) (CLI approach)
**Created**: 2026-01-21
**Status**: Ready (CLI Pivot)
**Mode**: Full

> **UPDATE 2026-01-21**: Pivoted from direct API calls to Claude CLI integration

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Technical Context](#technical-context)
3. [Critical Research Findings](#critical-research-findings)
4. [Testing Philosophy](#testing-philosophy)
5. [ADR Alignment Ledger](#adr-alignment-ledger)
6. [Implementation Phases](#implementation-phases)
   - [Phase 1: Core LLM Types and Interface](#phase-1-core-llm-types-and-interface)
   - [Phase 2: CLI Executor Implementation](#phase-2-cli-executor-implementation)
   - [Phase 3: Configuration Extension](#phase-3-configuration-extension)
   - [Phase 4: Agent Integration](#phase-4-agent-integration)
   - [Phase 5: Documentation and Polish](#phase-5-documentation-and-polish)
7. [Cross-Cutting Concerns](#cross-cutting-concerns)
8. [Complexity Tracking](#complexity-tracking)
9. [Progress Tracking](#progress-tracking)
10. [Change Footnotes Ledger](#change-footnotes-ledger)
11. [Appendix A: Default System Prompt](#appendix-a-default-system-prompt)
12. [Appendix B: CLI Reference](#appendix-b-cli-reference)

---

## Executive Summary

This plan implements Claude CLI integration for Wingmate agents, enabling intelligent natural language message processing. The integration follows the existing unified peer architecture (ADR-003) and adds a new `internal/llm/` package for LLM interactions via CLI.

**Key Outcomes**:
- Agents can process natural language messages via Claude CLI
- Session continuity via CLI `--resume` flag (Wingmate manages session ID mapping)
- All CLI invocations logged to Flight Log (P1: Observability First)
- No API key management required (CLI handles its own authentication)
- Graceful degradation when Claude CLI unavailable
- Full TDD coverage with mocked exec.Command for CLI tests

**Complexity**: CS-3 (Medium) - 5 implementation phases, ~6-8 new files

---

## Technical Context

### Integration Point

The primary integration point is `HandleMessage()` in `/Users/vaughanknight/GitHub/wingmate/internal/agent/agent.go:227`:

```go
// Current implementation (agent.go:247-272)
if isPing(msg) {
    response := pongResponse()
    // ... log and return pong
    return &types.A2AResponse{...}, nil
}

// Unknown message type - THIS IS WHERE LLM ROUTING GOES
return nil, &protocol.ErrorObject{
    Code:    protocol.CodeMethodNotFound,
    Message: "unknown message type",
}
```

After integration, the flow becomes:
1. Check if ping → return pong (unchanged)
2. Check if CLI available (`a.llmExecutor != nil`) → route to Claude CLI
3. If not ping AND no CLI available → return error code 2001

### CLI Invocation Pattern

```bash
# First message (no session)
claude -p "What does this code do?" --output-format json

# Follow-up message (with session)
claude -p "And what about error handling?" --resume "session-id" --output-format json
```

### Session Management Flow

```
Conversation Start:
  1. Wingmate receives A2A message
  2. Execute: claude -p "message" --output-format json
  3. Parse JSON response, extract session_id
  4. Store session_id mapped to conversation
  5. Return response to peer

Conversation Continue:
  1. Wingmate receives follow-up A2A message
  2. Look up session_id for this conversation
  3. Execute: claude -p "message" --resume "session-id" --output-format json
  4. Return response to peer
```

### Existing Patterns to Follow

| Pattern | Source | Reuse Strategy |
|---------|--------|----------------|
| Subprocess Exec | Go `os/exec` | Standard patterns for command execution |
| Error Codes | `/Users/vaughanknight/GitHub/wingmate/internal/protocol/errors.go` | Add LLM-specific codes 2001-2006 |
| Configuration | `/Users/vaughanknight/GitHub/wingmate/internal/agent/config.go` | Extend with CLI settings |
| Flight Log | `/Users/vaughanknight/GitHub/wingmate/internal/flightlog/` | Add CLI invocation/response entry types |

---

## Critical Research Findings

From `/Users/vaughanknight/GitHub/wingmate/docs/plans/002-claude-api-integration/research-cli-integration.md`:

### 1. CLI Invocation

```bash
# Non-interactive (programmatic) mode
claude -p "prompt" --output-format json

# With specific session
claude -p "prompt" --resume "session-id" --output-format json

# Continue most recent session
claude -p "prompt" --continue --output-format json
```

### 2. Key CLI Flags

| Flag | Short | Purpose |
|------|-------|---------|
| `--print` | `-p` | Non-interactive mode (required for programmatic use) |
| `--output-format json` | | Structured JSON response with session_id |
| `--resume` | `-r` | Resume specific session by ID |
| `--continue` | `-c` | Continue most recent session |
| `--max-turns` | | Limit agentic turns |
| `--model` | | Select model (sonnet, opus, haiku) |

### 3. JSON Response Format

```json
{
  "result": "The assistant's text response",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "usage": {
    "input_tokens": 150,
    "output_tokens": 523
  },
  "metadata": {
    "model": "claude-sonnet-4-20250514"
  }
}
```

### 4. Exit Codes

| Code | Meaning | Wingmate Action |
|------|---------|-----------------|
| 0 | Success | Parse JSON, return response |
| 1 | Non-blocking error | Log warning, return partial if available |
| 2 | Blocking error | Log error, return error to peer |
| 127 | Not found | Return "Claude CLI not installed" error |

### 5. Installation Detection

```bash
# Check if installed
which claude        # Returns path or empty
command -v claude   # Returns path or empty

# Check version
claude --version
```

---

## Testing Philosophy

**Approach**: Full TDD (per rules.md and spec clarification)

### Test Naming Convention
Go projects use `*_test.go` suffix for test files (e.g., `cli_test.go`). This is the standard Go convention and aligns with ADR-001's choice of Go as the implementation language.

### Test Documentation Requirements
Per rules.md section 3.2, all test files must include documentation with these five elements:
1. **Why**: Purpose of the test suite
2. **Contract**: What behavior is being verified
3. **Usage Notes**: How to run, dependencies, setup
4. **Quality Contribution**: How tests improve confidence
5. **Worked Example**: Representative test case explanation

### Test Structure
```
/Users/vaughanknight/GitHub/wingmate/internal/llm/
├── cli_test.go         # CLI executor tests (mock exec.Command)
├── session_test.go     # Session manager tests
├── errors_test.go      # Error handling tests
└── types_test.go       # Request/response type tests

/Users/vaughanknight/GitHub/wingmate/internal/agent/
├── agent_test.go       # Extended for LLM routing tests
└── config_test.go      # Extended for CLI config tests
```

### Mock Strategy
- **Mock**: `exec.Command` for all CLI executor tests
- **Real**: All other components (config, routing, Flight Log, session management)
- **Skip**: Integration tests when CLI not installed (`t.Skip` pattern)
- **Rationale**: Avoids requiring CLI installed in CI while maintaining realistic tests

### Test Coverage Targets
| Area | Coverage Target | Rationale |
|------|-----------------|-----------|
| CLI executor | 90%+ | External subprocess, error paths critical |
| Session manager | 100% | Core state management |
| Error handling | 100% | All error codes must be tested |
| Agent routing | 90%+ | Integration point, existing tests |
| Config loading | 80%+ | Extend existing tests |

---

## ADR Alignment Ledger

| ADR ID | Title | Status | Plan Alignment | Constraints Observed |
|--------|-------|--------|----------------|---------------------|
| ADR-001 | Language Choice (Go) | Decided | Aligned | Using `os/exec` standard library for CLI execution; no external dependencies |
| ADR-002 | Flight Log Storage | Decided | Aligned | CLI invocation/response entries use same JSONL format with `direction` field |
| ADR-003 | Unified Peer Architecture | Decided | Aligned | LLM integration happens in unified Agent; no mode-specific handling |

**Note**: Plan uses CLI-based integration which eliminates the need for API key management. The CLI handles its own authentication.

---

## Implementation Phases

### Phase 1: Core LLM Types and Interface

**Goal**: Establish the `internal/llm/` package with types and interface definition for CLI-based execution.

**Dependencies**: None (first phase)

**Constitution Alignment**: P4 (Protocol Compliance) - clean interface boundaries

**Dossier**: [tasks/phase-1-core-llm-types-and-interface/tasks.md](./tasks/phase-1-core-llm-types-and-interface/tasks.md)

#### Task Table (TDD)

| # | Task | Test First | Implementation | AC |
|---|------|------------|----------------|-----|
| 1.1 | Create `types.go` with CLI response types | `types_test.go`: JSON unmarshaling tests | Define `CLIResponse`, `Usage` structs matching CLI JSON output | AC8 |
| 1.2 | Create `client.go` with `LLMExecutor` interface | `client_test.go`: mock implementation test | Define interface: `Execute(ctx context.Context, prompt string, sessionID string) (*CLIResponse, error)`, `IsInstalled() bool` | AC8 |
| 1.3 | Create `errors.go` with error codes | `errors_test.go`: error code constants test | Define codes 2001-2006, `LLMError` type | AC4, AC5 |
| 1.4 | Create `convert.go` for response conversion | `convert_test.go`: CLI→A2A tests | `FromCLIResponse(resp *CLIResponse) (*types.Message, error)` | AC1 |

**Files Created**:
- `/Users/vaughanknight/GitHub/wingmate/internal/llm/types.go`
- `/Users/vaughanknight/GitHub/wingmate/internal/llm/types_test.go`
- `/Users/vaughanknight/GitHub/wingmate/internal/llm/client.go`
- `/Users/vaughanknight/GitHub/wingmate/internal/llm/client_test.go`
- `/Users/vaughanknight/GitHub/wingmate/internal/llm/errors.go`
- `/Users/vaughanknight/GitHub/wingmate/internal/llm/errors_test.go`
- `/Users/vaughanknight/GitHub/wingmate/internal/llm/convert.go`
- `/Users/vaughanknight/GitHub/wingmate/internal/llm/convert_test.go`

**Success Criteria**:
- [ ] CLIResponse type parses CLI JSON output correctly
- [ ] Interface allows mock implementation
- [ ] Error codes match spec (2001-2006)
- [ ] Conversion preserves text content
- [ ] `go test ./internal/llm/...` passes

**Verification Commands**:
```bash
# Run tests
go test -v ./internal/llm/...
# Expected: all tests pass (ok github.com/wingmate/wingmate/internal/llm)

# Check coverage
go test -cover ./internal/llm/...
# Expected: coverage >= 80%

# Verify package compiles
go build ./internal/llm/...
# Expected: no output (success)
```

**Complexity**: CS-1 (Low) - Types and interfaces only

---

### Phase 2: CLI Executor Implementation

**Goal**: Implement the Claude CLI executor with proper error handling.

**Dependencies**: Phase 1 (requires types and interface)

**Constitution Alignment**: P1 (Observability First)

**Dossier**: [tasks/phase-2-cli-executor-implementation/tasks.md](./tasks/phase-2-cli-executor-implementation/tasks.md)

#### Task Table (TDD)

| # | Task | Test First | Implementation | AC |
|---|------|------------|----------------|-----|
| 2.1 | Create `cli.go` with `CLIExecutor` struct | Test struct creation with config | Struct with cliPath, timeout, model | AC7 |
| 2.2 | Implement `IsInstalled()` method | Mock `exec.LookPath` | Check if `claude` binary exists in PATH | AC4 |
| 2.3 | Implement `Execute()` method | Mock `exec.Command` returning success JSON | Build command: `claude -p "prompt" --output-format json` | AC1 |
| 2.4 | Implement session resume | Mock with `--resume` flag | Add `--resume "session-id"` when sessionID provided | AC2 |
| 2.5 | Implement timeout handling | Mock slow command, verify context cancellation | Use `context.Context` with deadline | AC7 |
| 2.6 | Handle exit code 0 (success) | Mock exit 0 | Parse JSON, return CLIResponse | AC1 |
| 2.7 | Handle exit code 1 (non-blocking error) | Mock exit 1 | Log warning, return partial if available | AC5 |
| 2.8 | Handle exit code 2 (blocking error) | Mock exit 2 | Return `LLMError{Code: 2002}` | AC5 |
| 2.9 | Handle exit code 127 (not found) | Mock exit 127 | Return `LLMError{Code: 2001, Message: "Claude CLI not installed"}` | AC4 |

**Files Created**:
- `/Users/vaughanknight/GitHub/wingmate/internal/llm/cli.go`
- `/Users/vaughanknight/GitHub/wingmate/internal/llm/cli_test.go`

**Success Criteria**:
- [ ] Executor builds correct command line with flags
- [ ] Session resume uses `--resume` flag
- [ ] All exit codes return correct `LLMError`
- [ ] Timeout configurable, defaults to 120s
- [ ] `go test ./internal/llm/...` passes with 90%+ coverage

**Verification Commands**:
```bash
# Run tests with coverage
go test -v -coverprofile=coverage.out ./internal/llm/...
# Expected: all tests pass

# Check coverage percentage
go tool cover -func=coverage.out | grep total
# Expected: total coverage >= 90%

# Verify no race conditions
go test -race ./internal/llm/...
# Expected: no race conditions detected
```

**Complexity**: CS-2 (Low-Medium) - CLI execution with error handling

---

### Phase 3: Configuration Extension

**Goal**: Extend agent configuration to support Claude CLI settings.

**Dependencies**: Phase 1 (requires error codes for validation)

**Constitution Alignment**: P6 (Security by Design) - no sensitive data in config

**Dossier**: [tasks/phase-3-configuration-extension/tasks.md](./tasks/phase-3-configuration-extension/tasks.md)

#### Task Table (TDD)

| # | Task | Test First | Implementation | AC |
|---|------|------------|----------------|-----|
| 3.1 | Add ClaudeConfig struct (CLI version) | Test defaults and validation | Add `CLIPath string`, `Model string`, `Timeout time.Duration`, `SystemPrompt string` | AC7 |
| 3.2 | Add environment variable loading | Test env override for each field | Load from `WINGMATE_CLAUDE_CLI_PATH`, `WINGMATE_CLAUDE_MODEL`, `WINGMATE_CLAUDE_TIMEOUT` | AC7 |
| 3.3 | Add CLI path validation | Test invalid paths rejected | Validate: path exists if set, or fall back to PATH lookup | AC4 |
| 3.4 | Add default system prompt | Test default and custom prompts | Default prompt in code, override via `WINGMATE_CLAUDE_SYSTEM_PROMPT` | AC1 |
| 3.5 | Add timeout validation | Test timeout bounds | Validate: timeout > 0 | AC7 |

**Files Modified**:
- `/Users/vaughanknight/GitHub/wingmate/internal/agent/config.go`
- `/Users/vaughanknight/GitHub/wingmate/internal/agent/config_test.go`

**Environment Variables**:
| Variable | Default | Description |
|----------|---------|-------------|
| `WINGMATE_CLAUDE_CLI_PATH` | (auto-detect) | Custom path to claude binary |
| `WINGMATE_CLAUDE_MODEL` | (CLI default) | Model to use (optional, passed as --model) |
| `WINGMATE_CLAUDE_TIMEOUT` | `120s` | Execution timeout |
| `WINGMATE_CLAUDE_SYSTEM_PROMPT` | (default) | System prompt override |

**Note**: No `WINGMATE_CLAUDE_API_KEY` - CLI handles its own authentication.

**Success Criteria**:
- [ ] CLI path auto-detected or configurable
- [ ] Model passed to CLI if specified
- [ ] Timeout configurable with validation
- [ ] Default system prompt provides good debugging assistant behavior
- [ ] `go test ./internal/agent/...` passes

**Verification Commands**:
```bash
# Run config tests
go test -v ./internal/agent/... -run "Config"
# Expected: all config tests pass

# Run full agent test suite
go test -v ./internal/agent/...
# Expected: all tests pass
```

**Complexity**: CS-1 (Low) - Configuration extension

---

### Phase 4: Agent Integration

**Goal**: Modify `HandleMessage` to route non-ping messages to Claude CLI.

**Dependencies**: Phase 1, Phase 2, Phase 3 (requires types, executor, config)

**Constitution Alignment**: P1 (Observability First) - log all CLI interactions

**Dossier**: [tasks/phase-4-agent-integration/tasks.md](./tasks/phase-4-agent-integration/tasks.md)

#### Task Table (TDD)

| # | Task | Test First | Implementation | AC |
|---|------|------------|----------------|-----|
| 4.1 | Add LLM executor field to Agent | Test agent creation with executor | Add `llmExecutor llm.LLMExecutor` field | AC8 |
| 4.2 | Add session manager to Agent | Test session storage/retrieval | Add `sessionManager *llm.SessionManager` field | AC2 |
| 4.3 | Initialize executor in `New()` | Test executor initialized when CLI available | Create `CLIExecutor` if CLI is installed | AC4 |
| 4.4 | Modify `HandleMessage` routing | Test: ping→pong, text+CLI→Claude, text+noCLI→error | Routing: `if isPing → pong; else if executor != nil → CLI; else → error 2001` | AC6 |
| 4.5 | Implement LLM request path | Mock executor, verify response | Get/create session ID, call `Execute()`, store new session ID | AC1, AC2 |
| 4.6 | Handle missing CLI | Test error when CLI not installed | Return `protocol.ErrorObject{Code: 2001, Message: "Claude CLI not installed"}` | AC4 |
| 4.7 | Log CLI invocation to Flight Log | Verify entry with command, session | Create entry with `direction: outbound`, `payload.type: cli_request` | AC3 |
| 4.8 | Log CLI response to Flight Log | Verify entry with response, timing | Create entry with `direction: inbound`, `payload.type: cli_response` | AC3 |

**Routing Logic Clarification** (Task 4.4):
```go
func (a *Agent) HandleMessage(ctx context.Context, msg *types.Message) (*types.A2AResponse, error) {
    // 1. Handle ping (always works, even without CLI)
    if isPing(msg) {
        return pongResponse(), nil
    }

    // 2. If CLI available, route to Claude
    if a.llmExecutor != nil {
        return a.handleLLMMessage(ctx, msg)
    }

    // 3. No CLI available for non-ping message
    return nil, &protocol.ErrorObject{
        Code:    2001,  // LLMUnavailable
        Message: "Claude CLI not installed - install from https://claude.ai/download",
    }
}
```

**Files Modified**:
- `/Users/vaughanknight/GitHub/wingmate/internal/agent/agent.go`
- `/Users/vaughanknight/GitHub/wingmate/internal/agent/agent_test.go`

**Files Created**:
- `/Users/vaughanknight/GitHub/wingmate/internal/llm/session.go`
- `/Users/vaughanknight/GitHub/wingmate/internal/llm/session_test.go`

**Flight Log Entry Types Added**:
```json
// CLI Request entry
{
    "direction": "outbound",
    "summary": "CLI request to Claude",
    "payload": {
        "type": "cli_request",
        "session_id": "550e8400-e29b-41d4-a716-446655440000"
    }
}

// CLI Response entry
{
    "direction": "inbound",
    "summary": "CLI response received",
    "payload": {
        "type": "cli_response",
        "session_id": "550e8400-e29b-41d4-a716-446655440000",
        "execution_time_ms": 2340
    }
}
```

**Success Criteria**:
- [ ] Ping messages still return pong (AC6 verified)
- [ ] Non-ping text messages routed to CLI when available
- [ ] Non-ping without CLI returns error 2001
- [ ] Session ID stored and reused for follow-up messages
- [ ] Flight Log contains CLI request/response entries
- [ ] `go test ./internal/agent/...` passes

**Verification Commands**:
```bash
# Run agent tests including new LLM routing tests
go test -v ./internal/agent/... -run "HandleMessage"
# Expected: all HandleMessage tests pass

# Verify ping still works
go test -v ./internal/agent/... -run "Ping"
# Expected: ping/pong tests pass unchanged

# Run full test suite with coverage
go test -coverprofile=coverage.out ./internal/agent/...
go tool cover -func=coverage.out | grep total
# Expected: coverage >= 90%

# Verify Flight Log entries (integration test)
go test -v ./internal/agent/... -run "FlightLog"
# Expected: CLI entries present in log
```

**Complexity**: CS-2 (Medium) - Core integration point with session management

---

### Phase 5: Documentation and Polish

**Goal**: Create user documentation and finalize implementation.

**Dependencies**: Phase 1-4 (all implementation complete)

**Constitution Alignment**: Definition of Done (constitution section 3.3) - documentation required for completion

#### Task Table

| # | Task | Test First | Implementation | AC |
|---|------|------------|----------------|-----|
| 5.1 | Add LLM Support section to README | N/A | Quick setup (install CLI, configure), basic usage, link to guide | Docs |
| 5.2 | Create `docs/how/llm-setup.md` | N/A | CLI installation, env vars, model selection, troubleshooting | Docs |
| 5.3 | Update Agent Card with LLM skill | Test skill appears in card | Add "chat" skill to `buildAgentCard()` when CLI available | AC1 |
| 5.4 | Run full test suite | N/A | `go test ./...` passes | All |
| 5.5 | Run linter and format | N/A | `go fmt`, `go vet`, `staticcheck` | All |
| 5.6 | Update architecture.md | N/A | Add `internal/llm/` to module boundaries | Docs |

**Files Created/Modified**:
- `/Users/vaughanknight/GitHub/wingmate/README.md` (modified)
- `/Users/vaughanknight/GitHub/wingmate/docs/how/llm-setup.md` (new)
- `/Users/vaughanknight/GitHub/wingmate/internal/agent/agent.go` (Agent Card update)
- `/Users/vaughanknight/GitHub/wingmate/docs/project-rules/architecture.md` (add llm package)

**Documentation Structure**:

**README.md** (new section):
```markdown
## LLM Support

Wingmate agents can use Claude CLI to process natural language messages.

### Quick Setup

1. Install Claude CLI:
   - Visit https://claude.ai/download
   - Follow installation instructions for your platform

2. Authenticate Claude CLI (one-time):
   ```bash
   claude login
   ```

3. Run the agent:
   ```bash
   ./wingmate --port 9000
   ```

4. Send a message (ping still works, but now text messages get intelligent responses)

See [LLM Setup Guide](docs/how/llm-setup.md) for detailed configuration.
```

**Success Criteria**:
- [ ] README has clear quick-start for LLM
- [ ] Detailed guide covers CLI installation
- [ ] Troubleshooting section covers common errors (CLI not found, timeout)
- [ ] All tests pass
- [ ] Code passes linting
- [ ] Architecture doc updated with `internal/llm/` package

**Verification Commands**:
```bash
# Run full test suite
go test ./...
# Expected: exit 0, all tests pass

# Format code
go fmt ./...
# Expected: no output (already formatted) or files formatted

# Run vet
go vet ./...
# Expected: exit 0, no issues

# Verify docs exist
ls -la docs/how/llm-setup.md
# Expected: file exists
```

**Complexity**: CS-1 (Low) - Documentation only

---

## Cross-Cutting Concerns

### Error Handling Matrix

| Error Scenario | Exit Code | LLM Error Code | User Message | Log Entry |
|----------------|-----------|----------------|--------------|-----------|
| CLI not installed | 127 | 2001 | "Claude CLI not installed - install from https://claude.ai/download" | Warning |
| CLI execution failed | 2 | 2002 | "Claude CLI execution failed" | Error |
| CLI timeout | N/A | 2003 | "Claude CLI request timed out" | Warning |
| CLI non-blocking error | 1 | 2004 | "Claude CLI warning: {stderr}" | Warning |
| Session not found | N/A | 2005 | "Session expired or not found" | Info |

### Security Checklist

- [ ] No API keys managed by Wingmate (CLI handles auth)
- [ ] User prompts not logged in full (privacy)
- [ ] CLI stderr captured for debugging but not exposed to peers
- [ ] Session IDs are UUIDs (no information leakage)

### Observability Checklist

- [ ] CLI invocation logged before execution
- [ ] CLI response logged after execution
- [ ] Execution time recorded in milliseconds
- [ ] Session ID recorded (if any)
- [ ] Exit code recorded
- [ ] Timeout events logged as warnings

---

## Complexity Tracking

### Overall Complexity: CS-3 (Medium)

| Phase | Score | Rationale |
|-------|-------|-----------|
| Phase 1 | CS-1 | Types and interfaces only |
| Phase 2 | CS-2 | CLI execution with error handling |
| Phase 3 | CS-1 | Configuration extension |
| Phase 4 | CS-2 | Core integration with session management |
| Phase 5 | CS-1 | Documentation only |

### Risk Register

| Risk | Likelihood | Impact | Severity | Mitigation | Status |
|------|------------|--------|----------|------------|--------|
| CLI not installed | High | Medium | Medium | Clear error with installation link | Open |
| CLI version mismatch | Low | Medium | Low | Document minimum version | Open |
| Session state lost on restart | Medium | Low | Low | Document in-memory limitation | Open |
| CLI timeout | Medium | Medium | Medium | Configurable timeout (120s default) | Open |
| JSON format changes | Low | High | Medium | Abstract response parsing | Open |

**Severity Calculation**: High = (High likelihood OR High impact), Medium = (Medium likelihood AND Medium impact), Low = otherwise

---

## Progress Tracking

### Phase Status

| Phase | Status | Start | Complete | Dependencies | Notes |
|-------|--------|-------|----------|--------------|-------|
| Phase 1 | ✅ Complete | 2026-01-21 | 2026-01-21 | None | Types and interface; error codes 2001-2006 (needs CLI revision) |
| Phase 2 | Not Started | | | Phase 1 | CLI executor (replaces HTTP client) |
| Phase 3 | In Progress | 2026-01-21 | | Phase 1 | Configuration (needs CLI revision) |
| Phase 4 | Not Started | | | Phase 1, 2, 3 | Agent integration |
| Phase 5 | Not Started | | | Phase 1-4 | Documentation |

**NOTE**: Phases 1-3 were started under API approach and need revision for CLI approach. Phase 1 types may be partially reusable; Phase 2 and 3 need significant changes.

### Acceptance Criteria Mapping

| AC | Description | Phase | Task | Status |
|----|-------------|-------|------|--------|
| AC1 | Agent processes natural language via Claude CLI | 4 | 4.5 | Not Started |
| AC2 | Session continuity within conversations | 4 | 4.2, 4.5 | Not Started |
| AC3 | Flight Log captures CLI invocations | 4 | 4.7, 4.8 | Not Started |
| AC4 | Graceful handling of missing CLI | 4 | 4.6 | Not Started |
| AC5 | Graceful handling of CLI execution errors | 2 | 2.7, 2.8, 2.9 | Not Started |
| AC6 | Ping/pong unchanged | 4 | 4.4 | Not Started |
| AC7 | Configurable CLI path and timeout | 3 | 3.1, 3.2 | In Progress |
| AC8 | LLM executor interface enables testing | 1 | 1.2 | Complete (needs revision) |

---

## Change Footnotes Ledger

| ID | Date | Description | Affected Sections |
|----|------|-------------|-------------------|
| CFN-001 | 2026-01-21 | Initial plan creation (API approach) | All |
| CFN-002 | 2026-01-21 | Added TOC, ADR Ledger, verification commands | All |
| CFN-003 | 2026-01-21 | Phase 1 complete; error codes 2001-2006 | Phase 1, Progress Tracking |
| CFN-004 | 2026-01-21 | Phase 2 dossier created (API approach) | Phase 2 |
| CFN-005 | 2026-01-21 | Phase 3 dossier created (API approach) | Phase 3 |
| CFN-006 | 2026-01-21 | **CLI PIVOT**: Rewrote entire plan for CLI-based integration | All |

---

## Appendix A: Default System Prompt

```
You are a debugging assistant for the Wingmate A2A protocol. You help engineers:

1. Understand error messages and stack traces
2. Analyze logs and identify patterns
3. Explain system behavior and interactions
4. Suggest debugging steps and solutions

Be concise and technical. Focus on actionable insights.
When you don't have enough information, ask clarifying questions.
```

---

## Appendix B: CLI Reference

### Basic Invocation

```bash
# Non-interactive mode (required for programmatic use)
claude -p "What does this code do?" --output-format json
```

### With Session Resume

```bash
# Resume specific session
claude -p "And the error handling?" --resume "550e8400-e29b-41d4-a716-446655440000" --output-format json

# Continue most recent session
claude -p "What about performance?" --continue --output-format json
```

### With Model Selection

```bash
# Use specific model
claude -p "Complex analysis task" --model opus --output-format json
```

### JSON Response Format

```json
{
  "result": "The assistant's text response",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "usage": {
    "input_tokens": 150,
    "output_tokens": 523
  },
  "metadata": {
    "model": "claude-sonnet-4-20250514"
  }
}
```

### Exit Codes

| Code | Meaning | Wingmate Handling |
|------|---------|-------------------|
| 0 | Success | Parse JSON response |
| 1 | Non-blocking error | Log warning, attempt partial parsing |
| 2 | Blocking error | Return error to peer |
| 127 | Command not found | Return "CLI not installed" error |

### Installation Check

```bash
# Check if claude is in PATH
which claude || echo "Claude CLI not installed"

# Check version
claude --version

# Run diagnostics
claude doctor
```

---

<!--
MACHINE-READABLE CONTEXT
========================
```yaml
plan:
  slug: "claude-cli-integration"
  ordinal: 2
  spec_ref: "/Users/vaughanknight/GitHub/wingmate/docs/plans/002-claude-api-integration/claude-api-integration-spec.md"
  research_ref: "/Users/vaughanknight/GitHub/wingmate/docs/plans/002-claude-api-integration/research-cli-integration.md"
  created: "2026-01-21"
  status: "ready"
  mode: "full"
  pivot: "2026-01-21 - API to CLI"

  complexity:
    overall: 3
    label: "medium"
    phases:
      - phase: 1
        score: 1
        description: "Types and interface (CLI)"
        dependencies: []
      - phase: 2
        score: 2
        description: "CLI executor"
        dependencies: [1]
      - phase: 3
        score: 1
        description: "Configuration (CLI)"
        dependencies: [1]
      - phase: 4
        score: 2
        description: "Agent integration"
        dependencies: [1, 2, 3]
      - phase: 5
        score: 1
        description: "Documentation"
        dependencies: [1, 2, 3, 4]

  phases_count: 5
  tasks_count: 23
  ac_count: 8

  adr_alignment:
    - id: "ADR-001"
      status: "aligned"
      constraint: "Using os/exec standard library"
    - id: "ADR-002"
      status: "aligned"
      constraint: "JSONL format with direction field"
    - id: "ADR-003"
      status: "aligned"
      constraint: "LLM in unified Agent"

  files:
    created:
      - "/Users/vaughanknight/GitHub/wingmate/internal/llm/types.go"
      - "/Users/vaughanknight/GitHub/wingmate/internal/llm/types_test.go"
      - "/Users/vaughanknight/GitHub/wingmate/internal/llm/client.go"
      - "/Users/vaughanknight/GitHub/wingmate/internal/llm/client_test.go"
      - "/Users/vaughanknight/GitHub/wingmate/internal/llm/errors.go"
      - "/Users/vaughanknight/GitHub/wingmate/internal/llm/errors_test.go"
      - "/Users/vaughanknight/GitHub/wingmate/internal/llm/convert.go"
      - "/Users/vaughanknight/GitHub/wingmate/internal/llm/convert_test.go"
      - "/Users/vaughanknight/GitHub/wingmate/internal/llm/cli.go"
      - "/Users/vaughanknight/GitHub/wingmate/internal/llm/cli_test.go"
      - "/Users/vaughanknight/GitHub/wingmate/internal/llm/session.go"
      - "/Users/vaughanknight/GitHub/wingmate/internal/llm/session_test.go"
      - "/Users/vaughanknight/GitHub/wingmate/docs/how/llm-setup.md"
    modified:
      - "/Users/vaughanknight/GitHub/wingmate/internal/agent/config.go"
      - "/Users/vaughanknight/GitHub/wingmate/internal/agent/config_test.go"
      - "/Users/vaughanknight/GitHub/wingmate/internal/agent/agent.go"
      - "/Users/vaughanknight/GitHub/wingmate/internal/agent/agent_test.go"
      - "/Users/vaughanknight/GitHub/wingmate/README.md"
      - "/Users/vaughanknight/GitHub/wingmate/docs/project-rules/architecture.md"

  testing:
    approach: "full_tdd"
    mock_strategy: "targeted"
    naming_convention: "*_test.go (Go standard)"
    documentation_required:
      - "Why"
      - "Contract"
      - "Usage Notes"
      - "Quality Contribution"
      - "Worked Example"
    coverage_targets:
      cli_executor: 90
      session_manager: 100
      error_handling: 100
      agent_routing: 90

  principles:
    - "P1"  # Observability First
    - "P6"  # Security by Design
    - "P4"  # Protocol Compliance

  tags:
    - "llm"
    - "claude"
    - "cli-integration"
    - "tdd"
```
-->
