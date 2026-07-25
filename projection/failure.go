package projection

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Command failure model — plan items P3.8a and P3.8b.
//
// A command crossing a thread boundary can fail in ways an in-process call
// cannot, and the UI's correct response differs for each. Collapsing them into
// one error type forces every call site to string-match, which is how "retry on
// failure" turns into retrying a command the domain deliberately rejected.
//
// The kinds are distinguished because the RIGHT ACTION differs:
//
//	Rejected    the domain said no. Retrying repeats the same answer. Show it.
//	WorkerDied  the worker is gone. The command MAY have applied. Retrying is
//	            safe only if the command is idempotent — which is what the
//	            domain runtime's replay guarantee provides (P3.4).
//	TimedOut    no answer yet. Same ambiguity as death, and the work may still
//	            be running, so a retry can race the original.
//	Cancelled   the caller withdrew. Nothing to show and nothing to retry.
//	Transport   the message could not be delivered at all. The command did NOT
//	            apply, so retrying is unambiguously safe.
//
// The distinction between "did not apply" and "may have applied" is the one
// that matters most, and it is the reason Transport is separate from WorkerDied.

// FailureKind classifies why a command failed.
type FailureKind uint8

const (
	// FailureRejected means the domain received the command and refused it.
	FailureRejected FailureKind = iota
	// FailureWorkerDied means the worker terminated. The command may or may not
	// have applied before it went.
	FailureWorkerDied
	// FailureTimedOut means no answer arrived in time. The work may still be
	// running.
	FailureTimedOut
	// FailureCancelled means the caller withdrew the command.
	FailureCancelled
	// FailureTransport means the message never reached the worker, so the
	// command definitely did not apply.
	FailureTransport
)

func (parseKind FailureKind) String() string {
	switch parseKind {
	case FailureRejected:
		return "rejected"
	case FailureWorkerDied:
		return "worker-died"
	case FailureTimedOut:
		return "timed-out"
	case FailureCancelled:
		return "cancelled"
	case FailureTransport:
		return "transport"
	default:
		return fmt.Sprintf("unknown-failure(%d)", uint8(parseKind))
	}
}

// MayHaveApplied reports whether the command might have taken effect.
//
// This is the question a UI actually needs answered, and the reason the kinds
// exist. A failure that definitely did not apply can be retried freely and shown
// as "nothing happened"; one that may have applied must either be retried
// idempotently or reconciled against the domain before anything is shown.
func (parseKind FailureKind) MayHaveApplied() bool {
	switch parseKind {
	case FailureWorkerDied, FailureTimedOut:
		return true
	default:
		return false
	}
}

// Retryable reports whether retrying the same command could produce a different
// answer.
//
// A rejection cannot: the domain will say the same thing. A cancellation cannot:
// the caller withdrew it.
func (parseKind FailureKind) Retryable() bool {
	switch parseKind {
	case FailureTransport, FailureWorkerDied, FailureTimedOut:
		return true
	default:
		return false
	}
}

// CommandError is a classified command failure.
type CommandError struct {
	Kind    FailureKind
	Command string
	// Attempts is how many times the command was sent, so a caller can tell a
	// first-try failure from an exhausted retry policy.
	Attempts int
	Err      error
}

func (parseErr *CommandError) Error() string {
	return fmt.Sprintf("projection: command %q failed (%s) after %d attempt(s): %v",
		parseErr.Command, parseErr.Kind, parseErr.Attempts, parseErr.Err)
}

func (parseErr *CommandError) Unwrap() error { return parseErr.Err }

// ErrWorkerDied is returned by a transport whose worker has terminated.
//
// A transport reports it by wrapping this, which is how death is distinguished
// from an ordinary delivery failure without parsing messages.
var ErrWorkerDied = errors.New("projection: the domain worker terminated")

// ErrRejected is returned by a transport whose worker answered with a refusal.
//
// It exists because a rejection is otherwise indistinguishable from a delivery
// failure, and the two want opposite handling: a refusal reached the domain and
// repeating it yields the same answer, while an undelivered message should be
// retried. Without this, a "retry on failure" policy spends its attempts
// re-asking a question that has already been answered no.
//
// A transport reports it by wrapping this — the same mechanism as ErrWorkerDied
// — so classification never has to parse a message.
var ErrRejected = errors.New("projection: the domain rejected the command")

