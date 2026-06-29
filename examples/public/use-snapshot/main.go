//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/devtools"
	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

func useSnapshotExample() ui.Node {
	parseTick := ui.UseState(0)
	parseSnapshot := devtools.UseSnapshot(700 * time.Millisecond)
	return shared.ExamplePage(
		"devtools.UseSnapshot",
		"Subscribe to periodic runtime inspection snapshots from a component",
		"UseSnapshot is the lightweight hook form of the devtools surface. It lets normal UI render small health summaries without embedding the full panel.",
		shared.ExamplePanel("Snapshot summary",
			html.Div(html.Props{Class: "mt-3 flex gap-3"},
				shared.ExampleButton("Mutate local state", ui.UseEvent(func() { parseTick.Update(func(parsePrevious int) int { return parsePrevious + 1 }) })),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Route path", parseSnapshot.Route.Path),
				shared.ExampleStat("Fibers", fmt.Sprintf("%d", parseSnapshot.Stats.TotalFibers)),
				shared.ExampleStat("Hook entries", fmt.Sprintf("%d", parseSnapshot.Stats.HookEntries)),
				shared.ExampleStat("Diagnostics", fmt.Sprintf("%d", len(parseSnapshot.Diagnostics))),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(useSnapshotExample))
	exampleboot.WaitExampleRuntime()
}
