# MCP HTTP Transport Migration

**Mode**: Full

ℹ️ This specification incorporates findings from the `/plan-1a-explore` research session conducted in the current conversation, including MCP Streamable HTTP specification research and Claude Code transport capability verification.

---

## Research Context

**Source**: Conversation-based exploration with Perplexity research and codebase analysis

**Key Findings**:
- **MCP Streamable HTTP** (protocol version 2025-03-26) is the current standard, replacing deprecated HTTP+SSE transport
- **Single endpoint model**: `/mcp` supporting both POST (requests) and optional GET (SSE streaming)
- **Claude Code native support**: Confirmed via `claude mcp add --help` showing `--transport http` option
- **Current implementation**: `internal/mcp/` uses stdio transport over stdin/stdout (56 lines in `transport.go`)
- **Architecture gap**: `wingmate mcp` (stdio) runs as isolated process, cannot communicate with running `wingmate --port` A2A agent

**Components Affected**:
- `internal/mcp/transport.go` - Complete replacement (stdio → HTTP)
- `internal/mcp/server.go` - Transport abstraction changes
- `internal/agent/agent.go` - Add `/mcp` HTTP route
- `cmd/wingmate/main.go` - Remove `runMCP()` stdio mode

**Critical Dependencies**:
- Existing HTTP server infrastructure in `internal/agent/`
- MCP protocol handlers already implemented (`handleInitialize`, `handleToolsList`, `handleToolsCall`)
- Flight Log integration already working

**Modification Risks**:
- Breaking change for existing stdio users (accepted per user direction)
- Session state management across HTTP requests
- Concurrent request handling (multiple Claude Code connections)

---

## Summary

**WHAT**: Replace Wingmate's MCP server stdio transport with HTTP transport, integrating the MCP endpoint directly into the running Wingmate agent HTTP server.

**WHY**:
1. **Unified process architecture** - A single `wingmate --port 9000` process serves both A2A peers and MCP clients (Claude Code), eliminating the isolated `wingmate mcp` subprocess problem
2. **Real peer discovery** - The `wingmate_discover` tool can now return actual known peers since it shares state with the A2A agent
3. **Shared observability** - Single Flight Log captures both A2A and MCP activity
4. **Simplified deployment** - One process to manage, one port to configure
5. **Protocol compliance** - Adopts MCP Streamable HTTP spec (2025-03-26), the current standard

---

## Goals

1. **Single process operation** - Claude Code connects to the same `wingmate` process that handles A2A peer communication
2. **Native Claude Code integration** - Users configure with `claude mcp add --transport http wingmate http://localhost:PORT/mcp`
3. **Complete stdio removal** - All stdio-based MCP code is deleted, not deprecated
4. **Localhost security** - MCP endpoint rejects non-localhost requests (application-level check on `/mcp` route; server still listens on all interfaces for A2A)
5. **Preserved tool functionality** - All three MCP tools (`wingmate_chat`, `wingmate_status`, `wingmate_discover`) continue working
6. **Enhanced discovery** - `wingmate_discover` returns actual peer agents known to the A2A layer
7. **Session continuity** - Chat sessions persist across multiple HTTP requests via session ID
8. **Observability maintained** - All MCP tool invocations logged to Flight Log per Constitution P1

---

## Non-Goals

1. **Migration path for stdio users** - No backward compatibility; stdio is removed entirely
2. **Remote MCP access** - MCP endpoint rejects non-localhost requests at application level; A2A protocol handles remote communication
3. **Authentication for MCP** - Localhost binding provides sufficient security; no OAuth/tokens for MCP
4. **SSE streaming responses** - Initial implementation uses simple request/response; streaming can be added later
5. **Multiple MCP endpoints** - Single `/mcp` endpoint per agent instance
6. **MCP client implementation** - Wingmate is an MCP server only, not a client
7. **A2A security changes** - A2A protocol security is a separate concern, unchanged by this feature

---

## Complexity

**Score**: CS-3 (medium)

