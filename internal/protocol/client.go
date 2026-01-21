package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wingmate/wingmate/pkg/types"
)

// A2AClient implements the Client interface for A2A protocol.
// It handles outbound HTTP requests to other A2A agents.
type A2AClient struct {
	httpClient *http.Client
	requestID  atomic.Int64
}

// NewA2AClient creates a new A2A client with connection pooling.
func NewA2AClient() *A2AClient {
	return &A2AClient{
		httpClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
			Timeout: 30 * time.Second, // Default timeout
		},
	}
}

// NewA2AClientWithTimeout creates a new A2A client with a custom timeout.
func NewA2AClientWithTimeout(timeout time.Duration) *A2AClient {
	c := NewA2AClient()
	c.httpClient.Timeout = timeout
	return c
}

// SendMessage sends a message to a remote agent and waits for a response.
func (c *A2AClient) SendMessage(ctx context.Context, url string, msg *types.Message) (*types.A2AResponse, error) {
	// Build JSON-RPC request
	reqID := c.requestID.Add(1)
	params := map[string]interface{}{
		"message": msg,
	}

	req, err := NewRequest(reqID, "message/send", params)
	if err != nil {
		return nil, NewProtocolError("marshal", url, err)
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, NewProtocolError("marshal", url, err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, NewProtocolError("create_request", url, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, NewProtocolError("send", url, err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewProtocolError("read_response", url, err)
	}

	// Parse JSON-RPC response
	var jsonResp Response
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return nil, NewProtocolError("parse_response", url, err)
	}

	// Convert to types.A2AResponse
	a2aResp := &types.A2AResponse{
		JSONRPC: jsonResp.JSONRPC,
		ID:      jsonResp.ID,
		Result:  jsonResp.Result,
	}

	if jsonResp.Error != nil {
		a2aResp.Error = &types.JSONRPCError{
			Code:    jsonResp.Error.Code,
			Message: jsonResp.Error.Message,
			Data:    jsonResp.Error.Data,
		}
	}

	return a2aResp, nil
}

// GetAgentCard fetches and parses the Agent Card from a remote agent.
func (c *A2AClient) GetAgentCard(ctx context.Context, url string) (*types.AgentCard, error) {
	// Build Agent Card URL
	cardURL := strings.TrimSuffix(url, "/") + types.AgentCardWellKnownPath

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cardURL, nil)
	if err != nil {
		return nil, NewProtocolError("create_request", cardURL, err)
	}
	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, NewProtocolError("fetch", cardURL, err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, NewProtocolError("fetch", cardURL,
			fmt.Errorf("unexpected status: %d %s", resp.StatusCode, resp.Status))
	}

	// Read and parse body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewProtocolError("read_response", cardURL, err)
	}

	var card types.AgentCard
	if err := json.Unmarshal(body, &card); err != nil {
		return nil, NewProtocolError("parse_response", cardURL, err)
	}

	return &card, nil
}

// Close releases any resources held by the client.
func (c *A2AClient) Close() error {
	// Close idle connections
	if c.httpClient != nil && c.httpClient.Transport != nil {
		if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
	return nil
}

// Ensure A2AClient implements Client interface.
var _ Client = (*A2AClient)(nil)
