//go:build !js || !wasm

package fetch

import (
	"context"
	"errors"
	"testing"
	"time"
)

// parseNoopSleep is a context-aware sleep that returns immediately, used so
// backoff delays do not slow down the resilience test suite.
func parseNoopSleep(parseCtx context.Context, parseDur time.Duration) error {
	_ = parseDur
	select {
	case <-parseCtx.Done():
		return parseCtx.Err()
	default:
		return nil
	}
}

// parseDeterministicRand always returns 0.5, producing predictable jitter values.
func parseDeterministicRand() float64 { return 0.5 }

// parseZeroRand always returns 0.0 — the lower jitter bound.
func parseZeroRand() float64 { return 0.0 }

// parseOneRand always returns 0.999 — near the upper jitter bound.
func parseOneRand() float64 { return 0.999 }

// parseDefaultBreakerConfig returns a BreakerConfig wired for small-scale tests.
func parseDefaultBreakerConfig() BreakerConfig {
	return BreakerConfig{
		FailureThreshold: 3,
		OpenDuration:     5 * time.Second,
		HalfOpenMaxCalls: 1,
	}
}

// parseTestPolicy builds a ResiliencePolicy with no-op sleep and deterministic
// rand so tests run fast and produce reproducible results.
func parseTestPolicy(parseRetry RetryPolicy, parseBreaker *CircuitBreaker) ResiliencePolicy {
	parseP := NewResiliencePolicy(parseRetry, parseBreaker)
	parseP.sleep = parseNoopSleep
	parseP.randFloat = parseDeterministicRand
	return parseP
}

// TestBackoffDelaySequence verifies that delay(attempt) follows the exponential
// formula, caps at MaxDelay, and that jitter stays within the declared bounds.
func TestBackoffDelaySequence(parseT *testing.T) {
	parseRetry := RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    400 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      0.1,
	}

	// Attempt 1 is the first call — no prior failure, so delay should be zero.
	if parseD := parseRetry.delay(1, parseDeterministicRand); parseD != 0 {
		parseT.Fatalf("delay(1): expected 0, got %v", parseD)
	}

	// delay(2) = BaseDelay * Multiplier^0 = 100ms, with 0.0 rand giving min spread.
	parseD2Min := parseRetry.delay(2, parseZeroRand)
	parseD2Max := parseRetry.delay(2, parseOneRand)
	parseD2Base := 100 * time.Millisecond
	parseJitterBand := time.Duration(float64(parseD2Base) * parseRetry.Jitter)
	if parseD2Min < parseD2Base-parseJitterBand || parseD2Min > parseD2Base+parseJitterBand {
		parseT.Fatalf("delay(2, 0.0)=%v outside jitter band [%v, %v]", parseD2Min, parseD2Base-parseJitterBand, parseD2Base+parseJitterBand)
	}
	if parseD2Max < parseD2Base-parseJitterBand || parseD2Max > parseD2Base+parseJitterBand {
		parseT.Fatalf("delay(2, 1.0)=%v outside jitter band [%v, %v]", parseD2Max, parseD2Base-parseJitterBand, parseD2Base+parseJitterBand)
	}
	if parseD2Min >= parseD2Max {
		parseT.Fatalf("expected randFloat=0.0 to produce a lower delay than randFloat=1.0; got min=%v max=%v", parseD2Min, parseD2Max)
	}

	// delay(3) = BaseDelay * 2^1 = 200ms.
	parseD3 := parseRetry.delay(3, parseDeterministicRand)
	parseD3Base := 200 * time.Millisecond
	parseJ3 := time.Duration(float64(parseD3Base) * parseRetry.Jitter)
	if parseD3 < parseD3Base-parseJ3 || parseD3 > parseD3Base+parseJ3 {
		parseT.Fatalf("delay(3)=%v outside expected band [%v, %v]", parseD3, parseD3Base-parseJ3, parseD3Base+parseJ3)
	}

	// delay(4) = BaseDelay * 2^2 = 400ms = MaxDelay (capped).
	parseD4 := parseRetry.delay(4, parseDeterministicRand)
	parseD4Base := 400 * time.Millisecond // capped at MaxDelay
	parseJ4 := time.Duration(float64(parseD4Base) * parseRetry.Jitter)
	if parseD4 < parseD4Base-parseJ4 || parseD4 > parseD4Base+parseJ4 {
		parseT.Fatalf("delay(4)=%v outside expected capped band [%v, %v]", parseD4, parseD4Base-parseJ4, parseD4Base+parseJ4)
	}

	// delay(5) = BaseDelay * 2^3 = 800ms > MaxDelay → still capped at 400ms ± jitter.
	parseD5 := parseRetry.delay(5, parseDeterministicRand)
	if parseD5 < parseD4Base-parseJ4 || parseD5 > parseD4Base+parseJ4 {
		parseT.Fatalf("delay(5)=%v expected to be capped at MaxDelay band [%v, %v]", parseD5, parseD4Base-parseJ4, parseD4Base+parseJ4)
	}
}

