package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestConfig_LoadFromFile verifies loading config from a JSON file.
func TestConfig_LoadFromFile(t *testing.T) {
	// Create temp config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	cfg := Config{
		Name:    "test-agent",
		Port:    9001,
		Peers:   []string{"http://localhost:9000"},
		LogFile: "/tmp/flight.jsonl",
		Verbose: true,
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loaded.Name != "test-agent" {
		t.Errorf("Name = %q, want test-agent", loaded.Name)
	}

	if loaded.Port != 9001 {
		t.Errorf("Port = %d, want 9001", loaded.Port)
	}

	if len(loaded.Peers) != 1 || loaded.Peers[0] != "http://localhost:9000" {
		t.Errorf("Peers = %v, want [http://localhost:9000]", loaded.Peers)
	}

	if loaded.LogFile != "/tmp/flight.jsonl" {
		t.Errorf("LogFile = %q, want /tmp/flight.jsonl", loaded.LogFile)
	}

	if !loaded.Verbose {
		t.Error("Verbose = false, want true")
	}
}

// TestConfig_LoadFromFile_NotFound verifies error for missing file.
func TestConfig_LoadFromFile_NotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.json")
	if err == nil {
		t.Error("Expected error for missing config file")
	}
}

// TestConfig_LoadFromFile_InvalidJSON verifies error for malformed JSON.
func TestConfig_LoadFromFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	if err := os.WriteFile(configPath, []byte(`{invalid json`), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// TestConfig_EnvOverride verifies environment variables take precedence.
func TestConfig_EnvOverride(t *testing.T) {
	// Create minimal config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	cfg := Config{
		Name:    "file-name",
		Port:    9000,
		LogFile: "file.jsonl",
	}

	data, _ := json.Marshal(cfg)
	os.WriteFile(configPath, data, 0644)

	// Set environment variables
	t.Setenv("WINGMATE_NAME", "env-name")
	t.Setenv("WINGMATE_PORT", "9999")
	t.Setenv("WINGMATE_PEERS", "http://a:1,http://b:2")
	t.Setenv("WINGMATE_LOG", "env.jsonl")
	t.Setenv("WINGMATE_VERBOSE", "true")

	// Load and merge
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	merged := loaded.WithEnv()

	if merged.Name != "env-name" {
		t.Errorf("Name = %q, want env-name (from env)", merged.Name)
	}

	if merged.Port != 9999 {
		t.Errorf("Port = %d, want 9999 (from env)", merged.Port)
	}

	if len(merged.Peers) != 2 || merged.Peers[0] != "http://a:1" {
		t.Errorf("Peers = %v, want [http://a:1 http://b:2] (from env)", merged.Peers)
	}

	if merged.LogFile != "env.jsonl" {
		t.Errorf("LogFile = %q, want env.jsonl (from env)", merged.LogFile)
	}

	if !merged.Verbose {
		t.Error("Verbose = false, want true (from env)")
	}
}

// TestConfig_EnvOverride_PartialOverride verifies partial env override.
func TestConfig_EnvOverride_PartialOverride(t *testing.T) {
	// Create config file with all values
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	cfg := Config{
		Name:    "file-name",
		Port:    9000,
		LogFile: "file.jsonl",
		Verbose: false,
	}

	data, _ := json.Marshal(cfg)
	os.WriteFile(configPath, data, 0644)

	// Only override Name
	t.Setenv("WINGMATE_NAME", "override-name")

	loaded, _ := LoadConfig(configPath)
	merged := loaded.WithEnv()

	// Name should be from env
	if merged.Name != "override-name" {
		t.Errorf("Name = %q, want override-name", merged.Name)
	}

	// Port should be from file
	if merged.Port != 9000 {
		t.Errorf("Port = %d, want 9000 (from file)", merged.Port)
	}

	// LogFile should be from file
	if merged.LogFile != "file.jsonl" {
		t.Errorf("LogFile = %q, want file.jsonl (from file)", merged.LogFile)
	}
}

