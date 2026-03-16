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
	pending := ui.UseState(false)
	fail := ui.UseState(false)

	showPending := ui.UseEvent(func() { pending.Update(func(prev bool) bool { return !prev }) })
	showError := ui.UseEvent(func() { fail.Update(func(prev bool) bool { return !prev }) })

	var err error
	if fail.Get() {
		err = errors.New("boundary received a demo error")
	}

	content := html.Div(html.Props{Class: "rounded-2xl border border-emerald-400/30 bg-emerald-400/10 p-6 text-emerald-50"}, html.Text("Async content is ready."))
	fallback := html.Div(html.Props{Class: "rounded-2xl border border-cyan-400/30 bg-cyan-400/10 p-6 text-cyan-50"}, html.Text("Fallback shown while pending is true."))

	return shared.ExamplePage(
		"ui.AsyncBoundary",
		"Switch between content, fallback, and error fallback explicitly",
		"AsyncBoundary is an explicit rendering primitive: you tell it when work is pending and what to show for ready, loading, and error states.",
		shared.ExamplePanel("Boundary states",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Toggle pending", showPending),
				shared.ExampleButton("Toggle error", showError),
			),
			html.Div(html.Props{Class: "mt-6"}, ui.CreateElement(ui.AsyncBoundary, ui.AsyncBoundaryProps{
				Pending:  pending.Get(),
				Error:    err,
				Fallback: fallback,
				ErrorFallback: func(err error) ui.Node {
					return html.Div(html.Props{Class: "rounded-2xl border border-red-400/30 bg-red-400/10 p-6 text-red-50"}, html.Text(err.Error()))
				},
				Content: content,
			})),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(asyncBoundaryExample), "#app")
	select {}
}
