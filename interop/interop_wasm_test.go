//go:build js && wasm
// +build js,wasm

package interop

import (
	"context"
	"syscall/js"
	"testing"
	"time"
)

func setGlobalValue(name string, value interface{}) func() {
	global := js.Global()
	prev := global.Get(name)
	global.Set(name, value)
	return func() {
		global.Set(name, prev)
	}
}

func makePromise(value js.Value) js.Value {
	return js.Global().Get("Promise").Call("resolve", value)
}

func TestLocalStorageWrapperTracksKeysAndValues(t *testing.T) {
	var keys []string
	values := map[string]string{}
	storage := js.Global().Get("Object").New()
	reindex := func() {
		storage.Set("length", len(keys))
	}

	getItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return js.Null()
		}
		if value, ok := values[args[0].String()]; ok {
			return value
		}
		return js.Null()
	})
	defer getItemFn.Release()
	setItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		key := args[0].String()
		if _, ok := values[key]; !ok {
			keys = append(keys, key)
		}
		values[key] = args[1].String()
		reindex()
		return nil
	})
	defer setItemFn.Release()
	removeItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		key := args[0].String()
		delete(values, key)
		next := keys[:0]
		for _, item := range keys {
			if item != key {
				next = append(next, item)
			}
		}
		keys = next
		reindex()
		return nil
	})
	defer removeItemFn.Release()
	clearFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		values = map[string]string{}
		keys = nil
		reindex()
		return nil
	})
	defer clearFn.Release()
	keyFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		index := args[0].Int()
		if index < 0 || index >= len(keys) {
			return js.Null()
		}
		return keys[index]
	})
	defer keyFn.Release()

	storage.Set("getItem", getItemFn)
	storage.Set("setItem", setItemFn)
	storage.Set("removeItem", removeItemFn)
	storage.Set("clear", clearFn)
	storage.Set("key", keyFn)
	reindex()

	restoreStorage := setGlobalValue("localStorage", storage)
	defer restoreStorage()

	local, err := LocalStorage()
	if err != nil {
		t.Fatalf("expected localStorage wrapper, got %v", err)
	}
	if err := local.SetItem("theme", "dark"); err != nil {
		t.Fatalf("expected set item to succeed, got %v", err)
	}
	value, ok, err := local.GetItem("theme")
	if err != nil || !ok || value != "dark" {
		t.Fatalf("unexpected storage read: value=%q ok=%t err=%v", value, ok, err)
	}
	length, err := local.Len()
	if err != nil || length != 1 {
		t.Fatalf("expected storage len 1, got %d err=%v", length, err)
	}
	key, ok, err := local.Key(0)
	if err != nil || !ok || key != "theme" {
		t.Fatalf("unexpected storage key: key=%q ok=%t err=%v", key, ok, err)
	}
	if err := local.RemoveItem("theme"); err != nil {
		t.Fatalf("expected remove item to succeed, got %v", err)
	}
	if _, ok, err := local.GetItem("theme"); err != nil || ok {
		t.Fatalf("expected removed item to disappear, ok=%t err=%v", ok, err)
	}
}

func TestWindowHistoryPushStateRoundTripsDecodedState(t *testing.T) {
	history := js.Global().Get("Object").New()
	history.Set("length", 2)
	pushStateFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		history.Set("state", args[0])
		return nil
	})
	defer pushStateFn.Release()
	history.Set("pushState", pushStateFn)
	history.Set("replaceState", pushStateFn)
	history.Set("back", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
	history.Set("forward", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
	history.Set("go", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))

	window := js.Global().Get("Object").New()
	window.Set("history", history)
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()

	wrapped, err := WindowHistory()
	if err != nil {
		t.Fatalf("expected history wrapper, got %v", err)
	}
	if err := wrapped.PushState(map[string]any{"step": "upload", "count": 2}, "Upload", "/upload"); err != nil {
		t.Fatalf("expected pushState to succeed, got %v", err)
	}
	state, err := wrapped.State()
	if err != nil {
		t.Fatalf("expected history state, got %v", err)
	}
	payload := map[string]any{}
	if err := Decode(state, &payload); err != nil {
		t.Fatalf("expected decoded history state, got %v", err)
	}
	if payload["step"] != "upload" {
		t.Fatalf("unexpected history payload: %#v", payload)
	}
}

