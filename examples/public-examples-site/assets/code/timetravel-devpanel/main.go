//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/timetravel"
	"github.com/monstercameron/GoWebComponents/v5/timetravel/devpanel"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// panelHistory is the snapshot engine the devpanel scrubs over (a package var so it persists across
// renders).
var panelHistory = timetravel.New(50, 0)

// devpanelExample demonstrates timetravel/devpanel.Panel: a ready-made scrubber-timeline devtools UI
// over any *timetravel.History. Recording adds labeled snapshots; the panel drives undo/redo/scrub.
func devpanelExample() ui.Node {
	parseView := ui.UseState(panelHistory.Current())

	parseRecord := ui.UseEvent(func() {
		parseNext := panelHistory.Current() + 1
		panelHistory.Record(fmt.Sprintf("step %d", parseNext), parseNext)
		parseView.Set(parseNext)
	})

	parsePanel := devpanel.Panel(devpanel.Props{
		Model:   panelHistory,
		OnUndo:  func() { if parseValue, parseOk := panelHistory.Undo(); parseOk { parseView.Set(parseValue) } },
		OnRedo:  func() { if parseValue, parseOk := panelHistory.Redo(); parseOk { parseView.Set(parseValue) } },
		OnScrub: func(parseIndex int) { if parseValue, parseOk := panelHistory.ScrubTo(parseIndex); parseOk { parseView.Set(parseValue) } },
	})

	return shared.ExamplePage(
		"timetravel/devpanel.Panel",
		"Drop-in time-travel scrubber over a History",
		"The panel is a framework component dogfooding the framework: it renders a labeled timeline over any *timetravel.History and calls back on undo, redo, and scrub. Record a few steps, then scrub the timeline.",
		shared.ExamplePanel("Scrubbable history",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap items-center gap-4"},
				shared.ExampleButton("Record step", parseRecord),
				shared.ExampleStat("Current value", fmt.Sprintf("%d", parseView.Get())),
			),
			html.Div(html.Props{Class: "mt-6"}, parsePanel),
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(devpanelExample))
	exampleboot.WaitExampleRuntime()
}
