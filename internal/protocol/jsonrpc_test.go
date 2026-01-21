package protocol

import (
	"encoding/json"
	"testing"
)

// TestRequest_Marshal verifies JSON-RPC 2.0 request serialization.
func TestRequest_Marshal(t *testing.T) {
	tests := []struct {
		name     string
		request  Request
		wantJSON string
	}{
		{
			name: "basic request",
			request: Request{
				JSONRPC: "2.0",
				ID:      json.RawMessage(`1`),
				Method:  "message/send",
				Params:  json.RawMessage(`{"content":"hello"}`),
			},
			wantJSON: `{"jsonrpc":"2.0","id":1,"method":"message/send","params":{"content":"hello"}}`,
		},
		{
			name: "string id",
			request: Request{
				JSONRPC: "2.0",
				ID:      json.RawMessage(`"req-123"`),
				Method:  "ping",
				Params:  nil,
			},
			wantJSON: `{"jsonrpc":"2.0","id":"req-123","method":"ping"}`,
		},
		{
			name: "notification (no id)",
			request: Request{
				JSONRPC: "2.0",
				Method:  "notify",
				Params:  json.RawMessage(`{}`),
			},
			wantJSON: `{"jsonrpc":"2.0","method":"notify","params":{}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			// Parse both to compare as JSON (ignore whitespace differences)
			var gotMap, wantMap map[string]interface{}
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Fatalf("Failed to parse got: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.wantJSON), &wantMap); err != nil {
				t.Fatalf("Failed to parse want: %v", err)
			}

			// Check required fields
			if gotMap["jsonrpc"] != wantMap["jsonrpc"] {
				t.Errorf("jsonrpc = %v, want %v", gotMap["jsonrpc"], wantMap["jsonrpc"])
			}
			if gotMap["method"] != wantMap["method"] {
				t.Errorf("method = %v, want %v", gotMap["method"], wantMap["method"])
			}
		})
	}
}

// TestRequest_Unmarshal verifies JSON-RPC 2.0 request deserialization.
func TestRequest_Unmarshal(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantMethod string
		wantID     string
		wantErr    bool
	}{
		{
			name:       "valid request",
			input:      `{"jsonrpc":"2.0","id":1,"method":"message/send","params":{}}`,
			wantMethod: "message/send",
			wantID:     "1",
			wantErr:    false,
		},
		{
			name:       "string id",
			input:      `{"jsonrpc":"2.0","id":"abc","method":"test"}`,
			wantMethod: "test",
			wantID:     `"abc"`,
			wantErr:    false,
		},
		{
			name:       "notification (no id)",
			input:      `{"jsonrpc":"2.0","method":"notify"}`,
			wantMethod: "notify",
			wantID:     "",
			wantErr:    false,
		},
		{
			name:    "invalid json",
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req Request
			err := json.Unmarshal([]byte(tt.input), &req)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if req.Method != tt.wantMethod {
				t.Errorf("Method = %q, want %q", req.Method, tt.wantMethod)
			}

			gotID := string(req.ID)
			if gotID != tt.wantID {
				t.Errorf("ID = %q, want %q", gotID, tt.wantID)
			}
		})
	}
}

// TestResponse_Marshal verifies JSON-RPC 2.0 response serialization.
func TestResponse_Marshal(t *testing.T) {
	tests := []struct {
		name     string
		response Response
		wantJSON string
	}{
		{
			name: "success response",
			response: Response{
				JSONRPC: "2.0",
				ID:      json.RawMessage(`1`),
				Result:  json.RawMessage(`{"status":"ok"}`),
			},
			wantJSON: `{"jsonrpc":"2.0","id":1,"result":{"status":"ok"}}`,
		},
		{
			name: "error response",
			response: Response{
				JSONRPC: "2.0",
				ID:      json.RawMessage(`1`),
				Error: &ErrorObject{
					Code:    -32600,
					Message: "Invalid Request",
				},
			},
			wantJSON: `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid Request"}}`,
		},
		{
			name: "error with data",
			response: Response{
				JSONRPC: "2.0",
				ID:      json.RawMessage(`"req-1"`),
				Error: &ErrorObject{
					Code:    -32602,
					Message: "Invalid params",
					Data:    json.RawMessage(`{"field":"name"}`),
				},
			},
			wantJSON: `{"jsonrpc":"2.0","id":"req-1","error":{"code":-32602,"message":"Invalid params","data":{"field":"name"}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.response)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			// Verify it's valid JSON
			var gotMap map[string]interface{}
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Fatalf("Result is not valid JSON: %v", err)
			}

			// Check jsonrpc version
			if gotMap["jsonrpc"] != "2.0" {
				t.Errorf("jsonrpc = %v, want 2.0", gotMap["jsonrpc"])
			}
		})
	}
}

// TestResponse_Unmarshal verifies JSON-RPC 2.0 response deserialization.
func TestResponse_Unmarshal(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		hasResult bool
		hasError  bool
		errorCode int
	}{
		{
			name:      "success response",
			input:     `{"jsonrpc":"2.0","id":1,"result":{"data":"value"}}`,
			hasResult: true,
			hasError:  false,
		},
		{
			name:      "error response",
			input:     `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`,
			hasResult: false,
			hasError:  true,
			errorCode: -32601,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp Response
			if err := json.Unmarshal([]byte(tt.input), &resp); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if resp.JSONRPC != "2.0" {
				t.Errorf("JSONRPC = %q, want 2.0", resp.JSONRPC)
			}

			if tt.hasResult && resp.Result == nil {
				t.Error("Expected result, got nil")
			}

			if tt.hasError {
				if resp.Error == nil {
					t.Error("Expected error, got nil")
				} else if resp.Error.Code != tt.errorCode {
					t.Errorf("Error code = %d, want %d", resp.Error.Code, tt.errorCode)
				}
			}
		})
	}
}

// TestErrorObject_Marshal verifies error object serialization.
func TestErrorObject_Marshal(t *testing.T) {
	tests := []struct {
		name  string
		err   ErrorObject
		check func(t *testing.T, got map[string]interface{})
	}{
		{
			name: "basic error",
			err: ErrorObject{
				Code:    -32700,
				Message: "Parse error",
			},
			check: func(t *testing.T, got map[string]interface{}) {
				if got["code"].(float64) != -32700 {
					t.Errorf("code = %v, want -32700", got["code"])
				}
				if got["message"] != "Parse error" {
					t.Errorf("message = %v, want Parse error", got["message"])
				}
				if _, ok := got["data"]; ok {
					t.Error("data should be omitted when nil")
				}
			},
		},
		{
			name: "error with data",
			err: ErrorObject{
				Code:    -32602,
				Message: "Invalid params",
				Data:    json.RawMessage(`{"details":"missing field"}`),
			},
			check: func(t *testing.T, got map[string]interface{}) {
				if got["data"] == nil {
					t.Error("data should not be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.err)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var got map[string]interface{}
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal result failed: %v", err)
			}

			tt.check(t, got)
		})
	}
}

// TestNewRequest verifies the request constructor.
func TestNewRequest(t *testing.T) {
	params := map[string]string{"key": "value"}
	req, err := NewRequest(42, "test/method", params)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}

	if req.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", req.JSONRPC)
	}
	if req.Method != "test/method" {
		t.Errorf("Method = %q, want test/method", req.Method)
	}
	if string(req.ID) != "42" {
		t.Errorf("ID = %q, want 42", string(req.ID))
	}
}

// TestNewSuccessResponse verifies the success response constructor.
func TestNewSuccessResponse(t *testing.T) {
	result := map[string]string{"status": "ok"}
	resp, err := NewSuccessResponse(json.RawMessage(`1`), result)
	if err != nil {
		t.Fatalf("NewSuccessResponse failed: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", resp.JSONRPC)
	}
	if resp.Error != nil {
		t.Error("Error should be nil for success response")
	}
	if resp.Result == nil {
		t.Error("Result should not be nil")
	}
}

// TestNewErrorResponse verifies the error response constructor.
func TestNewErrorResponse(t *testing.T) {
	resp := NewErrorResponse(json.RawMessage(`1`), -32600, "Invalid Request", nil)

	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", resp.JSONRPC)
	}
	if resp.Result != nil {
		t.Error("Result should be nil for error response")
	}
	if resp.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("Error code = %d, want -32600", resp.Error.Code)
	}
}
