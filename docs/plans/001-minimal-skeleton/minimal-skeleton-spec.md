# Minimal Wingmate Skeleton

**Created**: 2026-01-21
**Status**: Draft
**Mode**: Full

📚 This specification incorporates findings from `docs/research/a2a-research.md`

---

## Research Context

Key findings from A2A protocol research that inform this specification:

- **A2A SDKs**: Official SDKs exist for Python, TypeScript, Java, C#, and Go (all Apache 2.0)
- **Go SDK**: `a2aproject/a2a-go` - less community adoption than Python but protocol is well-specified
- **Core Components**: Agent Card (capability advertisement), A2A Server (receive), A2A Client (send)
- **Protocol**: JSON-RPC 2.0 over HTTPS, SSE for streaming - language-agnostic by design
- **Authentication**: API keys simplest for MVP; OAuth/mTLS for production

See `docs/research/a2a-research.md` for full protocol analysis.

---

## Summary

**WHAT**: Create a minimal working Wingmate system with one Pilot agent and one Wingmate agent that can communicate bidirectionally using the A2A protocol.

**WHY**: Enable rapid deployment iteration so engineers can:
1. Install Wingmate from git on any machine
2. Start a Pilot or Wingmate agent with a single command
3. Verify two agents can discover and communicate with each other
4. Iterate on features with a working deployment pipeline from day one

This establishes the foundation for all future Wingmate development by proving the core communication path works before adding complexity.

---

## Goals

1. **Single binary distribution**: Download one binary, run - no runtime dependencies
2. **Cross-platform builds**: Compile for macOS, Linux, Windows, Android from same codebase
3. **Dual-mode execution**: Same binary runs as Pilot (initiator) or Wingmate (responder)
4. **Basic A2A communication**: Pilot can send a message, Wingmate can respond
5. **Agent Card exposure**: Each agent advertises capabilities at well-known URL
6. **Flight Log basics**: All exchanges logged with timestamp, summary, and payload
7. **Cross-machine verification**: Can test Pilot on Machine A talking to Wingmate on Machine B
8. **Configuration-driven**: Agent identity, port, and peer URL via config file or env vars

---

## Non-Goals

1. **Claude integration**: No LLM calls in MVP; just protocol plumbing
2. **MCP tools**: No local tool access yet
3. **TLS/Authentication**: HTTP-only for MVP (add security in next iteration)
4. **Streaming**: Request-response only; no SSE streaming yet
5. **Multi-agent orchestration**: One Pilot, one Wingmate only
6. **Production hardening**: No rate limiting, no graceful shutdown, minimal error handling
7. **UI/Dashboard**: CLI only
8. **Persistent state**: In-memory only; restarts clear state

---

## Complexity

**Score**: CS-2 (small)

**Breakdown**:
| Factor | Score | Rationale |
|--------|-------|-----------|
| Surface Area (S) | 1 | Multiple files but single module focus |
| Integration (I) | 1 | One external dep (A2A SDK) |
| Data/State (D) | 0 | No persistence, no schema |
| Novelty (N) | 1 | A2A SDK is new to us, but well-documented |
| Non-Functional (F) | 0 | No perf/security requirements for MVP |
| Testing/Rollout (T) | 1 | Integration test needed (two agents talking) |

**Total**: 4 points → **CS-2 (small)**

**Confidence**: 0.80

**Assumptions**:
- A2A Go SDK implements core protocol correctly (or we implement protocol directly)
- No firewall/network issues between test machines
- Go 1.21+ available for building (not required on target machines)

