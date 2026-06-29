//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/ui/erroroverlay"
)

// errorOverlayExample demonstrates ui/erroroverlay: an in-page, dismissible, accessible error modal
// that makes a development failure legible in the page instead of only the console. FromError builds
// the props from a Go error; you can add an actionable hint and a dismiss handler.
func errorOverlayExample() ui.Node {
	parseShow := ui.UseState(false)

	parseTrigger := ui.UseEvent(func() { parseShow.Set(true) })

	parseOverlay := html.Fragment()
	if parseShow.Get() {
		parseProps := erroroverlay.FromError(fmt.Errorf("failed to load dashboard: connection refused"))
		parseProps.Hint = "Check that the API server is running, then retry."
		parseProps.OnDismiss = func() { parseShow.Set(false) }
		parseOverlay = erroroverlay.ErrorOverlay(parseProps)
	}

	return shared.ExamplePage(
		"ui/erroroverlay",
		"In-page, dismissible, accessible error overlay",
		"Click to render an ErrorOverlay built from a Go error with FromError. It shows the title, message, an actionable hint, and an optional stack as a focus-managed modal — pair it with an ErrorBoundary fallback so failures are legible in the page.",
		shared.ExamplePanel("Trigger an error overlay",
			html.Div(html.Props{Class: "mt-3"}, shared.ExampleButton("Show error overlay", parseTrigger)),
			parseOverlay,
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(errorOverlayExample))
	exampleboot.WaitExampleRuntime()
}
