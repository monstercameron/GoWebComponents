//go:build js && wasm

package router

import (
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

// TestPopstateLeaveGuardCancelsBlockedBackForward pins the #82 finding (1) fix: a
// browser back/forward changes the URL before firing popstate, so a BeforeLeave that
// blocks must restore the URL rather than let the navigation slip through. When the
// guard allows, back/forward proceeds normally.
func TestPopstateLeaveGuardCancelsBlockedBackForward(parseT *testing.T) {
	installRouterBrowserEnv(parseT)

	parseRouter := NewHistoryRouter(RouterOptions{})
	parseT.Cleanup(func() { parseRouter.teardownHistoryListener() })

	parseComponent := func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("x"))
	}
	parseBlockLeaveB := false
	parseRouter.Register("/a", parseComponent)
	parseRouter.Register("/b", parseComponent, Options{
		BeforeLeave: func(parseFrom RouteContext, parseTo RouteContext) GuardResult {
			if parseBlockLeaveB {
				return BlockNavigation("stay on b")
			}
			return AllowNavigation()
		},
	})

	parseRouter.Navigate("/a")
	parseRouter.Navigate("/b")
	if parseRouter.lastCommittedPath != "/b" {
		parseT.Fatalf("expected committed path /b, got %q", parseRouter.lastCommittedPath)
	}

	parsePathname := func() string {
		return js.Global().Get("location").Get("pathname").String()
	}

	// Blocked back: pressing back from /b must be cancelled and the URL restored.
	parseBlockLeaveB = true
	js.Global().Get("history").Call("back")
	if parseGot := parsePathname(); parseGot != "/b" {
		parseT.Fatalf("blocked BeforeLeave must cancel back/forward and restore the URL, got %q", parseGot)
	}
	if parseRouter.lastCommittedPath != "/b" {
		parseT.Fatalf("committed path must stay /b after a cancelled back, got %q", parseRouter.lastCommittedPath)
	}

	// Allowed back: with the guard disabled, back from /b proceeds to /a.
	parseBlockLeaveB = false
	js.Global().Get("history").Call("back")
	if parseGot := parsePathname(); parseGot != "/a" {
		parseT.Fatalf("allowed BeforeLeave must let back/forward proceed to /a, got %q", parseGot)
	}
	if parseRouter.lastCommittedPath != "/a" {
		parseT.Fatalf("committed path must advance to /a after an allowed back, got %q", parseRouter.lastCommittedPath)
	}
}
