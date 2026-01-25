# Wingmate Architecture

**Version:** 1.0.0
**Last Updated:** 2026-01-21
**Constitution Reference:** [constitution.md](./constitution.md)

---

## 1. System Overview

### 1.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              WINGMATE SYSTEM                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Machine A (e.g., Server)              Machine B (e.g., Client/Device)    │
│   ┌─────────────────────┐              ┌─────────────────────┐             │
│   │      AGENT          │◄────A2A─────►│      AGENT          │             │
│   │  (peer instance)    │   Protocol   │  (peer instance)    │             │
│   │  ┌───────────────┐  │              │  ┌───────────────┐  │             │
│   │  │  Claude API   │  │              │  │  Claude API   │  │             │
│   │  └───────────────┘  │              │  └───────────────┘  │             │
│   │         │           │              │         │           │             │
│   │  ┌───────────────┐  │              │  ┌───────────────┐  │             │
│   │  │  Flight Log   │  │              │  │  Flight Log   │  │             │
│   │  └───────────────┘  │              │  └───────────────┘  │             │
│   │         │           │              │         │           │             │
│   │  ┌───────────────┐  │              │  ┌───────────────┐  │             │
│   │  │ Local Tools   │  │              │  │ Local Tools   │  │             │
│   │  │   (via MCP)   │  │              │  │   (via MCP)   │  │             │
│   │  └───────────────┘  │              │  └───────────────┘  │             │
│   └─────────────────────┘              └─────────────────────┘             │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 Design Goals

| Goal | Description | Constitution Principle |
|------|-------------|----------------------|
| **Bidirectional Communication** | Either agent can initiate or respond | P4 (Protocol Compliance) |
| **High Observability** | All exchanges logged for analysis | P1 (Observability First) |
| **Rapid Deployment** | Easy install/upgrade from git | P2 (Deploy Early, Deploy Often) |
| **Cross-Platform** | Works on servers, desktops, mobile | P3 (Engineer Autonomy) |
| **Secure by Default** | TLS, auth, audit trails | P6 (Security by Design) |

<!-- USER CONTENT START -->
<!-- Add project-specific design goals here -->
<!-- USER CONTENT END -->

---

## 2. Component Architecture

### 2.1 Core Components

```
┌────────────────────────────────────────────────────────────────────┐
│                        AGENT COMPONENT                              │
├────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐  │
│  │   A2A Server    │   │   A2A Client    │   │   Agent Card    │  │
│  │  (Receive Msgs) │   │  (Send Msgs)    │   │  (Capabilities) │  │
│  └────────┬────────┘   └────────┬────────┘   └────────┬────────┘  │
│           │                     │                     │            │
│           └──────────┬──────────┴──────────┬──────────┘            │
│                      │                     │                       │
│              ┌───────▼───────┐     ┌───────▼───────┐              │
│              │  Message      │     │   Capability   │              │
│              │  Router       │     │   Registry     │              │
│              └───────┬───────┘     └───────┬───────┘              │
│                      │                     │                       │
│              ┌───────▼─────────────────────▼───────┐              │
│              │         Agent Executor              │              │
│              │   (Claude Integration + Tools)      │              │
│              └───────┬─────────────────────────────┘              │
│                      │                                             │
│           ┌──────────┼──────────┐                                 │
│           │          │          │                                 │
│    ┌──────▼──────┐ ┌─▼─────────┐ ┌──────▼──────┐                 │
│    │ Flight Log  │ │ Claude    │ │ MCP Tools   │                 │
│    │ (Observ.)   │ │ API       │ │ (Local)     │                 │
│    └─────────────┘ └───────────┘ └─────────────┘                 │
│                                                                     │
└────────────────────────────────────────────────────────────────────┘
```

### 2.2 Component Responsibilities

