//go:build js && wasm
// +build js,wasm

package pwa

import (
	"context"
	"syscall/js"
	"testing"
)

func TestObserveInstallabilityTracksPromptAvailabilityAndInstall(t *testing.T) {
	window, restoreWindow := installMockInstallabilityWindow(t)
	defer restoreWindow()

	manager, err := ObserveInstallability(InstallabilityOptions{Manifest: &Manifest{Name: "Atlas", StartURL: "/"}})
	if err != nil {
		t.Fatalf("expected installability manager, got %v", err)
	}
	var snapshots []InstallabilityState
	sub, err := manager.Subscribe(func(state InstallabilityState) {
		snapshots = append(snapshots, state)
	})
	if err != nil {
		t.Fatalf("expected installability subscription, got %v", err)
	}
	defer sub.Cancel()

	beforeEvent := js.Global().Get("Object").New()
	beforeEvent.Set("type", "beforeinstallprompt")
	preventDefault := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	defer preventDefault.Release()
	beforeEvent.Set("preventDefault", preventDefault)
	prompt := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return js.Global().Get("Promise").Call("resolve", js.Undefined())
	})
	defer prompt.Release()
	beforeEvent.Set("prompt", prompt)
	choice := js.Global().Get("Object").New()
	choice.Set("outcome", "accepted")
	choice.Set("platform", "web")
	beforeEvent.Set("userChoice", js.Global().Get("Promise").Call("resolve", choice))
	window.Call("dispatchEvent", beforeEvent)

	state := manager.State()
	if !state.PromptAvailable || !state.ManifestValid {
		t.Fatalf("expected prompt availability after beforeinstallprompt, got %+v", state)
	}
	result, err := manager.Prompt(context.Background())
	if err != nil {
		t.Fatalf("expected prompt to succeed, got %v", err)
	}
	if result.Outcome != "accepted" || result.Platform != "web" {
		t.Fatalf("unexpected prompt result: %+v", result)
	}

	installedEvent := js.Global().Get("Object").New()
	installedEvent.Set("type", "appinstalled")
	window.Call("dispatchEvent", installedEvent)
	if !manager.State().Installed {
		t.Fatalf("expected installed state after appinstalled event, got %+v", manager.State())
	}
	if len(snapshots) < 3 {
		t.Fatalf("expected multiple installability snapshots, got %d", len(snapshots))
	}
}

func installMockInstallabilityWindow(t *testing.T) (js.Value, func()) {
	t.Helper()
	global := js.Global()
	objectCtor := global.Get("Object")
	listeners := map[string][]js.Value{}
	window := objectCtor.New()
	add := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		eventName := args[0].String()
		listeners[eventName] = append(listeners[eventName], args[1])
		return nil
	})
	remove := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		eventName := args[0].String()
		remaining := listeners[eventName][:0]
		for _, current := range listeners[eventName] {
			if !current.Equal(args[1]) {
				remaining = append(remaining, current)
			}
		}
		listeners[eventName] = remaining
		return nil
	})
	dispatch := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		eventName := args[0].Get("type").String()
		for _, listener := range listeners[eventName] {
			listener.Invoke(args[0])
		}
		return true
	})
	matchMedia := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		result := objectCtor.New()
		result.Set("matches", false)
		return result
	})
	window.Set("addEventListener", add)
	window.Set("removeEventListener", remove)
	window.Set("dispatchEvent", dispatch)
	window.Set("isSecureContext", true)
	window.Set("matchMedia", matchMedia)
	navigator := objectCtor.New()
	navigator.Set("standalone", false)
	window.Set("navigator", navigator)
	restoreWindow := setPWAServiceWorkerGlobal("window", window)
	restoreNavigator := setPWAServiceWorkerGlobal("navigator", navigator)
	return window, func() {
		restoreWindow()
		restoreNavigator()
		add.Release()
		remove.Release()
		dispatch.Release()
		matchMedia.Release()
	}
}
