//go:build js && wasm
// +build js,wasm

package pwa

import (
	"context"
	"syscall/js"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/interop"
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

// TestServiceWorkerRegistrationWasmCoversLifecycleReloadAndErrorGuards verifies the remaining registration APIs cover lifecycle subscriptions, reload wiring, and validation failures.
func TestServiceWorkerRegistrationWasmCoversLifecycleReloadAndErrorGuards(parseT *testing.T) {
	parseRestore := installMockServiceWorkerEnvironment(parseT, false)
	defer parseRestore()

	parseRegistration, parseErr := RegisterServiceWorker(context.Background(), ServiceWorkerOptions{URL: " /sw.js ", Scope: " /app ", Type: " module ", UpdateViaCache: " all "})
	if parseErr != nil {
		parseT.Fatalf("expected service worker registration, got %v", parseErr)
	}
	if parseErr = parseRegistration.Update(context.Background()); parseErr != nil {
		parseT.Fatalf("expected update to succeed, got %v", parseErr)
	}
	if parseRemoved, parseErr := parseRegistration.Unregister(context.Background()); parseErr != nil || !parseRemoved {
		parseT.Fatalf("expected unregister to resolve true, got removed=%t err=%v", parseRemoved, parseErr)
	}

	parseSnapshots := []ServiceWorkerSnapshot{}
	parseSub, parseErr := parseRegistration.SubscribeLifecycle(func(parseSnapshot ServiceWorkerSnapshot) {
		parseSnapshots = append(parseSnapshots, parseSnapshot)
	})
	if parseErr != nil {
		parseT.Fatalf("expected lifecycle subscription, got %v", parseErr)
	}
	defer parseSub.Cancel()
	if len(parseSnapshots) == 0 || parseSnapshots[0].Scope != "/app" {
		parseT.Fatalf("expected immediate lifecycle snapshot, got %+v", parseSnapshots)
	}

	parseWaiting := js.Global().Get("__pwaTestWaitingWorker")
	parseWaiting.Set("state", string(ServiceWorkerStateActivated))
	js.Global().Get("__pwaTestServiceWorkerRegistration").Call("dispatchEvent", map[string]interface{}{"type": "updatefound"})
	parseWaiting.Call("dispatchEvent", map[string]interface{}{"type": "statechange"})
	if len(parseSnapshots) < 3 {
		parseT.Fatalf("expected lifecycle notifications for updatefound and statechange, got %d snapshots", len(parseSnapshots))
	}

	parseReloadCount := 0
	parseReload := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseReloadCount++
		return nil
	})
	defer parseReload.Release()
	js.Global().Get("window").Get("location").Set("reload", parseReload)
	parseReloadSub, parseErr := parseRegistration.ReloadOnControllerChange()
	if parseErr != nil {
		parseT.Fatalf("expected reload-on-controller-change subscription, got %v", parseErr)
	}
	js.Global().Get("__pwaTestServiceWorkerContainer").Call("dispatchEvent", map[string]interface{}{"type": "controllerchange"})
	if parseReloadCount != 1 {
		parseT.Fatalf("expected controllerchange reload, got %d reloads", parseReloadCount)
	}
	parseReloadSub.Cancel()

	if parseErr = parseRegistration.RegisterSync(context.Background(), " "); !interop.IsCode(parseErr, interop.CodeInvalid) {
		parseT.Fatalf("expected empty sync tag to fail validation, got %v", parseErr)
	}
	if _, parseErr = parseRegistration.SubscribeLifecycle(nil); !interop.IsCode(parseErr, interop.CodeInvalid) {
		parseT.Fatalf("expected nil lifecycle handler validation error, got %v", parseErr)
	}
	js.Global().Get("__pwaTestServiceWorkerRegistration").Set("sync", js.Undefined())
	if parseErr = parseRegistration.RegisterSync(context.Background(), "offline-demo-replay"); !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseT.Fatalf("expected missing sync manager to be unavailable, got %v", parseErr)
	}
	parseSyncManager := js.Global().Get("Object").New()
	parseSyncManager.Set("register", js.Undefined())
	js.Global().Get("__pwaTestServiceWorkerRegistration").Set("sync", parseSyncManager)
	if parseErr = parseRegistration.RegisterSync(context.Background(), "offline-demo-replay"); !interop.IsCode(parseErr, interop.CodeNotFunction) {
		parseT.Fatalf("expected non-callable sync.register to fail, got %v", parseErr)
	}
	js.Global().Get("__pwaTestServiceWorkerRegistration").Set("waiting", js.Undefined())
	if parseErr = parseRegistration.SkipWaiting(context.Background()); !interop.IsCode(parseErr, interop.CodeInvalid) {
		parseT.Fatalf("expected missing waiting worker to fail, got %v", parseErr)
	}
	parseBrokenWaiting := js.Global().Get("Object").New()
	parseBrokenWaiting.Set("postMessage", js.Undefined())
	js.Global().Get("__pwaTestServiceWorkerRegistration").Set("waiting", parseBrokenWaiting)
	if parseErr = parseRegistration.SkipWaiting(context.Background()); !interop.IsCode(parseErr, interop.CodeNotFunction) {
		parseT.Fatalf("expected non-callable waiting.postMessage to fail, got %v", parseErr)
	}
	js.Global().Get("window").Get("location").Set("reload", js.Undefined())
	if _, parseErr = parseRegistration.ReloadOnControllerChange(); !interop.IsCode(parseErr, interop.CodeNotFunction) {
		parseT.Fatalf("expected non-callable location.reload to fail, got %v", parseErr)
	}
}