func TestNavigatorClipboardAwaitingPromise(t *testing.T) {
	var written string
	writeTextFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		written = args[0].String()
		return makePromise(js.Undefined())
	})
	defer writeTextFn.Release()
	readTextFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return makePromise(js.ValueOf("copied"))
	})
	defer readTextFn.Release()

	clipboard := js.Global().Get("Object").New()
	clipboard.Set("writeText", writeTextFn)
	clipboard.Set("readText", readTextFn)
	navigator := js.Global().Get("Object").New()
	navigator.Set("clipboard", clipboard)
	window := js.Global().Get("Object").New()
	window.Set("navigator", navigator)
	restoreNavigator := setGlobalValue("navigator", navigator)
	defer restoreNavigator()
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()

	wrapped, err := NavigatorClipboard()
	if err != nil {
		t.Fatalf("expected clipboard wrapper, got %v", err)
	}
	if err := wrapped.WriteText(context.Background(), "hello"); err != nil {
		t.Fatalf("expected writeText to succeed, got %v", err)
	}
	if written != "hello" {
		t.Fatalf("expected clipboard write payload, got %q", written)
	}
	text, err := wrapped.ReadText(context.Background())
	if err != nil || text != "copied" {
		t.Fatalf("expected clipboard read payload, got %q err=%v", text, err)
	}
}

func TestTimersUseBrowserCallbacks(t *testing.T) {
	var timeoutDelay int
	var clearedTimeout int
	setTimeoutFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		timeoutDelay = args[1].Int()
		args[0].Invoke()
		return 7
	})
	defer setTimeoutFn.Release()
	clearTimeoutFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		clearedTimeout++
		return nil
	})
	defer clearTimeoutFn.Release()

	restoreSetTimeout := setGlobalValue("setTimeout", setTimeoutFn)
	defer restoreSetTimeout()
	restoreClearTimeout := setGlobalValue("clearTimeout", clearTimeoutFn)
	defer restoreClearTimeout()

	fired := 0
	timer, err := SetTimeout(25*time.Millisecond, func() { fired++ })
	if err != nil {
		t.Fatalf("expected timeout to succeed, got %v", err)
	}
	if fired != 1 || timeoutDelay != 25 {
		t.Fatalf("unexpected timeout behavior: fired=%d delay=%d", fired, timeoutDelay)
	}
	if err := timer.Cancel(); err != nil {
		t.Fatalf("expected cancel to be safe after fire, got %v", err)
	}
	if clearedTimeout != 0 {
		t.Fatalf("expected fired timeout cancel to be a no-op, got %d clear calls", clearedTimeout)
	}
}

func TestWindowEventsDispatchCustomEvents(t *testing.T) {
	var listener js.Value
	window := js.Global().Get("Object").New()
	addEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 2 {
			listener = args[1]
		}
		return nil
	})
	defer addEventListenerFn.Release()
	removeEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		listener = js.Undefined()
		return nil
	})
	defer removeEventListenerFn.Release()
	dispatchEventFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 && listener.Truthy() {
			listener.Invoke(args[0])
		}
		return true
	})
	defer dispatchEventFn.Release()
	window.Set("addEventListener", addEventListenerFn)
	window.Set("removeEventListener", removeEventListenerFn)
	window.Set("dispatchEvent", dispatchEventFn)

	customEventCtor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := js.Global().Get("Object").New()
		event.Set("type", args[0].String())
		event.Set("detail", args[1].Get("detail"))
		return event
	})
	defer customEventCtor.Release()

	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()
	restoreCustomEvent := setGlobalValue("CustomEvent", customEventCtor)
	defer restoreCustomEvent()

	target, err := WindowEvents()
	if err != nil {
		t.Fatalf("expected window event target, got %v", err)
	}
	var received CustomEvent
	sub, err := target.Subscribe("asset-ready", func(event CustomEvent) {
		received = event
	})
	if err != nil {
		t.Fatalf("expected subscribe to succeed, got %v", err)
	}
	defer sub.Cancel()

	if err := target.Dispatch("asset-ready", map[string]any{"id": "asset-42"}); err != nil {
		t.Fatalf("expected dispatch to succeed, got %v", err)
	}
	payload := map[string]string{}
	if err := Decode(received.Detail, &payload); err != nil {
		t.Fatalf("expected event detail to decode, got %v", err)
	}
	if received.Type != "asset-ready" || payload["id"] != "asset-42" {
		t.Fatalf("unexpected event payload: %+v decoded=%#v", received, payload)
	}
}

