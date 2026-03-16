//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func bootstrapMessageCount(messages map[string]map[string]ui.SSRI18nMessage) int {
	total := 0
	for _, localeMessages := range messages {
		total += len(localeMessages)
	}
	return total
}

func setBootstrapStatus(message string) {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return
	}
	status := document.Call("getElementById", "bootstrap-client-status")
	if status.Truthy() {
		status.Set("textContent", message)
	}
}

func bootstrapI18nExample(payload ui.SSRBootstrap) ui.Node {
	bundle := i18n.BundleFromSSRBootstrap(payload.I18n)
	locale := i18n.UseLocale(i18n.LocaleOptions{
		InitialLocale:    payload.I18n.Locale,
		SupportedLocales: bundle.Locales(),
		FallbackLocale:   payload.I18n.FallbackLocale,
	})
	return i18n.Provider(i18n.ProviderProps{
		Locale: locale,
		Bundle: bundle,
		Child: ui.CreateElement(func() ui.Node {
			intl := i18n.UseI18n()
			return html.Div(html.Props{
				Class: "min-h-screen bg-[#08111d] text-slate-100",
				Raw: map[string]interface{}{
					"dir":                      string(intl.Direction()),
					"lang":                     intl.Locale(),
					"data-bootstrapped-locale": payload.I18n.Locale,
				},
			},
				html.Div(html.Props{Class: "mx-auto max-w-3xl px-6 py-12"},
					html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/80 p-8 shadow-2xl"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("SSR locale bootstrap")),
						html.H1(html.Props{ID: "bootstrap-locale-headline", Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text(intl.T("bootstrap", "headline", i18n.Arguments{"name": "Cam"}))),
						html.P(html.Props{ID: "bootstrap-locale-summary", Class: "mt-4 text-lg leading-8 text-slate-300"}, html.Text(intl.T("bootstrap", "summary"))),
						html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
							html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
								html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Bootstrap locale")),
								html.P(html.Props{ID: "bootstrap-locale-value", Class: "mt-3 text-2xl font-black text-white"}, html.Text(payload.I18n.Locale)),
							),
							html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
								html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Loaded locales")),
								html.P(html.Props{ID: "bootstrap-locale-count", Class: "mt-3 text-2xl font-black text-white"}, html.Text(fmt.Sprintf("%d", len(bundle.Locales())))),
							),
							html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
								html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Transferred messages")),
								html.P(html.Props{ID: "bootstrap-message-count", Class: "mt-3 text-2xl font-black text-white"}, html.Text(fmt.Sprintf("%d", bootstrapMessageCount(payload.I18n.Messages)))),
							),
						),
						html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
							html.Button(html.Props{ID: "bootstrap-switch-en", Class: "rounded-full border border-cyan-400/30 bg-cyan-400/10 px-4 py-2 text-sm font-semibold text-cyan-100", OnClick: ui.UseEvent(func() { intl.SetLocale("en") })}, html.Text("English")),
							html.Button(html.Props{ID: "bootstrap-switch-fr", Class: "rounded-full border border-emerald-400/30 bg-emerald-400/10 px-4 py-2 text-sm font-semibold text-emerald-100", OnClick: ui.UseEvent(func() { intl.SetLocale("fr") })}, html.Text("Francais")),
						),
						html.P(html.Props{ID: "bootstrap-locale-note", Class: "mt-6 text-sm leading-7 text-slate-400"}, html.Text(intl.T("bootstrap", "note"))),
					),
				),
			)
		}),
	})
}

func main() {
	utils.DisableAllDebug()
	payload, err := ui.ReadBootstrapScript("")
	if err != nil {
		setBootstrapStatus("Failed to read SSR locale bootstrap")
		select {}
	}
	root := ui.CreateElement(func() ui.Node { return bootstrapI18nExample(payload) })
	_, err = ui.Hydrate(root, "#app", ui.HydrationOptions{Bootstrap: payload})
	if err != nil {
		setBootstrapStatus("Hydration failed")
		select {}
	}
	setBootstrapStatus("Hydrated locale and messages from ui.SSRBootstrap.I18n")
	if strings.TrimSpace(payload.I18n.Locale) == "" {
		setBootstrapStatus("Hydrated, but bootstrap locale was empty")
	}
	select {}
}
