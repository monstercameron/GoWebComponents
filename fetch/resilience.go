package fetch

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

// ErrCircuitOpen is returned by ExecuteWithPolicy when the circuit breaker is
// in the open state and is not yet eligible to transition to half-open.
var ErrCircuitOpen = errors.New("fetch: circuit breaker is open")

// RetryPolicy describes how ExecuteWithPolicy retries a failing operation,
// using exponential backoff with optional jitter.
type RetryPolicy struct {
	// MaxAttempts is the total number of times the operation may be called,
	// including the first attempt. Zero or negative values disable retries.
	MaxAttempts int

	// BaseDelay is the wait time before the second attempt.
	BaseDelay time.Duration

	// MaxDelay caps the computed backoff delay so it never grows unboundedly.
	MaxDelay time.Duration

	// Multiplier scales the delay between successive retries. A value of 2.0
	// doubles the delay after each failure.
	Multiplier float64

	// Jitter is a fractional spread added to each delay to prevent thundering
	// herds. A value of 0.1 allows a ±10 % random spread around the base delay.
	Jitter float64

	// RetryIf, when non-nil, is called with each error to decide whether the
	// operation should be retried. Returning false stops retrying immediately
	// and surfaces the error to the caller. When nil, all errors are retried.
	RetryIf func(error) bool
}

// DefaultRetryPolicy returns a RetryPolicy suitable for most HTTP operations:
// three attempts, 100 ms base delay doubling up to 5 s, with 10 % jitter and
// no error filter (all errors are retried).
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		Multiplier:  2.0,
		Jitter:      0.1,
	}
}

// delay computes the backoff duration before attempt parseAttempt (1-indexed).
// parseRandFloat must return a value in [0, 1); it is injected so callers can
// supply a deterministic source in tests.
func (parseP RetryPolicy) delay(parseAttempt int, parseRandFloat func() float64) time.Duration {
	if parseAttempt <= 1 {
		return 0
	}
	parseExponent := float64(parseAttempt - 1)
	parseBase := float64(parseP.BaseDelay) * math.Pow(parseP.Multiplier, parseExponent-1)
	if parseMax := float64(parseP.MaxDelay); parseBase > parseMax {
		parseBase = parseMax
	}
	// Apply jitter: spread the delay uniformly within ±Jitter fraction.
	parseJitterRange := parseBase * parseP.Jitter
	parseJitterOffset := (parseRandFloat()*2 - 1) * parseJitterRange
	parseResult := parseBase + parseJitterOffset
	if parseResult < 0 {
		parseResult = 0
	}
	return time.Duration(parseResult)
}

// BreakerState represents the operational state of a CircuitBreaker.
type BreakerState int

const (
	// StateClosed means the circuit breaker is healthy and allows all calls.
	StateClosed BreakerState = iota

	// StateOpen means the circuit breaker has tripped and fast-fails all calls.
	StateOpen

	// StateHalfOpen means the circuit breaker is probing whether the downstream
	// is healthy again, allowing a limited number of calls through.
	StateHalfOpen
)

// BreakerConfig holds the tuning parameters for a CircuitBreaker.
type BreakerConfig struct {
	// FailureThreshold is the consecutive-failure count that trips the breaker.
	FailureThreshold int

	// OpenDuration is how long the breaker stays open before probing with
	// half-open calls.
	OpenDuration time.Duration

	// HalfOpenMaxCalls is the maximum number of calls allowed in the half-open
	// state before the breaker re-opens (if all half-open calls succeed, it
	// instead resets to closed).
	HalfOpenMaxCalls int
}

// CircuitBreaker protects a downstream dependency by tracking consecutive
// failures and temporarily refusing calls once a failure threshold is crossed.
// The zero value is not valid; use NewCircuitBreaker.
type CircuitBreaker struct {
	mu            sync.Mutex
	state         BreakerState
	failures      int
	openedAt      time.Time
	halfOpenCalls int
	config        BreakerConfig
	now           func() time.Time
}