// TestRegisterServiceWorkerWasmErrorsAndPromiseGuards verifies registration validation, container lookup failures, and promise await helpers surface structured errors.
func TestRegisterServiceWorkerWasmErrorsAndPromiseGuards(parseT *testing.T) {
	if _, parseErr := RegisterServiceWorker(context.Background(), ServiceWorkerOptions{}); !interop.IsCode(parseErr, interop.CodeInvalid) {
		parseT.Fatalf("expected empty URL validation error, got %v", parseErr)
	}

	parseRestoreNavigator := setPWAServiceWorkerGlobal("navigator", js.Undefined())
	parseRestoreWindow := setPWAServiceWorkerGlobal("window", js.Undefined())
	if _, parseErr := RegisterServiceWorker(context.Background(), ServiceWorkerOptions{URL: "/sw.js"}); !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseT.Fatalf("expected unavailable navigator.serviceWorker error, got %v", parseErr)
	}
	parseRestoreNavigator()
	parseRestoreWindow()

	parseRestore := installMockServiceWorkerEnvironment(parseT, false)
	defer parseRestore()
	js.Global().Get("__pwaTestServiceWorkerContainer").Set("register", js.Undefined())
	if _, parseErr := RegisterServiceWorker(context.Background(), ServiceWorkerOptions{URL: "/sw.js"}); !interop.IsCode(parseErr, interop.CodeNotFunction) {
		parseT.Fatalf("expected non-callable register error, got %v", parseErr)
	}

	parseValue, parseErr := awaitServiceWorkerValue(context.Background(), "Await", "target", js.ValueOf("ready"))
	if parseErr != nil || parseValue.String() != "ready" {
		parseT.Fatalf("expected non-promise values to return immediately, got value=%v err=%v", parseValue, parseErr)
	}

	parseRejectedPromise := js.Global().Get("Promise").Call("reject", "permission denied")
	_, parseErr = awaitServiceWorkerValue(context.Background(), "Await", "target", parseRejectedPromise)
	if !interop.IsCode(parseErr, interop.CodePromiseRejected) {
		parseT.Fatalf("expected rejected promise error, got %v", parseErr)
	}

	parsePendingExecutor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return nil
	})
	defer parsePendingExecutor.Release()
	parsePendingPromise := js.Global().Get("Promise").New(parsePendingExecutor)
	parseCtx, parseCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer parseCancel()
	_, parseErr = awaitServiceWorkerValue(parseCtx, "Await", "target", parsePendingPromise)
	if !interop.IsCode(parseErr, interop.CodeTimeout) {
		parseT.Fatalf("expected timed out promise await error, got %v", parseErr)
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
	parseRestoreContainer := setPWAServiceWorkerGlobal("__pwaTestServiceWorkerContainer", parseContainer)
	parseRestoreRegistration := setPWAServiceWorkerGlobal("__pwaTestServiceWorkerRegistration", parseRegistration)
	parseRestoreWaiting := setPWAServiceWorkerGlobal("__pwaTestWaitingWorker", parseWaiting)
	return func() {
		parseRestoreNavigator()
		parseRestoreWindow()
		parseRestoreContainer()
		parseRestoreRegistration()
		parseRestoreWaiting()
	}
}
