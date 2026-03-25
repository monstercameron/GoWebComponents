package render

import (
	"context"
	"errors"
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

