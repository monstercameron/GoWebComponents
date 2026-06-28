//go:build !js || !wasm

package ui

import (
	"testing"
	"time"
)

// TestDeferTriggersDefaultFalseOnNative proves the trigger hooks return false on native/SSR
// (no effect lifecycle, no DOM) — so deferred content is absent from the server render and
// mounts in the browser once the trigger fires. They must not panic when called.
func TestDeferTriggersDefaultFalseOnNative(parseT *testing.T) {
	if UseTimerTrigger(10 * time.Millisecond) {
		parseT.Fatal("UseTimerTrigger should be false on native/SSR")
	}
	if UseIdle() {
		parseT.Fatal("UseIdle should be false on native/SSR")
	}
	if UseInteraction(DOMRef{}) {
		parseT.Fatal("UseInteraction should be false on native/SSR")
	}
}

// TestDeferTriggersFeedUseDefer proves the triggers compose with the UseDefer latch: false
// triggers keep it deferred.
func TestDeferTriggersFeedUseDefer(parseT *testing.T) {
	if UseDefer(UseTimerTrigger(time.Hour)) {
		parseT.Fatal("an unfired timer trigger should keep the view deferred")
	}
}
