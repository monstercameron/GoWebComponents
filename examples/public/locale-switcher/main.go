//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/examples/shared"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/i18n"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"github.com/monstercameron/GoWebComponents/v6/utils"
)

var localeSwitcherBundle = func() *i18n.Bundle {
	bundle := i18n.NewBundle(i18n.BundleOptions{DefaultLocale: "en", FallbackLocale: "en"})
	bundle.Register("en", i18n.Catalog{
		"demo": {
			"feature":          i18n.Message{Text: "Locale context and formatting"},
			"headline":         i18n.Message{Text: "Ship every market from one component tree"},
			"summary":          i18n.Message{Text: "Switch locale at runtime, reuse one message catalog, and keep pluralized copy, direction, and formatting synchronized without hand-rolled string branching."},
			"cart":             i18n.Message{PluralArg: "count", Plural: map[i18n.PluralCategory]string{i18n.PluralOne: "{count} review queued", i18n.PluralOther: "{count} reviews queued"}},
			"direction":        i18n.Message{Text: "Direction"},
			"formattedRevenue": i18n.Message{Text: "Forecast revenue"},
			"releaseDate":      i18n.Message{Text: "Release date"},
		},
	})
	bundle.Register("fr", i18n.Catalog{
		"demo": {
			"feature":          i18n.Message{Text: "Contexte de langue et formatage"},
			"headline":         i18n.Message{Text: "Lancez chaque marche depuis le meme arbre de composants"},
			"summary":          i18n.Message{Text: "Changez de langue a l'execution et gardez la copie pluralisee, la direction et le formatage aligne sans concatener des chaines a la main."},
			"cart":             i18n.Message{PluralArg: "count", Plural: map[i18n.PluralCategory]string{i18n.PluralOne: "{count} validation en attente", i18n.PluralOther: "{count} validations en attente"}},
			"direction":        i18n.Message{Text: "Direction"},
			"formattedRevenue": i18n.Message{Text: "Revenu prevu"},
			"releaseDate":      i18n.Message{Text: "Date de lancement"},
		},
	})
	bundle.Register("ar", i18n.Catalog{
		"demo": {
			"feature":          i18n.Message{Text: "سياق اللغة والتنسيق"},
			"headline":         i18n.Message{Text: "اطلق كل سوق من شجرة مكونات واحدة"},
			"summary":          i18n.Message{Text: "بدل اللغة اثناء التشغيل وحافظ على النصوص المجمعة والاتجاه والتنسيق بدون تفريع يدوي في كل مكون."},
			"cart":             i18n.Message{PluralArg: "count", Plural: map[i18n.PluralCategory]string{i18n.PluralOne: "مراجعة واحدة قيد الانتظار", i18n.PluralTwo: "مراجعتان قيد الانتظار", i18n.PluralFew: "{count} مراجعات قيد الانتظار", i18n.PluralMany: "{count} مراجعة قيد الانتظار", i18n.PluralOther: "{count} مراجعة قيد الانتظار"}},
			"direction":        i18n.Message{Text: "الاتجاه"},
			"formattedRevenue": i18n.Message{Text: "الايراد المتوقع"},
			"releaseDate":      i18n.Message{Text: "تاريخ الاطلاق"},
		},
	})
	return bundle
}()

func localeSwitcherExample() ui.Node {
	parseLocale := i18n.UseLocale(i18n.LocaleOptions{
		InitialLocale:    "en",
		SupportedLocales: []string{"en", "fr", "ar"},
		FallbackLocale:   "en",
		PersistenceKey:   "gwc-example-locale-switcher",
	})
	return i18n.Provider(i18n.ProviderProps{
		Locale: parseLocale,
		Bundle: localeSwitcherBundle,
		Child:  ui.CreateElement(localeSwitcherPage),
	})
}

