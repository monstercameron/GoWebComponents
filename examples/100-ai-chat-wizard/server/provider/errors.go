package provider

import "fmt"

type ErrorKind string

const (
	ErrorKindAuthConfig ErrorKind = "auth_config"
	ErrorKindRateLimit  ErrorKind = "rate_limit"
	ErrorKindInvalid    ErrorKind = "invalid_request"
	ErrorKindUpstream   ErrorKind = "upstream_failure"
	ErrorKindTimeout    ErrorKind = "timeout"
	ErrorKindStream     ErrorKind = "stream_failure"
	ErrorKindUnknown    ErrorKind = "unknown"
)

// NormalizedError is the provider-agnostic error envelope for higher layers.
type NormalizedError struct {
	Kind       ErrorKind
	ProviderID string
	Model      string
	Message    string
	Retryable  bool
	StatusCode int
	Err        error
}

func (e *NormalizedError) Error() string {
	if e == nil {
		return "provider error"
	}
	if e.ProviderID == "" {
		return e.Message
	}
	if e.Model == "" {
		return fmt.Sprintf("%s provider error: %s", e.ProviderID, e.Message)
	}
	return fmt.Sprintf("%s provider error for model %q: %s", e.ProviderID, e.Model, e.Message)
}

func (e *NormalizedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
