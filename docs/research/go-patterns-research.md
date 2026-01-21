# Go Patterns Research for Wingmate A2A Implementation

**Research Session**: S1 - Codebase Pattern Analysis
**Date**: 2026-01-21
**Focus**: Go idioms, HTTP patterns, CLI conventions, and A2A implementation approaches

---

## Discovery S1-01: Go Project Layout Convention

**Category**: Convention
**Impact**: Critical

**What**: Go projects follow a standard layout with `cmd/`, `internal/`, and `pkg/` directories. For Wingmate, this maps to separate entry points for Pilot (client) and Wingmate (server) binaries.

**Why It Matters**: Proper layout enables clean separation of public APIs from internal implementation, supports multiple binaries from a single repo, and aligns with Go community expectations for maintainability.

**Go Example**:
```go
// Project structure for Wingmate
//
// wingmate/
// ├── cmd/
// │   ├── pilot/           # Client CLI binary
// │   │   └── main.go
// │   └── wingmate/        # Server binary
// │       └── main.go
// ├── internal/
// │   ├── a2a/             # A2A protocol implementation (not exported)
// │   │   ├── client.go
// │   │   ├── server.go
// │   │   └── types.go
// │   ├── flightlog/       # JSONL logging (not exported)
// │   │   └── writer.go
// │   └── agentcard/       # Agent card handling
// │       └── card.go
// ├── pkg/                 # Public APIs (if any third-party consumers)
// │   └── protocol/        # Exported A2A types
// │       └── messages.go
// └── go.mod

// cmd/pilot/main.go
package main

import (
    "os"

    "github.com/wingmate/wingmate/internal/a2a"
)

func main() {
    client := a2a.NewClient(os.Args[1:])
    if err := client.Run(); err != nil {
        os.Exit(1)
    }
}
```

**Action Required**: Create directory structure with `cmd/pilot/`, `cmd/wingmate/`, and `internal/` for core logic. Use `internal/` to prevent external imports of implementation details.

---

## Discovery S1-02: JSON-RPC 2.0 HTTP Server Pattern

**Category**: Pattern
**Impact**: Critical

**What**: A2A uses JSON-RPC 2.0 over HTTP. Go's `net/http` with a custom handler provides lightweight, dependency-free implementation. Use a single endpoint that dispatches by method name.

**Why It Matters**: The A2A protocol requires proper JSON-RPC compliance including request IDs, error codes, and batch support. A clean dispatcher pattern keeps the server maintainable.

**Go Example**:
```go
// internal/a2a/server.go
package a2a

import (
    "encoding/json"
    "net/http"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
    JSONRPC string          `json:"jsonrpc"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
    ID      interface{}     `json:"id,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
    JSONRPC string          `json:"jsonrpc"`
    Result  interface{}     `json:"result,omitempty"`
    Error   *JSONRPCError   `json:"error,omitempty"`
    ID      interface{}     `json:"id"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
    ErrParseError     = -32700
    ErrInvalidRequest = -32600
    ErrMethodNotFound = -32601
    ErrInvalidParams  = -32602
    ErrInternal       = -32603
)

// Server handles A2A JSON-RPC requests
type Server struct {
    handlers map[string]HandlerFunc
}

type HandlerFunc func(params json.RawMessage) (interface{}, error)

func NewServer() *Server {
    return &Server{
        handlers: make(map[string]HandlerFunc),
    }
}

func (s *Server) Handle(method string, handler HandlerFunc) {
    s.handlers[method] = handler
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req JSONRPCRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        s.writeError(w, nil, ErrParseError, "Parse error")
        return
    }

    if req.JSONRPC != "2.0" {
        s.writeError(w, req.ID, ErrInvalidRequest, "Invalid JSON-RPC version")
        return
    }

    handler, ok := s.handlers[req.Method]
    if !ok {
        s.writeError(w, req.ID, ErrMethodNotFound, "Method not found")
        return
    }

    result, err := handler(req.Params)
    if err != nil {
        s.writeError(w, req.ID, ErrInternal, err.Error())
        return
    }

    s.writeResult(w, req.ID, result)
}

