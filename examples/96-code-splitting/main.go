//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func isActivePath(parseCurrentPath, parseTargetPath string) bool {
	if parseCurrentPath == parseTargetPath {
		return true
	}
	if parseTargetPath == "/" {
		return parseCurrentPath == "/"
	}
	return len(parseCurrentPath) > len(parseTargetPath) && parseCurrentPath[:len(parseTargetPath)] == parseTargetPath && parseCurrentPath[len(parseTargetPath)] == '/'
}

func navButton(parseLabel, parsePath, parseCurrentPath string, parseNavigate func(string), parseActiveClass string) ui.Node {
	parseClassName := "inline-flex rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold text-slate-200 transition-colors hover:bg-cyan-500/15 hover:text-white"
	if isActivePath(parseCurrentPath, parsePath) {
		parseClassName = parseActiveClass
	}
	return html.Button(html.Props{Type: "button", OnClick: ui.UseEvent(func() { parseNavigate(parsePath) }), Class: parseClassName}, html.Text(parseLabel))
}

func shellPanel(parseTitle, parseSubtitle string, parseLinks ...ui.Node) ui.Node {
	return html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/75 p-6 shadow-[0_20px_70px_rgba(2,6,23,0.42)]"},
		html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-cyan-300"}, html.Text(parseTitle)),
		html.H2(html.Props{Class: "mt-3 text-3xl font-black text-white"}, html.Text(parseSubtitle)),
		html.Div(html.Props{Class: "mt-5 flex flex-wrap gap-3"}, parseLinks...),
	)
}

func lazyPanel(parsePanelKey, parseLabel, parseSummary string) ui.Node {
	parseVersion := ui.UseState(1)
	parseRefresh := ui.UseEvent(func() {
		parseVersion.Update(func(parsePrev int) int { return parsePrev + 1 })
	})

	parsePanel := ui.CreateElement(ui.Lazy, ui.LazyProps{
		Loader: func(parseCtx context.Context) (ui.Node, error) {
			select {
			case <-parseCtx.Done():
				return nil, parseCtx.Err()
			case <-time.After(320 * time.Millisecond):
			}

			return html.Div(html.Props{Class: "rounded-[1.5rem] border border-cyan-400/30 bg-cyan-500/10 p-6 text-cyan-50"},
				html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-cyan-200"}, html.Text("Deferred panel")),
				html.H3(html.Props{Class: "mt-3 text-2xl font-black"}, html.Text(parseLabel)),
				html.P(html.Props{Class: "mt-3 leading-7 text-cyan-50/80"}, html.Text(fmt.Sprintf("%s Version %d resolves after the route shell is already on screen.", parseLabel, parseVersion.Get()))),
				html.P(html.Props{Class: "mt-3 leading-7 text-cyan-50/80"}, html.Text(parseSummary)),
				html.P(html.Props{Class: "mt-4 text-sm font-semibold text-cyan-100"}, html.Text(fmt.Sprintf("Chunk key: %s", parsePanelKey))),
			), nil
		},
		Dependencies: []interface{}{parsePanelKey, parseVersion.Get()},
		Delay:        50 * time.Millisecond,
		Fallback: html.Div(html.Props{Class: "rounded-[1.5rem] border border-cyan-400/20 bg-cyan-500/8 p-6 text-cyan-100"},
			html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-cyan-200"}, html.Text("Deferred panel")),
			html.P(html.Props{Class: "mt-3 text-lg font-semibold"}, html.Text(fmt.Sprintf("Resolving %s...", parseLabel))),
			html.P(html.Props{Class: "mt-2 text-sm text-cyan-100/80"}, html.Text(parseSummary)),
		),
		ErrorFallback: func(parseErr error) ui.Node {
			return html.Div(html.Props{Class: "rounded-[1.5rem] border border-red-400/30 bg-red-500/10 p-6 text-red-50"},
				html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-red-200"}, html.Text("Deferred panel failed")),
				html.P(html.Props{Class: "mt-3 text-lg font-semibold"}, html.Text(parseErr.Error())),
			)
		},
	})

	return html.Div(html.Props{Class: "space-y-4"},
		html.Div(html.Props{Class: "flex flex-wrap items-center justify-between gap-3"},
			html.Div(html.Props{},
				html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Lazy subtree")),
				html.H3(html.Props{Class: "mt-2 text-2xl font-black text-white"}, html.Text(parseLabel)),
			),
			html.Button(html.Props{
				OnClick: parseRefresh,
				Class:   "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100 hover:bg-cyan-500/30",
			}, html.Text("Reload panel")),
		),
		html.P(html.Props{Class: "max-w-3xl text-base leading-7 text-slate-300"}, html.Text(parseSummary)),
		parsePanel,
	)
}

