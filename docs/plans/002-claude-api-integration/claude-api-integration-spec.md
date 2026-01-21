# Claude CLI Integration

**Created**: 2026-01-21
**Status**: Clarified (CLI Pivot)
**Mode**: Full

> This specification incorporates findings from `research-dossier.md` and `research-cli-integration.md`
> **UPDATE 2026-01-21**: Pivoted from direct API calls to Claude CLI integration

---

## Research Context

The research dossier (completed 2026-01-21) initially explored direct Claude API integration. A subsequent pivot identified **Claude CLI** (Claude Code) as the preferred integration approach.

**CLI Research Findings** (see `research-cli-integration.md`):
- Claude CLI provides non-interactive mode via `-p` flag
- JSON output available via `--output-format json`
- Session management via `--resume` and `--continue` flags
- CLI handles authentication internally (no API key needed in Wingmate)

**Components affected**:
- `internal/agent/agent.go` - HandleMessage routing (primary integration point)
- `internal/agent/config.go` - Configuration extension for CLI settings
- `internal/protocol/errors.go` - New LLM-specific error codes
- New package: `internal/llm/` - CLI executor and session management

**Critical dependencies**:
- Claude CLI binary installed on system
- Existing `os/exec` patterns for subprocess management
- Flight Log integration for observability

**Modification risks**:
- CLI not installed on target system
- CLI version incompatibility
- Session state management complexity
- Long-running CLI execution timeouts

**Links**:
- See `research-dossier.md` for initial API analysis
- See `research-cli-integration.md` for CLI integration details

---

## Summary

**WHAT**: Enable Wingmate agents to leverage Claude's language model capabilities via the **Claude CLI** (Claude Code), allowing agents to understand and respond to natural language queries during debugging sessions.

**WHY**: Currently, Wingmate agents can only respond to predefined messages (like ping/pong). Engineers need agents that can understand context, analyze logs, explain errors, and provide intelligent debugging assistance. By integrating Claude CLI, agents transform from simple request-response systems into intelligent debugging assistants. Using CLI (rather than direct API) provides:
- No API key management in Wingmate (CLI handles its own auth)
- Session management built into CLI
- Future extensibility to other CLI tools (GitHub Copilot, etc.)

---

## Goals

1. **Intelligent message handling**: Agents can process natural language messages and provide contextually relevant responses using Claude's capabilities via CLI
2. **Session continuity**: Wingmate manages Claude session IDs to maintain conversation context across messages
3. **Observable LLM interactions**: All CLI invocations are logged to the Flight Log with execution time and response details (per Constitution P1)
4. **CLI detection**: Wingmate detects if Claude CLI is installed and provides clear error messages if not
5. **Graceful degradation**: If Claude CLI is unavailable, agents continue functioning with appropriate error responses rather than crashing
6. **Provider abstraction**: Design allows for future addition of other CLI tools (GitHub Copilot, etc.) without architectural changes

---

## Non-Goals

1. **Direct API integration**: This implementation uses Claude CLI, not direct HTTP API calls
2. **Streaming responses**: CLI operates in blocking mode; streaming is not supported
3. **API key management**: CLI handles its own authentication; Wingmate does not manage API keys
4. **Multi-provider support now**: Only Claude CLI is implemented initially; interface allows future providers
5. **Auto-installation**: If CLI is not installed, Wingmate reports error; it does not auto-install
6. **Vision / image support**: Processing images through Claude CLI is not included
7. **Tool use / function calling**: Claude's tool use feature is out of scope for this phase

---

## Complexity

**Score**: CS-3 (Medium)

**Breakdown**:
| Factor | Score | Rationale |
|--------|-------|-----------|
| **S**urface Area | 1 | Multiple files: agent.go, config.go, new internal/llm/ package (~5-6 files) |
| **I**ntegration Breadth | 1 | One external dependency (Claude CLI subprocess) |
| **D**ata & State | 1 | Session ID mapping requires state management |
| **N**ovelty & Ambiguity | 1 | CLI invocation patterns well-documented; session management straightforward |
| Non-**F**unctional | 0 | Simpler than API: no auth, no rate limit handling needed |
| **T**esting & Rollout | 1 | Requires mocking exec.Command; skip tests if CLI not installed |

**Total**: 5 points -> CS-3

**Confidence**: 0.85

Higher confidence due to comprehensive CLI research that identified clear invocation patterns, JSON response format, and session management approach.

**Assumptions**:
- Claude CLI installed on target systems
- CLI version supports `-p` and `--output-format json` flags
- JSON response format is stable
- 120-second timeout is acceptable for CLI responses

**Dependencies**:
- `claude` binary in PATH (or custom path configured)
- Go `os/exec` package for subprocess management
- Go standard library for JSON parsing

**Risks**:
- **CLI not installed**: User may not have Claude CLI installed
  - Mitigation: Clear error message with installation instructions link
- **CLI version mismatch**: Older CLI versions may not support required flags
  - Mitigation: Document minimum version; check `claude --version`
