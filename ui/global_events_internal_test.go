//go:build !(js && wasm)

package ui

import "testing"

func TestGlobalEventDepsAreStableWhenNoneGiven(t *testing.T) {
	// No caller deps => a non-empty, stable dep set so the listener binds once on
	// mount (empty deps would mean "every render" in UseEffect → a leaked listener
	// per render).
	parseDeps := globalEventDeps(scopeDocument, "keydown", nil)
	if len(parseDeps) == 0 {
		t.Fatal("globalEventDeps must be non-empty when no deps are given")
	}
	parseAgain := globalEventDeps(scopeDocument, "keydown", nil)
	if len(parseAgain) != len(parseDeps) {
		t.Fatal("dep set length must be stable across calls")
	}
	for parseIndex := range parseDeps {
		if parseDeps[parseIndex] != parseAgain[parseIndex] {
			t.Fatalf("dep %d not stable: %v vs %v", parseIndex, parseDeps[parseIndex], parseAgain[parseIndex])
		}
	}
	// Distinct scope/event must produce distinct dep sets (so two hooks in one
	// component don't collide on a shared effect identity).
	parseWindowDeps := globalEventDeps(scopeWindow, "keydown", nil)
	if parseWindowDeps[1] == parseDeps[1] {
		t.Fatal("document and window scopes must differ in the dep set")
	}
}

func TestGlobalEventDepsPassThroughExplicitDeps(t *testing.T) {
	parseCallerDeps := []any{"a", 7}
	parseOut := globalEventDeps(scopeDocument, "keydown", parseCallerDeps)
	if len(parseOut) != 2 || parseOut[0] != "a" || parseOut[1] != 7 {
		t.Fatalf("explicit deps must pass through unchanged, got %v", parseOut)
	}
}

func TestNativeBindGlobalEventIsNoOp(t *testing.T) {
	// On native there is no DOM: binding returns nil (no cleanup) and never panics.
	if parseUnbind := defaultBindGlobalEvent(scopeDocument, "keydown", func(Event) {}); parseUnbind != nil {
		t.Fatal("native bindGlobalEvent should return nil")
	}
	if parseUnbind := defaultBindGlobalEvent(scopeWindow, "resize", nil); parseUnbind != nil {
		t.Fatal("native bindGlobalEvent should return nil even with a nil handler")
	}
}

func TestBindGlobalEventSeamIsSwappable(t *testing.T) {
	// The binder is a package var so tests/e2e can substitute behavior, and so the
	// unbind contract (returned func is the cleanup) is exercised on native.
	parsePrev := bindGlobalEvent
	defer func() { bindGlobalEvent = parsePrev }()

	var parseBound, parseUnbound int
	var parseGotScope globalScope
	var parseGotEvent string
	bindGlobalEvent = func(parseScope globalScope, parseEvent string, parseHandler func(Event)) func() {
		parseBound++
		parseGotScope = parseScope
		parseGotEvent = parseEvent
		return func() { parseUnbound++ }
	}

	parseUnbind := bindGlobalEvent(scopeWindow, "resize", func(Event) {})
	if parseBound != 1 || parseGotScope != scopeWindow || parseGotEvent != "resize" {
		t.Fatalf("seam did not record bind: bound=%d scope=%d event=%q", parseBound, parseGotScope, parseGotEvent)
	}
	parseUnbind()
	if parseUnbound != 1 {
		t.Fatalf("unbind contract not honored: unbound=%d", parseUnbound)
	}
}
