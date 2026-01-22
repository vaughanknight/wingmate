// Package helpers provides test utilities for Wingmate tests.
package helpers

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"
	"testing"
)

// TestTransport provides a simulated stdio transport for testing MCP servers.
// It uses io.Pipe internally to create bidirectional communication channels.
type TestTransport struct {
	// ClientReader is where the test reads server responses from.
	ClientReader *io.PipeReader
	// ClientWriter is where the test writes requests to the server.
	ClientWriter *io.PipeWriter

	// ServerReader is what the server reads client requests from.
	ServerReader *io.PipeReader
	// ServerWriter is what the server writes responses to.
	ServerWriter *io.PipeWriter

	mu     sync.Mutex
	closed bool
}

// NewTestTransport creates a new TestTransport with connected pipes.
// The transport simulates stdio communication:
//   - Write to ClientWriter -> Server sees it on ServerReader
//   - Server writes to ServerWriter -> Read from ClientReader
func NewTestTransport() *TestTransport {
	// Client writes -> Server reads
	serverReader, clientWriter := io.Pipe()
	// Server writes -> Client reads
	clientReader, serverWriter := io.Pipe()

	return &TestTransport{
		ClientReader: clientReader,
		ClientWriter: clientWriter,
		ServerReader: serverReader,
		ServerWriter: serverWriter,
	}
}

// Close closes all pipes in the transport.
func (t *TestTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true

	// Close all pipes - order matters to avoid deadlocks
	t.ClientWriter.Close()
	t.ServerWriter.Close()
	t.ClientReader.Close()
	t.ServerReader.Close()

	return nil
}

// SendJSON writes a JSON-encoded value with a newline to the server.
// This simulates a client sending an NDJSON message.
func (t *TestTransport) SendJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = t.ClientWriter.Write(data)
	return err
}

// ReadJSON reads a JSON-encoded value from the server response.
// This simulates a client reading an NDJSON message.
func (t *TestTransport) ReadJSON(v any) error {
	scanner := bufio.NewScanner(t.ClientReader)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return err
		}
		return io.EOF
	}
	return json.Unmarshal(scanner.Bytes(), v)
}

// JSONRPCRequest represents a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id,omitempty"`
	Result  any            `json:"result,omitempty"`
	Error   *JSONRPCError  `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// SendRequest sends a JSON-RPC request to the server.
func (t *TestTransport) SendRequest(id any, method string, params any) error {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	return t.SendJSON(req)
}

// ReadResponse reads a JSON-RPC response from the server.
func (t *TestTransport) ReadResponse() (*JSONRPCResponse, error) {
	var resp JSONRPCResponse
	if err := t.ReadJSON(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// AssertNoError is a test helper that fails if err is not nil.
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// AssertEqual is a test helper that fails if got != want.
func AssertEqual[T comparable](t *testing.T, got, want T, msg string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", msg, got, want)
	}
}
