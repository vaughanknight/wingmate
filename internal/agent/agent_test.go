package agent

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// TestAgent_New verifies agent constructor sets up server and client.
func TestAgent_New(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Name:    "test-agent",
		Port:    0, // Auto-assign port
		LogFile: filepath.Join(dir, "flight.jsonl"),
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer agent.Shutdown(context.Background())

	if agent.Name() != "test-agent" {
		t.Errorf("Name = %q, want test-agent", agent.Name())
	}

	if agent.server == nil {
		t.Error("server is nil")
	}

	if agent.client == nil {
		t.Error("client is nil")
	}

	if agent.flightLog == nil {
		t.Error("flightLog is nil")
	}
}

// TestAgent_New_InvalidConfig verifies error for invalid config.
func TestAgent_New_InvalidConfig(t *testing.T) {
	cfg := &Config{
		// Missing required Name
		Port: 9000,
	}

	_, err := New(cfg)
	if err == nil {
		t.Error("Expected error for invalid config")
	}
}

// TestAgent_Start verifies server starts and ready fires.
func TestAgent_Start(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Name:    "test-agent",
		Port:    0,
		LogFile: filepath.Join(dir, "flight.jsonl"),
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- agent.Start(ctx)
	}()

	// Wait for ready
	if err := agent.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("WaitUntilReady failed: %v", err)
	}

	// Verify address is set
	addr := agent.Addr()
	if addr == "" {
		t.Error("Addr() returned empty string after start")
	}

	// Shutdown
	if err := agent.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

// TestAgent_WaitUntilReady_Timeout verifies timeout on WaitUntilReady.
func TestAgent_WaitUntilReady_Timeout(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Name:    "test-agent",
		Port:    0,
		LogFile: filepath.Join(dir, "flight.jsonl"),
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer agent.Shutdown(context.Background())

	// Don't start the agent - WaitUntilReady should timeout
	err = agent.WaitUntilReady(100 * time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error")
	}
}

// TestAgent_Shutdown verifies graceful shutdown.
func TestAgent_Shutdown(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Name:    "test-agent",
		Port:    0,
		LogFile: filepath.Join(dir, "flight.jsonl"),
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agent.Start(ctx)

	if err := agent.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("WaitUntilReady failed: %v", err)
	}

	// Shutdown with context
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := agent.Shutdown(shutdownCtx); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Second shutdown should be safe (idempotent)
	if err := agent.Shutdown(context.Background()); err != nil {
		t.Errorf("Second Shutdown failed: %v", err)
	}
}

// TestAgent_Shutdown_NotStarted verifies shutdown before start is safe.
func TestAgent_Shutdown_NotStarted(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Name:    "test-agent",
		Port:    0,
		LogFile: filepath.Join(dir, "flight.jsonl"),
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Shutdown without starting
	if err := agent.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown before start failed: %v", err)
	}
}

// TestAgent_AgentCard verifies Agent Card is generated correctly.
func TestAgent_AgentCard(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Name:    "card-test-agent",
		Port:    0,
		LogFile: filepath.Join(dir, "flight.jsonl"),
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer agent.Shutdown(context.Background())

	card := agent.AgentCard()

	if card.Name != "card-test-agent" {
		t.Errorf("Card.Name = %q, want card-test-agent", card.Name)
	}

	if card.Version == "" {
		t.Error("Card.Version is empty")
	}

	// Should have ping skill
	found := false
	for _, skill := range card.Skills {
		if skill.Name == "ping" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Agent Card missing 'ping' skill")
	}
}

// TestAgent_URL verifies URL generation.
func TestAgent_URL(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Name:    "test-agent",
		Port:    0,
		LogFile: filepath.Join(dir, "flight.jsonl"),
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agent.Start(ctx)

	if err := agent.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("WaitUntilReady failed: %v", err)
	}
	defer agent.Shutdown(context.Background())

	url := agent.URL()
	if url == "" {
		t.Error("URL() returned empty string")
	}

	// URL should start with http://
	if len(url) < 7 || url[:7] != "http://" {
		t.Errorf("URL = %q, want http:// prefix", url)
	}
}

// TestAgent_IsReady verifies ready state.
func TestAgent_IsReady(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Name:    "test-agent",
		Port:    0,
		LogFile: filepath.Join(dir, "flight.jsonl"),
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Not ready before start
	if agent.IsReady() {
		t.Error("IsReady = true before start")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agent.Start(ctx)
	agent.WaitUntilReady(5 * time.Second)
	defer agent.Shutdown(context.Background())

	// Ready after start
	if !agent.IsReady() {
		t.Error("IsReady = false after start")
	}
}
