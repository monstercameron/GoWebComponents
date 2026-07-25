//go:build js && wasm
// +build js,wasm

// Command hydration-metrics is the wasm fixture for the #37 hydration e2e. It
// hydrates the shared HydrationProbe against server-rendered DOM and surfaces the
// framework-reported hydration metrics to window so a browser test can assert that
// hydration ADOPTED the server DOM (FallbackCount/DiscardedNodeCount == 0) rather
// than silently client-rendering.
package main

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v5/examples/shared/hydrationprobe"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

func main() {
	utils.DisableAllDebug()

	// Register BEFORE hydrating so the one-shot hydration observation is captured.
	ui.RegisterSSRObserver(func(parseObs ui.SSRObservation) {
		if parseObs.Hydration == nil {
			return
		}
		js.Global().Set("__hydrationFallbackCount", parseObs.Hydration.FallbackCount)
		js.Global().Set("__hydrationDiscarded", parseObs.Hydration.DiscardedNodeCount)
		js.Global().Set("__hydrationMismatch", parseObs.Hydration.MismatchCount)
		js.Global().Set("__hydrationExisting", parseObs.Hydration.ExistingDOMNodeCount)
		js.Global().Set("__hydrationFailed", parseObs.Hydration.Failed)
		js.Global().Set("__hydrationDone", true)
	})

	if _, parseErr := ui.Hydrate(ui.CreateElement(hydrationprobe.HydrationProbe), "#app"); parseErr != nil {
		js.Global().Set("__hydrationError", parseErr.Error())
	}

	// Keep the wasm instance alive so the page (and any post-hydration work) stays live.
	select {}
}
