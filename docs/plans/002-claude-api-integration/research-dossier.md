# Research Dossier: Claude API Integration

**Feature**: LLM Support - Claude API Integration
**Research Date**: 2026-01-21
**Research Type**: Deep Exploration
**Status**: COMPLETE

---

## Executive Summary

This research explores adding LLM (Large Language Model) support to Wingmate, starting with Claude API integration. The goal is to enable agents to leverage Claude's capabilities for intelligent message processing, debugging assistance, and natural language understanding in A2A conversations.

**Key Finding**: The existing codebase is well-architected for this integration. The message structure (`Message` with `Parts`), HTTP client patterns, configuration system, and error handling infrastructure all provide solid foundations. The main integration point is `HandleMessage()` in `internal/agent/agent.go:227`.

---

## Table of Contents

1. [Integration Points](#1-integration-points)
2. [Message Structure Compatibility](#2-message-structure-compatibility)
3. [Configuration Patterns](#3-configuration-patterns)
4. [HTTP Client Infrastructure](#4-http-client-infrastructure)
5. [Error Handling](#5-error-handling)
6. [Testing Patterns](#6-testing-patterns)
7. [Observability](#7-observability)
8. [External Research Gaps](#8-external-research-gaps)
9. [Architecture Recommendations](#9-architecture-recommendations)
10. [Risk Assessment](#10-risk-assessment)

---

## 1. Integration Points

### Primary Entry Point

**Location**: `internal/agent/agent.go:227`

```go
// HandleMessage processes an incoming message (acts as "wingmate" in this conversation).
// This implements protocol.MessageHandler.
func (a *Agent) HandleMessage(ctx context.Context, msg *types.Message) (*types.A2AResponse, error) {
```

This is where Claude API calls would be made. Currently handles only `ping` messages; needs extension to route appropriate messages to Claude.

### Secondary Integration Points

| ID | Location | Purpose | Integration Action |
|----|----------|---------|-------------------|
| IA-01 | `agent.go:227` | Message handler | Add Claude routing logic |
| IA-02 | `agent.go:39-80` | Agent constructor | Inject Claude client |
| IA-03 | `config.go:27-34` | Config struct | Add Claude API settings |
| IA-04 | `client.go:17-36` | HTTP client | Reference pattern for Claude client |
| IA-05 | `errors.go:9-54` | Error codes | Add LLM-specific error codes |

### Recommended New Files

```
internal/
├── agent/
│   └── agent.go          # Modify HandleMessage
├── llm/                   # NEW PACKAGE
│   ├── claude.go          # Claude API client
│   ├── claude_test.go     # Claude client tests
│   ├── types.go           # LLM request/response types
│   └── errors.go          # LLM-specific errors
└── protocol/
    └── errors.go          # Extend with LLM error codes
```

---

## 2. Message Structure Compatibility

### Existing A2A Message Structure

**Location**: `pkg/types/message.go:5-32`

```go
type Message struct {
    Role  string `json:"role"`    // "user", "agent"
    Parts []Part `json:"parts"`   // Content components
}

type Part struct {
    Kind     string                 `json:"kind"`     // "text", "data", "file"
    Text     string                 `json:"text,omitempty"`
    Data     map[string]interface{} `json:"data,omitempty"`
    FilePath string                 `json:"filePath,omitempty"`
    MimeType string                 `json:"mimeType,omitempty"`
}
```

### Claude API Message Structure

```go
// Claude API expects:
type ClaudeMessage struct {
    Role    string          `json:"role"`    // "user", "assistant"
    Content []ContentBlock  `json:"content"`
}

type ContentBlock struct {
    Type string `json:"type"`  // "text", "image", etc.
    Text string `json:"text,omitempty"`
}
```

### Mapping Strategy

The structures are highly compatible:

| A2A Field | Claude Field | Mapping Notes |
|-----------|--------------|---------------|
| `Message.Role` | `ClaudeMessage.Role` | "agent" → "assistant" |
| `Message.Parts` | `ClaudeMessage.Content` | 1:1 mapping |
| `Part.Kind="text"` | `ContentBlock.Type="text"` | Direct mapping |
| `Part.Text` | `ContentBlock.Text` | Direct mapping |
| `Part.Kind="file"` | `ContentBlock.Type="image"` | Base64 encoding needed |

**Recommendation**: Create `internal/llm/convert.go` with bidirectional converters:
- `A2AToClaudeMessage(*types.Message) *ClaudeMessage`
- `ClaudeToA2AMessage(*ClaudeMessage) *types.Message`

---

## 3. Configuration Patterns

### Existing Configuration System

**Location**: `internal/agent/config.go`

The config system already supports:
- JSON file loading (`LoadConfig`)
- Environment variable overrides (`WithEnv`)
- CLI flag merging (`Merge`)
- Default values (`WithDefaults`)

### Existing Environment Variables

```go
const (
    EnvName    = "WINGMATE_NAME"
    EnvPort    = "WINGMATE_PORT"
    EnvPeers   = "WINGMATE_PEERS"
    EnvLog     = "WINGMATE_LOG"
    EnvVerbose = "WINGMATE_VERBOSE"
)
```

### Recommended Claude API Configuration

**Add to `config.go`**:

```go
// New environment variables for Claude API
const (
    EnvClaudeAPIKey    = "WINGMATE_CLAUDE_API_KEY"    // Required for Claude
    EnvClaudeModel     = "WINGMATE_CLAUDE_MODEL"      // Default: claude-3-5-sonnet-20241022
    EnvClaudeMaxTokens = "WINGMATE_CLAUDE_MAX_TOKENS" // Default: 4096
    EnvClaudeBaseURL   = "WINGMATE_CLAUDE_BASE_URL"   // Default: https://api.anthropic.com
)

// Add to Config struct
type Config struct {
    // ... existing fields ...

    // Claude API settings
    Claude *ClaudeConfig `json:"claude,omitempty"`
}

type ClaudeConfig struct {
    APIKey       string `json:"-"`                    // Never serialize API key
    Model        string `json:"model,omitempty"`      // Default: claude-3-5-sonnet-20241022
    MaxTokens    int    `json:"maxTokens,omitempty"`  // Default: 4096
    BaseURL      string `json:"baseURL,omitempty"`    // For enterprise/proxy
    SystemPrompt string `json:"systemPrompt,omitempty"`
}
```

**Security Note**: API key MUST be loaded from environment only, never from config file. The `json:"-"` tag prevents accidental serialization.

---

## 4. HTTP Client Infrastructure

### Existing HTTP Client Pattern

**Location**: `internal/protocol/client.go:17-43`

```go
type A2AClient struct {
    httpClient *http.Client
    requestID  atomic.Int64
}

func NewA2AClient() *A2AClient {
    return &A2AClient{
        httpClient: &http.Client{
            Transport: &http.Transport{
                MaxIdleConns:        10,
                MaxIdleConnsPerHost: 5,
                IdleConnTimeout:     90 * time.Second,
            },
            Timeout: 30 * time.Second,
        },
    }
}
```

### Recommended Claude Client Pattern

```go
// internal/llm/claude.go

type ClaudeClient struct {
    httpClient *http.Client
    apiKey     string
    model      string
    maxTokens  int
    baseURL    string
}

func NewClaudeClient(cfg *ClaudeConfig) (*ClaudeClient, error) {
    if cfg.APIKey == "" {
        return nil, errors.New("claude API key required")
    }

    return &ClaudeClient{
        httpClient: &http.Client{
            Transport: &http.Transport{
                MaxIdleConns:        5,
                MaxIdleConnsPerHost: 5,
                IdleConnTimeout:     90 * time.Second,
            },
            Timeout: 120 * time.Second, // Longer for LLM responses
        },
        apiKey:    cfg.APIKey,
        model:     cfg.Model,
        maxTokens: cfg.MaxTokens,
        baseURL:   cfg.BaseURL,
    }, nil
}
```

### Claude API Request Pattern

```go
func (c *ClaudeClient) Complete(ctx context.Context, messages []ClaudeMessage, system string) (*ClaudeResponse, error) {
    reqBody := ClaudeRequest{
        Model:     c.model,
        MaxTokens: c.maxTokens,
        Messages:  messages,
        System:    system,
    }

    body, _ := json.Marshal(reqBody)

    req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("x-api-key", c.apiKey)
    req.Header.Set("anthropic-version", "2023-06-01")

    resp, err := c.httpClient.Do(req)
    // ... handle response
}
```

---

## 5. Error Handling

### Existing Error Code Ranges

**Location**: `internal/protocol/errors.go:9-26`

```go
// Standard JSON-RPC 2.0 error codes (-32700 to -32600)
const (
    CodeParseError     = -32700
    CodeInvalidRequest = -32600
    CodeMethodNotFound = -32601
    CodeInvalidParams  = -32602
    CodeInternalError  = -32603
)
```

### Recommended LLM Error Codes

Add to `internal/protocol/errors.go`:

```go
// LLM-specific error codes (application range: 1000+)
const (
    // CodeLLMUnavailable indicates the LLM service is not available.
    CodeLLMUnavailable = 1001

    // CodeLLMRateLimit indicates rate limiting from the LLM provider.
    CodeLLMRateLimit = 1002

    // CodeLLMTimeout indicates the LLM request timed out.
    CodeLLMTimeout = 1003

    // CodeLLMContextTooLong indicates the context exceeds model limits.
    CodeLLMContextTooLong = 1004

    // CodeLLMContentFiltered indicates content was filtered by safety.
    CodeLLMContentFiltered = 1005

    // CodeLLMInvalidAPIKey indicates the API key is invalid.
    CodeLLMInvalidAPIKey = 1006
)
```

### Claude API Error Mapping

| Claude HTTP Status | Claude Error Type | Wingmate Error Code |
|-------------------|-------------------|---------------------|
| 401 | authentication_error | CodeLLMInvalidAPIKey |
| 429 | rate_limit_error | CodeLLMRateLimit |
| 408, 504 | timeout | CodeLLMTimeout |
| 400 (context) | invalid_request_error | CodeLLMContextTooLong |
| 500+ | api_error | CodeLLMUnavailable |

---

## 6. Testing Patterns

### Existing Test Infrastructure

The codebase uses:
- `testing` package with table-driven tests
- `httptest` for HTTP mocking
- Test isolation with temp directories
- Mock interfaces for dependencies

**Example from** `internal/protocol/server_test.go`:

```go
func TestA2AServer_HandleAgentCard(t *testing.T) {
    server := NewA2AServer()
    // ... setup ...

    ts := httptest.NewServer(handler)
    defer ts.Close()

    resp, err := http.Get(ts.URL + "/.well-known/agent.json")
    // ... assertions ...
}
```

### Recommended Claude Client Testing

```go
// internal/llm/claude_test.go

func TestClaudeClient_Complete(t *testing.T) {
    tests := []struct {
        name       string
        messages   []ClaudeMessage
        serverResp string
        wantErr    bool
    }{
        {
            name: "successful completion",
            messages: []ClaudeMessage{
                {Role: "user", Content: []ContentBlock{{Type: "text", Text: "Hello"}}},
            },
            serverResp: `{"content":[{"type":"text","text":"Hi there!"}]}`,
            wantErr:    false,
        },
        {
            name:       "rate limit error",
            serverResp: `{"error":{"type":"rate_limit_error"}}`,
            wantErr:    true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                // Validate headers
                if r.Header.Get("x-api-key") == "" {
                    t.Error("missing API key header")
                }
                w.Write([]byte(tt.serverResp))
            }))
            defer ts.Close()

            client := &ClaudeClient{
                httpClient: ts.Client(),
                apiKey:     "test-key",
                baseURL:    ts.URL,
            }

            _, err := client.Complete(context.Background(), tt.messages, "")
            if (err != nil) != tt.wantErr {
                t.Errorf("Complete() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Interface for Mocking

```go
// internal/llm/interface.go

type LLMClient interface {
    Complete(ctx context.Context, messages []Message, system string) (*Response, error)
    Close() error
}

// Mock implementation for testing
type MockLLMClient struct {
    CompleteFunc func(ctx context.Context, messages []Message, system string) (*Response, error)
}

func (m *MockLLMClient) Complete(ctx context.Context, messages []Message, system string) (*Response, error) {
    return m.CompleteFunc(ctx, messages, system)
}
```

---

## 7. Observability

### Existing Flight Log Integration

**Location**: `internal/flightlog/`

The Flight Log already captures:
- Message direction (inbound/outbound)
- Conversation role (pilot/wingmate)
- Trace IDs via context

**Example from** `agent.go:237-245`:

```go
entry := flightlog.NewEntryWithRole(
    ctx,
    a.config.Name,
    flightlog.RoleWingmate,
    flightlog.Inbound,
    "Received message",
    msg,
)
a.flightLog.Record(entry)
```

### Recommended LLM Observability

Add LLM-specific logging:

```go
// Log LLM request
entry := flightlog.NewEntry(
    ctx,
    a.config.Name,
    flightlog.Outbound,
    "LLM request",
    map[string]interface{}{
        "provider":   "claude",
        "model":      c.model,
        "tokens_in":  estimatedInputTokens,
        "message_id": msg.ID,
    },
)
a.flightLog.Record(entry)

// Log LLM response
entry := flightlog.NewEntry(
    ctx,
    a.config.Name,
    flightlog.Inbound,
    "LLM response",
    map[string]interface{}{
        "provider":    "claude",
        "model":       resp.Model,
        "tokens_in":   resp.Usage.InputTokens,
        "tokens_out":  resp.Usage.OutputTokens,
        "stop_reason": resp.StopReason,
        "latency_ms":  latency.Milliseconds(),
    },
)
```

### Metrics to Track

| Metric | Type | Purpose |
|--------|------|---------|
| `llm_requests_total` | Counter | Total LLM API calls |
| `llm_tokens_input_total` | Counter | Input tokens consumed |
| `llm_tokens_output_total` | Counter | Output tokens generated |
| `llm_latency_seconds` | Histogram | API response times |
| `llm_errors_total` | Counter | Errors by type |
| `llm_rate_limits_total` | Counter | Rate limit events |

---

## 8. External Research Gaps

These items require web research or documentation review:

### Claude API Specifics

| Gap | Priority | Notes |
|-----|----------|-------|
| Go SDK availability | High | Check for official `anthropics-sdk-go` |
| Streaming response handling | High | SSE format, chunked responses |
| Rate limit headers | High | `retry-after`, `x-ratelimit-*` headers |
| Token counting | Medium | Client-side estimation vs API response |
| Vision/image support | Medium | Base64 encoding requirements |
| Tool use / function calling | Medium | Format for Claude tool definitions |

### Implementation Decisions Needed

1. **Streaming vs Blocking**
   - Streaming: Lower latency, complex handling
   - Blocking: Simpler, higher latency
   - Recommendation: Start with blocking, add streaming later

2. **Context Management**
   - How much conversation history to send?
   - Token budget allocation?
   - Sliding window vs summarization?

3. **Retry Strategy**
   - Exponential backoff for rate limits
   - Immediate retry for transient errors
   - Circuit breaker for sustained failures

4. **API Key Management**
   - Environment variable only (current recommendation)
   - Future: Secret manager integration

---

## 9. Architecture Recommendations

### Proposed Package Structure

```
internal/
├── llm/
│   ├── client.go          # LLMClient interface
│   ├── claude.go          # Claude implementation
│   ├── claude_test.go
│   ├── convert.go         # A2A <-> Claude message conversion
│   ├── convert_test.go
│   ├── errors.go          # LLM-specific errors
│   ├── types.go           # Request/Response types
│   └── mock.go            # Test mocks
├── agent/
│   ├── agent.go           # Modified HandleMessage
│   ├── config.go          # Extended with ClaudeConfig
│   └── llm_handler.go     # New: LLM routing logic
```

### Dependency Injection Pattern

```go
// Agent constructor modification
func New(cfg *Config, opts ...Option) (*Agent, error) {
    agent := &Agent{config: cfg}

    for _, opt := range opts {
        opt(agent)
    }

    // Default LLM client if not injected
    if agent.llmClient == nil && cfg.Claude != nil {
        client, err := llm.NewClaudeClient(cfg.Claude)
        if err != nil {
            return nil, err
        }
        agent.llmClient = client
    }

    return agent, nil
}

// Option for testing
func WithLLMClient(c llm.Client) Option {
    return func(a *Agent) {
        a.llmClient = c
    }
}
```

### Message Routing Logic

```go
// In HandleMessage, add routing:
func (a *Agent) HandleMessage(ctx context.Context, msg *types.Message) (*types.A2AResponse, error) {
    // Existing ping handling
    if isPing(msg) {
        return a.handlePing(ctx, msg)
    }

    // Route to LLM if client configured and message is suitable
    if a.llmClient != nil && isLLMSuitable(msg) {
        return a.handleWithLLM(ctx, msg)
    }

    // Fallback
    return nil, &protocol.ErrorObject{
        Code:    protocol.CodeMethodNotFound,
        Message: "unknown message type",
    }
}

func isLLMSuitable(msg *types.Message) bool {
    // Check for text content that would benefit from LLM
    for _, part := range msg.Parts {
        if part.Kind == "text" && len(part.Text) > 0 {
            return true
        }
    }
    return false
}
```

---

## 10. Risk Assessment

### Low Risk

| Item | Mitigation |
|------|------------|
| Message format mismatch | Well-defined converter with tests |
| Config complexity | Follow existing patterns exactly |
| Test coverage | Existing test patterns are robust |

### Medium Risk

| Item | Mitigation |
|------|------------|
| API key security | Environment-only loading, never serialize |
| Rate limiting | Implement exponential backoff |
| Token cost | Add budget controls, logging |
| Latency impact | Async option, timeout configuration |

### High Risk (Requires Careful Planning)

| Item | Mitigation |
|------|------------|
| Context overflow | Implement token counting, window management |
| Streaming complexity | Start without streaming, add incrementally |
| Provider lock-in | Abstract behind interface from day one |

---

## Appendix A: Code References

| File | Lines | Purpose |
|------|-------|---------|
| `internal/agent/agent.go` | 227-273 | HandleMessage - main integration point |
| `internal/agent/config.go` | 27-34 | Config struct to extend |
| `internal/agent/config.go` | 60-92 | WithEnv pattern to follow |
| `internal/protocol/client.go` | 17-36 | HTTP client pattern |
| `internal/protocol/errors.go` | 9-54 | Error code definitions |
| `pkg/types/message.go` | 5-32 | Message/Part structures |
| `internal/flightlog/entry.go` | (all) | Logging patterns |

---

## Appendix B: Prior Learnings from Codebase

1. **ADR-003 Unified Peers**: No separate modes - agent can be both pilot and wingmate
2. **Flight Log**: All conversations MUST be logged (Constitution P1)
3. **No time estimates**: Use complexity scores only (Rules)
4. **TDD Required**: Tests first for non-trivial changes
5. **No secrets in logs**: Never log API keys, credentials

---

## Next Steps

This research phase is complete. The dossier provides comprehensive context for:
1. Specification phase (`/plan-1b-specify`)
2. Architecture planning (`/plan-3-architect`)
3. Implementation (`/plan-6-implement-phase`)

**Research Status**: COMPLETE
**Recommendation**: Proceed to `/plan-1b-specify` when ready to define the feature specification.
