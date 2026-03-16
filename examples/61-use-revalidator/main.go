//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

var (
	revalidateMu   sync.Mutex
	revalidateTick int
)

func nextRevalidateTick() int {
	revalidateMu.Lock()
	defer revalidateMu.Unlock()
	revalidateTick++
	return revalidateTick
}

func revalidatorPage(props router.Attrs) *router.Element {
	revalidator := router.UseRevalidator()
	data := router.UseRouteData()
	if data == nil {
		data = props
	}

	stamp, _ := data["stamp"].(string)
	revision, _ := data["revision"].(string)
	return shared.ExamplePage(
		"router.UseRevalidator",
		"Rerun the current route loader without changing the URL",
		"UseRevalidator is the manual refresh hook for loader-backed routes. It keeps the same path and query string but asks the current route to fetch again.",
		shared.ExamplePanel("Revalidation",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Revalidate current route", ui.UseEvent(func() { revalidator.Revalidate() })),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Loader revision", revision),
				shared.ExampleStat("Loaded at", stamp),
				shared.ExampleStat("Loading", fmt.Sprintf("%t", revalidator.Loading())),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/dashboard"})
	r.Register("/dashboard", revalidatorPage, router.Options{
		Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(280 * time.Millisecond):
			}
			return router.Attrs{
				"revision": fmt.Sprintf("%d", nextRevalidateTick()),
				"stamp":    time.Now().Format("15:04:05.000"),
				"path":     routeCtx.Path,
			}, nil
		},
		Loading: func(props router.Attrs) *router.Element {
			return shared.ExamplePage(
				"router.UseRevalidator",
				"Loader pending",
				"The current route loader is running again without a path change.",
				shared.ExamplePanel("Pending state",
					html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Refreshing route data...")),
				),
			)
		},
	})
	r.Mount("#app")
	select {}
}