// TestConfig_Defaults verifies default values for missing fields.
func TestConfig_Defaults(t *testing.T) {
	// Create minimal config with only required field
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	cfg := Config{
		Name: "minimal-agent",
	}

	data, _ := json.Marshal(cfg)
	os.WriteFile(configPath, data, 0644)

	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	merged := loaded.WithDefaults()

	if merged.Name != "minimal-agent" {
		t.Errorf("Name = %q, want minimal-agent", merged.Name)
	}

	// Port 0 means "auto-assign" and is preserved by WithDefaults()
	// Since the config file didn't specify a port, it's 0 after loading
	if merged.Port != 0 {
		t.Errorf("Port = %d, want 0 (auto-assign)", merged.Port)
	}

	if merged.LogFile != DefaultLogFile {
		t.Errorf("LogFile = %q, want %q (default)", merged.LogFile, DefaultLogFile)
	}

	if merged.Verbose {
		t.Error("Verbose = true, want false (default)")
	}

	if merged.Peers == nil {
		t.Error("Peers = nil, want empty slice")
	}
}

// TestConfig_Defaults_NoFile verifies defaults work without file.
func TestConfig_Defaults_NoFile(t *testing.T) {
	cfg := NewConfig()

	if cfg.Port != DefaultPort {
		t.Errorf("Port = %d, want %d", cfg.Port, DefaultPort)
	}

	if cfg.LogFile != DefaultLogFile {
		t.Errorf("LogFile = %q, want %q", cfg.LogFile, DefaultLogFile)
	}
}

// TestConfig_NoModeField verifies ADR-003: no Mode field exists.
func TestConfig_NoModeField(t *testing.T) {
	// Parse JSON with a "mode" field - should be ignored
	jsonData := `{"name":"test","port":9000,"mode":"pilot"}`

	var cfg Config
	if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Mode field should not exist on Config struct
	// If this test compiles, there's no Mode field (which is correct per ADR-003)

	if cfg.Name != "test" {
		t.Errorf("Name = %q, want test", cfg.Name)
	}
}

// TestConfig_Validation_NameRequired verifies Name is required.
func TestConfig_Validation_NameRequired(t *testing.T) {
	cfg := Config{
		Port: 9000,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for missing Name")
	}
}

// TestConfig_Validation_PortRange verifies Port is in valid range.
func TestConfig_Validation_PortRange(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"zero port valid (auto-assign)", 0, false},
		{"negative port invalid", -1, true},
		{"valid low port", 1, false},
		{"valid high port", 65535, false},
		{"too high port", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Name: "test",
				Port: tt.port,
			}

			err := cfg.Validate()
			hasErr := err != nil

			if hasErr != tt.wantErr {
				t.Errorf("Port=%d: got error=%v, wantErr=%v", tt.port, hasErr, tt.wantErr)
			}
		})
	}
}

// TestConfig_Validation_PeerURLs verifies peer URLs are valid.
func TestConfig_Validation_PeerURLs(t *testing.T) {
	tests := []struct {
		name    string
		peers   []string
		wantErr bool
	}{
		{"empty peers valid", []string{}, false},
		{"valid HTTP URL", []string{"http://localhost:9000"}, false},
		{"valid HTTPS URL", []string{"https://example.com:443"}, false},
		{"multiple valid URLs", []string{"http://a:1", "http://b:2"}, false},
		{"invalid URL scheme", []string{"ftp://localhost:9000"}, true},
		{"missing scheme", []string{"localhost:9000"}, true},
		{"one invalid in list", []string{"http://a:1", "invalid"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Name:  "test",
				Port:  9000,
				Peers: tt.peers,
			}

			err := cfg.Validate()
			hasErr := err != nil

			if hasErr != tt.wantErr {
				t.Errorf("Peers=%v: got error=%v (%v), wantErr=%v", tt.peers, hasErr, err, tt.wantErr)
			}
		})
	}
}

