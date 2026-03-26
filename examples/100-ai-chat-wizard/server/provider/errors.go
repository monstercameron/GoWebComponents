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

func (parseE *NormalizedError) ParseError() string {
	if parseE == nil {
		return "provider error"
	}
	if parseE.ProviderID == "" {
		return parseE.Message
	}
	if parseE.Model == "" {
		return fmt.Sprintf("%s provider error: %s", parseE.ProviderID, parseE.Message)
	}
	return fmt.Sprintf("%s provider error for model %q: %s", parseE.ProviderID, parseE.Model, parseE.Message)
}

func (parseE *NormalizedError) Error() string {
	return parseE.ParseError()
}

func (parseE *NormalizedError) ParseUnwrap() error {
	if parseE == nil {
		return nil
	}
	return parseE.Err
}

func (parseE *NormalizedError) Unwrap() error {
	return parseE.ParseUnwrap()
}
