//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v6/examples/shared"
	"github.com/monstercameron/GoWebComponents/v6/html"
	h "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// renderCard is a layout component that places named slots: header and footer are filled by the
// caller, while body falls back to default content when the caller provides no "body" slot.
func renderCard(parseSlots h.Slots) ui.Node {
	return html.Div(html.Props{Class: "rounded-xl border border-slate-600 bg-slate-900"},
		html.Div(html.Props{Class: "border-b border-slate-700 px-4 py-3 text-lg font-semibold text-slate-100"},
			parseSlots.Render("header")...),
		html.Div(html.Props{Class: "px-4 py-5 text-slate-300"},
			parseSlots.Or("body", html.Text("(no body slot supplied — this is the default content)"))...),
		html.Div(html.Props{Class: "border-t border-slate-700 px-4 py-3 text-sm text-slate-400"},
			parseSlots.Render("footer")...),
	)
}

// namedSlotsExample demonstrates html/shorthand named slots: the caller passes typed named slots and
// the layout component renders them by name, with an overridable default for any missing slot.
func namedSlotsExample() ui.Node {
	parseFilled := h.NewSlots(
		h.Slot("header", html.Text("Release notes")),
		h.Slot("body", html.Text("Named slots let a layout component accept content by name, like Vue named slots or React render-children-by-name.")),
		h.Slot("footer", html.Text("GoWebComponents v4")),
	)
	parseDefaulted := h.NewSlots(
		h.Slot("header", html.Text("Header only")),
		h.Slot("footer", html.Text("Footer only")),
	)

	return shared.ExamplePage(
		"Named Slots",
		"Render children by name with overridable defaults",
		"Both cards use the same renderCard layout. The first supplies header, body, and footer slots. The second omits the body slot, so renderCard's Or(\"body\", default) fallback renders instead.",
		shared.ExamplePanel("Card with all slots filled", renderCard(parseFilled)),
		shared.ExamplePanel("Card with the body slot defaulted", renderCard(parseDefaulted)),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(namedSlotsExample))
	exampleboot.WaitExampleRuntime()
}
