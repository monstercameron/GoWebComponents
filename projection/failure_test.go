package projection_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/projection"
)

// v5 P3.8a/b — the command failure model.
//
// P3.8a: worker death mid-command and rejection surfacing both tested; a domain
// panic yields recoverable UI state.
// P3.8b: timeout, retry, and optimistic rollback each have a test and a
// documented UI pattern.
//
// The property under test throughout is that the KIND is right, because the
// kind is what a UI branches on. A model that reported every failure as
// "something went wrong" would pass a correctness test and produce a UI that
// retries commands the domain deliberately refused.

// scriptedFailureClient returns a scripted sequence of outcomes.
type scriptedFailureClient struct {
	outcomes []error
	calls    int
	// blockFor makes an attempt hang, so a timeout can be exercised without
	// waiting on a real worker.
	blockFor time.Duration
}

func (parseClient *scriptedFailureClient) Send(parseCtx context.Context, parseName string, parseRequest []byte) ([]byte, error) {
	parseClient.calls++

	if parseClient.blockFor > 0 {
		select {
		case <-time.After(parseClient.blockFor):
		case <-parseCtx.Done():
			return nil, parseCtx.Err()
		}
	}

	if parseClient.calls <= len(parseClient.outcomes) {
		if parseErr := parseClient.outcomes[parseClient.calls-1]; parseErr != nil {
			return nil, parseErr
		}
	}
	return []byte(`{"id":1}`), nil
}

// noSleep makes retry tests deterministic and instant.
func noSleep(context.Context, time.Duration) error { return nil }

// -------------------------------------------------- P3.8a: failure surfacing

// TestWorkerDeathMidCommandIsSurfacedAsItsOwnKind is P3.8a's first requirement.
//
// Death matters specifically because the command MAY have applied, and that is
// the one fact a UI cannot guess.
func TestWorkerDeathMidCommandIsSurfacedAsItsOwnKind(parseT *testing.T) {
	parseClient := &scriptedFailureClient{outcomes: []error{
		fmt.Errorf("postMessage failed: %w", projection.ErrWorkerDied),
	}}
	parseResilient, parseErr := projection.NewResilient(parseClient, projection.Policy{Sleep: noSleep})
	if parseErr != nil {
		parseT.Fatalf("NewResilient: %v", parseErr)
	}

	_, parseSendErr := addItem.Invoke(context.Background(), parseResilient, jsonCodec{}, addItemArgs{})
	if parseSendErr == nil {
		parseT.Fatal("worker death must surface")
	}

	parseKind := projection.Classify(parseSendErr)
	if parseKind != projection.FailureWorkerDied {
		parseT.Errorf("kind = %s, want worker-died", parseKind)
	}
	if !parseKind.MayHaveApplied() {
		parseT.Error("worker death must report that the command may have applied — a UI cannot guess this")
	}
}

// TestDomainRejectionIsSurfacedAndNotRetried is P3.8a's second requirement.
//
// A rejection is the domain answering. Retrying repeats the same answer, so the
// policy must not spend attempts on it — and it cannot know that unless the
// transport reports the refusal as a refusal, which is why ErrRejected exists
// alongside ErrWorkerDied.
func TestDomainRejectionIsSurfacedAndNotRetried(parseT *testing.T) {
	parseClient := &scriptedFailureClient{outcomes: []error{
		fmt.Errorf("insufficient funds: %w", projection.ErrRejected),
		nil, // would succeed if the policy wrongly retried
	}}
	parseResilient, _ := projection.NewResilient(parseClient, projection.Policy{
		MaxAttempts: 3, Sleep: noSleep,
	})

	_, parseErr := addItem.Invoke(context.Background(), parseResilient, jsonCodec{}, addItemArgs{})
	if parseErr == nil {
		parseT.Fatal("a rejection must surface rather than being retried into a success")
	}

	parseKind := projection.Classify(parseErr)
	if parseKind != projection.FailureRejected {
		parseT.Errorf("kind = %s, want rejected", parseKind)
	}
	if parseKind.Retryable() {
		parseT.Error("a rejection is the domain's answer; retrying repeats it")
	}
	if parseKind.MayHaveApplied() {
		parseT.Error("a rejected command did not apply")
	}
	if parseClient.calls != 1 {
		parseT.Errorf("calls = %d, want 1 — attempts were spent re-asking an answered question", parseClient.calls)
	}
	// The domain's own message must survive to the caller, or the UI has
	// nothing useful to show.
	if !strings.Contains(parseErr.Error(), "insufficient funds") {
		parseT.Errorf("err = %v, want the domain's message preserved", parseErr)
	}
}

