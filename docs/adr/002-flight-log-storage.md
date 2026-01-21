# ADR-002: Flight Log Storage

**Status**: DECIDED
**Date**: 2026-01-21
**Deciders**: Core maintainers
**Supersedes**: N/A
**Superseded by**: N/A

---

## Context

Constitution Principle P1 (Observability First) mandates that every agent conversation be traceable. We need a storage mechanism for the Flight Log that:
- Captures both human-readable summaries and raw payloads
- Enables Claude to read logs and understand conversation history
- Allows engineers to watch agent communication in real-time during debugging
- Requires no external dependencies (consistent with ADR-001's single-binary approach)

---

## Decision Drivers

- Simplicity for MVP implementation
- Claude needs to read logs to answer "where are we at?"
- Engineers will `tail -f` to watch agents communicate in real-time
- No external dependencies (no database servers)
- File format must be both human-readable and machine-parseable
- Must support the Flight Log entry structure defined in idioms.md

---

## Options Considered

### Option A: Append-only JSONL File

**Description**: One JSON object per line in a `.jsonl` file.

**Pros**:
- Human readable (`cat`, `tail -f`, `grep` all work)
- Machine parseable (each line is valid JSON)
- Zero dependencies (just file I/O)
- Append-only is simple and safe
- Claude can read and parse easily

**Cons**:
- No query capabilities (must scan entire file)
- File grows unbounded without rotation
- No transactional guarantees

### Option B: SQLite Database

**Description**: Embedded SQL database for structured storage.

**Pros**:
- Query capabilities (filter by time, agent, direction)
- Indexes for fast lookups
- Single file, portable
- Transactional writes

**Cons**:
- Requires SQLite dependency (increases binary size)
- Not human-readable without tooling
- Harder to `tail -f` for live debugging
- More complex implementation

### Option C: Structured Stdout (12-factor Style)

**Description**: Log to stdout, let external tools handle persistence.

**Pros**:
- Simple implementation
- 12-factor compliant
- Easy to pipe to tools (jq, tee, etc.)
- Works well with container orchestration

**Cons**:
- No persistence by default
- Requires user to redirect output
- Can't review logs after session ends without setup
- Claude can't easily "read the log" for context

---

## Decision

**Chosen Option**: A + C Hybrid (JSONL file with optional stdout mirroring)

This combines the benefits of persistent, human-readable files (Option A) with the debugging convenience of live stdout output (Option C).

**Implementation**:
- **Default**: Write all Flight Log entries to a `.jsonl` file
- **With `--verbose`**: Also mirror entries to stdout in real-time
- **File path**: Configurable via `--log` flag or config file

This approach satisfies all decision drivers: Claude can read the file for context, engineers can `tail -f` or use `--verbose` for live monitoring, and there are zero external dependencies.

---

## Consequences

### Positive

- JSONL is grep-friendly: `grep "direction.*outbound" flight.jsonl`
- Claude can read entire conversation history from single file
- Live debugging via `tail -f flight.jsonl` or `--verbose` flag
- Zero dependencies - just Go's standard `os` and `encoding/json`
- Tiny implementation overhead (essentially a tee pattern)
- Future migration to SQLite possible if query needs grow

### Negative

- No built-in log rotation (mitigation: add later or use external logrotate)
- Full file scan for queries (mitigation: acceptable for MVP, can add index later)
- File could grow large (mitigation: document recommended cleanup, add rotation option)

### Neutral

- Engineers need to know about `--verbose` flag for live output
- File location must be documented clearly

---

## Implementation Notes

**Entry Format** (per idioms.md):
```json
{"ts":"2026-01-21T10:30:00Z","trace":"abc123","pilot":"p1","wingmate":"w1","dir":"out","summary":"Requested CPU metrics","payload":{...}}
```

**Required Fields**:
- `ts`: ISO 8601 timestamp
- `trace`: Trace ID for correlation
- `pilot`: Pilot agent identifier
- `wingmate`: Wingmate agent identifier
- `dir`: Direction ("out" for outbound, "in" for inbound)
- `summary`: Human-readable one-line summary
- `payload`: Raw payload (JSON object or escaped string)

**Default File Location**: `./flight.jsonl` (configurable)

**Stdout Mirroring**: When `--verbose` is set, each log entry is written to both file and stdout simultaneously.

---

## Related Decisions

- [ADR-001](./001-language-choice.md): Go's standard library provides excellent file I/O
- Constitution Principle: P1 (Observability First)
- Idioms Reference: Section 3 (Observability Patterns)

---

## References

- [JSON Lines Specification](https://jsonlines.org/)
- [Flight Log Entry Structure](../project-rules/idioms.md#31-flight-log-entry-structure)
- [12-Factor App: Logs](https://12factor.net/logs)

---

<!--
MACHINE-READABLE CONTEXT
========================
This section provides structured metadata for AI assistants to quickly parse ADR context.

```yaml
adr:
  id: 2
  title: "Flight Log Storage"
  status: "decided"
  date: "2026-01-21"

  decision_summary: "Use JSONL file with optional stdout mirroring via --verbose flag"

  affects:
    - component: "flightlog"
      impact: "high"
    - component: "cli"
      impact: "medium"
    - component: "config"
      impact: "low"

  constraints:
    - "Must write to JSONL file by default"
    - "Must support --verbose flag for stdout mirroring"
    - "Must support --log flag for custom file path"
    - "Entry format must match idioms.md specification"
    - "No external dependencies for storage"

  depends_on:
    - 1  # Language choice (Go file I/O)

  dependents: []

  principles:
    - "P1"  # Observability First

  implementation:
    status: "not_started"
    location: "internal/flightlog/"

  tags:
    - "observability"
    - "storage"
    - "logging"

  complexity_impact:
    score_delta: 0
    reason: "JSONL is simpler than alternatives, minimal complexity"

  file_format:
    extension: ".jsonl"
    encoding: "UTF-8"
    default_path: "./flight.jsonl"
    fields:
      required:
        - "ts"       # ISO 8601 timestamp
        - "trace"    # Trace ID
        - "pilot"    # Pilot identifier
        - "wingmate" # Wingmate identifier
        - "dir"      # Direction (out/in)
        - "summary"  # Human-readable summary
        - "payload"  # Raw payload
      optional:
        - "mission"  # Mission ID
        - "parent"   # Parent sortie ID
```
-->
