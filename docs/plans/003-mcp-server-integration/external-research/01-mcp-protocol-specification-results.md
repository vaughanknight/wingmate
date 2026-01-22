# Research Results: MCP Protocol Specification

**Source**: Perplexity Deep Research
**Date**: 2026-01-22
**Status**: Complete

---

# Implementing an MCP Server in Go: A Comprehensive Protocol and Implementation Guide

The Model Context Protocol (MCP) represents a standardized approach to enabling language models to interact with external systems, tools, and data sources through a well-defined interface. This comprehensive report explores the precise technical specifications for implementing an MCP server in Go, covering the complete wire protocol, transport mechanisms, initialization procedures, tool registration and invocation, and error handling strategies.

## The Model Context Protocol Wire Format and JSON-RPC 2.0 Foundation

At the core of MCP lies a strict adherence to the JSON-RPC 2.0 specification for all message exchange between clients and servers. The Model Context Protocol does not introduce its own messaging format but instead leverages the well-established JSON-RPC 2.0 standard.

### Message Types

Every message exchanged between an MCP client and server follows the JSON-RPC 2.0 specification, which defines three primary message types:

1. **Requests** - invocation expecting a response
2. **Responses** - replies to requests
3. **Notifications** - one-way communication (no response expected)

### Request Structure

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list",
  "params": {}
}
```

- `jsonrpc`: Must be "2.0"
- `id`: Unique identifier (string or integer, NOT null)
- `method`: Remote procedure to invoke
- `params`: Optional object with parameters

### Response Structure (Success)

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "name": "wingmate_chat",
        "description": "Send a prompt to Claude CLI",
        "inputSchema": {
          "type": "object",
          "properties": {
            "prompt": {
              "type": "string",
              "description": "The prompt to send"
            },
            "session_id": {
              "type": "string",
              "description": "Optional session identifier"
            }
          },
          "required": ["prompt"]
        }
      }
    ]
  }
}
```

### Response Structure (Error)

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "error": {
    "code": -32602,
    "message": "Invalid params: prompt parameter is required",
    "data": {
      "parameter": "prompt",
      "expected": "string",
      "received": "null"
    }
  }
}
```

### Notification Structure (No Response Expected)

```json
{
  "jsonrpc": "2.0",
  "method": "notifications/tools/list_changed"
}
```

Note: Notifications **omit the `id` field** entirely.

---

## STDIO Transport Mechanism and Message Framing

### Key Points

- **Format**: Newline-delimited JSON (NDJSON)
- **Each message**: Single line of JSON terminated by newline (LF or CRLF)
- **No multiline JSON** permitted
- **All logging must go to stderr** - stdout is reserved for protocol messages only

### Go Implementation for Reading

```go
package main

import (
    "bufio"
    "encoding/json"
    "os"
    "io"
)

type JSONRPCRequest struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      interface{} `json:"id"`
    Method  string      `json:"method"`
    Params  interface{} `json:"params,omitempty"`
}

func main() {
    reader := bufio.NewReader(os.Stdin)

    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            if err == io.EOF {
                break
            }
            break
        }

        var request JSONRPCRequest
        if err := json.Unmarshal([]byte(line), &request); err != nil {
            sendError(nil, -32700, "Parse error")
            continue
        }

        handleRequest(request)
    }
}
```

### Go Implementation for Writing

```go
func sendResponse(id interface{}, result interface{}) {
    response := JSONRPCResponse{
        JSONRPC: "2.0",
        ID:      id,
        Result:  result,
    }

    data, err := json.Marshal(response)
    if err != nil {
        log.Fprintf(os.Stderr, "Error marshaling response: %v\n", err)
        return
    }

    fmt.Println(string(data))
    os.Stdout.Sync()  // Flush immediately
}
```

**Critical**: Never write to stdout except for JSON-RPC protocol messages. All diagnostic logging must go to stderr.

---

## Server Initialization Handshake

### Sequence

1. Client sends `initialize` request
2. Server responds with capabilities
3. Client sends `initialized` notification
4. Normal operation begins

### Initialize Request (from client)

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-03-26",
    "capabilities": {
      "roots": {
        "listChanged": true
      },
      "sampling": {}
    },
    "clientInfo": {
      "name": "claude-desktop",
      "version": "1.0.0"
    }
  }
}
```

### Initialize Response (from server)

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2025-03-26",
    "capabilities": {
      "tools": {
        "listChanged": true
      }
    },
    "serverInfo": {
      "name": "wingmate-server",
      "version": "1.0.0"
    }
  }
}
```

### Initialized Notification (from client)

```json
{
  "jsonrpc": "2.0",
  "method": "notifications/initialized"
}
```

**Important**: Server should NOT process requests (other than pings) before receiving `initialized` notification.

---

## Tool Registration and Discovery

### tools/list Request

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {}
}
```

### tools/list Response (for wingmate_chat)

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [
      {
        "name": "wingmate_chat",
        "description": "Send a prompt to Claude CLI and get a response",
        "inputSchema": {
          "type": "object",
          "properties": {
            "prompt": {
              "type": "string",
              "description": "The prompt to send to Claude CLI"
            },
            "session_id": {
              "type": "string",
              "description": "Optional session identifier for conversation continuity"
            }
          },
          "required": ["prompt"]
        }
      }
    ]
  }
}
```

### Pagination Support

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [...],
    "nextCursor": "page-2"
  }
}
```

---

## Tool Invocation

### tools/call Request

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "wingmate_chat",
    "arguments": {
      "prompt": "How do I approach someone I like?",
      "session_id": "session-abc123"
    }
  }
}
```

### tools/call Response (Success)

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Here's my advice: Be authentic and genuine..."
      }
    ],
    "isError": false
  }
}
```

