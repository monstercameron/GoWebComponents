//go:build js && wasm

package ui

import "testing"

// TestBindGlobalEventNoOpsWithoutAddEventListener proves the global-event binder
// degrades to a no-op (returns nil, no thrown/contained panic, no orphaned
// js.Func) when the global target has no addEventListener — e.g. a no-DOM-event
// host. Real browsers and Web Workers always have it.
func TestBindGlobalEventNoOpsWithoutAddEventListener(parseT *testing.T) {
	if parseUnbind := defaultBindGlobalEvent(scopeWindow, "online", func(Event) {}); parseUnbind != nil {
		parseT.Fatal("expected nil unbind when addEventListener is absent")
	}
}
