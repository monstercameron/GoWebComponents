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

func semanticHTMLExample() ui.Node {
	return shared.ExamplePage(
		"html semantic layout",
		"Compose a document with semantic sectioning elements",
		"The html package is not only for inputs and buttons. It also gives you typed wrappers for semantic structure like header, nav, main, article, aside, section, and footer.",
		html.Header(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Header")),
			html.H2(html.Props{Class: "mt-3 text-3xl font-black text-white"}, html.Text("Semantic document shell")),
			html.Nav(html.Props{Class: "mt-5 flex flex-wrap gap-3 text-sm text-slate-300"},
				html.A(html.Props{Href: "#overview", Class: "rounded-full border border-white/10 px-4 py-2"}, html.Text("Overview")),
				html.A(html.Props{Href: "#insights", Class: "rounded-full border border-white/10 px-4 py-2"}, html.Text("Insights")),
				html.A(html.Props{Href: "#footnotes", Class: "rounded-full border border-white/10 px-4 py-2"}, html.Text("Footnotes")),
			),
		),
		html.Main(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1.4fr)_320px]"},
			html.Article(html.Props{ID: "overview", Class: "rounded-[1.5rem] border border-white/10 bg-slate-950/40 p-6"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Article")),
				html.H2(html.Props{Class: "mt-3 text-2xl font-bold text-white"}, html.Text("Primary content")),
				html.Section(html.Props{Class: "mt-6 grid gap-4"},
					html.P(html.Props{Class: "leading-7 text-slate-300"}, html.Text("Articles and sections give the page structure that assistive technology and browser tooling can reason about more accurately than anonymous div stacks.")),
					html.P(html.Props{Class: "leading-7 text-slate-300"}, html.Text("This example keeps the visual shell simple and emphasizes which wrapper maps to which semantic region in the final markup.")),
				),
			),
			html.Aside(html.Props{ID: "insights", Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Aside")),
				html.H3(html.Props{Class: "mt-3 text-xl font-bold text-white"}, html.Text("Supporting context")),
				html.Ul(html.Props{Class: "mt-5 grid gap-3 text-sm text-slate-300"},
					html.Li(html.Props{}, html.Text("header: global introduction and navigation")),
					html.Li(html.Props{}, html.Text("main: primary page body")),
					html.Li(html.Props{}, html.Text("article: standalone content block")),
					html.Li(html.Props{}, html.Text("aside: supplementary information")),
				),
			),
		),
		html.Footer(html.Props{ID: "footnotes", Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6 text-sm text-slate-400"},
			html.Strong(html.Props{Class: "text-white"}, html.Text("Footer")),
			html.Text(" closes the page-level structure with supporting metadata or links."),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(semanticHTMLExample), "#app")
	select {}
}