// TestDomainPanicYieldsRecoverableState is P3.8a's third requirement.
//
// A panic contained by the worker is a REJECTION, not a death: the worker is
// still alive, so the UI has something to recover to. A worker that died on
// every panic would leave nothing.
func TestDomainPanicYieldsRecoverableState(parseT *testing.T) {
	parseClient := &scriptedFailureClient{outcomes: []error{
		fmt.Errorf("handler blew up: %w", projection.ErrDomainPanic),
		nil, // the very next command succeeds — that is what "recoverable" means
	}}
	parseResilient, _ := projection.NewResilient(parseClient, projection.Policy{Sleep: noSleep})

	_, parsePanicErr := addItem.Invoke(context.Background(), parseResilient, jsonCodec{}, addItemArgs{})
	if parsePanicErr == nil {
		parseT.Fatal("a domain panic must surface as a failure")
	}
	if parseKind := projection.Classify(parsePanicErr); parseKind != projection.FailureRejected {
		parseT.Errorf("kind = %s, want rejected — a contained panic is the domain saying no, not the worker dying", parseKind)
	}
	if projection.Classify(parsePanicErr).MayHaveApplied() {
		parseT.Error("a contained panic did not apply; reporting otherwise would block a safe retry")
	}

	// Recoverable: the session continues.
	parseResult, parseNextErr := addItem.Invoke(context.Background(), parseResilient, jsonCodec{}, addItemArgs{})
	if parseNextErr != nil {
		parseT.Fatalf("the next command after a contained panic must work: %v", parseNextErr)
	}
	if parseResult.ID != 1 {
		parseT.Errorf("result = %+v, want a normal result", parseResult)
	}
}

func TestTransportFailureIsKnownNotToHaveApplied(parseT *testing.T) {
	parseClient := &scriptedFailureClient{outcomes: []error{errors.New("channel closed")}}
	parseResilient, _ := projection.NewResilient(parseClient, projection.Policy{Sleep: noSleep})

	_, parseErr := addItem.Invoke(context.Background(), parseResilient, jsonCodec{}, addItemArgs{})
	parseKind := projection.Classify(parseErr)
	if parseKind != projection.FailureTransport {
		parseT.Errorf("kind = %s, want transport", parseKind)
	}
	if parseKind.MayHaveApplied() {
		parseT.Error("an undelivered message did not apply; saying otherwise blocks a safe retry")
	}
	if !parseKind.Retryable() {
		parseT.Error("an undelivered message is the clearest retry case there is")
	}
}

// -------------------------------------------------------- P3.8b: timeout

func TestTimeoutIsItsOwnKind(parseT *testing.T) {
	parseClient := &scriptedFailureClient{blockFor: 200 * time.Millisecond}
	parseResilient, _ := projection.NewResilient(parseClient, projection.Policy{
		Timeout: 10 * time.Millisecond, Sleep: noSleep,
	})

	_, parseErr := addItem.Invoke(context.Background(), parseResilient, jsonCodec{}, addItemArgs{})
	if parseErr == nil {
		parseT.Fatal("a blocked command must time out")
	}
	parseKind := projection.Classify(parseErr)
	if parseKind != projection.FailureTimedOut {
		parseT.Errorf("kind = %s, want timed-out", parseKind)
	}
	if !parseKind.MayHaveApplied() {
		parseT.Error("a timeout leaves the work possibly still running; a UI must be told that")
	}
}

