//go:build !js || !wasm
// +build !js !wasm

package interop

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestInteropNativeWrapperSurfaces(t *testing.T) {
	_, err := GlobalThis()
	requireInteropCode(t, err, CodeUnavailable)
	_, err = LocalStorage()
	requireInteropCode(t, err, CodeUnavailable)
	_, err = SessionStorage()
	requireInteropCode(t, err, CodeUnavailable)
	_, err = NavigatorClipboard()
	requireInteropCode(t, err, CodeUnavailable)
	_, err = CurrentDocument()
	requireInteropCode(t, err, CodeUnavailable)
	_, err = NewGoWASMWorker(context.Background(), GoWASMWorkerOptions{})
	requireInteropCode(t, err, CodeUnavailable)
	_, err = SetTimeout(time.Millisecond, func() {})
	requireInteropCode(t, err, CodeUnavailable)
	_, err = SetInterval(time.Millisecond, func() {})
	requireInteropCode(t, err, CodeUnavailable)

	var value Value
	_ = value.Present()
	_ = value.Truthy()
	_ = value.IsUndefined()
	_ = value.IsNull()
	_ = value.String()
	_ = value.Bool()
	_ = value.Int()
	_ = value.Float()
	_ = value.Get("x")
	requireInteropCode(t, value.Set("x", "y"), CodeUnavailable)
	requireInteropCode(t, value.Delete("x"), CodeUnavailable)
	_, err = value.Call("x")
	requireInteropCode(t, err, CodeUnavailable)
	_, err = value.Invoke()
	requireInteropCode(t, err, CodeUnavailable)
	_, err = value.ToGo()
	requireInteropCode(t, err, CodeUnavailable)
	_, err = value.SetFunction("x", func(...Value) any { return nil })
	requireInteropCode(t, err, CodeUnavailable)

	e := &Error{Op: "decode", Target: "payload", Code: CodeInvalid, Err: err}
	if !strings.Contains(e.Error(), "gwc-interop-invalid") {
		t.Fatalf("expected actionable invalid guidance in error")
	}
	if e.Unwrap() == nil {
		t.Fatalf("expected unwrap")
	}
	if wrapError("x", "y", CodeInvalid, nil) != nil {
		t.Fatalf("expected nil wrapError when source error is nil")
	}
}

