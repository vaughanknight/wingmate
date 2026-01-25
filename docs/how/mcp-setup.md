# MCP Server Setup Guide

This guide covers setting up Wingmate's MCP (Model Context Protocol) integration for Claude Code.

## Overview

Wingmate exposes MCP tools via HTTP at the `/mcp` endpoint. When running, any Claude Code instance can connect and use Wingmate's tools directly:

- **wingmate_chat** - Send messages through Wingmate's LLM integration
- **wingmate_status** - Get operational status and health information
- **wingmate_discover** - Discover available peer agents on the network

## Prerequisites

- Wingmate binary installed (built from source or downloaded)
- Claude Code CLI installed (`claude --version` to verify)
- Internet connection for Claude API access

## Quick Start

### Step 1: Start Wingmate Agent

```bash
wingmate --port 9000 --name my-agent
```

Output:
```
Agent "my-agent" listening on http://localhost:9000
Agent Card: http://localhost:9000/.well-known/agent.json
MCP available at: http://localhost:9000/mcp
Press Ctrl+C to stop
```

### Step 2: Configure Claude Code

Run this command once to register Wingmate as an MCP server:

```bash
claude mcp add --transport http wingmate http://localhost:9000/mcp
```

### Step 3: Restart Claude Code

Completely quit and reopen Claude Code. The Wingmate tools will now be available.

### Step 4: Verify

In Claude Code, ask:

> "Use the wingmate_status tool to check Wingmate's status"

You should see a response with server uptime and configuration details.

## Installation

### Option 1: Build from Source

```bash
# Clone the repository
git clone https://github.com/wingmate/wingmate.git
cd wingmate

# Build the binary
go build -o wingmate ./cmd/wingmate

# Move to PATH (optional)
sudo mv wingmate /usr/local/bin/
```

### Option 2: Download Pre-built Binary

