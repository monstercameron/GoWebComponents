//go:build js && wasm

package pwa

import (
	"context"
	"syscall/js"
	"testing"
)

// TestSubscribeLifecycleAttachesOncePerWorkerObject pins the statechange wiring of
// SubscribeLifecycle: each worker OBJECT gets exactly one listener (a worker that moves through
// installing -> installed -> activated must not notify twice per transition), and a NEW worker
// object that has the same script URL and state as an earlier one (every update of the same
// /sw.js) must still be observed.
func TestSubscribeLifecycleAttachesOncePerWorkerObject(parseT *testing.T) {
	parseRestore := installMockServiceWorkerEnvironment(parseT, false)
	defer parseRestore()

	parseRegistration, parseErr := RegisterServiceWorker(context.Background(), ServiceWorkerOptions{URL: "/sw.js"})
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseCalls := 0
	parseSub, parseErr := parseRegistration.SubscribeLifecycle(func(ServiceWorkerSnapshot) { parseCalls++ })
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	defer parseSub.Cancel()

	parseRaw := js.Global().Get("__pwaTestServiceWorkerRegistration")
	parseNewWorker := func(parseState string) js.Value {
		parseWorker := js.Global().Get("EventTarget").New()
		// A URL the mock environment's waiting worker does not use, so only this test's workers
		// participate in the per-worker bookkeeping.
		parseWorker.Set("scriptURL", "/sw-update.js")
		parseWorker.Set("state", parseState)
		return parseWorker
	}
	parseFire := func(parseTarget js.Value, parseName string) {
		parseTarget.Call("dispatchEvent", js.Global().Get("Event").New(parseName))
	}

	// Update 1: worker A installs and is then activated.
	parseA := parseNewWorker("installing")
	parseRaw.Set("installing", parseA)
	parseRaw.Call("dispatchEvent", map[string]interface{}{"type": "updatefound"})
	parseA.Set("state", "installed")
	parseFire(parseA, "statechange")
	parseA.Set("state", "activating")
	parseBefore := parseCalls
	parseFire(parseA, "statechange")
	if parseDelta := parseCalls - parseBefore; parseDelta != 1 {
		parseT.Fatalf("one statechange notified %d times; listeners are accumulating per transition", parseDelta)
	}

	// Update 2: a different worker object, same script URL and starting state as A had.
	parseB := parseNewWorker("installing")
	parseRaw.Set("installing", parseB)
	parseRaw.Call("dispatchEvent", map[string]interface{}{"type": "updatefound"})
	parseB.Set("state", "installed")
	parseBefore = parseCalls
	parseFire(parseB, "statechange")
	if parseCalls-parseBefore != 1 {
		parseT.Fatalf("statechange on the second update's worker produced %d notifications, want 1", parseCalls-parseBefore)
	}
}