**Breakdown**:
| Dimension | Score | Rationale |
|-----------|-------|-----------|
| Surface Area (S) | 1 | Multiple files: transport.go (replace), server.go (modify), agent.go (add route), main.go (remove runMCP) |
| Integration (I) | 1 | Integrates with existing HTTP server infrastructure; no new external dependencies |
| Data/State (D) | 1 | Session state management across HTTP requests; no schema changes |
| Novelty (N) | 1 | MCP HTTP spec is documented; Claude Code support verified; some implementation details to work out |
| Non-Functional (F) | 1 | Localhost-only security requirement; concurrent request handling |
| Testing/Rollout (T) | 1 | Integration tests needed; existing MCP tests need adaptation |

**Total**: S(1) + I(1) + D(1) + N(1) + F(1) + T(1) = **6** → **CS-3**

**Confidence**: 0.85

**Assumptions**:
- MCP protocol handlers (`handleInitialize`, `handleToolsList`, `handleToolsCall`) can be reused with minimal changes
- Existing HTTP server in `internal/agent/` supports adding new routes
- Claude Code's HTTP transport follows MCP Streamable HTTP spec
- Session manager can be shared between A2A and MCP layers

**Dependencies**:
- No new external Go dependencies required
- Claude Code must be updated to use HTTP transport configuration

**Risks**:
- Concurrent request handling may expose race conditions in shared state
- MCP protocol version negotiation details not fully explored
- Session cleanup uses 30-min idle TTL (defined in clarifications)

**Phases**:
1. **HTTP Transport Layer** - Create HTTP-based transport replacing stdio
2. **Agent Integration** - Add `/mcp` route to agent HTTP server
3. **Stdio Removal** - Delete all stdio MCP code and `wingmate mcp` command
4. **Documentation** - Update setup guides and CLAUDE.md

---

## Acceptance Criteria

1. **AC-1: HTTP endpoint available**
   - GIVEN a running Wingmate agent (`wingmate --port 9000`)
   - WHEN a client sends POST to `http://localhost:9000/mcp` with MCP initialize request
   - THEN the server responds with valid MCP initialize response including server info and capabilities

2. **AC-2: Claude Code can connect**
   - GIVEN `claude mcp add --transport http wingmate http://localhost:9000/mcp`
   - WHEN Claude Code lists available tools
   - THEN `wingmate_chat`, `wingmate_status`, and `wingmate_discover` are listed

3. **AC-3: Tool invocation works**
   - GIVEN Claude Code connected to Wingmate via HTTP
   - WHEN user invokes `wingmate_chat` with a prompt
   - THEN the response contains LLM output and session ID

4. **AC-4: Discovery returns peers**
   - GIVEN Wingmate has known peers from A2A layer
   - WHEN `wingmate_discover` is invoked via MCP
   - THEN the response includes the list of known peer agents

5. **AC-5: Localhost-only MCP access**
   - GIVEN Wingmate started with `--port 9000` (listening on all interfaces)
   - WHEN attempting to access `/mcp` endpoint from remote host
   - THEN server responds with 403 Forbidden (application-level check rejects non-localhost)
   - AND A2A endpoints remain accessible from remote hosts

6. **AC-6: stdio mode removed**
   - GIVEN the updated Wingmate binary
   - WHEN user runs `wingmate mcp`
   - THEN error message indicates MCP is now accessed via HTTP endpoint

7. **AC-7: Session continuity**
   - GIVEN a `wingmate_chat` call returns session_id "sess-123"
   - WHEN subsequent `wingmate_chat` call includes session_id "sess-123"
   - THEN conversation context is maintained

8. **AC-8: Flight Log integration**
   - GIVEN MCP tool invocation via HTTP
   - WHEN tool completes
   - THEN Flight Log contains entry with tool name, input summary, duration, and outcome

9. **AC-9: Concurrent requests handled**
   - GIVEN multiple Claude Code instances connected to same Wingmate
   - WHEN both invoke tools simultaneously
   - THEN both receive correct responses without interference

10. **AC-10: Graceful error handling**
    - GIVEN malformed MCP request
    - WHEN sent to `/mcp` endpoint
    - THEN server responds with appropriate JSON-RPC error (not crash)