**Dependencies**:
- `a2aproject/a2a-go` SDK (or implement A2A protocol directly - it's just JSON-RPC over HTTP)
- Go standard library (net/http, encoding/json, log)

**Risks**:
- Go SDK less mature than Python (mitigate: protocol is well-specified, can implement directly if needed)
- Less community examples to reference (mitigate: A2A spec is clear, Claude can implement)

**Phases**:
1. Project structure and dependencies
2. Implement Wingmate agent (server/responder)
3. Implement Pilot agent (client/initiator)
4. Add Flight Log
5. Configuration and CLI
6. Integration test (local two-port test)
7. Cross-machine verification

---

## Acceptance Criteria

1. **AC1: Build**
   - GIVEN a machine with Go 1.21+
   - WHEN user runs `git clone <repo> && cd wingmate && go build -o wingmate ./cmd/wingmate`
   - THEN binary compiles without error

2. **AC1b: Binary Distribution**
   - GIVEN a pre-built `wingmate` binary
   - WHEN user copies binary to a fresh machine (no Go installed)
   - THEN binary executes without runtime dependencies

3. **AC2: Start Wingmate**
   - GIVEN the `wingmate` binary exists
   - WHEN user runs `./wingmate --mode wingmate --port 9001`
   - THEN agent starts and logs "Wingmate listening on http://localhost:9001"
   - AND Agent Card is accessible at `http://localhost:9001/.well-known/agent.json`

4. **AC3: Start Pilot**
   - GIVEN the `wingmate` binary exists
   - WHEN user runs `./wingmate --mode pilot --port 9000 --peer http://localhost:9001`
   - THEN agent starts and logs "Pilot ready, peer: http://localhost:9001"

5. **AC4: Basic Communication**
   - GIVEN Pilot and Wingmate are both running (different ports)
   - WHEN Pilot sends message "ping" to Wingmate
   - THEN Wingmate receives message and responds with "pong"
   - AND Pilot receives the response

6. **AC5: Flight Log - Outbound**
   - GIVEN Pilot sends a message
   - THEN Flight Log contains entry with:
     - Timestamp
     - Direction: "outbound"
     - Summary: human-readable description
     - Payload: the actual message content

7. **AC6: Flight Log - Inbound**
   - GIVEN Wingmate receives a message
   - THEN Flight Log contains entry with:
     - Timestamp
     - Direction: "inbound"
     - Summary: human-readable description
     - Payload: the actual message content

8. **AC7: Agent Card Content**
   - GIVEN Wingmate is running
   - WHEN fetching `/.well-known/agent.json`
   - THEN response contains valid JSON with:
     - `name`: agent name
     - `url`: agent endpoint URL
     - `capabilities`: list (can be minimal for MVP)

9. **AC8: Cross-Machine Communication**
   - GIVEN Wingmate running on Machine B at `http://machineB:9001`
   - AND Pilot running on Machine A with `--peer http://machineB:9001`
   - WHEN Pilot sends "ping"
   - THEN Wingmate on Machine B receives and responds
   - AND Pilot on Machine A receives "pong"

10. **AC9: Configuration**
    - GIVEN a config file `config/agent.json` exists
    - WHEN starting agent without CLI flags
    - THEN agent reads configuration from file
    - AND environment variables override config file values

11. **AC10: Cross-Compile**
    - GIVEN the source code and Go 1.21+
    - WHEN running `GOOS=linux GOARCH=amd64 go build`
    - THEN a Linux binary is produced (even from macOS/Windows)

---

## Risks & Assumptions

### Risks

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Go A2A SDK incomplete | Medium | Medium | Protocol is JSON-RPC over HTTP; can implement directly |
| Network/firewall blocks cross-machine | Medium | Medium | Document required ports; test early |
| Go learning curve | Low | Low | Claude writes the code; Go is straightforward |

### Assumptions

1. Go 1.21+ available on build machine (not needed on target machines)
2. Engineers can open ports on their machines for testing
3. A2A protocol spec is sufficient to implement without SDK if needed
4. JSON-RPC over HTTP is sufficient (no need for WebSocket)
5. File-based Flight Log is acceptable for MVP
6. Single binary per platform is acceptable (no universal binary needed)

---

## Open Questions

*All questions resolved in clarification session 2026-01-21.*

1. ~~**Q1**: Should Flight Log be stdout, file, or both?~~
   - **RESOLVED**: JSONL file + optional stdout via `--verbose` (see ADR-002)

2. ~~**Q2**: Use A2A Go SDK or implement protocol directly?~~
   - **RESOLVED**: Try SDK first; fall back to direct implementation if incomplete
   - A2A is just JSON-RPC 2.0 over HTTP - can implement directly if needed

3. ~~**Q3**: Config file format - JSON, YAML, or TOML?~~
   - **RESOLVED**: JSON (no extra dependencies, matches Agent Card format)

4. ~~**Q4**: Build system - Makefile, Task, or just `go build`?~~
   - **RESOLVED**: Makefile for cross-compilation targets

---

## ADR Seeds (Optional)

### ADR-001: Language Choice

**Status**: DECIDED - **Go**

**Decision Drivers**:
- Single binary distribution (no runtime dependencies on target machines)
- Cross-platform compilation (macOS, Linux, Windows, Android from one codebase)
- Long-term maintainability (Claude writes the code; verbosity is non-issue)
- "It should just work" - copy binary, run it

**Candidate Alternatives**:
- A: Python - Most mature A2A SDK, widely known
- B: TypeScript - Good SDK, better for web-based tooling later
- C: **Go** - Fast, single binary, cross-compiles to any platform ✓

**Decision**: Go (Option C)

**Rationale**:
- Primary goal is rapid deployment iteration - single binary eliminates "install Python/Node" friction
- Android support (mentioned in original requirements) is native with Go cross-compilation
- A2A protocol is well-specified JSON-RPC over HTTP - can implement directly if Go SDK is incomplete
- Claude writes the code, so Go's verbosity is irrelevant
- Long-term investment: binary distribution scales better than interpreted languages

**Stakeholders**: Core maintainers

### ADR-002: Flight Log Storage

**Status**: DECIDED - **JSONL + optional stdout (A+C hybrid)**

**Decision Drivers**:
- Simplicity for MVP
- Claude needs to read logs to answer "where are we at?"
- Engineers will `tail -f` to watch agents communicate
- No external dependencies preferred

**Candidate Alternatives**:
- A: Append-only JSON lines file (.jsonl)
- B: SQLite database
- C: Structured log to stdout (12-factor style)

**Decision**: A+C Hybrid

**Implementation**:
- Default: Write to `.jsonl` file (one JSON object per line)
- With `--verbose`: Also mirror to stdout in real-time
- File path configurable via `--log` flag or config

**Rationale**:
- JSONL is human readable (`cat`, `tail -f`, `grep`) and machine parseable
- Zero dependencies - just file I/O
- Claude can read the file to understand conversation history
- stdout mirroring enables live watching during debugging
- Tiny implementation overhead (basically a tee pattern)
- Can add SQLite later if query needs grow

**Format**:
```json
{"ts":"2026-01-21T10:30:00Z","trace":"abc123","pilot":"p1","wingmate":"w1","dir":"out","summary":"Requested CPU metrics","payload":{...}}
```

**Stakeholders**: Engineers using Wingmate for debugging

---

## Testing Strategy

**Approach**: Full TDD
**Rationale**: This is the foundation of the project - establishing good testing patterns now pays dividends later.

**Focus Areas**:
- A2A protocol compliance (JSON-RPC request/response format)
- Agent Card serving and parsing
- Flight Log entry creation and formatting
- Pilot→Wingmate communication flow
- Cross-machine connectivity (integration tests)

**Excluded**:
- Performance benchmarking (not MVP scope)
- Security/auth testing (not MVP scope)

**Mock Usage**: Targeted mocks
- Mock external HTTP calls when testing protocol formatting
- Use real agent-to-agent communication for integration tests
- Never mock the Flight Log (always test real file I/O)

**TDD Workflow**:
1. Write failing test for expected behavior
2. Implement minimum code to pass
3. Refactor while keeping tests green
4. Integration tests run real Pilot↔Wingmate on separate ports

---

## Documentation Strategy

**Location**: README.md only
**Rationale**: MVP needs quick-start essentials. Detailed docs can come later once patterns stabilize.

**Target Audience**: Engineers who want to:
- Build from source
- Run Pilot and Wingmate agents
- Verify they can communicate
- Understand the Flight Log format

**README Content**:
- Project overview (what is Wingmate)
- Prerequisites (Go 1.21+)
- Build instructions (`go build` / `make`)
- Quick start (run Wingmate, run Pilot, see ping/pong)
- Configuration basics
- Flight Log location and format

**Maintenance**: Update README when CLI flags or config format changes.

---

## Unresolved Research

*None - existing A2A research in `docs/research/a2a-research.md` provides sufficient context for MVP.*

---

## Clarifications

### Session 2026-01-21

| Question | Decision | Rationale |
|----------|----------|-----------|
| Workflow Mode | **Full** | Foundation of project - want comprehensive documentation |
| Testing Approach | **Full TDD** | Establish good patterns from the start |
| Mock Usage | **Targeted** | Mock externals only; real agent-to-agent tests |
| Documentation | **README.md only** | Quick-start essentials for MVP |
| Config Format | **JSON** | No deps, matches Agent Card format |
| Build System | **Makefile** | Standard for Go cross-compilation |
| A2A SDK vs Direct | **Try SDK first** | Fall back to direct implementation if needed |

---

## Notes

This specification intentionally keeps scope minimal to achieve the primary goal: **a working deployment pipeline that enables rapid iteration**. Features like Claude integration, security, and streaming will be added in subsequent iterations once the basic communication path is proven.

The mantra for this phase: "Make it work, then make it right, then make it fast."
