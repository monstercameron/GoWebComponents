//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func transitionHooksExample() ui.Node {
	urgent := ui.UseState(0)
	deferredList := ui.UseState([]string{"warm cache"})
	transition := ui.UseTransition()

	start := ui.UseEvent(func() {
		urgent.Update(func(prev int) int { return prev + 1 })
		transition.Start(func() {
			next := make([]string, 0, 60)
			for i := range 60 {
				next = append(next, fmt.Sprintf("Background item %02d", i+1))
			}
			deferredList.Set(next)
		})
	})

	items := make([]ui.Node, 0, len(deferredList.Get()))
	for _, item := range deferredList.Get() {
		items = append(items, html.Li(html.Props{Class: "rounded-xl border border-white/10 bg-slate-950/50 px-4 py-3"}, html.Text(item)))
	}

	status := "Idle"
	if transition.Pending() {
		status = "Transition pending"
	}

	return shared.ExamplePage(
		"ui.StartTransition / ui.UseTransition",
		"Mark non-urgent work explicitly",
		"The urgent counter updates immediately while the larger list refresh is scheduled as background transition work.",
		shared.ExamplePanel("Transition state",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-4"},
				shared.ExampleButton("Start transition update", start),
				shared.ExampleStat("Urgent count", fmt.Sprintf("%d", urgent.Get())),
				shared.ExampleStat("Transition", status),
			),
			html.Ul(html.Props{Class: "mt-6 grid gap-3 md:grid-cols-2"}, items...),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(transitionHooksExample), "#app")
	select {}
}
