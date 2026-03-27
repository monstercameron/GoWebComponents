//go:build !js || !wasm
// +build !js !wasm

package interop

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestInteropNativeWrapperSurfaces(parseT *testing.T) {
	_, parseErr := GlobalThis()
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	_, parseErr = LocalStorage()
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	_, parseErr = SessionStorage()
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	_, parseErr = NavigatorClipboard()
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	_, parseErr = CurrentDocument()
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	_, parseErr = NewGoWASMWorker(context.Background(), GoWASMWorkerOptions{})
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	_, parseErr = SetTimeout(time.Millisecond, func() {})
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	_, parseErr = SetInterval(time.Millisecond, func() {})
	requireInteropCode(parseT, parseErr, CodeUnavailable)

	var parseValue Value
	_ = parseValue.Present()
	_ = parseValue.Truthy()
	_ = parseValue.IsUndefined()
	_ = parseValue.IsNull()
	_ = parseValue.String()
	_ = parseValue.Bool()
	_ = parseValue.Int()
	_ = parseValue.Float()
	_ = parseValue.Get("x")
	requireInteropCode(parseT, parseValue.Set("x", "y"), CodeUnavailable)
	requireInteropCode(parseT, parseValue.Delete("x"), CodeUnavailable)
	_, parseErr = parseValue.Call("x")
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	_, parseErr = parseValue.Invoke()
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	_, parseErr = parseValue.ToGo()
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	_, parseErr = parseValue.SetFunction("x", func(...Value) any { return nil })
	requireInteropCode(parseT, parseErr, CodeUnavailable)

	parseE := &Error{Op: "decode", Target: "payload", Code: CodeInvalid, Err: parseErr}
	if !strings.Contains(parseE.Error(), "gwc-interop-invalid") {
		parseT.Fatalf("expected actionable invalid guidance in error")
	}
	if parseE.Unwrap() == nil {
		parseT.Fatalf("expected unwrap")
	}
	if wrapError("x", "y", CodeInvalid, nil) != nil {
		parseT.Fatalf("expected nil wrapError when source error is nil")
	}
}