func homePageView() ui.Node {
	parseCurrentPath := router.GetCurrentPath()
	parseNav := router.UseNavigate()
	parseGoTo := func(parsePath string) { parseNav.Navigate(parsePath) }
	return html.Div(html.Props{Class: "min-h-screen bg-[radial-gradient(circle_at_top_left,_rgba(34,211,238,0.18),_transparent_28%),linear-gradient(180deg,#08111d_0%,#030712_100%)] px-6 py-10 text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-6xl"},
			html.Div(html.Props{Class: "rounded-[2.5rem] border border-white/10 bg-slate-950/80 p-8 shadow-[0_25px_90px_rgba(2,6,23,0.48)] md:p-12"},
				html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.32em] text-cyan-300"}, html.Text("Route and component splitting")),
				html.H1(html.Props{Class: "mt-4 max-w-3xl text-5xl font-black leading-tight text-white md:text-6xl"}, html.Text("Keep route shells stable while lazy panels resolve underneath them.")),
				html.P(html.Props{Class: "mt-5 max-w-2xl text-lg leading-8 text-slate-300"}, html.Text("This demo keeps the route shell mounted across navigation, while each leaf route resolves a deferred panel on demand. It is a compact reference for route-family boundaries, lazy child boundaries, and fallback behavior.")),
				html.Div(html.Props{Class: "mt-8 flex flex-wrap gap-4"},
					navButton("Open Catalog", "/catalog", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
					navButton("Open Operations", "/operations", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
				),
				html.Div(html.Props{Class: "mt-8 grid gap-4 md:grid-cols-3"},
					html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4 text-slate-200"}, html.Text("Route shells stay in the base app tree while the outlet changes underneath them.")),
					html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4 text-slate-200"}, html.Text("Each leaf route owns a lazy subtree with explicit pending and error fallbacks.")),
					html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4 text-slate-200"}, html.Text("Reloading a lazy panel reruns the deferred work without remounting the surrounding shell.")),
				),
			),
		),
	)
}

func homePage(_ router.Attrs) *router.Element {
	return ui.CreateElement(homePageView)
}

