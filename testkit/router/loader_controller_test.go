//go:build js && wasm
// +build js,wasm

package routertest

import (
	"context"
	"errors"
	"net/url"
	"testing"

	appRouter "github.com/monstercameron/GoWebComponents/router"
)

func TestLoaderControllerResolveRejectCancelAndNilSafety(t *testing.T) {
	t.Run("resolve", func(t *testing.T) {
		controller := NewLoaderController()
		type result struct {
			value appRouter.Attrs
			err   error
		}
		done := make(chan result, 1)
		go func() {
			value, err := controller.Loader()(context.Background(), appRouter.RouteContext{})
			done <- result{value: value, err: err}
		}()

		select {
		case attempt := <-controller.Started():
			if attempt != 1 {
				t.Fatalf("expected first attempt index 1, got %d", attempt)
			}
		case <-context.Background().Done():
		}
		if !controller.Pending() {
			t.Fatal("expected controller to report a pending attempt")
		}
		if controller.AttemptCount() != 1 {
			t.Fatalf("expected one attempt, got %d", controller.AttemptCount())
		}

		payload := appRouter.Attrs{"status": "ready"}
		controller.Resolve(payload)
		got := <-done
		if got.err != nil {
			t.Fatalf("expected resolve to succeed, got %v", got.err)
		}
		if got.value["status"] != "ready" {
			t.Fatalf("unexpected resolved payload: %#v", got.value)
		}
		if controller.Pending() {
			t.Fatal("expected no pending attempt after resolve")
		}
		attempts := controller.Attempts()
		if len(attempts) != 1 || attempts[0].Index != 1 || attempts[0].Cancelled {
			t.Fatalf("unexpected attempt snapshot after resolve: %+v", attempts)
		}
	})

	t.Run("reject", func(t *testing.T) {
		controller := NewLoaderController()
		done := make(chan error, 1)
		want := errors.New("boom")
		go func() {
			_, err := controller.Loader()(context.Background(), appRouter.RouteContext{})
			done <- err
		}()
		<-controller.Started()
		controller.Reject(want)
		if err := <-done; !errors.Is(err, want) {
			t.Fatalf("expected reject error %v, got %v", want, err)
		}
	})

	t.Run("cancel", func(t *testing.T) {
		controller := NewLoaderController()
		done := make(chan error, 1)
		go func() {
			_, err := controller.Loader()(context.Background(), appRouter.RouteContext{})
			done <- err
		}()
		<-controller.Started()
		controller.Cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancel error, got %v", err)
		}
		attempts := controller.Attempts()
		if len(attempts) != 1 || !attempts[0].Cancelled {
			t.Fatalf("expected cancelled attempt snapshot, got %+v", attempts)
		}
	})

	t.Run("nil safety", func(t *testing.T) {
		var controller *LoaderController
		if controller.Pending() {
			t.Fatal("nil controller should not report pending work")
		}
		if controller.AttemptCount() != 0 {
			t.Fatalf("nil controller attempt count = %d, want 0", controller.AttemptCount())
		}
		if attempts := controller.Attempts(); attempts != nil {
			t.Fatalf("nil controller attempts = %+v, want nil", attempts)
		}
		if started := controller.Started(); started != nil {
			t.Fatalf("nil controller started channel = %v, want nil", started)
		}
		controller.Resolve(appRouter.Attrs{"ignored": true})
		controller.Reject(errors.New("ignored"))
		controller.Cancel()
	})
}

func TestCloneURLValuesAndCloneStringMap(t *testing.T) {
	if cloned := cloneURLValues(nil); len(cloned) != 0 {
		t.Fatalf("cloneURLValues(nil) = %#v, want empty values", cloned)
	}
	if cloned := cloneStringMap(nil); len(cloned) != 0 {
		t.Fatalf("cloneStringMap(nil) = %#v, want empty map", cloned)
	}

	query := url.Values{"filter": {"active", "recent"}}
	clonedQuery := cloneURLValues(query)
	query.Set("filter", "mutated")
	if got := clonedQuery["filter"]; len(got) != 2 || got[0] != "active" || got[1] != "recent" {
		t.Fatalf("cloneURLValues copied values incorrectly: %#v", clonedQuery)
	}

	params := map[string]string{"id": "42", "tab": "history"}
	clonedParams := cloneStringMap(params)
	params["id"] = "99"
	if clonedParams["id"] != "42" || clonedParams["tab"] != "history" {
		t.Fatalf("cloneStringMap copied values incorrectly: %#v", clonedParams)
	}
}
