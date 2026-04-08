//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
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
		Summary: "The loader resolves the active locale from the URL prefix, then picks content for that locale without duplicating the route component per market.",
		Loader:  "Application loaders own locale-specific data selection; the i18n package only normalizes prefixes and path resolution.",
	},
	"fr": {
		Title:   "Route tarifaire avec prefixe de langue",
		Summary: "Le chargeur lit le prefixe de langue dans l'URL puis choisit le contenu adapte sans dupliquer tout le composant de route.",
		Loader:  "Le chargeur applicatif choisit le contenu localise; le package i18n fournit surtout la normalisation de prefixe et la resolution du chemin.",
	},
	"ar": {
		Title:   "مسار تسعير مع بادئة لغة",
		Summary: "يستخرج المحمل اللغة النشطة من بادئة المسار ثم يختار المحتوى المناسب بدون تكرار مكون الصفحة لكل سوق.",
		Loader:  "المحمل في التطبيق يختار المحتوى المحلي، بينما يوفر i18n تطبيع البادئة وتحليل المسار فقط.",
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

	return html.Div(html.Props{Class: "min-h-screen bg-[#08111d] text-slate-100", Raw: map[string]interface{}{"dir": parseDirection, "lang": parseProps.Locale, "data-route-locale": parseProps.Locale}},
		html.Div(html.Props{Class: "mx-auto max-w-4xl px-6 py-12"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/80 p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Locale routing guidance")),
				html.H1(html.Props{ID: "locale-routing-title", Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text(parseProps.Title)),
				html.P(html.Props{ID: "locale-routing-summary", Class: "mt-4 text-lg leading-8 text-slate-300"}, html.Text(parseProps.Summary)),
				html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
					html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Locale")),
						html.P(html.Props{ID: "locale-routing-locale", Class: "mt-3 text-2xl font-black text-white"}, html.Text(parseProps.Locale)),
					),
					html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Base path")),
						html.P(html.Props{ID: "locale-routing-base", Class: "mt-3 text-2xl font-black text-white"}, html.Text(parseProps.BasePath)),
					),
					html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Localized path")),
						html.P(html.Props{ID: "locale-routing-path", Class: "mt-3 text-2xl font-black text-white"}, html.Text(parseProps.LocalizedPath)),
					),
				),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					html.Button(html.Props{ID: "locale-route-en", Class: "rounded-full border border-cyan-400/30 bg-cyan-400/10 px-4 py-2 text-sm font-semibold text-cyan-100", OnClick: parseGoLocale("en")}, html.Text("English route")),
					html.Button(html.Props{ID: "locale-route-fr", Class: "rounded-full border border-emerald-400/30 bg-emerald-400/10 px-4 py-2 text-sm font-semibold text-emerald-100", OnClick: parseGoLocale("fr")}, html.Text("Route francaise")),
					html.Button(html.Props{ID: "locale-route-ar", Class: "rounded-full border border-amber-400/30 bg-amber-400/10 px-4 py-2 text-sm font-semibold text-amber-100", OnClick: parseGoLocale("ar")}, html.Text("المسار العربي")),
				),
				html.P(html.Props{ID: "locale-routing-loader", Class: "mt-6 text-sm leading-7 text-slate-400"}, html.Text(parseProps.Loader)),
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
