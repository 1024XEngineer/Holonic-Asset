package llmclient

import (
	"errors"
	"fmt"
)

// ErrorKind classifies failures returned by an LLM provider.
type ErrorKind string

const (
	ErrorKindAuthentication  ErrorKind = "authentication"
	ErrorKindInvalidRequest  ErrorKind = "invalid_request"
	ErrorKindRateLimited     ErrorKind = "rate_limited"
	ErrorKindUnavailable     ErrorKind = "unavailable"
	ErrorKindTransport       ErrorKind = "transport"
	ErrorKindTimeout         ErrorKind = "timeout"
	ErrorKindCanceled        ErrorKind = "canceled"
	ErrorKindInvalidResponse ErrorKind = "invalid_response"
)

// ProviderError is a stable error representation for upstream provider calls.
type ProviderError struct {
	Provider   string
	Kind       ErrorKind
	StatusCode int
	Transient  bool
	Message    string
	Cause      error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return ""
	}
	provider := "LLM provider"
	if e.Provider != "" {
		provider = e.Provider + " provider"
	}
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", provider, e.Message)
	}
	return fmt.Sprintf("%s: %s", provider, e.Kind)
}

// Unwrap exposes the underlying transport or decoding error.
func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// IsTransient reports whether an error represents a transient provider failure.
func IsTransient(err error) bool {
	var providerErr *ProviderError
	return errors.As(err, &providerErr) && providerErr.Transient
}

func invalidRequestError(message string, cause error) *ProviderError {
	return &ProviderError{
		Kind:      ErrorKindInvalidRequest,
		Transient: false,
		Message:   message,
		Cause:     cause,
	}
}
