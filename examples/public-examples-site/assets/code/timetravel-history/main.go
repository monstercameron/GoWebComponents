//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/timetravel"
	"github.com/monstercameron/GoWebComponents/ui"
)

// history is the bounded snapshot engine (capacity 50). Record pushes a new snapshot (truncating any
// redo branch); Undo/Redo/ScrubTo navigate it. It owns no clock or DOM, so a demo just mirrors its
// Current() into component state for display.
var history = timetravel.New(50, 0)

// timetravelExample demonstrates the timetravel.History undo/redo/scrub engine driving a counter.
func timetravelExample() ui.Node {
	parseView := ui.UseState(history.Current())

	parseInc := ui.UseEvent(func() {
		parseNext := history.Current() + 1
		history.Record(fmt.Sprintf("set %d", parseNext), parseNext)
		parseView.Set(parseNext)
	})
	parseUndo := ui.UseEvent(func() {
		if parseValue, parseOk := history.Undo(); parseOk {
			parseView.Set(parseValue)
		}
	})
	parseRedo := ui.UseEvent(func() {
		if parseValue, parseOk := history.Redo(); parseOk {
			parseView.Set(parseValue)
		}
	})

	return shared.ExamplePage(
		"timetravel.History",
		"Bounded undo / redo / scrub snapshot engine",
		"Increment to record snapshots, then step backward and forward through them. Recording after an undo truncates the redo branch, exactly like an editor's undo stack. The same engine backs the time-travel devtools panel.",
		shared.ExamplePanel("Counter with history",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-4"},
				shared.ExampleButton("Increment (record)", parseInc),
				shared.ExampleButton("Undo", parseUndo),
				shared.ExampleButton("Redo", parseRedo),
			),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-4"},
				shared.ExampleStat("Current value", fmt.Sprintf("%d", parseView.Get())),
				shared.ExampleStat("Snapshots", fmt.Sprintf("%d", history.Len())),
				shared.ExampleStat("Cursor", fmt.Sprintf("%d", history.Cursor())),
				shared.ExampleStat("Can undo", fmt.Sprintf("%t", history.CanUndo())),
				shared.ExampleStat("Can redo", fmt.Sprintf("%t", history.CanRedo())),
			),
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(timetravelExample))
	exampleboot.WaitExampleRuntime()
}
