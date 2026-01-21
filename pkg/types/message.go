package types

// Message represents an A2A communication unit.
// Messages flow between Pilot and Wingmate agents.
type Message struct {
	// Role identifies the sender (e.g., "user", "agent").
	Role string `json:"role"`

	// Parts contains the message content.
	// Use empty slice (not nil) to produce [] in JSON.
	Parts []Part `json:"parts"`
}

// Part represents a content component within a Message.
// Uses discriminated union pattern with Kind field.
type Part struct {
	// Kind determines which fields are populated.
	// Valid values: "text", "data", "file"
	Kind string `json:"kind"`

	// Text content (when Kind == "text")
	Text string `json:"text,omitempty"`

	// Data content as key-value pairs (when Kind == "data")
	Data map[string]interface{} `json:"data,omitempty"`

	// FilePath for file references (when Kind == "file")
	FilePath string `json:"filePath,omitempty"`

	// MimeType for file content type (when Kind == "file")
	MimeType string `json:"mimeType,omitempty"`
}

// NewTextPart creates a text Part.
func NewTextPart(text string) Part {
	return Part{
		Kind: "text",
		Text: text,
	}
}

// NewDataPart creates a data Part.
func NewDataPart(data map[string]interface{}) Part {
	return Part{
		Kind: "data",
		Data: data,
	}
}

// NewFilePart creates a file Part.
func NewFilePart(path, mimeType string) Part {
	return Part{
		Kind:     "file",
		FilePath: path,
		MimeType: mimeType,
	}
}

// A2ARequest represents a JSON-RPC 2.0 request for A2A.
type A2ARequest struct {
	// JSONRPC is always "2.0"
	JSONRPC string `json:"jsonrpc"`

	// ID is the request identifier
	ID interface{} `json:"id"`

	// Method is the RPC method name (e.g., "message/send")
	Method string `json:"method"`

	// Params contains method-specific parameters
	Params interface{} `json:"params,omitempty"`
}

// A2AResponse represents a JSON-RPC 2.0 response.
type A2AResponse struct {
	// JSONRPC is always "2.0"
	JSONRPC string `json:"jsonrpc"`

	// ID matches the request ID
	ID interface{} `json:"id"`

	// Result contains success response data (mutually exclusive with Error)
	Result interface{} `json:"result,omitempty"`

	// Error contains error details (mutually exclusive with Result)
	Error *JSONRPCError `json:"error,omitempty"`
}

// MessageSendParams contains parameters for message/send method.
type MessageSendParams struct {
	// Message is the message to send
	Message Message `json:"message"`

	// TaskID is the optional task context
	TaskID string `json:"taskId,omitempty"`
}

// MessageSendResult contains the result of message/send.
type MessageSendResult struct {
	// Message is the response message
	Message *Message `json:"message,omitempty"`

	// TaskID is the task context if applicable
	TaskID string `json:"taskId,omitempty"`

	// Status indicates message delivery status
	Status string `json:"status,omitempty"`
}
