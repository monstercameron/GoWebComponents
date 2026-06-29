//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
)

// nameSignal lives outside the render function so the same fine-grained signal is shared by both
// bound inputs and the live preview across rerenders.
var nameSignal = state.NewSignal("Ada")

// bindToExample demonstrates html.BindTo: any value satisfying html.Binding (Get/Set string) — here
// a state.Signal[string] — drives a controlled input with no manual value=+oninput pair. Two inputs
// bound to the SAME signal stay in lockstep, and the preview updates fine-grained via signal.Text.
func bindToExample() ui.Node {
	return shared.ExamplePage(
		"html.BindTo",
		"Two-way binding for any handle, not just ui.State",
		"Both inputs bind to one state.Signal[string] with html.BindTo. Type in either field: the other field and the live preview update together, because they all read and write the same signal. html.BindFunc binds an explicit getter/setter, and html.Bind binds a ui.State[string].",
		shared.ExamplePanel("Bound to a shared signal",
			html.Label(html.Props{Class: "mt-3 block text-sm text-slate-300"}, html.Text("Editor A")),
			html.Input(html.PropsOf(html.Class("mt-1 w-full rounded-lg border border-slate-600 bg-slate-900 px-3 py-2 text-slate-100"), html.BindTo(nameSignal))),
			html.Label(html.Props{Class: "mt-4 block text-sm text-slate-300"}, html.Text("Editor B (same signal)")),
			html.Input(html.PropsOf(html.Class("mt-1 w-full rounded-lg border border-slate-600 bg-slate-900 px-3 py-2 text-slate-100"), html.BindTo(nameSignal))),
			html.P(html.Props{Class: "mt-6 text-lg text-slate-100"},
				html.Text("Hello, "),
				nameSignal.Text(func(parseName string) string { return parseName }),
				html.Text("!"),
			),
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(bindToExample))
	exampleboot.WaitExampleRuntime()
}
