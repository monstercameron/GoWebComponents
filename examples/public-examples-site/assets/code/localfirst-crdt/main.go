//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/localfirst"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// localfirstExample demonstrates the localfirst LWW-Register CRDT: two replicas edit the same key
// offline, then reconnect and converge — every replica and the authority resolve to the one value,
// deterministically (higher logical clock wins; replica id breaks ties).
func localfirstExample() ui.Node {
	parseResult := ui.UseState("Press \"Run convergence\" to simulate two offline edits reconciling.")

	parseRun := ui.UseEvent(func() {
		parseAuthority := localfirst.NewAuthority()
		parseTab1 := localfirst.NewReplica("tab-1")
		parseTab2 := localfirst.NewReplica("tab-2")

		// Both tabs edit the same key while offline (concurrent, conflicting writes).
		parseTab1.Set("title", "Draft from tab 1")
		parseTab2.Set("title", "Draft from tab 2")

		// Reconnect: each tab pushes its pending writes and merges authoritative state back.
		localfirst.Sync(parseTab1, parseAuthority)
		localfirst.Sync(parseTab2, parseAuthority)
		// tab 1 syncs again to pull the converged value.
		localfirst.Sync(parseTab1, parseAuthority)

		parseV1, _ := parseTab1.Get("title")
		parseV2, _ := parseTab2.Get("title")
		parseAuth := parseAuthority.Snapshot()["title"]
		parseResult.Set(fmt.Sprintf("tab-1 = %q\ntab-2 = %q\nauthority = %q\n→ all converged to one value", parseV1, parseV2, parseAuth))
	})

	return shared.ExamplePage(
		"localfirst (CRDT sync)",
		"Local-first sync that converges deterministically",
		"Two replicas write the same key offline, then reconnect. The last-write-wins register resolves the conflict identically everywhere — no central lock, no lost-update guesswork. Export/RestoreReplica make the offline queue survive a reload.",
		shared.ExamplePanel("Two offline edits reconciling",
			html.Div(html.Props{Class: "mt-3"}, shared.ExampleButton("Run convergence", parseRun)),
			html.Pre(html.Props{Class: "mt-6 whitespace-pre-wrap rounded-lg bg-slate-950 p-4 font-mono text-sm text-emerald-300"},
				html.Text(parseResult.Get())),
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(localfirstExample))
	exampleboot.WaitExampleRuntime()
}
