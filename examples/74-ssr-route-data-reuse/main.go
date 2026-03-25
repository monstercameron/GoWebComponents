//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
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
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(parseLabel)),
		html.P(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text(parseValue)),
	)
}

func reuseButton(parseLabel string, parseHandler ui.Handler) ui.Node {
	return html.Button(html.Props{OnClick: parseHandler, Class: "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text(parseLabel))
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

	return html.Div(html.Props{Class: "min-h-screen bg-[#08111d] text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-4xl px-6 py-12"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/80 p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("SSR loader bootstrap")),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text("Route data reuse during hydration")),
				html.P(html.Props{Class: "mt-4 text-lg leading-8 text-slate-300"}, html.Text("The first client loader run reuses route data embedded in the bootstrap payload, then later revalidation falls back to the normal live client loader path.")),
				html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
					reuseStat("Data source", parseSource),
					reuseStat("Revision", fmt.Sprintf("%d", parseRevision)),
				),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					reuseButton("Revalidate route", ui.UseEvent(func() { parseRevalidator.Revalidate() })),
				),
				html.Div(html.Props{Class: "mt-6 grid gap-4"}, parseRows...),
			),
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
	_, _ = ui.Hydrate(parseRoot, "#app", ui.HydrationOptions{Bootstrap: initialBootstrap})
	parseR.HydrateMount("#app")
	select {}
}