// ErrDomainPanic is returned when a domain handler panicked.
//
// It is a REJECTION rather than a death: P3.8a requires a domain panic to leave
// recoverable UI state, and that is only true if the worker contained the panic
// and stayed alive. A worker that dies on every panic gives the UI nothing to
// recover to.
var ErrDomainPanic = errors.New("projection: the domain handler panicked")

// Rejection builds the error a transport returns when the domain refused a
// command, preserving the domain's own message.
//
// Constructors rather than a documented convention, because "wrap ErrRejected"
// is exactly the kind of instruction that gets followed on three of four call
// sites — and the fourth silently becomes a retryable transport failure.
func Rejection(parseMessage string) error {
	return fmt.Errorf("%s: %w", parseMessage, ErrRejected)
}

// WorkerDeath builds the error a transport returns when its worker terminated.
func WorkerDeath(parseMessage string) error {
	return fmt.Errorf("%s: %w", parseMessage, ErrWorkerDied)
}

// DomainPanic builds the error a transport returns when a domain handler
// panicked and the worker contained it.
func DomainPanic(parseMessage string) error {
	return fmt.Errorf("%s: %w", parseMessage, ErrDomainPanic)
}

// Policy configures timeout and retry behaviour.
type Policy struct {
	// Timeout bounds one attempt. Zero means no timeout.
	Timeout time.Duration
	// MaxAttempts is the total number of attempts, including the first. Zero or
	// one means no retry.
	MaxAttempts int
	// Backoff is the delay before the second attempt; it doubles thereafter.
	// Zero retries immediately, which is rarely right against a restarting
	// worker.
	Backoff time.Duration
	// RetryMayHaveApplied allows retrying failures that might already have taken
	// effect.
	//
	// Off by default, and that default is the safe one: retrying a
	// non-idempotent command that already applied duplicates it. Turn it on when
	// commands carry stable ids and the domain runtime's replay guarantee (P3.4)
	// makes a repeat a no-op.
	RetryMayHaveApplied bool
	// Sleep is the delay function, injectable so retry tests do not sleep.
	Sleep func(parseCtx context.Context, parseDuration time.Duration) error
}

// Resilient wraps a CommandClient with timeout, retry, and failure
// classification.
type Resilient struct {
	inner  CommandClient
	policy Policy
}

// NewResilient wraps a client.
func NewResilient(parseInner CommandClient, parsePolicy Policy) (*Resilient, error) {
	if parseInner == nil {
		return nil, errors.New("projection: a client is required")
	}
	if parsePolicy.Sleep == nil {
		parsePolicy.Sleep = sleepWithContext
	}
	if parsePolicy.MaxAttempts < 1 {
		parsePolicy.MaxAttempts = 1
	}
	return &Resilient{inner: parseInner, policy: parsePolicy}, nil
}

func sleepWithContext(parseCtx context.Context, parseDuration time.Duration) error {
	if parseDuration <= 0 {
		return nil
	}
	parseTimer := time.NewTimer(parseDuration)
	defer parseTimer.Stop()
	select {
	case <-parseTimer.C:
		return nil
	case <-parseCtx.Done():
		return parseCtx.Err()
	}
}

// Send delivers a command, applying the policy and classifying any failure.
func (parseResilient *Resilient) Send(parseCtx context.Context, parseName string, parseRequest []byte) ([]byte, error) {
	if parseResilient == nil {
		return nil, errors.New("projection: resilient client is nil")
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}

	parseBackoff := parseResilient.policy.Backoff
	var parseLastErr *CommandError

	for parseAttempt := 1; parseAttempt <= parseResilient.policy.MaxAttempts; parseAttempt++ {
		parseResponse, parseErr := parseResilient.sendOnce(parseCtx, parseName, parseRequest)
		if parseErr == nil {
			return parseResponse, nil
		}

		parseLastErr = &CommandError{
			Kind:     classify(parseCtx, parseErr),
			Command:  parseName,
			Attempts: parseAttempt,
			Err:      parseErr,
		}

		if !parseResilient.shouldRetry(parseLastErr.Kind, parseAttempt) {
			return nil, parseLastErr
		}
		if parseSleepErr := parseResilient.policy.Sleep(parseCtx, parseBackoff); parseSleepErr != nil {
			parseLastErr.Kind = FailureCancelled
			return nil, parseLastErr
		}
		// Doubling rather than a fixed delay: a worker that is restarting needs
		// progressively more time, and hammering it delays the restart.
		parseBackoff *= 2
	}
	return nil, parseLastErr
}

