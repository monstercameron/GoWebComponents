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

func htmlTagExample() ui.Node {
	tone := ui.UseState("calm")
	setCalm := ui.UseEvent(func() { tone.Set("calm") })
	setAlert := ui.UseEvent(func() { tone.Set("alert") })

	widgetClass := "mt-6 block rounded-[1.5rem] border border-white/10 bg-slate-950/40 p-6"
	if tone.Get() == "alert" {
		widgetClass = "mt-6 block rounded-[1.5rem] border border-amber-400/30 bg-amber-400/10 p-6"
	}

	customWidget := html.Tag("status-widget", html.Props{Class: widgetClass, Data: map[string]string{"tone": tone.Get()}, Raw: map[string]interface{}{"data-owner": "html.Tag"}},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Custom element")),
		html.H2(html.Props{Class: "mt-3 text-2xl font-bold text-white"}, html.Text("status-widget")),
		html.P(html.Props{Class: "mt-4 leading-7 text-slate-300"}, html.Text("html.Tag is the escape hatch for custom elements and uncommon tags that do not need a dedicated wrapper.")),
	)

	highlight := html.Tag("mark", html.Props{Class: "rounded px-2 py-1 bg-cyan-400/20 text-cyan-100"}, html.Text("Uncommon standard tag"))

	return shared.ExamplePage(
		"html.Tag",
		"Create custom or uncommon elements without waiting for a dedicated helper",
		"Use html.Tag when you need a custom element, a newer HTML tag, or a rarely used standard element that does not justify a named wrapper in the package.",
		shared.ExamplePanel("Generic tag builder",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Calm", setCalm),
				shared.ExampleButton("Alert", setAlert),
			),
			customWidget,
			html.P(html.Props{Class: "mt-6 text-slate-300"}, html.Text("The same helper also works for standard-but-uncommon tags like ")), highlight, html.Text("."),
			shared.ExampleCode(
				`html.Tag("status-widget", html.Props{Data: map[string]string{"tone": tone}} , children...)`,
				`html.Tag("mark", html.Props{}, html.Text("highlight"))`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(htmlTagExample), "#app")
	select {}
}