---

## Risks & Assumptions

### Risks

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| Race conditions in shared state | Medium | Medium | Use existing sync primitives; add mutex protection to MCP handlers |
| MCP protocol version mismatch | Low | Low | Start with 2024-11-05; inspect client version during negotiate |
| Session memory growth | Medium | Low | 30-minute idle TTL with background cleanup goroutine |
| Breaking existing users | High | High | Accepted per user direction; clear documentation |

### Assumptions

1. Claude Code's `--transport http` follows standard MCP Streamable HTTP spec
2. Existing MCP protocol handlers are transport-agnostic (only I/O changes)
3. Single `/mcp` endpoint can serve multiple concurrent clients
4. Session IDs from MCP layer can reuse existing session management
5. Application-level localhost check is sufficient security for MCP (no auth tokens needed)
6. No need for SSE streaming in initial implementation (synchronous responses)

---

## Open Questions

*All questions resolved in clarification session 2026-01-23. See ## Clarifications section.*

---

## Testing Strategy

**Approach**: Hybrid (Full TDD where it makes sense)

**Rationale**: HTTP transport layer and localhost security check warrant full TDD. Protocol handlers are already tested; adapt existing tests. Integration tests for end-to-end Claude Code connectivity.

**Focus Areas** (Full TDD):
- HTTP transport layer (`internal/mcp/http_transport.go`) - new code, complex
- Localhost-only access check middleware - security-critical
- Concurrent request handling - race condition potential
- Session management across HTTP requests - state management

**Lighter Testing**:
- Protocol handler integration (already tested, minimal changes)
- stdio removal (deletion, no new logic)
- Documentation updates (manual verification)

**Mock Usage**: Targeted mocks
- Mock `LLMExecutor` for tool handler tests (existing pattern)
- Mock `http.ResponseWriter` for transport tests
- Mock peer registry for `wingmate_discover` tests
- Real HTTP server for integration tests (no mocking HTTP stack)

**Test Categories**:
| Category | Approach | Files |
|----------|----------|-------|
| HTTP Transport | Full TDD | `http_transport_test.go` |
| Localhost Check | Full TDD | `middleware_test.go` |
| Session State | Full TDD | `session_test.go` |
| Integration | TAD | `mcp_integration_test.go` |
| Handler Adaption | Adapt existing | `handlers_test.go` |

---

## Documentation Strategy

**Location**: Hybrid (README.md + docs/how/)

**Rationale**: Users need quick-start in README for basic setup; detailed troubleshooting and architecture in docs/how/. Existing `docs/how/mcp-setup.md` already exists and needs updating.

**Content Split**:

| Content | Location | Reason |
|---------|----------|--------|
| Quick setup command | README.md | First thing users need |
| `claude mcp add` example | README.md | Copy-paste ready |
| Detailed configuration | docs/how/mcp-setup.md | Reference material |
| Troubleshooting | docs/how/mcp-setup.md | Edge cases, errors |
| Architecture overview | docs/how/mcp-setup.md | Understanding internals |
| Migration from stdio | docs/how/mcp-setup.md | Breaking change guidance |

**Target Audience**:
- Primary: Developers using Claude Code with local Wingmate
- Secondary: Contributors extending MCP tools

**CLAUDE.md Updates**:
- Update MCP section to reflect HTTP transport
- Remove stdio references
- Update configuration examples

**Maintenance**: Update docs when MCP tools or configuration changes.

---

## ADR Seeds (Optional)

**Decision Drivers**:
- Single process architecture requirement (eliminate isolated MCP subprocess)
- Claude Code native HTTP transport support
- Localhost-only security model for MCP
- Constitution P1 observability requirement

**Candidate Alternatives**:
- A: HTTP endpoint on same port as A2A (proposed) - simplest, single configuration
- B: HTTP endpoint on separate MCP-only port - isolation but more config
- C: Unix socket for MCP - stronger security but less portable

**Stakeholders**:
- Users running Wingmate locally with Claude Code
- Developers extending Wingmate's MCP tools

