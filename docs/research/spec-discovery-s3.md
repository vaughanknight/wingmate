# Spec Discovery Document S3: Ambiguities and Edge Cases

**Created**: 2026-01-21
**Source**: `docs/plans/001-minimal-skeleton/minimal-skeleton-spec.md`
**Research Context**: `docs/research/a2a-research.md`

---

## Summary

This document identifies specification gaps, ambiguities, and edge cases in the Minimal Wingmate Skeleton spec that require clarification or design decisions before implementation begins.

---

### Discovery S3-01: Configuration File Missing Behavior Undefined

**Category**: Ambiguity
**Impact**: High
**Spec Reference**:
> "AC9: GIVEN a config file `config/agent.json` exists WHEN starting agent without CLI flags THEN agent reads configuration from file AND environment variables override config file values"

**Issue**: The spec defines behavior when the config file exists but does not specify what happens when:
1. The config file path is specified but file is missing
2. The config file exists but contains invalid JSON
3. The config file exists but is missing required fields
4. No config file, no env vars, and no CLI flags are provided

The spec also doesn't define which fields are required vs. optional in the config file.

**Recommendation**: Define explicit fallback behavior:
- If config file specified via `--config` flag but missing: ERROR and exit with clear message
- If default path (`config/agent.json`) missing: WARN and continue with defaults/env/CLI
- If config file malformed JSON: ERROR and exit with parse error location
- If required fields missing: ERROR listing which fields are required
- Define minimum required fields: `mode` (or require via CLI)

**Example**:
```bash
# Config file specified but missing - should error
./wingmate --config /nonexistent/config.json
# Error: Config file not found: /nonexistent/config.json

# No config, no env, no CLI flags - what happens?
./wingmate
# Should either: (a) error with usage, or (b) start with sensible defaults
```

---

### Discovery S3-02: Port Conflict Handling Not Specified

**Category**: Edge Case
**Impact**: High
**Spec Reference**:
> "WHEN user runs `./wingmate --mode wingmate --port 9001` THEN agent starts and logs 'Wingmate listening on http://localhost:9001'"

**Issue**: The spec assumes the port is available. It does not specify behavior when:
1. The specified port is already in use
2. The port requires elevated privileges (ports < 1024)
3. The port number is invalid (negative, > 65535, non-numeric)

**Recommendation**: Define explicit error handling:
- Port in use: ERROR with message "Port 9001 already in use. Check for other Wingmate instances or specify --port <different_port>"
- Privileged port without permissions: ERROR with message "Port 80 requires elevated privileges"
- Invalid port: ERROR with message "Invalid port: must be 1-65535"
- Consider: auto-increment to find available port with `--port auto` option (future enhancement)

**Example**:
```bash
# First instance
./wingmate --mode wingmate --port 9001  # Success

# Second instance same port
./wingmate --mode pilot --port 9001
# Error: Port 9001 already in use (EADDRINUSE)
```

---

### Discovery S3-03: Peer Agent Unreachable Scenarios Undefined

**Category**: Edge Case
**Impact**: Critical
**Spec Reference**:
> "AC4: GIVEN Pilot and Wingmate are both running WHEN Pilot sends message 'ping' to Wingmate THEN Wingmate receives message and responds with 'pong'"

**Issue**: The spec assumes the happy path where the peer is reachable. It does not address:
1. Peer URL resolves but connection refused (peer not running)
2. Peer URL does not resolve (DNS failure)
3. Peer responds with non-A2A response (wrong service at URL)
4. Peer Agent Card fetch fails
5. Peer Agent Card malformed or incompatible version
6. Network timeout during communication
7. Peer responds with A2A error (unauthorized, method not found, etc.)

**Recommendation**: Define error handling for each scenario:
- Connection refused: "Cannot connect to peer at http://localhost:9001 - is Wingmate running?"
- DNS failure: "Cannot resolve host: machineB"
- Invalid response: "Peer at http://localhost:9001 is not an A2A agent (unexpected response format)"
- Agent Card issues: "Peer Agent Card invalid or incompatible"
- Timeout: "Request to peer timed out after 30s"
- A2A errors: Pass through error message from peer

Also define: Should Pilot retry on failure? How many times? With what backoff?

**Example**:
```bash
# Pilot starts before Wingmate
./wingmate --mode pilot --port 9000 --peer http://localhost:9001
# Option A: Error immediately "Peer not reachable"
# Option B: Warn and wait/retry "Waiting for peer at http://localhost:9001..."
```

---

### Discovery S3-04: Agent Card Schema Not Fully Specified

**Category**: Ambiguity
**Impact**: Medium
**Spec Reference**:
> "AC7: WHEN fetching `/.well-known/agent-card.json` THEN response contains valid JSON with: `name`: agent name, `url`: agent endpoint URL, `capabilities`: list (can be minimal for MVP)"

