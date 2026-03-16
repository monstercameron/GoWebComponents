//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
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

func reuseStat(label, value string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(label)),
		html.P(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text(value)),
	)
}

func reuseButton(label string, handler ui.Handler) ui.Node {
	return html.Button(html.Props{OnClick: handler, Class: "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text(label))
}

func decodeBootstrapCatalog(data map[string]interface{}) (bootstrapCatalog, bool) {
	rawItems, ok := data["items"].([]interface{})
	if !ok {
		return bootstrapCatalog{}, false
	}
	items := make([]string, 0, len(rawItems))
	for _, raw := range rawItems {
		value, ok := raw.(string)
		if !ok {
			continue
		}
		items = append(items, value)
	}
	revision := 0
	if rawRevision, ok := data["revision"].(float64); ok {
		revision = int(rawRevision)
	}
	return bootstrapCatalog{Items: items, Revision: revision}, len(items) > 0
}

func consumeInitialCatalog(routeCtx router.RouteContext) (bootstrapCatalog, bool) {
	if !initialCatalogAvailable || initialCatalogConsumed {
		return bootstrapCatalog{}, false
	}
	if strings.TrimSpace(initialBootstrap.Route.Path) != strings.TrimSpace(routeCtx.Path) {
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

func catalogPage(attrs router.Attrs) *router.Element {
	revalidator := router.UseRevalidator()
	items, _ := attrs["items"].([]string)
	source, _ := attrs["source"].(string)
	revision, _ := attrs["revision"].(int)

	rows := make([]ui.Node, 0, len(items))
	for _, item := range items {
		rows = append(rows, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4 text-slate-200"}, html.Text(item)))
	}

	return html.Div(html.Props{Class: "min-h-screen bg-[#08111d] text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-4xl px-6 py-12"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/80 p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("SSR loader bootstrap")),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text("Route data reuse during hydration")),
				html.P(html.Props{Class: "mt-4 text-lg leading-8 text-slate-300"}, html.Text("The first client loader run reuses route data embedded in the bootstrap payload, then later revalidation falls back to the normal live client loader path.")),
				html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
					reuseStat("Data source", source),
					reuseStat("Revision", fmt.Sprintf("%d", revision)),
				),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					reuseButton("Revalidate route", ui.UseEvent(func() { revalidator.Revalidate() })),
				),
				html.Div(html.Props{Class: "mt-6 grid gap-4"}, rows...),
			),
		),
	)
}

func ensureCatalogHash(path string) {
	window := js.Global().Get("window")
	if !window.Truthy() {
		return
	}
	location := window.Get("location")
	if location.Get("hash").String() == "" {
		location.Set("hash", "#"+path)
	}
}

func main() {
	utils.DisableAllDebug()
	ensureCatalogHash("/products")
	bootstrap, err := ui.ReadBootstrapScript("")
	if err == nil {
		initialBootstrap = bootstrap
		initialCatalog, initialCatalogAvailable = decodeBootstrapCatalog(bootstrap.Data)
	}

	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/products"})
	r.Register("/products", catalogPage, router.Options{
		Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
			if catalog, ok := consumeInitialCatalog(routeCtx); ok {
				return router.Attrs{"items": catalog.Items, "source": "bootstrap payload", "revision": catalog.Revision}, nil
			}
			catalog := liveCatalog()
			return router.Attrs{"items": catalog.Items, "source": "live loader", "revision": catalog.Revision}, nil
		},
	})

	root := ui.CreateElement(func() ui.Node { return r.Current() })
	_, _ = ui.Hydrate(root, "#app", ui.HydrationOptions{Bootstrap: initialBootstrap})
	r.HydrateMount("#app")
	select {}
}