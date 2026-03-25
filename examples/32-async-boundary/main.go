//go:build js && wasm
// +build js,wasm

package main

import (
	"errors"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func asyncBoundaryExample() ui.Node {
	parsePending := ui.UseState(false)
	parseFail := ui.UseState(false)

	parseShowPending := ui.UseEvent(func() { parsePending.Update(func(isPrev bool) bool { return !isPrev }) })
	parseShowError := ui.UseEvent(func() { parseFail.Update(func(isPrev2 bool) bool { return !isPrev2 }) })

	var parseErr error
	if parseFail.Get() {
		parseErr = errors.New("boundary received a demo error")
	}

	parseContent := html.Div(html.Props{Class: "rounded-2xl border border-emerald-400/30 bg-emerald-400/10 p-6 text-emerald-50"}, html.Text("Async content is ready."))
	parseFallback := html.Div(html.Props{Class: "rounded-2xl border border-cyan-400/30 bg-cyan-400/10 p-6 text-cyan-50"}, html.Text("Fallback shown while pending is true."))

	return shared.ExamplePage(
		"ui.AsyncBoundary",
		"Switch between content, fallback, and error fallback explicitly",
		"AsyncBoundary is an explicit rendering primitive: you tell it when work is pending and what to show for ready, loading, and error states.",
		shared.ExamplePanel("Boundary states",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Toggle pending", parseShowPending),
				shared.ExampleButton("Toggle error", parseShowError),
			),
			html.Div(html.Props{Class: "mt-6"}, ui.CreateElement(ui.AsyncBoundary, ui.AsyncBoundaryProps{
				Pending:  parsePending.Get(),
				Error:    parseErr,
				Fallback: parseFallback,
				ErrorFallback: func(parseErr2 error) ui.Node {
					return html.Div(html.Props{Class: "rounded-2xl border border-red-400/30 bg-red-400/10 p-6 text-red-50"}, html.Text(parseErr2.Error()))
				},
				Content: parseContent,
			})),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(asyncBoundaryExample), "#app")
	select {}
}
