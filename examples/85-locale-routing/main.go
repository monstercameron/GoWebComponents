//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"syscall/js"

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

func ensureLocaleRoutingHash(path string) {
	window := js.Global().Get("window")
	if !window.Truthy() {
		return
	}
	location := window.Get("location")
	if location.Get("hash").String() == "" {
		location.Set("hash", "#"+path)
	}
}

func localeRouteView(props localeRoutePageProps) ui.Node {
	navigate := router.UseNavigate()
	direction := string(i18n.DirectionForLocale(props.Locale))
	goLocale := func(next string) ui.Handler {
		return ui.UseEvent(func() {
			navigate.Navigate(i18n.PrefixPath(next, props.BasePath, i18n.RouteOptions{SupportedLocales: routeLocales, DefaultLocale: "en", OmitDefaultPrefix: true}))
		})
	}

	return html.Div(html.Props{Class: "min-h-screen bg-[#08111d] text-slate-100", Raw: map[string]interface{}{"dir": direction, "lang": props.Locale, "data-route-locale": props.Locale}},
		html.Div(html.Props{Class: "mx-auto max-w-4xl px-6 py-12"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/80 p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Locale routing guidance")),
				html.H1(html.Props{ID: "locale-routing-title", Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text(props.Title)),
				html.P(html.Props{ID: "locale-routing-summary", Class: "mt-4 text-lg leading-8 text-slate-300"}, html.Text(props.Summary)),
				html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
					html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Locale")),
						html.P(html.Props{ID: "locale-routing-locale", Class: "mt-3 text-2xl font-black text-white"}, html.Text(props.Locale)),
					),
					html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Base path")),
						html.P(html.Props{ID: "locale-routing-base", Class: "mt-3 text-2xl font-black text-white"}, html.Text(props.BasePath)),
					),
					html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Localized path")),
						html.P(html.Props{ID: "locale-routing-path", Class: "mt-3 text-2xl font-black text-white"}, html.Text(props.LocalizedPath)),
					),
				),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					html.Button(html.Props{ID: "locale-route-en", Class: "rounded-full border border-cyan-400/30 bg-cyan-400/10 px-4 py-2 text-sm font-semibold text-cyan-100", OnClick: goLocale("en")}, html.Text("English route")),
					html.Button(html.Props{ID: "locale-route-fr", Class: "rounded-full border border-emerald-400/30 bg-emerald-400/10 px-4 py-2 text-sm font-semibold text-emerald-100", OnClick: goLocale("fr")}, html.Text("Route francaise")),
					html.Button(html.Props{ID: "locale-route-ar", Class: "rounded-full border border-amber-400/30 bg-amber-400/10 px-4 py-2 text-sm font-semibold text-amber-100", OnClick: goLocale("ar")}, html.Text("المسار العربي")),
				),
				html.P(html.Props{ID: "locale-routing-loader", Class: "mt-6 text-sm leading-7 text-slate-400"}, html.Text(props.Loader)),
			),
		),
	)
}

func localeRoutePage(attrs router.Attrs) *router.Element {
	return ui.CreateElement(localeRouteView, localeRoutePageProps{
		Locale:        stringAttr(attrs, "locale"),
		BasePath:      stringAttr(attrs, "basePath"),
		LocalizedPath: stringAttr(attrs, "localizedPath"),
		Title:         stringAttr(attrs, "title"),
		Summary:       stringAttr(attrs, "summary"),
		Loader:        stringAttr(attrs, "loader"),
	})
}

func stringAttr(attrs router.Attrs, key string) string {
	value, _ := attrs[key].(string)
	return value
}

func localeRouteLoader(_ context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
	resolved := i18n.ResolvePath(routeCtx.Path, i18n.RouteOptions{SupportedLocales: routeLocales, DefaultLocale: "en", OmitDefaultPrefix: true})
	content, ok := routeContent[resolved.Locale]
	if !ok {
		content = routeContent["en"]
	}
	return router.Attrs{
		"locale":        resolved.Locale,
		"basePath":      resolved.BasePath,
		"localizedPath": resolved.LocalizedPath,
		"title":         content.Title,
		"summary":       content.Summary,
		"loader":        content.Loader,
	}, nil
}

func main() {
	utils.DisableAllDebug()
	ensureLocaleRoutingHash("/pricing")
	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/pricing"})
	for _, path := range []string{"/pricing", "/en/pricing", "/fr/pricing", "/ar/pricing"} {
		r.Register(path, localeRoutePage, router.Options{Loader: localeRouteLoader})
	}
	r.Mount("#app")
	select {}
}
