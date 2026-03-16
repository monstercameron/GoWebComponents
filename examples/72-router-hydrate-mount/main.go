//go:build js && wasm
// +build js,wasm

package main

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func routeStat(label, value string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(label)),
		html.P(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text(value)),
	)
}

func routeButton(label string, handler ui.Handler) ui.Node {
	return html.Button(html.Props{OnClick: handler, Class: "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text(label))
}

func routeShell(section, summary string) *router.Element {
	nav := router.UseNavigate()
	return html.Div(html.Props{Class: "min-h-screen bg-[#08111d] text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-4xl px-6 py-12"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/80 p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Hydrated router")),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text("router.HydrateMount")),
				html.P(html.Props{Class: "mt-4 text-lg leading-8 text-slate-300"}, html.Text("HydrateMount attaches router listeners after the current route has already been hydrated, so the first route is preserved instead of immediately being rendered again.")),
				html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
					routeStat("Current section", section),
					routeStat("Attach mode", "HydrateMount"),
				),
				html.P(html.Props{Class: "mt-6 text-sm leading-7 text-slate-400"}, html.Text(summary)),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					routeButton("Go to overview", ui.UseEvent(func() { nav.Navigate("/docs/overview") })),
					routeButton("Go to API", ui.UseEvent(func() { nav.Navigate("/docs/api") })),
				),
			),
		),
	)
}

func ensureHash(path string) {
	window := js.Global().Get("window")
	if !window.Truthy() {
		return
	}
	location := window.Get("location")
	if location.Get("hash").String() == "" {
		location.Set("hash", "#"+path)
	}
}

func main() {
	utils.DisableAllDebug()
	ensureHash("/docs/overview")

	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/docs/overview"})
	r.Register("/docs/:section", func(attrs router.Attrs) *router.Element {
		section := router.UseParams().Get("section")
		summary := "The first route was hydrated first, then the router started listening for future hash changes without forcing a replacement render."
		if section == "api" {
			summary = "You navigated after hydration. HydrateMount only special-cases the initial attach; future route updates render normally."
		}
		return routeShell(section, summary)
	})

	root := ui.CreateElement(func() ui.Node { return r.Current() })
	_, _ = ui.Hydrate(root, "#app")
	r.HydrateMount("#app")
	select {}
}