func TestMatchMediaSubscriptionReceivesChanges(t *testing.T) {
	var changeListener js.Value
	mediaQuery := js.Global().Get("Object").New()
	mediaQuery.Set("matches", false)
	mediaQuery.Set("media", "(prefers-color-scheme: dark)")
	addEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 2 {
			changeListener = args[1]
		}
		return nil
	})
	defer addEventListenerFn.Release()
	removeEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		changeListener = js.Undefined()
		return nil
	})
	defer removeEventListenerFn.Release()
	mediaQuery.Set("addEventListener", addEventListenerFn)
	mediaQuery.Set("removeEventListener", removeEventListenerFn)

	matchMediaFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return mediaQuery
	})
	defer matchMediaFn.Release()
	window := js.Global().Get("Object").New()
	window.Set("matchMedia", matchMediaFn)
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()

	list, err := MatchMedia("(prefers-color-scheme: dark)")
	if err != nil {
		t.Fatalf("expected media query list, got %v", err)
	}
	var event MediaQueryEvent
	sub, err := list.Subscribe(func(next MediaQueryEvent) {
		event = next
	})
	if err != nil {
		t.Fatalf("expected media query subscribe to succeed, got %v", err)
	}
	defer sub.Cancel()

	change := js.Global().Get("Object").New()
	change.Set("matches", true)
	change.Set("media", "(prefers-color-scheme: dark)")
	changeListener.Invoke(change)
	if !event.Matches || event.Media == "" {
		t.Fatalf("unexpected media query event: %+v", event)
	}
}

func TestImportModuleCallDefaultAndDispose(t *testing.T) {
	sumFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return js.ValueOf(args[0].Float() + args[1].Float())
	})
	defer sumFn.Release()
	defaultFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		payload := js.Global().Get("Object").New()
		payload.Set("ok", true)
		payload.Set("label", args[0].String())
		return payload
	})
	defer defaultFn.Release()

	moduleNS := js.Global().Get("Object").New()
	moduleNS.Set("sum", sumFn)
	moduleNS.Set("default", defaultFn)
	moduleNS.Set("version", "1.0.0")

	importFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return makePromise(moduleNS)
	})
	defer importFn.Release()
	restoreImport := setGlobalValue("__gwcImportModule", importFn)
	defer restoreImport()

	module, err := ImportModule(context.Background(), "/demo/math.js")
	if err != nil {
		t.Fatalf("expected import to succeed, got %v", err)
	}
	sum, err := module.Call(context.Background(), "sum", 2, 3)
	if err != nil {
		t.Fatalf("expected named export call to succeed, got %v", err)
	}
	if sum.(float64) != 5 {
		t.Fatalf("expected module sum result 5, got %#v", sum)
	}
	defaultValue, err := module.CallDefault(context.Background(), "asset")
	if err != nil {
		t.Fatalf("expected default export to succeed, got %v", err)
	}
	payload := map[string]any{}
	if err := Decode(defaultValue, &payload); err != nil {
		t.Fatalf("expected default export payload to decode, got %v", err)
	}
	if payload["label"] != "asset" {
		t.Fatalf("unexpected default export payload: %#v", payload)
	}
	version, err := module.Value(context.Background(), "version")
	if err != nil || version.(string) != "1.0.0" {
		t.Fatalf("expected module value export, got %#v err=%v", version, err)
	}
	if err := module.Dispose(); err != nil {
		t.Fatalf("expected dispose to succeed, got %v", err)
	}
	if _, err := module.Value(context.Background(), "version"); !IsCode(err, CodeDisposed) {
		t.Fatalf("expected disposed module error, got %v", err)
	}
}

func BenchmarkLocalStorageGetItem(b *testing.B) {
	storage := js.Global().Get("Object").New()
	getItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return "dark"
	})
	defer getItemFn.Release()
	storage.Set("getItem", getItemFn)
	storage.Set("setItem", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
	storage.Set("removeItem", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
	storage.Set("clear", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
	storage.Set("key", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return js.Null() }))
	storage.Set("length", 1)
	restoreStorage := setGlobalValue("localStorage", storage)
	defer restoreStorage()

	local, err := LocalStorage()
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		value, ok, err := local.GetItem("theme")
		if err != nil || !ok || value != "dark" {
			b.Fatalf("unexpected storage read: %q ok=%t err=%v", value, ok, err)
		}
	}
}

func BenchmarkModuleCall(b *testing.B) {
	exportFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return js.ValueOf(args[0].Float() + 1)
	})
	defer exportFn.Release()
	moduleNS := js.Global().Get("Object").New()
	moduleNS.Set("next", exportFn)
	importFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return makePromise(moduleNS)
	})
	defer importFn.Release()
	restoreImport := setGlobalValue("__gwcImportModule", importFn)
	defer restoreImport()

	module, err := ImportModule(context.Background(), "/bench/module.js")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		value, err := module.Call(context.Background(), "next", 41)
		if err != nil || value.(float64) != 42 {
			b.Fatalf("unexpected module call result %#v err=%v", value, err)
		}
	}
}
