package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wingmate/wingmate/internal/flightlog"
	"github.com/wingmate/wingmate/internal/mcp"
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

// DelegateMessage implements mcp.PeerDelegator.
// Resolves a peer by name or URL, sends a message, and returns the response text.
func (a *Agent) DelegateMessage(ctx context.Context, peer string, message string, sessionID string) (string, error) {
	// Resolve peer to URL
	peerURL := a.resolvePeer(peer)
	if peerURL == "" {
		return "", mcp.NewMCPError(mcp.ErrCodePeerNotFound, fmt.Sprintf("peer not found: %s", peer))
	}

	// Build message
	msg := &types.Message{
		Role: "user",
		Parts: []types.Part{
			{Kind: "text", Text: message},
		},
	}

	// Send via A2A
	resp, err := a.SendMessage(ctx, peerURL, msg)
	if err != nil {
		return "", mcp.WrapMCPError(mcp.ErrCodeDelegationFailed, fmt.Sprintf("delegation to %s failed", peer), err)
	}

	// Check for error response
	if resp.Error != nil {
		return "", mcp.NewMCPError(mcp.ErrCodeDelegationFailed,
			fmt.Sprintf("peer %s returned error: %s (code %d)", peer, resp.Error.Message, resp.Error.Code))
	}

	// Extract text from result
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return "", mcp.WrapMCPError(mcp.ErrCodeDelegationFailed, "failed to marshal peer response", err)
	}

	respMsg, err := parseMessageFromResult(resultBytes)
	if err != nil {
		return "", mcp.WrapMCPError(mcp.ErrCodeDelegationFailed, "failed to parse peer response", err)
	}

	return extractTextFromMessage(respMsg), nil
}

// resolvePeer resolves a peer identifier (name or URL) to a URL.
// If peer starts with "http", it's treated as a URL directly.
// Otherwise, it's looked up by name in knownPeers.
func (a *Agent) resolvePeer(peer string) string {
	// Direct URL
	if len(peer) >= 4 && peer[:4] == "http" {
		return peer
	}

	// Resolve by name
	a.peersMu.RLock()
	defer a.peersMu.RUnlock()

	for url, card := range a.knownPeers {
		if card.Name == peer {
			return url
		}
	}

	return ""
}

// extractPeerName extracts a peer name from URL for logging.
func extractPeerName(peerURL string) string {
	// Simple extraction: just use the URL as-is for now
	// In a real implementation, we might fetch the Agent Card first
	return peerURL
}