| Component | Responsibility | Dependencies |
|-----------|---------------|--------------|
| **A2A Server** | Listen for incoming requests, validate auth | A2A SDK, HTTP server |
| **A2A Client** | Send requests to other agents | A2A SDK, HTTP client |
| **Agent Card** | Declare capabilities, auth requirements | Static config |
| **Message Router** | Route incoming messages to appropriate handlers | Agent Executor |
| **Capability Registry** | Track available local tools and skills | MCP tools |
| **Agent Executor** | Process requests, invoke Claude, use tools | Claude API, MCP |
| **Flight Log** | Record all exchanges for observability | Logging library |
| **LLM Executor** | Execute prompts via Claude CLI, manage sessions | Claude CLI binary |
| **Session Manager** | Maintain conversation context via session IDs | In-memory storage |
| **Claude API** | LLM reasoning and generation | Anthropic API (via CLI) |
| **MCP Tools** | Local system access (files, metrics, etc.) | MCP server |

### 2.3 Module Boundaries

```
wingmate/
├── cmd/
│   └── wingmate/           # Main entry point
│       └── main.go         # CLI entry point
│
├── internal/
│   ├── agent/              # Unified agent (see ADR-003)
│   │   ├── agent.go        # Agent struct - both client + server
│   │   ├── server.go       # A2A server (handle incoming requests)
│   │   ├── client.go       # A2A client (make outgoing requests)
│   │   ├── conversation.go # Conversation/mission management
│   │   └── config.go       # Configuration management
│   │
│   ├── protocol/           # A2A protocol implementation
│   │   ├── client.go       # A2A HTTP client
│   │   ├── server.go       # A2A HTTP server
│   │   ├── messages.go     # Message types
│   │   └── jsonrpc.go      # JSON-RPC 2.0 implementation
│   │
│   ├── llm/                # LLM integration (Claude CLI)
│   │   ├── types.go        # CLIResponse, Usage, Metadata types
│   │   ├── client.go       # LLMExecutor interface
│   │   ├── cli.go          # CLIExecutor implementation
│   │   ├── errors.go       # Error codes 2001-2006
│   │   ├── convert.go      # Response conversion to A2A format
│   │   ├── config.go       # LLM configuration
│   │   ├── validate.go     # Config validation
│   │   └── session.go      # SessionManager for conversation continuity
│   │
│   └── flightlog/          # Flight Log and tracing
│       ├── flightlog.go    # FlightLog struct and methods
│       └── entry.go        # Log entry types
│
├── pkg/                    # Public packages (if needed by external tools)
│   └── types/              # Shared types for external use
│       └── agentcard.go    # Agent Card types
│
├── config/                 # Configuration templates
│   └── agent.example.json
│
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
└── Makefile                # Build targets for cross-compilation
```

**Dependency Rules:**
- `internal/agent/` MAY import `internal/protocol/`, `internal/flightlog/`, `internal/llm/`, `internal/mcp/`
- `internal/mcp/` MAY import `internal/flightlog/`, `internal/llm/`
- `internal/llm/` MAY import `internal/flightlog/`, `pkg/types/` (for message conversion)
- `internal/protocol/` MUST NOT import `internal/agent/`
- `internal/flightlog/` MUST NOT import `internal/agent/`
- `internal/llm/` MUST NOT import `internal/agent/`, `internal/protocol/`
- `internal/mcp/` MUST NOT import `internal/agent/` (use interfaces for dependency injection)
- `pkg/` packages are for external consumption; `internal/` packages are private

**Note:** Per [ADR-003](../adr/003-unified-peer-architecture.md), there are no separate `internal/pilot/` or `internal/wingmate/` directories. Every agent instance can act as either pilot (initiator) or wingmate (responder) in any conversation.

### 2.4 MCP HTTP Integration

The MCP server is integrated into the A2A server via HTTP transport at the `/mcp` endpoint:

```
┌─────────────────────────────────────────────────────────────────┐
│                    A2A SERVER HTTP ROUTES                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   /.well-known/agent.json  →  handleAgentCard()                 │
│   /a2a                     →  handleA2A()                       │
│   /mcp                     →  LocalhostMiddleware(MCPHandler)   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**MCP Security Model:**
- `/mcp` endpoint is wrapped with `LocalhostMiddleware`
- Non-localhost requests receive HTTP 403 Forbidden
- MCP handlers access agent state via `PeerProvider` interface (dependency injection)

**Dependency Flow:**
```
internal/agent/
    │
    ├──imports──► internal/mcp/ (NewHTTPHandler, LocalhostMiddleware)
    │
    └──implements──► mcp.PeerProvider (Agent.GetPeers())