func TestStoragePersistentAndLocationHistoryWrappers(t *testing.T) {
	storeData := map[string]string{"a": "1"}
	storage := Storage{
		getItem: func(key string) (string, bool, error) { v, ok := storeData[key]; return v, ok, nil },
		setItem: func(key, value string) error { storeData[key] = value; return nil },
		removeItem: func(key string) error {
			delete(storeData, key)
			return nil
		},
		clear:  func() error { storeData = map[string]string{}; return nil },
		length: func() (int, error) { return len(storeData), nil },
		key:    func(int) (string, bool, error) { return "a", true, nil },
	}
	if _, _, err := storage.GetItem("a"); err != nil {
		t.Fatalf("storage get item: %v", err)
	}
	if _, err := storage.GetMany("a"); err != nil {
		t.Fatalf("storage get many fallback: %v", err)
	}
	if err := storage.SetItem("b", "2"); err != nil {
		t.Fatalf("storage set item: %v", err)
	}
	if _, err := storage.Len(); err != nil {
		t.Fatalf("storage len: %v", err)
	}
	if _, _, err := storage.Key(0); err != nil {
		t.Fatalf("storage key: %v", err)
	}
	if err := storage.RemoveItem("b"); err != nil {
		t.Fatalf("storage remove item: %v", err)
	}
	if err := storage.Clear(); err != nil {
		t.Fatalf("storage clear: %v", err)
	}

	persistentData := map[string]string{"json": `{"n":7}`}
	persistent := PersistentStore{
		backend: func() string { return "memory" },
		getItem: func(context.Context, string) (string, bool, error) {
			v, ok := persistentData["json"]
			return v, ok, nil
		},
		setItem:    func(context.Context, string, string) error { return nil },
		removeItem: func(context.Context, string) error { return nil },
		clear:      func(context.Context) error { return nil },
		keys:       func(context.Context) ([]string, error) { return []string{"json"}, nil },
		length:     func(context.Context) (int, error) { return len(persistentData), nil },
		close:      func() error { return nil },
	}
	if persistent.Backend() != "memory" {
		t.Fatalf("unexpected backend")
	}
	if _, _, err := persistent.GetItem(nil, "json"); err != nil {
		t.Fatalf("persistent get item: %v", err)
	}
	if _, err := persistent.GetMany(nil, "json"); err != nil {
		t.Fatalf("persistent get many: %v", err)
	}
	if err := persistent.SetItem(nil, "json", "{}"); err != nil {
		t.Fatalf("persistent set item: %v", err)
	}
	if err := persistent.SetJSON(nil, "json", map[string]int{"n": 8}); err != nil {
		t.Fatalf("persistent set json: %v", err)
	}
	var decoded struct {
		N int `json:"n"`
	}
	if _, err := persistent.DecodeJSON(nil, "json", &decoded); err != nil {
		t.Fatalf("persistent decode json: %v", err)
	}
	if _, _, err := LoadPersistentJSON[struct {
		N int `json:"n"`
	}](nil, persistent, "json"); err != nil {
		t.Fatalf("load persistent json: %v", err)
	}
	if err := persistent.RemoveItem(nil, "json"); err != nil {
		t.Fatalf("persistent remove: %v", err)
	}
	if err := persistent.Clear(nil); err != nil {
		t.Fatalf("persistent clear: %v", err)
	}
	if _, err := persistent.Keys(nil); err != nil {
		t.Fatalf("persistent keys: %v", err)
	}
	if _, err := persistent.Len(nil); err != nil {
		t.Fatalf("persistent len: %v", err)
	}
	if err := persistent.Close(); err != nil {
		t.Fatalf("persistent close: %v", err)
	}

	location := Location{
		href:     func() string { return "https://example.com/path?q=1#h" },
		pathname: func() string { return "/path" },
		search:   func() string { return "?q=1" },
		hash:     func() string { return "#h" },
		origin:   func() string { return "https://example.com" },
		assign:   func(string) error { return nil },
		replace:  func(string) error { return nil },
		reload:   func() error { return nil },
	}
	_ = location.Href()
	_ = location.Pathname()
	_ = location.Search()
	_ = location.Hash()
	_ = location.Origin()
	_ = location.Assign("/next")
	_ = location.Replace("/next")
	_ = location.Reload()

	history := History{
		length:       func() (int, error) { return 2, nil },
		state:        func() (any, error) { return map[string]any{"p": 1}, nil },
		back:         func() error { return nil },
		forward:      func() error { return nil },
		goDelta:      func(int) error { return nil },
		pushState:    func(any, string, string) error { return nil },
		replaceState: func(any, string, string) error { return nil },
	}
	if _, err := history.Len(); err != nil {
		t.Fatalf("history len: %v", err)
	}
	if _, err := history.State(); err != nil {
		t.Fatalf("history state: %v", err)
	}
	_ = history.Back()
	_ = history.Forward()
	_ = history.Go(1)
	_ = history.PushState(nil, "t", "/x")
	_ = history.ReplaceState(nil, "t", "/x")
}

