package agent

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/wingmate/wingmate/internal/flightlog"
	"github.com/wingmate/wingmate/internal/protocol"
	"github.com/wingmate/wingmate/pkg/types"
)

// Version is the agent version.
const Version = "0.1.0"

// Agent is a unified A2A peer that can both initiate and respond to conversations.
// Per ADR-003: "pilot" and "wingmate" are conversation roles, not instance modes.
type Agent struct {
	config    *Config
	server    protocol.Server
	client    protocol.Client
	flightLog *flightlog.FlightLog
	card      *types.AgentCard

	// State
	ready    chan struct{}
	started  bool
	shutdown bool
	mu       sync.RWMutex

	// Peer management
	knownPeers map[string]*types.AgentCard
	peersMu    sync.RWMutex
}

// New creates a new Agent with the given configuration.
func New(cfg *Config) (*Agent, error) {
	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Apply defaults
	cfg = cfg.WithDefaults()

	// Create Flight Log
	var opts []flightlog.Option
	if cfg.Verbose {
		opts = append(opts, flightlog.WithStdout(os.Stdout))
	}
	fl, err := flightlog.New(cfg.LogFile, opts...)
	if err != nil {
		return nil, fmt.Errorf("create flight log: %w", err)
	}

	// Create server and client
	server := protocol.NewA2AServer()
	client := protocol.NewA2AClient()

	// Build Agent Card
	card := buildAgentCard(cfg)

	agent := &Agent{
		config:     cfg,
		server:     server,
		client:     client,
		flightLog:  fl,
		card:       card,
		ready:      make(chan struct{}),
		knownPeers: make(map[string]*types.AgentCard),
	}

	// Register handler
	server.SetHandler(agent)
	server.SetAgentCard(card)

	return agent, nil
}

// buildAgentCard creates the Agent Card from config.
func buildAgentCard(cfg *Config) *types.AgentCard {
	return &types.AgentCard{
		Name:        cfg.Name,
		Description: "Wingmate A2A agent",
		URL:         "", // Set when server starts
		Version:     Version,
		Capabilities: types.Capabilities{
			Streaming:         false,
			PushNotifications: false,
		},
		Skills: []types.Skill{
			{
				Name:        "ping",
				Description: "Respond to ping with pong",
			},
		},
		Authentication: &types.Authentication{
			Schemes: []string{},
		},
	}
}

// Name returns the agent name.
func (a *Agent) Name() string {
	return a.config.Name
}

// Addr returns the address the server is listening on.
func (a *Agent) Addr() string {
	return a.server.Addr()
}

// URL returns the full URL of the agent.
func (a *Agent) URL() string {
	addr := a.Addr()
	if addr == "" {
		return ""
	}
	return fmt.Sprintf("http://%s", addr)
}

// AgentCard returns the agent's Agent Card.
func (a *Agent) AgentCard() *types.AgentCard {
	return a.card
}

// IsReady returns true if the agent is ready to accept connections.
func (a *Agent) IsReady() bool {
	select {
	case <-a.ready:
		return true
	default:
		return false
	}
}

// Start begins listening for incoming requests.
// This method blocks until the context is cancelled or an error occurs.
func (a *Agent) Start(ctx context.Context) error {
	a.mu.Lock()
	if a.started {
		a.mu.Unlock()
		return fmt.Errorf("agent already started")
	}
	if a.shutdown {
		a.mu.Unlock()
		return fmt.Errorf("agent already shutdown")
	}
	a.started = true
	a.mu.Unlock()

	// Build listen address
	addr := fmt.Sprintf(":%d", a.config.Port)

	// Start server in goroutine so we can monitor ready
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.server.ListenAndServe(ctx, addr)
	}()

	// Wait for server ready
	if rn, ok := a.server.(protocol.ReadyNotifier); ok {
		select {
		case <-rn.Ready():
			// Update Agent Card URL now that we know the address
			a.card.URL = a.URL()
			a.server.SetAgentCard(a.card)
			close(a.ready)
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Wait for completion
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return a.Shutdown(context.Background())
	}
}

// WaitUntilReady blocks until the agent is ready or timeout expires.
func (a *Agent) WaitUntilReady(timeout time.Duration) error {
	select {
	case <-a.ready:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for agent to be ready")
	}
}

// Shutdown gracefully shuts down the agent.
func (a *Agent) Shutdown(ctx context.Context) error {
	a.mu.Lock()
	if a.shutdown {
		a.mu.Unlock()
		return nil
	}
	a.shutdown = true
	a.mu.Unlock()

	// Close client
	if err := a.client.Close(); err != nil {
		// Log but don't fail
	}

	// Shutdown server
	if err := a.server.Shutdown(ctx); err != nil {
		return err
	}

	// Close Flight Log
	if err := a.flightLog.Close(); err != nil {
		return err
	}

	return nil
}

// HandleMessage processes an incoming message (acts as "wingmate" in this conversation).
// This implements protocol.MessageHandler.
func (a *Agent) HandleMessage(ctx context.Context, msg *types.Message) (*types.A2AResponse, error) {
	// Handle nil or empty message
	if msg == nil || len(msg.Parts) == 0 {
		return nil, &protocol.ErrorObject{
			Code:    protocol.CodeInvalidParams,
			Message: "empty or nil message",
		}
	}

	// Log inbound as wingmate
	entry := flightlog.NewEntryWithRole(
		ctx,
		a.config.Name,
		flightlog.RoleWingmate,
		flightlog.Inbound,
		"Received message",
		msg,
	)
	a.flightLog.Record(entry)

	// Handle based on message content
	if isPing(msg) {
		response := pongResponse()

		// Log outbound response
		outEntry := flightlog.NewEntryWithRole(
			ctx,
			a.config.Name,
			flightlog.RoleWingmate,
			flightlog.Outbound,
			"Sending pong",
			response,
		)
		a.flightLog.Record(outEntry)

		return &types.A2AResponse{
			JSONRPC: "2.0",
			Result:  mustMarshal(response),
		}, nil
	}

	// Unknown message type
	return nil, &protocol.ErrorObject{
		Code:    protocol.CodeMethodNotFound,
		Message: "unknown message type",
	}
}

// Ensure Agent implements protocol.MessageHandler.
var _ protocol.MessageHandler = (*Agent)(nil)
