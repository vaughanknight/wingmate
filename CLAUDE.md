# CLAUDE.md - Wingmate Project Context

This file provides essential context for Claude Code when working on the Wingmate project.

## Project Overview

Wingmate enables Claude instances on different machines to communicate bidirectionally for collaborative debugging. It implements the A2A (Agent-to-Agent) protocol.

**Key Terminology**:
- **Pilot**: The agent that **initiated** a specific conversation (role, not mode)
- **Wingmate**: The agent **responding** in a specific conversation (role, not mode)
- **Agent**: A running instance that can be either pilot or wingmate depending on who initiated
- **Flight Log**: Observability log of agent conversations
- **Mission**: A debugging session or task
- **Sortie**: A single request-response exchange

## Architecture Quick Reference

```
wingmate/
├── cmd/wingmate/main.go      # Single binary entry point
├── internal/
│   ├── agent/                # Unified agent (can be pilot OR wingmate per conversation)
│   ├── flightlog/            # Observability logging
│   ├── llm/                  # LLM integration (Claude CLI executor)
│   ├── mcp/                  # MCP server for Claude Code integration
│   └── protocol/             # A2A JSON-RPC implementation
├── pkg/types/                # Public types (AgentCard, etc.)
└── config/                   # Configuration templates
```

**Note**: There are no separate `internal/pilot/` or `internal/wingmate/` directories. Per [ADR-003](docs/adr/003-unified-peer-architecture.md), every agent instance is a full peer that can initiate or respond to conversations.

## Critical Decisions (ADRs)

Before making changes, consult the Architecture Decision Records in `docs/adr/`:

| ADR | Decision | Key Constraint |
|-----|----------|----------------|
| [001](docs/adr/001-language-choice.md) | **Go** | Single binary, cross-platform compilation |
| [002](docs/adr/002-flight-log-storage.md) | **JSONL + stdout** | File at `./flight.jsonl`, `--verbose` for stdout |
| [003](docs/adr/003-unified-peer-architecture.md) | **Unified Peers** | No `--mode` flag; pilot/wingmate are conversation roles |
| [004](docs/adr/004-mcp-server-implementation.md) | **MCP Server** | HTTP transport at /mcp, Flight Log integration, 3 tools |

**Reading ADRs efficiently**: Each ADR has a machine-readable YAML block at the bottom under `MACHINE-READABLE CONTEXT`. Parse this section for quick structured context about constraints, affected components, and implementation status.

## MCP Server

Wingmate exposes MCP (Model Context Protocol) tools via HTTP at the `/mcp` endpoint. This enables Claude Code to invoke Wingmate tools directly.

### Running MCP Mode

MCP is available automatically when running the agent:

```bash
# Start agent (MCP available at http://localhost:9000/mcp)
wingmate --port 9000 --name my-agent
```

### Configuring Claude Code

```bash
claude mcp add --transport http wingmate http://localhost:9000/mcp
```

### Tools Exposed

| Tool | Purpose |
|------|---------|
| `wingmate_chat` | Send messages through LLM integration |
| `wingmate_status` | Get operational status and health |
| `wingmate_discover` | Discover available peer agents |

### MCP Error Codes

When working on MCP-related code, use these error codes (`internal/mcp/errors.go`):

| Code | Name | When to Use |
|------|------|-------------|
| 3001 | MCPInvalidRequest | Malformed JSON-RPC request |
| 3002 | MCPMethodNotFound | Unknown method/tool called |
| 3003 | MCPInvalidParams | Invalid tool parameters |
| 3004 | MCPInternalError | Unrecoverable server error |
| 3010 | MCPToolNotFound | Tool not registered |
| 3011 | MCPToolExecutionFailed | Tool handler returned error |
| 3020 | MCPTransportError | HTTP I/O error |
| 3021 | MCPParseError | Cannot parse JSON message |
| 3022 | MCPShutdown | Server shutting down |

### Key MCP Files

