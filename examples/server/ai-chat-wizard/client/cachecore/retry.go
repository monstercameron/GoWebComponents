package cachecore

import (
	"math"
	"strings"
	"time"
)

// OutboxRetryPolicy stores retry-engine controls for queued-write delivery.
type OutboxRetryPolicy struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	JitterRatio  float64
	MaxAttempts  int
}

// RetryDecision stores retry/terminal outcomes for one queued-write attempt.
type RetryDecision struct {
	ShouldRetry bool
	IsTerminal  bool
	NextRecord  OutboxRecordEnvelope
}

// BuildOutboxRetryPolicyDefaults returns one conservative retry policy for queued writes.
func BuildOutboxRetryPolicyDefaults() OutboxRetryPolicy {
	return NormalizeOutboxRetryPolicy(OutboxRetryPolicy{
		InitialDelay: 2 * time.Second,
		MaxDelay:     2 * time.Minute,
		JitterRatio:  0.20,
		MaxAttempts:  8,
	})
}

// NormalizeOutboxRetryPolicy normalizes retry policy bounds.
func NormalizeOutboxRetryPolicy(parsePolicy OutboxRetryPolicy) OutboxRetryPolicy {
	if parsePolicy.InitialDelay <= 0 {
		parsePolicy.InitialDelay = 2 * time.Second
	}
	if parsePolicy.MaxDelay < parsePolicy.InitialDelay {
		parsePolicy.MaxDelay = parsePolicy.InitialDelay
	}
	if parsePolicy.JitterRatio < 0 {
		parsePolicy.JitterRatio = 0
	}
	if parsePolicy.JitterRatio > 1 {
		parsePolicy.JitterRatio = 1
	}
	if parsePolicy.MaxAttempts <= 0 {
		parsePolicy.MaxAttempts = 1
	}
	return parsePolicy
}

// ResolveOutboxRetryDecision applies retry policy to one outbox record and returns the next retry decision.
func ResolveOutboxRetryDecision(parseRecord OutboxRecordEnvelope, parseNow time.Time, parsePolicy OutboxRetryPolicy, parseRandomFloat func() float64) RetryDecision {
	parsePolicy = NormalizeOutboxRetryPolicy(parsePolicy)
	parseRecord = NormalizeOutboxRecordEnvelope(parseRecord)
	if parseNow.IsZero() {
		parseNow = time.Now().UTC()
	}
	if parseRandomFloat == nil {
		parseRandomFloat = func() float64 { return 0.5 }
	}
	switch strings.TrimSpace(strings.ToLower(parseRecord.Status)) {
	case OutboxStatusAcked:
		return RetryDecision{ShouldRetry: false, IsTerminal: true, NextRecord: parseRecord}
	}
	if parseRecord.AttemptCount >= parsePolicy.MaxAttempts {
		parseRecord.Status = OutboxStatusFailed
		return RetryDecision{ShouldRetry: false, IsTerminal: true, NextRecord: parseRecord}
	}
	parseBackoffDelay := parseResolveExponentialBackoffDelay(parseRecord.AttemptCount, parsePolicy.InitialDelay, parsePolicy.MaxDelay)
	parseJitterMultiplier := parseResolveJitterMultiplier(parseRandomFloat, parsePolicy.JitterRatio)
	parseNextDelay := time.Duration(float64(parseBackoffDelay) * parseJitterMultiplier)
	if parseNextDelay < parsePolicy.InitialDelay {
		parseNextDelay = parsePolicy.InitialDelay
	}
	if parseNextDelay > parsePolicy.MaxDelay {
		parseNextDelay = parsePolicy.MaxDelay
	}
	parseRecord.AttemptCount++
	parseRecord.LastAttemptAt = parseNow.UTC().Format(time.RFC3339)
	parseRecord.NextRetryAt = parseNow.UTC().Add(parseNextDelay).Format(time.RFC3339)
	parseRecord.Status = OutboxStatusQueued
	return RetryDecision{
		ShouldRetry: true,
		IsTerminal:  false,
		NextRecord:  parseRecord,
	}
}

// parseResolveExponentialBackoffDelay resolves one capped exponential retry delay.
func parseResolveExponentialBackoffDelay(parseAttemptCount int, parseInitialDelay time.Duration, parseMaxDelay time.Duration) time.Duration {
	if parseAttemptCount < 0 {
		parseAttemptCount = 0
	}
	parseDelay := float64(parseInitialDelay) * math.Pow(2, float64(parseAttemptCount))
	if parseDelay > float64(parseMaxDelay) {
		return parseMaxDelay
	}
	return time.Duration(parseDelay)
}

// parseResolveJitterMultiplier resolves one jitter multiplier centered around 1.0.
func parseResolveJitterMultiplier(parseRandomFloat func() float64, parseJitterRatio float64) float64 {
	parseRandomValue := parseRandomFloat()
	if parseRandomValue < 0 {
		parseRandomValue = 0
	}
	if parseRandomValue > 1 {
		parseRandomValue = 1
	}
	parseJitterOffset := (parseRandomValue*2 - 1) * parseJitterRatio
	return 1 + parseJitterOffset
}
