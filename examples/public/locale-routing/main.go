//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/i18n"
	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

type localeRouteContent struct {
	Title   string
	Summary string
	Loader  string
}

type localeRoutePageProps struct {
	Locale        string
	BasePath      string
	LocalizedPath string
	Title         string
	Summary       string
	Loader        string
}

var routeLocales = []string{"en", "fr", "ar"}

var routeContent = map[string]localeRouteContent{
	"en": {
		Title:   "Locale-prefixed pricing route",
		Summary: "Resolve the active locale from the URL prefix and reuse one route component for each market.",
		Loader:  "The route loader chooses locale-specific content while i18n normalizes prefixes and path resolution.",
	},
	"fr": {
		Title:   "Route tarifaire avec prefixe de langue",
		Summary: "Lire le prefixe dans l URL puis servir le bon contenu sans dupliquer le composant de route.",
		Loader:  "Le loader choisit le contenu localise et i18n gere la resolution du chemin.",
	},
	"ar": {
		Title:   "Arabic locale pricing route",
		Summary: "Keep one pricing page component and switch locale behavior through the route prefix.",
		Loader:  "The loader maps URL prefix to locale data before rendering the routed view.",
	},
}

func ensureLocaleRoutingHash(parsePath string) {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() {
		return
	}
	parseLocation := parseWindow.Get("location")
	if parseLocation.Get("hash").String() == "" {
		parseLocation.Set("hash", "#"+parsePath)
	}
}

func localeRouteView(parseProps localeRoutePageProps) ui.Node {
	parseNavigate := router.UseNavigate()
	parseDirection := string(i18n.DirectionForLocale(parseProps.Locale))
	parseGoLocale := func(parseNext string) ui.Handler {
		return ui.UseEvent(func() {
			parseNavigate.Navigate(i18n.PrefixPath(parseNext, parseProps.BasePath, i18n.RouteOptions{SupportedLocales: routeLocales, DefaultLocale: "en", OmitDefaultPrefix: true}))
		})
	}

	return html.Div(html.Props{Raw: map[string]interface{}{"dir": parseDirection, "lang": parseProps.Locale, "data-route-locale": parseProps.Locale}},
		shared.ExamplePage(
			"Locale Routing",
			"i18n.ResolvePath + router.Loader",
			"Resolve locale from the URL prefix and keep one routed view alive across localized paths.",
			shared.ExamplePanel("Route State",
				html.Div(html.Props{Class: "grid gap-3 md:grid-cols-3"},
					shared.ExampleStat("Locale", parseProps.Locale),
					shared.ExampleStat("Base Path", parseProps.BasePath),
					shared.ExampleStat("Localized Path", parseProps.LocalizedPath),
				),
			),
			shared.ExamplePanel("Switch Locale",
				html.Div(html.Props{Class: "flex flex-wrap gap-2"},
					shared.ExampleButton("English", parseGoLocale("en")),
					shared.ExampleButton("Francais", parseGoLocale("fr")),
					shared.ExampleButton("Arabic", parseGoLocale("ar")),
				),
				html.P(html.Props{ID: "locale-routing-title", Class: "text-lg font-semibold text-white"}, html.Text(parseProps.Title)),
				html.P(html.Props{ID: "locale-routing-summary", Class: "text-sm leading-6 text-slate-300"}, html.Text(parseProps.Summary)),
				html.P(html.Props{ID: "locale-routing-loader", Class: "text-sm leading-6 text-slate-400"}, html.Text(parseProps.Loader)),
			),
		),
	)
}

func localeRoutePage(parseAttrs router.Attrs) *router.Element {
	return ui.CreateElement(localeRouteView, localeRoutePageProps{
		Locale:        stringAttr(parseAttrs, "locale"),
		BasePath:      stringAttr(parseAttrs, "basePath"),
		LocalizedPath: stringAttr(parseAttrs, "localizedPath"),
		Title:         stringAttr(parseAttrs, "title"),
		Summary:       stringAttr(parseAttrs, "summary"),
		Loader:        stringAttr(parseAttrs, "loader"),
	})
}

func stringAttr(parseAttrs router.Attrs, parseKey string) string {
	parseValue, _ := parseAttrs[parseKey].(string)
	return parseValue
}

func localeRouteLoader(_ context.Context, parseRouteCtx router.RouteContext) (router.Attrs, error) {
	parseResolved := i18n.ResolvePath(parseRouteCtx.Path, i18n.RouteOptions{SupportedLocales: routeLocales, DefaultLocale: "en", OmitDefaultPrefix: true})
	parseContent, parseOk := routeContent[parseResolved.Locale]
	if !parseOk {
		parseContent = routeContent["en"]
	}
	return router.Attrs{
		"locale":        parseResolved.Locale,
		"basePath":      parseResolved.BasePath,
		"localizedPath": parseResolved.LocalizedPath,
		"title":         parseContent.Title,
		"summary":       parseContent.Summary,
		"loader":        parseContent.Loader,
	}, nil
}

func main() {
	utils.DisableAllDebug()
	ensureLocaleRoutingHash("/pricing")
	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/pricing"})
	for _, parsePath := range []string{"/pricing", "/en/pricing", "/fr/pricing", "/ar/pricing"} {
		parseR.Register(parsePath, localeRoutePage, router.Options{Loader: localeRouteLoader})
	}
	exampleboot.RenderExampleRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