func TestRetryDelayZeroMultiplierPinsZeroDelayFootGun(parseT *testing.T) {
	parseRetry := RetryPolicy{
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   time.Second,
		Multiplier: 0,
		Jitter:     0,
	}

	if parseDelay := parseRetry.delay(3, parseZeroRand); parseDelay != 0 {
		parseT.Fatalf("expected zero multiplier to compute zero delay for attempt 3, got %v", parseDelay)
	}
}

// TestRetryExhaustsMaxAttempts verifies that an always-failing operation is
// called exactly MaxAttempts times and the last error is returned.
func TestRetryExhaustsMaxAttempts(parseT *testing.T) {
	parseSentinel := errors.New("always fails")
	var parseCalls int

	parsePolicy := parseTestPolicy(RetryPolicy{
		MaxAttempts: 4,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
	}, nil)

	_, parseErr := ExecuteWithPolicy(context.Background(), parsePolicy, func(_ context.Context) (int, error) {
		parseCalls++
		return 0, parseSentinel
	})

	if !errors.Is(parseErr, parseSentinel) {
		parseT.Fatalf("expected sentinel error, got %v", parseErr)
	}
	if parseCalls != 4 {
		parseT.Fatalf("expected op called 4 times, got %d", parseCalls)
	}
}

// TestRetrySucceedsOnNthAttempt confirms that a transient failure followed by
// a success returns the successful result after the correct number of calls.
func TestRetrySucceedsOnNthAttempt(parseT *testing.T) {
	var parseCalls int
	parseSentinel := errors.New("transient")

	parsePolicy := parseTestPolicy(RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
	}, nil)

	parseResult, parseErr := ExecuteWithPolicy(context.Background(), parsePolicy, func(_ context.Context) (string, error) {
		parseCalls++
		if parseCalls < 3 {
			return "", parseSentinel
		}
		return "ok", nil
	})

	if parseErr != nil {
		parseT.Fatalf("expected success, got %v", parseErr)
	}
	if parseResult != "ok" {
		parseT.Fatalf("expected result %q, got %q", "ok", parseResult)
	}
	if parseCalls != 3 {
		parseT.Fatalf("expected op called 3 times, got %d", parseCalls)
	}
}

// TestRetryIfShortCircuits verifies that when RetryIf returns false the
// operation is never retried and the error surfaces immediately.
func TestRetryIfShortCircuits(parseT *testing.T) {
	parseSentinel := errors.New("non-retryable")
	var parseCalls int

	parsePolicy := parseTestPolicy(RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		RetryIf:     func(parseErr error) bool { return false },
	}, nil)

	_, parseErr := ExecuteWithPolicy(context.Background(), parsePolicy, func(_ context.Context) (int, error) {
		parseCalls++
		return 0, parseSentinel
	})

	if !errors.Is(parseErr, parseSentinel) {
		parseT.Fatalf("expected sentinel error, got %v", parseErr)
	}
	if parseCalls != 1 {
		parseT.Fatalf("expected op called exactly once, got %d", parseCalls)
	}
}