func (s *Server) writeResult(w http.ResponseWriter, id interface{}, result interface{}) {
    resp := JSONRPCResponse{
        JSONRPC: "2.0",
        Result:  result,
        ID:      id,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}

func (s *Server) writeError(w http.ResponseWriter, id interface{}, code int, message string) {
    resp := JSONRPCResponse{
        JSONRPC: "2.0",
        Error:   &JSONRPCError{Code: code, Message: message},
        ID:      id,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}
```

**Action Required**: Implement JSON-RPC server with method dispatcher. Register A2A methods like `tasks/send`, `message/send`, and `tasks/get`.

---

## Discovery S1-03: Server-Sent Events (SSE) for Streaming

**Category**: Pattern
**Impact**: High

**What**: A2A uses SSE for streaming task updates. Go's `http.Flusher` interface enables SSE without external dependencies. Each event is a JSON-RPC response fragment.

**Why It Matters**: Streaming enables real-time updates during long-running diagnostic tasks. Proper SSE implementation prevents connection drops and enables graceful recovery.

**Go Example**:
```go
// internal/a2a/sse.go
package a2a

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

// TaskEvent represents a streaming task update
type TaskEvent struct {
    Type   string      `json:"type"` // "status", "artifact", "message"
    TaskID string      `json:"taskId"`
    Data   interface{} `json:"data"`
}

// StreamHandler handles SSE streaming for task subscriptions
func (s *Server) StreamHandler(w http.ResponseWriter, r *http.Request, taskID string, events <-chan TaskEvent) {
    // Verify SSE support
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "SSE not supported", http.StatusInternalServerError)
        return
    }

    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

    ctx := r.Context()

    for {
        select {
        case <-ctx.Done():
            // Client disconnected
            return
        case event, ok := <-events:
            if !ok {
                // Channel closed, task complete
                fmt.Fprintf(w, "event: done\ndata: {\"taskId\":\"%s\"}\n\n", taskID)
                flusher.Flush()
                return
            }

            data, err := json.Marshal(event)
            if err != nil {
                continue
            }

            fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
            flusher.Flush()
        }
    }
}

// TaskManager manages running tasks and their event channels
type TaskManager struct {
    tasks map[string]chan TaskEvent
}

func NewTaskManager() *TaskManager {
    return &TaskManager{
        tasks: make(map[string]chan TaskEvent),
    }
}

func (tm *TaskManager) CreateTask(id string) chan TaskEvent {
    ch := make(chan TaskEvent, 100) // Buffered to prevent blocking
    tm.tasks[id] = ch
    return ch
}

func (tm *TaskManager) SendEvent(id string, event TaskEvent) {
    if ch, ok := tm.tasks[id]; ok {
        select {
        case ch <- event:
        default:
            // Channel full, event dropped (consider logging)
        }
    }
}

func (tm *TaskManager) CompleteTask(id string) {
    if ch, ok := tm.tasks[id]; ok {
        close(ch)
        delete(tm.tasks, id)
    }
}
```

**Action Required**: Implement SSE handler for `tasks/sendSubscribe` method. Use buffered channels to prevent blocking on slow clients.

---

## Discovery S1-04: CLI Argument Parsing with Standard Library

**Category**: Convention
**Impact**: Medium

**What**: For simple CLIs, Go's standard `flag` package is sufficient and dependency-free. For complex CLIs with subcommands, consider `cobra` or build a lightweight subcommand dispatcher.

**Why It Matters**: Pilot CLI needs minimal flags (server URL, auth). Wingmate server needs config flags (port, log path). Keeping dependencies minimal reduces attack surface and build complexity.

**Go Example**:
```go
// cmd/pilot/main.go - Simple flag-based CLI
package main

import (
    "flag"
    "fmt"
    "os"
)

type Config struct {
    ServerURL string
    AuthToken string
    LogPath   string
    Verbose   bool
}

func main() {
    cfg := parseFlags()

    if err := run(cfg); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}

