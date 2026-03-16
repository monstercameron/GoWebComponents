//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

var themeContext = ui.CreateContext("Slate")

func themeCard(label string) ui.Node {
	theme := ui.UseContext(themeContext)
	className := "mt-4 rounded-2xl border p-4 "
	switch theme {
	case "Sunrise":
		className += "border-amber-400/30 bg-amber-400/10 text-amber-50"
	case "Ocean":
		className += "border-cyan-400/30 bg-cyan-400/10 text-cyan-50"
	default:
		className += "border-white/10 bg-slate-950/50 text-slate-100"
	}
	return html.Div(html.Props{Class: className},
		html.P(html.Props{Class: "text-sm uppercase tracking-[0.25em] opacity-70"}, html.Text(label)),
		html.H3(html.Props{Class: "mt-2 text-2xl font-bold"}, html.Text(theme)),
	)
}

func contextAPIExample() ui.Node {
	globalTheme := ui.UseState("Slate")
	setSlate := ui.UseEvent(func() { globalTheme.Set("Slate") })
	setOcean := ui.UseEvent(func() { globalTheme.Set("Ocean") })

	provided := ui.CreateElement(themeContext.Provider, ui.ContextProviderProps[string]{
		Value: globalTheme.Get(),
		Child: ui.Fragment(
			ui.CreateElement(themeCard, "Inherited provider theme"),
			ui.CreateElement(themeContext.Provider, ui.ContextProviderProps[string]{
				Value: "Sunrise",
				Child: ui.CreateElement(themeCard, "Nested provider override"),
			}),
		),
	})

	return shared.ExamplePage(
		"ui.CreateContext / ui.UseContext",
		"Share subtree-scoped values without prop threading",
		"The first card inherits the outer provider. The second card lives under a nested provider and overrides the same context locally.",
		shared.ExamplePanel("Provider scope",
			html.Div(html.Props{Class: "mt-3 flex gap-3"},
				shared.ExampleButton("Slate", setSlate),
				shared.ExampleButton("Ocean", setOcean),
			),
			provided,
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(contextAPIExample), "#app")
	select {}
}