- **Long-running commands**: CLI may take extended time for complex prompts
  - Mitigation: Configurable timeout (default: 120s)

**Phases** (suggested high-level):
1. **Core LLM package**: Create `internal/llm/` with types, interface, and CLI executor
2. **Session management**: Implement conversation -> session ID mapping
3. **Configuration**: Extend agent config with CLI settings (path, timeout, model)
4. **Integration**: Modify HandleMessage to route to LLM when appropriate
5. **Observability**: Add Flight Log entries for CLI invocations/responses
6. **Error handling**: Implement LLM-specific error codes for CLI failures

---

## Acceptance Criteria

### AC1: Agent processes natural language messages via Claude CLI
**Given** a Wingmate agent with Claude CLI installed and accessible
**When** the agent receives a message with text content (not "ping")
**Then** the agent invokes Claude CLI with the message and returns Claude's response

**Observable outcome**: Agent responds with intelligent, contextual text rather than "unknown message type" error

### AC2: Session continuity within conversations
**Given** an ongoing conversation between agents
**When** the agent receives follow-up messages
**Then** the agent uses `--resume` with the stored session ID to maintain context

**Observable outcome**: Claude responses reference earlier messages in the same conversation

### AC3: Flight Log captures CLI invocations
**Given** an agent processes a message through Claude CLI
**When** the CLI execution completes
**Then** the Flight Log contains entries for:
- Outbound: CLI command, session ID (if any)
- Inbound: Response text, execution time, new session ID

**Observable outcome**: `flight.jsonl` shows CLI invocation/response pairs with timing

### AC4: Graceful handling of missing CLI
**Given** an agent without Claude CLI installed (not in PATH)
**When** the agent receives a non-ping message
**Then** the agent returns an error with code `2001` (LLMUnavailable) and clear message with installation link

**Observable outcome**: Error response indicates CLI not installed; agent does not crash

### AC5: Graceful handling of CLI execution errors
**Given** Claude CLI returns non-zero exit code
**When** the agent handles the execution result
**Then** the agent returns an appropriate error based on exit code (1=non-blocking, 2=blocking, 127=not found)

**Observable outcome**: Error response includes relevant details; Flight Log records the failure

### AC6: Existing ping/pong functionality unchanged
**Given** an agent with Claude CLI available
**When** the agent receives a "ping" message
**Then** the agent responds with "pong" (existing behavior)

**Observable outcome**: Ping messages bypass CLI routing; response time similar to pre-integration

### AC7: Configurable CLI path and timeout
**Given** environment variables `WINGMATE_CLAUDE_CLI_PATH` and `WINGMATE_CLAUDE_TIMEOUT`
**When** the agent invokes Claude CLI
**Then** the agent uses the configured binary path and timeout

**Observable outcome**: Custom CLI path is used; timeout is respected

### AC8: LLM executor interface enables testing
**Given** the LLM implementation uses an interface
**When** tests need to mock CLI responses
**Then** tests can inject a mock executor without subprocess calls

**Observable outcome**: Agent tests run without invoking Claude CLI; mock returns controlled responses

---

## Risks & Assumptions

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| CLI not installed | High | Medium | Clear error message with installation link |
| CLI version mismatch | Low | Medium | Document minimum version; check at startup |
| Session state lost on restart | Medium | Low | Accept limitation; document in-memory-only sessions |
| CLI execution timeout | Medium | Medium | Configurable timeout (default 120s); clear error messages |
| JSON response format change | Low | High | Version check; abstract response parsing |

### Assumptions

1. Users have Claude CLI installed (can be checked with `which claude` or `command -v claude`)
2. Claude CLI supports `-p`, `--output-format json`, `--resume` flags
3. CLI execution time is acceptable (2-60 seconds typical)
4. Session IDs returned by CLI are UUIDs
5. Single-message-at-a-time processing is acceptable (Go handles concurrency)
6. In-memory session storage is acceptable (lost on restart)

---

## Open Questions

