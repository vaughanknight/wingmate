package agent

import (
	"context"

	"github.com/wingmate/wingmate/internal/flightlog"
	"github.com/wingmate/wingmate/pkg/types"
)

// Conversation represents an ongoing A2A conversation.
// It tracks the role of this agent in the conversation.
type Conversation struct {
	TraceID   string
	Role      flightlog.Role // pilot or wingmate for this conversation
	PeerURL   string
	PeerName  string
	StartedAt int64
}

// NewPilotConversation creates a conversation where this agent is the pilot (initiator).
func NewPilotConversation(ctx context.Context, peerURL string) *Conversation {
	return &Conversation{
		TraceID: flightlog.TraceIDFromContext(ctx),
		Role:    flightlog.RolePilot,
		PeerURL: peerURL,
	}
}

// NewWingmateConversation creates a conversation where this agent is the wingmate (responder).
func NewWingmateConversation(ctx context.Context) *Conversation {
	return &Conversation{
		TraceID: flightlog.TraceIDFromContext(ctx),
		Role:    flightlog.RoleWingmate,
	}
}

// LogOutbound creates a Flight Log entry for an outbound message.
func (c *Conversation) LogOutbound(agentName, summary string, payload interface{}) *flightlog.Entry {
	entry := &flightlog.Entry{
		Agent:     agentName,
		Role:      c.Role,
		Direction: flightlog.Outbound,
		TraceID:   c.TraceID,
		Peer:      c.PeerName,
		Summary:   summary,
		Payload:   payload,
	}
	return entry
}

// LogInbound creates a Flight Log entry for an inbound message.
func (c *Conversation) LogInbound(agentName, summary string, payload interface{}) *flightlog.Entry {
	entry := &flightlog.Entry{
		Agent:     agentName,
		Role:      c.Role,
		Direction: flightlog.Inbound,
		TraceID:   c.TraceID,
		Peer:      c.PeerName,
		Summary:   summary,
		Payload:   payload,
	}
	return entry
}

// MessageContext holds context for message handling.
type MessageContext struct {
	Conversation *Conversation
	Message      *types.Message
}

// NewMessageContext creates a context for handling a message.
func NewMessageContext(ctx context.Context, msg *types.Message) *MessageContext {
	return &MessageContext{
		Conversation: NewWingmateConversation(ctx),
		Message:      msg,
	}
}