---

## Document Metadata

| Field | Value |
|-------|-------|
| Created | 2026-01-23 |
| Author | Claude (plan-1b-specify) |
| Plan | 004-mcp-http-transport |
| Status | Draft |
| Branch | main |
| Path | docs/plans/004-mcp-http-transport/mcp-http-transport-spec.md |
| Next | /plan-3-architect |

---

## Clarifications

### Session 2026-01-23

**Q1: Workflow Mode**
| Option | Mode | Selected |
|--------|------|----------|
| A | Simple | |
| B | Full | ✓ |

**Answer**: B (Full)
**Rationale**: CS-3 feature with multiple phases, security considerations, and comprehensive testing needs.

---

**Q2: Testing Strategy**
| Option | Approach | Selected |
|--------|----------|----------|
| A | Full TDD | |
| B | TAD | |
| C | Lightweight | |
| D | Manual Only | |
| E | Hybrid | ✓ |

**Answer**: E (Hybrid)
**Rationale**: "Full TDD where it makes sense" - HTTP transport and security checks need TDD; existing handlers adapt with lighter touch.

---

**Q3: Mock Usage**
| Option | Policy | Selected |
|--------|--------|----------|
| A | Avoid mocks entirely | |
| B | Allow targeted mocks | ✓ |
| C | Allow liberal mocking | |

**Answer**: B (Targeted mocks)
**Rationale**: "Targeted mocks" - Mock external dependencies (LLM executor) and slow components; use real HTTP for integration tests.

---

**Q4: Documentation Strategy**
| Option | Location | Selected |
|--------|----------|----------|
| A | README.md only | |
| B | docs/how/ only | |
| C | Hybrid | ✓ |
| D | No new documentation | |

**Answer**: C (Hybrid)
**Rationale**: "Document in readme and how to, you decide the split" - Quick-start in README, details in docs/how/mcp-setup.md.
**Split**: README gets setup command + example; docs/how/ gets configuration, troubleshooting, migration guide.

---

**Q5: Port Configuration** (from Open Questions)
**Question**: Should `/mcp` be on the same port as A2A endpoints?
**Answer**: Single port
**Rationale**: User confirmed "it's a single port" - simplest configuration, single endpoint.
**Impact**: Updated architecture to use existing agent HTTP server, no separate listener.

---

**Q6: MCP Protocol Version** (from Open Questions)
**Question**: What MCP protocol version should we report?
**Answer**: Explore at implementation time
**Rationale**: User said "we can explore this" - Claude Code 2.1.15 supports Streamable HTTP. Check client's requested version during initialize handshake.
**Recommendation**: Start with "2024-11-05" for compatibility; server can inspect client's requested version and respond accordingly. Log version negotiation for observability.

---

**Q7: Discovery Response** (from Open Questions)
**Question**: Should `wingmate_discover` return the local agent's info as well as peers?
**Answer**: Peers only, not self
**Rationale**: User clarified "invoked from the local agent, so it shouldn't call itself ever" - discovery returns known remote peers, excluding the local agent.
**Impact**: Updated tool behavior - `wingmate_discover` returns `[]` if no peers known, never includes self.

---

**Q8: Session Cleanup** (from Open Questions)
**Question**: How should session cleanup work for HTTP-based sessions?
**Answer**: TTL-based idle timeout (30 minutes recommended)
**Rationale**: User deferred to research/recommendation. MCP spec recommends "short expiration (e.g., 10 minutes)" for security tokens. For chat sessions with longer interactions, 30-minute idle timeout balances usability and resource cleanup.
**Implementation**:
- Sessions expire after 30 minutes of inactivity
- Each request with session_id refreshes the TTL
- Background goroutine periodically cleans expired sessions
- Expired session returns error prompting new session creation

---

### Coverage Summary

| Category | Status | Count |
|----------|--------|-------|
| Resolved | ✓ | 8 |
| Deferred | - | 0 |
| Outstanding | - | 0 |

All critical ambiguities resolved. Spec ready for `/plan-3-architect`.
