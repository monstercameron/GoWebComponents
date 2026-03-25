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

func useIDExample() ui.Node {
	parseNameID := ui.UseId()
	parseNotesID := ui.UseId()

	return shared.ExamplePage(
		"ui.UseId",
		"Generate stable IDs for accessible markup",
		"UseId is handy for wiring labels, helper text, and repeated form sections without hand-managed string IDs.",
		shared.ExamplePanel("Generated IDs",
			html.Div(html.Props{Class: "mt-3 grid gap-4"},
				html.Div(html.Props{},
					html.Label(html.Props{For: parseNameID, Class: "block text-sm uppercase tracking-[0.25em] text-slate-400"}, html.Text("Project name")),
					html.Input(html.Props{ID: parseNameID, Placeholder: "Stable generated ID", Class: "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"}),
					html.P(html.Props{Class: "mt-2 text-xs text-slate-500"}, html.Text("Input ID: "+parseNameID)),
				),
				html.Div(html.Props{},
					html.Label(html.Props{For: parseNotesID, Class: "block text-sm uppercase tracking-[0.25em] text-slate-400"}, html.Text("Release notes")),
					html.Textarea(html.Props{ID: parseNotesID, Rows: 4, Placeholder: "Another stable generated ID", Class: "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"}),
					html.P(html.Props{Class: "mt-2 text-xs text-slate-500"}, html.Text("Textarea ID: "+parseNotesID)),
				),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useIDExample), "#app")
	select {}
}
