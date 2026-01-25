package agent

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"
)

// Default configuration values.
const (
	DefaultPort    = 9000
	DefaultLogFile = "./flight.jsonl"
)

// Claude CLI default configuration values.
const (
	DefaultClaudeTimeout      = 120 * time.Second
	DefaultClaudeSystemPrompt = "You are a debugging assistant helping engineers identify and resolve issues in distributed systems. Provide clear, actionable advice."
)

// Environment variable names.
const (
	EnvName    = "WINGMATE_NAME"
	EnvPort    = "WINGMATE_PORT"
	EnvPeers   = "WINGMATE_PEERS"
	EnvLog     = "WINGMATE_LOG"
	EnvVerbose = "WINGMATE_VERBOSE"
)

// Claude CLI environment variable names.
const (
	EnvClaudeCLIPath      = "WINGMATE_CLAUDE_CLI_PATH"
	EnvClaudeModel        = "WINGMATE_CLAUDE_MODEL"
	EnvClaudeTimeout      = "WINGMATE_CLAUDE_TIMEOUT"
	EnvClaudeSystemPrompt = "WINGMATE_CLAUDE_SYSTEM_PROMPT"
)

// ClaudeConfig holds Claude CLI-specific configuration.
type ClaudeConfig struct {
	CLIPath      string        `json:"cliPath,omitempty"`      // Path to claude binary (empty = auto-detect)
	Model        string        `json:"model,omitempty"`        // Model to use (sonnet, opus, haiku)
	Timeout      time.Duration `json:"timeout,omitempty"`      // Command timeout (default: 120s)
	SystemPrompt string        `json:"systemPrompt,omitempty"` // System prompt for debugging context
}

// Config holds the agent configuration.
// Per ADR-003: There is NO Mode field - every agent is a full peer.
type Config struct {
	Name         string       `json:"name"`                   // Agent name (required)
	Port         int          `json:"port"`                   // Listen port (default: 9000)
	Peers        []string     `json:"peers,omitempty"`        // Known peer URLs
	LogFile      string       `json:"logFile"`                // Flight log path (default: ./flight.jsonl)
	Verbose      bool         `json:"verbose"`                // Mirror logs to stdout
	Capabilities []string     `json:"capabilities,omitempty"` // Advertised capabilities
	Claude       ClaudeConfig `json:"claude,omitempty"`       // Claude CLI configuration
}

// NewConfig creates a new Config with default values.
func NewConfig() *Config {
	return &Config{
		Port:    DefaultPort,
		LogFile: DefaultLogFile,
		Peers:   []string{},
	}
}

// LoadConfig loads configuration from a JSON file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// WithEnv returns a new Config with environment variable overrides applied.
// Precedence: environment variable > file value.
func (c *Config) WithEnv() *Config {
	result := c.Clone()

	if name := os.Getenv(EnvName); name != "" {
		result.Name = name
	}

	if portStr := os.Getenv(EnvPort); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			result.Port = port
		}
	}

	if peersStr := os.Getenv(EnvPeers); peersStr != "" {
		result.Peers = strings.Split(peersStr, ",")
		// Trim whitespace from each peer
		for i, p := range result.Peers {
			result.Peers[i] = strings.TrimSpace(p)
		}
	}

	if logFile := os.Getenv(EnvLog); logFile != "" {
		result.LogFile = logFile
	}

	if verboseStr := os.Getenv(EnvVerbose); verboseStr != "" {
		result.Verbose = verboseStr == "true" || verboseStr == "1"
	}

	// Claude CLI environment variables
	if cliPath := os.Getenv(EnvClaudeCLIPath); cliPath != "" {
		result.Claude.CLIPath = cliPath
	}

	if model := os.Getenv(EnvClaudeModel); model != "" {
		result.Claude.Model = model
	}

	if timeoutStr := os.Getenv(EnvClaudeTimeout); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			result.Claude.Timeout = timeout
		}
	}

	if systemPrompt := os.Getenv(EnvClaudeSystemPrompt); systemPrompt != "" {
		result.Claude.SystemPrompt = systemPrompt
	}

	return result
}

// WithDefaults returns a new Config with default values for missing fields.
// Note: Port == 0 is NOT overridden because 0 means "auto-assign" (OS picks available port).
// Use Port: -1 if you want to explicitly signal "use default port".
func (c *Config) WithDefaults() *Config {
	result := c.Clone()

	if result.Port < 0 {
		result.Port = DefaultPort
	}

	if result.LogFile == "" {
		result.LogFile = DefaultLogFile
	}

	if result.Peers == nil {
		result.Peers = []string{}
	}

	if result.Capabilities == nil {
		result.Capabilities = []string{}
	}

	// Claude CLI defaults
	if result.Claude.Timeout == 0 {
		result.Claude.Timeout = DefaultClaudeTimeout
	}

	if result.Claude.SystemPrompt == "" {
		result.Claude.SystemPrompt = DefaultClaudeSystemPrompt
	}

	return result
}

// Clone creates a deep copy of the Config.
func (c *Config) Clone() *Config {
	result := &Config{
		Name:    c.Name,
		Port:    c.Port,
		LogFile: c.LogFile,
		Verbose: c.Verbose,
		Claude: ClaudeConfig{
			CLIPath:      c.Claude.CLIPath,
			Model:        c.Claude.Model,
			Timeout:      c.Claude.Timeout,
			SystemPrompt: c.Claude.SystemPrompt,
		},
	}

	// Deep copy slices
	if c.Peers != nil {
		result.Peers = make([]string, len(c.Peers))
		copy(result.Peers, c.Peers)
	}

	if c.Capabilities != nil {
		result.Capabilities = make([]string, len(c.Capabilities))
		copy(result.Capabilities, c.Capabilities)
	}

	return result
}

// Merge applies CLI flags on top of the current config.
// Returns a new Config without modifying the receiver.
func (c *Config) Merge(flags *Config) *Config {
	result := c.Clone()

	if flags.Name != "" {
		result.Name = flags.Name
	}

	if flags.Port != 0 {
		result.Port = flags.Port
	}

	if len(flags.Peers) > 0 {
		result.Peers = flags.Peers
	}

	if flags.LogFile != "" {
		result.LogFile = flags.LogFile
	}

	if flags.Verbose {
		result.Verbose = true
	}

	return result
}