func parseFlags() Config {
    cfg := Config{}

    flag.StringVar(&cfg.ServerURL, "server", "http://localhost:8080", "Wingmate server URL")
    flag.StringVar(&cfg.AuthToken, "token", "", "Authentication token")
    flag.StringVar(&cfg.LogPath, "log", "", "Path to flight log file (JSONL)")
    flag.BoolVar(&cfg.Verbose, "v", false, "Verbose output")

    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage: pilot [options] <command>\n\n")
        fmt.Fprintf(os.Stderr, "Commands:\n")
        fmt.Fprintf(os.Stderr, "  query <message>    Send a diagnostic query\n")
        fmt.Fprintf(os.Stderr, "  status             Check connection status\n")
        fmt.Fprintf(os.Stderr, "\nOptions:\n")
        flag.PrintDefaults()
    }

    flag.Parse()
    return cfg
}

// Subcommand pattern without external dependencies
func run(cfg Config) error {
    args := flag.Args()
    if len(args) == 0 {
        flag.Usage()
        return fmt.Errorf("no command specified")
    }

    switch args[0] {
    case "query":
        if len(args) < 2 {
            return fmt.Errorf("query requires a message argument")
        }
        return runQuery(cfg, args[1])
    case "status":
        return runStatus(cfg)
    default:
        return fmt.Errorf("unknown command: %s", args[0])
    }
}

func runQuery(cfg Config, message string) error {
    // Implementation here
    return nil
}

func runStatus(cfg Config) error {
    // Implementation here
    return nil
}
```

**Action Required**: Use `flag` package for initial implementation. Define flags for server URL, auth token, and log output path. Add subcommand support for `query`, `status`, and `serve` commands.

---

## Discovery S1-05: Structured JSONL Logging Pattern

**Category**: Pattern
**Impact**: High

**What**: Flight Log uses JSONL (newline-delimited JSON) for structured logging. Each line is a complete JSON object with timestamp, level, and structured fields. Use `encoding/json` with a custom writer.

**Why It Matters**: JSONL enables easy parsing, streaming, and analysis of agent communications. Critical for debugging multi-agent interactions and audit trails.

**Go Example**:
```go
// internal/flightlog/writer.go
package flightlog

import (
    "encoding/json"
    "io"
    "os"
    "sync"
    "time"
)

// Entry represents a single log entry
type Entry struct {
    Timestamp time.Time              `json:"timestamp"`
    Level     string                 `json:"level"`
    Type      string                 `json:"type"` // "request", "response", "event", "error"
    TaskID    string                 `json:"taskId,omitempty"`
    Method    string                 `json:"method,omitempty"`
    Duration  int64                  `json:"durationMs,omitempty"`
    Message   string                 `json:"message,omitempty"`
    Data      map[string]interface{} `json:"data,omitempty"`
    Error     string                 `json:"error,omitempty"`
}

// Writer writes JSONL formatted log entries
type Writer struct {
    mu     sync.Mutex
    w      io.Writer
    enc    *json.Encoder
}

// NewWriter creates a new JSONL writer
func NewWriter(w io.Writer) *Writer {
    return &Writer{
        w:   w,
        enc: json.NewEncoder(w),
    }
}

// NewFileWriter creates a writer that appends to a file
func NewFileWriter(path string) (*Writer, error) {
    f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return nil, err
    }
    return NewWriter(f), nil
}

// Write writes a log entry
func (w *Writer) Write(entry Entry) error {
    w.mu.Lock()
    defer w.mu.Unlock()

    if entry.Timestamp.IsZero() {
        entry.Timestamp = time.Now().UTC()
    }

    return w.enc.Encode(entry)
}

// Convenience methods for common log types
func (w *Writer) Request(taskID, method string, data map[string]interface{}) {
    w.Write(Entry{
        Level:  "info",
        Type:   "request",
        TaskID: taskID,
        Method: method,
        Data:   data,
    })
}

func (w *Writer) Response(taskID, method string, durationMs int64, data map[string]interface{}) {
    w.Write(Entry{
        Level:    "info",
        Type:     "response",
        TaskID:   taskID,
        Method:   method,
        Duration: durationMs,
        Data:     data,
    })
}