```

**Key MCP Components:**
| Component | File | Purpose |
|-----------|------|---------|
| HTTPHandler | `http_transport.go` | HTTP handler implementing `http.Handler` |
| LocalhostMiddleware | `localhost.go` | Security wrapper returning 403 for non-localhost |
| MCPSessionManager | `session.go` | Session management with 30-min TTL |
| PeerProvider | `handlers.go` | Interface for peer discovery injection |

<!-- USER CONTENT START -->
<!-- Add project-specific component details here -->
<!-- USER CONTENT END -->

---

## 3. Data Flow

### 3.1 Basic Request-Response Flow

```
    PILOT                                           WINGMATE
      │                                                  │
      │  1. Create Mission                               │
      ├──────────────────────┐                          │
      │                      │                          │
      │  2. Format A2A Request                          │
      ├──────────────────────┤                          │
      │                      │                          │
      │  3. Log outbound     │                          │
      │     (Flight Log)     │                          │
      ├──────────────────────┤                          │
      │                      │                          │
      │  4. Send via A2A ─────────────────────────────► │
      │                      │                          │
      │                      │  5. Receive & validate   │
      │                      │ ◄────────────────────────┤
      │                      │                          │
      │                      │  6. Log inbound          │
      │                      │     (Flight Log)         │
      │                      │ ◄────────────────────────┤
      │                      │                          │
      │                      │  7. Process with Claude  │
      │                      │     + MCP tools          │
      │                      │ ◄────────────────────────┤
      │                      │                          │
      │                      │  8. Log outbound         │
      │                      │     (Flight Log)         │
      │                      │ ◄────────────────────────┤
      │                      │                          │
      │  9. Receive ◄────────────────────────────────── │
      │                      │                          │
      │  10. Log inbound     │                          │
      │      (Flight Log)    │                          │
      ├──────────────────────┤                          │
      │                      │                          │
      │  11. Process response│                          │
      ├──────────────────────┘                          │
      │                                                  │
```

### 3.2 Multi-Turn Conversation Flow

```
    PILOT                                           WINGMATE
      │                                                  │
      │  Request: "Why is CPU high?"                    │
      ├──────────────────────────────────────────────► │
      │                                                  │
      │  ◄────────── Response: "Need timeframe"  ────── │
      │              (Wingmate asks clarification)      │
      │                                                  │
      │  Request: "Last 10 minutes"                     │
      ├──────────────────────────────────────────────► │
      │                                                  │
      │  [Wingmate uses MCP to get metrics]             │
      │  [Wingmate asks Claude to analyze]              │
      │                                                  │
      │  ◄────────── Response: "Found runaway  ──────── │
      │              process PID 12345..."              │
      │                                                  │