func TestEventsElementDocumentModuleWorkerAndSignals(t *testing.T) {
	events := EventTarget{
		dispatch: func(string, any) error { return nil },
		listen: func(name string, handler func(BrowserEvent)) (Subscription, error) {
			handler(BrowserEvent{Type: name, Detail: map[string]any{"count": 2}})
			return Subscription{}, nil
		},
	}
	_ = events.Dispatch("ready", nil)
	if _, err := events.Listen("ready", func(BrowserEvent) {}); err != nil {
		t.Fatalf("listen: %v", err)
	}
	if _, err := events.Subscribe("ready", func(CustomEvent) {}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if _, err := SubscribeDecoded[map[string]int](events, "ready", func(DecodedCustomEvent[map[string]int], error) {}); err != nil {
		t.Fatalf("subscribe decoded: %v", err)
	}

	element := Element{
		tagName:            func() string { return "DIV" },
		id:                 func() string { return "root" },
		className:          func() string { return "demo" },
		focus:              func() error { return nil },
		blur:               func() error { return nil },
		click:              func() error { return nil },
		setScrollTop:       func(float64) error { return nil },
		scrollIntoView:     func(ScrollIntoViewOptions) error { return nil },
		boundingClientRect: func() (Rect, error) { return Rect{Width: 1}, nil },
		events:             func() (EventTarget, error) { return events, nil },
		observeResize:      func(func(ResizeEntry)) (Subscription, error) { return Subscription{}, nil },
		observeIntersection: func(IntersectionObserverOptions, func(IntersectionEntry)) (Subscription, error) {
			return Subscription{}, nil
		},
		scrollMetrics: func() (float64, float64, float64, error) { return 1, 2, 3, nil },
	}
	_ = element.TagName()
	_ = element.ID()
	_ = element.ClassName()
	_ = element.Focus()
	_ = element.Blur()
	_ = element.Click()
	_ = element.SetScrollTop(1)
	_ = element.ScrollIntoView()
	_, _ = element.BoundingClientRect()
	if _, err := element.Events(); err != nil {
		t.Fatalf("events: %v", err)
	}
	_, _ = element.Listen("ready", func(BrowserEvent) {})
	_, _ = element.Subscribe("ready", func(CustomEvent) {})
	_ = element.Dispatch("ready", nil)
	_, _ = element.ObserveResize(func(ResizeEntry) {})
	_, _ = element.ObserveIntersection(func(IntersectionEntry) {})
	_, _, _, _ = element.ScrollMetrics()

	document := Document{
		elementByID:   func(string) (Element, bool, error) { return element, true, nil },
		querySelector: func(string) (Element, bool, error) { return element, true, nil },
	}
	_, _, _ = document.ElementByID("root")
	_, _, _ = document.QuerySelector("#root")
	_, _ = document.ElementsByID("root")

	module := Module{
		call:        func(context.Context, string, ...any) (any, error) { return "ok", nil },
		callDefault: func(context.Context, ...any) (any, error) { return "default", nil },
		value:       func(context.Context, string) (any, error) { return 1, nil },
		dispose:     func() error { return nil },
	}
	_, _ = module.Call(nil, "x")
	_, _ = module.CallDefault(nil)
	_, _ = module.Value(nil, "x")
	_ = module.Dispose()

	worker := Worker{
		post:      func(any) error { return nil },
		subscribe: func(func(WorkerMessage, error)) (Subscription, error) { return Subscription{}, nil },
		request: func(context.Context, string, any, func(WorkerMessage, error)) (WorkerMessage, error) {
			return WorkerMessage{Phase: "result", Name: "task", Payload: map[string]any{"n": 1}}, nil
		},
		terminate: func() error { return nil },
		restart:   func(context.Context) error { return nil },
	}
	_ = worker.Post("x")
	_, _ = worker.Subscribe(func(WorkerMessage, error) {})
	_, _ = worker.Request(nil, "task", nil, nil)
	_ = worker.Terminate()
	_ = worker.Restart(nil)
	_, _ = DecodeWorkerMessage[map[string]int](WorkerMessage{Name: "task", Payload: map[string]any{"n": 1}})
	_, _ = SubscribeDecodedWorker[map[string]int](worker, func(DecodedWorkerMessage[map[string]int], error) {})
	_, _ = RequestWorkerDecoded[map[string]int, map[string]int, map[string]int](nil, worker, "task", nil, nil)

	window := WindowChannel{
		name:         func() string { return "atlas-window" },
		targetOrigin: func() string { return "https://atlas.local" },
		publish:      func(any) error { return nil },
		subscribe:    func(func(WindowEnvelope, error)) (Subscription, error) { return Subscription{}, nil },
		focus:        func() error { return nil },
		close:        func() error { return nil },
		closed:       func() bool { return false },
	}
	_ = PublishLogout(window, "logout")
	_ = PublishSessionExpired(window, "expired", "/login", time.Now().UTC())
	_ = PublishRouteFocus(window, "/app/inventory", "status=risk", "row")
	_ = PublishSelection(window, "inventory", "frame-desk", "rev-1")
	_ = PublishIntent(window, SurfaceIntentFocusPanel, "panel", map[string]string{"id": "1"})
}

func requireInteropCode(t *testing.T, err error, code ErrorCode) {
	t.Helper()
	if !IsCode(err, code) {
		t.Fatalf("expected %q, got %v", code, err)
	}
}
