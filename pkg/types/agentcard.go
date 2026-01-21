// Package types provides A2A protocol type definitions for Wingmate.
//
// These types define the wire format for agent-to-agent communication
// per the A2A specification. All types serialize to/from JSON.
package types

// AgentCardWellKnownPath is the standard path where AgentCard must be served.
// Per A2A spec, this is /.well-known/agent.json (NOT agent-card.json).
const AgentCardWellKnownPath = "/.well-known/agent.json"

// AgentCard represents an A2A agent's capability advertisement.
// Served at AgentCardWellKnownPath for agent discovery.
type AgentCard struct {
	// Name is the agent's display name. Required.
	Name string `json:"name"`

	// Description is a human-readable description of the agent.
	Description string `json:"description,omitempty"`

	// URL is the base URL where this agent can be reached. Required.
	URL string `json:"url"`

	// Version is the agent's version string. Required.
	Version string `json:"version"`

	// Capabilities describes what the agent supports.
	Capabilities Capabilities `json:"capabilities"`

	// Skills lists the specific capabilities this agent offers.
	// May be empty but must be present in JSON output.
	Skills []Skill `json:"skills"`

	// Authentication describes supported auth methods.
	// Omitted for MVP (per DEV-002 deviation).
	Authentication *Authentication `json:"authentication,omitempty"`
}

// Capabilities describes agent protocol features.
type Capabilities struct {
	// Streaming indicates support for SSE streaming responses.
	Streaming bool `json:"streaming"`

	// PushNotifications indicates support for push notifications.
	PushNotifications bool `json:"pushNotifications"`
}

// Skill describes a specific capability an agent offers.
type Skill struct {
	// Name is the skill identifier (e.g., "ping").
	Name string `json:"name"`

	// Description explains what this skill does.
	Description string `json:"description,omitempty"`

	// InputSchema is a JSON Schema for skill input (optional).
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`

	// OutputSchema is a JSON Schema for skill output (optional).
	OutputSchema map[string]interface{} `json:"outputSchema,omitempty"`
}

// Authentication describes supported authentication methods.
// Deferred for MVP per DEV-002.
type Authentication struct {
	// Schemes lists supported auth schemes (e.g., "api_key", "oauth2").
	Schemes []string `json:"schemes"`
}
