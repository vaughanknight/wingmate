package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wingmate/wingmate/internal/flightlog"
	"github.com/wingmate/wingmate/pkg/types"
)

// Ping sends a ping to a peer and expects a pong response.
// This agent acts as "pilot" in this conversation.
func (a *Agent) Ping(ctx context.Context, peerURL string) error {
	// Ensure trace ID exists
	ctx = flightlog.EnsureTraceID(ctx)

	// Build ping message
	msg := &types.Message{
		Role: "user",
		Parts: []types.Part{
			{Kind: "text", Text: "ping"},
		},
	}

	// Log outbound as pilot
	outEntry := flightlog.NewEntryWithRole(
		ctx,
		a.config.Name,
		flightlog.RolePilot,
		flightlog.Outbound,
		fmt.Sprintf("Sending ping to %s", peerURL),
		msg,
	)
	outEntry.Peer = extractPeerName(peerURL)
	outEntry.Method = "ping"
	a.flightLog.Record(outEntry)

	// Send message
	resp, err := a.client.SendMessage(ctx, peerURL, msg)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Check for error response
	if resp.Error != nil {
		return fmt.Errorf("ping error: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	// Parse response - convert interface{} to json.RawMessage
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}
	respMsg, err := parseMessageFromResult(resultBytes)
	if err != nil {
		return fmt.Errorf("invalid response: %w", err)
	}

	// Log inbound as pilot
	inEntry := flightlog.NewEntryWithRole(
		ctx,
		a.config.Name,
		flightlog.RolePilot,
		flightlog.Inbound,
		"Received pong",
		respMsg,
	)
	inEntry.Peer = extractPeerName(peerURL)
	inEntry.Method = "pong"
	a.flightLog.Record(inEntry)

	// Verify it's a pong
	if !isPong(respMsg) {
		return fmt.Errorf("expected pong, got: %+v", respMsg)
	}

	return nil
}

// SendMessage sends a message to a peer and returns the response.
// This agent acts as "pilot" in this conversation.
func (a *Agent) SendMessage(ctx context.Context, peerURL string, msg *types.Message) (*types.A2AResponse, error) {
	// Ensure trace ID exists
	ctx = flightlog.EnsureTraceID(ctx)

	// Log outbound as pilot
	outEntry := flightlog.NewEntryWithRole(
		ctx,
		a.config.Name,
		flightlog.RolePilot,
		flightlog.Outbound,
		fmt.Sprintf("Sending message to %s", peerURL),
		msg,
	)
	outEntry.Peer = extractPeerName(peerURL)
	a.flightLog.Record(outEntry)

	// Send message
	resp, err := a.client.SendMessage(ctx, peerURL, msg)
	if err != nil {
		return nil, err
	}

	// Log inbound as pilot
	inEntry := flightlog.NewEntryWithRole(
		ctx,
		a.config.Name,
		flightlog.RolePilot,
		flightlog.Inbound,
		"Received response",
		resp,
	)
	inEntry.Peer = extractPeerName(peerURL)
	a.flightLog.Record(inEntry)

	return resp, nil
}

// GetPeerCard fetches the Agent Card from a peer.
func (a *Agent) GetPeerCard(ctx context.Context, peerURL string) (*types.AgentCard, error) {
	card, err := a.client.GetAgentCard(ctx, peerURL)
	if err != nil {
		return nil, fmt.Errorf("get peer card: %w", err)
	}

	// Cache peer card
	a.peersMu.Lock()
	a.knownPeers[peerURL] = card
	a.peersMu.Unlock()

	return card, nil
}

// GetKnownPeer returns a cached peer Agent Card, if available.
func (a *Agent) GetKnownPeer(peerURL string) *types.AgentCard {
	a.peersMu.RLock()
	defer a.peersMu.RUnlock()
	return a.knownPeers[peerURL]
}

// extractPeerName extracts a peer name from URL for logging.
func extractPeerName(peerURL string) string {
	// Simple extraction: just use the URL as-is for now
	// In a real implementation, we might fetch the Agent Card first
	return peerURL
}
