# MCP Server Integration

**Created**: 2026-01-22
**Status**: Clarified
**Mode**: Full

📚 This specification incorporates findings from research-dossier.md and external-research/

---

## Research Context

This specification builds on comprehensive codebase research conducted via 7 parallel subagents (65+ findings).

**Components Affected**:
- `cmd/wingmate/main.go` - Add `mcp` subcommand
- `internal/mcp/` (NEW) - MCP server implementation
- `internal/agent/` - Integration point for LLM access
- Configuration system - MCPConfig extension

**Critical Dependencies**:
- `internal/llm/LLMExecutor` - Primary capability to expose
- `internal/llm/SessionManager` - Conversation continuity
- `internal/flightlog/` - Observability (Constitution P1)
- Claude CLI - Underlying LLM execution

**Modification Risks**:
- MCP SDK maturity unknown (may need custom implementation)
- stdio transport differs from existing HTTP patterns
- Session state management in stdio context

**Link**: See `research-dossier.md` for full analysis (65+ findings across 7 domains)

---

## Summary

**What**: Add a Model Context Protocol (MCP) server to Wingmate, enabling Claude Code and other MCP-compatible clients to invoke Wingmate's LLM capabilities through a simple, standardized interface.

**Why**: Currently, using Wingmate from Claude Code requires understanding the A2A protocol, configuring HTTP ports, and manual setup. MCP provides a standardized way for LLM tools to communicate, allowing users to simply add Wingmate as an MCP server and immediately access its chat capabilities without understanding the underlying infrastructure.

**User Value**: Users can invoke `wingmate_chat` directly from Claude Code (or any MCP client) with a single tool call, getting LLM responses with conversation continuity - no port configuration, no HTTP setup, no protocol knowledge required.

---

## Goals

- **G1**: Enable Claude Code to invoke Wingmate's LLM capabilities via MCP tools
- **G2**: Provide zero-configuration experience for end users (just add MCP server, start using)
- **G3**: Expose `wingmate_chat` as the primary tool with conversation continuity via session IDs
- **G4**: Maintain observability through Flight Log integration (Constitution P1)
- **G5**: Follow existing Wingmate patterns (error codes, graceful shutdown, configuration)
- **G6**: Support session continuity across multiple tool invocations within a conversation

---

## Non-Goals

- **NG1**: Replacing A2A protocol - MCP complements A2A for local tool access, doesn't replace peer-to-peer communication
- **NG2**: Exposing all Wingmate functionality - Focus on core `wingmate_chat`; additional tools (status, discover, send) are optional/future
- **NG3**: HTTP transport for MCP - stdio is the standard for local MCP servers
- **NG4**: Authentication/authorization - Local MCP servers typically don't require auth
- **NG5**: Multi-tenant support - Single user, single machine context
- **NG6**: GUI or configuration wizard - CLI-only, following Wingmate's existing patterns
- **NG7**: Backward compatibility with pre-MCP versions - New feature, no migration needed

---

## Complexity

**Score**: CS-3 (medium)

**Breakdown**:
| Factor | Score | Rationale |
|--------|-------|-----------|
| Surface Area (S) | 1 | New package (internal/mcp/) + CLI addition, but well-bounded |
| Integration (I) | 1 | One external: MCP protocol/SDK; internal deps are stable |
| Data/State (D) | 0 | No schema changes; reuses existing SessionManager |
| Novelty (N) | 2 | MCP protocol is new to codebase; some discovery needed |
| Non-Functional (F) | 1 | Moderate: observability required (P1), audit logging (P6) |
| Testing/Rollout (T) | 1 | Integration tests needed for stdio transport |

**Total**: S(1) + I(1) + D(0) + N(2) + F(1) + T(1) = **6 points → CS-3**

**Confidence**: 0.90
- High confidence in scope, integration points, and protocol details (external research complete)
- Official SDK evaluation complete, Claude Code config documented

**Assumptions**:
- Official Go MCP SDK (modelcontextprotocol/go-sdk) is stable and usable ✅ Confirmed
- MCP stdio transport follows newline-delimited JSON-RPC ✅ Confirmed
- Claude Code MCP configuration via ~/.claude.json ✅ Confirmed
- No major protocol changes imminent

**Dependencies**:
- MCP protocol specification (external)
- Go MCP SDK evaluation (external)
- Claude CLI must be installed for LLM functionality

**Risks**:
- MCP SDK immaturity may require custom implementation (Medium)
- stdio transport testing patterns differ from HTTP (Low)
- Protocol versioning may require future updates (Low)

