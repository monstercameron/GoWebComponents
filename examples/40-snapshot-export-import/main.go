//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func snapshotText(snapshot state.Snapshot) string {
	if len(snapshot) == 0 {
		return "{}"
	}
	data, err := state.MarshalSnapshotJSON(snapshot)
	if err != nil {
		return err.Error()
	}
	return string(data)
}

func snapshotExportImportExample() ui.Node {
	theme := state.UseAtom("catalog-state-snapshot-theme", "Launch")
	seats := state.UseAtom("catalog-state-snapshot-seats", 12)
	captured := ui.UseState(state.Snapshot{})
	status := ui.UseState("Capture a subset of atoms, mutate them, then restore the exact in-memory values.")

	setLaunch := ui.UseEvent(func() { theme.Set("Launch") })
	setGrowth := ui.UseEvent(func() { theme.Set("Growth") })
	addSeats := ui.UseEvent(func() { seats.Update(func(previous int) int { return previous + 4 }) })
	removeSeats := ui.UseEvent(func() {
		if seats.Get() > 4 {
			seats.Update(func(previous int) int { return previous - 4 })
		}
	})
	capture := ui.UseEvent(func() {
		snap, _ := state.GetSnapshot()
		snapshot := snap.Select("catalog-state-snapshot-theme", "catalog-state-snapshot-seats")
		captured.Set(snapshot)
		status.Set("Captured the selected atoms into a memory snapshot.")
	})
	mutate := ui.UseEvent(func() {
		theme.Set("Recovery")
		seats.Set(2)
		status.Set("Live atoms changed after capture. Restore the snapshot to roll back.")
	})
	restore := ui.UseEvent(func() {
		if len(captured.Get()) == 0 {
			status.Set("Capture a snapshot first.")
			return
		}
		if err := state.ApplySnapshot(captured.Get()); err != nil {
			status.Set("Import failed: " + err.Error())
			return
		}
		status.Set("Imported the saved snapshot back into the runtime.")
	})

	return shared.ExamplePage(
		"state.GetSnapshot / state.ApplySnapshot",
		"Capture and restore exact in-memory atom values",
		"Snapshot export/import is useful for undo flows, SSR hydration state handoff, and developer tooling where exact live Go values need to round-trip within the same process.",
		shared.ExamplePanel("Live atoms",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Theme: Launch", setLaunch),
				shared.ExampleButton("Theme: Growth", setGrowth),
				shared.ExampleButton("+4 seats", addSeats),
				shared.ExampleButton("-4 seats", removeSeats),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Theme", theme.Get()),
				shared.ExampleStat("Seats", fmt.Sprintf("%d", seats.Get())),
			),
		),
		shared.ExamplePanel("Snapshot controls",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Capture", capture),
				shared.ExampleButton("Mutate live state", mutate),
				shared.ExampleButton("Restore", restore),
			),
			html.P(html.Props{Class: "mt-6 text-slate-300"}, html.Text(status.Get())),
			shared.ExampleCode(snapshotText(captured.Get())),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(snapshotExportImportExample), "#app")
	select {}
}
