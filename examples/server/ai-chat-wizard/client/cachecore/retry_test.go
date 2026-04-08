package cachecore

import (
	"testing"
	"time"
)

// TestResolveOutboxRetryDecisionBackoffAndCaps verifies retry backoff/jitter/cap behavior.
func TestResolveOutboxRetryDecisionBackoffAndCaps(parseT *testing.T) {
	parseNow := time.Now().UTC()
	parsePolicy := NormalizeOutboxRetryPolicy(OutboxRetryPolicy{
		InitialDelay: 2 * time.Second,
		MaxDelay:     8 * time.Second,
		JitterRatio:  0,
		MaxAttempts:  5,
	})
	parseRecord := OutboxRecordEnvelope{
		QueueKey:     "queue.logs",
		ResourceKey:  "resource.logs",
		OperationKey: "op-1",
		AttemptCount: 2,
		Status:       OutboxStatusSending,
	}
	parseDecision := ResolveOutboxRetryDecision(parseRecord, parseNow, parsePolicy, func() float64 { return 0.5 })
	if !parseDecision.ShouldRetry || parseDecision.IsTerminal {
		parseT.Fatalf("expected retry decision, got %+v", parseDecision)
	}
	if parseDecision.NextRecord.AttemptCount != 3 {
		parseT.Fatalf("expected attempt count increment to 3, got %d", parseDecision.NextRecord.AttemptCount)
	}
	parseNextRetryAt, isParseNextRetryAtValid := parseParseRFC3339(parseDecision.NextRecord.NextRetryAt)
	if !isParseNextRetryAtValid {
		parseT.Fatalf("invalid next_retry_at value %q", parseDecision.NextRecord.NextRetryAt)
	}
	parseDelay := parseNextRetryAt.Sub(parseNow)
	if parseDelay < 7*time.Second || parseDelay > 9*time.Second {
		parseT.Fatalf("expected capped backoff delay near 8s, got %s", parseDelay)
	}
}

// TestResolveOutboxRetryDecisionTerminalAfterMaxAttempts verifies terminal transition once max attempts are reached.
func TestResolveOutboxRetryDecisionTerminalAfterMaxAttempts(parseT *testing.T) {
	parsePolicy := NormalizeOutboxRetryPolicy(OutboxRetryPolicy{
		InitialDelay: 1 * time.Second,
		MaxDelay:     4 * time.Second,
		JitterRatio:  0.2,
		MaxAttempts:  2,
	})
	parseRecord := OutboxRecordEnvelope{
		QueueKey:     "queue.logs",
		ResourceKey:  "resource.logs",
		OperationKey: "op-2",
		AttemptCount: 2,
		Status:       OutboxStatusSending,
	}
	parseDecision := ResolveOutboxRetryDecision(parseRecord, time.Now().UTC(), parsePolicy, func() float64 { return 0.7 })
	if parseDecision.ShouldRetry || !parseDecision.IsTerminal {
		parseT.Fatalf("expected terminal no-retry decision, got %+v", parseDecision)
	}
	if parseDecision.NextRecord.Status != OutboxStatusFailed {
		parseT.Fatalf("expected failed terminal status, got %q", parseDecision.NextRecord.Status)
	}
}

// TestResolveOutboxRetryDecisionAckedTerminal verifies acked rows do not get retried.
func TestResolveOutboxRetryDecisionAckedTerminal(parseT *testing.T) {
	parseRecord := OutboxRecordEnvelope{
		QueueKey:     "queue.logs",
		ResourceKey:  "resource.logs",
		OperationKey: "op-3",
		AttemptCount: 1,
		Status:       OutboxStatusAcked,
	}
	parseDecision := ResolveOutboxRetryDecision(parseRecord, time.Now().UTC(), BuildOutboxRetryPolicyDefaults(), func() float64 { return 0.3 })
	if parseDecision.ShouldRetry || !parseDecision.IsTerminal {
		parseT.Fatalf("expected acked record to be terminal/no-retry, got %+v", parseDecision)
	}
	if parseDecision.NextRecord.Status != OutboxStatusAcked {
		parseT.Fatalf("expected acked status to remain unchanged, got %q", parseDecision.NextRecord.Status)
	}
}
