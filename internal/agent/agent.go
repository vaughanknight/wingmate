package agent

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/wingmate/wingmate/internal/flightlog"
	"github.com/wingmate/wingmate/internal/llm"
	"github.com/wingmate/wingmate/internal/mcp"
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

	// LLM integration
	llmExecutor    llm.LLMExecutor
	sessionManager *llm.SessionManager

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

	// Create LLM executor if CLI is available
	var llmExec llm.LLMExecutor
	var sessionMgr *llm.SessionManager

	// Build CLI executor options from config
	var cliOpts []llm.Option
	if cfg.Claude.CLIPath != "" {
		cliOpts = append(cliOpts, llm.WithCLIPath(cfg.Claude.CLIPath))
	}
	if cfg.Claude.Timeout > 0 {
		cliOpts = append(cliOpts, llm.WithTimeout(cfg.Claude.Timeout))
	}
	if cfg.Claude.Model != "" {
		cliOpts = append(cliOpts, llm.WithModel(cfg.Claude.Model))
	}

	// Create executor and check if CLI is installed
	executor := llm.NewCLIExecutor(cliOpts...)
	if executor.IsInstalled() {
		llmExec = executor
		sessionMgr = llm.NewSessionManager()
	}

	// Build Agent Card (includes chat skill if LLM available)
	card := buildAgentCard(cfg, llmExec != nil)

	agent := &Agent{
		config:         cfg,
		server:         server,
		client:         client,
		flightLog:      fl,
		card:           card,
		llmExecutor:    llmExec,
		sessionManager: sessionMgr,
		ready:          make(chan struct{}),
		knownPeers:     make(map[string]*types.AgentCard),
	}

	// Register A2A handler
	server.SetHandler(agent)
	server.SetAgentCard(card)

	// Initialize MCP HTTP handler for /mcp route
	mcpSessionMgr := mcp.NewMCPSessionManager(30*time.Minute, 5*time.Minute)
	mcpConfig := mcp.ServerConfig{
		Name:    "wingmate",
		Version: Version,
	}
	mcpHandler := mcp.NewHTTPHandler(mcpConfig, mcpSessionMgr)

	// Register MCP tools
	for _, tool := range mcp.DefaultTools() {
		mcpHandler.RegisterTool(tool)
	}

	// Register MCP handlers
	// Use existing llmExec (may be nil if CLI not available - handlers check IsInstalled)
	mcpHandler.RegisterHandler(mcp.ToolNameChat, mcp.NewChatHandler(llmExec, nil, nil))
	mcpHandler.RegisterHandler(mcp.ToolNameStatus, mcp.NewStatusHandler(llmExec, time.Now()))
	mcpHandler.RegisterHandler(mcp.ToolNameDiscover, mcp.NewDiscoverHandler(cfg.Name, agent)) // Agent implements PeerProvider

	// Register MCP handler with server (wrapped with localhost middleware for security)
	server.SetMCPHandler(mcp.LocalhostMiddleware(mcpHandler))

	return agent, nil
}