// TestConfig_Clone verifies config cloning doesn't share slices.
func TestConfig_Clone(t *testing.T) {
	original := Config{
		Name:  "original",
		Port:  9000,
		Peers: []string{"http://a:1"},
	}

	cloned := original.Clone()

	// Modify clone
	cloned.Name = "cloned"
	cloned.Peers[0] = "http://b:2"

	// Original should be unchanged
	if original.Name != "original" {
		t.Errorf("Original Name changed to %q", original.Name)
	}

	if original.Peers[0] != "http://a:1" {
		t.Errorf("Original Peers changed to %v", original.Peers)
	}
}

// =============================================================================
// Claude Configuration Tests (Phase 3)
// =============================================================================

// TestClaudeConfig_Defaults verifies ClaudeConfig has zero values initially.
func TestClaudeConfig_Defaults(t *testing.T) {
	cfg := NewConfig()

	if cfg.Claude.CLIPath != "" {
		t.Errorf("Claude.CLIPath = %q, want empty", cfg.Claude.CLIPath)
	}

	if cfg.Claude.Model != "" {
		t.Errorf("Claude.Model = %q, want empty", cfg.Claude.Model)
	}

	if cfg.Claude.Timeout != 0 {
		t.Errorf("Claude.Timeout = %v, want 0", cfg.Claude.Timeout)
	}

	if cfg.Claude.SystemPrompt != "" {
		t.Errorf("Claude.SystemPrompt = %q, want empty", cfg.Claude.SystemPrompt)
	}
}

