//go:build js && wasm

package router

import (
	"errors"
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// TestEvaluateNavigationWasmAllowsGuardedRoute verifies the guard-attempt wrapper returns normalized targets for allowed routes.
func TestEvaluateNavigationWasmAllowsGuardedRoute(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHistoryRouter(RouterOptions{DefaultRoute: "/"})
	js.Global().Get("location").Set("pathname", "/")

	parseRouter.GoRegisterRoute("/", func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("home"))
	})
	parseRouter.GoRegisterRoute("/docs", func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("docs"))
	}, Options{
		BeforeEnter: func(parseCtx RouteContext) GuardResult {
			return AllowNavigation()
		},
	})

	parseTarget, isParseAllowed := parseRouter.evaluateNavigation("/docs")
	if !isParseAllowed {
		parseT.Fatal("expected evaluateNavigation to allow the guarded route")
	}
	if parseTarget != "/docs" {
		parseT.Fatalf("expected normalized navigation target /docs, got %q", parseTarget)
	}
	if parseRouter.guardState.active != 0 || parseRouter.guardState.cancel != nil {
		parseT.Fatalf("expected evaluateNavigation to finish its guard attempt, got active=%d cancel=%v", parseRouter.guardState.active, parseRouter.guardState.cancel)
	}
}

// TestRenderRouteStateFallbacksWasm verifies the default loader-error and guard-state renderers expose actionable fallback text.
func TestRenderRouteStateFallbacksWasm(parseT *testing.T) {
	parseErrorElem := renderRouteError(nil, errors.New("loader boom"), nil)
	if parseGot := collectElementText(parseErrorElem); parseGot != "loader boom" {
		parseT.Fatalf("expected loader error text fallback, got %q", parseGot)
	}

	parseDefaultErrorElem := renderRouteError(nil, nil, nil)
	if parseGot2 := collectElementText(parseDefaultErrorElem); parseGot2 != "Route load failed" {
		parseT.Fatalf("expected default route error text, got %q", parseGot2)
	}

	parseGuardElem := renderRouteGuardState(nil, Attrs{"reason": "Sign in required"})
	if parseGot3 := collectElementText(parseGuardElem); parseGot3 != "Sign in required" {
		parseT.Fatalf("expected explicit guard reason, got %q", parseGot3)
	}

	parseDefaultGuardElem := renderRouteGuardState(nil, Attrs{})
	if parseGot4 := collectElementText(parseDefaultGuardElem); parseGot4 != navigationBlocked {
		parseT.Fatalf("expected default guard-blocked text %q, got %q", navigationBlocked, parseGot4)
	}
}
