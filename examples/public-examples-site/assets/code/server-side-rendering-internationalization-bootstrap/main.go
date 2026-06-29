//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/i18n"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

func bootstrapMessageCount(parseMessages map[string]map[string]ui.SSRI18nMessage) int {
	parseTotal := 0
	for _, parseLocaleMessages := range parseMessages {
		parseTotal += len(parseLocaleMessages)
	}
	return parseTotal
}

func setBootstrapStatus(parseMessage string) {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseStatus := parseDocument.Call("getElementById", "bootstrap-client-status")
	if parseStatus.Truthy() {
		parseStatus.Set("textContent", parseMessage)
	}
}

// buildBootstrapFallback returns a small SSR i18n payload for preview hosts that do not provide a real bootstrap script.
func buildBootstrapFallback() ui.SSRBootstrap {
	return ui.SSRBootstrap{
		I18n: ui.SSRI18nBootstrap{
			Locale:         "en",
			FallbackLocale: "en",
			Direction:      "ltr",
			Messages: map[string]map[string]ui.SSRI18nMessage{
				"en": {
					"bootstrap.headline": {Text: "Hello, {name}."},
					"bootstrap.summary":  {Text: "The preview host seeded locale and messages from a synthetic SSR bootstrap payload so the same component tree stays readable outside the server pipeline."},
					"bootstrap.note":     {Text: "Switch locales to confirm the resumed bundle changes text without a fresh network request."},
				},
				"fr": {
					"bootstrap.headline": {Text: "Bonjour, {name}."},
					"bootstrap.summary":  {Text: "L'hote de previsualisation a injecte une charge SSR i18n minimale pour garder l'exemple runnable sans serveur dedie."},
					"bootstrap.note":     {Text: "Changez de locale pour verifier que le bundle hydrate reste actif."},
				},
			},
		},
	}
}

func bootstrapI18nExample(parsePayload ui.SSRBootstrap) ui.Node {
	parseBundle := i18n.BundleFromSSRBootstrap(parsePayload.I18n)
	parseLocale := i18n.UseLocale(i18n.LocaleOptions{
		InitialLocale:    parsePayload.I18n.Locale,
		SupportedLocales: parseBundle.Locales(),
		FallbackLocale:   parsePayload.I18n.FallbackLocale,
	})
	return i18n.Provider(i18n.ProviderProps{
		Locale: parseLocale,
		Bundle: parseBundle,
		Child: ui.CreateElement(func() ui.Node {
			parseIntl := i18n.UseI18n()
			return html.Div(html.Props{
				Raw: map[string]interface{}{
					"dir":                      string(parseIntl.Direction()),
					"lang":                     parseIntl.Locale(),
					"data-bootstrapped-locale": parsePayload.I18n.Locale,
				},
			},
				shared.ExamplePage(
					"SSR Locale Bootstrap",
					"ui.SSRBootstrap.I18n",
					"Restore locale and translated messages from the server bootstrap payload before the first interactive render.",
					shared.ExamplePanel("Locale State",
						html.Div(html.Props{Class: "grid gap-3 md:grid-cols-3"},
							shared.ExampleStat("Bootstrap Locale", parsePayload.I18n.Locale),
							shared.ExampleStat("Loaded Locales", fmt.Sprintf("%d", len(parseBundle.Locales()))),
							shared.ExampleStat("Messages", fmt.Sprintf("%d", bootstrapMessageCount(parsePayload.I18n.Messages))),
						),
						html.Div(html.Props{Class: "rounded-[20px] border border-white/10 bg-white/5 p-4"},
							html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Client Bootstrap Status")),
							html.P(html.Props{ID: "bootstrap-client-status", Class: "mt-3 text-sm leading-7 text-slate-200"}, html.Text("Waiting for client bootstrap status...")),
						),
					),
					shared.ExamplePanel("Translations",
						html.P(html.Props{ID: "bootstrap-locale-headline", Class: "text-3xl font-semibold tracking-tight text-white"}, html.Text(parseIntl.T("bootstrap", "headline", i18n.Arguments{"name": "Cam"}))),
						html.P(html.Props{ID: "bootstrap-locale-summary", Class: "text-sm leading-6 text-slate-300"}, html.Text(parseIntl.T("bootstrap", "summary"))),
						html.Div(html.Props{Class: "flex flex-wrap gap-2"},
							html.Button(html.Props{ID: "bootstrap-switch-en", Class: "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-2 text-sm font-medium text-cyan-100 transition-all duration-200 hover:-translate-y-0.5 hover:bg-cyan-400/20 active:translate-y-0 active:scale-95", OnClick: ui.UseEvent(func() { parseIntl.SetLocale("en") })}, html.Text("English")),
							html.Button(html.Props{ID: "bootstrap-switch-fr", Class: "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-2 text-sm font-medium text-cyan-100 transition-all duration-200 hover:-translate-y-0.5 hover:bg-cyan-400/20 active:translate-y-0 active:scale-95", OnClick: ui.UseEvent(func() { parseIntl.SetLocale("fr") })}, html.Text("Francais")),
						),
						html.P(html.Props{ID: "bootstrap-locale-note", Class: "text-sm leading-6 text-slate-400"}, html.Text(parseIntl.T("bootstrap", "note"))),
					),
				),
			)
		}),
	})
}

func main() {
	utils.DisableAllDebug()
	parsePayload, parseErr := ui.ReadBootstrapScript("")
	isParseHydrated := parseErr == nil
	if !isParseHydrated {
		parsePayload = buildBootstrapFallback()
	}
	parseRoot := ui.CreateElement(func() ui.Node { return bootstrapI18nExample(parsePayload) })
	if isParseHydrated {
		_, parseErr = exampleboot.ApplyExampleHydration(parseRoot, ui.HydrationOptions{Bootstrap: parsePayload})
		if parseErr == nil {
			setBootstrapStatus("Hydrated locale and messages from ui.SSRBootstrap.I18n")
			if strings.TrimSpace(parsePayload.I18n.Locale) == "" {
				setBootstrapStatus("Hydrated, but bootstrap locale was empty")
			}
			exampleboot.WaitExampleRuntime()
			return
		}
		setBootstrapStatus("Hydration failed, rendering fallback locale sample")
	}
	exampleboot.RenderExampleRoot(parseRoot)
	setBootstrapStatus("Rendered fallback locale bootstrap sample")
	exampleboot.WaitExampleRuntime()
}
