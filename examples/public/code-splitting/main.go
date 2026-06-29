//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
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
				html.P(html.Props{Class: "mt-3 leading-7 text-cyan-50/80"}, html.Text(fmt.Sprintf("%s version %d resolves after the route shell is already on screen.", parseLabel, parseVersion.Get()))),
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
			shared.ExampleButton("Reload panel", parseRefresh),
		),
		html.P(html.Props{Class: "max-w-3xl text-base leading-7 text-slate-300"}, html.Text(parseSummary)),
		parsePanel,
	)
}

func homePageView() ui.Node {
	parseCurrentPath := router.GetCurrentPath()
	parseNav := router.UseNavigate()
	parseGoTo := func(parsePath string) { parseNav.Navigate(parsePath) }

	return shared.ExamplePage(
		"Code Splitting",
		"router.HashRouter + ui.Lazy",
		"Keep route-family chrome stable while leaf panels load on demand.",
		shared.ExamplePanel("Navigate",
			html.Div(html.Props{Class: "flex flex-wrap gap-3"},
				navButton("Catalog family", "/catalog/overview", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
				navButton("Operations family", "/operations/queue", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
			),
		),
		shared.ExamplePanel("Signals",
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Shell", "Stable"),
				shared.ExampleStat("Families", "2"),
				shared.ExampleStat("Reloads", "Explicit"),
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
	parseWorkspace := router.UseOutlet()
	if parseWorkspace == nil {
		parseWorkspace = lazyPanel("catalog-shell", "Catalog overview panel", "The deferred panel resolves after the catalog shell is already visible.")
	}

	return shared.ExamplePage(
		"Code Splitting",
		"Catalog route family",
		"Use a layout route to keep navigation mounted while child content changes under it.",
		shared.ExamplePanel("Navigate",
			html.Div(html.Props{Class: "flex flex-wrap gap-3"},
				navButton("Home", "/", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
				navButton("Overview", "/catalog/overview", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
				navButton("Insights", "/catalog/insights", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
				navButton("Operations", "/operations/queue", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
			),
		),
		shared.ExamplePanel("Workspace", parseWorkspace),
	)
}

func catalogLayout(_ router.Attrs) *router.Element {
	return ui.CreateElement(catalogLayoutView)
}

func operationsLayoutView() ui.Node {
	parseCurrentPath := router.GetCurrentPath()
	parseNav := router.UseNavigate()
	parseGoTo := func(parsePath string) { parseNav.Navigate(parsePath) }
	parseWorkspace := router.UseOutlet()
	if parseWorkspace == nil {
		parseWorkspace = lazyPanel("operations-shell", "Operations queue panel", "The deferred queue resolves under the operations shell without disturbing navigation.")
	}

	return shared.ExamplePage(
		"Code Splitting",
		"Operations route family",
		"Separate route families can reuse the same lazy pattern while keeping their own shell state.",
		shared.ExamplePanel("Navigate",
			html.Div(html.Props{Class: "flex flex-wrap gap-3"},
				navButton("Home", "/", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
				navButton("Catalog", "/catalog/overview", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
				navButton("Queue", "/operations/queue", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
				navButton("Risk", "/operations/risk", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
			),
		),
		shared.ExamplePanel("Workspace", parseWorkspace),
	)
}

func operationsLayout(_ router.Attrs) *router.Element {
	return ui.CreateElement(operationsLayoutView)
}

func catalogOverviewView() ui.Node {
	return html.Div(html.Props{Class: "space-y-4"},
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
			shared.ExampleStat("Route", "/catalog/overview"),
			shared.ExampleStat("Shell", "Catalog"),
			shared.ExampleStat("Leaf", "Overview"),
		),
		lazyPanel("catalog-overview", "Catalog overview panel", "This leaf route resolves under the catalog shell and keeps the route-family controls in place."),
	)
}

func catalogOverviewPage(_ router.Attrs) *router.Element {
	return ui.CreateElement(catalogOverviewView)
}

func catalogInsightsView() ui.Node {
	return html.Div(html.Props{Class: "space-y-4"},
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
			shared.ExampleStat("Route", "/catalog/insights"),
			shared.ExampleStat("Shell", "Catalog"),
			shared.ExampleStat("Leaf", "Insights"),
		),
		lazyPanel("catalog-insights", "Catalog insights panel", "Switching child routes swaps the deferred panel while the catalog frame stays mounted."),
	)
}

func catalogInsightsPage(_ router.Attrs) *router.Element {
	return ui.CreateElement(catalogInsightsView)
}

func operationsQueueView() ui.Node {
	return html.Div(html.Props{Class: "space-y-4"},
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
			shared.ExampleStat("Route", "/operations/queue"),
			shared.ExampleStat("Shell", "Operations"),
			shared.ExampleStat("Leaf", "Queue"),
		),
		lazyPanel("operations-queue", "Operations queue panel", "This family keeps its own shell and lazy leaf behavior isolated from the catalog stack."),
	)
}

func operationsQueuePage(_ router.Attrs) *router.Element {
	return ui.CreateElement(operationsQueueView)
}

func operationsRiskView() ui.Node {
	return html.Div(html.Props{Class: "space-y-4"},
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
			shared.ExampleStat("Route", "/operations/risk"),
			shared.ExampleStat("Shell", "Operations"),
			shared.ExampleStat("Leaf", "Risk"),
		),
		lazyPanel("operations-risk", "Operations risk panel", "A second leaf route confirms that only the lazy subtree changes identity inside the operations family."),
	)
}

func operationsRiskPage(_ router.Attrs) *router.Element {
	return ui.CreateElement(operationsRiskView)
}

func notFoundView() ui.Node {
	parseCurrentPath := router.GetCurrentPath()
	parseNav := router.UseNavigate()
	parseGoTo := func(parsePath string) { parseNav.Navigate(parsePath) }

	return shared.ExamplePage(
		"Code Splitting",
		"Not found",
		"Unknown routes should still route users back into a valid family quickly.",
		shared.ExamplePanel("Current route", shared.ExampleStat("Path", parseCurrentPath)),
		shared.ExamplePanel("Navigate",
			html.Div(html.Props{Class: "flex flex-wrap gap-3"},
				navButton("Home", "/", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
				navButton("Catalog", "/catalog/overview", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
				navButton("Operations", "/operations/queue", parseCurrentPath, parseGoTo, "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"),
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

	exampleboot.RenderExampleRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
