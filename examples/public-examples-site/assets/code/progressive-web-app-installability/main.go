//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/pwa"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func installabilityManifest() pwa.Manifest {
	return pwa.Manifest{
		ID:              "/97-pwa-installability/",
		Name:            "GoWebComponents PWA Installability Demo",
		ShortName:       "PWA Install",
		Description:     "Focused installability example for ObserveInstallability and RegisterServiceWorker.",
		StartURL:        "/97-pwa-installability/installability.html",
		Scope:           "/97-pwa-installability/",
		Display:         "standalone",
		ThemeColor:      "#0f172a",
		BackgroundColor: "#08111d",
		Icons: []pwa.ManifestImage{
			{Src: "/static/images/favicon/android-chrome-192x192.png", Sizes: "192x192", Type: "image/png"},
			{Src: "/static/images/favicon/android-chrome-512x512.png", Sizes: "512x512", Type: "image/png"},
		},
	}
}

func describePWAError(parsePrefix string, parseErr error) string {
	if parseErr == nil {
		return parsePrefix
	}
	if parseCode, parseOk := interop.CodeOf(parseErr); parseOk {
		return fmt.Sprintf("%s [%s]: %v", parsePrefix, parseCode, parseErr)
	}
	return fmt.Sprintf("%s: %v", parsePrefix, parseErr)
}

func boolLabel(isValue bool) string {
	if isValue {
		return "yes"
	}
	return "no"
}

func formatReasons(parseReasons []string) string {
	if len(parseReasons) == 0 {
		return "No blockers currently reported by ObserveInstallability()."
	}
	return strings.Join(parseReasons, " | ")
}

func formatServiceWorkerSnapshot(parseSnapshot pwa.ServiceWorkerSnapshot) string {
	parseParts := make([]string, 0, 4)
	if parseSnapshot.Scope != "" {
		parseParts = append(parseParts, "scope="+parseSnapshot.Scope)
	}
	parseParts = append(parseParts, "controller="+boolLabel(parseSnapshot.HasController))
	if parseSnapshot.Active.ScriptURL != "" {
		parseParts = append(parseParts, "active="+string(parseSnapshot.Active.State)+" @ "+parseSnapshot.Active.ScriptURL)
	}
	if parseSnapshot.Waiting.ScriptURL != "" {
		parseParts = append(parseParts, "waiting="+string(parseSnapshot.Waiting.State)+" @ "+parseSnapshot.Waiting.ScriptURL)
	}
	if parseSnapshot.Installing.ScriptURL != "" {
		parseParts = append(parseParts, "installing="+string(parseSnapshot.Installing.State)+" @ "+parseSnapshot.Installing.ScriptURL)
	}
	return strings.Join(parseParts, " | ")
}

