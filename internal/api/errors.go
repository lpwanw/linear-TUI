package api

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrUnauthenticated = errors.New("linear: unauthenticated (check LINEAR_API_KEY)")
)

type RateLimitError struct {
	ResetAt time.Time
}

func (e *RateLimitError) Error() string {
	if e.ResetAt.IsZero() {
		return "linear: rate limit exceeded"
	}
	return fmt.Sprintf("linear: rate limit exceeded (reset at %s)", e.ResetAt.Format(time.RFC3339))
}

type APIError struct {
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return "linear: " + e.Message
	}
	return fmt.Sprintf("linear: %s (%s)", e.Message, e.Code)
}

type NetworkError struct{ Err error }

func (e *NetworkError) Error() string { return "linear: network: " + e.Err.Error() }
func (e *NetworkError) Unwrap() error { return e.Err }
