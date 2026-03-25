//go:build js && wasm
// +build js,wasm

package pwa

import (
	"context"
	"syscall/js"
	"testing"
)

func setPWAServiceWorkerGlobal(parseName string, parseValue interface{}) func() {
	parseGlobal := js.Global()
	parsePrevious := parseGlobal.Get(parseName)
	parseGlobal.Set(parseName, parseValue)
	return func() {
		parseGlobal.Set(parseName, parsePrevious)
	}
}

func TestRegisterServiceWorkerReturnsLifecycleSnapshot(parseT *testing.T) {
	parseRestore := installMockServiceWorkerEnvironment(parseT, true)
	defer parseRestore()

	parseRegistration, parseErr := RegisterServiceWorker(context.Background(), ServiceWorkerOptions{URL: "/sw.js", Scope: "/app"})
	if parseErr != nil {
		parseT.Fatalf("expected service worker registration, got %v", parseErr)
	}
	parseSnapshot := parseRegistration.Snapshot()
	if parseSnapshot.Scope != "/app" {
		parseT.Fatalf("expected scope /app, got %q", parseSnapshot.Scope)
	}
	if parseSnapshot.Waiting.ScriptURL != "/sw.js" {
		parseT.Fatalf("expected waiting worker script, got %+v", parseSnapshot.Waiting)
	}
	if parseErr2 := parseRegistration.SkipWaiting(context.Background()); parseErr2 != nil {
		parseT.Fatalf("expected skip waiting to succeed, got %v", parseErr2)
	}
	if parseSnapshot2 := parseRegistration.Snapshot(); parseSnapshot2.Waiting.ScriptURL != "/sw.js" {
		parseT.Fatalf("expected waiting worker to remain addressable, got %+v", parseSnapshot2)
	}
	parseCapabilities := parseRegistration.BackgroundSyncCapabilities()
	if !parseCapabilities.OneShot || parseCapabilities.Periodic {
		parseT.Fatalf("expected one-shot background sync support only, got %+v", parseCapabilities)
	}
	if parseErr3 := parseRegistration.RegisterSync(context.Background(), "offline-demo-replay"); parseErr3 != nil {
		parseT.Fatalf("expected background sync registration to succeed, got %v", parseErr3)
	}
}

func installMockServiceWorkerEnvironment(parseT *testing.T, isAssertSyncRegistration bool) func() {
	parseT.Helper()
	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parseMakePromise := func(parseValue js.Value) js.Value {
		return parseGlobal.Get("Promise").Call("resolve", parseValue)
	}
	parseMakeEventTarget := func() js.Value {
		parseTarget := parseObjectCtor.New()
		parseListeners := map[string][]js.Value{}
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
		parseTarget.Set("addEventListener", parseAdd)
		parseTarget.Set("removeEventListener", parseRemove)
		parseTarget.Set("dispatchEvent", parseDispatch)
		parseT.Cleanup(func() {
			parseAdd.Release()
			parseRemove.Release()
			parseDispatch.Release()
		})
		return parseTarget
	}
	parseWaiting := parseMakeEventTarget()
	parseWaiting.Set("scriptURL", "/sw.js")
	parseWaiting.Set("state", "installed")
	parsePostMessage := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		return nil
	})
	parseWaiting.Set("postMessage", parsePostMessage)
	parseT.Cleanup(parsePostMessage.Release)

	parseRegistration := parseMakeEventTarget()
	parseRegistration.Set("scope", "/app")
	parseRegistration.Set("waiting", parseWaiting)
	parseSyncManager := parseObjectCtor.New()
	parseRegisteredTags := []string{}
	parseRegisterSyncFn := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		parseRegisteredTags = append(parseRegisteredTags, parseArgs5[0].String())
		return parseMakePromise(js.Undefined())
	})
	parseSyncManager.Set("register", parseRegisterSyncFn)
	parseRegistration.Set("sync", parseSyncManager)
	parseUpdateFn := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} { return parseMakePromise(js.Undefined()) })
	parseUnregisterFn := js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} {
		return parseMakePromise(js.ValueOf(true))
	})
	parseRegistration.Set("update", parseUpdateFn)
	parseRegistration.Set("unregister", parseUnregisterFn)
	parseT.Cleanup(func() {
		parseRegisterSyncFn.Release()
		parseUpdateFn.Release()
		parseUnregisterFn.Release()
		if !isAssertSyncRegistration {
			return
		}
		if len(parseRegisteredTags) != 1 || parseRegisteredTags[0] != "offline-demo-replay" {
			parseT.Fatalf("expected sync.register to receive the replay tag, got %#v", parseRegisteredTags)
		}
	})

	parseContainer := parseMakeEventTarget()
	parseContainer.Set("controller", parseObjectCtor.New())
	parseRegister := js.FuncOf(func(parseThis8 js.Value, parseArgs8 []js.Value) interface{} {
		if len(parseArgs8) > 1 && parseArgs8[1].Truthy() && !parseArgs8[1].Get("scope").IsUndefined() {
			parseRegistration.Set("scope", parseArgs8[1].Get("scope").String())
		}
		parseRegistration.Set("waiting", parseWaiting)
		return parseMakePromise(parseRegistration)
	})
	parseContainer.Set("register", parseRegister)
	parseT.Cleanup(parseRegister.Release)

	parseNavigator := parseObjectCtor.New()
	parseNavigator.Set("serviceWorker", parseContainer)
	parseWindow := parseObjectCtor.New()
	parseWindow.Set("navigator", parseNavigator)
	parseLocation := parseObjectCtor.New()
	parseReload := js.FuncOf(func(parseThis9 js.Value, parseArgs9 []js.Value) interface{} { return nil })
	parseLocation.Set("reload", parseReload)
	parseWindow.Set("location", parseLocation)
	parseT.Cleanup(parseReload.Release)

	parseRestoreNavigator := setPWAServiceWorkerGlobal("navigator", parseNavigator)
	parseRestoreWindow := setPWAServiceWorkerGlobal("window", parseWindow)
	return func() {
		parseRestoreNavigator()
		parseRestoreWindow()
	}
}
