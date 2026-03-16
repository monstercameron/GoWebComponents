//go:build js && wasm
// +build js,wasm

package main

import (
	"errors"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type crashProps struct {
	ShouldCrash bool
}

func crashyPanel(props crashProps) ui.Node {
	if props.ShouldCrash {
		panic("demo component crash")
	}
	return html.Div(html.Props{Class: "rounded-2xl border border-emerald-400/30 bg-emerald-400/10 p-6 text-emerald-50"}, html.Text("Child rendered successfully."))
}

func errorBoundaryExample() ui.Node {
	armed := ui.UseState(false)
	arm := ui.UseEvent(func() { armed.Set(true) })
	reset := ui.UseEvent(func() { armed.Set(false) })

	boundary := ui.CreateElement(ui.ErrorBoundary, ui.ErrorBoundaryProps{
		ErrorFallback: func(err error, _ func()) ui.Node {
			if err == nil {
				err = errors.New("unknown error")
			}
			return html.Div(html.Props{Class: "rounded-2xl border border-red-400/30 bg-red-400/10 p-6 text-red-50"}, html.Text("Boundary caught: "+err.Error()))
		},
		Child: ui.CreateElement(crashyPanel, crashProps{ShouldCrash: armed.Get()}),
	})

	return shared.ExamplePage(
		"ui.ErrorBoundary",
		"Contain subtree failures with a fallback UI",
		"When the child panics during render, the boundary shows its fallback instead of letting the whole tree fail.",
		shared.ExamplePanel("Boundary recovery",
			html.Div(html.Props{Class: "mt-3 flex gap-3"},
				shared.ExampleButton("Crash child", arm),
				shared.ExampleButton("Reset child state", reset),
			),
			html.Div(html.Props{Class: "mt-6"}, boundary),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(errorBoundaryExample), "#app")
	select {}
}