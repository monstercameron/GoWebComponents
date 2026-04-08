//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func routeStat(parseLabel, parseValue string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(parseLabel)),
		html.P(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text(parseValue)),
	)
}

func routeButton(parseLabel string, parseHandler ui.Handler) ui.Node {
	return html.Button(html.Props{OnClick: parseHandler, Class: "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text(parseLabel))
}

func routeShell(parseSection, parseSummary string) *router.Element {
	parseNav := router.UseNavigate()
	return html.Div(html.Props{Class: "min-h-screen bg-[#08111d] text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-4xl px-6 py-12"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/80 p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Hydrated router")),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text("router.HydrateMount")),
				html.P(html.Props{Class: "mt-4 text-lg leading-8 text-slate-300"}, html.Text("HydrateMount attaches router listeners after the current route has already been hydrated, so the first route is preserved instead of immediately being rendered again.")),
				html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
					routeStat("Current section", parseSection),
					routeStat("Attach mode", "HydrateMount"),
				),
				html.P(html.Props{Class: "mt-6 text-sm leading-7 text-slate-400"}, html.Text(parseSummary)),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					routeButton("Go to overview", ui.UseEvent(func() { parseNav.Navigate("/docs/overview") })),
					routeButton("Go to API", ui.UseEvent(func() { parseNav.Navigate("/docs/api") })),
				),
			),
		),
	)
}

func ensureHash(parsePath string) {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() {
		return
	}
	parseLocation := parseWindow.Get("location")
	if parseLocation.Get("hash").String() == "" {
		parseLocation.Set("hash", "#"+parsePath)
	}
}

func main() {
	utils.DisableAllDebug()
	ensureHash("/docs/overview")

	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/docs/overview"})
	parseR.Register("/docs/:section", func(parseAttrs router.Attrs) *router.Element {
		parseSection := router.UseParams().Get("section")
		parseSummary := "The first route was hydrated first, then the router started listening for future hash changes without forcing a replacement render."
		if parseSection == "api" {
			parseSummary = "You navigated after hydration. HydrateMount only special-cases the initial attach; future route updates render normally."
		}
		return routeShell(parseSection, parseSummary)
	})

	parseRoot := ui.CreateElement(func() ui.Node { return parseR.Current() })
	_, _ = exampleboot.ApplyExampleHydration(parseRoot)
	exampleboot.ApplyExampleHydratedRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