*All questions resolved in clarification session 2026-01-21. See [Clarifications](#clarifications) section.*

---

## Testing Strategy

**Approach**: Full TDD

**Rationale**: CLI integration requires comprehensive test coverage. Project rules mandate TDD for non-trivial changes. The `internal/llm/` package needs thorough unit tests with mocked exec.Command.

**Focus Areas**:
- CLI executor (command building, response parsing)
- Session manager (ID storage, retrieval)
- Error handling (exit codes, timeout, not found)
- Agent routing logic (ping bypass, LLM dispatch)
- Configuration loading (environment variables, defaults)

**Excluded**:
- Live Claude CLI calls in CI (requires CLI installed)
- Performance benchmarking (future enhancement)

**Mock Usage**: Targeted mocks

- Mock `exec.Command` for all CLI executor tests
- Use real code for all other components (config, routing, Flight Log, session management)
- Integration tests with `t.Skip` pattern for CI environments without CLI

**Test Markers**:
```go
func TestCLI_Integration(t *testing.T) {
    if os.Getenv("WINGMATE_TEST_CLI") == "" {
        t.Skip("Skipping CLI integration test - set WINGMATE_TEST_CLI=1")
    }
    // ... actual CLI tests
}
```

---

## Documentation Strategy

**Location**: Hybrid (README + docs/how/)

**Rationale**: Claude CLI integration is a major user-facing feature that requires both quick-start instructions and detailed guidance.

**Content Split**:
- **README.md**: Add "LLM Support" section with:
  - Quick setup (install Claude CLI, configure Wingmate)
  - Basic usage example
  - Link to detailed guide
- **docs/how/llm-setup.md**: Detailed guide with:
  - Claude CLI installation instructions (link to official docs)
  - Wingmate environment variables explained
  - Model selection guidance (via CLI --model flag)
  - System prompt customization
  - Troubleshooting (CLI not found, timeout, errors)

**Target Audience**: Engineers setting up Wingmate for debugging workflows

**Maintenance**: Update docs when new CLI features are added or environment variables change

---

## ADR Seeds (Optional)

### ADR Seed: CLI vs API Integration

**Decision Drivers**:
- API key security (API approach requires key management)
- Session management (API requires manual implementation; CLI provides it)
- Future extensibility (CLI approach generalizes to other tools)
- Installation dependency (CLI requires user to install; API doesn't)

**Candidate Alternatives**:
- A: Direct HTTP API calls (requires API key, manual session management)
- B: Claude CLI invocation (no API key, built-in sessions, extensible)

**Decision**: Option B (CLI) chosen for simplicity, security, and extensibility

**Stakeholders**: Core maintainers

### ADR Seed: LLM Provider Abstraction

**Decision Drivers**:
- Future support for other CLI tools (GitHub Copilot, etc.)
- Testing requires mock implementations
- Provider-specific error handling

**Candidate Alternatives**:
- A: Claude CLI executor only (simplest, but locked in)
- B: Interface with single implementation (balance flexibility/complexity)
- C: Full plugin architecture (over-engineered for current needs)

**Decision**: Option B chosen for testability and future extensibility

**Stakeholders**: Core maintainers, future contributors

---

## Clarifications

### Session 2026-01-21 (Initial)

**Q1: Workflow Mode**
- **Answer**: Full
- **Rationale**: CS-3 complexity with external dependency, new package, 5+ phases. Full mode provides proper gates.

**Q2: Testing Strategy**
- **Answer**: Full TDD
- **Rationale**: Project rules require TDD for non-trivial changes. CLI integration needs comprehensive coverage.

**Q3: Mock Usage**
- **Answer**: Targeted mocks (exec.Command only)
- **Rationale**: Mock subprocess execution to avoid requiring CLI in CI. Real code for everything else.

**Q4: Documentation Strategy**
- **Answer**: Hybrid (README + docs/how/)
- **Rationale**: Major user-facing feature needs both quick-start and detailed guide.

### Session 2026-01-21 (CLI Pivot)

**Q5: API vs CLI Integration**
- **Answer**: CLI (Claude Code)
- **Rationale**: CLI handles auth internally, provides session management, enables future support for other CLI tools.

**Q6: API Key Handling**
- **Answer**: Not needed - CLI manages its own authentication
- **Rationale**: Simplifies Wingmate config; user authenticates CLI separately.

**Q7: Session Management**
- **Answer**: Wingmate manages session ID mapping (conversation_id -> claude_session_id)
- **Rationale**: Enables multi-turn conversations; session IDs come from CLI JSON response.

**Q8: CLI Not Installed**
- **Answer**: Return error with installation instructions link
- **Rationale**: Don't auto-install; be explicit about dependency.

**Q9: Future CLI Providers**
- **Answer**: Design interface to support other CLIs (GitHub Copilot, etc.)
- **Rationale**: Abstract executor interface allows future provider additions.

---

<!--
MACHINE-READABLE CONTEXT
========================
```yaml
spec:
  slug: "claude-cli-integration"
  ordinal: 2
  created: "2026-01-21"
  status: "clarified"
  pivot: "2026-01-21 - API to CLI"

  summary: "Enable Wingmate agents to use Claude CLI for intelligent message processing"

  complexity:
    score: 3
    label: "medium"
    factors:
      S: 1  # Multiple files
      I: 1  # One external (Claude CLI)
      D: 1  # Session state management
      N: 1  # Some ambiguity
      F: 0  # Simpler NFRs (no auth)
      T: 1  # Mock exec.Command
    confidence: 0.85

  goals_count: 6
  non_goals_count: 7
  acceptance_criteria_count: 8
  open_questions_count: 0  # All resolved in session 2026-01-21

  research:
    dossier_exists: true
    cli_research_exists: true
    external_research_exists: false

  affects:
    - component: "internal/agent"
      impact: "high"
    - component: "internal/llm"
      impact: "high (new)"
    - component: "internal/protocol/errors"
      impact: "low"
    - component: "config"
      impact: "medium"

  principles:
    - "P1"  # Observability First
    - "P6"  # Security by Design

  tags:
    - "llm"
    - "claude"
    - "cli-integration"
```
-->