**Phases** (suggested):
1. Core MCP types and server interface
2. Tool implementation (wingmate_chat)
3. CLI integration and configuration
4. Documentation and testing
5. Optional: Additional tools (status, discover)

---

## Acceptance Criteria

### AC1: MCP Server Starts Successfully
**Given** the user runs `wingmate mcp`
**When** the server initializes
**Then** it:
- Starts listening on stdio
- Logs startup to Flight Log
- Responds to MCP `initialize` request with capabilities
- Lists `wingmate_chat` tool on `tools/list` request

### AC2: Tool Invocation Works
**Given** the MCP server is running
**When** a client invokes `wingmate_chat` with a prompt
**Then** the tool:
- Passes the prompt to LLMExecutor
- Returns the response with result, session_id, model, and duration
- Logs the invocation to Flight Log

### AC3: Session Continuity
**Given** a previous `wingmate_chat` invocation returned a session_id
**When** a subsequent invocation includes that session_id
**Then** the conversation continues with context preserved

### AC4: Error Handling
**Given** Claude CLI is not installed or available
**When** a client invokes `wingmate_chat`
**Then** the tool returns error code 3001 with message "Claude CLI not available"

### AC5: Graceful Shutdown
**Given** the MCP server is running with in-flight requests
**When** the server receives SIGINT/SIGTERM
**Then** it:
- Completes in-flight requests
- Logs shutdown to Flight Log
- Exits cleanly

### AC6: Claude Code Integration
**Given** the user configures Claude Code with Wingmate MCP server
**When** Claude Code starts
**Then** the `wingmate_chat` tool appears in available tools

### AC7: Configuration
**Given** environment variables WINGMATE_LLM_MODEL and WINGMATE_LLM_TIMEOUT are set
**When** the MCP server starts
**Then** those settings are applied to LLM invocations

---

## Risks & Assumptions

### Risks

| Risk | Severity | Probability | Mitigation |
|------|----------|-------------|------------|
| MCP SDK is immature/buggy | Medium | Medium | Evaluate SDK; fallback to custom implementation from spec |
| stdio blocking causes hangs | Medium | Low | Use goroutines, context cancellation, documented patterns |
| Session state lost on restart | Low | Certain | Document limitation; sessions are in-memory only |
| MCP protocol version changes | Low | Low | Target current spec; version capabilities |
| Testing stdio is complex | Medium | Medium | Use io.Pipe patterns; adapt from existing tests |

### Assumptions

1. **A1**: MCP protocol is stable (no breaking changes expected)
2. **A2**: Go MCP library exists and is usable, OR implementing from spec is tractable
3. **A3**: Claude Code's MCP configuration supports local binary servers
4. **A4**: Users have Claude CLI installed (prerequisite documented)
5. **A5**: stdio transport is sufficient (HTTP transport not needed)
6. **A6**: Single-tool focus (wingmate_chat) is sufficient for MVP

---

## Open Questions

### Q1: SDK vs Custom Implementation ✅ RESOLVED
**Decision**: Use official `modelcontextprotocol/go-sdk`

**Rationale**:
- Zero external dependencies in core package (matches Wingmate philosophy)
- Type-safe API with automatic JSON Schema generation
- v1.1.0 stable release, backed by Google/MCP team
- 50-100x less effort than implementing from spec

### Q2: Additional Tools Scope ✅ RESOLVED
**Decision**: Full A2A bridge (Option C)

**Tools to implement**:
- `wingmate_chat` - Primary LLM invocation (P0)
- `wingmate_status` - Health monitoring (P1)
- `wingmate_discover` - List peer agents (P1)
- `wingmate_send` - Send A2A message to peer (P2)

### Q3: Configuration Location ✅ RESOLVED
**Decision**: Both with precedence (Option C)

**Implementation**:
- Config file provides defaults (location TBD in architecture)
- Environment variables override config file values
- Existing WINGMATE_LLM_* vars continue to work
- Claude Code passes env vars in MCP server config

### Q4: Flight Log Entry Format ✅ RESOLVED (Default)
**Decision**: Follow existing patterns

**Implementation**:
- Consistent with existing LLM entries in flightlog
- Include: tool name, input summary (truncated), output summary, duration, session_id
- Security: Truncate prompts >500 chars in logs, full content only in debug mode

---

## Testing Strategy

**Approach**: Hybrid (TDD where it makes sense)

**Rationale**: MCP protocol handling and server lifecycle are complex and benefit from TDD. CLI integration and config loading are simpler and need only basic validation tests.