func (parseResilient *Resilient) sendOnce(parseCtx context.Context, parseName string, parseRequest []byte) ([]byte, error) {
	if parseResilient.policy.Timeout <= 0 {
		return parseResilient.inner.Send(parseCtx, parseName, parseRequest)
	}
	parseAttemptCtx, parseCancel := context.WithTimeout(parseCtx, parseResilient.policy.Timeout)
	defer parseCancel()
	return parseResilient.inner.Send(parseAttemptCtx, parseName, parseRequest)
}

func (parseResilient *Resilient) shouldRetry(parseKind FailureKind, parseAttempt int) bool {
	if parseAttempt >= parseResilient.policy.MaxAttempts {
		return false
	}
	if !parseKind.Retryable() {
		return false
	}
	if parseKind.MayHaveApplied() && !parseResilient.policy.RetryMayHaveApplied {
		return false
	}
	return true
}

// classify turns a transport error into a failure kind.
//
// Order matters. A cancelled parent context reads as a deadline on the attempt
// context too, so the caller's own cancellation is checked FIRST — otherwise a
// user navigating away would be reported as a timeout and retried.
func classify(parseCtx context.Context, parseErr error) FailureKind {
	if parseCtx.Err() != nil && errors.Is(parseCtx.Err(), context.Canceled) {
		return FailureCancelled
	}
	if errors.Is(parseErr, ErrRejected) {
		return FailureRejected
	}
	if errors.Is(parseErr, ErrWorkerDied) {
		return FailureWorkerDied
	}
	if errors.Is(parseErr, ErrDomainPanic) {
		// A contained panic is the domain answering "no", not the worker dying.
		return FailureRejected
	}
	if errors.Is(parseErr, context.DeadlineExceeded) {
		return FailureTimedOut
	}
	if errors.Is(parseErr, context.Canceled) {
		return FailureCancelled
	}
	return FailureTransport
}

// Classify exposes the classification for callers holding an error from a
// plain, unwrapped client.
func Classify(parseErr error) FailureKind {
	if parseErr == nil {
		return FailureRejected
	}
	var parseCommandErr *CommandError
	if errors.As(parseErr, &parseCommandErr) {
		return parseCommandErr.Kind
	}
	return classify(context.Background(), parseErr)
}

// ------------------------------------------------------- optimistic updates

// Optimistic applies a local change immediately, runs a command, and undoes the
// local change if the command fails.
//
// The rollback condition is the point, and it is not "any error". A failure that
// MAY have applied must not be rolled back: undoing a change the domain actually
// made would leave the UI showing state the domain does not have, and the next
// published delta would silently contradict it. Those cases keep the optimistic
// state and are reported so the caller can reconcile — usually by waiting for
// the authoritative projection.
//
// The UI pattern this supports: apply, keep, and let the projection correct you.
// Roll back only when you know nothing happened.
func Optimistic[R any](
	parseCtx context.Context,
	parseApply func(),
	parseRollback func(),
	parseInvoke func(context.Context) (R, error),
) (R, error) {
	var parseZero R
	if parseApply == nil || parseRollback == nil || parseInvoke == nil {
		return parseZero, errors.New("projection: optimistic update needs apply, rollback, and invoke")
	}

	parseApply()

	parseResult, parseErr := parseInvoke(parseCtx)
	if parseErr == nil {
		return parseResult, nil
	}

	parseKind := Classify(parseErr)
	if parseKind.MayHaveApplied() {
		// Keep the optimistic state. Rolling back here is the tempting mistake:
		// it looks safe and produces a UI that disagrees with the domain.
		return parseZero, fmt.Errorf("projection: command outcome unknown (%s); optimistic state kept for the projection to reconcile: %w",
			parseKind, parseErr)
	}

	parseRollback()
	return parseZero, parseErr
}
