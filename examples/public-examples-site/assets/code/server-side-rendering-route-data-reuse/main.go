//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

type bootstrapCatalog struct {
	Items    []string
	Revision int
}

var initialBootstrap ui.SSRBootstrap
var initialCatalog bootstrapCatalog
var initialCatalogAvailable bool
var initialCatalogConsumed bool

func reuseStat(parseLabel, parseValue string) ui.Node {
	return shared.ExampleStat(parseLabel, parseValue)
}

func reuseButton(parseLabel string, parseHandler ui.Handler) ui.Node {
	return shared.ExampleButton(parseLabel, parseHandler)
}

func decodeBootstrapCatalog(parseData map[string]interface{}) (bootstrapCatalog, bool) {
	parseRawItems, parseOk := parseData["items"].([]interface{})
	if !parseOk {
		return bootstrapCatalog{}, false
	}
	parseItems := make([]string, 0, len(parseRawItems))
	for _, parseRaw := range parseRawItems {
		parseValue, parseOk2 := parseRaw.(string)
		if !parseOk2 {
			continue
		}
		parseItems = append(parseItems, parseValue)
	}
	parseRevision := 0
	if parseRawRevision, parseOk3 := parseData["revision"].(float64); parseOk3 {
		parseRevision = int(parseRawRevision)
	}
	return bootstrapCatalog{Items: parseItems, Revision: parseRevision}, len(parseItems) > 0
}

func consumeInitialCatalog(parseRouteCtx router.RouteContext) (bootstrapCatalog, bool) {
	if !initialCatalogAvailable || initialCatalogConsumed {
		return bootstrapCatalog{}, false
	}
	if strings.TrimSpace(initialBootstrap.Route.Path) != strings.TrimSpace(parseRouteCtx.Path) {
		return bootstrapCatalog{}, false
	}
	initialCatalogConsumed = true
	return initialCatalog, true
}

func liveCatalog() bootstrapCatalog {
	return bootstrapCatalog{
		Items: []string{
			"Client revalidation refreshed the catalog",
			"Only the first loader pass reused bootstrap data",
			"Subsequent runs behave like normal client routing",
		},
		Revision: 2,
	}
}

func catalogPage(parseAttrs router.Attrs) *router.Element {
	parseRevalidator := router.UseRevalidator()
	parseItems, _ := parseAttrs["items"].([]string)
	parseSource, _ := parseAttrs["source"].(string)
	parseRevision, _ := parseAttrs["revision"].(int)

	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4 text-slate-200"}, html.Text(parseItem)))
	}

	return shared.ExamplePage(
		"SSR Route Data Reuse",
		"ui.SSRBootstrap.Route + router.Loader",
		"Reuse route data embedded in the bootstrap payload during hydration, then fall back to the normal live loader path on revalidation.",
		shared.ExamplePanel("Loader State",
			html.Div(html.Props{Class: "grid gap-3 md:grid-cols-2"},
				reuseStat("Data Source", parseSource),
				reuseStat("Revision", fmt.Sprintf("%d", parseRevision)),
			),
			html.Div(html.Props{Class: "flex flex-wrap gap-2"},
				reuseButton("Revalidate Route", ui.UseEvent(func() { parseRevalidator.Revalidate() })),
			),
		),
		shared.ExamplePanel("Catalog Rows",
			html.Div(html.Props{Class: "grid gap-4"}, parseRows...),
		),
	)
}

func ensureCatalogHash(parsePath string) {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() {
		return
	}
	parseLocation := parseWindow.Get("location")
	if parseLocation.Get("hash").String() == "" {
		parseLocation.Set("hash", "#"+parsePath)
	}
}

func main() {
	utils.DisableAllDebug()
	ensureCatalogHash("/products")
	parseBootstrap, parseErr := ui.ReadBootstrapScript("")
	if parseErr == nil {
		initialBootstrap = parseBootstrap
		initialCatalog, initialCatalogAvailable = decodeBootstrapCatalog(parseBootstrap.Data)
	}

	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/products"})
	parseR.Register("/products", catalogPage, router.Options{
		Loader: func(parseCtx context.Context, parseRouteCtx router.RouteContext) (router.Attrs, error) {
			if parseCatalog, parseOk := consumeInitialCatalog(parseRouteCtx); parseOk {
				return router.Attrs{"items": parseCatalog.Items, "source": "bootstrap payload", "revision": parseCatalog.Revision}, nil
			}
			parseCatalog2 := liveCatalog()
			return router.Attrs{"items": parseCatalog2.Items, "source": "live loader", "revision": parseCatalog2.Revision}, nil
		},
	})

	parseRoot := ui.CreateElement(func() ui.Node { return parseR.Current() })
	if _, parseErr := exampleboot.ApplyExampleHydration(parseRoot, ui.HydrationOptions{Bootstrap: initialBootstrap}); parseErr == nil {
		exampleboot.ApplyExampleHydratedRouter(parseR)
		exampleboot.WaitExampleRuntime()
		return
	}
	exampleboot.RenderExampleRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
