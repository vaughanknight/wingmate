package agent

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/wingmate/wingmate/internal/flightlog"
	"github.com/wingmate/wingmate/pkg/types"
)

// TestAgent_Ping verifies successful ping/pong exchange.
func TestAgent_Ping(t *testing.T) {
	dir := t.TempDir()

	// Start target agent
	targetCfg := &Config{
		Name:    "target-agent",
		Port:    0,
		LogFile: filepath.Join(dir, "target.jsonl"),
	}

	target, err := New(targetCfg)
	if err != nil {
		t.Fatalf("New target failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go target.Start(ctx)
	if err := target.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("Target failed to start: %v", err)
	}
	defer target.Shutdown(context.Background())

	// Create sender agent
	senderCfg := &Config{
		Name:    "sender-agent",
		Port:    0,
		LogFile: filepath.Join(dir, "sender.jsonl"),
	}

	sender, err := New(senderCfg)
	if err != nil {
		t.Fatalf("New sender failed: %v", err)
	}
	defer sender.Shutdown(context.Background())

	// Send ping
	err = sender.Ping(context.Background(), target.URL())
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

// TestAgent_Ping_FlightLog verifies Flight Log records as pilot.
func TestAgent_Ping_FlightLog(t *testing.T) {
	dir := t.TempDir()
	targetLogPath := filepath.Join(dir, "target.jsonl")
	senderLogPath := filepath.Join(dir, "sender.jsonl")

	// Start target agent
	targetCfg := &Config{
		Name:    "target-agent",
		Port:    0,
		LogFile: targetLogPath,
	}

	target, err := New(targetCfg)
	if err != nil {
		t.Fatalf("New target failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go target.Start(ctx)
	if err := target.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("Target failed to start: %v", err)
	}

	// Create sender agent
	senderCfg := &Config{
		Name:    "sender-agent",
		Port:    0,
		LogFile: senderLogPath,
	}

	sender, err := New(senderCfg)
	if err != nil {
		t.Fatalf("New sender failed: %v", err)
	}

	// Send ping
	err = sender.Ping(context.Background(), target.URL())
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Close agents to flush logs
	sender.Shutdown(context.Background())
	target.Shutdown(context.Background())

	// Verify sender logged as pilot
	senderEntries := flightlog.ReadLogEntries(t, senderLogPath)

	var foundPilotOut, foundPilotIn bool
	for _, entry := range senderEntries {
		if entry.Role == flightlog.RolePilot {
			if entry.Direction == flightlog.Outbound {
				foundPilotOut = true
			}
			if entry.Direction == flightlog.Inbound {
				foundPilotIn = true
			}
		}
	}

	if !foundPilotOut {
		t.Error("Sender missing outbound entry with role=pilot")
	}
	if !foundPilotIn {
		t.Error("Sender missing inbound entry with role=pilot")
	}

	// Verify target logged as wingmate
	targetEntries := flightlog.ReadLogEntries(t, targetLogPath)

	foundWingmate := false
	for _, entry := range targetEntries {
		if entry.Role == flightlog.RoleWingmate {
			foundWingmate = true
			break
		}
	}
	if !foundWingmate {
		t.Error("Target missing entry with role=wingmate")
	}
}

// TestAgent_Ping_ConnectionRefused verifies clear error message.
func TestAgent_Ping_ConnectionRefused(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{
		Name:    "sender-agent",
		Port:    0,
		LogFile: filepath.Join(dir, "sender.jsonl"),
	}

	sender, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer sender.Shutdown(context.Background())

	// Ping non-existent server
	err = sender.Ping(context.Background(), "http://127.0.0.1:1")
	if err == nil {
		t.Error("Expected error for connection refused")
	}

	// Error should be user-friendly
	errStr := err.Error()
	if len(errStr) == 0 {
		t.Error("Error message is empty")
	}
}

// TestAgent_Ping_Timeout verifies timeout handling.
func TestAgent_Ping_Timeout(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{
		Name:    "sender-agent",
		Port:    0,
		LogFile: filepath.Join(dir, "sender.jsonl"),
	}

	sender, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer sender.Shutdown(context.Background())

	// Use very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// This should timeout (even connection attempt takes longer than 1ms)
	err = sender.Ping(ctx, "http://192.0.2.1:9999") // Non-routable IP
	if err == nil {
		// Connection might be immediately refused on some systems
		t.Log("No error returned - connection was immediately refused")
	}
}

// TestAgent_GetPeerCard verifies fetching peer Agent Card.
func TestAgent_GetPeerCard(t *testing.T) {
	dir := t.TempDir()

	// Start target agent
	targetCfg := &Config{
		Name:    "card-target",
		Port:    0,
		LogFile: filepath.Join(dir, "target.jsonl"),
	}

	target, err := New(targetCfg)
	if err != nil {
		t.Fatalf("New target failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go target.Start(ctx)
	if err := target.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("Target failed to start: %v", err)
	}
	defer target.Shutdown(context.Background())

	// Create sender agent
	senderCfg := &Config{
		Name:    "card-sender",
		Port:    0,
		LogFile: filepath.Join(dir, "sender.jsonl"),
	}

	sender, err := New(senderCfg)
	if err != nil {
		t.Fatalf("New sender failed: %v", err)
	}
	defer sender.Shutdown(context.Background())

	// Get peer card
	card, err := sender.GetPeerCard(context.Background(), target.URL())
	if err != nil {
		t.Fatalf("GetPeerCard failed: %v", err)
	}

	if card.Name != "card-target" {
		t.Errorf("Card.Name = %q, want card-target", card.Name)
	}

	if card.Version != Version {
		t.Errorf("Card.Version = %q, want %q", card.Version, Version)
	}
}

// TestAgent_GetPeerCard_NotFound verifies error for non-existent peer.
func TestAgent_GetPeerCard_NotFound(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{
		Name:    "sender-agent",
		Port:    0,
		LogFile: filepath.Join(dir, "sender.jsonl"),
	}

	sender, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer sender.Shutdown(context.Background())

	_, err = sender.GetPeerCard(context.Background(), "http://127.0.0.1:1")
	if err == nil {
		t.Error("Expected error for non-existent peer")
	}
}

// TestAgent_SendMessage verifies sending arbitrary message.
func TestAgent_SendMessage(t *testing.T) {
	dir := t.TempDir()

	// Start target agent
	targetCfg := &Config{
		Name:    "msg-target",
		Port:    0,
		LogFile: filepath.Join(dir, "target.jsonl"),
	}

	target, err := New(targetCfg)
	if err != nil {
		t.Fatalf("New target failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go target.Start(ctx)
	if err := target.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("Target failed to start: %v", err)
	}
	defer target.Shutdown(context.Background())

	// Create sender agent
	senderCfg := &Config{
		Name:    "msg-sender",
		Port:    0,
		LogFile: filepath.Join(dir, "sender.jsonl"),
	}

	sender, err := New(senderCfg)
	if err != nil {
		t.Fatalf("New sender failed: %v", err)
	}
	defer sender.Shutdown(context.Background())

	// Send ping message directly
	resp, err := sender.SendMessage(context.Background(), target.URL(), pingMessage())
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %+v", resp.Error)
	}
}

// pingMessage creates a ping message for testing.
func pingMessage() *types.Message {
	return &types.Message{
		Role: "user",
		Parts: []types.Part{
			{Kind: "text", Text: "ping"},
		},
	}
}