func catalogLayoutView() ui.Node {
	parseCurrentPath := router.GetCurrentPath()
	parseNav := router.UseNavigate()
	parseGoTo := func(parsePath string) { parseNav.Navigate(parsePath) }
	return html.Div(html.Props{Class: "min-h-screen bg-[linear-gradient(180deg,#08111d_0%,#020617_100%)] px-6 py-8 text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-6xl"},
			html.Div(html.Props{Class: "flex flex-wrap items-center justify-between gap-4 rounded-[2rem] border border-white/10 bg-slate-950/85 px-6 py-5 text-slate-100 shadow-[0_24px_80px_rgba(2,6,23,0.45)]"},
				html.Div(html.Props{},
					html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Catalog Shell")),
					html.H1(html.Props{Class: "mt-2 text-3xl font-black"}, html.Text("Catalog route family")),
				),
				html.Div(html.Props{Class: "flex flex-wrap gap-3"},
					navButton("Home", "/", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
					navButton("Catalog", "/catalog", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
					navButton("Operations", "/operations", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
				),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-6 lg:grid-cols-[280px_minmax(0,1fr)]"},
				html.Aside(html.Props{Class: "space-y-6"},
					shellPanel("Route shell", "Stable navigation and shared chrome"),
					html.Div(html.Props{Class: "rounded-[2rem] border border-cyan-500/20 bg-cyan-950/35 p-6 text-slate-100 shadow-[0_18px_60px_rgba(2,6,23,0.4)]"},
						html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.24em] text-cyan-300"}, html.Text("What this proves")),
						html.P(html.Props{Class: "mt-3 leading-7 text-slate-200"}, html.Text("The catalog route family stays mounted while the deferred panel resolves underneath the shared shell.")),
					),
				),
				html.Main(html.Props{Class: "space-y-6"},
					lazyPanel("catalog-shell", "Catalog overview panel", "The deferred panel summarizes first-paint state for the catalog route family."),
				),
			),
		),
	)
}

func catalogLayout(_ router.Attrs) *router.Element {
	return ui.CreateElement(catalogLayoutView)
}

func operationsLayoutView() ui.Node {
	parseCurrentPath := router.GetCurrentPath()
	parseNav := router.UseNavigate()
	parseGoTo := func(parsePath string) { parseNav.Navigate(parsePath) }
	return html.Div(html.Props{Class: "min-h-screen bg-[linear-gradient(180deg,#111827_0%,#020617_100%)] px-6 py-8 text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-6xl"},
			html.Div(html.Props{Class: "flex flex-wrap items-center justify-between gap-4 rounded-[2rem] border border-white/10 bg-slate-950/85 px-6 py-5 text-slate-100 shadow-[0_24px_80px_rgba(2,6,23,0.45)]"},
				html.Div(html.Props{},
					html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-amber-300"}, html.Text("Operations Shell")),
					html.H1(html.Props{Class: "mt-2 text-3xl font-black"}, html.Text("Operations route family")),
				),
				html.Div(html.Props{Class: "flex flex-wrap gap-3"},
					navButton("Home", "/", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-amber-500/30 bg-amber-500/20 px-4 py-2 text-sm font-semibold text-amber-100"),
					navButton("Catalog", "/catalog", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-amber-500/30 bg-amber-500/20 px-4 py-2 text-sm font-semibold text-amber-100"),
					navButton("Operations", "/operations", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-amber-500/30 bg-amber-500/20 px-4 py-2 text-sm font-semibold text-amber-100"),
				),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-6 lg:grid-cols-[280px_minmax(0,1fr)]"},
				html.Aside(html.Props{Class: "space-y-6"},
					shellPanel("Route shell", "Independent shell with the same lazy pattern"),
					html.Div(html.Props{Class: "rounded-[2rem] border border-amber-500/20 bg-amber-950/35 p-6 text-slate-100 shadow-[0_18px_60px_rgba(2,6,23,0.4)]"},
						html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.24em] text-amber-300"}, html.Text("What this proves")),
						html.P(html.Props{Class: "mt-3 leading-7 text-slate-200"}, html.Text("The operations route family has its own shell and can swap to the catalog family without losing the persistent nav and layout chrome.")),
					),
				),
				html.Main(html.Props{Class: "space-y-6"},
					lazyPanel("operations-shell", "Operations queue panel", "The deferred queue panel resolves independently from the catalog family and uses the same lazy boundary rules."),
				),
			),
		),
	)
}

func operationsLayout(_ router.Attrs) *router.Element {
	return ui.CreateElement(operationsLayoutView)
}

func catalogOverviewView() ui.Node {
	return html.Div(html.Props{Class: "space-y-6"},
		html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/75 p-8 shadow-[0_20px_70px_rgba(2,6,23,0.42)]"},
			html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Leaf Route")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Catalog overview")),
			html.P(html.Props{Class: "mt-5 max-w-2xl text-lg leading-8 text-slate-300"}, html.Text("This leaf route lives under the catalog shell, so the shared navigation stays mounted while the outlet content changes.")),
		),
		lazyPanel("catalog-overview", "Catalog overview panel", "The deferred panel summarizes first-paint state for the catalog route family."),
	)
}

func catalogOverviewPage(_ router.Attrs) *router.Element {
	return ui.CreateElement(catalogOverviewView)
}

func catalogInsightsView() ui.Node {
	return html.Div(html.Props{Class: "space-y-6"},
		html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/75 p-8 shadow-[0_20px_70px_rgba(2,6,23,0.42)]"},
			html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Leaf Route")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Catalog insights")),
			html.P(html.Props{Class: "mt-5 max-w-2xl text-lg leading-8 text-slate-300"}, html.Text("Switching to a different child route keeps the catalog shell mounted while the lazy subtree resolves fresh content for the new route.")),
		),
		lazyPanel("catalog-insights", "Catalog insights panel", "The deferred subtree refreshes when the route family changes or when you reload it explicitly."),
	)
}