| File | Purpose |
|------|---------|
| `internal/mcp/http_transport.go` | HTTP handler implementing http.Handler |
| `internal/mcp/localhost.go` | Localhost-only middleware (returns 403) |
| `internal/mcp/session.go` | Session manager with 30-min TTL |
| `internal/mcp/tools.go` | Tool definitions and schema |
| `internal/mcp/handlers.go` | Tool execution handlers with PeerProvider |
| `internal/mcp/types.go` | Protocol types (ServerConfig, ServerState, etc.) |
| `internal/mcp/errors.go` | Error codes and MCPError type |

### Configuration

Environment variables for MCP:

| Variable | Description | Default |
|----------|-------------|---------|
| `WINGMATE_LLM_MODEL` | Claude model for chat tool | CLI default |
| `WINGMATE_LLM_TIMEOUT` | Request timeout (seconds) | `120` |

For detailed setup, see [docs/how/mcp-setup.md](docs/how/mcp-setup.md).

## Constitution Principles

The project follows 6 core principles in `docs/project-rules/constitution.md`:

| ID | Principle | Quick Summary |
|----|-----------|---------------|
| P1 | Observability First | Every conversation logged to Flight Log |
| P2 | Deploy Early, Deploy Often | Installation/upgrade mechanisms first |
| P3 | Engineer Autonomy | Manual git install acceptable, explicit config |
| P4 | Protocol Compliance | A2A protocol, MCP for local tools |
| P5 | Themed but Tasteful | Wingmate terminology, but clarity > cleverness |
| P6 | Security by Design | TLS, auth, audit logging required |

## Development Rules

From `docs/project-rules/rules.md`:

### No Time Estimates
Use Complexity Score (CS 1-5) only. Never estimate hours/days.

### Testing (TDD Required)
- Test docs must include: Why, Contract, Usage Notes, Quality Contribution
- Scratch tests in `tests/scratch/` (not in CI)
- Promote using CROP heuristic: Critical, Regression-prone, Opaque, edge-case Prone

### Code Standards
- Files: lowercase-hyphenated (`agent-executor.go`)
- Use Wingmate terminology in public APIs
- Never log sensitive data

## Common Commands

```bash
# Build
go build -o wingmate ./cmd/wingmate

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o wingmate-linux ./cmd/wingmate

# Run an agent (listens on port, connects to peers)
./wingmate --port 9000 --peers http://localhost:9001

# Run standalone (no peers, just accepts incoming)
./wingmate --port 9001

# Agent also serves MCP at /mcp endpoint
# Configure Claude Code: claude mcp add --transport http wingmate http://localhost:9001/mcp

# Run tests
go test ./...

# Run tests with race detector
go test ./... -race
```

## Key File Locations

| Purpose | Location |
|---------|----------|
| Constitution | `docs/project-rules/constitution.md` |
| Rules | `docs/project-rules/rules.md` |
| Idioms | `docs/project-rules/idioms.md` |
| Architecture | `docs/project-rules/architecture.md` |
| ADRs | `docs/adr/*.md` |
| Feature Specs | `docs/plans/*/` |
| Research | `docs/research/` |
| MCP Setup Guide | `docs/how/mcp-setup.md` |
| LLM Setup Guide | `docs/how/llm-setup.md` |

## Before Making Changes

1. **Check ADRs**: Read relevant ADRs for constraints
2. **Follow Constitution**: Ensure changes align with principles
3. **TDD**: Write tests first for non-trivial changes
4. **Flight Log**: All agent communication must be logged
5. **No Secrets**: Never log tokens, credentials, PII

## ADR Quick Parse Guide

When you need fast context from ADRs, look for the `MACHINE-READABLE CONTEXT` section at the bottom of each file. It contains YAML with:

```yaml
adr:
  decision_summary: "One-line summary"
  constraints: ["List of constraints"]
  affects: [{component, impact}]
  implementation: {status, location}
```

This lets you quickly understand decisions without reading full prose.
