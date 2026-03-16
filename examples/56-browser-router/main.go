//go:build js && wasm
// +build js,wasm

package main

import (
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const browserDemoBase = "/catalog-browser-router"

type browserPageProps struct {
	Title   string
	Summary string
}

func browserPageView(props browserPageProps) ui.Node {
	nav := router.UseNavigate()
	inspection := router.InspectCurrentRoute()
	return shared.ExamplePage(
		props.Title,
		"router.NewRouter / history API",
		props.Summary,
		shared.ExamplePanel("Browser routes",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("History routers use clean paths, but they require server rewrites if you want refreshes and deep links to keep working.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Home", ui.UseEvent(func() { nav.Navigate(browserDemoBase) })),
				shared.ExampleButton("Pricing", ui.UseEvent(func() { nav.Navigate(browserDemoBase + "/pricing") })),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Current path", inspection.Path),
				shared.ExampleStat("URL style", "/path"),
			),
		),
	)
}

func browserPage(title, summary string) *router.Element {
	return ui.CreateElement(browserPageView, browserPageProps{Title: title, Summary: summary})
}

func main() {
	utils.DisableAllDebug()
	window := js.Global().Get("window")
	pathname := window.Get("location").Get("pathname").String()
	if strings.Contains(pathname, ".html") {
		window.Get("history").Call("replaceState", nil, "", browserDemoBase)
	}

	r := router.NewRouter(router.RouterOptions{DefaultRoute: browserDemoBase})
	r.Register(browserDemoBase, func(router.Attrs) *router.Element {
		return browserPage("Browser router basics", "This route uses the History API instead of hash fragments.")
	})
	r.Register(browserDemoBase+"/pricing", func(router.Attrs) *router.Element {
		return browserPage("Pricing route", "Path-based navigation feels cleaner, but it assumes your server rewrites unknown paths to the app entrypoint.")
	})
	r.Register("*", func(router.Attrs) *router.Element {
		return browserPage("Browser 404", "A history router still benefits from a catch-all route inside the app.")
	})
	r.Mount("#app")
	select {}
}