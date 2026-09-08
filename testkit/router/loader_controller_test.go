//go:build js && wasm

package routertest

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"

	appRouter "github.com/monstercameron/GoWebComponents/v6/router"
	baseRender "github.com/monstercameron/GoWebComponents/v6/testkit/render"
)

func TestLoaderControllerResolveRejectCancelAndNilSafety(parseT *testing.T) {
	parseT.Run("resolve", func(parseT2 *testing.T) {
		parseController := NewLoaderController()
		type result struct {
			value appRouter.Attrs
			err   error
		}
		parseDone := make(chan result, 1)
		go func() {
			parseValue, parseErr := parseController.Loader()(context.Background(), appRouter.RouteContext{})
			parseDone <- result{value: parseValue, err: parseErr}
		}()

		select {
		case parseAttempt := <-parseController.Started():
			if parseAttempt != 1 {
				parseT2.Fatalf("expected first attempt index 1, got %d", parseAttempt)
			}
		case <-context.Background().Done():
		}
		if !parseController.Pending() {
			parseT2.Fatal("expected controller to report a pending attempt")
		}
		if parseController.AttemptCount() != 1 {
			parseT2.Fatalf("expected one attempt, got %d", parseController.AttemptCount())
		}

		parsePayload := appRouter.Attrs{"status": "ready"}
		parseController.Resolve(parsePayload)
		parseGot := <-parseDone
		if parseGot.err != nil {
			parseT2.Fatalf("expected resolve to succeed, got %v", parseGot.err)
		}
		if parseGot.value["status"] != "ready" {
			parseT2.Fatalf("unexpected resolved payload: %#v", parseGot.value)
		}
		if parseController.Pending() {
			parseT2.Fatal("expected no pending attempt after resolve")
		}
		parseAttempts := parseController.Attempts()
		if len(parseAttempts) != 1 || parseAttempts[0].Index != 1 || parseAttempts[0].Cancelled {
			parseT2.Fatalf("unexpected attempt snapshot after resolve: %+v", parseAttempts)
		}
	})

	parseT.Run("reject", func(parseT3 *testing.T) {
		parseController2 := NewLoaderController()
		parseDone2 := make(chan error, 1)
		parseWant := errors.New("boom")
		go func() {
			_, parseErr2 := parseController2.Loader()(context.Background(), appRouter.RouteContext{})
			parseDone2 <- parseErr2
		}()
		<-parseController2.Started()
		parseController2.Reject(parseWant)
		if parseErr3 := <-parseDone2; !errors.Is(parseErr3, parseWant) {
			parseT3.Fatalf("expected reject error %v, got %v", parseWant, parseErr3)
		}
	})

	parseT.Run("cancel", func(parseT4 *testing.T) {
		parseController3 := NewLoaderController()
		parseDone3 := make(chan error, 1)
		go func() {
			_, parseErr4 := parseController3.Loader()(context.Background(), appRouter.RouteContext{})
			parseDone3 <- parseErr4
		}()
		<-parseController3.Started()
		parseController3.Cancel()
		if parseErr5 := <-parseDone3; !errors.Is(parseErr5, context.Canceled) {
			parseT4.Fatalf("expected cancel error, got %v", parseErr5)
		}
		parseAttempts2 := parseController3.Attempts()
		if len(parseAttempts2) != 1 || !parseAttempts2[0].Cancelled {
			parseT4.Fatalf("expected cancelled attempt snapshot, got %+v", parseAttempts2)
		}
	})

	parseT.Run("failure helpers", func(parseT5 *testing.T) {
		parseController4 := NewLoaderController()

		parseWaitLoaderError := func(buildReject func()) error {
			parseDone4 := make(chan error, 1)
			go func() {
				_, parseErr6 := parseController4.Loader()(context.Background(), appRouter.RouteContext{})
				parseDone4 <- parseErr6
			}()
			<-parseController4.Started()
			buildReject()
			return <-parseDone4
		}

		if parseErr7 := parseWaitLoaderError(func() { parseController4.RejectLoaderFailure("/orders", "timeout") }); parseErr7 == nil || !strings.Contains(parseErr7.Error(), baseRender.FailureCodeLoaderFailure) {
			parseT5.Fatalf("expected loader failure helper to set typed code, got %v", parseErr7)
		}
		if parseErr8 := parseWaitLoaderError(func() { parseController4.RejectRouteGuardFailure("/billing", "denied") }); parseErr8 == nil || !strings.Contains(parseErr8.Error(), baseRender.FailureCodeRouteGuardFailure) {
			parseT5.Fatalf("expected route guard failure helper to set typed code, got %v", parseErr8)
		}
		if parseErr9 := parseWaitLoaderError(func() { parseController4.RejectCacheConflict("cart:42") }); parseErr9 == nil || !strings.Contains(parseErr9.Error(), baseRender.FailureCodeCacheConflict) {
			parseT5.Fatalf("expected cache conflict helper to set typed code, got %v", parseErr9)
		}
		if parseErr10 := parseWaitLoaderError(func() { parseController4.RejectOfflineReplay("mut-7", "offline") }); parseErr10 == nil || !strings.Contains(parseErr10.Error(), baseRender.FailureCodeOfflineReplay) {
			parseT5.Fatalf("expected offline replay helper to set typed code, got %v", parseErr10)
		}
	})

	parseT.Run("nil safety", func(parseT6 *testing.T) {
		var parseController5 *LoaderController
		if parseController5.Pending() {
			parseT6.Fatal("nil controller should not report pending work")
		}
		if parseController5.AttemptCount() != 0 {
			parseT6.Fatalf("nil controller attempt count = %d, want 0", parseController5.AttemptCount())
		}
		if parseAttempts3 := parseController5.Attempts(); parseAttempts3 != nil {
			parseT6.Fatalf("nil controller attempts = %+v, want nil", parseAttempts3)
		}
		if parseStarted := parseController5.Started(); parseStarted != nil {
			parseT6.Fatalf("nil controller started channel = %v, want nil", parseStarted)
		}
		parseController5.Resolve(appRouter.Attrs{"ignored": true})
		parseController5.Reject(errors.New("ignored"))
		parseController5.Cancel()
	})
}