func (w *Writer) Error(taskID string, err error, data map[string]interface{}) {
    w.Write(Entry{
        Level:  "error",
        Type:   "error",
        TaskID: taskID,
        Error:  err.Error(),
        Data:   data,
    })
}

// Example JSONL output:
// {"timestamp":"2026-01-21T10:30:00Z","level":"info","type":"request","taskId":"abc123","method":"tasks/send","data":{"skill":"GetLogs"}}
// {"timestamp":"2026-01-21T10:30:01Z","level":"info","type":"response","taskId":"abc123","method":"tasks/send","durationMs":1234,"data":{"status":"completed"}}
```

**Action Required**: Implement JSONL writer in `internal/flightlog/`. Include timestamps, task IDs, and structured data fields. Ensure thread-safe writes with mutex.

---

## Discovery S1-06: Table-Driven Tests Pattern

**Category**: Convention
**Impact**: Medium

**What**: Go's testing convention uses table-driven tests for comprehensive coverage with minimal code. Use subtests with `t.Run()` for clear test output and parallel execution.

**Why It Matters**: A2A message handling has many edge cases (invalid JSON, missing fields, unknown methods). Table-driven tests make it easy to add cases and identify failures.

**Go Example**:
```go
// internal/a2a/server_test.go
package a2a

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestJSONRPCServer(t *testing.T) {
    server := NewServer()
    server.Handle("echo", func(params json.RawMessage) (interface{}, error) {
        return json.RawMessage(params), nil
    })

    tests := []struct {
        name           string
        requestBody    string
        wantStatusCode int
        wantError      *int // nil means expect success
        wantResult     string
    }{
        {
            name:           "valid request",
            requestBody:    `{"jsonrpc":"2.0","method":"echo","params":{"msg":"hello"},"id":1}`,
            wantStatusCode: http.StatusOK,
            wantResult:     `{"msg":"hello"}`,
        },
        {
            name:           "invalid JSON",
            requestBody:    `{invalid`,
            wantStatusCode: http.StatusOK,
            wantError:      intPtr(ErrParseError),
        },
        {
            name:           "method not found",
            requestBody:    `{"jsonrpc":"2.0","method":"unknown","params":{},"id":1}`,
            wantStatusCode: http.StatusOK,
            wantError:      intPtr(ErrMethodNotFound),
        },
        {
            name:           "invalid version",
            requestBody:    `{"jsonrpc":"1.0","method":"echo","params":{},"id":1}`,
            wantStatusCode: http.StatusOK,
            wantError:      intPtr(ErrInvalidRequest),
        },
        {
            name:           "notification (no id)",
            requestBody:    `{"jsonrpc":"2.0","method":"echo","params":{}}`,
            wantStatusCode: http.StatusOK,
            // Notifications should still return a response
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.requestBody))
            req.Header.Set("Content-Type", "application/json")
            rec := httptest.NewRecorder()

            server.ServeHTTP(rec, req)

            if rec.Code != tt.wantStatusCode {
                t.Errorf("status code = %d, want %d", rec.Code, tt.wantStatusCode)
            }

            var resp JSONRPCResponse
            if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
                t.Fatalf("failed to decode response: %v", err)
            }

            if tt.wantError != nil {
                if resp.Error == nil {
                    t.Error("expected error, got success")
                } else if resp.Error.Code != *tt.wantError {
                    t.Errorf("error code = %d, want %d", resp.Error.Code, *tt.wantError)
                }
            }

            if tt.wantResult != "" {
                got, _ := json.Marshal(resp.Result)
                if string(got) != tt.wantResult {
                    t.Errorf("result = %s, want %s", got, tt.wantResult)
                }
            }
        })
    }
}

func intPtr(i int) *int { return &i }

