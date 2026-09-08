//go:build js && wasm

package pwa

import (
	"context"
	"syscall/js"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

func TestObserveInstallabilityTracksPromptAvailabilityAndInstall(parseT *testing.T) {
	parseWindow, parseRestoreWindow := installMockInstallabilityWindow(parseT)
	defer parseRestoreWindow()

	parseManager, parseErr := ObserveInstallability(InstallabilityOptions{Manifest: &Manifest{Name: "Atlas", StartURL: "/"}})
	if parseErr != nil {
		parseT.Fatalf("expected installability manager, got %v", parseErr)
	}
	var parseSnapshots []InstallabilityState
	parseSub, parseErr := parseManager.Subscribe(func(parseState2 InstallabilityState) {
		parseSnapshots = append(parseSnapshots, parseState2)
	})
	if parseErr != nil {
		parseT.Fatalf("expected installability subscription, got %v", parseErr)
	}
	defer parseSub.Cancel()

	parseBeforeEvent := js.Global().Get("Object").New()
	parseBeforeEvent.Set("type", "beforeinstallprompt")
	parsePreventDefault := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} { return nil })
	defer parsePreventDefault.Release()
	parseBeforeEvent.Set("preventDefault", parsePreventDefault)
	parsePrompt := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		return js.Global().Get("Promise").Call("resolve", js.Undefined())
	})
	defer parsePrompt.Release()
	parseBeforeEvent.Set("prompt", parsePrompt)
	parseChoice := js.Global().Get("Object").New()
	parseChoice.Set("outcome", "accepted")
	parseChoice.Set("platform", "web")
	parseBeforeEvent.Set("userChoice", js.Global().Get("Promise").Call("resolve", parseChoice))
	parseWindow.Call("dispatchEvent", parseBeforeEvent)

	parseState := parseManager.State()
	if !parseState.PromptAvailable || !parseState.ManifestValid {
		parseT.Fatalf("expected prompt availability after beforeinstallprompt, got %+v", parseState)
	}
	parseResult, parseErr := parseManager.Prompt(context.Background())
	if parseErr != nil {
		parseT.Fatalf("expected prompt to succeed, got %v", parseErr)
	}
	if parseResult.Outcome != "accepted" || parseResult.Platform != "web" {
		parseT.Fatalf("unexpected prompt result: %+v", parseResult)
	}

	parseInstalledEvent := js.Global().Get("Object").New()
	parseInstalledEvent.Set("type", "appinstalled")
	parseWindow.Call("dispatchEvent", parseInstalledEvent)
	if !parseManager.State().Installed {
		parseT.Fatalf("expected installed state after appinstalled event, got %+v", parseManager.State())
	}
	if len(parseSnapshots) < 3 {
		parseT.Fatalf("expected multiple installability snapshots, got %d", len(parseSnapshots))
	}
}

// TestInstallabilityWasmErrorsAndPromiseGuards verifies unavailable, validation, and promise-await error handling around installability helpers.
func TestInstallabilityWasmErrorsAndPromiseGuards(parseT *testing.T) {
	parseRestoreWindow := setPWAServiceWorkerGlobal("window", js.Undefined())
	if _, parseErr := ObserveInstallability(InstallabilityOptions{}); !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseT.Fatalf("expected unavailable window error, got %v", parseErr)
	}
	parseRestoreWindow()

	_, parseRestore := installMockInstallabilityWindow(parseT)
	defer parseRestore()

	parseManager, parseErr := ObserveInstallability(InstallabilityOptions{})
	if parseErr != nil {
		parseT.Fatalf("expected installability manager, got %v", parseErr)
	}
	if _, parseErr = parseManager.Subscribe(nil); !interop.IsCode(parseErr, interop.CodeInvalid) {
		parseT.Fatalf("expected nil subscription handler validation error, got %v", parseErr)
	}
	if _, parseErr = parseManager.Prompt(context.Background()); !interop.IsCode(parseErr, interop.CodeInvalid) {
		parseT.Fatalf("expected missing prompt validation error, got %v", parseErr)
	}

	parseValue, parseErr := awaitInstallabilityValue(context.Background(), "Await", "target", js.ValueOf("ready"))
	if parseErr != nil || parseValue.String() != "ready" {
		parseT.Fatalf("expected non-promise installability values to return immediately, got value=%v err=%v", parseValue, parseErr)
	}

	parseRejectedPromise := js.Global().Get("Promise").Call("reject", "prompt denied")
	_, parseErr = awaitInstallabilityValue(context.Background(), "Await", "target", parseRejectedPromise)
	if !interop.IsCode(parseErr, interop.CodePromiseRejected) {
		parseT.Fatalf("expected rejected installability promise error, got %v", parseErr)
	}

	parsePendingExecutor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return nil
	})
	defer parsePendingExecutor.Release()
	parsePendingPromise := js.Global().Get("Promise").New(parsePendingExecutor)
	parseCtx, parseCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer parseCancel()
	_, parseErr = awaitInstallabilityValue(parseCtx, "Await", "target", parsePendingPromise)
	if !interop.IsCode(parseErr, interop.CodeTimeout) {
		parseT.Fatalf("expected timed out installability promise error, got %v", parseErr)
	}
}

func installMockInstallabilityWindow(parseT *testing.T) (js.Value, func()) {
	parseT.Helper()
	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parseListeners := map[string][]js.Value{}
	parseWindow := parseObjectCtor.New()
	parseAdd := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseEventName := parseArgs[0].String()
		parseListeners[parseEventName] = append(parseListeners[parseEventName], parseArgs[1])
		return nil
	})
	parseRemove := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseEventName2 := parseArgs2[0].String()
		parseRemaining := parseListeners[parseEventName2][:0]
		for _, parseCurrent := range parseListeners[parseEventName2] {
			if !parseCurrent.Equal(parseArgs2[1]) {
				parseRemaining = append(parseRemaining, parseCurrent)
			}
		}
		parseListeners[parseEventName2] = parseRemaining
		return nil
	})
	parseDispatch := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		parseEventName3 := parseArgs3[0].Get("type").String()
		for _, parseListener := range parseListeners[parseEventName3] {
			parseListener.Invoke(parseArgs3[0])
		}
		return true
	})
	parseMatchMedia := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		parseResult := parseObjectCtor.New()
		parseResult.Set("matches", false)
		return parseResult
	})
	parseWindow.Set("addEventListener", parseAdd)
	parseWindow.Set("removeEventListener", parseRemove)
	parseWindow.Set("dispatchEvent", parseDispatch)
	parseWindow.Set("isSecureContext", true)
	parseWindow.Set("matchMedia", parseMatchMedia)
	parseNavigator := parseObjectCtor.New()
	parseNavigator.Set("standalone", false)
	parseWindow.Set("navigator", parseNavigator)
	parseRestoreWindow := setPWAServiceWorkerGlobal("window", parseWindow)
	parseRestoreNavigator := setPWAServiceWorkerGlobal("navigator", parseNavigator)
	return parseWindow, func() {
		parseRestoreWindow()
		parseRestoreNavigator()
		parseAdd.Release()
		parseRemove.Release()
		parseDispatch.Release()
		parseMatchMedia.Release()
	}
}
