package ui

import "github.com/monstercameron/GoWebComponents/v5/internal/runtime"

// OnReady registers fn to run once, right after the first render commits (the
// initial UI is on screen and its effects have run). If the app has already
// rendered, fn runs immediately. Use it to drop a loading splash, kick off
// post-mount work, or signal readiness — instead of guessing with timeouts.
//
//	ui.OnReady(func() { /* hide splash, start telemetry, … */ })
//
// On wasm the runtime additionally dispatches a "gwc:ready" event on document, so
// a plain host page can drop its splash without any Go glue:
//
//	document.addEventListener("gwc:ready", () => splash.remove())
func OnReady(parseFn func()) {
	runtime.OnFirstCommit(parseFn)
}
