//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v6/examples/shared"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/state"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"github.com/monstercameron/GoWebComponents/v6/utils"
)

const snapshotStorageKey = "catalog-state-storage-demo"

func snapshotStorageText(parseSnapshot state.Snapshot) string {
	if len(parseSnapshot) == 0 {
		return "{}"
	}
	parseData, parseErr := state.MarshalSnapshotJSON(parseSnapshot)
	if parseErr != nil {
		return parseErr.Error()
	}
	return string(parseData)
}

func snapshotStorageExample() ui.Node {
	parseStage := state.UseAtom("catalog-state-storage-stage", "Draft")
	parseVisitors := state.UseAtom("catalog-state-storage-visitors", 1200)
	parseStored := ui.UseState("{}")
	parseStatus := ui.UseState("Persist selected atoms into LocalStorage, then restore them later.")

	parseSeedLaunch := ui.UseEvent(func() {
		parseStage.Set("Launch")
		parseVisitors.Set(4800)
	})
	parseSeedScale := ui.UseEvent(func() {
		parseStage.Set("Scale")
		parseVisitors.Set(12500)
	})
	parseMutate := ui.UseEvent(func() {
		parseStage.Set("Recovery")
		parseVisitors.Set(900)
		parseStatus.Set("Live atoms changed. Restore the persisted snapshot to recover the saved values.")
	})
	parsePersist := ui.UseEvent(func() {
		parseSnap, _ := state.GetSnapshot()
		parseSnapshot := parseSnap.Select("catalog-state-storage-stage", "catalog-state-storage-visitors")
		if parseErr := state.SaveSnapshot(snapshotStorageKey, parseSnapshot, state.LocalStorage); parseErr != nil {
			parseStatus.Set("Save failed: " + parseErr.Error())
			return
		}
		parseLoaded, parseOk, parseErr2 := state.LoadSnapshot(snapshotStorageKey, state.LocalStorage)
		if parseErr2 != nil {
			parseStatus.Set("Load failed: " + parseErr2.Error())
			return
		}
		if !parseOk {
			parseStatus.Set("Saved, but the snapshot was not found on read-back.")
			return
		}
		parseStored.Set(snapshotStorageText(parseLoaded))
		parseStatus.Set("Saved the selected atoms to LocalStorage and verified the payload with LoadSnapshot.")
	})
	parseRestore := ui.UseEvent(func() {
		parseRestored, parseErr3 := state.RestoreSnapshot(snapshotStorageKey, state.LocalStorage)
		if parseErr3 != nil {
			parseStatus.Set("Restore failed: " + parseErr3.Error())
			return
		}
		if !parseRestored {
			parseStatus.Set("No snapshot found. Persist one first.")
			return
		}
		parseLoaded2, parseOk2, parseErr3 := state.LoadSnapshot(snapshotStorageKey, state.LocalStorage)
		if parseErr3 == nil && parseOk2 {
			parseStored.Set(snapshotStorageText(parseLoaded2))
		}
		parseStatus.Set("Restored the saved LocalStorage snapshot back into runtime atoms.")
	})

	return shared.ExamplePage(
		"state.SaveSnapshot / state.LoadSnapshot / state.RestoreSnapshot",
		"Persist JSON snapshots to browser storage",
		"Browser storage helpers serialize snapshots to JSON, making it easy to keep lightweight shared state across refreshes or manual recovery flows.",
		shared.ExamplePanel("Live atoms",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Seed launch", parseSeedLaunch),
				shared.ExampleButton("Seed scale", parseSeedScale),
				shared.ExampleButton("Mutate live state", parseMutate),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Stage", parseStage.Get()),
				shared.ExampleStat("Visitors", fmt.Sprintf("%d", parseVisitors.Get())),
			),
		),
		shared.ExamplePanel("Storage controls",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Persist to LocalStorage", parsePersist),
				shared.ExampleButton("Restore from LocalStorage", parseRestore),
			),
			html.P(html.Props{Class: "mt-6 text-slate-300"}, html.Text(parseStatus.Get())),
			shared.ExampleCode(
				`state.SaveSnapshot("catalog-state-storage-demo", snapshot, state.LocalStorage)`,
				`state.RestoreSnapshot("catalog-state-storage-demo", state.LocalStorage)`,
				parseStored.Get(),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(snapshotStorageExample))
	exampleboot.WaitExampleRuntime()
}