```

### 3.3 Data Formats

**Flight Log Entry:**
```json
{
  "timestamp": "2026-01-21T14:32:17.123Z",
  "trace_id": "abc123-def456",
  "pilot_id": "pilot-server-01",
  "wingmate_id": "wingmate-device-42",
  "direction": "outbound",
  "summary": "Pilot requested CPU metrics for last 10 minutes",
  "payload_type": "json",
  "payload": {
    "operation": "get_metrics",
    "params": {
      "metrics": ["cpu"],
      "duration": "10m"
    }
  },
  "mission_id": "mission-789"
}
```

**Agent Card (A2A Standard):**
```json
{
  "name": "Wingmate-Device-42",
  "description": "Debugging agent on mobile device",
  "url": "https://device42.local:9000/a2a",
  "version": "1.0.0",
  "capabilities": [
    {
      "name": "GetMetrics",
      "description": "Retrieve system metrics (CPU, memory, etc.)",
      "parameters": {
        "metrics": {"type": "array", "items": {"type": "string"}},
        "duration": {"type": "string"}
      }
    },
    {
      "name": "GetLogs",
      "description": "Retrieve application logs",
      "parameters": {
        "app": {"type": "string"},
        "lines": {"type": "integer"}
      }
    }
  ],
  "authentication": {
    "schemes": ["bearer", "apiKey"]
  }
}
```

<!-- USER CONTENT START -->
<!-- Add project-specific data flow details here -->
<!-- USER CONTENT END -->

---

## 4. Integration Points

### 4.1 External Dependencies

| Dependency | Purpose | Interface |
|------------|---------|-----------|
| **Claude API** | LLM reasoning | HTTP API (or Go SDK when available) |
| **A2A Protocol** | Agent communication | Direct implementation or `a2aproject/a2a-go` |
| **MCP** | Local tool access | MCP protocol (when needed) |
| **OpenTelemetry** | Distributed tracing | `go.opentelemetry.io/otel` (optional) |

### 4.2 Integration Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     EXTERNAL INTEGRATIONS                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌────────────────┐    ┌────────────────┐    ┌──────────────┐ │
│   │  Claude API    │    │  Other A2A     │    │  Tracing     │ │
│   │  (Anthropic)   │    │  Agents        │    │  Backend     │ │
│   └───────┬────────┘    └───────┬────────┘    └──────┬───────┘ │
│           │                     │                     │         │
│           │ Anthropic SDK       │ A2A Protocol        │ OTel    │
│           │                     │                     │         │
│   ┌───────▼─────────────────────▼─────────────────────▼───────┐ │
│   │                    WINGMATE AGENT                          │ │
│   └───────┬─────────────────────────────────────────────────┬─┘ │
│           │                                                 │   │
│           │ MCP Protocol                           Local FS │   │
│           │                                                 │   │
│   ┌───────▼────────┐                              ┌─────────▼─┐ │
│   │  MCP Servers   │                              │  Local    │ │
│   │  (tools)       │                              │  Storage  │ │
│   └────────────────┘                              └───────────┘ │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.3 API Contracts

**A2A Task Request (to Wingmate):**
```json
{
  "jsonrpc": "2.0",
  "method": "tasks/send",
  "id": "req-001",
  "params": {
    "task_id": "task-abc123",
    "message": {
      "role": "user",
      "parts": [
        {
          "kind": "text",
          "content": "What is the current CPU usage?"
        }
      ]
    }
  }
}
```

**A2A Task Response (from Wingmate):**
```json
{
  "jsonrpc": "2.0",
  "id": "req-001",
  "result": {
    "task_id": "task-abc123",
    "status": "completed",
    "message": {
      "role": "agent",
      "parts": [
        {
          "kind": "text",
          "content": "Current CPU usage is 45% (user: 30%, system: 15%)"
        },
        {
          "kind": "data",
          "content": {
            "cpu_percent": 45,
            "user_percent": 30,
            "system_percent": 15
          }
        }
      ]
    }
  }
}
```

<!-- USER CONTENT START -->
<!-- Add project-specific integration details here -->
<!-- USER CONTENT END -->

---

## 5. Deployment Architecture

### 5.1 Installation Model

**Option A: Download Pre-built Binary (Recommended)**
```
┌─────────────────────────────────────────────────────────────────┐
│                    BINARY INSTALLATION                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   1. Download binary for your platform                          │
│      $ curl -LO https://github.com/<org>/wingmate/releases/     │
│              latest/download/wingmate-linux-amd64               │
│      $ chmod +x wingmate-linux-amd64                            │
│      $ mv wingmate-linux-amd64 /usr/local/bin/wingmate          │
│                                                                  │
│   2. Run (no dependencies needed!)                               │
│      $ wingmate --port 9001 --peers https://other:9000          │
│                                                                  │
│   3. Upgrade                                                     │
│      $ curl -LO <new-release-url>                               │
│      $ mv wingmate-new /usr/local/bin/wingmate                  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Option B: Build from Source**
```
┌─────────────────────────────────────────────────────────────────┐
│                    BUILD FROM SOURCE                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   1. Clone repository                                           │
│      $ git clone https://github.com/<org>/wingmate.git          │
│                                                                  │
│   2. Build (requires Go 1.21+)                                  │
│      $ cd wingmate && go build -o wingmate ./cmd/wingmate       │
│                                                                  │
│   3. Cross-compile for other platforms                          │
│      $ make build-linux    # Linux amd64                        │
│      $ make build-darwin   # macOS arm64                        │
│      $ make build-windows  # Windows amd64                      │
│      $ make build-android  # Android arm64                      │
│                                                                  │
│   4. Configure (optional)                                        │
│      $ cp config/agent.example.json config/agent.json           │
│      $ $EDITOR config/agent.json                                │
│                                                                  │
│   5. Run                                                         │
│      $ ./wingmate --port 9000 --peers https://other:9001        │
│                                                                  │
│   6. Upgrade                                                     │
│      $ git pull && go build -o wingmate ./cmd/wingmate          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2 Deployment Topology

**Scenario: Server and Client debugging each other**
```
┌──────────────────┐              ┌──────────────────┐
│    SERVER        │              │    CLIENT        │
│  (Data Center)   │              │  (User Machine)  │
│                  │              │                  │
│  ┌────────────┐  │              │  ┌────────────┐  │
│  │   AGENT    │  │◄────TLS─────►│  │   AGENT    │  │
│  │            │  │   A2A       │  │            │  │
│  └────────────┘  │              │  └────────────┘  │
│        │         │              │        │         │
│  ┌─────▼──────┐  │              │  ┌─────▼──────┐  │
│  │ Flight Log │  │              │  │ Flight Log │  │
│  └────────────┘  │              │  └────────────┘  │
│                  │              │        │         │
│  Either can      │              │  ┌─────▼──────┐  │
│  initiate!       │              │  │ Local Logs │  │
│                  │              │  │ & Metrics  │  │
│                  │              │  └────────────┘  │
└──────────────────┘              └──────────────────┘

