//go:build js && wasm
// +build js,wasm

package main

import (
	"errors"
	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type crashProps struct {
	ShouldCrash bool
}

func crashyPanel(parseProps crashProps) ui.Node {
	if parseProps.ShouldCrash {
		panic("demo component crash")
	}
	return html.Div(html.Props{Class: "rounded-2xl border border-emerald-400/30 bg-emerald-400/10 p-6 text-emerald-50"}, html.Text("Child rendered successfully."))
}

func errorBoundaryExample() ui.Node {
	parseArmed := ui.UseState(false)
	parseArm := ui.UseEvent(func() { parseArmed.Set(true) })
	reset := ui.UseEvent(func() { parseArmed.Set(false) })

	parseBoundary := ui.CreateElement(ui.ErrorBoundary, ui.ErrorBoundaryProps{
		ErrorFallback: func(parseErr error, _ func()) ui.Node {
			if parseErr == nil {
				parseErr = errors.New("unknown error")
			}
			return html.Div(html.Props{Class: "rounded-2xl border border-red-400/30 bg-red-400/10 p-6 text-red-50"}, html.Text("Boundary caught: "+parseErr.Error()))
		},
		Child: ui.CreateElement(crashyPanel, crashProps{ShouldCrash: parseArmed.Get()}),
	})

	return shared.ExamplePage(
		"ui.ErrorBoundary",
		"Contain subtree failures with a fallback UI",
		"When the child panics during render, the boundary shows its fallback instead of letting the whole tree fail.",
		shared.ExamplePanel("Boundary recovery",
			html.Div(html.Props{Class: "mt-3 flex gap-3"},
				shared.ExampleButton("Crash child", parseArm),
				shared.ExampleButton("Reset child state", reset),
			),
			html.Div(html.Props{Class: "mt-6"}, parseBoundary),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(errorBoundaryExample))
	exampleboot.WaitExampleRuntime()
}
