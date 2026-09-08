//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v6/examples/shared"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"github.com/monstercameron/GoWebComponents/v6/utils"
)

type loaderPageProps struct {
	FallbackData router.Attrs
}

type errorPageProps struct {
	Message string
}

func loaderPageView(parseProps loaderPageProps) ui.Node {
	parseNav := router.UseNavigate()
	parseData := router.UseRouteData()
	if parseData == nil {
		parseData = parseProps.FallbackData
	}

	parseTitle, _ := parseData["title"].(string)
	parseStatus, _ := parseData["status"].(string)
	parsePath := router.InspectCurrentRoute().Path

	return shared.ExamplePage(
		"router route loaders",
		"Load route data before rendering the matched page",
		"Route loaders let a route fetch or derive data with a typed route context. The loading and error states can also be routed through dedicated route-level views.",
		shared.ExamplePanel("Loader-driven route",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Report 7", ui.UseEvent(func() { parseNav.Navigate("/reports/7") })),
				shared.ExampleButton("Report 12", ui.UseEvent(func() { parseNav.Navigate("/reports/12") })),
				shared.ExampleButton("Trigger error", ui.UseEvent(func() { parseNav.Navigate("/reports/500") })),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Path", parsePath),
				shared.ExampleStat("Title", parseTitle),
				shared.ExampleStat("Status", parseStatus),
			),
			shared.ExampleCode(
				`r.Register("/reports/:id", ReportPage, router.Options{Loader: ...})`,
				`data := router.UseRouteData()`,
			),
		),
	)
}

func loaderPage(parseProps router.Attrs) *router.Element {
	return ui.CreateElement(loaderPageView, loaderPageProps{FallbackData: parseProps})
}

func loadingPage(parseProps router.Attrs) *router.Element {
	parsePath, _ := parseProps["path"].(string)
	return shared.ExamplePage(
		"Loader pending",
		"Route-level loading fallback",
		"This route is showing the explicit Loading renderer while the loader is still in flight.",
		shared.ExamplePanel("Pending state",
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Path", parsePath),
				shared.ExampleStat("Loading", "true"),
			),
		),
	)
}

func errorPageView(parseProps errorPageProps) ui.Node {
	parseNav := router.UseNavigate()
	return shared.ExamplePage(
		"Loader error",
		"Route-level error renderer",
		"The route provided an Error renderer, so a loader failure becomes a focused route-level fallback instead of a generic crash.",
		shared.ExamplePanel("Error state",
			html.P(html.Props{Class: "mt-3 text-rose-300"}, html.Text(parseProps.Message)),
			html.Div(html.Props{Class: "mt-6 flex gap-3"}, shared.ExampleButton("Back to report 7", ui.UseEvent(func() { parseNav.Navigate("/reports/7") }))),
		),
	)
}

func errorPage(parseProps router.Attrs) *router.Element {
	parseMessage, _ := parseProps["error"].(string)
	return ui.CreateElement(errorPageView, errorPageProps{Message: parseMessage})
}

func main() {
	utils.DisableAllDebug()
	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/reports/7"})
	parseR.Register("/reports/:id", loaderPage, router.Options{
		Loader: func(parseCtx context.Context, parseRouteCtx router.RouteContext) (router.Attrs, error) {
			select {
			case <-parseCtx.Done():
				return nil, parseCtx.Err()
			case <-time.After(320 * time.Millisecond):
			}
			parseId := parseRouteCtx.Params.Get("id")
			if parseId == "500" {
				return nil, fmt.Errorf("loader failed for report %s", parseId)
			}
			return router.Attrs{
				"title":  "Report " + parseId,
				"status": "Loaded through router.Options.Loader",
			}, nil
		},
		Loading: loadingPage,
		Error:   errorPage,
	})
	exampleboot.RenderExampleRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