Server asks: "What error did you see?"  → Server is pilot
Client asks: "What did my request return?" → Client is pilot
```

**Scenario: Mobile-to-Mobile debugging**
```
┌──────────────────┐              ┌──────────────────┐
│  ANDROID DEV A   │              │  ANDROID DEV B   │
│                  │              │                  │
│  ┌────────────┐  │              │  ┌────────────┐  │
│  │   AGENT    │  │◄────WiFi────►│  │   AGENT    │  │
│  │            │  │   A2A       │  │            │  │
│  └────────────┘  │              │  └────────────┘  │
│        │         │              │        │         │
│  ┌─────▼──────┐  │              │  ┌─────▼──────┐  │
│  │ Flight Log │  │              │  │ Flight Log │  │
│  └────────────┘  │              │  └────────────┘  │
└──────────────────┘              └──────────────────┘

Either device can ask the other for help debugging.
```

### 5.3 Configuration

**Minimal Configuration (config/agent.json):**
```json
{
  "name": "agent-device-01",
  "port": 9000,
  "peers": [
    "https://machineB:9001",
    "https://machineC:9002"
  ],
  "tls": {
    "enabled": true,
    "cert": "/path/to/cert.pem",
    "key": "/path/to/key.pem"
  },
  "authentication": {
    "type": "apiKey",
    "key_env": "WINGMATE_API_KEY"
  },
  "claude": {
    "api_key_env": "ANTHROPIC_API_KEY",
    "model": "claude-sonnet-4-20250514"
  },
  "flight_log": {
    "path": "./logs/flight.jsonl",
    "level": "info"
  },
  "capabilities": ["GetMetrics", "GetLogs"]
}
```

**Note:** There is no `mode` field. Every agent can both receive requests (act as wingmate) and send requests to peers (act as pilot). See [ADR-003](../adr/003-unified-peer-architecture.md).

<!-- USER CONTENT START -->
<!-- Add project-specific deployment details here -->
<!-- USER CONTENT END -->

---

## 6. Security Architecture

### 6.1 Security Layers

```
┌─────────────────────────────────────────────────────────────────┐
│                     SECURITY LAYERS                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   Layer 1: Transport Security                                    │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │  TLS 1.3 encryption for all A2A communication           │   │
│   │  Optional: Mutual TLS (mTLS) for high-security deployments │   │
│   └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│   Layer 2: Authentication                                        │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │  API Keys (simple deployments)                          │   │
│   │  OAuth2 / JWT (enterprise deployments)                  │   │
│   │  Validated on every request                             │   │
│   └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│   Layer 3: Authorization                                         │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │  Capability-based: Only declared operations allowed     │   │
│   │  Scope-limited tokens (e.g., read:logs, read:metrics)   │   │
│   └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│   Layer 4: Audit                                                 │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │  Flight Log records all exchanges                       │   │
│   │  Sensitive data redacted before logging                 │   │
│   └─────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 6.2 Threat Model

