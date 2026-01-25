# ADR-004: MCP Server Implementation Approach

**Status**: DECIDED (Amended 2026-01-24)
**Date**: 2026-01-22
**Deciders**: Core maintainers
**Supersedes**: N/A
**Superseded by**: N/A
**Amendments**: HTTP Transport Migration (2026-01-24)

---

## Context

Wingmate needs to expose its LLM capabilities to Claude Code and other MCP-compatible clients. The Model Context Protocol (MCP) is the standardized way for LLM tools to communicate, allowing users to add Wingmate as an MCP server and immediately access its `wingmate_chat` tool without understanding A2A protocol or HTTP configuration.

Key considerations:
- Constitution P4 sanctions MCP for local tool integration
- ADR-001 mandates single Go binary with minimal dependencies
- MCP uses JSON-RPC 2.0 over stdio (newline-delimited)
- The protocol is well-specified but implementation is non-trivial

---

## Decision Drivers

- Single binary distribution (ADR-001)
- All operations must be observable (Constitution P1)
- MCP sanctioned for local tools (Constitution P4)
- Audit logging required (Constitution P6)
- Zero or minimal external dependencies preferred
- Development effort vs. implementation correctness tradeoff

---

## Options Considered

### Option A: Official SDK (modelcontextprotocol/go-sdk)

**Description**: Use the official Go MCP SDK maintained by the MCP team and Google.

**Pros**:
- Zero external dependencies in core `mcp` package
- Type-safe API with automatic JSON Schema generation from Go structs
- Stable v1.1.0 release (not beta)
- Backed by MCP team and Google
- Protocol compliance guaranteed
- stdio and HTTP transport support
- Session-based client-server architecture

**Cons**:
- Smaller community than mark3labs (3.1k vs 7.5k stars)
- Less documentation/examples currently
- API may evolve (mitigated by adapter pattern)

### Option B: Community SDK (mark3labs/mcp-go)

**Description**: Use the popular community-maintained MCP Go library.

**Pros**:
- Larger community (7,500 stars)
- More examples and documentation
- Fluent builder API reduces boilerplate
- Per-session tool customization

**Cons**:
- Pre-v1.0 beta status (v0.43.0-beta)
- Breaking changes possible (v0.29.0 had breaking changes)
- Slightly higher dependency footprint
- Not officially backed by MCP team

### Option C: Custom Implementation

**Description**: Implement MCP protocol from specification directly.

**Pros**:
- Full control over implementation
- Zero new dependencies
- No coupling to external library API changes

**Cons**:
- 5,000-10,000 lines of code
- Months of development time
- Higher risk of protocol compliance bugs
- Maintenance burden for protocol updates

---

## Decision

**Chosen Option**: Official SDK (Option A) - `modelcontextprotocol/go-sdk v1.1.0`

The official SDK aligns perfectly with Wingmate's architectural philosophy:

1. **Zero external dependencies** in core package matches ADR-001's minimal-dependency mandate
2. **Stable v1.x release** provides API stability guarantees
3. **Official backing** ensures protocol compliance and long-term maintenance
4. **Type-safe API** with automatic JSON Schema generation reduces implementation errors

The SDK provides 50-100x complexity reduction compared to custom implementation, and its zero-dependency design means adding it doesn't compromise our single-binary distribution model.

---

## Consequences

### Positive

- Single new dependency in go.mod (`github.com/modelcontextprotocol/go-sdk v1.1.0`)
- Protocol compliance guaranteed by official implementation
- Automatic JSON Schema generation from Go structs
- Built-in stdio transport handling
- Reduced development time (hours vs. months)

### Negative

- Tied to SDK API (mitigated by adapter pattern in `internal/mcp/`)
- Less community examples than mark3labs (mitigated: official examples are sufficient)

### Neutral

- Learning curve for SDK API (acceptable: well-documented)
- May need to update when SDK releases new versions

---

## Implementation Notes

### Package Structure

```
internal/mcp/
├── http_transport.go  # HTTP handler implementing http.Handler
├── localhost.go       # Localhost-only middleware (returns 403)
├── session.go         # Session manager with 30-min TTL
├── tools.go           # Tool definitions using SDK types
├── handlers.go        # Tool handlers with PeerProvider interface
├── types.go           # ServerConfig, ServerState, Logger types
└── errors.go          # Error codes 3001-3099
```

### Adapter Pattern

To isolate SDK API changes, wrap SDK types in internal types:

```go
// internal/mcp/types.go
type ToolInput struct {
    Prompt    string `json:"prompt"`
    SessionID string `json:"session_id,omitempty"`
}

// Handler receives internal type, SDK types stay in transport layer
func handleChat(ctx context.Context, input ToolInput) (*ToolOutput, error)
```

### Error Code Allocation

MCP-specific error codes reserved in 3001-3099 range:

