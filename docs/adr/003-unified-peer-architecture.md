# ADR-003: Unified Peer Architecture

**Status**: DECIDED
**Date**: 2026-01-21
**Deciders**: Core maintainers
**Supersedes**: N/A
**Superseded by**: N/A

---

## Context

The original architecture defined two separate modes for the wingmate binary:

- **Pilot mode** (`--mode pilot`): Acts as A2A client, initiates requests
- **Wingmate mode** (`--mode wingmate`): Acts as A2A server, responds to requests

However, real-world debugging scenarios reveal this is unnecessarily restrictive:

1. **Server debugging client**: Server asks client device for logs
2. **Client debugging server**: Client asks server for request traces
3. **Phone debugging backend**: Phone asks what happened to its recent request
4. **Backend debugging phone**: Backend asks phone to pull APK and install for testing

In all these scenarios, **any node might need to initiate or respond**. The "pilot vs wingmate" distinction is about the **role in a specific conversation**, not about how an instance should be configured.

The A2A protocol itself supports this - it's peer-to-peer by design:

> "A2A is peer-to-peer in the sense that any agent can expose an endpoint to serve requests, and also itself call others' endpoints."
> — A2A Research Document

---

## Decision Drivers

- Any agent should be able to ask questions of any other agent
- Conversation roles (initiator vs responder) are dynamic, not static
- Simpler mental model: "peers that can talk to each other"
- Reduced configuration complexity
- No code has been written in `internal/pilot/` or `internal/wingmate/` yet

---

## Options Considered

### Option A: Keep Separate Modes (Status Quo)

**Description**: Maintain `--mode pilot` and `--mode wingmate` as distinct operating modes.

**Pros**:
- Clear separation of concerns in code
- Simpler security model (some nodes can be "receive only")

**Cons**:
- Artificial restriction on real debugging workflows
- Confusing: "Why can't my server ask the client a question?"
- Requires running two instances on the same machine for full capability
- Mode is a deployment-time decision, but needs change at runtime

### Option B: Unified Peer Model (Recommended)

**Description**: Every wingmate instance is both a server (can receive) AND a client (can initiate). "Pilot" and "Wingmate" become conversation-level roles.

**Pros**:
- Matches real debugging workflows
- Simpler configuration: just specify port and peers
- Any node can ask any other node
- Terminology still useful (pilot = initiator of THIS conversation)
- Simpler codebase: one agent type, not two

**Cons**:
- Must handle concurrent incoming and outgoing requests
- Security policy slightly more complex (must authorize both directions)

### Option C: Hybrid - Default Unified, Optional Restriction

**Description**: Default to unified peer model, but allow `--receive-only` or `--initiate-only` flags for restricted deployments.

**Pros**:
- Best of both worlds
- Supports security-restricted scenarios

**Cons**:
- Added complexity for edge cases
- Can be added later if needed (YAGNI)

---

## Decision

**Chosen Option**: Option B - Unified Peer Model

Every wingmate instance runs as a full peer:
- Listens on a port for incoming A2A requests (can be a "wingmate" in conversations)
- Can connect to known peers to send A2A requests (can be a "pilot" in conversations)

The terminology shifts:
- **Pilot**: The agent that **initiated** a specific task/conversation
- **Wingmate**: The agent **responding** to that specific task/conversation
- These are **roles within a conversation**, not instance configurations

---

## Consequences

### Positive

- **Simpler mental model**: "These machines can talk to each other"
- **Flexible workflows**: Server can debug client, client can debug server
- **Less configuration**: No mode selection required
- **Cleaner codebase**: One `internal/agent/` instead of `internal/pilot/` + `internal/wingmate/`
- **Future-proof**: Supports multi-agent mesh topologies

### Negative

- **Concurrent request handling**: Agent must handle being pilot and wingmate simultaneously
  - Mitigation: Go's concurrency model handles this well
- **Security surface**: Both inbound and outbound paths need authorization
  - Mitigation: Unified auth layer, capability-based restrictions

### Neutral

- Flight Log still records direction (inbound/outbound) per conversation
- Agent Card still declares capabilities for incoming requests

---

## Implementation Changes

### CLI Changes

**Before:**
```bash
# Two separate modes
./wingmate --mode wingmate --port 9001
./wingmate --mode pilot --port 9000 --peer http://localhost:9001
```

**After:**
```bash
# Single unified mode
./wingmate --port 9000 --peers http://machineB:9001,http://machineC:9002

# Minimal (no peers, just accepts incoming)
./wingmate --port 9000

# Peer discovery later (future enhancement)
./wingmate --port 9000 --discover
```

### Directory Structure Changes

**Before:**
```
internal/
├── pilot/              # Pilot-specific code
│   ├── pilot.go
│   └── mission.go
├── wingmate/           # Wingmate-specific code
│   ├── wingmate.go
│   └── handlers.go
├── agent/              # Shared code
└── protocol/
```

