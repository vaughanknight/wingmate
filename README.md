# Wingmate

An Agent-to-Agent (A2A) communication tool implementing the A2A protocol for autonomous agent coordination.

## Overview

Wingmate enables agents to communicate with each other using a standardized protocol. Each agent can:

- **Discover peers** via Agent Cards (published at `/.well-known/agent.json`)
- **Exchange messages** using JSON-RPC 2.0 over HTTP
- **Log all activity** to a Flight Log for audit and debugging

## Prerequisites

- Go 1.21 or later

## Quick Start

### Build

```bash
make build
```

This produces `bin/wingmate` for your platform.

### Start an Agent

```bash
./bin/wingmate --name agent-a --port 9000
```

Output:
```
Agent "agent-a" listening on http://localhost:9000
Agent Card: http://localhost:9000/.well-known/agent.json
Press Ctrl+C to stop
```

### Ping Another Agent

In a second terminal, start another agent:

```bash
./bin/wingmate --name agent-b --port 9001
```

In a third terminal, send a ping:

```bash
./bin/wingmate ping http://localhost:9000 --name sender
```

Output:
```
Sending ping to http://localhost:9000...
Received pong!
```

### Check Agent Status

```bash
./bin/wingmate status http://localhost:9000
```

Output:
```
Fetching Agent Card from http://localhost:9000...

Name:        agent-a
Version:     1.0.0
Description: A2A Agent
URL:         http://localhost:9000
Streaming:   false
Push:        false
```

## Usage

```
Usage: wingmate [options] [command]

Options:
  --name      Agent name (required for server mode)
  --port      Listen port (default: 9000)
  --peers     Comma-separated list of peer URLs
  --log       Flight log path (default: ./flight.jsonl)
  --verbose   Mirror logs to stdout
  --config    Config file path
  --help      Show help

Commands:
  (none)              Start agent server
  ping <peer-url>     Send ping to a peer
  status <peer-url>   Fetch peer's Agent Card

Examples:
  wingmate --port 9000 --name my-agent
  wingmate ping http://localhost:9001 --name sender
  wingmate status http://localhost:9001
```

## Configuration

Configuration can be provided via:

1. **Command-line flags** (highest priority)
2. **Environment variables**
3. **Config file** (JSON)
4. **Defaults** (lowest priority)

### Environment Variables

| Variable | Description |
|----------|-------------|
| `WINGMATE_NAME` | Agent name |
| `WINGMATE_PORT` | Listen port |
| `WINGMATE_PEERS` | Comma-separated peer URLs |
| `WINGMATE_LOG` | Flight log path |
| `WINGMATE_VERBOSE` | Enable verbose output (`true`/`false`) |

### Config File

See `config/agent.json.example` for a sample configuration:

```json
{
  "name": "my-agent",
  "port": 9000,
  "peers": ["http://localhost:9001"],
  "logFile": "./flight.jsonl",
  "verbose": false,
  "capabilities": ["ping", "streaming"]
}
```

Load with:

```bash
./bin/wingmate --config config/agent.json
```

## Flight Log

All messages are logged to `flight.jsonl` (configurable via `--log`). Each line is a JSON object:

```json
{"timestamp":"2026-01-21T10:00:00Z","agent":"agent-a","peer":"agent-b","direction":"inbound","role":"wingmate","message":{...}}
```

Fields:
- `timestamp` - ISO 8601 timestamp
- `agent` - Local agent name
- `peer` - Remote agent name
- `direction` - `inbound` or `outbound`
- `role` - `pilot` (initiated) or `wingmate` (responded)
- `message` - The A2A message

## Agent Card

Each agent publishes an Agent Card at `/.well-known/agent.json`:

```json
{
  "name": "agent-a",
  "version": "1.0.0",
  "description": "A2A Agent",
  "url": "http://localhost:9000",
  "capabilities": {
    "streaming": false,
    "pushNotifications": false
  },
  "skills": []
}
```

## LLM Support

Wingmate can use Claude CLI to provide intelligent responses to messages. When Claude CLI is installed, agents gain a `chat` skill that processes messages through Claude.

### Quick Setup

1. **Install Claude CLI** from [claude.ai/download](https://claude.ai/download)
2. **Authenticate**: Run `claude` once and follow the prompts to sign in
3. **Start Wingmate**: The agent automatically detects Claude CLI

```bash
# Verify Claude CLI is installed
claude --version

# Start agent (chat skill enabled automatically)
./bin/wingmate --name my-agent --port 9000
```

When Claude CLI is available, the Agent Card includes a `chat` skill:

```json
{
  "skills": [
    {
      "id": "chat",
      "name": "Chat",
      "description": "Process messages using Claude"
    }
  ]
}
```

### Configuration

Configure LLM behavior via environment variables or config file:

| Variable | Description | Default |
|----------|-------------|---------|
| `WINGMATE_LLM_MODEL` | Claude model to use | `claude-sonnet-4-20250514` |
| `WINGMATE_LLM_TIMEOUT` | Request timeout | `30s` |
| `WINGMATE_LLM_CLI_PATH` | Path to Claude CLI | Auto-detected |

### Troubleshooting

| Error | Cause | Solution |
|-------|-------|----------|
| Error 2001 | Claude CLI not installed | Install from [claude.ai/download](https://claude.ai/download) |
| Error 2002 | CLI execution failed | Check `claude --version` works |
| Error 2003 | Response parse error | Ensure CLI version is compatible |

For detailed setup instructions, see [docs/how/llm-setup.md](docs/how/llm-setup.md).

## Development

### Build for All Platforms

```bash
make build-all
```

Produces binaries in `bin/`:
- `wingmate-darwin-amd64`
- `wingmate-darwin-arm64`
- `wingmate-linux-amd64`
- `wingmate-linux-arm64`
- `wingmate-windows-amd64.exe`
- `wingmate-android-arm64`

### Run Tests

```bash
make test
```

### Run Integration Tests

```bash
make test-integration
```

### Run Linter

```bash
make lint
```

### Clean Build Artifacts

```bash
make clean
```

## Architecture

```
wingmate/
├── cmd/wingmate/       # CLI entry point
├── internal/
│   ├── agent/          # Unified agent implementation
│   ├── flightlog/      # Flight log (audit trail)
│   └── protocol/       # A2A protocol (server/client)
├── pkg/types/          # Shared types (Message, AgentCard)
├── config/             # Example configuration
└── tests/integration/  # End-to-end tests
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Configuration error |
| 2 | Network/peer error |
| 3 | Internal error |

## License

MIT
