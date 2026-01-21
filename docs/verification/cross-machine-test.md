# Cross-Machine Verification Test Procedure

This document describes the verification procedure for testing Wingmate communication across network boundaries.

## Prerequisites

1. Two machines on the same network (or with network access between them)
2. Wingmate binary built for each target platform
3. Network ports open (default: 9001, 9002)
4. IP addresses known for each machine

## Setup

### Machine A

1. Get Machine A's IP address:
   ```bash
   # macOS/Linux
   ifconfig | grep "inet " | grep -v 127.0.0.1

   # or
   ip addr show | grep "inet "
   ```

2. Transfer the binary (if needed):
   ```bash
   # Build for target platform
   make build-all

   # Transfer
   scp bin/wingmate-linux-amd64 user@machineA:/home/user/wingmate
   ```

3. Open firewall port:
   ```bash
   # Linux (ufw)
   sudo ufw allow 9001/tcp

   # macOS (disable or configure)
   # System Preferences > Security > Firewall > Options
   ```

### Machine B

Same process as Machine A, using different port if on same machine.

## Test 1: Unidirectional Ping (Machine B to Machine A)

### On Machine A (start agent)

```bash
./wingmate --name machine-a --port 9001 --verbose
```

Expected output:
```
Agent "machine-a" listening on http://0.0.0.0:9001
Agent Card: http://0.0.0.0:9001/.well-known/agent.json
Press Ctrl+C to stop
```

### On Machine B (send ping)

```bash
./wingmate ping http://<machine-a-ip>:9001 --name machine-b --verbose
```

Expected output:
```
Sending ping to http://192.168.1.100:9001...
Received pong!
```

### Verification

- Machine B shows "Received pong!"
- Machine A shows incoming connection in verbose output (if enabled)
- Exit code is 0

## Test 2: Agent Card Access

### On Machine B

```bash
curl http://<machine-a-ip>:9001/.well-known/agent.json
```

Expected output:
```json
{
  "name": "machine-a",
  "version": "1.0.0",
  "description": "A2A Agent",
  "url": "http://0.0.0.0:9001",
  "capabilities": {
    "streaming": false,
    "pushNotifications": false
  },
  "skills": [
    {
      "name": "ping",
      "description": "Respond to ping with pong"
    }
  ]
}
```

### Verification

- Valid JSON returned
- `name` matches agent name
- `url` shows listening address

## Test 3: Bidirectional Communication (ADR-003 Validation)

This test validates that any agent can initiate communication with any other agent (unified peer model).

### On Machine B (start agent)

```bash
./wingmate --name machine-b --port 9002 --verbose
```

### On Machine A (ping Machine B)

```bash
./wingmate ping http://<machine-b-ip>:9002 --name machine-a --verbose
```

Expected output:
```
Sending ping to http://192.168.1.101:9002...
Received pong!
```

### Verification

- Machine A successfully pings Machine B (reverse direction)
- Both agents can act as either pilot or wingmate
- This validates the unified peer architecture from ADR-003

## Test 4: Flight Log Correlation

### Check Flight Logs

On Machine A:
```bash
cat flight.jsonl | jq .
```

Expected entries (as wingmate - responded to ping):
```json
{
  "ts": "2026-01-21T10:00:00Z",
  "agent": "machine-a",
  "peer": "machine-b",
  "dir": "inbound",
  "role": "wingmate",
  "summary": "Received message",
  "payload": {...}
}
```

On Machine B:
```bash
cat flight.jsonl | jq .
```

Expected entries (as pilot - initiated ping):
```json
{
  "ts": "2026-01-21T10:00:00Z",
  "agent": "machine-b",
  "peer": "machine-a",
  "dir": "outbound",
  "role": "pilot",
  "summary": "Sending ping",
  "payload": {...}
}
```

### Verification

- Machine A's log shows `role: wingmate` (responded to conversation)
- Machine B's log shows `role: pilot` (initiated conversation)
- Timestamps are close (within seconds)
- Trace IDs match if present

## Success Criteria

All tests pass when:

| Test | Expected Result | Status |
|------|-----------------|--------|
| Unidirectional ping | "Received pong!" | [ ] |
| Agent Card access | Valid JSON | [ ] |
| Bidirectional ping | Both directions work | [ ] |
| Flight Log roles | Correct pilot/wingmate | [ ] |

## Common Issues

See [troubleshooting.md](troubleshooting.md) for solutions to common problems.

## Test Record

| Date | Machine A | Machine B | Result | Notes |
|------|-----------|-----------|--------|-------|
| | | | | |

Fill in this table when running tests for record-keeping.
