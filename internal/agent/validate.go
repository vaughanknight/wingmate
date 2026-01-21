package agent

import (
	"fmt"
	"net/url"
)

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("config: %s: %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// Validate checks if the configuration is valid.
// Returns nil if valid, or a ValidationError if invalid.
func (c *Config) Validate() error {
	// Name is required
	if c.Name == "" {
		return NewValidationError("name", "required")
	}

	// Port must be in valid range (0 is allowed for auto-assign)
	if c.Port < 0 {
		return NewValidationError("port", "must be non-negative")
	}
	if c.Port > 65535 {
		return NewValidationError("port", "must be at most 65535")
	}

	// Validate peer URLs
	for i, peer := range c.Peers {
		if err := validatePeerURL(peer); err != nil {
			return NewValidationError(
				fmt.Sprintf("peers[%d]", i),
				fmt.Sprintf("invalid URL %q: %v", peer, err),
			)
		}
	}

	return nil
}

// validatePeerURL checks if a peer URL is valid.
func validatePeerURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	// Must have a scheme
	if parsed.Scheme == "" {
		return fmt.Errorf("missing scheme (http:// or https://)")
	}

	// Must be HTTP or HTTPS
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https, got %q", parsed.Scheme)
	}

	// Must have a host
	if parsed.Host == "" {
		return fmt.Errorf("missing host")
	}

	return nil
}

// IsValid returns true if the configuration is valid.
func (c *Config) IsValid() bool {
	return c.Validate() == nil
}

// MustValidate panics if the configuration is invalid.
// Useful for tests and initialization where invalid config is programmer error.
func (c *Config) MustValidate() {
	if err := c.Validate(); err != nil {
		panic(err)
	}
}
