# A2A Protocol Technical Constraints and Go Implementation Gotchas

**Research Focus**: Technical Investigator - A2A protocol constraints and Go-specific implementation challenges
**Date**: 2026-01-21
**Project**: Wingmate - Claude A2A Communication

---

## Discovery S2-01: JSON-RPC 2.0 Reserved Error Code Range

**Category**: Constraint
**Impact**: High
**Problem**: JSON-RPC 2.0 reserves error codes from -32768 to -32000 for protocol-level errors. Using codes in this range for application errors violates the specification and may cause interoperability issues with other A2A agents.

**Root Cause**: The [JSON-RPC 2.0 specification](https://www.jsonrpc.org/specification) defines strict error code ranges. Standard codes include:
- `-32700`: Parse error (invalid JSON)
- `-32600`: Invalid Request
- `-32601`: Method not found
- `-32602`: Invalid params
- `-32603`: Internal error
- `-32000` to `-32099`: Server-defined errors

**Solution**: Use positive integers or codes outside the reserved range for Wingmate-specific errors. Define an application error schema.

**Go Example**:
```go
// WRONG: Using reserved error codes
type JSONRPCError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

func handleError() JSONRPCError {
    return JSONRPCError{Code: -32001, Message: "Task not found"} // Reserved range!
}

// CORRECT: Use application-specific positive codes
const (
    ErrTaskNotFound      = 1001
    ErrAgentUnavailable  = 1002
    ErrCapabilityMissing = 1003
    ErrAuthExpired       = 1004
)

func handleError() JSONRPCError {
    return JSONRPCError{Code: ErrTaskNotFound, Message: "Task not found"}
}
```

**Action Required**: Define Wingmate error codes starting at 1000+. Document all custom error codes in agent-card metadata or separate schema. Ensure JSON-RPC error responses include required `id` field (must be null for parse errors where id cannot be determined).

---

## Discovery S2-02: Agent Card Location and Schema Requirements

**Category**: Constraint
**Impact**: Critical
**Problem**: A2A protocol mandates the Agent Card be served at `/.well-known/agent.json` (per RFC 8615). Earlier drafts used `/.well-known/agent-card.json` which may cause discovery failures with newer clients.

**Root Cause**: The [A2A Protocol Specification](https://a2a-protocol.org/latest/specification/) standardized on `/.well-known/agent.json` following IANA feedback. The Agent Card must include required fields: `name`, `url`, `version`, `capabilities`, and `skills`.

**Solution**: Serve Agent Card at the correct well-known URI. Include all required schema fields. Consider supporting both paths during transition period for backward compatibility.

**Go Example**:
```go
// WRONG: Incorrect path or missing required fields
mux.HandleFunc("/.well-known/agent-card.json", agentCardHandler) // Old path

type AgentCard struct {
    Name string `json:"name"`
    // Missing url, version, capabilities, skills
}

// CORRECT: Proper path and complete schema
mux.HandleFunc("/.well-known/agent.json", agentCardHandler)

type AgentCard struct {
    Name         string            `json:"name"`
    Description  string            `json:"description,omitempty"`
    URL          string            `json:"url"`          // REQUIRED: endpoint URL
    Version      string            `json:"version"`      // REQUIRED: protocol version
    Capabilities AgentCapabilities `json:"capabilities"` // REQUIRED
    Skills       []AgentSkill      `json:"skills"`       // REQUIRED: at least empty array
    Auth         *AuthConfig       `json:"authentication,omitempty"`
}

type AgentCapabilities struct {
    Streaming         bool `json:"streaming"`
    PushNotifications bool `json:"pushNotifications"`
}
```

**Action Required**: Implement Agent Card endpoint at `/.well-known/agent.json`. Validate all required fields are present. Set appropriate Cache-Control headers for card caching. Consider CORS headers if browser clients will fetch the card.

---

## Discovery S2-03: Go time.Time JSON Encoding with omitempty

**Category**: Framework Gotcha
**Impact**: High
**Problem**: The `omitempty` tag does NOT work with `time.Time` fields in Go. A zero-value `time.Time` will serialize as `"0001-01-01T00:00:00Z"` instead of being omitted, causing confusion in A2A message timestamps and task metadata.

**Root Cause**: According to [Go issue #42735](https://github.com/golang/go/issues/42735), `omitempty` never considers struct types empty, even if all fields are zero-valued. `time.Time` is a struct, so it is never omitted. Go 1.24 introduces `omitzero` which uses `IsZero()` method, but may not be available yet.

**Solution**: Use pointer types `*time.Time` for optional timestamp fields. Nil pointers are correctly omitted with `omitempty`.

**Go Example**:
```go
// WRONG: omitempty ignored for time.Time
type TaskStatus struct {
    ID        string    `json:"id"`
    CreatedAt time.Time `json:"createdAt,omitempty"` // Always serialized!
    UpdatedAt time.Time `json:"updatedAt,omitempty"` // Always serialized!
}

// Produces: {"id":"123","createdAt":"0001-01-01T00:00:00Z","updatedAt":"0001-01-01T00:00:00Z"}

// CORRECT: Use pointer for optional timestamps
type TaskStatus struct {
    ID        string     `json:"id"`
    CreatedAt time.Time  `json:"createdAt"`           // Required, always present
    UpdatedAt *time.Time `json:"updatedAt,omitempty"` // Optional, omitted when nil
}

// Helper for setting optional time
func timePtr(t time.Time) *time.Time {
    return &t
}

// Usage
status := TaskStatus{
    ID:        "123",
    CreatedAt: time.Now(),
    // UpdatedAt omitted - will not appear in JSON
}
```

**Action Required**: Audit all struct definitions for `time.Time` fields. Use `*time.Time` for optional timestamps. For Go 1.24+, consider using `omitzero` tag which properly checks `IsZero()`.

---

## Discovery S2-04: Go HTTP Server Context Cancellation Does Not Fire on WriteTimeout

**Category**: Framework Gotcha
**Impact**: High
**Problem**: When `http.Server.WriteTimeout` is reached, the request context (`r.Context()`) is NOT automatically canceled. Long-running handlers continue executing, potentially wasting resources or causing data corruption.

**Root Cause**: [Go issue #59602](https://github.com/golang/go/issues/59602) documents this behavior. The `WriteTimeout` only affects the network write deadline, not the request context. This means your handler code has no way to know the timeout was hit unless you implement your own deadline.

**Solution**: Use `http.TimeoutHandler` middleware to wrap handlers, or pass explicit context deadlines derived from configured timeouts.

**Go Example**:
```go
// WRONG: Relying on WriteTimeout to cancel context
server := &http.Server{
    Addr:         ":8080",
    WriteTimeout: 30 * time.Second, // Context NOT canceled when this fires
    Handler:      mux,
}

func slowHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    // This long operation continues even after WriteTimeout!
    result := expensiveOperation(ctx) // ctx never canceled by WriteTimeout
    json.NewEncoder(w).Encode(result) // Write fails silently
}

// CORRECT: Use TimeoutHandler or explicit deadline
func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/task", taskHandler)

    // Option 1: TimeoutHandler middleware
    handler := http.TimeoutHandler(mux, 30*time.Second, "Request timeout")

    server := &http.Server{
        Addr:    ":8080",
        Handler: handler,
    }
}

// Option 2: Explicit context deadline in handler
func taskHandler(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
    defer cancel()

    result, err := expensiveOperation(ctx)
    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            http.Error(w, "Processing timeout", http.StatusGatewayTimeout)
            return
        }
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(result)
}
```

**Action Required**: Do NOT rely on `WriteTimeout` for handler cancellation. Implement explicit context deadlines in A2A request handlers. Use `http.TimeoutHandler` as a safety net. Log when timeouts occur for debugging.

---

## Discovery S2-05: SSE Streaming Requires Explicit Flush and Flusher Check

**Category**: Framework Gotcha
**Impact**: Critical
**Problem**: Go's `http.ResponseWriter` buffers output by default. SSE events will not reach the client until the buffer fills or the handler returns, breaking real-time streaming for A2A task updates.

**Root Cause**: The `http.ResponseWriter` interface does not require `Flush()` support. You must type-assert to `http.Flusher` and call `Flush()` after every SSE event. Additionally, reverse proxies (nginx) may buffer responses unless `X-Accel-Buffering: no` header is set.

**Solution**: Check for `http.Flusher` support, set proper headers, and flush after every event. Use `http.NewResponseController` in modern Go.

**Go Example**:
```go
// WRONG: No flush, no flusher check
func sseHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")

    for event := range events {
        fmt.Fprintf(w, "data: %s\n\n", event)
        // Client sees nothing until handler returns or buffer fills!
    }
}

// CORRECT: Full SSE implementation
func sseHandler(w http.ResponseWriter, r *http.Request) {
    // Check Flusher support
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "SSE not supported", http.StatusInternalServerError)
        return
    }

    // Set required headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no") // For nginx

    ctx := r.Context()
    keepAlive := time.NewTicker(15 * time.Second)
    defer keepAlive.Stop()

    for {
        select {
        case <-ctx.Done():
            return // Client disconnected
        case event := <-events:
            _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.Data)
            if err != nil {
                return // Write failed, client gone
            }
            flusher.Flush() // CRITICAL: Flush after every event
        case <-keepAlive.C:
            fmt.Fprintf(w, ": keepalive\n\n") // SSE comment as heartbeat
            flusher.Flush()
        }
    }
}

// Modern approach with ResponseController (Go 1.20+)
func sseHandlerModern(w http.ResponseWriter, r *http.Request) {
    rc := http.NewResponseController(w)

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")

    for event := range events {
        fmt.Fprintf(w, "data: %s\n\n", event)
        if err := rc.Flush(); err != nil {
            return // Client disconnected
        }
    }
}
```

**Action Required**: Always check `http.Flusher` support before SSE streaming. Set all four headers (Content-Type, Cache-Control, Connection, X-Accel-Buffering). Implement keep-alive pings (every 15-30s) to prevent proxy/load balancer timeouts. Check flush errors to detect client disconnection.

---

## Discovery S2-06: A2A Message Part Types Require Discriminator Field

**Category**: Constraint
**Impact**: High
**Problem**: A2A message parts (TextPart, DataPart, FilePart) use a `kind` discriminator field for type identification. Go's standard JSON unmarshaling cannot handle union types without custom logic, leading to incorrect deserialization.

**Root Cause**: The [A2A Protocol](https://a2a-protocol.org/latest/topics/key-concepts/) defines Part as a union type with `kind` field values of `"text"`, `"data"`, or `"file"`. Go has no native union type support, requiring manual two-phase unmarshaling.

**Solution**: Implement custom `UnmarshalJSON` that first reads the `kind` field, then deserializes into the correct concrete type.

**Go Example**:
```go
// WRONG: Single struct cannot represent union type
type Part struct {
    Kind     string         `json:"kind"`
    Text     string         `json:"text,omitempty"`
    Data     map[string]any `json:"data,omitempty"`
    File     *FileContent   `json:"file,omitempty"`
}
// Problem: All fields present, unclear which to use

// CORRECT: Union type with custom unmarshaling
type Part interface {
    partMarker()
}

type TextPart struct {
    Kind     string         `json:"kind"` // Always "text"
    Text     string         `json:"text"`
    Metadata map[string]any `json:"metadata,omitempty"`
}
func (TextPart) partMarker() {}

type DataPart struct {
    Kind     string         `json:"kind"` // Always "data"
    Data     map[string]any `json:"data"`
    Metadata map[string]any `json:"metadata,omitempty"`
}
func (DataPart) partMarker() {}

type FilePart struct {
    Kind     string         `json:"kind"` // Always "file"
    File     FileContent    `json:"file"`
    Metadata map[string]any `json:"metadata,omitempty"`
}
func (FilePart) partMarker() {}

type FileContent struct {
    URI       string `json:"uri,omitempty"`       // FileWithUri
    Bytes     string `json:"bytes,omitempty"`     // FileWithBytes (base64)
    MediaType string `json:"mediaType,omitempty"`
    Name      string `json:"name,omitempty"`
}

// Custom unmarshaler for Part slice
type PartList []Part

func (pl *PartList) UnmarshalJSON(data []byte) error {
    var raw []json.RawMessage
    if err := json.Unmarshal(data, &raw); err != nil {
        return err
    }

    *pl = make([]Part, len(raw))
    for i, r := range raw {
        var probe struct {
            Kind string `json:"kind"`
        }
        if err := json.Unmarshal(r, &probe); err != nil {
            return fmt.Errorf("part %d: %w", i, err)
        }

        var part Part
        switch probe.Kind {
        case "text":
            var tp TextPart
            if err := json.Unmarshal(r, &tp); err != nil {
                return fmt.Errorf("text part %d: %w", i, err)
            }
            part = tp
        case "data":
            var dp DataPart
            if err := json.Unmarshal(r, &dp); err != nil {
                return fmt.Errorf("data part %d: %w", i, err)
            }
            part = dp
        case "file":
            var fp FilePart
            if err := json.Unmarshal(r, &fp); err != nil {
                return fmt.Errorf("file part %d: %w", i, err)
            }
            part = fp
        default:
            return fmt.Errorf("part %d: unknown kind %q", i, probe.Kind)
        }
        (*pl)[i] = part
    }
    return nil
}
```

**Action Required**: Implement union type handling for A2A Parts. Validate `kind` field matches expected values. FilePart must have exactly one of `uri` or `bytes`. Include `metadata` support on all part types for forward compatibility.

---

## Discovery S2-07: Graceful Shutdown Must Handle In-Flight A2A Tasks

**Category**: Framework Gotcha
**Impact**: Critical
**Problem**: Calling `server.Shutdown()` stops accepting new connections but in-flight SSE streams and long-running task handlers may be abruptly terminated if shutdown timeout is too short, causing task state corruption.

**Root Cause**: The [http.Server.Shutdown](https://pkg.go.dev/net/http#Server.Shutdown) waits for connections to become idle, but SSE connections never become idle. Background goroutines spawned by handlers are not tracked by the server. Kubernetes sends SIGKILL after 30 seconds by default.

**Solution**: Use context propagation with `BaseContext`, implement WaitGroup for background work, and set shutdown timeout to 80% of Kubernetes grace period.

**Go Example**:
```go
// WRONG: No coordination between shutdown and task handlers
func main() {
    server := &http.Server{Addr: ":8080", Handler: mux}

    go server.ListenAndServe()

    <-sigChan
    server.Shutdown(context.Background()) // SSE streams hang forever
}

// CORRECT: Coordinated shutdown with task tracking
type Server struct {
    httpServer *http.Server
    taskWG     sync.WaitGroup
    shutdownCh chan struct{}
}

func NewServer() *Server {
    s := &Server{
        shutdownCh: make(chan struct{}),
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/a2a", s.a2aHandler)

    s.httpServer = &http.Server{
        Addr:    ":8080",
        Handler: mux,
        BaseContext: func(l net.Listener) context.Context {
            // All request contexts derive from this
            ctx, _ := context.WithCancel(context.Background())
            return ctx
        },
    }
    return s
}

func (s *Server) a2aHandler(w http.ResponseWriter, r *http.Request) {
    s.taskWG.Add(1)
    defer s.taskWG.Done()

    ctx := r.Context()

    select {
    case <-ctx.Done():
        return // Client disconnected or server shutting down
    case <-s.shutdownCh:
        return // Graceful shutdown signal
    case result := <-processTask(ctx):
        json.NewEncoder(w).Encode(result)
    }
}

func (s *Server) Shutdown() error {
    // Signal handlers to stop accepting new work
    close(s.shutdownCh)

    // Kubernetes default is 30s, use 24s (80%)
    ctx, cancel := context.WithTimeout(context.Background(), 24*time.Second)
    defer cancel()

    // Stop accepting new connections
    if err := s.httpServer.Shutdown(ctx); err != nil {
        log.Printf("HTTP shutdown error: %v", err)
    }

    // Wait for in-flight tasks with timeout
    done := make(chan struct{})
    go func() {
        s.taskWG.Wait()
        close(done)
    }()

    select {
    case <-done:
        log.Println("All tasks completed")
    case <-ctx.Done():
        log.Println("Shutdown timeout, some tasks may be incomplete")
    }

    return nil
}

func main() {
    server := NewServer()

    go server.httpServer.ListenAndServe()

    stop := make(chan os.Signal, 1)
    signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
    <-stop

    log.Println("Shutting down gracefully...")
    server.Shutdown()
}
```

**Action Required**: Implement task tracking with `sync.WaitGroup`. Use shutdown channel to signal handlers. Set shutdown timeout to 80% of infrastructure grace period. Persist incomplete task state for recovery if needed. Log incomplete tasks on forced shutdown.

---

## Discovery S2-08: JSON Null vs Empty Array/Object Semantics

**Category**: Constraint
**Impact**: Medium
**Problem**: A2A protocol distinguishes between `null`, empty arrays `[]`, and absent fields. Go's `nil` slice encodes to `null`, not `[]`, which may violate A2A schema expectations for arrays like `skills` or `parts`.

**Root Cause**: Per [Go JSON encoding](https://pkg.go.dev/encoding/json), nil slices marshal to `null` while empty but non-nil slices marshal to `[]`. A2A schemas may require empty arrays rather than null for certain fields.

**Solution**: Initialize slices to empty (not nil) when the field must serialize as `[]`. Use explicit slice literals.

**Go Example**:
```go
// WRONG: Nil slices become null
type AgentCard struct {
    Name   string       `json:"name"`
    Skills []AgentSkill `json:"skills"` // Required field
}

card := AgentCard{Name: "wingmate"}
// Produces: {"name":"wingmate","skills":null}
// A2A may expect: {"name":"wingmate","skills":[]}

// WRONG: omitempty hides required field
type AgentCard struct {
    Name   string       `json:"name"`
    Skills []AgentSkill `json:"skills,omitempty"` // Omitted when nil!
}
// Produces: {"name":"wingmate"}
// Missing required "skills" field!

// CORRECT: Initialize required arrays
type AgentCard struct {
    Name   string       `json:"name"`
    Skills []AgentSkill `json:"skills"`
}

func NewAgentCard(name string) AgentCard {
    return AgentCard{
        Name:   name,
        Skills: []AgentSkill{}, // Empty slice, not nil
    }
}
// Produces: {"name":"wingmate","skills":[]}

// CORRECT: Constructor pattern ensures valid state
type Message struct {
    Role  string `json:"role"`
    Parts []Part `json:"parts"`
}

func NewMessage(role string) Message {
    return Message{
        Role:  role,
        Parts: make([]Part, 0), // Explicit empty slice
    }
}

// For optional arrays that CAN be null
type TaskResult struct {
    Artifacts []Artifact `json:"artifacts,omitempty"` // null/absent OK
}
```

**Action Required**: Identify which A2A fields require `[]` vs allow `null`. Initialize required array fields in constructors. Use `make([]T, 0)` or `[]T{}` for empty-not-null semantics. Add JSON schema validation tests to catch null vs empty mismatches.

---

## Summary Table

| Discovery | Category | Impact | Key Takeaway |
|-----------|----------|--------|--------------|
| S2-01 | Constraint | High | Use error codes outside -32768 to -32000 range |
| S2-02 | Constraint | Critical | Serve Agent Card at `/.well-known/agent.json` |
| S2-03 | Framework Gotcha | High | Use `*time.Time` for optional timestamps |
| S2-04 | Framework Gotcha | High | WriteTimeout does not cancel context |
| S2-05 | Framework Gotcha | Critical | SSE requires Flusher check and explicit Flush() |
| S2-06 | Constraint | High | Implement custom unmarshaling for Part union types |
| S2-07 | Framework Gotcha | Critical | Track in-flight tasks for graceful shutdown |
| S2-08 | Constraint | Medium | Initialize required arrays to empty, not nil |

---

## Sources

- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)
- [A2A Protocol Specification](https://a2a-protocol.org/latest/specification/)
- [A2A Core Concepts](https://a2a-protocol.org/latest/topics/key-concepts/)
- [Go JSON encoding/json](https://pkg.go.dev/encoding/json)
- [Go issue #42735 - omitempty time.Time](https://github.com/golang/go/issues/42735)
- [Go issue #59602 - WriteTimeout context](https://github.com/golang/go/issues/59602)
- [Graceful Shutdown in Go](https://victoriametrics.com/blog/go-graceful-shutdown/)
- [SSE Implementation in Go](https://dev.to/mirzaakhena/server-sent-events-sse-server-implementation-with-go-4ck2)
- [Go JSON Gotchas](https://www.alexedwards.net/blog/json-surprises-and-gotchas)
