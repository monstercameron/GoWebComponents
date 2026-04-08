//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
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
				Class: "min-h-screen bg-[#08111d] text-slate-100",
				Raw: map[string]interface{}{
					"dir":                      string(parseIntl.Direction()),
					"lang":                     parseIntl.Locale(),
					"data-bootstrapped-locale": parsePayload.I18n.Locale,
				},
			},
				html.Div(html.Props{Class: "mx-auto max-w-3xl px-6 py-12"},
					html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/80 p-8 shadow-2xl"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("SSR locale bootstrap")),
						html.H1(html.Props{ID: "bootstrap-locale-headline", Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text(parseIntl.T("bootstrap", "headline", i18n.Arguments{"name": "Cam"}))),
						html.P(html.Props{ID: "bootstrap-locale-summary", Class: "mt-4 text-lg leading-8 text-slate-300"}, html.Text(parseIntl.T("bootstrap", "summary"))),
						html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
							html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
								html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Bootstrap locale")),
								html.P(html.Props{ID: "bootstrap-locale-value", Class: "mt-3 text-2xl font-black text-white"}, html.Text(parsePayload.I18n.Locale)),
							),
							html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
								html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Loaded locales")),
								html.P(html.Props{ID: "bootstrap-locale-count", Class: "mt-3 text-2xl font-black text-white"}, html.Text(fmt.Sprintf("%d", len(parseBundle.Locales())))),
							),
							html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
								html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Transferred messages")),
								html.P(html.Props{ID: "bootstrap-message-count", Class: "mt-3 text-2xl font-black text-white"}, html.Text(fmt.Sprintf("%d", bootstrapMessageCount(parsePayload.I18n.Messages)))),
							),
						),
						html.Div(html.Props{Class: "mt-6 rounded-2xl border border-white/10 bg-white/5 p-4"},
							html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Client bootstrap status")),
							html.P(html.Props{ID: "bootstrap-client-status", Class: "mt-3 text-sm leading-7 text-slate-200"}, html.Text("Waiting for client bootstrap status...")),
						),
						html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
							html.Button(html.Props{ID: "bootstrap-switch-en", Class: "rounded-full border border-cyan-400/30 bg-cyan-400/10 px-4 py-2 text-sm font-semibold text-cyan-100", OnClick: ui.UseEvent(func() { parseIntl.SetLocale("en") })}, html.Text("English")),
							html.Button(html.Props{ID: "bootstrap-switch-fr", Class: "rounded-full border border-emerald-400/30 bg-emerald-400/10 px-4 py-2 text-sm font-semibold text-emerald-100", OnClick: ui.UseEvent(func() { parseIntl.SetLocale("fr") })}, html.Text("Francais")),
						),
						html.P(html.Props{ID: "bootstrap-locale-note", Class: "mt-6 text-sm leading-7 text-slate-400"}, html.Text(parseIntl.T("bootstrap", "note"))),
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