**After:**
```
internal/
├── agent/              # THE agent (unified)
│   ├── agent.go        # Agent struct with both client + server
│   ├── server.go       # A2A server capabilities (handle incoming)
│   ├── client.go       # A2A client capabilities (make outgoing)
│   ├── conversation.go # Conversation/mission management
│   └── config.go       # Configuration
├── protocol/           # A2A protocol (unchanged)
└── flightlog/          # Flight log (unchanged)
```

### Architecture Document Updates

Update `docs/project-rules/architecture.md`:
- Remove references to `--mode` flag
- Update deployment diagrams to show bidirectional arrows on both sides
- Update module boundaries to remove pilot/wingmate split
- Clarify that pilot/wingmate are conversation roles

### Configuration Changes

**Before:**
```json
{
  "mode": "wingmate",
  "port": 9000,
  "peer": "http://other:9001"
}
```

**After:**
```json
{
  "name": "agent-device-01",
  "port": 9000,
  "peers": [
    "http://machineB:9001",
    "http://machineC:9002"
  ],
  "capabilities": ["GetMetrics", "GetLogs"]
}
```

---

## Flight Log Implications

The Flight Log already tracks `direction` (inbound/outbound). This remains correct:

```json
{
  "timestamp": "2026-01-21T14:32:17.123Z",
  "direction": "outbound",
  "local_agent": "agent-device-01",
  "remote_agent": "agent-server-02",
  "role": "pilot",
  "summary": "Requested server logs for last 10 minutes"
}
```

The `role` field indicates whether this agent was the pilot (initiator) or wingmate (responder) **in this specific exchange**.

---

## Terminology Clarification

| Term | Old Meaning | New Meaning |
|------|-------------|-------------|
| **Pilot** | A binary mode (`--mode pilot`) | The initiator of a specific conversation |
| **Wingmate** | A binary mode (`--mode wingmate`) | The responder in a specific conversation |
| **Agent** | Abstract base | The actual running instance (can be either role) |
| **Mission** | A task initiated by pilot | A task/conversation between two agents |
| **Sortie** | A request-response in a mission | Unchanged |

---

## Migration Path

Since `internal/pilot/` and `internal/wingmate/` are currently empty, no migration is needed. This ADR should be decided **before** implementing those directories.

If decided:
1. Do NOT create `internal/pilot/` or `internal/wingmate/`
2. Implement unified agent in `internal/agent/`
3. Update architecture.md and CLAUDE.md
4. Update minimal-skeleton-spec.md CLI design

---

## Security Considerations

With unified peers, security must cover both directions:

1. **Incoming requests**: Validate caller identity, check capabilities
2. **Outgoing requests**: Authenticate to remote, respect remote's capabilities
3. **Mutual TLS**: Recommended for production (both sides verify)
4. **Capability restrictions**: An agent can still limit what it exposes

Future enhancement (Option C): Add `--capabilities-inbound` and `--capabilities-outbound` for fine-grained control if needed.

---

## Related Decisions

- [ADR-001](./001-language-choice.md): Go language choice (unchanged)
- [ADR-002](./002-flight-log-storage.md): Flight Log format (minor field additions)
- Constitution Principle P4: Protocol Compliance (A2A peer-to-peer nature)

---

## References

- [A2A Protocol Specification](https://github.com/a2aproject/A2A/blob/main/specification/json/a2a.json)
- A2A Research: "A2A is peer-to-peer... any agent can expose an endpoint and call others' endpoints"

---

<!--
MACHINE-READABLE CONTEXT
========================
This section provides structured metadata for AI assistants to quickly parse ADR context.

```yaml
adr:
  id: 3
  title: "Unified Peer Architecture"
  status: "decided"
  date: "2026-01-21"

  decision_summary: "Remove pilot/wingmate modes; every instance is a full peer that can initiate or respond to conversations"

  affects:
    - component: "internal/agent"
      impact: "high"
    - component: "cmd/wingmate"
      impact: "high"
    - component: "internal/pilot"
      impact: "removed"
    - component: "internal/wingmate"
      impact: "removed"
    - component: "config"
      impact: "medium"

  constraints:
    - "No --mode flag; single operating mode"
    - "Every agent can both send and receive A2A requests"
    - "Pilot/Wingmate are conversation roles, not instance modes"
    - "internal/pilot/ and internal/wingmate/ directories should not be created"

  depends_on:
    - 1  # Language choice (Go)

  dependents: []

  principles:
    - "P4"  # Protocol Compliance (A2A peer-to-peer)
    - "P3"  # Engineer Autonomy (simpler config)

  implementation:
    status: "not_started"
    location: "internal/agent/"

  tags:
    - "architecture"
    - "simplification"
    - "a2a"

  complexity_impact:
    score_delta: -1
    reason: "Removes unnecessary mode distinction, simplifies mental model"
```
-->
