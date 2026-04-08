//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func snapshotText(parseSnapshot state.Snapshot) string {
	if len(parseSnapshot) == 0 {
		return "{}"
	}
	parseData, parseErr := state.MarshalSnapshotJSON(parseSnapshot)
	if parseErr != nil {
		return parseErr.Error()
	}
	return string(parseData)
}

func snapshotExportImportExample() ui.Node {
	parseTheme := state.UseAtom("catalog-state-snapshot-theme", "Launch")
	parseSeats := state.UseAtom("catalog-state-snapshot-seats", 12)
	parseCaptured := ui.UseState(state.Snapshot{})
	parseStatus := ui.UseState("Capture a subset of atoms, mutate them, then restore the exact in-memory values.")

	setLaunch := ui.UseEvent(func() { parseTheme.Set("Launch") })
	setGrowth := ui.UseEvent(func() { parseTheme.Set("Growth") })
	parseAddSeats := ui.UseEvent(func() { parseSeats.Update(func(parsePrevious int) int { return parsePrevious + 4 }) })
	parseRemoveSeats := ui.UseEvent(func() {
		if parseSeats.Get() > 4 {
			parseSeats.Update(func(parsePrevious2 int) int { return parsePrevious2 - 4 })
		}
	})
	parseCapture := ui.UseEvent(func() {
		parseSnap, _ := state.GetSnapshot()
		parseSnapshot := parseSnap.Select("catalog-state-snapshot-theme", "catalog-state-snapshot-seats")
		parseCaptured.Set(parseSnapshot)
		parseStatus.Set("Captured the selected atoms into a memory snapshot.")
	})
	parseMutate := ui.UseEvent(func() {
		parseTheme.Set("Recovery")
		parseSeats.Set(2)
		parseStatus.Set("Live atoms changed after capture. Restore the snapshot to roll back.")
	})
	parseRestore := ui.UseEvent(func() {
		if len(parseCaptured.Get()) == 0 {
			parseStatus.Set("Capture a snapshot first.")
			return
		}
		if parseErr := state.ApplySnapshot(parseCaptured.Get()); parseErr != nil {
			parseStatus.Set("Import failed: " + parseErr.Error())
			return
		}
		parseStatus.Set("Imported the saved snapshot back into the runtime.")
	})

	return shared.ExamplePage(
		"state.GetSnapshot / state.ApplySnapshot",
		"Capture and restore exact in-memory atom values",
		"Snapshot export/import is useful for undo flows, SSR hydration state handoff, and developer tooling where exact live Go values need to round-trip within the same process.",
		shared.ExamplePanel("Live atoms",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Theme: Launch", setLaunch),
				shared.ExampleButton("Theme: Growth", setGrowth),
				shared.ExampleButton("+4 seats", parseAddSeats),
				shared.ExampleButton("-4 seats", parseRemoveSeats),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Theme", parseTheme.Get()),
				shared.ExampleStat("Seats", fmt.Sprintf("%d", parseSeats.Get())),
			),
		),
		shared.ExamplePanel("Snapshot controls",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Capture", parseCapture),
				shared.ExampleButton("Mutate live state", parseMutate),
				shared.ExampleButton("Restore", parseRestore),
			),
			html.P(html.Props{Class: "mt-6 text-slate-300"}, html.Text(parseStatus.Get())),
			shared.ExampleCode(snapshotText(parseCaptured.Get())),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(snapshotExportImportExample))
	exampleboot.WaitExampleRuntime()
}
