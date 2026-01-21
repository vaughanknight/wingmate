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
│   ├── protocol/             # A2A JSON-RPC implementation
│   └── flightlog/            # Observability logging
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

**Reading ADRs efficiently**: Each ADR has a machine-readable YAML block at the bottom under `MACHINE-READABLE CONTEXT`. Parse this section for quick structured context about constraints, affected components, and implementation status.

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

# Run tests
go test ./...
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
