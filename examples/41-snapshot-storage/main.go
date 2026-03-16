//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const snapshotStorageKey = "catalog-state-storage-demo"

func snapshotStorageText(snapshot state.Snapshot) string {
	if len(snapshot) == 0 {
		return "{}"
	}
	data, err := state.MarshalSnapshotJSON(snapshot)
	if err != nil {
		return err.Error()
	}
	return string(data)
}

func snapshotStorageExample() ui.Node {
	stage := state.UseAtom("catalog-state-storage-stage", "Draft")
	visitors := state.UseAtom("catalog-state-storage-visitors", 1200)
	stored := ui.UseState("{}")
	status := ui.UseState("Persist selected atoms into LocalStorage, then restore them later.")

	seedLaunch := ui.UseEvent(func() {
		stage.Set("Launch")
		visitors.Set(4800)
	})
	seedScale := ui.UseEvent(func() {
		stage.Set("Scale")
		visitors.Set(12500)
	})
	mutate := ui.UseEvent(func() {
		stage.Set("Recovery")
		visitors.Set(900)
		status.Set("Live atoms changed. Restore the persisted snapshot to recover the saved values.")
	})
	persist := ui.UseEvent(func() {
		snapshot := state.ExportSnapshot().Select("catalog-state-storage-stage", "catalog-state-storage-visitors")
		if err := state.SaveSnapshot(snapshotStorageKey, snapshot, state.LocalStorage); err != nil {
			status.Set("Save failed: " + err.Error())
			return
		}
		loaded, ok, err := state.LoadSnapshot(snapshotStorageKey, state.LocalStorage)
		if err != nil {
			status.Set("Load failed: " + err.Error())
			return
		}
		if !ok {
			status.Set("Saved, but the snapshot was not found on read-back.")
			return
		}
		stored.Set(snapshotStorageText(loaded))
		status.Set("Saved the selected atoms to LocalStorage and verified the payload with LoadSnapshot.")
	})
	restore := ui.UseEvent(func() {
		restored, err := state.RestoreSnapshot(snapshotStorageKey, state.LocalStorage)
		if err != nil {
			status.Set("Restore failed: " + err.Error())
			return
		}
		if !restored {
			status.Set("No snapshot found. Persist one first.")
			return
		}
		loaded, ok, err := state.LoadSnapshot(snapshotStorageKey, state.LocalStorage)
		if err == nil && ok {
			stored.Set(snapshotStorageText(loaded))
		}
		status.Set("Restored the saved LocalStorage snapshot back into runtime atoms.")
	})

	return shared.ExamplePage(
		"state.SaveSnapshot / state.LoadSnapshot / state.RestoreSnapshot",
		"Persist JSON snapshots to browser storage",
		"Browser storage helpers serialize snapshots to JSON, making it easy to keep lightweight shared state across refreshes or manual recovery flows.",
		shared.ExamplePanel("Live atoms",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Seed launch", seedLaunch),
				shared.ExampleButton("Seed scale", seedScale),
				shared.ExampleButton("Mutate live state", mutate),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Stage", stage.Get()),
				shared.ExampleStat("Visitors", fmt.Sprintf("%d", visitors.Get())),
			),
		),
		shared.ExamplePanel("Storage controls",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Persist to LocalStorage", persist),
				shared.ExampleButton("Restore from LocalStorage", restore),
			),
			html.P(html.Props{Class: "mt-6 text-slate-300"}, html.Text(status.Get())),
			shared.ExampleCode(
				`state.SaveSnapshot("catalog-state-storage-demo", snapshot, state.LocalStorage)`,
				`state.RestoreSnapshot("catalog-state-storage-demo", state.LocalStorage)`,
				stored.Get(),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(snapshotStorageExample), "#app")
	select {}
}