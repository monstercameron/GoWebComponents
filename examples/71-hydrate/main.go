//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const hydrationTopicAtom = "catalog-hydrate-topic"

func statCard(parseLabel, parseValue string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(parseLabel)),
		html.P(html.Props{Class: "mt-3 text-3xl font-black text-white"}, html.Text(parseValue)),
	)
}

func actionButton(parseLabel string, parseHandler ui.Handler) ui.Node {
	return html.Button(html.Props{OnClick: parseHandler, Class: "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text(parseLabel))
}

func hydrateExample() ui.Node {
	parseTopic := state.UseAtom(hydrationTopicAtom, "client-fallback")
	parseCount := ui.UseState(2)
	parseIncrement := ui.UseEvent(func() { parseCount.Update(func(parsePrevious int) int { return parsePrevious + 1 }) })
	parseToggleTopic := ui.UseEvent(func() {
		if parseTopic.Get() == "server-rendered" {
			parseTopic.Set("client-hydrated")
			return
		}
		parseTopic.Set("server-rendered")
	})

	return html.Div(html.Props{Class: "min-h-screen bg-[#08111d] text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-3xl px-6 py-12"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/80 p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Hydration")),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text("ui.Hydrate")),
				html.P(html.Props{Class: "mt-4 text-lg leading-8 text-slate-300"}, html.Text("Hydrate reuses the HTML that already exists in the DOM, restores bootstrap state, and attaches future updates without replacing the whole tree.")),
				html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
					statCard("Bootstrap topic", parseTopic.Get()),
					statCard("Counter", fmt.Sprintf("%d", parseCount.Get())),
				),
				html.P(html.Props{Class: "mt-6 text-sm leading-7 text-slate-400"}, html.Text("This exact markup shipped in the HTML file before wasm ran. Use the buttons below to prove the resumed tree stayed interactive.")),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					actionButton("Increment counter", parseIncrement),
					actionButton("Toggle topic", parseToggleTopic),
				),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	_, _ = ui.Hydrate(ui.CreateElement(hydrateExample), "#app", ui.HydrationOptions{ScriptID: ui.DefaultBootstrapScriptID})
	select {}
}