| Threat | Mitigation |
|--------|------------|
| **Eavesdropping** | TLS encryption mandatory |
| **Impersonation** | Certificate validation, API key/token auth |
| **Unauthorized access** | Capability-based authorization |
| **Replay attacks** | Request nonces, short-lived tokens |
| **Prompt injection** | Input sanitization before Claude |
| **Data exfiltration** | Scoped capabilities, rate limiting |

### 6.3 Security Checklist

- [ ] TLS enabled for all production deployments
- [ ] Authentication configured (not running in "open" mode)
- [ ] Credentials stored securely (env vars, not in code)
- [ ] Agent Card does not expose sensitive information
- [ ] Flight Log redacts credentials and PII
- [ ] Input validation on all external data
- [ ] TODO(SECURITY_AUDIT): Complete security review before release

<!-- USER CONTENT START -->
<!-- Add project-specific security details here -->
<!-- USER CONTENT END -->

---

## 7. Anti-Patterns

### 7.1 Architecture Anti-Patterns

| Anti-Pattern | Why It's Bad | Better Approach |
|--------------|--------------|-----------------|
| **Tight Pilot-Wingmate coupling** | Can't deploy independently | Use protocol contracts |
| **Shared state between agents** | Hard to debug, race conditions | Explicit message passing |
| **Skipping Flight Log** | Lose observability | Always log exchanges |
| **Hardcoded agent URLs** | Inflexible deployment | Configuration-driven |
| **Auth bypass "for testing"** | Security holes leak to prod | Use test credentials |

### 7.2 Reviewer Checklist

When reviewing PRs, verify:

- [ ] New code follows module boundaries (see section 2.3)
- [ ] All A2A exchanges are logged to Flight Log
- [ ] No credentials in code or logs
- [ ] Protocol changes include integration tests
- [ ] Configuration changes are documented
- [ ] New capabilities are declared in Agent Card schema

<!-- USER CONTENT START -->
<!-- Add project-specific anti-patterns here -->
<!-- USER CONTENT END -->

---

## Appendix A: Technology Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Primary Language** | **Go** | Single binary distribution, cross-compilation, no runtime dependencies |
| **Protocol** | A2A | Standard for agent-to-agent, vendor-neutral |
| **Local Tools** | MCP (optional) | Standard for agent-to-tool, complements A2A |
| **Tracing** | OpenTelemetry | Industry standard, A2A has native support |
| **Build System** | Makefile | Standard for Go projects, cross-compilation targets |
| **Config Format** | JSON | No dependencies, matches Agent Card format |

*See ADR-001 in `docs/plans/001-minimal-skeleton/minimal-skeleton-spec.md` for language decision rationale.*

---

## Appendix B: Future Considerations

- **Multi-agent orchestration**: One Pilot coordinating multiple Wingmates
- **Agent discovery**: Dynamic discovery via registry (A2A roadmap feature)
- **Persistent missions**: Resume debugging sessions across restarts
- **UI dashboard**: Visualize Flight Log and agent status

*These are not commitments; they represent potential evolution paths.*

---

*This architecture document describes the structure and boundaries of the Wingmate system. All decisions should trace back to [constitution principles](./constitution.md).*
