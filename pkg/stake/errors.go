package stake

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidLogin is returned when credentials or a session token are rejected.
	ErrInvalidLogin = errors.New("stake: invalid login")
	// ErrMissingSessionToken is returned when token-based login has no token.
	ErrMissingSessionToken = errors.New("stake: missing session token")
	// ErrUnsupportedExchange is returned when an operation is unavailable for an exchange.
	ErrUnsupportedExchange = errors.New("stake: unsupported exchange")
	// ErrNotFound is returned when a requested local API object cannot be found.
	ErrNotFound = errors.New("stake: not found")
	// ErrTradeFailed is returned when Stake accepts a trade request that later fails.
	ErrTradeFailed = errors.New("stake: trade failed")
)

// APIError describes a non-2xx response from the Stake API.
type APIError struct {
	StatusCode int
	Method     string
	URL        string
	Body       []byte
}

func (e *APIError) Error() string {
	if e == nil {
		return "stake: api error"
	}

	if len(e.Body) == 0 {
		return fmt.Sprintf("stake: %s %s returned status %d", e.Method, e.URL, e.StatusCode)
	}

	return fmt.Sprintf("stake: %s %s returned status %d: %s", e.Method, e.URL, e.StatusCode, string(e.Body))
}