func TestStoragePersistentAndLocationHistoryWrappers(parseT *testing.T) {
	parseCtx := context.TODO()

	storeData := map[string]string{"a": "1"}
	parseStorage := Storage{
		getItem: func(parseKey string) (string, bool, error) {
			parseV, parseOk := storeData[parseKey]
			return parseV, parseOk, nil
		},
		setItem: func(parseKey2, parseValue string) error { storeData[parseKey2] = parseValue; return nil },
		removeItem: func(parseKey3 string) error {
			delete(storeData, parseKey3)
			return nil
		},
		clear:  func() error { storeData = map[string]string{}; return nil },
		length: func() (int, error) { return len(storeData), nil },
		key:    func(int) (string, bool, error) { return "a", true, nil },
	}
	if _, _, parseErr := parseStorage.GetItem("a"); parseErr != nil {
		parseT.Fatalf("storage get item: %v", parseErr)
	}
	if _, parseErr2 := parseStorage.GetMany("a"); parseErr2 != nil {
		parseT.Fatalf("storage get many fallback: %v", parseErr2)
	}
	if parseErr3 := parseStorage.SetItem("b", "2"); parseErr3 != nil {
		parseT.Fatalf("storage set item: %v", parseErr3)
	}
	if _, parseErr4 := parseStorage.Len(); parseErr4 != nil {
		parseT.Fatalf("storage len: %v", parseErr4)
	}
	if _, _, parseErr5 := parseStorage.Key(0); parseErr5 != nil {
		parseT.Fatalf("storage key: %v", parseErr5)
	}
	if parseErr6 := parseStorage.RemoveItem("b"); parseErr6 != nil {
		parseT.Fatalf("storage remove item: %v", parseErr6)
	}
	if parseErr7 := parseStorage.Clear(); parseErr7 != nil {
		parseT.Fatalf("storage clear: %v", parseErr7)
	}

	parsePersistentData := map[string]string{"json": `{"n":7}`}
	parsePersistent := PersistentStore{
		backend: func() string { return "memory" },
		getItem: func(context.Context, string) (string, bool, error) {
			parseV2, parseOk2 := parsePersistentData["json"]
			return parseV2, parseOk2, nil
		},
		setItem:    func(context.Context, string, string) error { return nil },
		removeItem: func(context.Context, string) error { return nil },
		clear:      func(context.Context) error { return nil },
		keys:       func(context.Context) ([]string, error) { return []string{"json"}, nil },
		length:     func(context.Context) (int, error) { return len(parsePersistentData), nil },
		close:      func() error { return nil },
	}
	if parsePersistent.Backend() != "memory" {
		parseT.Fatalf("unexpected backend")
	}
	if _, _, parseErr8 := parsePersistent.GetItem(parseCtx, "json"); parseErr8 != nil {
		parseT.Fatalf("persistent get item: %v", parseErr8)
	}
	if _, parseErr9 := parsePersistent.GetMany(parseCtx, "json"); parseErr9 != nil {
		parseT.Fatalf("persistent get many: %v", parseErr9)
	}
	if parseErr10 := parsePersistent.SetItem(parseCtx, "json", "{}"); parseErr10 != nil {
		parseT.Fatalf("persistent set item: %v", parseErr10)
	}
	if parseErr11 := parsePersistent.SetJSON(parseCtx, "json", map[string]int{"n": 8}); parseErr11 != nil {
		parseT.Fatalf("persistent set json: %v", parseErr11)
	}
	var parseDecoded struct {
		N int `json:"n"`
	}
	if _, parseErr12 := parsePersistent.DecodeJSON(parseCtx, "json", &parseDecoded); parseErr12 != nil {
		parseT.Fatalf("persistent decode json: %v", parseErr12)
	}
	if _, _, parseErr13 := LoadPersistentJSON[struct {
		N int `json:"n"`
	}](parseCtx, parsePersistent, "json"); parseErr13 != nil {
		parseT.Fatalf("load persistent json: %v", parseErr13)
	}
	if parseErr14 := parsePersistent.RemoveItem(parseCtx, "json"); parseErr14 != nil {
		parseT.Fatalf("persistent remove: %v", parseErr14)
	}
	if parseErr15 := parsePersistent.Clear(parseCtx); parseErr15 != nil {
		parseT.Fatalf("persistent clear: %v", parseErr15)
	}
	if _, parseErr16 := parsePersistent.Keys(parseCtx); parseErr16 != nil {
		parseT.Fatalf("persistent keys: %v", parseErr16)
	}
	if _, parseErr17 := parsePersistent.Len(parseCtx); parseErr17 != nil {
		parseT.Fatalf("persistent len: %v", parseErr17)
	}
	if parseErr18 := parsePersistent.Close(); parseErr18 != nil {
		parseT.Fatalf("persistent close: %v", parseErr18)
	}

	parseLocation := Location{
		href:     func() string { return "https://example.com/path?q=1#h" },
		pathname: func() string { return "/path" },
		search:   func() string { return "?q=1" },
		hash:     func() string { return "#h" },
		origin:   func() string { return "https://example.com" },
		assign:   func(string) error { return nil },
		replace:  func(string) error { return nil },
		reload:   func() error { return nil },
	}
	_ = parseLocation.Href()
	_ = parseLocation.Pathname()
	_ = parseLocation.Search()
	_ = parseLocation.Hash()
	_ = parseLocation.Origin()
	_ = parseLocation.Assign("/next")
	_ = parseLocation.Replace("/next")
	_ = parseLocation.Reload()

	parseHistory := History{
		length:       func() (int, error) { return 2, nil },
		state:        func() (any, error) { return map[string]any{"p": 1}, nil },
		back:         func() error { return nil },
		forward:      func() error { return nil },
		goDelta:      func(int) error { return nil },
		pushState:    func(any, string, string) error { return nil },
		replaceState: func(any, string, string) error { return nil },
	}
	if _, parseErr19 := parseHistory.Len(); parseErr19 != nil {
		parseT.Fatalf("history len: %v", parseErr19)
	}
	if _, parseErr20 := parseHistory.State(); parseErr20 != nil {
		parseT.Fatalf("history state: %v", parseErr20)
	}
	_ = parseHistory.Back()
	_ = parseHistory.Forward()
	_ = parseHistory.Go(1)
	_ = parseHistory.PushState(nil, "t", "/x")
	_ = parseHistory.ReplaceState(nil, "t", "/x")
}