// Test fixtures pattern
func TestWithFixtures(t *testing.T) {
    // Load test fixtures from testdata directory
    // testdata/ is ignored by Go tooling but included in module
    fixtures := []string{
        "testdata/valid_task_request.json",
        "testdata/agent_card.json",
    }

    for _, f := range fixtures {
        t.Run(f, func(t *testing.T) {
            // Load and test with fixture
        })
    }
}
```

**Action Required**: Create `*_test.go` files alongside implementation. Use table-driven tests for JSON-RPC parsing, message validation, and error handling. Store test fixtures in `testdata/` directories.

---

## Discovery S1-07: Agent Card JSON Schema and Serving

**Category**: Integration
**Impact**: High

**What**: A2A agents serve their Agent Card at `/.well-known/agent-card.json`. The card describes capabilities, auth requirements, and endpoint URL. Go structs map directly to the JSON schema.

**Why It Matters**: Agent Card is the discovery mechanism. Wingmate server must serve its card for Pilot to understand available capabilities and auth requirements.

**Go Example**:
```go
// internal/agentcard/card.go
package agentcard

import (
    "encoding/json"
    "net/http"
)

// Card represents an A2A Agent Card
type Card struct {
    Name          string       `json:"name"`
    Description   string       `json:"description"`
    URL           string       `json:"url"`
    Version       string       `json:"version"`
    Capabilities  []Capability `json:"capabilities"`
    Authentication *Auth       `json:"authentication,omitempty"`
}

// Capability describes a skill the agent can perform
type Capability struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Parameters  map[string]ParamSchema `json:"parameters,omitempty"`
}

// ParamSchema describes a parameter's type and constraints
type ParamSchema struct {
    Type        string `json:"type"`
    Description string `json:"description,omitempty"`
    Required    bool   `json:"required,omitempty"`
}

// Auth describes supported authentication methods
type Auth struct {
    Schemes []string `json:"schemes"` // "Bearer", "ApiKey", etc.
}

// Handler returns an HTTP handler that serves the agent card
func Handler(card *Card) http.HandlerFunc {
    cardJSON, _ := json.MarshalIndent(card, "", "  ")

    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Header().Set("Cache-Control", "public, max-age=3600")
        w.Write(cardJSON)
    }
}

// Example Agent Card for Wingmate
func NewWingmateCard(baseURL string) *Card {
    return &Card{
        Name:        "Wingmate",
        Description: "Claude A2A diagnostic agent for debugging assistance",
        URL:         baseURL,
        Version:     "1.0.0",
        Capabilities: []Capability{
            {
                Name:        "GetLogs",
                Description: "Retrieve application logs for analysis",
                Parameters: map[string]ParamSchema{
                    "since": {Type: "string", Description: "ISO8601 timestamp"},
                    "lines": {Type: "integer", Description: "Max lines to return"},
                },
            },
            {
                Name:        "GetMetrics",
                Description: "Retrieve system metrics (CPU, memory, etc.)",
                Parameters:  map[string]ParamSchema{},
            },
            {
                Name:        "AnalyzeError",
                Description: "Analyze an error message with context",
                Parameters: map[string]ParamSchema{
                    "error":   {Type: "string", Required: true},
                    "context": {Type: "string"},
                },
            },
        },
        Authentication: &Auth{
            Schemes: []string{"Bearer"},
        },
    }
}

// Setup registers the agent card endpoint
func Setup(mux *http.ServeMux, card *Card) {
    mux.HandleFunc("/.well-known/agent-card.json", Handler(card))
}
```

**Action Required**: Define Agent Card struct matching A2A spec. Serve at `/.well-known/agent-card.json`. Include all Wingmate capabilities (GetLogs, GetMetrics, AnalyzeError, etc.).

---

## Discovery S1-08: HTTP Client with Context and Timeouts

**Category**: Pattern
**Impact**: High

**What**: Go HTTP clients must use `context.Context` for cancellation and timeouts. For A2A, the client needs connection pooling, proper timeout handling, and SSE stream reading.

**Why It Matters**: Pilot must reliably communicate with Wingmate server, handle network issues gracefully, and support streaming responses for long-running tasks.

**Go Example**:
```go
// internal/a2a/client.go
package a2a