func localeSwitcherPage() ui.Node {
	parseIntl := i18n.UseI18n()
	parseQueuedReviews := ui.UseState(2)
	parseShipDate := time.Date(2026, time.March, 16, 10, 30, 0, 0, time.UTC)

	setEnglish := ui.UseEvent(func() { parseIntl.SetLocale("en") })
	setFrench := ui.UseEvent(func() { parseIntl.SetLocale("fr") })
	setArabic := ui.UseEvent(func() { parseIntl.SetLocale("ar") })
	parseAddReview := ui.UseEvent(func() { parseQueuedReviews.Update(func(parsePrevious int) int { return parsePrevious + 1 }) })
	parseRemoveReview := ui.UseEvent(func() {
		parseQueuedReviews.Update(func(parsePrevious2 int) int {
			if parsePrevious2 <= 0 {
				return 0
			}
			return parsePrevious2 - 1
		})
	})

	return html.Div(html.Props{
		Class: "min-h-screen bg-[#08111d] text-slate-100",
		Raw: map[string]interface{}{
			"dir":                 string(parseIntl.Direction()),
			"lang":                parseIntl.Locale(),
			"data-current-locale": parseIntl.Locale(),
		},
	},
		shared.ExamplePage(
			"i18n locale switching",
			parseIntl.T("demo", "feature"),
			parseIntl.T("demo", "summary"),
			shared.ExamplePanel("Runtime locale",
				html.P(html.Props{ID: "locale-switcher-headline", Class: "mt-3 text-3xl font-black tracking-tight text-white"}, html.Text(parseIntl.T("demo", "headline"))),
				html.P(html.Props{ID: "locale-switcher-summary", Class: "mt-4 text-base leading-7 text-slate-300"}, html.Text(parseIntl.T("demo", "summary"))),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					html.Button(html.Props{ID: "switch-locale-en", Class: "rounded-full border border-cyan-400/30 bg-cyan-400/10 px-4 py-2 text-sm font-semibold text-cyan-100", OnClick: setEnglish}, html.Text("English")),
					html.Button(html.Props{ID: "switch-locale-fr", Class: "rounded-full border border-emerald-400/30 bg-emerald-400/10 px-4 py-2 text-sm font-semibold text-emerald-100", OnClick: setFrench}, html.Text("Francais")),
					html.Button(html.Props{ID: "switch-locale-ar", Class: "rounded-full border border-amber-400/30 bg-amber-400/10 px-4 py-2 text-sm font-semibold text-amber-100", OnClick: setArabic}, html.Text("العربية")),
				),
				html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
					shared.ExampleStat("Locale", parseIntl.Locale()),
					shared.ExampleStat(parseIntl.T("demo", "direction"), string(parseIntl.Direction())),
					shared.ExampleStat("Plural copy", parseIntl.T("demo", "cart", i18n.Arguments{"count": parseQueuedReviews.Get()})),
				),
			),
			shared.ExamplePanel("Formatting helpers",
				html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-2"},
					html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.28em] text-slate-400"}, html.Text(parseIntl.T("demo", "formattedRevenue"))),
						html.P(html.Props{ID: "locale-switcher-number", Class: "mt-4 text-3xl font-black text-white"}, html.Text(parseIntl.FormatNumber(12540.75))),
					),
					html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.28em] text-slate-400"}, html.Text(parseIntl.T("demo", "releaseDate"))),
						html.P(html.Props{ID: "locale-switcher-date", Class: "mt-4 text-3xl font-black text-white"}, html.Text(parseIntl.FormatDate(parseShipDate, i18n.DateOptions{Style: i18n.DateStyleLong}))),
					),
				),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					html.Button(html.Props{ID: "locale-switcher-add", Class: "rounded-full border border-white/10 px-4 py-2 text-sm font-semibold text-slate-200", OnClick: parseAddReview}, html.Text("Add review")),
					html.Button(html.Props{ID: "locale-switcher-remove", Class: "rounded-full border border-white/10 px-4 py-2 text-sm font-semibold text-slate-200", OnClick: parseRemoveReview}, html.Text("Remove review")),
				),
				html.P(html.Props{ID: "locale-switcher-cart-copy", Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(fmt.Sprintf("%s | %s", parseIntl.T("demo", "cart", i18n.Arguments{"count": parseQueuedReviews.Get()}), parseIntl.FormatNumber(float64(parseQueuedReviews.Get()))))),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(localeSwitcherExample))
	exampleboot.WaitExampleRuntime()
}
