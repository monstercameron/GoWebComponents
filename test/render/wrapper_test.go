package render

import (
	"context"
	"testing"
	"time"
)

func expectPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic")
		}
	}()
	fn()
}

func TestWithQueuedSchedulerReturnsOption(t *testing.T) {
	if option := WithQueuedScheduler(); option == nil {
		t.Fatalf("expected queued scheduler option")
	}
}

func TestNewResourceControllerWrapperResolves(t *testing.T) {
	controller := NewResourceController[string]()
	result := make(chan string, 1)

	go func() {
		value, err := controller.Await(context.Background())
		if err != nil {
			result <- "error"
			return
		}
		result <- value
	}()

	select {
	case <-controller.Started():
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for resource attempt start")
	}

	controller.Resolve("ready")
	select {
	case got := <-result:
		if got != "ready" {
			t.Fatalf("expected resolved value ready, got %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for resolved value")
	}
}

func TestNewPanicsWithNilTestingTB(t *testing.T) {
	expectPanic(t, func() {
		_ = New(nil)
	})
}