**Focus Areas (TDD)**:
- MCP server initialization handshake
- Tool registration and invocation (wingmate_chat, wingmate_status, etc.)
- Error handling (protocol errors vs tool execution errors)
- Session continuity across tool calls
- Graceful shutdown with in-flight requests

**Lightweight Testing**:
- CLI `mcp` subcommand parsing
- Configuration loading and precedence
- Flight Log entry format

**Mock Usage**: Targeted mocks only
- Mock: Claude CLI (external process), stdio transport (for unit tests)
- Real: LLMExecutor interface, SessionManager, FlightLog, internal components
- Use `io.Pipe` for stdio testing patterns

**Excluded from extensive testing**:
- Claude Code integration (manual verification documented)
- MCP protocol compliance (trust official SDK)

---

## Documentation Strategy

**Location**: Hybrid (README + docs/how/)

**Rationale**: Users need quick-start in README to get running, plus detailed guide for troubleshooting and advanced config.

**Content Split**:
| Location | Content |
|----------|---------|
| README.md | Quick-start: install, add to Claude Code, verify works |
| docs/how/mcp-setup.md | Detailed: full config options, troubleshooting, all tools |

**Target Audience**:
- README: Users who want to quickly add Wingmate to Claude Code
- docs/how/: Users debugging issues or configuring advanced options

**Maintenance**:
- Update README when major features change
- Update docs/how/ when config options or tools change
- Include verification commands in docs

---

## ADR Seeds (Optional)

**Decision Title**: MCP Server Implementation Approach

**Decision Drivers**:
- Constitution P1: All operations must be observable (Flight Log)
- Constitution P4: MCP sanctioned for local tool access
- Constitution P6: Audit logging required
- ADR-001: Must remain single Go binary
- Current zero-dependency philosophy

**Candidate Alternatives**:
- **A) Use existing Go MCP SDK** (e.g., mark3labs/mcp-go) - Faster development, potential dependency risk
- **B) Implement MCP protocol from spec** - No new deps, more implementation work
- **C) Hybrid** - Use SDK for transport, custom tool handlers - Balance of both

**Stakeholders**:
- Users wanting simple Claude Code integration
- Developers maintaining the codebase
- Future contributors extending MCP tools

---

## External Research ✅ COMPLETE

**Incorporated**: All 3 research topics from research-dossier.md

**Files**:
- `external-research/01-mcp-protocol-specification-results.md`
- `external-research/02-go-mcp-libraries-results.md`
- `external-research/03-claude-code-mcp-configuration-results.md`

**Key Findings Applied**:

| Topic | Finding | Applied To |
|-------|---------|------------|
| MCP Protocol | Newline-delimited JSON, stdio transport, initialize→tools/list→tools/call lifecycle | AC1, AC2, Implementation |
| Go Libraries | Official SDK (modelcontextprotocol/go-sdk) has zero deps, v1.1.0 stable | Q1 Decision, ADR Seeds |
| Claude Code Config | ~/.claude.json with mcpServers object, stdio type | AC6, Documentation Strategy |

**Impact**:
- Q1 (SDK vs Custom) fully resolved → Official SDK
- AC6 (Claude Code Integration) now documentable with exact config format
- Complexity confidence increased from 0.75 to 0.90

---

## Clarifications

### Session 2026-01-22

| # | Question | Answer | Updated Section |
|---|----------|--------|-----------------|
| Q1 | Workflow Mode | **Full** - CS-3 feature needs multi-phase plan | Mode header |
| Q2 | Testing Strategy | **Hybrid** - TDD for MCP protocol, lightweight for CLI | Testing Strategy |
| Q3 | Mock Policy | **Targeted mocks** - Mock external systems only | Testing Strategy |
| Q4 | Documentation | **Hybrid** - README quick-start + docs/how/ detailed | Documentation Strategy |
| Q5 | Tools Scope | **Full A2A bridge** - chat, status, discover, send | Open Questions Q2 |
| Q6 | Configuration | **Both with precedence** - file defaults, env overrides | Open Questions Q3 |
| Q7 | SDK Choice | **Official SDK** - modelcontextprotocol/go-sdk | Open Questions Q1 |

---

## Document Metadata

| Field | Value |
|-------|-------|
| Spec Version | 1.1 |
| Created | 2026-01-22 |
| Clarified | 2026-01-22 |
| Author | Claude (plan-1b-specify, plan-2-clarify) |
| Research Source | research-dossier.md, external-research/*.md |
| Plan Folder | docs/plans/003-mcp-server-integration/ |
| Next Step | /plan-3-architect |
