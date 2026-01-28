# Feature Specification: Peer Delegation

**Plan ID**: 005
**Feature Name**: Peer Delegation & Agent Specialization
**Status**: DRAFT
**Created**: 2026-01-28
**Mode**: Full
**Complexity Score**: CS-4 (significant changes across multiple packages, new protocol surface, new MCP tools)
**Research Dossier**: [research-dossier.md](./research-dossier.md)

---

## 1. Problem Statement

When Claude Code connects to a Wingmate agent via MCP, all tool invocations execute locally on the connected agent. There is no way for Claude Code to reach remote peer agents through MCP. The `wingmate_chat` tool runs the Claude CLI on the local machine, `wingmate_discover` returns an empty peer list (due to a bug where `knownPeers` is never populated), and no tool exists to send a message to a specific peer.

This means a user running multiple specialized agents (e.g., an iOS builder in one container, a backend developer in another, a data pipeline agent in a third) cannot leverage those agents from their local Claude Code session. Each agent is an island.

## 2. User Value

### Who Benefits

Engineers running Claude Code on a primary machine who want to delegate specialized tasks to purpose-built agents running in containers or on other machines.

### What They Can Do Today

- Connect Claude Code to one local Wingmate agent via MCP
- Use `wingmate_chat` to talk to the local agent's LLM
- Use `wingmate_status` to check local health
- Use `wingmate_discover` to see peers (always empty)
- Manually use `wingmate ping` and `wingmate status` CLI commands to interact with peers

### What They'll Be Able to Do

- **Discover peers with purpose**: Ask "what agents are available?" and get back a list with each agent's name, purpose (e.g., "Build native iOS applications"), and capabilities
- **Delegate to a peer by name**: Ask Claude Code to send a message to a specific peer agent (e.g., "ask the iOS builder to create a SwiftUI view for user settings") and get the response back through MCP
- **Run specialized agents**: Start agents with a declared purpose and optional system prompt that shapes how they respond to incoming messages
- **See the full network**: The local agent probes configured peers on startup, maintains liveness awareness, and surfaces this through the discover tool

### Why This Matters

The local machine is the coordination hub with strict security. Containers can run with looser controls (file access, code execution) because they're sandboxed. Peer delegation lets Claude Code on the host orchestrate work across specialized, sandboxed agents without the user manually switching between terminals or copying messages.

## 3. Acceptance Criteria

### AC-01: Peer Discovery on Startup
**When** an agent starts with `--peers http://host1:9000,http://host2:9001`
**Then** it probes each peer URL, fetches their Agent Cards, and populates the internal peer registry
**And** `wingmate_discover` returns the peer names, URLs, purposes, and skills (not just empty)

### AC-02: Agent Purpose Declaration
**When** an agent starts with `--purpose "Build native iOS applications"`
**Then** its Agent Card at `/.well-known/agent.json` uses the `description` field for that value
**And** remote agents that discover this peer can see its purpose via the standard `description` field

### AC-03: System Prompt for Specialization
**When** an agent is configured with a system prompt (via config file or environment variable)
**Then** incoming A2A messages are handled with that system prompt passed to the LLM
**So that** the agent responds in character as its specialized role

### AC-04: Peer Delegation via MCP
**When** Claude Code calls a new MCP tool (e.g., `wingmate_ask`) with a peer identifier (name or URL) and a message
**Then** the local agent forwards the message to the specified peer via A2A protocol
**And** returns the peer's response back to Claude Code through MCP
**And** supports an optional `session_id` parameter for multi-turn conversations with the same peer
**And** both the MCP inbound and A2A outbound are logged to the Flight Log under the same trace ID

### AC-05: Enhanced Discovery Response
**When** Claude Code calls `wingmate_discover`
**Then** the response includes for each peer: name, URL, purpose, and list of skills
**So that** Claude Code (or the user) can make informed decisions about which peer to delegate to

### AC-06: Peer Liveness Awareness
**When** a configured peer is unreachable during startup probing or becomes unreachable later
**Then** the discover tool indicates the peer's status (available/unavailable)
**And** attempting to delegate to an unavailable peer returns a clear error