// TestCircuitBreakerOpensAfterThreshold drives enough failures through
// ExecuteWithPolicy to trip the circuit breaker, then asserts that the next
// call is rejected without invoking the operation.
func TestCircuitBreakerOpensAfterThreshold(parseT *testing.T) {
	parseConfig := BreakerConfig{
		FailureThreshold: 3,
		OpenDuration:     10 * time.Second,
		HalfOpenMaxCalls: 1,
	}
	parseBreaker := NewCircuitBreaker(parseConfig)

	parsePolicy := parseTestPolicy(RetryPolicy{
		MaxAttempts: 1, // one attempt per call so we control the failure count
		BaseDelay:   0,
		MaxDelay:    0,
		Multiplier:  1,
	}, parseBreaker)

	parseSentinel := errors.New("boom")
	var parseCalls int

	// Drive exactly FailureThreshold failures to open the breaker.
	for parseI := 0; parseI < parseConfig.FailureThreshold; parseI++ {
		parseCalls = 0
		ExecuteWithPolicy(context.Background(), parsePolicy, func(_ context.Context) (int, error) { //nolint:errcheck
			parseCalls++
			return 0, parseSentinel
		})
	}

	if parseBreaker.State() != StateOpen {
		parseT.Fatalf("expected breaker to be open after %d failures, got %v", parseConfig.FailureThreshold, parseBreaker.State())
	}

	// The next call must be rejected without touching the op.
	parseCallsBeforeReject := 0
	_, parseErr := ExecuteWithPolicy(context.Background(), parsePolicy, func(_ context.Context) (int, error) {
		parseCallsBeforeReject++
		return 0, nil
	})

	if !errors.Is(parseErr, ErrCircuitOpen) {
		parseT.Fatalf("expected ErrCircuitOpen, got %v", parseErr)
	}
	if parseCallsBeforeReject != 0 {
		parseT.Fatalf("expected op not to be called when circuit is open, got %d calls", parseCallsBeforeReject)
	}
}

// TestCircuitBreakerHalfOpenAndClose opens the breaker, advances the injected
// clock past OpenDuration, then confirms that a successful probe transitions
// the breaker back to closed.
func TestCircuitBreakerHalfOpenAndClose(parseT *testing.T) {
	parseConfig := parseDefaultBreakerConfig()
	parseBreaker := NewCircuitBreaker(parseConfig)

	// Fake clock starts in the past so we can advance it.
	parseFakeNow := time.Now()
	parseBreaker.setNow(func() time.Time { return parseFakeNow })

	// Open the breaker by recording failures directly.
	for parseI := 0; parseI < parseConfig.FailureThreshold; parseI++ {
		parseBreaker.recordFailure()
	}
	if parseBreaker.State() != StateOpen {
		parseT.Fatal("expected breaker to be open after threshold failures")
	}

	// Advance clock past OpenDuration so the next allow() transitions to half-open.
	parseFakeNow = parseFakeNow.Add(parseConfig.OpenDuration + time.Millisecond)

	parsePolicy := parseTestPolicy(RetryPolicy{
		MaxAttempts: 1,
		BaseDelay:   0,
		MaxDelay:    0,
		Multiplier:  1,
	}, parseBreaker)

	// Successful probe should close the breaker.
	_, parseErr := ExecuteWithPolicy(context.Background(), parsePolicy, func(_ context.Context) (string, error) {
		return "recovered", nil
	})
	if parseErr != nil {
		parseT.Fatalf("expected successful half-open probe, got %v", parseErr)
	}
	if parseBreaker.State() != StateClosed {
		parseT.Fatalf("expected breaker to be closed after successful probe, got %v", parseBreaker.State())
	}
}

