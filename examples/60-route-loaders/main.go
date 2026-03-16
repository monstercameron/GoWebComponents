//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"time"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type loaderPageProps struct {
	FallbackData router.Attrs
}

type errorPageProps struct {
	Message string
}

func loaderPageView(props loaderPageProps) ui.Node {
	nav := router.UseNavigate()
	data := router.UseRouteData()
	if data == nil {
		data = props.FallbackData
	}

	title, _ := data["title"].(string)
	status, _ := data["status"].(string)
	path := router.InspectCurrentRoute().Path

	return shared.ExamplePage(
		"router route loaders",
		"Load route data before rendering the matched page",
		"Route loaders let a route fetch or derive data with a typed route context. The loading and error states can also be routed through dedicated route-level views.",
		shared.ExamplePanel("Loader-driven route",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Report 7", ui.UseEvent(func() { nav.Navigate("/reports/7") })),
				shared.ExampleButton("Report 12", ui.UseEvent(func() { nav.Navigate("/reports/12") })),
				shared.ExampleButton("Trigger error", ui.UseEvent(func() { nav.Navigate("/reports/500") })),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Path", path),
				shared.ExampleStat("Title", title),
				shared.ExampleStat("Status", status),
			),
			shared.ExampleCode(
				`r.Register("/reports/:id", ReportPage, router.Options{Loader: ...})`,
				`data := router.UseRouteData()`,
			),
		),
	)
}

func loaderPage(props router.Attrs) *router.Element {
	return ui.CreateElement(loaderPageView, loaderPageProps{FallbackData: props})
}

func loadingPage(props router.Attrs) *router.Element {
	path, _ := props["path"].(string)
	return shared.ExamplePage(
		"Loader pending",
		"Route-level loading fallback",
		"This route is showing the explicit Loading renderer while the loader is still in flight.",
		shared.ExamplePanel("Pending state",
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Path", path),
				shared.ExampleStat("Loading", "true"),
			),
		),
	)
}

func errorPageView(props errorPageProps) ui.Node {
	nav := router.UseNavigate()
	return shared.ExamplePage(
		"Loader error",
		"Route-level error renderer",
		"The route provided an Error renderer, so a loader failure becomes a focused route-level fallback instead of a generic crash.",
		shared.ExamplePanel("Error state",
			html.P(html.Props{Class: "mt-3 text-rose-300"}, html.Text(props.Message)),
			html.Div(html.Props{Class: "mt-6 flex gap-3"}, shared.ExampleButton("Back to report 7", ui.UseEvent(func() { nav.Navigate("/reports/7") }))),
		),
	)
}

func errorPage(props router.Attrs) *router.Element {
	message, _ := props["error"].(string)
	return ui.CreateElement(errorPageView, errorPageProps{Message: message})
}

func main() {
	utils.DisableAllDebug()
	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/reports/7"})
	r.Register("/reports/:id", loaderPage, router.Options{
		Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(320 * time.Millisecond):
			}
			id := routeCtx.Params.Get("id")
			if id == "500" {
				return nil, fmt.Errorf("loader failed for report %s", id)
			}
			return router.Attrs{
				"title":  "Report " + id,
				"status": "Loaded through router.Options.Loader",
			}, nil
		},
		Loading: loadingPage,
		Error:   errorPage,
	})
	r.Mount("#app")
	select {}
}
