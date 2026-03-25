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

func awaitAsync[T any](c *ResourceController[T], ctx context.Context) <-chan awaitResult[T] {
	ch := make(chan awaitResult[T], 1)
	go func() {
		value, err := c.Await(ctx)
		ch <- awaitResult[T]{value: value, err: err}
	}()
	return ch
}

func requireStartedIndex[T any](t *testing.T, c *ResourceController[T], want int) {
	t.Helper()
	select {
	case got := <-c.Started():
		if got != want {
			t.Fatalf("expected started index %d, got %d", want, got)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for started index %d", want)
	}
}

func TestResourceControllerResolveRejectCancelAndContextCancel(t *testing.T) {
	controller := NewResourceController[string]()

	resolveResult := awaitAsync(controller, context.Background())
	requireStartedIndex(t, controller, 1)
	if !controller.Pending() || controller.AttemptCount() != 1 {
		t.Fatalf("expected pending attempt 1, pending=%t attempts=%d", controller.Pending(), controller.AttemptCount())
	}
	controller.Resolve("ready")
	resolved := <-resolveResult
	if resolved.err != nil || resolved.value != "ready" {
		t.Fatalf("expected resolve result ready,nil got value=%q err=%v", resolved.value, resolved.err)
	}

	rejectResult := awaitAsync(controller, context.Background())
	requireStartedIndex(t, controller, 2)
	controller.Reject(errors.New("boom"))
	rejected := <-rejectResult
	if rejected.err == nil || rejected.err.Error() != "boom" {
		t.Fatalf("expected reject error boom, got %v", rejected.err)
	}

	cancelResult := awaitAsync(controller, context.Background())
	requireStartedIndex(t, controller, 3)
	controller.Cancel()
	cancelled := <-cancelResult
	if !errors.Is(cancelled.err, context.Canceled) {
		t.Fatalf("expected cancel error context.Canceled, got %v", cancelled.err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	contextCancelResult := awaitAsync(controller, ctx)
	requireStartedIndex(t, controller, 4)
	cancel()
	contextCancelled := <-contextCancelResult
	if !errors.Is(contextCancelled.err, context.Canceled) {
		t.Fatalf("expected context cancellation error, got %v", contextCancelled.err)
	}

	if controller.Pending() {
		t.Fatalf("expected no pending attempt after completions")
	}
	attempts := controller.Attempts()
	if len(attempts) != 4 {
		t.Fatalf("expected four attempts, got %d", len(attempts))
	}
	if !attempts[2].Cancelled || !attempts[3].Cancelled {
		t.Fatalf("expected cancel and context-cancel attempts to be marked cancelled, got %+v", attempts)
	}
}

func TestResourceControllerNilReceiverAndLoader(t *testing.T) {
	var controller *ResourceController[string]

	value, err := controller.Loader()(context.Background())
	if value != "" || err != nil {
		t.Fatalf("expected nil controller loader to return zero,nil got %q,%v", value, err)
	}

	controller.Resolve("ignored")
	controller.Reject(errors.New("ignored"))
	controller.Cancel()

	if controller.Pending() {
		t.Fatalf("expected nil controller Pending to be false")
	}
	if controller.AttemptCount() != 0 {
		t.Fatalf("expected nil controller AttemptCount to be zero")
	}
	if attempts := controller.Attempts(); attempts != nil {
		t.Fatalf("expected nil controller Attempts to be nil, got %+v", attempts)
	}
	if started := controller.Started(); started != nil {
		t.Fatalf("expected nil controller Started channel to be nil")
	}
}

func TestResourceControllerFailureInjectionHelpers(t *testing.T) {
	controller := NewResourceController[string]()

	cacheResult := awaitAsync(controller, context.Background())
	requireStartedIndex(t, controller, 1)
	controller.RejectCacheConflict("cart:42")
	if err := (<-cacheResult).err; err == nil || !strings.Contains(err.Error(), FailureCodeCacheConflict) {
		t.Fatalf("expected cache conflict failure code in error, got %v", err)
	}

	replayResult := awaitAsync(controller, context.Background())
	requireStartedIndex(t, controller, 2)
	controller.RejectOfflineReplay("mutation:17", "network down")
	if err := (<-replayResult).err; err == nil || !strings.Contains(err.Error(), FailureCodeOfflineReplay) {
		t.Fatalf("expected offline replay failure code in error, got %v", err)
	}

	if message := BuildHydrationMismatchError("/dashboard", "node mismatch").Error(); !strings.Contains(message, FailureCodeHydrationMismatch) || !strings.Contains(message, "path=/dashboard") {
		t.Fatalf("expected hydration mismatch helper to include code and path, got %q", message)
	}
	if message := BuildLoaderFailureError("/orders", "timeout").Error(); !strings.Contains(message, FailureCodeLoaderFailure) || !strings.Contains(message, "path=/orders") {
		t.Fatalf("expected loader failure helper to include code and path, got %q", message)
	}
	if message := BuildRouteGuardFailureError("/billing", "denied").Error(); !strings.Contains(message, FailureCodeRouteGuardFailure) || !strings.Contains(message, "path=/billing") {
		t.Fatalf("expected route guard failure helper to include code and path, got %q", message)
	}
	if message := BuildCacheConflictError("orders:1").Error(); !strings.Contains(message, FailureCodeCacheConflict) || !strings.Contains(message, "entity=orders:1") {
		t.Fatalf("expected cache conflict helper to include code and entity, got %q", message)
	}
	if message := BuildOfflineReplayError("mut-4", "queued").Error(); !strings.Contains(message, FailureCodeOfflineReplay) || !strings.Contains(message, "entity=mut-4") {
		t.Fatalf("expected offline replay helper to include code and entity, got %q", message)
	}
}