### AC-07: Flight Log Observability
**When** a delegation occurs (MCP -> A2A -> peer -> response -> MCP)
**Then** at least two Flight Log entries are created: one for the MCP tool invocation, one for the A2A outbound call
**And** they share a trace ID for correlation

## 4. Scope

### In Scope

- Populating `knownPeers` on agent startup from `Config.Peers`
- Mapping `--purpose` CLI flag to `AgentCard.Description` (reuse existing field, A2A spec compliant)
- Adding `--purpose` CLI flag and `Config.Purpose` (maps to Description in AgentCard)
- Wiring `SystemPrompt` from config into A2A message handling
- New MCP tool for peer delegation (`wingmate_ask` or similar)
- Extending `PeerProvider` interface to return rich peer data
- Enhancing `wingmate_discover` to return peer details
- Background peer health checks (periodic re-probe)
- Flight Log entries for delegated calls

### Out of Scope

- Container orchestration or Docker/Podman management (user manages their own containers)
- Authentication between peers (deferred; currently open per DEV-002 deviation)
- Multi-hop delegation (agent A delegates to B which delegates to C)
- Streaming responses through delegation
- MCP tool forwarding (exposing remote peer's MCP tools locally)
- Changes to the MCP localhost-only security boundary

## 5. Constraints

| Source | Constraint |
|--------|-----------|
| ADR-003 | Unified peer architecture - no separate pilot/wingmate modes |
| ADR-004 | MCP via HTTP only at `/mcp` endpoint |
| P1 (Constitution) | All delegation must be logged to Flight Log |
| P4 (Constitution) | A2A protocol compliance for peer communication |
| P6 (Constitution) | MCP remains localhost-only even when delegating |
| PL-03 | JSON-RPC IDs must be preserved as `any` when forwarding |
| PL-06 | New interfaces defined in consumer package (mcp), implemented in provider (agent) |
| PL-07 | Delegation handler should be self-contained, not coupled to Agent internals |
| PL-09 | Dual Flight Log entries (MCP + A2A) with correlated trace IDs |

## 6. Open Questions

### Resolved in Dossier

- **Where to define new interfaces?** In `mcp/` package, implemented by `agent/` (PL-06)
- **New tool or extend existing?** New tool (`wingmate_ask`) preferred over extending `wingmate_chat` - keeps local and remote semantics separate
- **How to identify peers?** By name (from AgentCard) or URL; name is more user-friendly

### Unresolved

- **Container security model**: The vision involves containers with relaxed security and host with strict security. The specifics of how to configure this (Claude Code permissions per environment, container networking) are not yet researched. See [Research Opportunity 2](./research-dossier.md#research-opportunity-2-container-security-models-for-delegated-agent-execution) in the dossier. This is out of scope for implementation but affects how users set up their environments.

### Resolved by Clarification

- **A2A spec compliance for AgentCard fields**: Resolved - reuse existing `description` field for agent purpose. No custom fields needed. `--purpose` flag maps to `Config.Purpose` which populates `AgentCard.Description`. (Q5, 2026-01-28)

## 7. Success Metrics

- `wingmate_discover` returns non-empty peer list with purposes when peers are configured and running
- `wingmate_ask` successfully delegates a message to a remote peer and returns the response via MCP
- All delegation round-trips are traceable in the Flight Log via shared trace ID
- Existing A2A ping/status functionality and MCP tools continue to work unchanged

## 8. User Scenarios

### Scenario A: iOS Build Delegation

1. User starts a specialized agent in a container: `wingmate --port 9100 --name ios-builder --purpose "Build native iOS applications" --config ios.yaml` (where `ios.yaml` sets a system prompt for iOS expertise)
2. User starts their local agent: `wingmate --port 9000 --name coordinator --peers http://container:9100`
3. User opens Claude Code, connected to local agent via MCP
4. User asks Claude: "Use wingmate to ask the iOS builder to create a SwiftUI settings view"
5. Claude calls `wingmate_ask` with peer "ios-builder" and the message
6. Local agent looks up "ios-builder" in knownPeers, sends A2A message
7. ios-builder agent processes with its iOS-specialized system prompt, responds
8. Response flows back through A2A -> local agent -> MCP -> Claude Code

### Scenario B: Multi-Agent Discovery

1. User has 3 containers running specialized agents
2. User asks Claude: "What wingmate agents are available?"
3. Claude calls `wingmate_discover`
4. Response shows: ios-builder ("Build native iOS applications"), backend-dev ("Develop Go microservices"), data-eng ("Build data pipelines")
5. User can now direct work to the appropriate agent

### Scenario C: Peer Unavailable

1. User configured a peer that is not currently running
2. User asks Claude to delegate to that peer
3. `wingmate_ask` returns a clear error: "Peer 'ios-builder' is not available (last seen: never)"
4. User can start the container and the next discover/ask will find it

## 9. Testing Strategy

- **Approach**: Full TDD
- **Rationale**: CS-4 feature with new protocol surface, interface changes, and cross-package dependencies. Tests before implementation catches integration issues early.
- **Focus Areas**: Peer delegation round-trip (MCP -> A2A -> peer -> response), peer registry population/liveness, PeerProvider interface contract, `wingmate_ask` tool handler, enhanced discover response format
- **Excluded**: Container setup/orchestration (out of scope), CLI flag parsing (trivial)
- **Mock Usage**: Targeted mocks - mock external systems (Claude CLI subprocess, HTTP peer endpoints) but use real code for internal logic. Extends existing `MockLLMExecutor`, `mockPeerProvider`, `mockHandler` patterns.
- **Integration Tests**: Multi-agent integration test (start two agents, delegate from one to the other via MCP tool call)

## 10. Documentation Strategy

- **Location**: Hybrid (README + docs/how/)
- **Rationale**: Peer delegation is a primary use case that needs visibility in README, with detailed setup in docs/how/
- **Content Split**:
  - **README.md**: Add peer delegation to feature list, add example showing `--purpose` and `--peers` flags, mention `wingmate_ask` tool
  - **docs/how/**: New `peer-delegation.md` guide covering multi-agent setup, purpose configuration, system prompts, delegation workflow, troubleshooting
- **Target Audience**: Engineers setting up multi-agent environments
- **Maintenance**: Update when new delegation tools or peer features are added

## Clarifications

### Session 2026-01-28

| # | Question | Answer | Spec Impact |
|---|----------|--------|-------------|
| Q1 | Workflow mode? | **Full** | Added `**Mode**: Full` to header |
| Q2 | Testing approach? | **Full TDD** | Added Section 9: Testing Strategy |
| Q3 | Mock usage? | **Targeted mocks** | Added to Section 9 |
| Q4 | Documentation location? | **Hybrid (README + docs/how/)** | Added Section 10: Documentation Strategy |
| Q5 | Purpose field naming? | **Reuse `description`** | Resolves Research Opportunity 1. Updated AC-02, Scope, Open Questions. Use existing `description` field for agent purpose instead of adding new `purpose` field. `--purpose` flag maps to `Config.Description` -> `AgentCard.Description`. |
| Q6 | Conversation continuity? | **Yes, with `session_id`** | Updated AC-04. `wingmate_ask` takes optional `session_id` parameter for multi-turn peer conversations. |

**Coverage Summary:**

| Category | Status |
|----------|--------|
| Workflow Mode | Resolved (Full) |
| Testing Strategy | Resolved (Full TDD, targeted mocks) |
| Documentation Strategy | Resolved (Hybrid) |
| Purpose field naming | Resolved (reuse `description`) |
| Conversation continuity | Resolved (session_id parameter) |
| A2A AgentCard extensions | Resolved (reuse `description` - no custom fields needed) |
| Container security model | Deferred (out of scope, affects user setup docs only) |

---

**Soft Warning**: One external research opportunity remains deferred (container security models). This does not block implementation but may inform the peer-delegation setup guide in docs/how/.