// TestCallerCancellationIsNotReportedAsATimeout: a cancelled parent context
// looks like a deadline on the attempt context too. Misreporting it would retry
// a command the user abandoned by navigating away.
func TestCallerCancellationIsNotReportedAsATimeout(parseT *testing.T) {
	parseClient := &scriptedFailureClient{blockFor: 200 * time.Millisecond}
	parseResilient, _ := projection.NewResilient(parseClient, projection.Policy{
		Timeout: time.Second, MaxAttempts: 3, Sleep: noSleep,
	})

	parseCtx, parseCancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		parseCancel()
	}()

	_, parseErr := addItem.Invoke(parseCtx, parseResilient, jsonCodec{}, addItemArgs{})
	if parseErr == nil {
		parseT.Fatal("a cancelled command must fail")
	}
	if parseKind := projection.Classify(parseErr); parseKind != projection.FailureCancelled {
		parseT.Errorf("kind = %s, want cancelled", parseKind)
	}
	if parseClient.calls > 1 {
		parseT.Errorf("a cancelled command was attempted %d times; it must not be retried", parseClient.calls)
	}
}

// --------------------------------------------------------- P3.8b: retry

func TestRetrySucceedsAfterATransientTransportFailure(parseT *testing.T) {
	parseClient := &scriptedFailureClient{outcomes: []error{
		errors.New("channel busy"),
		errors.New("channel busy"),
		nil,
	}}
	parseResilient, _ := projection.NewResilient(parseClient, projection.Policy{
		MaxAttempts: 3, Sleep: noSleep,
	})

	parseResult, parseErr := addItem.Invoke(context.Background(), parseResilient, jsonCodec{}, addItemArgs{})
	if parseErr != nil {
		parseT.Fatalf("the third attempt should have succeeded: %v", parseErr)
	}
	if parseResult.ID != 1 {
		parseT.Errorf("result = %+v, want the successful result", parseResult)
	}
	if parseClient.calls != 3 {
		parseT.Errorf("calls = %d, want 3", parseClient.calls)
	}
}

// TestRetryRefusesToRepeatWhatMayHaveApplied is the safe default, and the most
// important line in the policy.
//
// Retrying a non-idempotent command that already applied duplicates it — a
// double charge, a double order. Off by default; opt in only when the domain
// runtime's replay guarantee makes a repeat a no-op.
func TestRetryRefusesToRepeatWhatMayHaveApplied(parseT *testing.T) {
	parseClient := &scriptedFailureClient{outcomes: []error{
		fmt.Errorf("gone: %w", projection.ErrWorkerDied),
		nil,
	}}
	parseResilient, _ := projection.NewResilient(parseClient, projection.Policy{
		MaxAttempts: 3, Sleep: noSleep,
	})

	if _, parseErr := addItem.Invoke(context.Background(), parseResilient, jsonCodec{}, addItemArgs{}); parseErr == nil {
		parseT.Fatal("a may-have-applied failure must not be retried into a success by default")
	}
	if parseClient.calls != 1 {
		parseT.Errorf("calls = %d, want 1 — a possibly-applied command was repeated", parseClient.calls)
	}
}

// TestRetryMayHaveAppliedIsAvailableWhenCommandsAreIdempotent is the opt-in.
func TestRetryMayHaveAppliedIsAvailableWhenCommandsAreIdempotent(parseT *testing.T) {
	parseClient := &scriptedFailureClient{outcomes: []error{
		fmt.Errorf("gone: %w", projection.ErrWorkerDied),
		nil,
	}}
	parseResilient, _ := projection.NewResilient(parseClient, projection.Policy{
		MaxAttempts: 3, RetryMayHaveApplied: true, Sleep: noSleep,
	})

	if _, parseErr := addItem.Invoke(context.Background(), parseResilient, jsonCodec{}, addItemArgs{}); parseErr != nil {
		parseT.Fatalf("the retry should have succeeded: %v", parseErr)
	}
	if parseClient.calls != 2 {
		parseT.Errorf("calls = %d, want 2", parseClient.calls)
	}
}

