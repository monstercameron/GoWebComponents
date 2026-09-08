//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"github.com/monstercameron/GoWebComponents/v6/utils"
)

// parsePrefsProps holds the (empty) props for the preference-probe component.
type parsePrefsProps struct{}

// parsePrefsComponent is a real GWC component that calls the preference hooks
// and renders their current values into stable DOM selectors. Because the hooks
// use UseEffect + Subscribe internally, the framework re-renders this component
// automatically whenever the emulated media preferences change.
func parsePrefsComponent(_ parsePrefsProps) ui.Node {
	parseReduced := ui.UsePrefersReducedMotion()
	parseScheme := ui.UsePrefersColorScheme()
	return html.Div(html.Props{ID: "prefs"},
		html.Span(html.Props{ID: "motion"}, html.Text(fmt.Sprintf("%v", parseReduced))),
		html.Span(html.Props{ID: "scheme"}, html.Text(string(parseScheme))),
	)
}

func main() {
	// Mount the component into the <div id="app"> provided by the boot HTML.
	ui.Render(ui.CreateElement(parsePrefsComponent, parsePrefsProps{}), "#app")

	// Block forever — standard pattern for a wasm main that drives the page.
	utils.WaitForever()
}
