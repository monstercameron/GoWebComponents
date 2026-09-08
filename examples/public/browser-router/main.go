//go:build js && wasm
// +build js,wasm

package main

import (
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v6/examples/shared"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"github.com/monstercameron/GoWebComponents/v6/utils"
)

const browserDemoBase = "/catalog-browser-router"

type browserPageProps struct {
	Title   string
	Summary string
}

func browserPageView(parseProps browserPageProps) ui.Node {
	parseNav := router.UseNavigate()
	parseInspection := router.InspectCurrentRoute()
	return shared.ExamplePage(
		parseProps.Title,
		"router.NewHistoryRouter / history API",
		parseProps.Summary,
		shared.ExamplePanel("Browser routes",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("History routers use clean paths, but they require server rewrites if you want refreshes and deep links to keep working.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Home", ui.UseEvent(func() { parseNav.Navigate(browserDemoBase) })),
				shared.ExampleButton("Pricing", ui.UseEvent(func() { parseNav.Navigate(browserDemoBase + "/pricing") })),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Current path", parseInspection.Path),
				shared.ExampleStat("URL style", "/path"),
			),
		),
	)
}

func browserPage(parseTitle, parseSummary string) *router.Element {
	return ui.CreateElement(browserPageView, browserPageProps{Title: parseTitle, Summary: parseSummary})
}

func main() {
	utils.DisableAllDebug()
	parseWindow := js.Global().Get("window")
	parsePathname := parseWindow.Get("location").Get("pathname").String()
	if strings.Contains(parsePathname, ".html") {
		parseWindow.Get("history").Call("replaceState", nil, "", browserDemoBase)
	}

	parseR := router.NewHistoryRouter(router.RouterOptions{DefaultRoute: browserDemoBase})
	parseR.Register(browserDemoBase, func(router.Attrs) *router.Element {
		return browserPage("Browser router basics", "This route uses the History API instead of hash fragments.")
	})
	parseR.Register(browserDemoBase+"/pricing", func(router.Attrs) *router.Element {
		return browserPage("Pricing route", "Path-based navigation feels cleaner, but it assumes your server rewrites unknown paths to the app entrypoint.")
	})
	parseR.Register("*", func(router.Attrs) *router.Element {
		return browserPage("Browser 404", "A history router still benefits from a catch-all route inside the app.")
	})
	exampleboot.RenderExampleRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
