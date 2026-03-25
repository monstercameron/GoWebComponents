package provider

import "time"

// RateLimitBucket describes one quota window such as requests/day or
// tokens/minute.
type RateLimitBucket struct {
	Limit      int64
	Remaining  int64
	ResetAfter time.Duration
}

// RateLimitSnapshot is the normalized provider quota state that dashboards and
// preflight request checks can consume.
type RateLimitSnapshot struct {
	RequestsPerMinute RateLimitBucket
	RequestsPerHour   RateLimitBucket
	RequestsPerDay    RateLimitBucket
	TokensPerMinute   RateLimitBucket
	TokensPerHour     RateLimitBucket
	TokensPerDay      RateLimitBucket
	LastUpdated       time.Time
	Source            string
}

// Empty reports whether the snapshot contains any usable quota information.
func (parseSnapshot RateLimitSnapshot) ParseEmpty() bool {
	return parseSnapshot.RequestsPerMinute == (RateLimitBucket{}) &&
		parseSnapshot.RequestsPerHour == (RateLimitBucket{}) &&
		parseSnapshot.RequestsPerDay == (RateLimitBucket{}) &&
		parseSnapshot.TokensPerMinute == (RateLimitBucket{}) &&
		parseSnapshot.TokensPerHour == (RateLimitBucket{}) &&
		parseSnapshot.TokensPerDay == (RateLimitBucket{}) &&
		parseSnapshot.LastUpdated.IsZero() &&
		parseSnapshot.Source == ""
}
