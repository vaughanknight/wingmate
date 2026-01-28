# Peer Delegation & Agent Specialization Implementation Plan

**Plan Version**: 1.0.0
**Created**: 2026-01-28
**Spec**: [./peer-delegation-spec.md](./peer-delegation-spec.md)
**Research Dossier**: [./research-dossier.md](./research-dossier.md)
**Status**: READY
**Mode**: Full

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Technical Context](#technical-context)
3. [Critical Research Findings](#critical-research-findings)
4. [Testing Philosophy](#testing-philosophy)
5. [Implementation Phases](#implementation-phases)
   - [Phase 1: Config, Types & Purpose Plumbing](#phase-1-config-types--purpose-plumbing)
   - [Phase 2: Peer Bootstrap & Discovery Enrichment](#phase-2-peer-bootstrap--discovery-enrichment)
   - [Phase 3: Delegation Tool (wingmate_ask)](#phase-3-delegation-tool-wingmate_ask)
   - [Phase 4: System Prompt & Specialization](#phase-4-system-prompt--specialization)
   - [Phase 5: Documentation](#phase-5-documentation)
6. [Cross-Cutting Concerns](#cross-cutting-concerns)
7. [Complexity Tracking](#complexity-tracking)
8. [Progress Tracking](#progress-tracking)
9. [Deviation Ledger](#deviation-ledger)
10. [ADR Ledger](#adr-ledger)
11. [Change Footnotes Ledger](#change-footnotes-ledger)

---

## Executive Summary

Wingmate's MCP tools currently execute locally -- `wingmate_chat` runs Claude CLI on the connected agent, `wingmate_discover` returns an empty peer list, and no tool exists to send messages to remote peers. This plan adds peer delegation: the local agent probes peers on startup, exposes their purposes and skills through an enriched discover tool, and provides a new `wingmate_ask` tool that forwards messages to named peers via A2A protocol.

**Solution approach:**
- Add `--purpose` CLI flag mapped to `AgentCard.Description` for agent specialization
- Populate `knownPeers` on startup by probing configured `--peers` URLs
- Extend `PeerProvider` interface to return rich peer data (name, URL, description, skills, status)
- Add `wingmate_ask` MCP tool that delegates to peers via `protocol.Client.SendMessage()`
- Wire `SystemPrompt` into A2A message handling for specialized responses
- All delegation logged to Flight Log with correlated trace IDs

**Success metrics:**
- `wingmate_discover` returns non-empty peer list with descriptions
- `wingmate_ask` successfully round-trips a message through a remote peer
- All delegation traceable via shared Flight Log trace IDs
- Existing A2A and MCP functionality unchanged

---

## Technical Context

### Current System State

| Component | State | Gap |
|-----------|-------|-----|
| `knownPeers` map | Initialized empty, never written on startup | Discover always returns `[]` |
| `AgentCard.Description` | Hardcoded to "Wingmate A2A agent" | No purpose differentiation |
| MCP handlers | No `protocol.Client` access | Cannot forward to peers |
| `PeerProvider` | Returns `[]string` (URLs only) | No name/description/skills |
| `Config.Claude.SystemPrompt` | Set but only used by CLI executor | Not used for A2A responses |

### Constraints

| Source | Constraint |
|--------|-----------|
| ADR-003 | Unified peer architecture -- no modes |
| ADR-004 | MCP via HTTP at `/mcp` only |
| Architecture rules | `internal/mcp/` MUST NOT import `internal/agent/` |
| P1 | All delegation logged to Flight Log |
| P6 | MCP remains localhost-only |
| PL-03 | JSON-RPC IDs preserved as `any` |
| PL-06 | Interfaces defined in consumer (`mcp/`), implemented in provider (`agent/`) |
| PL-07 | Delegation handler self-contained |

### Assumptions

- Peers may be unavailable at startup; probing must be non-blocking
- Peer names are unique within a network (no name collision handling)
- `protocol.Client.SendMessage()` is the existing A2A outbound path

---

## Critical Research Findings

Synthesized from research dossier (65+ findings) and 2 implementation subagents (16 discoveries).

### 01: knownPeers Never Populated (Critical)
**Sources**: Dossier Discovery 01, I1-01, I1-04
**Problem**: `Agent.knownPeers` initialized empty at `agent.go:103`, only written by `GetPeerCard()` which is called only by CLI commands. Despite `Config.Peers` containing URLs, they are never probed on startup.
**Action**: Add `probePeers()` goroutine in `Agent.Start()` after server ready, iterating `config.Peers` and calling `GetPeerCard()`.
**Affects**: Phase 2

### 02: PeerProvider Returns URLs Only (Critical)
**Sources**: Dossier Discovery 04, I1-02
**Problem**: `PeerProvider.GetPeers()` returns `[]string`. `wingmate_ask` needs name-based peer resolution, `wingmate_discover` needs rich data.
**Action**: Define `PeerInfo` struct in `mcp/`. Change interface to `GetPeerInfo() []PeerInfo`. Agent implements by reading from `knownPeers` cache.
**Affects**: Phase 2, Phase 3

### 03: MCP Handlers Lack A2A Client (Critical)
**Sources**: Dossier Discovery 03, I1-03
**Problem**: `NewChatHandler` receives `LLMExecutor` but no `protocol.Client`. Cannot send A2A messages.
**Action**: New `NewAskHandler` accepts a `PeerDelegator` interface (with `SendMessage` method) defined in `mcp/`. Agent implements it.
**Affects**: Phase 3

### 04: JSON-RPC ID Type Must Be Preserved (Critical)
**Sources**: PL-03, R1-01
**Problem**: IDs arrive as `float64` from `json.Unmarshal`. Delegation must not cast or narrow IDs.
**Action**: Use `json.RawMessage` for ID fields in delegation structs. Unit test with string, integer, float ID types.
**Affects**: Phase 3

### 05: Don't Modify HandleMessage (Critical)
**Sources**: R1-06, PL-07
**Problem**: `Agent.HandleMessage()` is the central A2A router. Touching it risks breaking existing A2A compatibility.
**Action**: `wingmate_ask` handler owns its own A2A call via injected `PeerDelegator` interface. `HandleMessage` unchanged.
**Affects**: Phase 3

### 06: Treat Peer Responses as Untrusted (Critical)
**Sources**: R1-04
**Problem**: Delegation proxies remote A2A responses back through MCP. A malicious peer could craft responses with extra JSON-RPC fields.
**Action**: Parse only the `result.message` field from A2A responses. Never forward raw JSON-RPC frames. Validate response structure.
**Affects**: Phase 3

### 07: Description Field Hardcoded in buildAgentCard (High)
**Sources**: I1-07, R1-08
**Problem**: `buildAgentCard()` at `agent.go:153` hardcodes `Description: "Wingmate A2A agent"`.
**Action**: Add `Description` field to `Config`. `--purpose` flag sets it. `buildAgentCard()` reads from config with fallback to default.
**Affects**: Phase 1

### 08: Config.Merge() Zero-Value Handling (High)
**Sources**: R1-03, PL-08
**Problem**: `Merge()` skips zero-value fields. New `Description` field with empty string default must not overwrite an existing value.
**Action**: Follow existing pattern -- `Merge()` only overwrites if `flags.Description != ""`. This is consistent with how `flags.Name` and `flags.LogFile` work.
**Affects**: Phase 1

### 09: Dual Flight Log Entries Need Trace Correlation (High)
**Sources**: PL-09, R1-07
**Problem**: Delegation produces MCP inbound + A2A outbound entries. Without shared trace ID, they cannot be correlated.
**Action**: Generate trace ID at MCP handler entry. Pass via `context.Context` to A2A client call. Both entries share the ID.
**Affects**: Phase 3

### 10: Separate PeerDelegator Interface (High)
**Sources**: R1-05, PL-06
**Problem**: Extending `PeerProvider` with delegation methods forces all existing mocks to update.
**Action**: Create separate `PeerDelegator` interface in `mcp/` with `DelegateMessage(ctx, peerNameOrURL, message) (response, error)`. Agent implements both `PeerProvider` and `PeerDelegator`.
**Affects**: Phase 3

### 11: Background Peer Health Checks (Medium)
**Sources**: Dossier AC-06
**Problem**: Peers may go offline after initial probe. Discover should indicate availability.
**Action**: Add `PeerInfo.Available bool` field. Periodic re-probe goroutine (configurable interval). Mark unavailable peers in discover response.
**Affects**: Phase 2

### 12: SystemPrompt Not Used in A2A Handling (Medium)
**Sources**: Dossier Discovery 02, Dossier AC-03
**Problem**: `Config.Claude.SystemPrompt` exists but is only passed to CLI executor, not to A2A message handling context.
**Action**: Pass system prompt to `llmExecutor.Execute()` call in `handleLLMMessage()`. Verify CLI executor uses it.
**Affects**: Phase 4

---

## Testing Philosophy

### Testing Approach
- **Selected Approach**: Full TDD
- **Rationale**: CS-4 feature with new protocol surface, interface changes, and cross-package dependencies
- **Focus Areas**: Peer delegation round-trip, peer registry population, PeerProvider/PeerDelegator contracts, `wingmate_ask` handler, enhanced discover response
- **Mock Usage**: Targeted mocks -- mock external systems (Claude CLI, HTTP peer endpoints) only. Use real code for internal logic. Extend existing `MockLLMExecutor`, `mockPeerProvider`, `mockHandler` patterns.

### Test-Driven Development
- Write tests FIRST (RED)
- Implement minimal code (GREEN)
- Refactor for quality (REFACTOR)

### Test Documentation
Every test must include:
```
Purpose: [what truth this test proves]
Quality Contribution: [how this prevents bugs]
Acceptance Criteria: [measurable assertions]
```

### Mock Strategy
- **Mock**: `protocol.Client` (for A2A outbound in ask handler tests)
- **Mock**: HTTP peers (for peer probing tests)
- **Mock**: `LLMExecutor` (existing pattern, for chat handler)
- **Real**: PeerProvider/PeerDelegator implementations, Config, AgentCard building

---

## Implementation Phases

### Phase 1: Config, Types & Purpose Plumbing

**Objective**: Add `Description` field to Config, `--purpose` CLI flag, and wire through to `AgentCard.Description`.

**Deliverables**:
- `Config.Description` field with WithEnv/WithDefaults/Merge support
- `--purpose` CLI flag
- `WINGMATE_PURPOSE` environment variable
- `buildAgentCard()` uses config description
- All existing tests passing

**Dependencies**: None (foundational phase)

**Risks**:
| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Merge() zero-value issue | Low | Medium | Follow existing `Name` pattern -- only overwrite if non-empty |

### Tasks (Full TDD)

| # | Status | Task | CS | Success Criteria | Log | Notes |
|---|--------|------|----|------------------|-----|-------|
| 1.1 | [x] | Write tests for Config.Description field | 2 | Tests cover: WithEnv, WithDefaults, Merge, Clone, JSON round-trip | T001 | config_test.go |
| 1.2 | [x] | Add Description to Config struct | 1 | Tests from 1.1 pass | T002 | config.go |
| 1.3 | [x] | Write tests for buildAgentCard with description | 1 | Tests verify: config description used, default fallback | T003 | agent_test.go |
| 1.4 | [x] | Update buildAgentCard to use Config.Description | 1 | Tests from 1.3 pass; hardcoded string replaced | T004 | agent.go:154 |
| 1.5 | [x] | Add --purpose flag to CLI | 1 | Flag parsed, sets Config.Description | T005 | cmd/wingmate/main.go |
| 1.6 | [x] | Run full test suite | 1 | `go test ./... -race` passes | T006 | Regression check |

### Acceptance Criteria
- [x] `--purpose "Build iOS apps"` results in AgentCard.Description = "Build iOS apps"
- [x] Missing `--purpose` falls back to default description ("Wingmate A2A agent")
- [x] `WINGMATE_PURPOSE` env var works
- [x] All existing tests pass: `go test ./... -race`

### Verification Commands
```bash
# Build and verify --purpose flag
go build -o wingmate ./cmd/wingmate
./wingmate --port 0 --name test-agent --purpose "Build iOS apps" &
curl -s http://localhost:<port>/.well-known/agent.json | jq '.description'
# Expected: "Build iOS apps"
```

---

### Phase 2: Peer Bootstrap & Discovery Enrichment

**Objective**: Populate `knownPeers` on startup, extend PeerProvider to return rich data, enhance `wingmate_discover` response.

**Deliverables**:
- `probePeers()` goroutine in Agent.Start()
- `PeerInfo` struct and extended `PeerProvider` interface
- Enhanced `DiscoverInfo` with rich peer data
- Background peer health re-probe

**Dependencies**: Phase 1 (Config.Description must exist for peers to advertise purpose)

**Risks**:
| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Peer unavailable at startup | High | Low | Non-blocking goroutine, log warning, retry |
| PeerProvider interface change | Low | High | Single implementor, compile-time catch |

### Tasks (Full TDD)

| # | Status | Task | CS | Success Criteria | Log | Notes |
|---|--------|------|----|------------------|-----|-------|
| 2.1 | [x] | Write tests for PeerInfo struct and extended PeerProvider | 2 | Tests cover: PeerInfo fields, GetPeerInfo contract | - | handlers_test.go |
| 2.2 | [x] | Define PeerInfo struct and update PeerProvider interface | 2 | Compile succeeds; tests from 2.1 pass | - | handlers.go |
| 2.3 | [x] | Update Agent.GetPeerInfo() implementation | 2 | Agent implements new interface, reads from knownPeers | - | agent.go |
| 2.4 | [x] | Write tests for probePeers() | 2 | Tests cover: all peers probed, unreachable peers handled, knownPeers populated | - | agent_test.go; mock HTTP server |
| 2.5 | [x] | Implement probePeers() in Agent.Start() | 3 | Startup probes configured peers, populates knownPeers | - | agent.go after line 233 |
| 2.6 | [x] | Write tests for enhanced DiscoverHandler | 2 | Tests cover: rich response format, empty peers, peer with/without description | - | handlers_test.go |
| 2.7 | [x] | Update NewDiscoverHandler to use PeerInfo | 2 | Returns name, URL, description, skills, available status | - | handlers.go |
| 2.8 | [x] | Write tests for background health re-probe | 2 | Tests cover: periodic re-probe (default 30s, configurable via `WINGMATE_PEER_PROBE_INTERVAL`), peer goes offline after 2 failed probes, peer recovery detected within 1 interval | - | agent_test.go |
| 2.9 | [x] | Implement background health re-probe goroutine | 2 | Periodic check updates knownPeers availability; selects on ctx.Done() for clean shutdown | - | agent.go |
| 2.10 | [x] | Write integration test: two agents, discover returns peer | 3 | Start agent1 (port 0, name=alpha, purpose="iOS dev", peers=agent2 URL) and agent2 (port 0, name=bravo, purpose="Backend dev"). Call wingmate_discover via MCP on agent1. Response includes bravo with description "Backend dev". | - | tests/integration/ |
| 2.11 | [x] | Run full test suite | 1 | `go test ./... -race` passes | - | Regression check |

### Acceptance Criteria
- [x] Agent with `--peers http://localhost:9100` probes peer on startup
- [x] `wingmate_discover` returns peer name, URL, description, skills
- [x] Unreachable peers shown as unavailable
- [x] Background re-probe detects peer recovery
- [x] All existing tests pass

---

### Phase 3: Delegation Tool (wingmate_ask)

**Objective**: Add `wingmate_ask` MCP tool that delegates messages to named peers via A2A protocol.

**Deliverables**:
- `PeerDelegator` interface in `mcp/`
- `wingmate_ask` tool definition and handler
- Name-based and URL-based peer resolution
- Session ID support for multi-turn delegation
- Flight Log dual entries with correlated trace IDs

**Dependencies**: Phase 2 (knownPeers populated, PeerProvider enriched)

**Risks**:
| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| JSON-RPC ID corruption | High | Critical | Use json.RawMessage; round-trip test |
| Untrusted peer response | Medium | Critical | Parse only result.message; validate structure |
| HandleMessage regression | Medium | Critical | Don't touch it; self-contained handler |

### Tasks (Full TDD)

| # | Status | Task | CS | Success Criteria | Log | Notes |
|---|--------|------|----|------------------|-----|-------|
| 3.1 | [x] | Write tests for PeerDelegator interface | 2 | Tests cover: delegate by name, by URL, unknown peer error, unavailable peer error | - | handlers_test.go |
| 3.2 | [x] | Define PeerDelegator interface in mcp/ | 1 | Interface compiles, separate from PeerProvider | - | handlers.go |
| 3.3 | [x] | Implement PeerDelegator on Agent | 3 | Agent.DelegateMessage resolves peer, calls SendMessage, logs both entries | - | agent.go or client.go |
| 3.4 | [x] | Write tests for wingmate_ask tool definition | 1 | Schema has peer (required), message (required), session_id (optional) | - | tools_test.go |
| 3.5 | [x] | Add wingmate_ask tool definition | 1 | Tool in DefaultTools(), schema correct | - | tools.go |
| 3.6 | [x] | Write tests for NewAskHandler | 3 | Tests: successful delegation, unknown peer (error code 3010), unavailable peer (error with last-seen), empty message (error code 3003), session_id passthrough. Mock PeerDelegator captures ctx; assert trace ID from ctx matches Flight Log entries. | - | handlers_test.go |
| 3.7 | [x] | Implement NewAskHandler | 3 | Handler delegates via PeerDelegator, returns response text | - | handlers.go |
| 3.8 | [x] | Write tests for JSON-RPC ID round-trip | 2 | IDs of type string, int, float survive delegation without type change | - | handlers_test.go |
| 3.9 | [x] | Register wingmate_ask in Agent.New() | 1 | Tool appears in tools/list, handler registered | - | agent.go after line 127 |
| 3.10 | [x] | Write tests for Flight Log trace correlation | 2 | MCP inbound + A2A outbound share same trace ID | - | handlers_test.go |
| 3.11 | [x] | Write integration test: full delegation round-trip | 3 | Start agent1 (port 0, name=alpha, peers=agent2 URL) and agent2 (port 0, name=bravo) with MockLLMExecutor. Send MCP tools/call wingmate_ask(peer="bravo", message="hello") to agent1. Assert response contains bravo's mock LLM output. Assert Flight Log has 2 entries with matching trace ID. | - | tests/integration/ |
| 3.12 | [x] | Run full test suite | 1 | `go test ./... -race` passes | - | Regression check |

### Acceptance Criteria
- [x] `wingmate_ask(peer="bravo", message="hello")` returns bravo's LLM response
- [x] Name and URL peer resolution both work
- [x] Unknown peer returns clear error
- [x] Unavailable peer returns clear error with last-seen info
- [x] `session_id` parameter enables multi-turn delegation
- [x] Flight Log has correlated MCP + A2A entries
- [x] JSON-RPC IDs preserved through round-trip
- [x] `HandleMessage()` untouched

---

### Phase 4: System Prompt & Specialization

**Objective**: Wire `Config.Claude.SystemPrompt` into A2A message handling so specialized agents respond in character.

**Deliverables**:
- System prompt passed to LLM executor for A2A messages
- Verification that peer agents respond according to their system prompt

**Dependencies**: Phase 1 (config), Phase 3 (delegation for end-to-end test)

**Risks**:
| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| CLI executor ignores system prompt | Low | Medium | Verify CLIExecutor uses --system-prompt flag |

### Tasks (Full TDD)

| # | Status | Task | CS | Success Criteria | Log | Notes |
|---|--------|------|----|------------------|-----|-------|
| 4.1 | [x] | Write tests for system prompt in A2A handling | 2 | Tests: system prompt passed to executor, empty prompt handled | - | agent_test.go |
| 4.2 | [x] | Wire system prompt into handleLLMMessage | 2 | System prompt from config passed to executor.Execute() | - | agent.go:352-353 |
| 4.3 | [x] | Write test verifying CLIExecutor receives system prompt | 1 | Test asserts --system-prompt arg present in CLI invocation when config.SystemPrompt is non-empty, absent when empty | - | llm/cli_test.go |
| 4.4 | [x] | Run full test suite | 1 | `go test ./... -race` passes | - | Regression check |

### Acceptance Criteria
- [x] Agent with system prompt "You are an iOS expert" responds in character to A2A messages
- [x] Empty system prompt uses existing default behavior
- [x] Existing A2A behavior unchanged when no custom prompt set

---

### Phase 5: Documentation

**Objective**: Document peer delegation for users following hybrid approach (README + docs/how/).

**Deliverables**:
- README.md updated with peer delegation section
- `docs/how/peer-delegation.md` detailed guide
- CLAUDE.md updated with new tools and concepts

**Dependencies**: All implementation phases complete

**Risks**:
| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Documentation drift | Medium | Low | Include doc review in phase acceptance |

### Discovery & Placement Decision

**Existing docs/how/ structure**: `mcp-setup.md`, `llm-setup.md`
**Decision**: Create new `docs/how/peer-delegation.md` (distinct from MCP setup)

### Tasks

| # | Status | Task | CS | Success Criteria | Log | Notes |
|---|--------|------|----|------------------|-----|-------|
| 5.1 | [ ] | Update README.md with peer delegation section | 2 | Features list, `--purpose`/`--peers` example, `wingmate_ask` mention | - | README.md |
| 5.2 | [ ] | Create docs/how/peer-delegation.md | 3 | Covers sections: Overview, Multi-Agent Setup, Purpose Config, System Prompts, Delegation Workflow, Troubleshooting | - | docs/how/peer-delegation.md |
| 5.3 | [ ] | Update CLAUDE.md with new tools and concepts | 2 | wingmate_ask tool, PeerDelegator interface, peer delegation section | - | CLAUDE.md |
| 5.4 | [ ] | Update docs/how/mcp-setup.md with wingmate_ask | 1 | Add wingmate_ask to available tools section | - | docs/how/mcp-setup.md |

### Acceptance Criteria
- [ ] README shows peer delegation in features
- [ ] docs/how/peer-delegation.md covers full setup workflow
- [ ] CLAUDE.md reflects all new tools and interfaces
- [ ] No broken internal links

---

## Cross-Cutting Concerns

### Security
- MCP remains localhost-only (`LocalhostMiddleware` unchanged)
- A2A peer responses treated as untrusted -- parse only expected fields
- No credentials exposed in delegation (authentication deferred per DEV-002)

### Observability
- All delegation logged to Flight Log with dual entries (MCP inbound + A2A outbound)
- Trace ID correlation via `context.Context`
- Peer probing results logged (success/failure per peer)

### New CLI Flags & Environment Variables

| Flag | Env Var | Default | Purpose |
|------|---------|---------|---------|
| `--purpose` | `WINGMATE_PURPOSE` | "Wingmate A2A agent" | Agent description/purpose in AgentCard |
| - | `WINGMATE_PEER_PROBE_INTERVAL` | `30s` | Background peer health check interval |

### Delegation Error Codes

| Code | Name | When |
|------|------|------|
| 3010 | MCPToolNotFound | Unknown peer name or URL |
| 3011 | MCPToolExecutionFailed | Peer returned error or delegation failed |
| 3003 | MCPInvalidParams | Empty message or missing required params |

### Documentation
- **Location**: Hybrid (README + docs/how/)
- **README**: Feature overview, basic examples
- **docs/how/peer-delegation.md**: Full setup guide, system prompts, troubleshooting
- **Target audience**: Engineers setting up multi-agent environments

### docs/how/peer-delegation.md Outline

1. Overview -- what peer delegation is, architecture diagram
2. Multi-Agent Setup -- starting agents with `--purpose` and `--peers`
3. Purpose & System Prompt Configuration -- per-agent specialization
4. Delegation Workflow -- using `wingmate_ask` via MCP, examples with curl
5. Discovery -- using `wingmate_discover` to see available peers
6. Troubleshooting -- peer unavailable, JSON-RPC errors, probe intervals
7. Security Considerations -- localhost MCP, A2A open, container model

---

## Complexity Tracking

| Component | CS | Label | Breakdown (S,I,D,N,F,T) | Justification | Mitigation |
|-----------|-----|-------|--------------------------|---------------|------------|
| wingmate_ask handler | 4 | Large | S=1,I=2,D=1,N=1,F=1,T=2 | New tool with A2A forwarding, peer resolution, error handling, trace correlation | Self-contained handler (PL-07); mock-based unit tests; integration test |
| Peer bootstrap (probePeers) | 3 | Medium | S=1,I=1,D=1,N=1,F=1,T=1 | Background goroutine with retry, concurrent map writes | Existing RWMutex pattern; race detector tests |
| PeerProvider extension | 3 | Medium | S=1,I=2,D=1,N=0,F=1,T=1 | Interface change affects implementor + test mocks | Single implementor; compile-time safety |

---

## Progress Tracking

### Phase Completion Checklist
- [x] Phase 1: Config, Types & Purpose Plumbing
- [x] Phase 2: Peer Bootstrap & Discovery Enrichment
- [x] Phase 3: Delegation Tool (wingmate_ask)
- [x] Phase 4: System Prompt & Specialization
- [ ] Phase 5: Documentation

### STOP Rule
This plan must be validated before creating tasks. After review:
1. Run `/plan-4-complete-the-plan` to validate readiness
2. Only proceed to `/plan-5-phase-tasks-and-brief` after validation passes

---

## Deviation Ledger

No constitution or architecture deviations required. All changes follow:
- P1: Delegation logged to Flight Log
- P4: A2A protocol compliance for peer communication
- P6: MCP localhost-only preserved
- Architecture: Interfaces in consumer (mcp), implemented in provider (agent)

---

## ADR Ledger

| ADR | Status | Affects Phases | Notes |
|-----|--------|----------------|-------|
| ADR-003 | Accepted | All | Unified peers -- no modes. Peer delegation is a conversation-role feature. |
| ADR-004 | Accepted (amended) | 3, 5 | MCP via HTTP at /mcp. New tool registered in same handler. |

No new ADR recommended -- peer delegation extends existing architecture without introducing new architectural decisions.

---

## Change Footnotes Ledger

[^1]: [To be added during implementation via plan-6a]
[^2]: [To be added during implementation via plan-6a]
[^3]: [To be added during implementation via plan-6a]