// NewCircuitBreaker constructs a CircuitBreaker with the supplied configuration
// and a real wall-clock source. Inject a synthetic clock via setNow for tests.
func NewCircuitBreaker(parseConfig BreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: parseConfig,
		now:    time.Now,
	}
}

// setNow replaces the wall-clock source used by the breaker. It is intended
// exclusively for same-package tests that need deterministic time control.
func (parseB *CircuitBreaker) setNow(parseF func() time.Time) {
	parseB.mu.Lock()
	defer parseB.mu.Unlock()
	parseB.now = parseF
}

// State returns the current BreakerState without modifying any internal fields.
func (parseB *CircuitBreaker) State() BreakerState {
	parseB.mu.Lock()
	defer parseB.mu.Unlock()
	return parseB.state
}

// allow decides whether an incoming call should proceed. It transitions the
// breaker from open to half-open when OpenDuration has elapsed. Returns
// false + ErrCircuitOpen when the breaker rejects the call.
func (parseB *CircuitBreaker) allow() (bool, error) {
	parseB.mu.Lock()
	defer parseB.mu.Unlock()

	switch parseB.state {
	case StateClosed:
		return true, nil

	case StateOpen:
		parseElapsed := parseB.now().Sub(parseB.openedAt)
		if parseElapsed < parseB.config.OpenDuration {
			return false, ErrCircuitOpen
		}
		// Transition to half-open for a single probe.
		parseB.state = StateHalfOpen
		parseB.halfOpenCalls = 0
		fallthrough

	case StateHalfOpen:
		if parseB.halfOpenCalls >= parseB.config.HalfOpenMaxCalls && parseB.config.HalfOpenMaxCalls > 0 {
			return false, ErrCircuitOpen
		}
		parseB.halfOpenCalls++
		return true, nil
	}

	return true, nil
}

// recordSuccess resets the breaker to closed, clearing failure counts.
func (parseB *CircuitBreaker) recordSuccess() {
	parseB.mu.Lock()
	defer parseB.mu.Unlock()
	parseB.state = StateClosed
	parseB.failures = 0
	parseB.halfOpenCalls = 0
}

// recordFailure increments the consecutive-failure counter and opens the
// breaker when the threshold is reached or a half-open probe fails.
func (parseB *CircuitBreaker) recordFailure() {
	parseB.mu.Lock()
	defer parseB.mu.Unlock()

	switch parseB.state {
	case StateClosed:
		parseB.failures++
		if parseB.config.FailureThreshold > 0 && parseB.failures >= parseB.config.FailureThreshold {
			parseB.state = StateOpen
			parseB.openedAt = parseB.now()
		}

	case StateHalfOpen:
		// Any failure in half-open immediately re-opens the breaker.
		parseB.state = StateOpen
		parseB.openedAt = parseB.now()
		parseB.halfOpenCalls = 0
	}
}

// ResiliencePolicy bundles a RetryPolicy and an optional CircuitBreaker into a
// single unit that ExecuteWithPolicy uses to drive resilient operation calls.
// Construct via NewResiliencePolicy; tests may overwrite the unexported fields
// directly because they live in the same package.
type ResiliencePolicy struct {
	// Retry controls backoff and attempt limits.
	Retry RetryPolicy

	// Breaker, when non-nil, gates every call through circuit-breaker logic.
	Breaker *CircuitBreaker

	// sleep is context-aware; it returns ctx.Err() if the context is cancelled
	// during the wait. Injected in tests to make sleep a no-op.
	sleep func(context.Context, time.Duration) error

	// randFloat supplies random values in [0, 1) for jitter. Injected in tests
	// for deterministic delay sequences.
	randFloat func() float64
}

// NewResiliencePolicy constructs a ResiliencePolicy wired to real-time sleep
// and real random jitter. Pass the result to ExecuteWithPolicy.
func NewResiliencePolicy(parseRetry RetryPolicy, parseBreaker *CircuitBreaker) ResiliencePolicy {
	return ResiliencePolicy{
		Retry:   parseRetry,
		Breaker: parseBreaker,
		sleep: func(parseCtx context.Context, parseDur time.Duration) error {
			if parseDur <= 0 {
				return nil
			}
			parseTimer := time.NewTimer(parseDur)
			defer parseTimer.Stop()
			select {
			case <-parseCtx.Done():
				return parseCtx.Err()
			case <-parseTimer.C:
				return nil
			}
		},
		randFloat: rand.Float64,
	}
}