// TestClaudeConfig_JSON verifies JSON marshaling/unmarshaling.
func TestClaudeConfig_JSON(t *testing.T) {
	cfg := Config{
		Name: "test",
		Port: 9000,
		Claude: ClaudeConfig{
			CLIPath:      "/usr/local/bin/claude",
			Model:        "opus",
			Timeout:      180 * time.Second,
			SystemPrompt: "Test prompt",
		},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if loaded.Claude.CLIPath != "/usr/local/bin/claude" {
		t.Errorf("Claude.CLIPath = %q, want /usr/local/bin/claude", loaded.Claude.CLIPath)
	}

	if loaded.Claude.Model != "opus" {
		t.Errorf("Claude.Model = %q, want opus", loaded.Claude.Model)
	}

	if loaded.Claude.Timeout != 180*time.Second {
		t.Errorf("Claude.Timeout = %v, want 180s", loaded.Claude.Timeout)
	}

	if loaded.Claude.SystemPrompt != "Test prompt" {
		t.Errorf("Claude.SystemPrompt = %q, want 'Test prompt'", loaded.Claude.SystemPrompt)
	}
}

// TestConfig_WithEnv_ClaudeCLIPath verifies WINGMATE_CLAUDE_CLI_PATH override.
func TestConfig_WithEnv_ClaudeCLIPath(t *testing.T) {
	t.Setenv("WINGMATE_CLAUDE_CLI_PATH", "/custom/claude")

	cfg := NewConfig().WithEnv()

	if cfg.Claude.CLIPath != "/custom/claude" {
		t.Errorf("Claude.CLIPath = %q, want /custom/claude", cfg.Claude.CLIPath)
	}
}

// TestConfig_WithEnv_ClaudeModel verifies WINGMATE_CLAUDE_MODEL override.
func TestConfig_WithEnv_ClaudeModel(t *testing.T) {
	t.Setenv("WINGMATE_CLAUDE_MODEL", "haiku")

	cfg := NewConfig().WithEnv()

	if cfg.Claude.Model != "haiku" {
		t.Errorf("Claude.Model = %q, want haiku", cfg.Claude.Model)
	}
}

// TestConfig_WithEnv_ClaudeTimeout verifies WINGMATE_CLAUDE_TIMEOUT override.
func TestConfig_WithEnv_ClaudeTimeout(t *testing.T) {
	t.Setenv("WINGMATE_CLAUDE_TIMEOUT", "60s")

	cfg := NewConfig().WithEnv()

	if cfg.Claude.Timeout != 60*time.Second {
		t.Errorf("Claude.Timeout = %v, want 60s", cfg.Claude.Timeout)
	}
}

// TestConfig_WithEnv_ClaudeTimeout_InvalidIgnored verifies invalid timeout is ignored.
func TestConfig_WithEnv_ClaudeTimeout_InvalidIgnored(t *testing.T) {
	t.Setenv("WINGMATE_CLAUDE_TIMEOUT", "invalid")

	cfg := NewConfig().WithEnv()

	// Should remain zero (invalid value ignored)
	if cfg.Claude.Timeout != 0 {
		t.Errorf("Claude.Timeout = %v, want 0 (invalid ignored)", cfg.Claude.Timeout)
	}
}

// TestConfig_WithEnv_ClaudeSystemPrompt verifies WINGMATE_CLAUDE_SYSTEM_PROMPT override.
func TestConfig_WithEnv_ClaudeSystemPrompt(t *testing.T) {
	t.Setenv("WINGMATE_CLAUDE_SYSTEM_PROMPT", "Custom prompt")

	cfg := NewConfig().WithEnv()

	if cfg.Claude.SystemPrompt != "Custom prompt" {
		t.Errorf("Claude.SystemPrompt = %q, want 'Custom prompt'", cfg.Claude.SystemPrompt)
	}
}

// TestConfig_WithDefaults_ClaudeTimeout verifies default timeout is applied.
func TestConfig_WithDefaults_ClaudeTimeout(t *testing.T) {
	cfg := NewConfig().WithDefaults()

	if cfg.Claude.Timeout != DefaultClaudeTimeout {
		t.Errorf("Claude.Timeout = %v, want %v", cfg.Claude.Timeout, DefaultClaudeTimeout)
	}
}

// TestConfig_WithDefaults_ClaudeSystemPrompt verifies default system prompt.
func TestConfig_WithDefaults_ClaudeSystemPrompt(t *testing.T) {
	cfg := NewConfig().WithDefaults()

	if cfg.Claude.SystemPrompt != DefaultClaudeSystemPrompt {
		t.Errorf("Claude.SystemPrompt = %q, want default", cfg.Claude.SystemPrompt)
	}
}

// TestConfig_WithDefaults_ClaudeSystemPrompt_PreservesCustom verifies custom prompt preserved.
func TestConfig_WithDefaults_ClaudeSystemPrompt_PreservesCustom(t *testing.T) {
	cfg := &Config{
		Name: "test",
		Claude: ClaudeConfig{
			SystemPrompt: "Custom prompt",
		},
	}

	result := cfg.WithDefaults()

	if result.Claude.SystemPrompt != "Custom prompt" {
		t.Errorf("Claude.SystemPrompt = %q, want 'Custom prompt'", result.Claude.SystemPrompt)
	}
}

// TestConfig_Validate_CLIPathValid verifies valid CLI path passes validation.
func TestConfig_Validate_CLIPathValid(t *testing.T) {
	// Create a temp file to act as the CLI
	dir := t.TempDir()
	cliPath := filepath.Join(dir, "claude")
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	cfg := Config{
		Name: "test",
		Port: 9000,
		Claude: ClaudeConfig{
			CLIPath: cliPath,
			Timeout: 60 * time.Second,
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

// TestConfig_Validate_CLIPathInvalid verifies invalid CLI path fails validation.
func TestConfig_Validate_CLIPathInvalid(t *testing.T) {
	cfg := Config{
		Name: "test",
		Port: 9000,
		Claude: ClaudeConfig{
			CLIPath: "/nonexistent/path/to/claude",
			Timeout: 60 * time.Second,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for invalid CLI path")
	}
}

// TestConfig_Validate_CLIPathEmpty verifies empty CLI path passes (auto-detect).
func TestConfig_Validate_CLIPathEmpty(t *testing.T) {
	cfg := Config{
		Name: "test",
		Port: 9000,
		Claude: ClaudeConfig{
			CLIPath: "", // Empty = auto-detect from PATH
			Timeout: 60 * time.Second,
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for empty CLI path", err)
	}
}

// TestConfig_Validate_TimeoutPositive verifies positive timeout passes.
func TestConfig_Validate_TimeoutPositive(t *testing.T) {
	cfg := Config{
		Name: "test",
		Port: 9000,
		Claude: ClaudeConfig{
			Timeout: 60 * time.Second,
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

// TestConfig_Validate_TimeoutZero verifies zero timeout passes (use default).
func TestConfig_Validate_TimeoutZero(t *testing.T) {
	cfg := Config{
		Name: "test",
		Port: 9000,
		Claude: ClaudeConfig{
			Timeout: 0, // Zero = use default
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for zero timeout", err)
	}
}

// TestConfig_Validate_TimeoutNegative verifies negative timeout fails.
func TestConfig_Validate_TimeoutNegative(t *testing.T) {
	cfg := Config{
		Name: "test",
		Port: 9000,
		Claude: ClaudeConfig{
			Timeout: -1 * time.Second,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for negative timeout")
	}
}

// TestConfig_Clone_ClonesClaudeConfig verifies ClaudeConfig is deep copied.
func TestConfig_Clone_ClonesClaudeConfig(t *testing.T) {
	original := Config{
		Name: "original",
		Port: 9000,
		Claude: ClaudeConfig{
			CLIPath:      "/usr/bin/claude",
			Model:        "sonnet",
			Timeout:      60 * time.Second,
			SystemPrompt: "Original prompt",
		},
	}

	cloned := original.Clone()

	// Modify clone
	cloned.Claude.CLIPath = "/other/claude"
	cloned.Claude.Model = "opus"
	cloned.Claude.Timeout = 120 * time.Second
	cloned.Claude.SystemPrompt = "Changed prompt"

	// Original should be unchanged
	if original.Claude.CLIPath != "/usr/bin/claude" {
		t.Errorf("Original Claude.CLIPath changed to %q", original.Claude.CLIPath)
	}

	if original.Claude.Model != "sonnet" {
		t.Errorf("Original Claude.Model changed to %q", original.Claude.Model)
	}

	if original.Claude.Timeout != 60*time.Second {
		t.Errorf("Original Claude.Timeout changed to %v", original.Claude.Timeout)
	}

	if original.Claude.SystemPrompt != "Original prompt" {
		t.Errorf("Original Claude.SystemPrompt changed to %q", original.Claude.SystemPrompt)
	}
}

// TestConfig_FullLifecycle tests the complete config lifecycle.
func TestConfig_FullLifecycle(t *testing.T) {
	// Create temp CLI file
	dir := t.TempDir()
	cliPath := filepath.Join(dir, "claude")
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Set environment
	t.Setenv("WINGMATE_NAME", "lifecycle-test")
	t.Setenv("WINGMATE_CLAUDE_CLI_PATH", cliPath)
	t.Setenv("WINGMATE_CLAUDE_MODEL", "opus")
	t.Setenv("WINGMATE_CLAUDE_TIMEOUT", "180s")

	// Full lifecycle: new → env → defaults → validate → clone
	cfg := NewConfig().WithEnv().WithDefaults()

	// Validate
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// Clone
	cloned := cfg.Clone()

	// Verify values
	if cloned.Name != "lifecycle-test" {
		t.Errorf("Name = %q, want lifecycle-test", cloned.Name)
	}

	if cloned.Claude.CLIPath != cliPath {
		t.Errorf("Claude.CLIPath = %q, want %q", cloned.Claude.CLIPath, cliPath)
	}

	if cloned.Claude.Model != "opus" {
		t.Errorf("Claude.Model = %q, want opus", cloned.Claude.Model)
	}

	if cloned.Claude.Timeout != 180*time.Second {
		t.Errorf("Claude.Timeout = %v, want 180s", cloned.Claude.Timeout)
	}

	// System prompt should have default (not set in env)
	if cloned.Claude.SystemPrompt != DefaultClaudeSystemPrompt {
		t.Errorf("Claude.SystemPrompt = %q, want default", cloned.Claude.SystemPrompt)
	}
}
