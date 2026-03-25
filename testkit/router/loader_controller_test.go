//go:build js && wasm
// +build js,wasm

package routertest

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"

	appRouter "github.com/monstercameron/GoWebComponents/router"
	baseRender "github.com/monstercameron/GoWebComponents/testkit/render"
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

	t.Run("failure helpers", func(t *testing.T) {
		controller := NewLoaderController()

		waitLoaderError := func(buildReject func()) error {
			done := make(chan error, 1)
			go func() {
				_, err := controller.Loader()(context.Background(), appRouter.RouteContext{})
				done <- err
			}()
			<-controller.Started()
			buildReject()
			return <-done
		}

		if err := waitLoaderError(func() { controller.RejectLoaderFailure("/orders", "timeout") }); err == nil || !strings.Contains(err.Error(), baseRender.FailureCodeLoaderFailure) {
			t.Fatalf("expected loader failure helper to set typed code, got %v", err)
		}
		if err := waitLoaderError(func() { controller.RejectRouteGuardFailure("/billing", "denied") }); err == nil || !strings.Contains(err.Error(), baseRender.FailureCodeRouteGuardFailure) {
			t.Fatalf("expected route guard failure helper to set typed code, got %v", err)
		}
		if err := waitLoaderError(func() { controller.RejectCacheConflict("cart:42") }); err == nil || !strings.Contains(err.Error(), baseRender.FailureCodeCacheConflict) {
			t.Fatalf("expected cache conflict helper to set typed code, got %v", err)
		}
		if err := waitLoaderError(func() { controller.RejectOfflineReplay("mut-7", "offline") }); err == nil || !strings.Contains(err.Error(), baseRender.FailureCodeOfflineReplay) {
			t.Fatalf("expected offline replay helper to set typed code, got %v", err)
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

func TestGuardFailureBuilders(t *testing.T) {
	blocked := BuildGuardBlocked("policy denied")(appRouter.RouteContext{Path: "/billing"})
	if !blocked.Blocked || blocked.Reason != "policy denied" {
		t.Fatalf("expected blocked guard result with reason, got %+v", blocked)
	}

	redirected := BuildGuardRedirect("/login")(appRouter.RouteContext{Path: "/billing"})
	if redirected.Redirect != "/login" {
		t.Fatalf("expected redirect guard result /login, got %+v", redirected)
	}

	denied := BuildAsyncGuardDenied("reauth required")(context.Background(), appRouter.RouteContext{Path: "/billing"})
	if !denied.Blocked || !denied.Denied || denied.Reason != "reauth required" {
		t.Fatalf("expected denied async guard decision, got %+v", denied)
	}
}