| Code | Description |
|------|-------------|
| 3001 | Claude CLI not available |
| 3002 | Tool execution failed |
| 3003 | Tool timeout |
| 3004 | Session not found |
| 3005 | Invalid tool arguments |

### go.mod Update

```
require github.com/modelcontextprotocol/go-sdk v1.1.0
```

This is the only new dependency. The SDK has no transitive external dependencies.

---

## Related Decisions

- [ADR-001](./001-language-choice.md): Go language, single binary, minimal dependencies
- [ADR-002](./002-flight-log-storage.md): Flight Log format for observability
- [ADR-003](./003-unified-peer-architecture.md): MCP runs independently of A2A
- Constitution Principles: P1 (Observability), P4 (MCP sanctioned), P6 (Audit logging)

---

## References

- [MCP Protocol Specification](https://modelcontextprotocol.io/specification/2025-03-26/basic)
- [Official Go SDK](https://github.com/modelcontextprotocol/go-sdk) - v1.1.0
- [Claude Code MCP Configuration](https://code.claude.com/docs/en/mcp)
- Feature Spec: `docs/plans/003-mcp-server-integration/mcp-server-integration-spec.md`
- Research: `docs/plans/003-mcp-server-integration/external-research/02-go-mcp-libraries-results.md`

---

## Amendment: HTTP Transport Migration (2026-01-24)

### Context

The original ADR specified "stdio transport only (no HTTP for MCP)". This constraint has been superseded by the MCP HTTP Transport Migration (Plan 004).

### Rationale for Change

1. **Stdio isolation** - The subprocess model prevented `wingmate_discover` from returning actual peers
2. **Dual process burden** - Users had to manage two separate processes (agent + MCP subprocess)
3. **Unified architecture** - Single process now serves both A2A and MCP, simplifying operations
4. **Protocol alignment** - MCP Streamable HTTP (2025-03-26) is the current protocol standard

### New Decision

- MCP is available via HTTP transport only at `/mcp` endpoint
- Stdio transport has been removed (files deleted)
- Users configure via `claude mcp add --transport http`
- `wingmate mcp` command shows migration guidance and exits

### Updated Implementation

```
internal/mcp/
├── http_transport.go   # HTTP handler implementing http.Handler
├── localhost.go        # Localhost-only middleware (returns 403 for non-localhost)
├── session.go          # Session manager with 30-min TTL
├── tools.go            # Tool definitions (unchanged)
├── handlers.go         # Tool handlers with PeerProvider interface
├── types.go            # ServerConfig, ServerState, Logger (moved from server.go)
└── errors.go           # Error codes 3001-3099 (unchanged)
```

### Files Removed

- `transport.go` (stdio NDJSON transport)
- `transport_test.go` (stdio transport tests)
- `server.go` (stdio server lifecycle)
- `server_test.go` (stdio server tests)

### Superseded Constraint

~~"stdio transport only (no HTTP for MCP)"~~ → HTTP transport only at `/mcp` endpoint

### Migration Path

Users with existing stdio configuration should:

1. Remove old config: `claude mcp remove wingmate`
2. Start agent: `wingmate --port 9000 --name my-agent`
3. Add HTTP transport: `claude mcp add --transport http wingmate http://localhost:9000/mcp`

See [docs/how/mcp-setup.md](../../docs/how/mcp-setup.md) for detailed migration guide.

---

<!--
MACHINE-READABLE CONTEXT
========================
This section provides structured metadata for AI assistants to quickly parse ADR context.

```yaml
adr:
  id: 4
  title: "MCP Server Implementation Approach"
  status: "decided"
  date: "2026-01-22"

  decision_summary: "Use official modelcontextprotocol/go-sdk v1.1.0 for MCP server implementation"

  affects:
    - component: "internal/mcp"
      impact: "high"
    - component: "cmd/wingmate/main.go"
      impact: "medium"
    - component: "go.mod"
      impact: "medium"

  constraints:
    - "Use modelcontextprotocol/go-sdk v1.1.0 (no other MCP libraries)"
    - "MCP error codes must be in range 3001-3099"
    - "All MCP tool invocations must log to Flight Log"
    - "Use adapter pattern to isolate SDK API changes"
    - "HTTP transport only at /mcp endpoint (stdio removed 2026-01-24)"

  depends_on:
    - 1  # Language choice (Go)
    - 2  # Flight Log storage

  dependents: []

  principles:
    - "P1"  # Observability First
    - "P4"  # MCP sanctioned
    - "P6"  # Security by Design

  implementation:
    status: "complete"
    location: "internal/mcp/"

  amendments:
    - date: "2026-01-24"
      change: "HTTP Transport Migration"
      summary: "Replaced stdio transport with HTTP at /mcp endpoint"
      plan: "docs/plans/004-mcp-http-transport/"

  tags:
    - "mcp"
    - "sdk"
    - "integration"
    - "claude-code"

  complexity_impact:
    score_delta: 1
    reason: "Adds new external dependency and protocol layer"
```
-->