// ExecuteWithPolicy runs parseOp under the retry and circuit-breaker rules
// described by parsePolicy. It is generic so the caller keeps type safety
// without casting the result.
//
// Execution order:
//  1. If the circuit breaker is open, return immediately with ErrCircuitOpen.
//  2. Call parseOp. On success, record success and return.
//  3. On error, consult RetryIf; if it returns false, record failure and return.
//  4. Record the failure, then sleep (respecting context cancellation).
//  5. Repeat up to RetryPolicy.MaxAttempts times, then return the last error.
func ExecuteWithPolicy[T any](parseCtx context.Context, parsePolicy ResiliencePolicy, parseOp func(context.Context) (T, error)) (T, error) {
	var parseZero T

	// Resolve defaults for injected dependencies so callers never have to think
	// about them outside of tests.
	parseRandFloat := parsePolicy.randFloat
	if parseRandFloat == nil {
		parseRandFloat = rand.Float64
	}
	parseSleep := parsePolicy.sleep
	if parseSleep == nil {
		parseSleep = func(parseCtx2 context.Context, parseDur time.Duration) error {
			if parseDur <= 0 {
				return nil
			}
			parseTimer := time.NewTimer(parseDur)
			defer parseTimer.Stop()
			select {
			case <-parseCtx2.Done():
				return parseCtx2.Err()
			case <-parseTimer.C:
				return nil
			}
		}
	}

	// Fast-fail if the breaker is already open before we attempt anything.
	if parsePolicy.Breaker != nil {
		if parseAllowed, parseBreakerErr := parsePolicy.Breaker.allow(); !parseAllowed {
			return parseZero, parseBreakerErr
		}
		// allow() incremented halfOpenCalls; if we succeed or fail we must
		// record that below. But we need to re-enter allow() on each attempt,
		// so undo the increment here: the per-attempt allow() call below will
		// count properly.
		//
		// Actually — the design calls for checking allow() once per
		// ExecuteWithPolicy invocation, then looping internally. Half-open
		// counts one call per ExecuteWithPolicy call, not per retry. We keep
		// the check above as the gate and do NOT call allow() again inside the
		// loop.  The halfOpenCalls counter was already incremented; proceed.
	}

	parseMaxAttempts := parsePolicy.Retry.MaxAttempts
	if parseMaxAttempts <= 0 {
		parseMaxAttempts = 1
	}

	var parseLastErr error
	for parseAttempt := 1; parseAttempt <= parseMaxAttempts; parseAttempt++ {
		// Honour context cancellation before every attempt.
		select {
		case <-parseCtx.Done():
			return parseZero, parseCtx.Err()
		default:
		}

		parseResult, parseOpErr := parseOp(parseCtx)
		if parseOpErr == nil {
			if parsePolicy.Breaker != nil {
				parsePolicy.Breaker.recordSuccess()
			}
			return parseResult, nil
		}

		// RetryIf veto: do not retry, surface the error immediately.
		if parsePolicy.Retry.RetryIf != nil && !parsePolicy.Retry.RetryIf(parseOpErr) {
			if parsePolicy.Breaker != nil {
				parsePolicy.Breaker.recordFailure()
			}
			return parseZero, parseOpErr
		}

		if parsePolicy.Breaker != nil {
			parsePolicy.Breaker.recordFailure()
		}

		parseLastErr = parseOpErr

		// No sleep needed after the final attempt.
		if parseAttempt == parseMaxAttempts {
			break
		}

		parseDelay := parsePolicy.Retry.delay(parseAttempt+1, parseRandFloat)
		if parseSleepErr := parseSleep(parseCtx, parseDelay); parseSleepErr != nil {
			return parseZero, parseSleepErr
		}
	}

	return parseZero, parseLastErr
}
