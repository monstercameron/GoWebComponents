//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

func fragmentNotices(isExpanded bool) ui.Node {
	if !isExpanded {
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
	parseExpanded := ui.UseState(false)
	parseToggle := ui.UseEvent(func() {
		parseExpanded.Update(func(isPrev bool) bool { return !isPrev })
	})

	return shared.ExamplePage(
		"ui.Fragment",
		"Return sibling nodes without wrapper markup",
		"Fragments are useful when a component needs to yield multiple sibling nodes but you do not want to add an extra DOM container.",
		shared.ExamplePanel("Fragment output",
			html.Div(html.Props{Class: "mt-6 flex gap-3"}, shared.ExampleButton("Toggle sibling set", parseToggle)),
			html.Div(html.Props{Class: "mt-6 grid gap-3"}, fragmentNotices(parseExpanded.Get())),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(fragmentExample))
	exampleboot.WaitExampleRuntime()
}