import (
    "bufio"
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

// Client is an A2A protocol client
type Client struct {
    baseURL    string
    httpClient *http.Client
    authToken  string
}

// NewClient creates a new A2A client
func NewClient(baseURL, authToken string) *Client {
    return &Client{
        baseURL:   baseURL,
        authToken: authToken,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        10,
                IdleConnTimeout:     90 * time.Second,
                DisableCompression:  false,
            },
        },
    }
}

// Call makes a JSON-RPC call to the server
func (c *Client) Call(ctx context.Context, method string, params interface{}) (*JSONRPCResponse, error) {
    req := JSONRPCRequest{
        JSONRPC: "2.0",
        Method:  method,
        ID:      time.Now().UnixNano(), // Simple unique ID
    }

    if params != nil {
        paramsJSON, err := json.Marshal(params)
        if err != nil {
            return nil, fmt.Errorf("marshal params: %w", err)
        }
        req.Params = paramsJSON
    }

    body, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("marshal request: %w", err)
    }

    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    httpReq.Header.Set("Content-Type", "application/json")
    if c.authToken != "" {
        httpReq.Header.Set("Authorization", "Bearer "+c.authToken)
    }

    httpResp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("http request: %w", err)
    }
    defer httpResp.Body.Close()

    var resp JSONRPCResponse
    if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    return &resp, nil
}

// Subscribe opens an SSE stream for task updates
func (c *Client) Subscribe(ctx context.Context, method string, params interface{}) (<-chan TaskEvent, error) {
    req := JSONRPCRequest{
        JSONRPC: "2.0",
        Method:  method,
        ID:      time.Now().UnixNano(),
    }

    if params != nil {
        paramsJSON, _ := json.Marshal(params)
        req.Params = paramsJSON
    }

    body, _ := json.Marshal(req)

    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
    if err != nil {
        return nil, err
    }

    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Accept", "text/event-stream")
    if c.authToken != "" {
        httpReq.Header.Set("Authorization", "Bearer "+c.authToken)
    }

    httpResp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, err
    }

    events := make(chan TaskEvent)

    go func() {
        defer close(events)
        defer httpResp.Body.Close()

        scanner := bufio.NewScanner(httpResp.Body)
        for scanner.Scan() {
            line := scanner.Text()
            if len(line) > 6 && line[:6] == "data: " {
                var event TaskEvent
                if err := json.Unmarshal([]byte(line[6:]), &event); err == nil {
                    select {
                    case events <- event:
                    case <-ctx.Done():
                        return
                    }
                }
            }
        }
    }()

    return events, nil
}

// FetchAgentCard retrieves the agent card from a server
func (c *Client) FetchAgentCard(ctx context.Context) (*Card, error) {
    httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/.well-known/agent-card.json", nil)
    if err != nil {
        return nil, err
    }

    httpResp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer httpResp.Body.Close()

    var card Card
    if err := json.NewDecoder(httpResp.Body).Decode(&card); err != nil {
        return nil, err
    }

    return &card, nil
}
```

**Action Required**: Implement HTTP client with proper context handling, timeouts, and connection pooling. Support both synchronous calls and SSE streaming. Include agent card fetching for capability discovery.

---

## Summary

| ID | Discovery | Category | Impact | Key Action |
|----|-----------|----------|--------|------------|
| S1-01 | Project Layout | Convention | Critical | Use cmd/, internal/, pkg/ structure |
| S1-02 | JSON-RPC Server | Pattern | Critical | Implement method dispatcher |
| S1-03 | SSE Streaming | Pattern | High | Use http.Flusher for events |
| S1-04 | CLI Flags | Convention | Medium | Use standard flag package |
| S1-05 | JSONL Logging | Pattern | High | Thread-safe JSON line writer |
| S1-06 | Table-Driven Tests | Convention | Medium | Comprehensive edge case coverage |
| S1-07 | Agent Card | Integration | High | Serve at well-known path |
| S1-08 | HTTP Client | Pattern | High | Context-aware with SSE support |

## References

- A2A Protocol Specification: https://a2a-protocol.org/latest/specification/
- Go Project Layout: https://github.com/golang-standards/project-layout
- JSON-RPC 2.0 Specification: https://www.jsonrpc.org/specification
- Server-Sent Events: https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events
