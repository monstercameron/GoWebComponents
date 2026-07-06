//go:build js && wasm

package router

import "testing"

// TestHistoryRouterDetachesListenersWhenReplaced pins the #82 listener-leak fix: a
// history router attaches popstate + hashchange listeners in setupHistoryListener,
// and when it is superseded by another router (setGlobalRouter) those listeners are
// detached and freed rather than left attached to window forever.
func TestHistoryRouterDetachesListenersWhenReplaced(parseT *testing.T) {
	installRouterBrowserEnv(parseT)

	parseFirst := NewHistoryRouter()
	if !parseFirst.historyListenersAttached {
		parseT.Fatal("a fresh history router must attach its history listeners")
	}
	setGlobalRouter(parseFirst)

	parseSecond := NewHistoryRouter()
	setGlobalRouter(parseSecond)

	// The superseded router must have been torn down: not attached, handlers cleared.
	if parseFirst.historyListenersAttached {
		parseT.Fatal("the replaced router's listeners must be detached")
	}
	if parseFirst.popstateHandler.Truthy() || parseFirst.hashchangeHandler.Truthy() {
		parseT.Fatal("the replaced router's handlers must be released and cleared")
	}
	if !parseFirst.disposed {
		parseT.Fatal("the replaced router must be marked disposed")
	}
	// The live router keeps its listeners.
	if !parseSecond.historyListenersAttached {
		parseT.Fatal("the live router must keep its history listeners attached")
	}

	// teardown is idempotent — a second call must not double-release/panic.
	parseFirst.teardownHistoryListener()

	// Reset global state for other tests.
	setGlobalRouter(nil)
	parseSecond.teardownHistoryListener()
}