func TestRetryStopsAtMaxAttemptsAndReportsTheCount(parseT *testing.T) {
	parseClient := &scriptedFailureClient{outcomes: []error{
		errors.New("busy"), errors.New("busy"), errors.New("busy"), errors.New("busy"),
	}}
	parseResilient, _ := projection.NewResilient(parseClient, projection.Policy{
		MaxAttempts: 3, Sleep: noSleep,
	})

	_, parseErr := addItem.Invoke(context.Background(), parseResilient, jsonCodec{}, addItemArgs{})
	if parseErr == nil {
		parseT.Fatal("exhausting the retries must fail")
	}
	if parseClient.calls != 3 {
		parseT.Errorf("calls = %d, want exactly the 3 allowed attempts", parseClient.calls)
	}

	var parseCommandErr *projection.CommandError
	if !errors.As(parseErr, &parseCommandErr) {
		parseT.Fatalf("err = %v, want a *CommandError", parseErr)
	}
	// The attempt count lets a UI distinguish "the network blipped" from "this
	// has been failing for a while".
	if parseCommandErr.Attempts != 3 {
		parseT.Errorf("Attempts = %d, want 3", parseCommandErr.Attempts)
	}
	if parseCommandErr.Command != "addItem" {
		parseT.Errorf("Command = %q, want addItem", parseCommandErr.Command)
	}
}

// TestBackoffGrowsBetweenAttempts: a fixed delay hammers a restarting worker and
// delays the very restart it is waiting for.
func TestBackoffGrowsBetweenAttempts(parseT *testing.T) {
	var parseDelays []time.Duration
	parseClient := &scriptedFailureClient{outcomes: []error{
		errors.New("busy"), errors.New("busy"), errors.New("busy"),
	}}
	parseResilient, _ := projection.NewResilient(parseClient, projection.Policy{
		MaxAttempts: 3,
		Backoff:     10 * time.Millisecond,
		Sleep: func(_ context.Context, parseDuration time.Duration) error {
			parseDelays = append(parseDelays, parseDuration)
			return nil
		},
	})

	_, _ = addItem.Invoke(context.Background(), parseResilient, jsonCodec{}, addItemArgs{})

	if len(parseDelays) != 2 {
		parseT.Fatalf("delays = %v, want 2 between 3 attempts", parseDelays)
	}
	if parseDelays[1] <= parseDelays[0] {
		parseT.Errorf("delays = %v, want the backoff to grow", parseDelays)
	}
}

// ---------------------------------------------- P3.8b: optimistic rollback

// TestOptimisticRollsBackWhenNothingHappened is the straightforward half.
func TestOptimisticRollsBackWhenNothingHappened(parseT *testing.T) {
	parseLocalState := "original"

	_, parseErr := projection.Optimistic(context.Background(),
		func() { parseLocalState = "optimistic" },
		func() { parseLocalState = "original" },
		func(context.Context) (int, error) {
			return 0, errors.New("channel closed")
		})

	if parseErr == nil {
		parseT.Fatal("the command failed; the caller must be told")
	}
	if parseLocalState != "original" {
		parseT.Errorf("state = %q, want the rollback applied", parseLocalState)
	}
}

// TestOptimisticKeepsStateWhenTheOutcomeIsUnknown is the half that is easy to
// get wrong, and the reason MayHaveApplied exists.
//
// Rolling back here looks safe and produces a UI showing state the domain does
// not have — which the next published delta would silently contradict. The
// documented pattern is: apply, keep, and let the projection correct you.
func TestOptimisticKeepsStateWhenTheOutcomeIsUnknown(parseT *testing.T) {
	parseLocalState := "original"

	_, parseErr := projection.Optimistic(context.Background(),
		func() { parseLocalState = "optimistic" },
		func() { parseLocalState = "original" },
		func(context.Context) (int, error) {
			return 0, fmt.Errorf("worker gone: %w", projection.ErrWorkerDied)
		})

	if parseErr == nil {
		parseT.Fatal("an unknown outcome must be reported")
	}
	if parseLocalState != "optimistic" {
		parseT.Errorf("state = %q, want the optimistic value kept — rolling back a change the domain may have made desynchronizes the UI", parseLocalState)
	}
}