func TestEventsElementDocumentModuleWorkerAndSignals(parseT *testing.T) {
	parseCtx := context.TODO()

	parseEvents := EventTarget{
		dispatch: func(string, any) error { return nil },
		listen: func(parseName string, handler func(BrowserEvent)) (Subscription, error) {
			handler(BrowserEvent{Type: parseName, Detail: map[string]any{"count": 2}})
			return Subscription{}, nil
		},
	}
	_ = parseEvents.Dispatch("ready", nil)
	if _, parseErr := parseEvents.Listen("ready", func(BrowserEvent) {}); parseErr != nil {
		parseT.Fatalf("listen: %v", parseErr)
	}
	if _, parseErr2 := parseEvents.Subscribe("ready", func(CustomEvent) {}); parseErr2 != nil {
		parseT.Fatalf("subscribe: %v", parseErr2)
	}
	if _, parseErr3 := SubscribeDecoded(parseEvents, "ready", func(DecodedCustomEvent[map[string]int], error) {}); parseErr3 != nil {
		parseT.Fatalf("subscribe decoded: %v", parseErr3)
	}

	parseElement := Element{
		tagName:            func() string { return "DIV" },
		id:                 func() string { return "root" },
		className:          func() string { return "demo" },
		focus:              func() error { return nil },
		blur:               func() error { return nil },
		click:              func() error { return nil },
		setScrollTop:       func(float64) error { return nil },
		scrollIntoView:     func(ScrollIntoViewOptions) error { return nil },
		boundingClientRect: func() (Rect, error) { return Rect{Width: 1}, nil },
		events:             func() (EventTarget, error) { return parseEvents, nil },
		observeResize:      func(func(ResizeEntry)) (Subscription, error) { return Subscription{}, nil },
		observeIntersection: func(IntersectionObserverOptions, func(IntersectionEntry)) (Subscription, error) {
			return Subscription{}, nil
		},
		scrollMetrics: func() (float64, float64, float64, error) { return 1, 2, 3, nil },
	}
	_ = parseElement.TagName()
	_ = parseElement.ID()
	_ = parseElement.ClassName()
	_ = parseElement.Focus()
	_ = parseElement.Blur()
	_ = parseElement.Click()
	_ = parseElement.SetScrollTop(1)
	_ = parseElement.ScrollIntoView()
	_, _ = parseElement.BoundingClientRect()
	if _, parseErr4 := parseElement.Events(); parseErr4 != nil {
		parseT.Fatalf("events: %v", parseErr4)
	}
	_, _ = parseElement.Listen("ready", func(BrowserEvent) {})
	_, _ = parseElement.Subscribe("ready", func(CustomEvent) {})
	_ = parseElement.Dispatch("ready", nil)
	_, _ = parseElement.ObserveResize(func(ResizeEntry) {})
	_, _ = parseElement.ObserveIntersection(func(IntersectionEntry) {})
	_, _, _, _ = parseElement.ScrollMetrics()

	parseDocument := Document{
		elementByID:   func(string) (Element, bool, error) { return parseElement, true, nil },
		querySelector: func(string) (Element, bool, error) { return parseElement, true, nil },
	}
	_, _, _ = parseDocument.ElementByID("root")
	_, _, _ = parseDocument.QuerySelector("#root")
	_, _ = parseDocument.ElementsByID("root")

	parseModule := Module{
		call:        func(context.Context, string, ...any) (any, error) { return "ok", nil },
		callDefault: func(context.Context, ...any) (any, error) { return "default", nil },
		value:       func(context.Context, string) (any, error) { return 1, nil },
		dispose:     func() error { return nil },
	}
	_, _ = parseModule.Call(parseCtx, "x")
	_, _ = parseModule.CallDefault(parseCtx)
	_, _ = parseModule.Value(parseCtx, "x")
	_ = parseModule.Dispose()

	parseWorker := Worker{
		post:      func(any) error { return nil },
		postPorts: func(any, ...MessagePort) error { return nil },
		subscribe: func(func(WorkerMessage, error)) (Subscription, error) { return Subscription{}, nil },
		request: func(context.Context, string, any, func(WorkerMessage, error)) (WorkerMessage, error) {
			return WorkerMessage{Phase: "result", Name: "task", Payload: map[string]any{"n": 1}}, nil
		},
		terminate: func() error { return nil },
		restart:   func(context.Context) error { return nil },
	}
	_ = parseWorker.Post("x")
	_ = parseWorker.PostPorts("x")
	_, _ = parseWorker.Subscribe(func(WorkerMessage, error) {})
	_, _ = parseWorker.Request(parseCtx, "task", nil, nil)
	_ = parseWorker.Terminate()
	_ = parseWorker.Restart(parseCtx)
	_, _ = DecodeWorkerMessage[map[string]int](WorkerMessage{Name: "task", Payload: map[string]any{"n": 1}})
	_, _ = SubscribeDecodedWorker(parseWorker, func(DecodedWorkerMessage[map[string]int], error) {})
	_, _ = RequestWorkerDecoded[map[string]int, map[string]int, map[string]int](parseCtx, parseWorker, "task", nil, nil)

	parsePort := MessagePort{
		post:      func(any) error { return nil },
		postPorts: func(any, ...MessagePort) error { return nil },
		subscribe: func(func(MessagePortMessage, error)) (Subscription, error) { return Subscription{}, nil },
		close:     func() error { return nil },
	}
	parseChannel := MessageChannel{port1: parsePort, port2: parsePort}
	_ = parseChannel.Port1().Post("x")
	_ = parseChannel.Port2().PostPorts("x")
	_, _ = parsePort.Subscribe(func(MessagePortMessage, error) {})
	_ = parsePort.Close()
	_, _ = DecodeMessagePortMessage[map[string]int](MessagePortMessage{Payload: map[string]any{"n": 1}})
	_, _ = SubscribeDecodedMessagePort(parsePort, func(DecodedMessagePortMessage[map[string]int], error) {})

	parseShared := SharedBuffer{
		getByteLength:  func() int { return 16 },
		readBytes:      func(int, []byte) (int, error) { return 0, nil },
		writeBytes:     func(int, []byte) (int, error) { return 0, nil },
		getInt32Length: func() int { return 4 },
		loadInt32:      func(int) (int32, error) { return 1, nil },
		storeInt32:     func(int, int32) error { return nil },
		addInt32:       func(int, int32) (int32, error) { return 1, nil },
		subInt32:       func(int, int32) (int32, error) { return 1, nil },
		andInt32:       func(int, int32) (int32, error) { return 1, nil },
		orInt32:        func(int, int32) (int32, error) { return 1, nil },
		xorInt32:       func(int, int32) (int32, error) { return 1, nil },
		exchangeInt32:  func(int, int32) (int32, error) { return 1, nil },
		compareExchangeInt32: func(int, int32, int32) (int32, error) {
			return 1, nil
		},
	}
	_, _ = GetSharedMemorySupport()
	_ = parseShared.GetByteLength()
	_, _ = parseShared.ReadBytes(0, make([]byte, 4))
	_, _ = parseShared.WriteBytes(0, []byte{1, 2, 3, 4})
	_ = parseShared.GetInt32Length()
	_, _ = parseShared.LoadInt32(0)
	_ = parseShared.StoreInt32(0, 1)
	_, _ = parseShared.AddInt32(0, 1)
	_, _ = parseShared.SubInt32(0, 1)
	_, _ = parseShared.AndInt32(0, 1)
	_, _ = parseShared.OrInt32(0, 1)
	_, _ = parseShared.XorInt32(0, 1)
	_, _ = parseShared.ExchangeInt32(0, 1)
	_, _ = parseShared.CompareExchangeInt32(0, 1, 2)
	_, _ = parseShared.WaitInt32(0, 1, time.Millisecond)
	_, _ = parseShared.NotifyInt32(0, 1)

	parsePool := WorkerPool{
		request: func(context.Context, string, any, func(WorkerMessage, error)) (WorkerMessage, error) {
			return WorkerMessage{Phase: "result", Name: "task", Payload: map[string]any{"n": 3}}, nil
		},
		close:         func() error { return nil },
		drain:         func(context.Context) error { return nil },
		getSize:       func() int { return 2 },
		getQueueLimit: func() int { return 4 },
	}
	_, _ = parsePool.Request(parseCtx, "task", nil, nil)
	_ = parsePool.Close()
	_ = parsePool.Drain(parseCtx)
	_ = parsePool.GetSize()
	_ = parsePool.GetQueueLimit()
	_, _ = RequestWorkerDecoded[map[string]int, map[string]int, map[string]int](parseCtx, parsePool, "task", nil, nil)

	parseWindow := WindowChannel{
		name:         func() string { return "atlas-window" },
		targetOrigin: func() string { return "https://atlas.local" },
		publish:      func(any) error { return nil },
		subscribe:    func(func(WindowEnvelope, error)) (Subscription, error) { return Subscription{}, nil },
		focus:        func() error { return nil },
		close:        func() error { return nil },
		closed:       func() bool { return false },
	}
	_ = PublishLogout(parseWindow, "logout")
	_ = PublishSessionExpired(parseWindow, "expired", "/login", time.Now().UTC())
	_ = PublishRouteFocus(parseWindow, "/app/inventory", "status=risk", "row")
	_ = PublishSelection(parseWindow, "inventory", "frame-desk", "rev-1")
	_ = PublishIntent(parseWindow, SurfaceIntentFocusPanel, "panel", map[string]string{"id": "1"})
}

func requireInteropCode(parseT *testing.T, parseErr error, parseCode ErrorCode) {
	parseT.Helper()
	if !IsCode(parseErr, parseCode) {
		parseT.Fatalf("expected %q, got %v", parseCode, parseErr)
	}
}