**Issue**: The spec provides a minimal schema but leaves many questions:
1. What is the exact JSON structure (flat vs nested)?
2. What are valid capability entries for MVP?
3. Should the card include protocol version?
4. Should it include authentication requirements (even if "none" for MVP)?
5. What URL should be in the card - the agent's own URL or the A2A endpoint URL?
6. Should there be a description field?

The A2A research doc shows the standard Agent Card is much richer, including: description, version, skills/operations, modalities, and auth schemes.

**Recommendation**: Define a concrete MVP Agent Card schema that is A2A-compatible:

```json
{
  "name": "wingmate-001",
  "description": "Wingmate diagnostic agent",
  "url": "http://localhost:9001",
  "version": "0.1.0",
  "protocolVersion": "0.3.0",
  "capabilities": [
    {
      "name": "ping",
      "description": "Respond to ping with pong"
    }
  ],
  "authentication": {
    "schemes": []
  }
}
```

**Example**: Difference in interpretation:
```json
// Minimal interpretation (too minimal?)
{"name": "agent", "url": "http://x", "capabilities": []}

// A2A-compliant interpretation (recommended)
{"name": "agent", "url": "http://x", "version": "0.1.0", "protocolVersion": "0.3.0", "capabilities": [{"name": "ping"}], "authentication": {"schemes": []}}
```

---

### Discovery S3-05: Flight Log Entry Format Partially Specified

**Category**: Ambiguity
**Impact**: Medium
**Spec Reference**:
> ADR-002 Format: `{"ts":"2026-01-21T10:30:00Z","trace":"abc123","pilot":"p1","wingmate":"w1","dir":"out","summary":"Requested CPU metrics","payload":{...}}`

**Issue**: The example format raises several questions:
1. What is "trace" - a correlation ID? How is it generated/propagated?
2. "pilot" and "wingmate" fields - what values go here? Names? URLs? IDs?
3. What if the agent is standalone (no peer yet)? Are pilot/wingmate fields required?
4. What is the "payload" structure for different message types?
5. Is "dir" always "out"/"in" or are there other values like "internal"/"error"?
6. Should errors be logged with a different structure?
7. What timezone for timestamps - always UTC (the Z suffix suggests yes)?
8. Line length limits for JSONL?

**Recommendation**: Define a complete schema:

```json
{
  "ts": "2026-01-21T10:30:00.123Z",  // ISO8601 UTC with milliseconds
  "traceId": "abc123",               // UUID, generated per conversation/task
  "level": "info",                   // info | warn | error
  "agent": "wingmate-001",           // This agent's name
  "peer": "pilot-001",               // Peer agent name (null if none)
  "direction": "inbound",            // inbound | outbound | internal
  "method": "ping",                  // A2A method name
  "summary": "Received ping request",
  "payload": {},                     // Full request/response body
  "error": null                      // Error details if level=error
}
```

**Example**: Ambiguous current spec vs. recommended:
```jsonl
// Current spec - what goes in pilot/wingmate?
{"ts":"...","trace":"x","pilot":"?","wingmate":"?","dir":"out","summary":"...","payload":{}}

// Recommended - clear semantics
{"ts":"...","traceId":"x","agent":"pilot-001","peer":"wingmate-001","direction":"outbound","method":"ping","summary":"Sent ping","payload":{"message":"ping"}}
```

---

### Discovery S3-06: Configuration Precedence Not Fully Defined

**Category**: Ambiguity
**Impact**: Medium
**Spec Reference**:
> "AC9: environment variables override config file values"

**Issue**: The spec mentions CLI flags, env vars, and config file but doesn't fully define:
1. What is the complete precedence order? (Assumed: CLI > env > file > defaults)
2. What are the env var names? (e.g., `WINGMATE_PORT`, `WINGMATE_MODE`?)
3. What are all the configurable values and their defaults?
4. Are there config file-only options? CLI-only options?
5. How are nested config values handled in env vars?
6. Is there a `--config` flag to specify config file path?

**Recommendation**: Define explicit precedence and naming:

| Setting | CLI Flag | Env Var | Config Key | Default |
|---------|----------|---------|------------|---------|
| Mode | `--mode` | `WINGMATE_MODE` | `mode` | (required) |
| Port | `--port` | `WINGMATE_PORT` | `port` | `9000` |
| Peer URL | `--peer` | `WINGMATE_PEER` | `peer` | (none) |
| Log File | `--log` | `WINGMATE_LOG` | `logFile` | `./flight.jsonl` |
| Verbose | `--verbose` | `WINGMATE_VERBOSE` | `verbose` | `false` |
| Config | `--config` | (n/a) | (n/a) | `./config/agent.json` |