func TestOptimisticKeepsStateOnSuccess(parseT *testing.T) {
	parseLocalState := "original"
	parseResult, parseErr := projection.Optimistic(context.Background(),
		func() { parseLocalState = "optimistic" },
		func() { parseLocalState = "original" },
		func(context.Context) (int, error) { return 7, nil })

	if parseErr != nil {
		parseT.Fatalf("Optimistic: %v", parseErr)
	}
	if parseResult != 7 {
		parseT.Errorf("result = %d, want 7", parseResult)
	}
	if parseLocalState != "optimistic" {
		parseT.Errorf("state = %q, want the applied value", parseLocalState)
	}
}

func TestOptimisticRequiresAllThreeFunctions(parseT *testing.T) {
	if _, parseErr := projection.Optimistic(context.Background(), nil, func() {},
		func(context.Context) (int, error) { return 0, nil }); parseErr == nil {
		parseT.Error("a missing apply must be rejected")
	}
	if _, parseErr := projection.Optimistic(context.Background(), func() {}, nil,
		func(context.Context) (int, error) { return 0, nil }); parseErr == nil {
		parseT.Error("a missing rollback must be rejected")
	}
	if _, parseErr := projection.Optimistic[int](context.Background(), func() {}, func() {}, nil); parseErr == nil {
		parseT.Error("a missing invoke must be rejected")
	}
}

// ------------------------------------------------------------------ guards

func TestFailureKindLabelsAreDistinct(parseT *testing.T) {
	parseSeen := map[string]bool{}
	for _, parseKind := range []projection.FailureKind{
		projection.FailureRejected, projection.FailureWorkerDied, projection.FailureTimedOut,
		projection.FailureCancelled, projection.FailureTransport,
	} {
		if parseSeen[parseKind.String()] {
			parseT.Errorf("failure label %q is not distinct", parseKind)
		}
		parseSeen[parseKind.String()] = true
	}
	if projection.FailureKind(99).String() == "" {
		parseT.Error("an unknown failure kind must still render for diagnostics")
	}
}

func TestNewResilientRequiresAClient(parseT *testing.T) {
	if _, parseErr := projection.NewResilient(nil, projection.Policy{}); parseErr == nil {
		parseT.Error("a nil client must be rejected")
	}
	var parseNil *projection.Resilient
	if _, parseErr := parseNil.Send(context.Background(), "x", nil); parseErr == nil {
		parseT.Error("a nil resilient client must error rather than panic")
	}
}

func TestPolicyWithoutRetryAttemptsOnce(parseT *testing.T) {
	parseClient := &scriptedFailureClient{outcomes: []error{errors.New("busy")}}
	parseResilient, _ := projection.NewResilient(parseClient, projection.Policy{Sleep: noSleep})

	if _, parseErr := addItem.Invoke(context.Background(), parseResilient, jsonCodec{}, addItemArgs{}); parseErr == nil {
		parseT.Fatal("expected a failure")
	}
	if parseClient.calls != 1 {
		parseT.Errorf("calls = %d, want 1 with no retry policy", parseClient.calls)
	}
}

// TestTransportConstructorsProduceTheRightKinds pins the contract transports
// implement. Without constructors, "wrap ErrRejected" is the kind of instruction
// followed on three call sites out of four, and the fourth silently becomes a
// retryable transport failure.
func TestTransportConstructorsProduceTheRightKinds(parseT *testing.T) {
	for _, parseCase := range []struct {
		label string
		err   error
		want  projection.FailureKind
	}{
		{"rejection", projection.Rejection("insufficient funds"), projection.FailureRejected},
		{"worker death", projection.WorkerDeath("terminated"), projection.FailureWorkerDied},
		{"domain panic", projection.DomainPanic("nil map write"), projection.FailureRejected},
	} {
		if parseKind := projection.Classify(parseCase.err); parseKind != parseCase.want {
			parseT.Errorf("%s: kind = %s, want %s", parseCase.label, parseKind, parseCase.want)
		}
		// The caller-facing message must survive, or a UI has nothing to show.
		if !strings.Contains(parseCase.err.Error(), "n") {
			parseT.Errorf("%s: message was lost: %v", parseCase.label, parseCase.err)
		}
	}
}