func catalogInsightsPage(_ router.Attrs) *router.Element {
	return ui.CreateElement(catalogInsightsView)
}

func operationsQueueView() ui.Node {
	return html.Div(html.Props{Class: "space-y-6"},
		html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/75 p-8 shadow-[0_20px_70px_rgba(2,6,23,0.42)]"},
			html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-amber-300"}, html.Text("Leaf Route")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Operations queue")),
			html.P(html.Props{Class: "mt-5 max-w-2xl text-lg leading-8 text-slate-300"}, html.Text("This route family keeps its own shell and lazy panel while staying isolated from the catalog stack.")),
		),
		lazyPanel("operations-queue", "Operations queue panel", "The queue panel represents deferred, route-specific work inside the operations family."),
	)
}

func operationsQueuePage(_ router.Attrs) *router.Element {
	return ui.CreateElement(operationsQueueView)
}

func operationsRiskView() ui.Node {
	return html.Div(html.Props{Class: "space-y-6"},
		html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/75 p-8 shadow-[0_20px_70px_rgba(2,6,23,0.42)]"},
			html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-amber-300"}, html.Text("Leaf Route")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Operations risk")),
			html.P(html.Props{Class: "mt-5 max-w-2xl text-lg leading-8 text-slate-300"}, html.Text("A second route family confirms that the shell swap stays stable even when the deferred subtree changes identity.")),
		),
		lazyPanel("operations-risk", "Operations risk panel", "The risk panel resolves independently from the catalog family and uses the same lazy boundary rules."),
	)
}

func operationsRiskPage(_ router.Attrs) *router.Element {
	return ui.CreateElement(operationsRiskView)
}

func notFoundView() ui.Node {
	parseNav := router.UseNavigate()
	parseGoTo := func(parsePath string) { parseNav.Navigate(parsePath) }
	return html.Div(html.Props{Class: "min-h-screen bg-[linear-gradient(180deg,#08111d_0%,#020617_100%)] px-6 py-10 text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-4xl rounded-[2.5rem] border border-white/10 bg-slate-950/80 p-10 shadow-[0_24px_80px_rgba(2,6,23,0.45)]"},
			html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Not Found")),
			html.H1(html.Props{Class: "mt-3 text-5xl font-black text-white"}, html.Text("That route does not exist.")),
			html.P(html.Props{Class: "mt-5 text-lg leading-8 text-slate-300"}, html.Text("Use the demo links to jump back into the catalog or operations route trees.")),
			html.Div(html.Props{Class: "mt-8 flex flex-wrap gap-4"},
				navButton("Home", "/", router.GetCurrentPath(), parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
				navButton("Catalog", "/catalog/overview", router.GetCurrentPath(), parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
				navButton("Operations", "/operations/queue", router.GetCurrentPath(), parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
			),
		),
	)
}

func notFoundPage(_ router.Attrs) *router.Element {
	return ui.CreateElement(notFoundView)
}

func main() {
	utils.DisableAllDebug()

	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	parseR.Register("/", homePage)
	parseR.Register("/catalog", catalogLayout, router.Options{Layout: true, Title: "Route and Component Splitting - Catalog"})
	parseR.Register("/catalog/overview", catalogOverviewPage, router.Options{Title: "Catalog Overview"})
	parseR.Register("/catalog/insights", catalogInsightsPage, router.Options{Title: "Catalog Insights"})
	parseR.Register("/operations", operationsLayout, router.Options{Layout: true, Title: "Route and Component Splitting - Operations"})
	parseR.Register("/operations/queue", operationsQueuePage, router.Options{Title: "Operations Queue"})
	parseR.Register("/operations/risk", operationsRiskPage, router.Options{Title: "Operations Risk"})
	parseR.Register("*", notFoundPage, router.Options{Title: "Route and Component Splitting Not Found"})

	parseR.Mount("#app")
	select {}
}