### tools/call Response (Tool Execution Error)

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Session session-invalid not found. Please start a new session."
      }
    ],
    "isError": true
  }
}
```

**Key Distinction**:
- **Tool execution errors**: Return with `isError: true` in result
- **Protocol errors**: Return as JSON-RPC error response

### Protocol Error Example

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "error": {
    "code": -32602,
    "message": "Unknown tool: nonexistent_tool"
  }
}
```

---

## Error Codes

### Standard JSON-RPC 2.0 Codes

| Code | Meaning |
|------|---------|
| -32700 | Parse error (invalid JSON) |
| -32600 | Invalid Request |
| -32601 | Method not found |
| -32602 | Invalid params |
| -32603 | Internal error |

### Implementation-Specific Range

- **-32000 to -32099**: Reserved for server-specific errors
- Example: Session not found, rate limit exceeded

---

## Complete Go Server Example

```go
package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "os"
    "strings"
)

type JSONRPCRequest struct {
    JSONRPC string                 `json:"jsonrpc"`
    ID      interface{}            `json:"id"`
    Method  string                 `json:"method"`
    Params  map[string]interface{} `json:"params,omitempty"`
}

type JSONRPCResponse struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      interface{} `json:"id"`
    Result  interface{} `json:"result,omitempty"`
    Error   interface{} `json:"error,omitempty"`
}

type ToolResult struct {
    Content []map[string]interface{} `json:"content"`
    IsError bool                     `json:"isError"`
}

type WingmateServer struct {
    initialized bool
    sessions    map[string][]string
}

func NewWingmateServer() *WingmateServer {
    return &WingmateServer{
        initialized: false,
        sessions:    make(map[string][]string),
    }
}

func (ws *WingmateServer) handleInitialize(request JSONRPCRequest) {
    response := JSONRPCResponse{
        JSONRPC: "2.0",
        ID:      request.ID,
        Result: map[string]interface{}{
            "protocolVersion": "2025-03-26",
            "capabilities": map[string]interface{}{
                "tools": map[string]interface{}{
                    "listChanged": false,
                },
            },
            "serverInfo": map[string]interface{}{
                "name":    "wingmate-server",
                "version": "1.0.0",
            },
        },
    }
    ws.sendResponse(response)
}

func (ws *WingmateServer) handleToolsList(request JSONRPCRequest) {
    response := JSONRPCResponse{
        JSONRPC: "2.0",
        ID:      request.ID,
        Result: map[string]interface{}{
            "tools": []map[string]interface{}{
                {
                    "name":        "wingmate_chat",
                    "description": "Send a prompt to Claude CLI",
                    "inputSchema": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                            "prompt": map[string]interface{}{
                                "type":        "string",
                                "description": "The prompt to send",
                            },
                            "session_id": map[string]interface{}{
                                "type":        "string",
                                "description": "Optional session identifier",
                            },
                        },
                        "required": []string{"prompt"},
                    },
                },
            },
        },
    }
    ws.sendResponse(response)
}

func (ws *WingmateServer) sendResponse(response JSONRPCResponse) {
    data, err := json.Marshal(response)
    if err != nil {
        log.Fprintf(os.Stderr, "Error marshaling response: %v\n", err)
        return
    }
    fmt.Println(string(data))
    os.Stdout.Sync()
}

func (ws *WingmateServer) sendErrorResponse(id interface{}, code int, message string) {
    response := JSONRPCResponse{
        JSONRPC: "2.0",
        ID:      id,
        Error: map[string]interface{}{
            "code":    code,
            "message": message,
        },
    }
    ws.sendResponse(response)
}

func (ws *WingmateServer) run() {
    log.SetOutput(os.Stderr)
    log.Println("Wingmate MCP Server starting...")

    reader := bufio.NewReader(os.Stdin)

    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            if err == io.EOF {
                log.Println("EOF received, shutting down")
                break
            }
            continue
        }

        line = strings.TrimSpace(line)
        if line == "" {
            continue
        }

        var request JSONRPCRequest
        if err := json.Unmarshal([]byte(line), &request); err != nil {
            ws.sendErrorResponse(nil, -32700, "Parse error")
            continue
        }

        switch request.Method {
        case "initialize":
            ws.handleInitialize(request)
        case "notifications/initialized":
            ws.initialized = true
            log.Println("Client initialized")
        case "tools/list":
            if !ws.initialized {
                ws.sendErrorResponse(request.ID, -32600, "Not initialized")
                continue
            }
            ws.handleToolsList(request)
        case "tools/call":
            if !ws.initialized {
                ws.sendErrorResponse(request.ID, -32600, "Not initialized")
                continue
            }
            ws.handleToolsCall(request)
        case "ping":
            ws.handlePing(request)
        default:
            ws.sendErrorResponse(request.ID, -32601, "Method not found")
        }
    }
}

func main() {
    server := NewWingmateServer()
    server.run()
}
```

---

## Key Takeaways for Wingmate Implementation

1. **Wire Format**: Newline-delimited JSON over stdio
2. **Logging**: All debug output to stderr, only protocol to stdout
3. **Initialization**: Must complete handshake before processing tools
4. **Tool Errors vs Protocol Errors**: Use `isError: true` for tool failures, JSON-RPC errors for protocol violations
5. **Error Codes**: Use -32xxx for standard errors, custom range for app-specific
6. **Flush Output**: Call `os.Stdout.Sync()` after each response
7. **Graceful Shutdown**: Detect EOF on stdin and clean up

---

## Citations

Key sources:
- https://modelcontextprotocol.io/specification/2025-03-26/basic
- https://modelcontextprotocol.io/specification/2025-06-18/server/tools
- https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle
- https://foojay.io/today/understanding-mcp-through-raw-stdio-communication/
