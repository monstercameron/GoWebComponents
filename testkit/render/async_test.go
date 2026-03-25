package render

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type awaitResult[T any] struct {
	value T
	err   error
}

func awaitAsync[T any](parseC *ResourceController[T], parseCtx context.Context) <-chan awaitResult[T] {
	parseCh := make(chan awaitResult[T], 1)
	go func() {
		parseValue, parseErr := parseC.Await(parseCtx)
		parseCh <- awaitResult[T]{value: parseValue, err: parseErr}
	}()
	return parseCh
}

func requireStartedIndex[T any](parseT *testing.T, parseC *ResourceController[T], parseWant int) {
	parseT.Helper()
	select {
	case parseGot := <-parseC.Started():
		if parseGot != parseWant {
			parseT.Fatalf("expected started index %d, got %d", parseWant, parseGot)
		}
	case <-time.After(2 * time.Second):
		parseT.Fatalf("timed out waiting for started index %d", parseWant)
	}
}

func TestResourceControllerResolveRejectCancelAndContextCancel(parseT *testing.T) {
	parseController := NewResourceController[string]()

	parseResolveResult := awaitAsync(parseController, context.Background())
	requireStartedIndex(parseT, parseController, 1)
	if !parseController.Pending() || parseController.AttemptCount() != 1 {
		parseT.Fatalf("expected pending attempt 1, pending=%t attempts=%d", parseController.Pending(), parseController.AttemptCount())
	}
	parseController.Resolve("ready")
	parseResolved := <-parseResolveResult
	if parseResolved.err != nil || parseResolved.value != "ready" {
		parseT.Fatalf("expected resolve result ready,nil got value=%q err=%v", parseResolved.value, parseResolved.err)
	}

	parseRejectResult := awaitAsync(parseController, context.Background())
	requireStartedIndex(parseT, parseController, 2)
	parseController.Reject(errors.New("boom"))
	parseRejected := <-parseRejectResult
	if parseRejected.err == nil || parseRejected.err.Error() != "boom" {
		parseT.Fatalf("expected reject error boom, got %v", parseRejected.err)
	}

	parseCancelResult := awaitAsync(parseController, context.Background())
	requireStartedIndex(parseT, parseController, 3)
	parseController.Cancel()
	parseCancelled := <-parseCancelResult
	if !errors.Is(parseCancelled.err, context.Canceled) {
		parseT.Fatalf("expected cancel error context.Canceled, got %v", parseCancelled.err)
	}

	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseContextCancelResult := awaitAsync(parseController, parseCtx)
	requireStartedIndex(parseT, parseController, 4)
	parseCancel()
	parseContextCancelled := <-parseContextCancelResult
	if !errors.Is(parseContextCancelled.err, context.Canceled) {
		parseT.Fatalf("expected context cancellation error, got %v", parseContextCancelled.err)
	}

	if parseController.Pending() {
		parseT.Fatalf("expected no pending attempt after completions")
	}
	parseAttempts := parseController.Attempts()
	if len(parseAttempts) != 4 {
		parseT.Fatalf("expected four attempts, got %d", len(parseAttempts))
	}
	if !parseAttempts[2].Cancelled || !parseAttempts[3].Cancelled {
		parseT.Fatalf("expected cancel and context-cancel attempts to be marked cancelled, got %+v", parseAttempts)
	}
}

func TestResourceControllerNilReceiverAndLoader(parseT *testing.T) {
	var parseController *ResourceController[string]

	parseValue, parseErr := parseController.Loader()(context.Background())
	if parseValue != "" || parseErr != nil {
		parseT.Fatalf("expected nil controller loader to return zero,nil got %q,%v", parseValue, parseErr)
	}

	parseController.Resolve("ignored")
	parseController.Reject(errors.New("ignored"))
	parseController.Cancel()

	if parseController.Pending() {
		parseT.Fatalf("expected nil controller Pending to be false")
	}
	if parseController.AttemptCount() != 0 {
		parseT.Fatalf("expected nil controller AttemptCount to be zero")
	}
	if parseAttempts := parseController.Attempts(); parseAttempts != nil {
		parseT.Fatalf("expected nil controller Attempts to be nil, got %+v", parseAttempts)
	}
	if parseStarted := parseController.Started(); parseStarted != nil {
		parseT.Fatalf("expected nil controller Started channel to be nil")
	}
}

func TestResourceControllerFailureInjectionHelpers(parseT *testing.T) {
	parseController := NewResourceController[string]()

	cacheResult := awaitAsync(parseController, context.Background())
	requireStartedIndex(parseT, parseController, 1)
	parseController.RejectCacheConflict("cart:42")
	if parseErr := (<-cacheResult).err; parseErr == nil || !strings.Contains(parseErr.Error(), FailureCodeCacheConflict) {
		parseT.Fatalf("expected cache conflict failure code in error, got %v", parseErr)
	}

	parseReplayResult := awaitAsync(parseController, context.Background())
	requireStartedIndex(parseT, parseController, 2)
	parseController.RejectOfflineReplay("mutation:17", "network down")
	if parseErr2 := (<-parseReplayResult).err; parseErr2 == nil || !strings.Contains(parseErr2.Error(), FailureCodeOfflineReplay) {
		parseT.Fatalf("expected offline replay failure code in error, got %v", parseErr2)
	}

	if parseMessage := BuildHydrationMismatchError("/dashboard", "node mismatch").Error(); !strings.Contains(parseMessage, FailureCodeHydrationMismatch) || !strings.Contains(parseMessage, "path=/dashboard") {
		parseT.Fatalf("expected hydration mismatch helper to include code and path, got %q", parseMessage)
	}
	if parseMessage2 := BuildLoaderFailureError("/orders", "timeout").Error(); !strings.Contains(parseMessage2, FailureCodeLoaderFailure) || !strings.Contains(parseMessage2, "path=/orders") {
		parseT.Fatalf("expected loader failure helper to include code and path, got %q", parseMessage2)
	}
	if parseMessage3 := BuildRouteGuardFailureError("/billing", "denied").Error(); !strings.Contains(parseMessage3, FailureCodeRouteGuardFailure) || !strings.Contains(parseMessage3, "path=/billing") {
		parseT.Fatalf("expected route guard failure helper to include code and path, got %q", parseMessage3)
	}
	if parseMessage4 := BuildCacheConflictError("orders:1").Error(); !strings.Contains(parseMessage4, FailureCodeCacheConflict) || !strings.Contains(parseMessage4, "entity=orders:1") {
		parseT.Fatalf("expected cache conflict helper to include code and entity, got %q", parseMessage4)
	}
	if parseMessage5 := BuildOfflineReplayError("mut-4", "queued").Error(); !strings.Contains(parseMessage5, FailureCodeOfflineReplay) || !strings.Contains(parseMessage5, "entity=mut-4") {
		parseT.Fatalf("expected offline replay helper to include code and entity, got %q", parseMessage5)
	}
}
