//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/devtools"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func useSnapshotExample() ui.Node {
	tick := ui.UseState(0)
	snapshot := devtools.UseSnapshot(700 * time.Millisecond)
	return shared.ExamplePage(
		"devtools.UseSnapshot",
		"Subscribe to periodic runtime inspection snapshots from a component",
		"UseSnapshot is the lightweight hook form of the devtools surface. It lets normal UI render small health summaries without embedding the full panel.",
		shared.ExamplePanel("Snapshot summary",
			html.Div(html.Props{Class: "mt-3 flex gap-3"},
				shared.ExampleButton("Mutate local state", ui.UseEvent(func() { tick.Update(func(previous int) int { return previous + 1 }) })),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Route path", snapshot.Route.Path),
				shared.ExampleStat("Fibers", fmt.Sprintf("%d", snapshot.Stats.TotalFibers)),
				shared.ExampleStat("Hook entries", fmt.Sprintf("%d", snapshot.Stats.HookEntries)),
				shared.ExampleStat("Diagnostics", fmt.Sprintf("%d", len(snapshot.Diagnostics))),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useSnapshotExample), "#app")
	select {}
}
