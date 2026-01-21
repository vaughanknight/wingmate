# ADR-001: Language Choice

**Status**: DECIDED
**Date**: 2026-01-21
**Deciders**: Core maintainers
**Supersedes**: N/A
**Superseded by**: N/A

---

## Context

Wingmate needs to be distributed as a tool that engineers can quickly install and run on any machine for debugging purposes. The A2A protocol has official SDKs available in Python, TypeScript, Java, C#, and Go. We need to choose a language that balances development speed, distribution simplicity, and long-term maintainability.

Key consideration: Claude writes the code, so developer familiarity and verbosity are non-issues. What matters is the end-user experience and operational characteristics.

---

## Decision Drivers

- Must compile to single binary (no runtime dependencies on target machines)
- Cross-platform compilation required (macOS, Linux, Windows, Android from one codebase)
- "It should just work" - copy binary, run it
- A2A protocol is well-specified JSON-RPC over HTTP; can implement directly if SDK incomplete
- Long-term maintainability over short-term velocity

---

## Options Considered

### Option A: Python

**Description**: Use Python with the most mature A2A SDK.

**Pros**:
- Most mature and battle-tested A2A SDK
- Widely known, extensive ecosystem
- Rapid prototyping

**Cons**:
- Requires Python runtime on target machines
- Packaging for distribution (PyInstaller, etc.) adds complexity
- Cross-platform distribution is non-trivial
- Android support requires additional tooling

### Option B: TypeScript

**Description**: Use TypeScript/Node.js with good A2A SDK support.

**Pros**:
- Good A2A SDK available
- Strong typing with TypeScript
- Better positioned for web-based tooling later

**Cons**:
- Requires Node.js runtime on target machines
- Packaging (pkg, nexe) has limitations
- Larger binary sizes when packaged
- Android support limited

### Option C: Go

**Description**: Use Go with the a2aproject/a2a-go SDK.

**Pros**:
- Single binary compilation with zero dependencies
- Native cross-compilation to any platform (`GOOS=linux GOARCH=arm`)
- Android support built into Go toolchain
- Fast startup, low memory footprint
- Strong concurrency primitives for agent communication
- Static typing catches errors at compile time

**Cons**:
- Less community adoption for A2A Go SDK
- Go is more verbose than Python
- Team may have less Go experience

---

## Decision

**Chosen Option**: Go (Option C)

Go's single-binary distribution model directly serves Constitution Principle P2 (Deploy Early, Deploy Often) and P3 (Engineer Autonomy). Engineers can download one file and run it immediately, without installing runtimes or managing dependencies.

The A2A protocol is fundamentally JSON-RPC 2.0 over HTTP - a protocol that's straightforward to implement in any language. If the Go SDK proves incomplete, direct implementation is a viable fallback since the protocol is well-specified.

Since Claude writes the code, Go's verbosity is irrelevant. The focus is on the binary distribution model that enables rapid iteration across multiple machines.

---

## Consequences

### Positive

- Zero-dependency binaries enable frictionless installation
- Single `go build` command produces platform-specific binary
- Android deployment via `GOOS=android GOARCH=arm64`
- Fast compilation supports rapid iteration
- Strong stdlib (net/http, encoding/json) reduces external dependencies

### Negative

- May need to implement A2A protocol directly if SDK lacks features (mitigation: protocol is well-documented)
- Fewer A2A Go examples in community (mitigation: Python examples translate conceptually)

### Neutral

- Different development experience than typical web projects
- Binary size larger than scripted languages (but still reasonable, ~10-20MB)

---

## Implementation Notes

- Use Go 1.21+ for consistent module behavior
- Standard project layout: `cmd/wingmate/`, `internal/`, `pkg/`
- Makefile for cross-compilation targets
- Consider `go:embed` for static assets if needed later

---

## Related Decisions

- [ADR-002](./002-flight-log-storage.md): Flight Log storage format (informed by Go's excellent file I/O)
- Constitution Principle: P2 (Deploy Early), P3 (Engineer Autonomy)

---

## References

- [A2A Go SDK](https://github.com/a2aproject/a2a-go) - Apache 2.0 licensed
- [A2A Protocol Specification](https://github.com/a2aproject/A2A/blob/main/specification/json/a2a.json)
- [Go Cross-Compilation Guide](https://go.dev/doc/install/source#environment)

---

<!--
MACHINE-READABLE CONTEXT
========================
This section provides structured metadata for AI assistants to quickly parse ADR context.

```yaml
adr:
  id: 1
  title: "Language Choice"
  status: "decided"
  date: "2026-01-21"

  decision_summary: "Use Go for single-binary distribution and native cross-platform compilation"

  affects:
    - component: "all"
      impact: "high"
    - component: "build-system"
      impact: "high"
    - component: "a2a-integration"
      impact: "medium"

  constraints:
    - "All code must be written in Go"
    - "Must use Go 1.21+ for module compatibility"
    - "Standard library preferred over external dependencies"
    - "Project follows standard Go layout (cmd/, internal/, pkg/)"

  depends_on: []

  dependents:
    - 2  # Flight Log storage leverages Go file I/O

  principles:
    - "P2"  # Deploy Early, Deploy Often
    - "P3"  # Engineer Autonomy

  implementation:
    status: "not_started"
    location: "/"

  tags:
    - "infrastructure"
    - "foundational"
    - "language"

  complexity_impact:
    score_delta: 0
    reason: "Language choice is foundational, complexity neutral"
```
-->