func TestCloneURLValuesAndCloneStringMap(parseT *testing.T) {
	if parseCloned := cloneURLValues(nil); len(parseCloned) != 0 {
		parseT.Fatalf("cloneURLValues(nil) = %#v, want empty values", parseCloned)
	}
	if parseCloned2 := cloneStringMap(nil); len(parseCloned2) != 0 {
		parseT.Fatalf("cloneStringMap(nil) = %#v, want empty map", parseCloned2)
	}

	parseQuery := url.Values{"filter": {"active", "recent"}}
	parseClonedQuery := cloneURLValues(parseQuery)
	parseQuery.Set("filter", "mutated")
	if parseGot := parseClonedQuery["filter"]; len(parseGot) != 2 || parseGot[0] != "active" || parseGot[1] != "recent" {
		parseT.Fatalf("cloneURLValues copied values incorrectly: %#v", parseClonedQuery)
	}

	parseParams := map[string]string{"id": "42", "tab": "history"}
	parseClonedParams := cloneStringMap(parseParams)
	parseParams["id"] = "99"
	if parseClonedParams["id"] != "42" || parseClonedParams["tab"] != "history" {
		parseT.Fatalf("cloneStringMap copied values incorrectly: %#v", parseClonedParams)
	}
}

func TestGuardFailureBuilders(parseT *testing.T) {
	parseBlocked := BuildGuardBlocked("policy denied")(appRouter.RouteContext{Path: "/billing"})
	if !parseBlocked.Blocked || parseBlocked.Reason != "policy denied" {
		parseT.Fatalf("expected blocked guard result with reason, got %+v", parseBlocked)
	}

	parseRedirected := BuildGuardRedirect("/login")(appRouter.RouteContext{Path: "/billing"})
	if parseRedirected.Redirect != "/login" {
		parseT.Fatalf("expected redirect guard result /login, got %+v", parseRedirected)
	}

	parseDenied := BuildAsyncGuardDenied("reauth required")(context.Background(), appRouter.RouteContext{Path: "/billing"})
	if !parseDenied.Blocked || !parseDenied.Denied || parseDenied.Reason != "reauth required" {
		parseT.Fatalf("expected denied async guard decision, got %+v", parseDenied)
	}
}
