//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/localfirst"
	"github.com/monstercameron/GoWebComponents/v4/localfirst/facepile"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// presence is the awareness set (TTL of 3 ticks). It is a package var so it persists across renders.
var presence = localfirst.NewPresenceSet(3)

// presenceExample demonstrates localfirst.PresenceSet + facepile.Facepile: ephemeral "who's here"
// awareness with heartbeat expiry and no wall clock — the app advances logical time via Tick, so a
// peer that stops heartbeating deterministically drops off.
func presenceExample() ui.Node {
	parseTick := ui.UseState(0)
	parseRerender := func() { parseTick.Update(func(parsePrev int) int { return parsePrev + 1 }) }

	parseJoinAlice := ui.UseEvent(func() { presence.Update(localfirst.Presence{ClientID: "alice", State: "editing"}); parseRerender() })
	parseJoinBob := ui.UseEvent(func() { presence.Update(localfirst.Presence{ClientID: "bob", State: "viewing"}); parseRerender() })
	parseHeartbeat := ui.UseEvent(func() { presence.Update(localfirst.Presence{ClientID: "alice", State: "editing"}); parseRerender() })
	parseAdvance := ui.UseEvent(func() { presence.Tick(); parseRerender() })

	return shared.ExamplePage(
		"localfirst.PresenceSet",
		"Ephemeral collaboration presence with TTL expiry",
		"Join peers, then advance logical time with Tick. A peer expires after its TTL unless it heartbeats (re-Update). facepile.Facepile renders the live peers. Presence is distinct from synced records: \"who is here now\" has no history.",
		shared.ExamplePanel("Who's here",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Alice joins", parseJoinAlice),
				shared.ExampleButton("Bob joins", parseJoinBob),
				shared.ExampleButton("Alice heartbeat", parseHeartbeat),
				shared.ExampleButton("Advance time (Tick)", parseAdvance),
			),
			html.Div(html.Props{Class: "mt-6"}, facepile.Facepile(presence)),
			html.P(html.Props{Class: "mt-4 text-slate-300"}, html.Text(fmt.Sprintf("Live peers: %d", presence.Count()))),
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(presenceExample))
	exampleboot.WaitExampleRuntime()
}
