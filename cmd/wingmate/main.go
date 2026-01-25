package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/wingmate/wingmate/internal/agent"
)

// Exit codes.
const (
	ExitSuccess     = 0
	ExitConfigError = 1
	ExitPeerError   = 2
	ExitInternal    = 3
)

// Command-line flags.
var (
	name       = flag.String("name", "", "Agent name (required for server mode)")
	port       = flag.Int("port", 0, "Listen port (default: 9000)")
	peers      = flag.String("peers", "", "Comma-separated list of peer URLs")
	logFile    = flag.String("log", "", "Flight log path (default: ./flight.jsonl)")
	verbose    = flag.Bool("verbose", false, "Mirror logs to stdout")
	configPath = flag.String("config", "", "Config file path")
	help       = flag.Bool("help", false, "Show help")
)

func main() {
	os.Exit(run())
}

func run() int {
	flag.Usage = usage
	flag.Parse()

	if *help {
		usage()
		return ExitSuccess
	}

	// Get command and arguments
	args := flag.Args()
	command := ""
	cmdArgs := []string{}
	if len(args) > 0 {
		command = args[0]
		cmdArgs = args[1:]
	}

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		return ExitConfigError
	}

	// Execute command
	switch command {
	case "":
		return runServer(cfg)
	case "mcp":
		return runMCPMigrationMessage()
	case "ping":
		if len(cmdArgs) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: wingmate ping <peer-url>")
			return ExitConfigError
		}
		return runPing(cfg, cmdArgs[0])
	case "status":
		if len(cmdArgs) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: wingmate status <peer-url>")
			return ExitConfigError
		}
		return runStatus(cfg, cmdArgs[0])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		usage()
		return ExitConfigError
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: wingmate [options] [command]

Wingmate is an A2A (Agent-to-Agent) communication tool with MCP support.

Options:
  --name      Agent name (required for server mode)
  --port      Listen port (default: 9000)
  --peers     Comma-separated list of peer URLs
  --log       Flight log path (default: ./flight.jsonl)
  --verbose   Mirror logs to stdout
  --config    Config file path
  --help      Show help

Commands:
  (none)              Start agent server (serves A2A and MCP)
  ping <peer-url>     Send ping to a peer
  status <peer-url>   Fetch peer's Agent Card

MCP Integration:
  MCP tools are available at http://localhost:<port>/mcp
  Configure Claude Code: claude mcp add --transport http wingmate http://localhost:9000/mcp

Examples:
  wingmate --port 9000 --name my-agent
  wingmate ping http://localhost:9001 --name sender
  wingmate status http://localhost:9001
`)
}

func loadConfig() (*agent.Config, error) {
	// Start with defaults
	cfg := agent.NewConfig()

	// Load from file if specified
	if *configPath != "" {
		loaded, err := agent.LoadConfig(*configPath)
		if err != nil {
			return nil, fmt.Errorf("load config: %w", err)
		}
		cfg = loaded
	}

	// Apply environment variables
	cfg = cfg.WithEnv()

	// Apply defaults
	cfg = cfg.WithDefaults()

	// Apply CLI flags
	flagCfg := &agent.Config{}
	if *name != "" {
		flagCfg.Name = *name
	}
	if *port != 0 {
		flagCfg.Port = *port
	}
	if *peers != "" {
		flagCfg.Peers = parsePeers(*peers)
	}
	if *logFile != "" {
		flagCfg.LogFile = *logFile
	}
	if *verbose {
		flagCfg.Verbose = true
	}
	cfg = cfg.Merge(flagCfg)

	return cfg, nil
}

func parsePeers(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	peers := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			peers = append(peers, p)
		}
	}
	return peers
}

func runServer(cfg *agent.Config) int {
	// Validate config for server mode
	if cfg.Name == "" {
		fmt.Fprintln(os.Stderr, "Error: --name is required for server mode")
		return ExitConfigError
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		return ExitConfigError
	}

	// Create agent
	a, err := agent.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create agent: %v\n", err)
		return ExitInternal
	}

	// Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start agent in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.Start(ctx)
	}()

	// Wait for ready
	if err := a.WaitUntilReady(10 * time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start agent: %v\n", err)
		return ExitInternal
	}

	fmt.Printf("Agent %q listening on %s\n", cfg.Name, a.URL())
	fmt.Printf("Agent Card: %s/.well-known/agent.json\n", a.URL())
	fmt.Println("Press Ctrl+C to stop")

	// Wait for shutdown signal or error
	select {
	case sig := <-sigCh:
		fmt.Printf("\nReceived %v, shutting down...\n", sig)
	case err := <-errCh:
		if err != nil {
			fmt.Fprintf(os.Stderr, "Agent error: %v\n", err)
			return ExitInternal
		}
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := a.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "Shutdown error: %v\n", err)
		return ExitInternal
	}

	fmt.Println("Agent stopped")
	return ExitSuccess
}

func runPing(cfg *agent.Config, peerURL string) int {
	// Generate name if not provided
	if cfg.Name == "" {
		cfg.Name = "wingmate-ping"
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		return ExitConfigError
	}

	// Create agent (don't start server)
	a, err := agent.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create agent: %v\n", err)
		return ExitInternal
	}
	defer a.Shutdown(context.Background())

	// Send ping
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Sending ping to %s...\n", peerURL)

	if err := a.Ping(ctx, peerURL); err != nil {
		fmt.Fprintf(os.Stderr, "Ping failed: %v\n", err)
		return ExitPeerError
	}

	fmt.Println("Received pong!")
	return ExitSuccess
}

func runStatus(cfg *agent.Config, peerURL string) int {
	// Generate name if not provided
	if cfg.Name == "" {
		cfg.Name = "wingmate-status"
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		return ExitConfigError
	}

	// Create agent (don't start server)
	a, err := agent.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create agent: %v\n", err)
		return ExitInternal
	}
	defer a.Shutdown(context.Background())

	// Get peer card
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Fetching Agent Card from %s...\n", peerURL)

	card, err := a.GetPeerCard(ctx, peerURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get Agent Card: %v\n", err)
		return ExitPeerError
	}

	// Print agent card
	fmt.Println()
	fmt.Printf("Name:        %s\n", card.Name)
	fmt.Printf("Version:     %s\n", card.Version)
	fmt.Printf("Description: %s\n", card.Description)
	fmt.Printf("URL:         %s\n", card.URL)
	fmt.Printf("Streaming:   %v\n", card.Capabilities.Streaming)
	fmt.Printf("Push:        %v\n", card.Capabilities.PushNotifications)

	if len(card.Skills) > 0 {
		fmt.Println("Skills:")
		for _, skill := range card.Skills {
			fmt.Printf("  - %s: %s\n", skill.Name, skill.Description)
		}
	}

	return ExitSuccess
}

// runMCPMigrationMessage prints guidance for users trying to use the removed stdio MCP mode.
// The 'mcp' command was removed in Phase 3 of the MCP HTTP Transport migration.
// MCP is now available via HTTP at the /mcp endpoint.
func runMCPMigrationMessage() int {
	fmt.Fprintln(os.Stderr, `Error: The 'mcp' command has been removed. MCP is now available via HTTP.

Start the agent:
  wingmate --port 9000 --name my-agent

Configure Claude Code:
  claude mcp add --transport http wingmate http://localhost:9000/mcp

See docs/how/mcp-setup.md for detailed configuration.`)
	return ExitConfigError
}
