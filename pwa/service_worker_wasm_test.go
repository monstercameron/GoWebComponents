//go:build js && wasm
// +build js,wasm

package pwa

import (
	"context"
	"syscall/js"
	"testing"
)

func setPWAServiceWorkerGlobal(name string, value interface{}) func() {
	global := js.Global()
	previous := global.Get(name)
	global.Set(name, value)
	return func() {
		global.Set(name, previous)
	}
}

func TestRegisterServiceWorkerReturnsLifecycleSnapshot(t *testing.T) {
	restore := installMockServiceWorkerEnvironment(t)
	defer restore()

	registration, err := RegisterServiceWorker(context.Background(), ServiceWorkerOptions{URL: "/sw.js", Scope: "/app"})
	if err != nil {
		t.Fatalf("expected service worker registration, got %v", err)
	}
	snapshot := registration.Snapshot()
	if snapshot.Scope != "/app" {
		t.Fatalf("expected scope /app, got %q", snapshot.Scope)
	}
	if snapshot.Waiting.ScriptURL != "/sw.js" {
		t.Fatalf("expected waiting worker script, got %+v", snapshot.Waiting)
	}
	if err := registration.SkipWaiting(context.Background()); err != nil {
		t.Fatalf("expected skip waiting to succeed, got %v", err)
	}
	if snapshot := registration.Snapshot(); snapshot.Waiting.ScriptURL != "/sw.js" {
		t.Fatalf("expected waiting worker to remain addressable, got %+v", snapshot)
	}
	capabilities := registration.BackgroundSyncCapabilities()
	if !capabilities.OneShot || capabilities.Periodic {
		t.Fatalf("expected one-shot background sync support only, got %+v", capabilities)
	}
	if err := registration.RegisterSync(context.Background(), "offline-demo-replay"); err != nil {
		t.Fatalf("expected background sync registration to succeed, got %v", err)
	}
}

func installMockServiceWorkerEnvironment(t *testing.T) func() {
	t.Helper()
	global := js.Global()
	objectCtor := global.Get("Object")
	makePromise := func(value js.Value) js.Value {
		return global.Get("Promise").Call("resolve", value)
	}
	makeEventTarget := func() js.Value {
		target := objectCtor.New()
		listeners := map[string][]js.Value{}
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
		target.Set("addEventListener", add)
		target.Set("removeEventListener", remove)
		target.Set("dispatchEvent", dispatch)
		t.Cleanup(func() {
			add.Release()
			remove.Release()
			dispatch.Release()
		})
		return target
	}
	waiting := makeEventTarget()
	waiting.Set("scriptURL", "/sw.js")
	waiting.Set("state", "installed")
	postMessage := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return nil
	})
	waiting.Set("postMessage", postMessage)
	t.Cleanup(postMessage.Release)

	registration := makeEventTarget()
	registration.Set("scope", "/app")
	registration.Set("waiting", waiting)
	syncManager := objectCtor.New()
	registeredTags := []string{}
	registerSyncFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		registeredTags = append(registeredTags, args[0].String())
		return makePromise(js.Undefined())
	})
	syncManager.Set("register", registerSyncFn)
	registration.Set("sync", syncManager)
	updateFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return makePromise(js.Undefined()) })
	unregisterFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return makePromise(js.ValueOf(true)) })
	registration.Set("update", updateFn)
	registration.Set("unregister", unregisterFn)
	t.Cleanup(func() {
		registerSyncFn.Release()
		updateFn.Release()
		unregisterFn.Release()
		if len(registeredTags) != 1 || registeredTags[0] != "offline-demo-replay" {
			t.Fatalf("expected sync.register to receive the replay tag, got %#v", registeredTags)
		}
	})

	container := makeEventTarget()
	container.Set("controller", objectCtor.New())
	register := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 1 && args[1].Truthy() && !args[1].Get("scope").IsUndefined() {
			registration.Set("scope", args[1].Get("scope").String())
		}
		registration.Set("waiting", waiting)
		return makePromise(registration)
	})
	container.Set("register", register)
	t.Cleanup(register.Release)

	navigator := objectCtor.New()
	navigator.Set("serviceWorker", container)
	window := objectCtor.New()
	location := objectCtor.New()
	reload := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	location.Set("reload", reload)
	window.Set("location", location)
	t.Cleanup(reload.Release)

	restoreNavigator := setPWAServiceWorkerGlobal("navigator", navigator)
	restoreWindow := setPWAServiceWorkerGlobal("window", window)
	return func() {
		restoreNavigator()
		restoreWindow()
	}
}