Download the latest release for your platform from the [releases page](https://github.com/wingmate/wingmate/releases).

**macOS (Apple Silicon):**
```bash
curl -L https://github.com/wingmate/wingmate/releases/latest/download/wingmate-darwin-arm64 -o wingmate
chmod +x wingmate
sudo mv wingmate /usr/local/bin/
```

**macOS (Intel):**
```bash
curl -L https://github.com/wingmate/wingmate/releases/latest/download/wingmate-darwin-amd64 -o wingmate
chmod +x wingmate
sudo mv wingmate /usr/local/bin/
```

**Linux (x64):**
```bash
curl -L https://github.com/wingmate/wingmate/releases/latest/download/wingmate-linux-amd64 -o wingmate
chmod +x wingmate
sudo mv wingmate /usr/local/bin/
```

**Windows:**
Download `wingmate-windows-amd64.exe` from the releases page and add to PATH.

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `WINGMATE_LLM_MODEL` | Claude model for chat tool | CLI default |
| `WINGMATE_LLM_TIMEOUT` | Request timeout in seconds | `120` |

Example:
```bash
WINGMATE_LLM_MODEL=claude-sonnet-4-20250514 wingmate --port 9000 --name my-agent
```

### Agent Options

| Option | Description | Default |
|--------|-------------|---------|
| `--port` | Listen port | `9000` |
| `--name` | Agent name (required) | - |
| `--peers` | Comma-separated peer URLs | - |
| `--log` | Flight log path | `./flight.jsonl` |
| `--verbose` | Mirror logs to stdout | `false` |

## Available Tools

### wingmate_chat

Send a chat message through Wingmate's LLM integration.

**Schema:**
```json
{
  "name": "wingmate_chat",
  "description": "Send a chat message through Wingmate",
  "inputSchema": {
    "type": "object",
    "properties": {
      "prompt": {
        "type": "string",
        "description": "The message to send"
      },
      "session_id": {
        "type": "string",
        "description": "Optional session ID for conversation continuity"
      }
    },
    "required": ["prompt"]
  }
}
```

### wingmate_status

Get Wingmate's current operational status.

**Schema:**
```json
{
  "name": "wingmate_status",
  "description": "Get Wingmate status and health information",
  "inputSchema": {
    "type": "object",
    "properties": {}
  }
}
```

**Response includes:**
- Server uptime
- LLM availability
- Known peers
- Active MCP sessions

### wingmate_discover

Discover available peer agents on the network.

**Schema:**
```json
{
  "name": "wingmate_discover",
  "description": "Discover available Wingmate agents",
  "inputSchema": {
    "type": "object",
    "properties": {}
  }
}
```

**Returns:**
- List of known peer agents
- Each peer's name, URL, and capabilities

## Architecture

### How It Works

```
┌──────────────┐                              ┌──────────────────────┐
│              │     HTTP (JSON-RPC 2.0)      │                      │
│  Claude Code │ ────────────────────────────▶│   Wingmate Agent     │
│              │ ◀────────────────────────────│                      │
└──────────────┘         /mcp                 │  ┌────────────────┐  │
                                              │  │  MCP Handler   │  │
                                              │  │  (HTTP)        │  │
                                              │  └───────┬────────┘  │
                                              │          │           │
                                              │  ┌───────▼────────┐  │
                                              │  │  Tool Handlers │  │
                                              │  └───────┬────────┘  │
                                              │          │           │
                                              │  ┌───────▼────────┐  │
                                              │  │  Claude CLI    │  │
                                              │  │  (for chat)    │  │
                                              │  └────────────────┘  │
                                              └──────────────────────┘
```

### Key Design Points

1. **Single Process** - MCP is served from the same agent that handles A2A requests
2. **HTTP Transport** - Uses MCP Streamable HTTP (2025-03-26 spec)
3. **Localhost Only** - MCP endpoint only accepts requests from localhost (returns 403 otherwise)
4. **Session Support** - Uses `Mcp-Session-Id` header for session management (30-min TTL)

### Flight Log

All MCP tool invocations are logged to the Flight Log (`./flight.jsonl` by default):

```json
{
  "timestamp": "2026-01-22T10:30:00Z",
  "trace_id": "abc123",
  "agent": "my-agent",
  "direction": "outbound",
  "summary": "mcp: wingmate_chat response",
  "payload": { "tool": "wingmate_chat", "status": "success" }
}
```

## Troubleshooting

### Error Codes

| Code | Name | Description | Solution |
|------|------|-------------|----------|
| 3001 | MCPInvalidRequest | Malformed JSON-RPC request | Check request format |
| 3002 | MCPMethodNotFound | Unknown method called | Verify tool name |
| 3003 | MCPInvalidParams | Invalid tool parameters | Check parameter schema |
| 3004 | MCPInternalError | Server internal error | Check logs, restart agent |
| 3010 | MCPToolNotFound | Tool not registered | Restart agent |
| 3011 | MCPToolExecutionFailed | Tool handler error | Check specific tool error |

### Common Issues

**"Connection refused" when Claude Code tries to connect**

1. Verify agent is running: `curl http://localhost:9000/.well-known/agent.json`
2. Check the port matches your configuration
3. Ensure no firewall blocking localhost connections

**"403 Forbidden" response**

MCP endpoint only accepts requests from localhost. This is expected behavior for non-local requests.

**Tools not appearing in Claude Code**

1. Verify agent is running with MCP available
2. Check Claude Code configuration: `claude mcp list`
3. Completely restart Claude Code (not just reload)
4. Verify URL matches: `http://localhost:PORT/mcp`

**Tool calls timing out**

1. Increase `WINGMATE_LLM_TIMEOUT` environment variable
2. Check network connectivity to Claude API
3. Use `--verbose` flag to see where it's hanging

**"The 'mcp' command has been removed"**

This is expected. The old stdio transport has been removed. Start the agent normally:
```bash
wingmate --port 9000 --name my-agent
```

Then configure Claude Code with HTTP transport:
```bash
claude mcp add --transport http wingmate http://localhost:9000/mcp
```

### Debug Mode

For detailed debugging, run with verbose logging:

```bash
wingmate --port 9000 --name my-agent --verbose
```

This mirrors all log output to stdout for real-time monitoring.

### Verifying MCP Endpoint

Test the MCP endpoint directly:

```bash
# Initialize session
curl -s -X POST http://localhost:9000/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'

# List tools (use session ID from initialize response)
curl -s -X POST http://localhost:9000/mcp \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: <session-id>" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
```

## Security Considerations

- **Localhost Only** - MCP endpoint returns 403 for non-localhost requests
- **No API Keys Stored** - Wingmate uses Claude CLI which handles authentication
- **Flight Log** - Tool invocations are logged; ensure appropriate file permissions
- **Session Timeout** - MCP sessions expire after 30 minutes of inactivity

## Platform-Specific Notes

### macOS

- Binary may need to be allowed in System Preferences > Security & Privacy
- Use absolute path when running from non-PATH location

### Linux

- Ensure binary has execute permissions: `chmod +x wingmate`
- For system-wide installation: `/usr/local/bin/wingmate`
- For user installation: `~/.local/bin/wingmate`

### Windows

- Add `.exe` extension when running
- May need to allow through Windows Firewall (localhost only)

## Migrating from Stdio Transport

If you previously configured Wingmate with stdio transport, follow these steps to migrate to HTTP transport.

### Step 1: Remove Old Configuration

If you manually edited `~/.claude.json`, remove the wingmate entry:

```json
// DELETE this configuration:
{
  "mcpServers": {
    "wingmate": {  // <-- Remove this entire block
      "command": "/usr/local/bin/wingmate",
      "args": ["mcp"],
      "type": "stdio"
    }
  }
}
```

Or use the Claude CLI to remove it:
```bash
claude mcp remove wingmate
```

### Step 2: Start Wingmate Agent

The `wingmate mcp` command has been removed. Instead, start the full agent:

```bash
wingmate --port 9000 --name my-agent
```

### Step 3: Configure HTTP Transport

```bash
claude mcp add --transport http wingmate http://localhost:9000/mcp
```

### Step 4: Restart Claude Code

Completely quit and reopen Claude Code for changes to take effect.

### What Changed

| Aspect | Before (Stdio) | After (HTTP) |
|--------|----------------|--------------|
| Command | `wingmate mcp` | `wingmate --port 9000 --name my-agent` |
| Configuration | `~/.claude.json` manual edit | `claude mcp add --transport http` |
| Process Model | Claude Code spawns subprocess | You start agent, Claude Code connects |
| Peer Discovery | Always empty | Returns actual known peers |
| Session State | Per-subprocess | Shared with A2A agent |

### Troubleshooting Migration

**"The 'mcp' command has been removed"**

This is expected. The old stdio transport is no longer available. Start the agent with `wingmate --port 9000 --name my-agent` instead.

**Tools not appearing after migration**

1. Verify agent is running: `curl http://localhost:9000/.well-known/agent.json`
2. Verify MCP configured: `claude mcp list`
3. Restart Claude Code completely

**Rollback (if needed)**

A git tag `pre-stdio-removal` exists if you need to revert to the old stdio transport for any reason. This is not recommended for normal use.

## Related Documentation

- [LLM Setup Guide](./llm-setup.md) - Claude CLI integration for agents
- [README - Quick Start](../../README.md)
- [CLAUDE.md - Project Context](../../CLAUDE.md)
- [ADR-004 - MCP Implementation](../../docs/adr/004-mcp-server-implementation.md)
- [MCP Protocol Specification](https://modelcontextprotocol.io/specification)