func installabilityExample() ui.Node {
	parseManifest := installabilityManifest()
	parseManifestJSON, parseErr := pwa.MarshalManifestJSONIndented(parseManifest, "", "  ")
	parseManifestPreview := "manifest preview unavailable"
	if parseErr == nil {
		parseManifestPreview = string(parseManifestJSON)
	}
	parseInitialInstallability := pwa.InstallabilityState{ManifestValid: true}
	if parseValidateErr := parseManifest.Validate(); parseValidateErr != nil {
		parseInitialInstallability.ManifestValid = false
		parseInitialInstallability.ManifestError = parseValidateErr.Error()
		parseInitialInstallability.Reasons = []string{parseValidateErr.Error()}
	} else {
		parseInitialInstallability.Reasons = []string{"Waiting for browser installability signals."}
	}

	parseManagerRef := ui.UseRef[*pwa.InstallabilityManager](nil)
	parseRegistrationRef := ui.UseRef[*pwa.ServiceWorkerRegistration](nil)
	parseServiceWorkerStartedRef := ui.UseRef(false)
	parseServiceWorkerCancelRef := ui.UseRef[func()](nil)
	parseInstallState := ui.UseState(parseInitialInstallability)
	parsePromptStatus := ui.UseState("Install prompt idle.")
	parseServiceWorkerStatus := ui.UseState("Service worker registration pending.")
	parseServiceWorkerSnapshot := ui.UseState("No service worker snapshot yet.")

	applyInstallabilityState := func(parseState2 pwa.InstallabilityState) {
		parseInstallState.Set(parseState2)
	}
	applyServiceWorkerSnapshot := func(parseSnapshot pwa.ServiceWorkerSnapshot) {
		parseFormatted := formatServiceWorkerSnapshot(parseSnapshot)
		if strings.TrimSpace(parseFormatted) == "" {
			parseFormatted = "No service worker snapshot yet."
		}
		parseServiceWorkerSnapshot.Set(parseFormatted)
	}

	ui.UseEffect(func() func() {
		parseCleanups := make([]func(), 0, 1)
		if parseManagerRef.Get() == nil {
			parseManager, parseObserveErr := pwa.ObserveInstallability(pwa.InstallabilityOptions{Manifest: &parseManifest})
			if parseObserveErr != nil {
				parsePromptStatus.Set(describePWAError("Installability observation unavailable", parseObserveErr))
			} else {
				parseManagerRef.Set(&parseManager)
				applyInstallabilityState(parseManager.State())
				parseSubscription, parseSubscribeErr := parseManager.Subscribe(func(parseState3 pwa.InstallabilityState) {
					applyInstallabilityState(parseState3)
				})
				if parseSubscribeErr == nil {
					parseCleanups = append(parseCleanups, parseSubscription.Cancel)
				}
			}
		}
		return func() {
			for _, parseCancel := range parseCleanups {
				parseCancel()
			}
		}
	}, "installability-observer")

	ui.UseEffect(func() func() {
		if parseRegistrationRef.Get() == nil && !parseServiceWorkerStartedRef.Get() {
			parseServiceWorkerStartedRef.Set(true)
			parseServiceWorkerStatus.Set("Registering service worker...")
			go func() {
				parseRegistration, parseRegisterErr := pwa.RegisterServiceWorker(context.Background(), pwa.ServiceWorkerOptions{
					URL:   "/97-pwa-installability/sw.js",
					Scope: "/97-pwa-installability/",
				})
				if parseRegisterErr != nil {
					parseServiceWorkerStatus.Set(describePWAError("Service worker registration failed", parseRegisterErr))
					return
				}
				parseRegistrationRef.Set(&parseRegistration)
				parseServiceWorkerStatus.Set("Service worker registered through pwa.RegisterServiceWorker(...).")
				applyServiceWorkerSnapshot(parseRegistration.Snapshot())
				parseSubscription2, parseSubscribeErr2 := parseRegistration.SubscribeLifecycle(func(parseSnapshot2 pwa.ServiceWorkerSnapshot) {
					applyServiceWorkerSnapshot(parseSnapshot2)
					parseServiceWorkerStatus.Set("Service worker lifecycle changed.")
				})
				if parseSubscribeErr2 == nil {
					parseServiceWorkerCancelRef.Set(parseSubscription2.Cancel)
				}
			}()
		}
		return func() {
			if parseCancel2 := parseServiceWorkerCancelRef.Get(); parseCancel2 != nil {
				parseCancel2()
				parseServiceWorkerCancelRef.Set(nil)
			}
		}
	}, "installability-service-worker")

	parseRefreshInstallability := ui.UseEvent(func() {
		if parseManager2 := parseManagerRef.Get(); parseManager2 != nil {
			applyInstallabilityState(parseManager2.State())
			parsePromptStatus.Set("Installability state refreshed from ObserveInstallability().")
		}
	})
	parsePromptInstall := ui.UseEvent(func() {
		parseManager3 := parseManagerRef.Get()
		if parseManager3 == nil {
			parsePromptStatus.Set("Installability manager is not ready yet.")
			return
		}
		parsePromptStatus.Set("Requesting install prompt...")
		go func() {
			parseResult, parsePromptErr := parseManager3.Prompt(context.Background())
			if parsePromptErr != nil {
				parsePromptStatus.Set(describePWAError("Install prompt unavailable", parsePromptErr))
				return
			}
			parsePromptStatus.Set(fmt.Sprintf("Browser prompt outcome=%s platform=%s", parseResult.Outcome, parseResult.Platform))
		}()
	})
	parseUpdateServiceWorker := ui.UseEvent(func() {
		parseRegistration2 := parseRegistrationRef.Get()
		if parseRegistration2 == nil {
			parseServiceWorkerStatus.Set("Service worker registration is not ready yet.")
			return
		}
		parseServiceWorkerStatus.Set("Requesting service worker update...")
		go func() {
			if parseUpdateErr := parseRegistration2.Update(context.Background()); parseUpdateErr != nil {
				parseServiceWorkerStatus.Set(describePWAError("Service worker update failed", parseUpdateErr))
				return
			}
			parseServiceWorkerStatus.Set("Service worker update requested.")
			applyServiceWorkerSnapshot(parseRegistration2.Snapshot())
		}()
	})

	parseState := parseInstallState.Get()
	return shared.ExamplePage(
		"PWA installability",
		"pwa.ObserveInstallability, pwa.RegisterServiceWorker",
		"Observe manifest validity, install-prompt readiness, and service-worker lifecycle without hiding PWA ownership behind runtime side effects.",
		shared.ExamplePanel("Live installability state",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("This example links a real web manifest, registers a scoped service worker, and then surfaces the current installability state through pwa.ObserveInstallability(...).")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Refresh installability", parseRefreshInstallability),
				shared.ExampleButton("Prompt install", parsePromptInstall),
				shared.ExampleButton("Update service worker", parseUpdateServiceWorker),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-3 md:grid-cols-3"},
				shared.ExampleStat("Manifest valid", boolLabel(parseState.ManifestValid)),
				shared.ExampleStat("Prompt available", boolLabel(parseState.PromptAvailable)),
				shared.ExampleStat("Installed", boolLabel(parseState.Installed)),
			),
			html.P(html.Props{Class: "mt-4 text-sm text-slate-300", ID: "pwa-installability-reasons"}, html.Text(formatReasons(parseState.Reasons))),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "pwa-installability-prompt-status"}, html.Text(parsePromptStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "pwa-installability-sw-status"}, html.Text(parseServiceWorkerStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "pwa-installability-sw-snapshot"}, html.Text(parseServiceWorkerSnapshot.Get())),
		),
		shared.ExamplePanel("Manifest wiring",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Keep the manifest app-owned and explicit. The same structure below is used for validation in Go and linked from the HTML entrypoint for the browser.")),
			html.Pre(html.Props{Class: "mt-4 overflow-x-auto rounded-2xl border border-white/10 bg-black/40 p-4 text-sm text-slate-300", ID: "pwa-installability-manifest-preview"}, html.Text(parseManifestPreview)),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(installabilityExample))
	exampleboot.WaitExampleRuntime()
}
