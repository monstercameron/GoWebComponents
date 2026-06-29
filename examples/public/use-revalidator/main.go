//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

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

type revalidatorPageProps struct {
	FallbackData router.Attrs
}

func revalidatorPageView(parseProps revalidatorPageProps) ui.Node {
	parseRevalidator := router.UseRevalidator()
	parseData := router.UseRouteData()
	if parseData == nil {
		parseData = parseProps.FallbackData
	}

	parseStamp, _ := parseData["stamp"].(string)
	parseRevision, _ := parseData["revision"].(string)
	return shared.ExamplePage(
		"router.UseRevalidator",
		"Rerun the current route loader without changing the URL",
		"UseRevalidator is the manual refresh hook for loader-backed routes. It keeps the same path and query string but asks the current route to fetch again.",
		shared.ExamplePanel("Revalidation",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Revalidate current route", ui.UseEvent(func() { parseRevalidator.Revalidate() })),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Loader revision", parseRevision),
				shared.ExampleStat("Loaded at", parseStamp),
				shared.ExampleStat("Loading", fmt.Sprintf("%t", parseRevalidator.Loading())),
			),
		),
	)
}

func revalidatorPage(parseProps router.Attrs) *router.Element {
	return ui.CreateElement(revalidatorPageView, revalidatorPageProps{FallbackData: parseProps})
}

func main() {
	utils.DisableAllDebug()
	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/dashboard"})
	parseR.Register("/dashboard", revalidatorPage, router.Options{
		Loader: func(parseCtx context.Context, parseRouteCtx router.RouteContext) (router.Attrs, error) {
			select {
			case <-parseCtx.Done():
				return nil, parseCtx.Err()
			case <-time.After(280 * time.Millisecond):
			}
			return router.Attrs{
				"revision": fmt.Sprintf("%d", nextRevalidateTick()),
				"stamp":    time.Now().Format("15:04:05.000"),
				"path":     parseRouteCtx.Path,
			}, nil
		},
		Loading: func(parseProps router.Attrs) *router.Element {
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
	exampleboot.RenderExampleRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
