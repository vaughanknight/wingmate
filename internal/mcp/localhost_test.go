package mcp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

/*
Test Doc:
- Why: AC-5 requires localhost-only MCP access via application check (Critical Discovery 04)
- Contract: IsLocalhost returns true only for loopback addresses (127.x.x.x, ::1)
- Usage Notes: Check Request.RemoteAddr, not proxy headers like X-Forwarded-For
- Quality Contribution: Prevents remote MCP access security vulnerability
- Worked Example: "127.0.0.1:54321" → true, "192.168.1.1:54321" → false
*/

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		want       bool
	}{
		// IPv4 localhost cases
		{"IPv4 localhost with port", "127.0.0.1:54321", true},
		{"IPv4 localhost different port", "127.0.0.1:8080", true},
		{"IPv4 localhost range 127.0.0.2", "127.0.0.2:54321", true},
		{"IPv4 localhost range 127.255.255.255", "127.255.255.255:54321", true},

		// IPv6 localhost cases
		{"IPv6 localhost with brackets", "[::1]:54321", true},
		{"IPv6 localhost different port", "[::1]:8080", true},

		// Non-localhost IPv4
		{"IPv4 private 192.168", "192.168.1.1:54321", false},
		{"IPv4 private 10.x", "10.0.0.1:54321", false},
		{"IPv4 private 172.16", "172.16.0.1:54321", false},
		{"IPv4 public", "8.8.8.8:54321", false},

		// Non-localhost IPv6
		{"IPv6 remote", "[2001:db8::1]:54321", false},
		{"IPv6 link-local", "[fe80::1]:54321", false},

		// Edge cases
		{"empty string", "", false},
		{"malformed no port", "127.0.0.1", false},
		{"malformed garbage", "not-an-ip", false},
		{"malformed partial", "127.0.0", false},
		{"port only", ":54321", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			req.RemoteAddr = tt.remoteAddr

			if got := IsLocalhost(req); got != tt.want {
				t.Errorf("IsLocalhost() with RemoteAddr=%q = %v, want %v", tt.remoteAddr, got, tt.want)
			}
		})
	}
}

func TestIsLocalhost_IgnoresProxyHeaders(t *testing.T) {
	/*
	Test Doc:
	- Why: Security - proxy headers can be spoofed by attackers
	- Contract: IsLocalhost ignores X-Forwarded-For and X-Real-IP headers
	- Usage Notes: Only trust RemoteAddr which is set by Go's HTTP server
	- Quality Contribution: Prevents header-spoofing bypass of localhost check
	- Worked Example: RemoteAddr="192.168.1.1:54321" with X-Forwarded-For="127.0.0.1" → false
	*/

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.RemoteAddr = "192.168.1.1:54321" // Non-localhost actual address
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	req.Header.Set("X-Real-IP", "127.0.0.1")

	if got := IsLocalhost(req); got != false {
		t.Errorf("IsLocalhost() should ignore proxy headers, got %v, want false", got)
	}
}

func TestLocalhostMiddleware(t *testing.T) {
	/*
	Test Doc:
	- Why: AC-5 requires 403 Forbidden for non-localhost MCP access
	- Contract: LocalhostMiddleware returns 403 for non-localhost, passes through for localhost
	- Usage Notes: Wrap the MCP handler with this middleware
	- Quality Contribution: Enforces localhost-only access at HTTP layer
	- Worked Example: Non-localhost request → 403 Forbidden; localhost request → next handler called
	*/

	// Create a test handler that records if it was called
	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := LocalhostMiddleware(nextHandler)

	t.Run("localhost request passes through", func(t *testing.T) {
		called = false
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.RemoteAddr = "127.0.0.1:54321"
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		if !called {
			t.Error("expected next handler to be called for localhost request")
		}
		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("non-localhost request returns 403", func(t *testing.T) {
		called = false
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.RemoteAddr = "192.168.1.1:54321"
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		if called {
			t.Error("expected next handler NOT to be called for non-localhost request")
		}
		if rr.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", rr.Code)
		}
	})

	t.Run("IPv6 localhost passes through", func(t *testing.T) {
		called = false
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.RemoteAddr = "[::1]:54321"
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		if !called {
			t.Error("expected next handler to be called for IPv6 localhost request")
		}
		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})
}
