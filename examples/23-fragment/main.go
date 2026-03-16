//go:build js && wasm
// +build js,wasm

package main

import (
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func fragmentNotices(expanded bool) ui.Node {
	if !expanded {
		return ui.Fragment(
			html.P(html.Props{Class: "rounded-xl border border-cyan-400/20 bg-cyan-400/10 p-4 text-cyan-50"}, html.Text("Fragments can return sibling nodes without adding an extra wrapper.")),
		)
	}

	return ui.Fragment(
		html.P(html.Props{Class: "rounded-xl border border-cyan-400/20 bg-cyan-400/10 p-4 text-cyan-50"}, html.Text("First sibling from a fragment.")),
		html.P(html.Props{Class: "rounded-xl border border-emerald-400/20 bg-emerald-400/10 p-4 text-emerald-50"}, html.Text("Second sibling from the same fragment.")),
		html.P(html.Props{Class: "rounded-xl border border-fuchsia-400/20 bg-fuchsia-400/10 p-4 text-fuchsia-50"}, html.Text("Fragments keep sibling composition explicit without a synthetic div.")),
	)
}

func fragmentExample() ui.Node {
	expanded := ui.UseState(false)
	toggle := ui.UseEvent(func() {
		expanded.Update(func(prev bool) bool { return !prev })
	})

	return shared.ExamplePage(
		"ui.Fragment",
		"Return sibling nodes without wrapper markup",
		"Fragments are useful when a component needs to yield multiple sibling nodes but you do not want to add an extra DOM container.",
		shared.ExamplePanel("Fragment output",
			html.Div(html.Props{Class: "mt-6 flex gap-3"}, shared.ExampleButton("Toggle sibling set", toggle)),
			html.Div(html.Props{Class: "mt-6 grid gap-3"}, fragmentNotices(expanded.Get())),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(fragmentExample), "#app")
	select {}
}
