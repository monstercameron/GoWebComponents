//go:build js && wasm
// +build js,wasm

// Command typed-css is the runnable demo for the F3 typed-CSS package. The view
// (counter.go) is authored fully bare — dot-importing css/u for styling and
// html/shorthand for elements, with no css./u. qualifiers anywhere.
package main

import (
	"github.com/monstercameron/GoWebComponents/v5/css"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func main() {
	// Suppress re-injection of any server-rendered rules on hydration.
	css.SeedFromDocument()
	ui.Render(ui.CreateElement(Counter), "#app")
	select {} // keep the wasm instance alive so styles stay mounted.
}