Precedence: CLI > Env > Config File > Defaults

**Example**:
```bash
# All three sources conflict - CLI wins
WINGMATE_PORT=8080 ./wingmate --port 9001 --config config.json
# (config.json has port: 7000)
# Result: port = 9001 (CLI wins)
```

---

### Discovery S3-07: "Ping" Semantics in A2A Context Unclear

**Category**: Design Decision
**Impact**: Medium
**Spec Reference**:
> "AC4: WHEN Pilot sends message 'ping' to Wingmate THEN Wingmate receives message and responds with 'pong'"

**Issue**: The spec uses "ping/pong" but doesn't clarify how this maps to A2A protocol:
1. Is "ping" a task (tasks/send) or a message (message/send)?
2. Is there a dedicated health-check endpoint separate from A2A?
3. What is the JSON-RPC structure of a ping request?
4. Should ping be a listed capability in the Agent Card?
5. Is ping synchronous (request/response) or could it be async?

The A2A research suggests message/send for simple exchanges, tasks/send for work with lifecycle.

**Recommendation**: Define ping as a simple A2A message exchange:

```json
// Request (Pilot -> Wingmate)
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "message/send",
  "params": {
    "message": {
      "role": "user",
      "parts": [{"kind": "text", "text": "ping"}]
    }
  }
}

// Response (Wingmate -> Pilot)
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "message": {
      "role": "agent",
      "parts": [{"kind": "text", "text": "pong"}]
    }
  }
}
```

Alternative: Define a non-A2A `/health` endpoint for simple liveness checks, and use A2A for actual ping/pong capability testing.

**Example**: Confusion without clarity:
```bash
# Is this hitting A2A or a health endpoint?
curl http://localhost:9001/ping  # ???

# Or is it a JSON-RPC call?
curl -X POST http://localhost:9001 -d '{"jsonrpc":"2.0","method":"message/send",...}'
```

---

### Discovery S3-08: Graceful Shutdown Not Addressed (Contradiction with Non-Goals)

**Category**: Edge Case / Design Decision
**Impact**: Medium
**Spec Reference**:
> Non-Goals: "No graceful shutdown"

**Issue**: While graceful shutdown is explicitly listed as a non-goal, some minimal handling is still needed for:
1. Ctrl+C (SIGINT) - should the agent log shutdown or just terminate?
2. SIGTERM - same question for container/process manager signals
3. Open connections during shutdown - abrupt close or drain?
4. In-flight requests - fail immediately or complete first?
5. Flight Log file handle - should it be flushed/closed properly?

Even "minimal error handling" implies some signal handling to avoid data corruption (e.g., partial JSONL lines).

**Recommendation**: Implement minimal graceful shutdown for MVP:
- Catch SIGINT/SIGTERM
- Log "Shutting down..." to stdout
- Flush and close Flight Log file
- Close HTTP listener (no new connections)
- Exit with code 0 for clean shutdown, non-zero for errors

This is NOT the full "graceful shutdown" (draining connections, waiting for in-flight) but prevents common issues.

**Example**:
```bash
./wingmate --mode wingmate --port 9001
# Ctrl+C pressed
# Bad: Process killed, last log line truncated
# Better: "Shutting down...", log flushed, clean exit
```

---

## Summary Matrix

| ID | Title | Category | Impact | Resolution Required |
|----|-------|----------|--------|---------------------|
| S3-01 | Config File Missing | Ambiguity | High | Before implementation |
| S3-02 | Port Conflict Handling | Edge Case | High | Before implementation |
| S3-03 | Peer Unreachable | Edge Case | Critical | Before implementation |
| S3-04 | Agent Card Schema | Ambiguity | Medium | Before implementation |
| S3-05 | Flight Log Format | Ambiguity | Medium | Before implementation |
| S3-06 | Config Precedence | Ambiguity | Medium | Before implementation |
| S3-07 | Ping Semantics | Design Decision | Medium | Before implementation |
| S3-08 | Minimal Shutdown | Edge Case | Medium | During implementation |

---

## Recommended Next Steps

1. **Clarification Session**: Review S3-01 through S3-07 with stakeholders to make decisions
2. **Update Spec**: Incorporate decisions into the spec before Phase 1 begins
3. **Add Error Handling AC**: Create explicit acceptance criteria for error scenarios
4. **Define Schemas**: Create formal JSON schemas for Agent Card and Flight Log entries

---

## References

- Source Spec: `/docs/plans/001-minimal-skeleton/minimal-skeleton-spec.md`
- A2A Protocol Research: `/docs/research/a2a-research.md`
- A2A Official Spec: https://a2a-protocol.org/latest/specification/
