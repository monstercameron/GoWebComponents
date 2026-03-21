//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"strings"

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

func describePWAError(prefix string, err error) string {
	if err == nil {
		return prefix
	}
	if code, ok := interop.CodeOf(err); ok {
		return fmt.Sprintf("%s [%s]: %v", prefix, code, err)
	}
	return fmt.Sprintf("%s: %v", prefix, err)
}

func boolLabel(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func formatReasons(reasons []string) string {
	if len(reasons) == 0 {
		return "No blockers currently reported by ObserveInstallability()."
	}
	return strings.Join(reasons, " | ")
}

func formatServiceWorkerSnapshot(snapshot pwa.ServiceWorkerSnapshot) string {
	parts := make([]string, 0, 4)
	if snapshot.Scope != "" {
		parts = append(parts, "scope="+snapshot.Scope)
	}
	parts = append(parts, "controller="+boolLabel(snapshot.HasController))
	if snapshot.Active.ScriptURL != "" {
		parts = append(parts, "active="+string(snapshot.Active.State)+" @ "+snapshot.Active.ScriptURL)
	}
	if snapshot.Waiting.ScriptURL != "" {
		parts = append(parts, "waiting="+string(snapshot.Waiting.State)+" @ "+snapshot.Waiting.ScriptURL)
	}
	if snapshot.Installing.ScriptURL != "" {
		parts = append(parts, "installing="+string(snapshot.Installing.State)+" @ "+snapshot.Installing.ScriptURL)
	}
	return strings.Join(parts, " | ")
}

func installabilityExample() ui.Node {
	manifest := installabilityManifest()
	manifestJSON, err := pwa.MarshalManifestJSONIndented(manifest, "", "  ")
	manifestPreview := "manifest preview unavailable"
	if err == nil {
		manifestPreview = string(manifestJSON)
	}
	initialInstallability := pwa.InstallabilityState{ManifestValid: true}
	if validateErr := manifest.Validate(); validateErr != nil {
		initialInstallability.ManifestValid = false
		initialInstallability.ManifestError = validateErr.Error()
		initialInstallability.Reasons = []string{validateErr.Error()}
	} else {
		initialInstallability.Reasons = []string{"Waiting for browser installability signals."}
	}

	managerRef := ui.UseRef[*pwa.InstallabilityManager](nil)
	registrationRef := ui.UseRef[*pwa.ServiceWorkerRegistration](nil)
	serviceWorkerStartedRef := ui.UseRef(false)
	serviceWorkerCancelRef := ui.UseRef[func()](nil)
	installState := ui.UseState(initialInstallability)
	promptStatus := ui.UseState("Install prompt idle.")
	serviceWorkerStatus := ui.UseState("Service worker registration pending.")
	serviceWorkerSnapshot := ui.UseState("No service worker snapshot yet.")

	applyInstallabilityState := func(state pwa.InstallabilityState) {
		installState.Set(state)
	}
	applyServiceWorkerSnapshot := func(snapshot pwa.ServiceWorkerSnapshot) {
		formatted := formatServiceWorkerSnapshot(snapshot)
		if strings.TrimSpace(formatted) == "" {
			formatted = "No service worker snapshot yet."
		}
		serviceWorkerSnapshot.Set(formatted)
	}

	ui.UseEffect(func() func() {
		cleanups := make([]func(), 0, 1)
		if managerRef.Get() == nil {
			manager, observeErr := pwa.ObserveInstallability(pwa.InstallabilityOptions{Manifest: &manifest})
			if observeErr != nil {
				promptStatus.Set(describePWAError("Installability observation unavailable", observeErr))
			} else {
				managerRef.Set(&manager)
				applyInstallabilityState(manager.State())
				subscription, subscribeErr := manager.Subscribe(func(state pwa.InstallabilityState) {
					applyInstallabilityState(state)
				})
				if subscribeErr == nil {
					cleanups = append(cleanups, subscription.Cancel)
				}
			}
		}
		return func() {
			for _, cancel := range cleanups {
				cancel()
			}
		}
	}, "installability-observer")

	ui.UseEffect(func() func() {
		if registrationRef.Get() == nil && !serviceWorkerStartedRef.Get() {
			serviceWorkerStartedRef.Set(true)
			serviceWorkerStatus.Set("Registering service worker...")
			go func() {
				registration, registerErr := pwa.RegisterServiceWorker(context.Background(), pwa.ServiceWorkerOptions{
					URL:   "/97-pwa-installability/sw.js",
					Scope: "/97-pwa-installability/",
				})
				if registerErr != nil {
					serviceWorkerStatus.Set(describePWAError("Service worker registration failed", registerErr))
					return
				}
				registrationRef.Set(&registration)
				serviceWorkerStatus.Set("Service worker registered through pwa.RegisterServiceWorker(...).")
				applyServiceWorkerSnapshot(registration.Snapshot())
				subscription, subscribeErr := registration.SubscribeLifecycle(func(snapshot pwa.ServiceWorkerSnapshot) {
					applyServiceWorkerSnapshot(snapshot)
					serviceWorkerStatus.Set("Service worker lifecycle changed.")
				})
				if subscribeErr == nil {
					serviceWorkerCancelRef.Set(subscription.Cancel)
				}
			}()
		}
		return func() {
			if cancel := serviceWorkerCancelRef.Get(); cancel != nil {
				cancel()
				serviceWorkerCancelRef.Set(nil)
			}
		}
	}, "installability-service-worker")

	refreshInstallability := ui.UseEvent(func() {
		if manager := managerRef.Get(); manager != nil {
			applyInstallabilityState(manager.State())
			promptStatus.Set("Installability state refreshed from ObserveInstallability().")
		}
	})
	promptInstall := ui.UseEvent(func() {
		manager := managerRef.Get()
		if manager == nil {
			promptStatus.Set("Installability manager is not ready yet.")
			return
		}
		promptStatus.Set("Requesting install prompt...")
		go func() {
			result, promptErr := manager.Prompt(context.Background())
			if promptErr != nil {
				promptStatus.Set(describePWAError("Install prompt unavailable", promptErr))
				return
			}
			promptStatus.Set(fmt.Sprintf("Browser prompt outcome=%s platform=%s", result.Outcome, result.Platform))
		}()
	})
	updateServiceWorker := ui.UseEvent(func() {
		registration := registrationRef.Get()
		if registration == nil {
			serviceWorkerStatus.Set("Service worker registration is not ready yet.")
			return
		}
		serviceWorkerStatus.Set("Requesting service worker update...")
		go func() {
			if updateErr := registration.Update(context.Background()); updateErr != nil {
				serviceWorkerStatus.Set(describePWAError("Service worker update failed", updateErr))
				return
			}
			serviceWorkerStatus.Set("Service worker update requested.")
			applyServiceWorkerSnapshot(registration.Snapshot())
		}()
	})

	state := installState.Get()
	return shared.ExamplePage(
		"PWA installability",
		"pwa.ObserveInstallability, pwa.RegisterServiceWorker",
		"Observe manifest validity, install-prompt readiness, and service-worker lifecycle without hiding PWA ownership behind runtime side effects.",
		shared.ExamplePanel("Live installability state",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("This example links a real web manifest, registers a scoped service worker, and then surfaces the current installability state through pwa.ObserveInstallability(...).")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Refresh installability", refreshInstallability),
				shared.ExampleButton("Prompt install", promptInstall),
				shared.ExampleButton("Update service worker", updateServiceWorker),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-3 md:grid-cols-3"},
				shared.ExampleStat("Manifest valid", boolLabel(state.ManifestValid)),
				shared.ExampleStat("Prompt available", boolLabel(state.PromptAvailable)),
				shared.ExampleStat("Installed", boolLabel(state.Installed)),
			),
			html.P(html.Props{Class: "mt-4 text-sm text-slate-300", ID: "pwa-installability-reasons"}, html.Text(formatReasons(state.Reasons))),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "pwa-installability-prompt-status"}, html.Text(promptStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "pwa-installability-sw-status"}, html.Text(serviceWorkerStatus.Get())),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300", ID: "pwa-installability-sw-snapshot"}, html.Text(serviceWorkerSnapshot.Get())),
		),
		shared.ExamplePanel("Manifest wiring",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Keep the manifest app-owned and explicit. The same structure below is used for validation in Go and linked from the HTML entrypoint for the browser.")),
			html.Pre(html.Props{Class: "mt-4 overflow-x-auto rounded-2xl border border-white/10 bg-black/40 p-4 text-sm text-slate-300", ID: "pwa-installability-manifest-preview"}, html.Text(manifestPreview)),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(installabilityExample), "#app")
	select {}
}