// TestCircuitBreakerHalfOpenReopen opens the breaker, advances the clock, then
// confirms that a failing half-open probe re-opens the breaker.
func TestCircuitBreakerHalfOpenReopen(parseT *testing.T) {
	parseConfig := parseDefaultBreakerConfig()
	parseBreaker := NewCircuitBreaker(parseConfig)

	parseFakeNow := time.Now()
	parseBreaker.setNow(func() time.Time { return parseFakeNow })

	for parseI := 0; parseI < parseConfig.FailureThreshold; parseI++ {
		parseBreaker.recordFailure()
	}
	if parseBreaker.State() != StateOpen {
		parseT.Fatal("expected breaker to be open after threshold failures")
	}

	parseFakeNow = parseFakeNow.Add(parseConfig.OpenDuration + time.Millisecond)

	parsePolicy := parseTestPolicy(RetryPolicy{
		MaxAttempts: 1,
		BaseDelay:   0,
		MaxDelay:    0,
		Multiplier:  1,
	}, parseBreaker)

	parseSentinel := errors.New("still broken")
	_, parseErr := ExecuteWithPolicy(context.Background(), parsePolicy, func(_ context.Context) (int, error) {
		return 0, parseSentinel
	})
	if parseErr == nil {
		parseT.Fatal("expected error from failing half-open probe")
	}
	if parseBreaker.State() != StateOpen {
		parseT.Fatalf("expected breaker to re-open after failing half-open probe, got %v", parseBreaker.State())
	}
}

func TestCircuitBreakerHalfOpenZeroMaxCallsAllowsUnlimitedProbes(parseT *testing.T) {
	parseConfig := BreakerConfig{
		FailureThreshold: 1,
		OpenDuration:     time.Second,
		HalfOpenMaxCalls: 0,
	}
	parseBreaker := NewCircuitBreaker(parseConfig)

	parseFakeNow := time.Now()
	parseBreaker.setNow(func() time.Time { return parseFakeNow })
	parseBreaker.recordFailure()
	if parseBreaker.State() != StateOpen {
		parseT.Fatal("expected breaker to open after one failure")
	}

	parseFakeNow = parseFakeNow.Add(parseConfig.OpenDuration + time.Millisecond)
	for parseI := 0; parseI < 100; parseI++ {
		parseAllowed, parseErr := parseBreaker.allow()
		if parseErr != nil || !parseAllowed {
			parseT.Fatalf("expected unlimited half-open probe %d to be allowed, allowed=%t err=%v", parseI+1, parseAllowed, parseErr)
		}
	}
	if parseBreaker.State() != StateHalfOpen {
		parseT.Fatalf("expected breaker to stay half-open while probes remain unresolved, got %v", parseBreaker.State())
	}
}

// TestCtxCancellationAbortsRetry cancels the context after the first failure
// and verifies that the retry loop exits with the context error rather than
// continuing to retry.
func TestCtxCancellationAbortsRetry(parseT *testing.T) {
	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseSentinel := errors.New("transient")
	var parseCalls int

	// Use a sleep that honours the context so cancellation is detectable.
	parsePolicy := NewResiliencePolicy(RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   50 * time.Millisecond,
		MaxDelay:    500 * time.Millisecond,
		Multiplier:  2.0,
	}, nil)
	parsePolicy.randFloat = parseDeterministicRand
	// Real sleep so cancellation actually propagates through the timer select.
	// We cancel after the first failure, so the sleep fires before any second attempt.

	_, parseErr := ExecuteWithPolicy(parseCtx, parsePolicy, func(parseOpCtx context.Context) (int, error) {
		parseCalls++
		// Cancel the context on the first call so the subsequent sleep aborts.
		parseCancel()
		return 0, parseSentinel
	})

	if !errors.Is(parseErr, context.Canceled) {
		parseT.Fatalf("expected context.Canceled, got %v", parseErr)
	}
	if parseCalls != 1 {
		parseT.Fatalf("expected op called once before cancellation abort, got %d", parseCalls)
	}
}