// buildAgentCard creates the Agent Card from config.
func buildAgentCard(cfg *Config, llmAvailable bool) *types.AgentCard {
	skills := []types.Skill{
		{
			Name:        "ping",
			Description: "Respond to ping with pong",
		},
	}

	// Add chat skill if LLM is available
	if llmAvailable {
		skills = append(skills, types.Skill{
			Name:        "chat",
			Description: "Process natural language messages using Claude",
		})
	}

	return &types.AgentCard{
		Name:        cfg.Name,
		Description: cfg.Description,
		URL:         "", // Set when server starts
		Version:     Version,
		Capabilities: types.Capabilities{
			Streaming:         false,
			PushNotifications: false,
		},
		Skills: skills,
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

			// Probe configured peers in background
			go a.probePeers(ctx)
			// Start background health re-probe
			go a.backgroundProbe(ctx)
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

// extractTextFromMessage extracts all text parts from a message, concatenated.
func extractTextFromMessage(msg *types.Message) string {
	if msg == nil || len(msg.Parts) == 0 {
		return ""
	}

	var texts []string
	for _, part := range msg.Parts {
		if part.Kind == "text" && part.Text != "" {
			texts = append(texts, part.Text)
		}
	}

	if len(texts) == 0 {
		return ""
	}

	// Join multiple text parts with newlines
	result := texts[0]
	for i := 1; i < len(texts); i++ {
		result += "\n" + texts[i]
	}
	return result
}

// handleLLMMessage processes a non-ping message using the LLM executor.
// It manages session continuity and logs all CLI interactions to the Flight Log.
func (a *Agent) handleLLMMessage(ctx context.Context, msg *types.Message) (*types.A2AResponse, error) {
	// Extract text from message for the prompt
	prompt := extractTextFromMessage(msg)
	if prompt == "" {
		return nil, &protocol.ErrorObject{
			Code:    protocol.CodeInvalidParams,
			Message: "message contains no text",
		}
	}

	// Get conversation ID for session tracking
	// Use the trace ID from context as conversation ID
	conversationID := flightlog.TraceIDFromContext(ctx)
	if conversationID == "" {
		// Fallback to a generated ID if no trace ID
		conversationID = fmt.Sprintf("conv-%d", time.Now().UnixNano())
	}

	// Get existing session ID for conversation continuity
	sessionID := a.sessionManager.Get(conversationID)

	// Log CLI request before execution (T008)
	startTime := time.Now()
	reqEntry := flightlog.NewEntryWithRole(
		ctx,
		a.config.Name,
		flightlog.RoleWingmate,
		flightlog.Outbound,
		"CLI request",
		map[string]interface{}{
			"type":      "cli_request",
			"prompt":    prompt,
			"sessionID": sessionID,
		},
	)
	a.flightLog.Record(reqEntry)

	// Execute CLI
	resp, err := a.llmExecutor.Execute(ctx, prompt, sessionID)
	if err != nil {
		// Log the error
		errEntry := flightlog.NewEntryWithRole(
			ctx,
			a.config.Name,
			flightlog.RoleWingmate,
			flightlog.Inbound,
			"CLI error",
			map[string]interface{}{
				"type":     "cli_error",
				"error":    err.Error(),
				"duration": time.Since(startTime).String(),
			},
		)
		a.flightLog.Record(errEntry)

		// Return the LLM error
		if llmErr, ok := err.(*llm.LLMError); ok {
			return nil, &protocol.ErrorObject{
				Code:    llmErr.Code,
				Message: llmErr.Message,
			}
		}
		return nil, &protocol.ErrorObject{
			Code:    llm.CodeLLMExecutionFailed,
			Message: err.Error(),
		}
	}

	// Store new session ID for continuity
	if resp.SessionID != "" {
		a.sessionManager.Set(conversationID, resp.SessionID)
	}

	// Log CLI response after execution (T009)
	duration := time.Since(startTime)
	respEntry := flightlog.NewEntryWithRole(
		ctx,
		a.config.Name,
		flightlog.RoleWingmate,
		flightlog.Inbound,
		"CLI response",
		map[string]interface{}{
			"type":         "cli_response",
			"sessionID":    resp.SessionID,
			"inputTokens":  resp.Usage.InputTokens,
			"outputTokens": resp.Usage.OutputTokens,
			"model":        resp.Metadata.Model,
			"duration":     duration.String(),
			"durationMs":   duration.Milliseconds(),
		},
	)
	a.flightLog.Record(respEntry)

	// Convert CLI response to A2A message
	responseMsg, err := llm.FromCLIResponse(resp)
	if err != nil {
		return nil, &protocol.ErrorObject{
			Code:    llm.CodeLLMInvalidResponse,
			Message: "failed to convert CLI response",
		}
	}

	// Log outbound response
	outEntry := flightlog.NewEntryWithRole(
		ctx,
		a.config.Name,
		flightlog.RoleWingmate,
		flightlog.Outbound,
		"Sending LLM response",
		responseMsg,
	)
	a.flightLog.Record(outEntry)

	return &types.A2AResponse{
		JSONRPC: "2.0",
		Result:  mustMarshal(responseMsg),
	}, nil
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

	// Route non-ping messages to LLM if available (T006, T007)
	if a.llmExecutor != nil {
		return a.handleLLMMessage(ctx, msg)
	}

	// No LLM available - return error 2001 (T007)
	return nil, &protocol.ErrorObject{
		Code:    llm.CodeLLMUnavailable,
		Message: "Claude CLI not installed - chat requires Claude CLI (https://claude.ai/download)",
	}
}

// GetPeers implements mcp.PeerProvider.
// Returns URLs of all known peer agents in a thread-safe manner.
func (a *Agent) GetPeers() []string {
	a.peersMu.RLock()
	defer a.peersMu.RUnlock()

	peers := make([]string, 0, len(a.knownPeers))
	for url := range a.knownPeers {
		peers = append(peers, url)
	}
	return peers
}

// backgroundProbe periodically re-probes peers to keep availability current.
func (a *Agent) backgroundProbe(ctx context.Context) {
	ticker := time.NewTicker(a.config.PeerProbeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.probePeers(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// probePeers probes all configured peers and populates knownPeers.
// Called as a goroutine after server is ready.
func (a *Agent) probePeers(ctx context.Context) {
	a.mu.RLock()
	peers := make([]string, len(a.config.Peers))
	copy(peers, a.config.Peers)
	a.mu.RUnlock()

	for _, peerURL := range peers {
		if ctx.Err() != nil {
			return
		}
		_, err := a.GetPeerCard(ctx, peerURL)
		if err != nil {
			// Log warning but don't fail — peer may come online later
			entry := flightlog.NewEntry(ctx, a.config.Name, flightlog.Outbound,
				fmt.Sprintf("Failed to probe peer %s: %v", peerURL, err), nil)
			a.flightLog.Record(entry)
		}
	}
}

// GetPeerInfo implements mcp.PeerProvider.
// Returns rich information about all known peer agents.
func (a *Agent) GetPeerInfo() []mcp.PeerInfo {
	a.peersMu.RLock()
	defer a.peersMu.RUnlock()

	info := make([]mcp.PeerInfo, 0, len(a.knownPeers))
	for url, card := range a.knownPeers {
		pi := mcp.PeerInfo{
			Name:        card.Name,
			URL:         url,
			Description: card.Description,
			Available:   true,
		}
		for _, skill := range card.Skills {
			pi.Skills = append(pi.Skills, mcp.PeerSkill{
				Name:        skill.Name,
				Description: skill.Description,
			})
		}
		info = append(info, pi)
	}
	return info
}

// Ensure Agent implements required interfaces.
var (
	_ protocol.MessageHandler = (*Agent)(nil)
	_ mcp.PeerProvider        = (*Agent)(nil)
)
