package mcp

import (
	"net"
	"net/http"
)

// IsLocalhost checks if the HTTP request originated from a loopback address.
// It examines Request.RemoteAddr (set by Go's HTTP server) and returns true
// only for 127.x.x.x (IPv4) or ::1 (IPv6) addresses.
//
// Security: This function intentionally ignores proxy headers like X-Forwarded-For
// and X-Real-IP because they can be spoofed by attackers. Only RemoteAddr,
// which is set by the TCP connection, is trusted.
func IsLocalhost(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr is malformed (missing port, empty, etc.)
		return false
	}

	ip := net.ParseIP(host)
	if ip == nil {
		// Could not parse as valid IP address
		return false
	}

	return ip.IsLoopback()
}

// LocalhostMiddleware wraps an http.Handler and rejects requests that do not
// originate from localhost. Non-localhost requests receive a 403 Forbidden
// response with a plain text error message.
//
// This middleware is used to protect the MCP endpoint, ensuring only local
// Claude Code instances can access it while the agent's A2A endpoints remain
// accessible from remote peers.
func LocalhostMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsLocalhost(r) {
			http.Error(w, "Forbidden: MCP endpoint is localhost-only", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
