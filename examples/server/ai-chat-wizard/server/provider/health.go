package provider

import "time"

type ProviderHealthStatus string

const (
	ProviderHealthUnknown     ProviderHealthStatus = "unknown"
	ProviderHealthHealthy     ProviderHealthStatus = "healthy"
	ProviderHealthDegraded    ProviderHealthStatus = "degraded"
	ProviderHealthUnavailable ProviderHealthStatus = "unavailable"
)

// ProviderHealth is the normalized runtime/operational snapshot for one
// provider.
type ProviderHealth struct {
	ProviderID   string
	Status       ProviderHealthStatus
	LastSuccess  time.Time
	LastFailure  time.Time
	LastError    string
	LastLatency  time.Duration
	RequestCount int64
	TokenCount   int64
	RateLimits   RateLimitSnapshot
}
