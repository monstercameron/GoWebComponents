//go:build !js || !wasm
// +build !js !wasm

package interop

import (
	"context"
	"testing"
	"time"
)

func TestClipboardTimerMediaAndWindowEnvWrappers(t *testing.T) {
	env := WindowEnv{}
	if _, ok := env.Lookup("x"); ok {
		t.Fatalf("expected missing lookup when env lookup is nil")
	}
	if _, ok := env.LookupString("x"); ok {
		t.Fatalf("expected missing lookup string for nil env")
	}
	if got := env.String("x", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback string, got %q", got)
	}

	clipboard := Clipboard{}
	if err := clipboard.WriteText(nil, "hello"); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable clipboard write, got %v", err)
	}
	if _, err := clipboard.ReadText(nil); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable clipboard read, got %v", err)
	}

	clipboard = Clipboard{
		writeText: func(ctx context.Context, text string) error {
			if ctx == nil || text != "hello" {
				t.Fatalf("unexpected write params: ctx=%v text=%q", ctx, text)
			}
			return nil
		},
		readText: func(ctx context.Context) (string, error) {
			if ctx == nil {
				t.Fatalf("expected context fallback")
			}
			return "hello", nil
		},
	}
	if err := clipboard.WriteText(nil, "hello"); err != nil {
		t.Fatalf("clipboard write: %v", err)
	}
	if text, err := clipboard.ReadText(nil); err != nil || text != "hello" {
		t.Fatalf("clipboard read = %q err=%v", text, err)
	}

	timer := Timer{}
	if err := timer.Cancel(); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable timer cancel, got %v", err)
	}
	timer = Timer{cancel: func() error { return nil }}
	if err := timer.Cancel(); err != nil {
		t.Fatalf("timer cancel: %v", err)
	}

	media := MediaQueryList{}
	if media.Matches() {
		t.Fatalf("expected default media query matches=false")
	}
	if media.Media() != "" {
		t.Fatalf("expected default media query media empty")
	}
	if _, err := media.Subscribe(func(MediaQueryEvent) {}); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable media subscribe, got %v", err)
	}
	media = MediaQueryList{
		matches: func() bool { return true },
		media:   func() string { return "(min-width: 1200px)" },
		subscribe: func(handler func(MediaQueryEvent)) (Subscription, error) {
			handler(MediaQueryEvent{Matches: true, Media: "(min-width: 1200px)"})
			return Subscription{}, nil
		},
	}
	if !media.Matches() || media.Media() == "" {
		t.Fatalf("expected media query accessors")
	}
	if _, err := media.Subscribe(func(MediaQueryEvent) {}); err != nil {
		t.Fatalf("media subscribe: %v", err)
	}
}

func TestChannelAndWorkerScopeZeroCoverageBranches(t *testing.T) {
	crossTab := CrossTabChannel{}
	if err := crossTab.Close(); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable cross-tab close, got %v", err)
	}
	crossTab = CrossTabChannel{close: func() error { return nil }}
	if err := crossTab.Close(); err != nil {
		t.Fatalf("cross-tab close: %v", err)
	}

	window := WindowChannel{}
	if got := window.TargetOrigin(); got != "" {
		t.Fatalf("expected empty target origin, got %q", got)
	}
	if err := window.Focus(); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable window focus, got %v", err)
	}
	if err := window.Close(); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable window close, got %v", err)
	}
	if window.Closed() {
		t.Fatalf("expected closed=false when closed callback is nil")
	}

	window = WindowChannel{
		targetOrigin: func() string { return "https://atlas.local" },
		focus:        func() error { return nil },
		close:        func() error { return nil },
		closed:       func() bool { return true },
	}
	if got := window.TargetOrigin(); got != "https://atlas.local" {
		t.Fatalf("target origin = %q", got)
	}
	if err := window.Focus(); err != nil {
		t.Fatalf("window focus: %v", err)
	}
	if err := window.Close(); err != nil {
		t.Fatalf("window close: %v", err)
	}
	if !window.Closed() {
		t.Fatalf("expected closed=true")
	}

	scope := WorkerScope{}
	if err := scope.Post(WorkerMessage{}); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable worker scope post, got %v", err)
	}
	if _, err := scope.Subscribe(func(WorkerMessage, error) {}); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable worker scope subscribe, got %v", err)
	}

	posted := make([]WorkerMessage, 0, 6)
	scope = WorkerScope{
		post: func(message WorkerMessage) error {
			posted = append(posted, message)
			return nil
		},
		subscribe: func(handler func(WorkerMessage, error)) (Subscription, error) {
			handler(WorkerMessage{Phase: "message", Name: "task", Payload: map[string]any{"ok": true}}, nil)
			return Subscription{}, nil
		},
	}
	if _, err := scope.Subscribe(func(WorkerMessage, error) {}); err != nil {
		t.Fatalf("worker scope subscribe: %v", err)
	}
	_ = scope.Ready("task")
	_ = scope.Message("task", map[string]any{"step": 1})
	_ = scope.Progress("id-1", "task", map[string]any{"progress": 50})
	_ = scope.Result("id-1", "task", map[string]any{"done": true})
	_ = scope.Error("id-1", "task", "failed", map[string]any{"reason": "timeout"})
	if len(posted) != 5 {
		t.Fatalf("expected worker scope helper posts, got %d", len(posted))
	}
}

func TestDecodedSubscribeAndPublishWindowHelpers(t *testing.T) {
	envelope := CrossTabEnvelope{
		Name:     "clients",
		Source:   "tab-1",
		Sequence: 7,
		SentAt:   time.Date(2026, 3, 24, 18, 0, 0, 0, time.UTC),
		Payload:  map[string]any{"count": 2},
	}
	decoded, err := DecodeCrossTabEnvelope[map[string]int](envelope)
	if err != nil {
		t.Fatalf("decode cross-tab envelope: %v", err)
	}
	if decoded.Name != "clients" || decoded.Payload["count"] != 2 {
		t.Fatalf("unexpected decoded cross-tab payload: %+v", decoded)
	}
	if _, err := DecodeCrossTabEnvelope[map[string]int](CrossTabEnvelope{Name: "clients", Payload: "bad"}); !IsCode(err, CodeDecode) {
		t.Fatalf("expected decode error for malformed cross-tab payload, got %v", err)
	}

	var crossTabHandler func(CrossTabEnvelope, error)
	channel := CrossTabChannel{
		name: func() string { return "clients" },
		publish: func(payload any) error {
			_ = payload
			return nil
		},
		subscribe: func(handler func(CrossTabEnvelope, error)) (Subscription, error) {
			crossTabHandler = handler
			return Subscription{}, nil
		},
	}
	if _, err := SubscribeDecodedCrossTab[map[string]int](channel, nil); !IsCode(err, CodeInvalid) {
		t.Fatalf("expected nil handler error for decoded cross-tab subscribe, got %v", err)
	}
	var gotDecoded DecodedCrossTabEnvelope[map[string]int]
	var gotErr error
	if _, err := SubscribeDecodedCrossTab[map[string]int](channel, func(msg DecodedCrossTabEnvelope[map[string]int], err error) {
		gotDecoded = msg
		gotErr = err
	}); err != nil {
		t.Fatalf("subscribe decoded cross-tab: %v", err)
	}
	crossTabHandler(envelope, nil)
	if gotErr != nil || gotDecoded.Payload["count"] != 2 {
		t.Fatalf("decoded cross-tab callback mismatch: msg=%+v err=%v", gotDecoded, gotErr)
	}
	crossTabHandler(CrossTabEnvelope{}, context.DeadlineExceeded)
	if gotErr != context.DeadlineExceeded {
		t.Fatalf("expected subscribe decoded cross-tab to pass through transport error")
	}

	var windowHandler func(WindowEnvelope, error)
	window := WindowChannel{
		name: func() string { return "atlas-window" },
		publish: func(payload any) error {
			_ = payload
			return nil
		},
		subscribe: func(handler func(WindowEnvelope, error)) (Subscription, error) {
			windowHandler = handler
			return Subscription{}, nil
		},
	}
	if _, err := SubscribeDecodedWindow[map[string]int](window, nil); !IsCode(err, CodeInvalid) {
		t.Fatalf("expected nil handler error for decoded window subscribe, got %v", err)
	}
	if _, err := SubscribeSurfaceSignals(window, nil); !IsCode(err, CodeInvalid) {
		t.Fatalf("expected nil handler error for surface signal subscribe, got %v", err)
	}

	var decodedWindow DecodedWindowEnvelope[map[string]int]
	if _, err := SubscribeDecodedWindow[map[string]int](window, func(msg DecodedWindowEnvelope[map[string]int], err error) {
		if err == nil {
			decodedWindow = msg
		}
	}); err != nil {
		t.Fatalf("subscribe decoded window: %v", err)
	}
	windowHandler(WindowEnvelope{Name: "atlas-window", Payload: map[string]any{"count": 4}}, nil)
	if decodedWindow.Payload["count"] != 4 {
		t.Fatalf("expected decoded window payload")
	}

	self := ClientIdentity{ID: "ops-1", App: "atlas", Surface: "window", Role: "operator"}
	if err := PublishClientHelloWindow(window, self); err != nil {
		t.Fatalf("publish client hello window: %v", err)
	}
	if err := PublishClientHelloWindowWithCapabilities(window, self, ClientCapabilities{ProtocolVersion: "v1"}); err != nil {
		t.Fatalf("publish hello window with capabilities: %v", err)
	}
	if err := PublishClientGoodbye(channel, self); err != nil {
		t.Fatalf("publish client goodbye: %v", err)
	}
	if err := PublishClientGoodbyeWindow(window, self); err != nil {
		t.Fatalf("publish client goodbye window: %v", err)
	}
	if err := PublishClientEvent(channel, "inventory:event", self, map[string]any{"sku": "SKU-1"}); err != nil {
		t.Fatalf("publish client event: %v", err)
	}

	caps := defaultWindowClientCapabilities(window)
	if caps.ProtocolVersion != "v1" || len(caps.Transports) != 1 || caps.Transports[0] != "window-message" {
		t.Fatalf("unexpected default window capabilities: %+v", caps)
	}
}

func TestUnavailableBranchesAcrossInteropWrappers(t *testing.T) {
	var storage Storage
	_, _, err := storage.GetItem("k")
	requireInteropCode(t, err, CodeUnavailable)
	requireInteropCode(t, storage.SetItem("k", "v"), CodeUnavailable)
	_, err = storage.GetMany("k")
	requireInteropCode(t, err, CodeUnavailable)
	requireInteropCode(t, storage.RemoveItem("k"), CodeUnavailable)
	requireInteropCode(t, storage.Clear(), CodeUnavailable)
	_, err = storage.Len()
	requireInteropCode(t, err, CodeUnavailable)
	_, _, err = storage.Key(0)
	requireInteropCode(t, err, CodeUnavailable)

	var persistent PersistentStore
	_, _, err = persistent.GetItem(nil, "k")
	requireInteropCode(t, err, CodeUnavailable)
	requireInteropCode(t, persistent.SetItem(nil, "k", "v"), CodeUnavailable)
	requireInteropCode(t, persistent.RemoveItem(nil, "k"), CodeUnavailable)
	requireInteropCode(t, persistent.Clear(nil), CodeUnavailable)
	_, err = persistent.Keys(nil)
	requireInteropCode(t, err, CodeUnavailable)
	_, err = persistent.Len(nil)
	requireInteropCode(t, err, CodeUnavailable)
	if err := persistent.Close(); err != nil {
		t.Fatalf("expected nil close for zero-value persistent store, got %v", err)
	}

	var location Location
	requireInteropCode(t, location.Assign("/next"), CodeUnavailable)
	requireInteropCode(t, location.Replace("/next"), CodeUnavailable)
	requireInteropCode(t, location.Reload(), CodeUnavailable)
	if location.Href() != "" || location.Pathname() != "" || location.Search() != "" || location.Hash() != "" || location.Origin() != "" {
		t.Fatalf("expected zero-value location getters to return empty strings")
	}

	var history History
	_, err = history.Len()
	requireInteropCode(t, err, CodeUnavailable)
	_, err = history.State()
	requireInteropCode(t, err, CodeUnavailable)
	requireInteropCode(t, history.Back(), CodeUnavailable)
	requireInteropCode(t, history.Forward(), CodeUnavailable)
	requireInteropCode(t, history.Go(1), CodeUnavailable)
	requireInteropCode(t, history.PushState(nil, "", "/"), CodeUnavailable)
	requireInteropCode(t, history.ReplaceState(nil, "", "/"), CodeUnavailable)

	var events EventTarget
	requireInteropCode(t, events.Dispatch("ready", nil), CodeUnavailable)
	if _, err := events.Listen("ready", func(BrowserEvent) {}); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable event listen, got %v", err)
	}
	if _, err := events.Subscribe("ready", func(CustomEvent) {}); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable event subscribe, got %v", err)
	}

	var element Element
	if element.TagName() != "" || element.ID() != "" || element.ClassName() != "" {
		t.Fatalf("expected empty zero-value element metadata")
	}
	requireInteropCode(t, element.Focus(), CodeUnavailable)
	requireInteropCode(t, element.Blur(), CodeUnavailable)
	requireInteropCode(t, element.Click(), CodeUnavailable)
	requireInteropCode(t, element.SetScrollTop(1), CodeUnavailable)
	requireInteropCode(t, element.ScrollIntoView(), CodeUnavailable)
	if _, err := element.BoundingClientRect(); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable bounding rect, got %v", err)
	}
	if _, err := element.Events(); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable element events, got %v", err)
	}
	if _, err := element.Listen("ready", func(BrowserEvent) {}); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable element listen, got %v", err)
	}
	if _, err := element.Subscribe("ready", func(CustomEvent) {}); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable element subscribe, got %v", err)
	}
	if err := element.Dispatch("ready", nil); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable element dispatch, got %v", err)
	}
	if _, err := element.ObserveResize(func(ResizeEntry) {}); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable resize observe, got %v", err)
	}
	if _, err := element.ObserveIntersection(func(IntersectionEntry) {}); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable intersection observe, got %v", err)
	}
	if _, _, _, err := element.ScrollMetrics(); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable scroll metrics, got %v", err)
	}

	var document Document
	if _, _, err := document.ElementByID("root"); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable ElementByID, got %v", err)
	}
	if _, _, err := document.QuerySelector("#root"); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable QuerySelector, got %v", err)
	}
	if _, err := document.ElementsByID("root"); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable ElementsByID, got %v", err)
	}

	var module Module
	if _, err := module.Call(nil, "x"); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable module call, got %v", err)
	}
	if _, err := module.CallDefault(nil); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable module default call, got %v", err)
	}
	if _, err := module.Value(nil, "x"); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable module value, got %v", err)
	}
	if err := module.Dispose(); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable module dispose, got %v", err)
	}

	var crossTab CrossTabChannel
	if crossTab.Name() != "" || crossTab.Transport() != "" {
		t.Fatalf("expected empty zero-value cross-tab metadata")
	}
	if err := crossTab.Publish(nil); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable cross-tab publish, got %v", err)
	}
	if _, err := crossTab.Subscribe(func(CrossTabEnvelope, error) {}); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable cross-tab subscribe, got %v", err)
	}

	var window WindowChannel
	if window.Name() != "" {
		t.Fatalf("expected empty zero-value window name")
	}
	if err := window.Publish(nil); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable window publish, got %v", err)
	}
	if _, err := window.Subscribe(func(WindowEnvelope, error) {}); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable window subscribe, got %v", err)
	}

	var worker Worker
	if err := worker.Post("msg"); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable worker post, got %v", err)
	}
	if _, err := worker.Subscribe(func(WorkerMessage, error) {}); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable worker subscribe, got %v", err)
	}
	if _, err := worker.Request(nil, "task", nil, nil); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable worker request, got %v", err)
	}
	if err := worker.Terminate(); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable worker terminate, got %v", err)
	}
	if err := worker.Restart(nil); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected unavailable worker restart, got %v", err)
	}
	if _, err := SubscribeDecodedWorker[map[string]int](worker, nil); !IsCode(err, CodeInvalid) {
		t.Fatalf("expected invalid nil-handler decoded worker subscribe, got %v", err)
	}

	worker = Worker{
		subscribe: func(handler func(WorkerMessage, error)) (Subscription, error) {
			handler(WorkerMessage{}, context.DeadlineExceeded)
			handler(WorkerMessage{Name: "task", Payload: "bad"}, nil)
			handler(WorkerMessage{Name: "task", Payload: map[string]any{"n": 2}}, nil)
			return Subscription{}, nil
		},
		request: func(_ context.Context, _ string, _ any, onProgress func(WorkerMessage, error)) (WorkerMessage, error) {
			if onProgress != nil {
				onProgress(WorkerMessage{Name: "task", Payload: "bad"}, nil)
			}
			return WorkerMessage{Name: "task", Payload: map[string]any{"n": 9}}, nil
		},
	}
	calls := 0
	if _, err := SubscribeDecodedWorker[map[string]int](worker, func(DecodedWorkerMessage[map[string]int], error) {
		calls++
	}); err != nil {
		t.Fatalf("subscribe decoded worker: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected decoded worker callback invocations, got %d", calls)
	}
	if result, err := RequestWorkerDecoded[map[string]int, map[string]int, map[string]int](nil, worker, "task", map[string]int{"x": 1}, func(DecodedWorkerMessage[map[string]int], error) {}); err != nil || result["n"] != 9 {
		t.Fatalf("request worker decoded result=%v err=%v", result, err)
	}
}

func TestClientDecodeScalarHelpers(t *testing.T) {
	fields := map[string]any{
		"name": "atlas",
		"kind": 7,
	}
	if got := stringField(fields, "name"); got != "atlas" {
		t.Fatalf("stringField string = %q", got)
	}
	if got := stringField(fields, "kind"); got != "" {
		t.Fatalf("stringField non-string = %q", got)
	}
	if got := stringField(fields, "missing"); got != "" {
		t.Fatalf("stringField missing = %q", got)
	}

	if got := intField(7); got != 7 {
		t.Fatalf("intField int = %d", got)
	}
	if got := intField(int64(8)); got != 8 {
		t.Fatalf("intField int64 = %d", got)
	}
	if got := intField(9.75); got != 9 {
		t.Fatalf("intField float64 = %d", got)
	}
	if got := intField("bad"); got != 0 {
		t.Fatalf("intField default = %d", got)
	}

	if parsed, ok := clientTimeField(" "); ok || !parsed.IsZero() {
		t.Fatalf("clientTimeField blank = %v ok=%t", parsed, ok)
	}
	if parsed, ok := clientTimeField("not-a-time"); ok || !parsed.IsZero() {
		t.Fatalf("clientTimeField invalid = %v ok=%t", parsed, ok)
	}

	expected := time.Date(2026, 3, 25, 13, 30, 0, 123456789, time.UTC)
	parsed, ok := clientTimeField(expected.Format(time.RFC3339Nano))
	if !ok || !parsed.Equal(expected) {
		t.Fatalf("clientTimeField valid = %v ok=%t", parsed, ok)
	}
}

func TestClientCapabilityDefaultsAndProtocolHelpers(t *testing.T) {
	broadcastCaps := defaultCrossTabClientCapabilities(CrossTabChannel{
		transport: func() string { return "broadcast-channel" },
	})
	if broadcastCaps.ProtocolVersion != "v1" || len(broadcastCaps.Transports) != 1 || broadcastCaps.Transports[0] != "broadcast-channel" {
		t.Fatalf("unexpected broadcast default capabilities: %+v", broadcastCaps)
	}
	if !ClientSupportsEncoding(broadcastCaps, ClientPayloadBinary) {
		t.Fatalf("expected broadcast channel defaults to include binary encoding")
	}

	fallbackCaps := defaultCrossTabClientCapabilities(CrossTabChannel{
		transport: func() string { return "  " },
	})
	if len(fallbackCaps.Transports) != 0 {
		t.Fatalf("expected blank transport to be omitted, got %+v", fallbackCaps.Transports)
	}
	if !ClientSupportsEncoding(fallbackCaps, "") {
		t.Fatalf("expected blank encoding requests to default to json support")
	}

	if protocolMajor("") != "" {
		t.Fatalf("expected empty protocol major for blank input")
	}
	if protocolMajor("v2") != "2" {
		t.Fatalf("expected protocol major without dot to preserve full version")
	}
	if !ClientProtocolCompatible(ClientCapabilities{ProtocolVersion: " v2 "}, ClientCapabilities{ProtocolVersion: "2.4"}) {
		t.Fatalf("expected normalized protocol versions with matching majors to be compatible")
	}
	if ClientProtocolCompatible(ClientCapabilities{}, ClientCapabilities{ProtocolVersion: "v1"}) {
		t.Fatalf("expected missing local protocol version to be incompatible")
	}

	if ClientSupportsTopic(ClientCapabilities{}, "") {
		t.Fatalf("expected blank topic to be unsupported")
	}
}

func TestSurfaceSignalDecodeAndValidationBranches(t *testing.T) {
	decoded, err := DecodeWindowEnvelope[SurfaceSignal](WindowEnvelope{
		Name:    "ops",
		Source:  "popup-1",
		SentAt:  time.Date(2026, 3, 25, 15, 0, 0, 0, time.UTC),
		Payload: "bad",
	})
	if !IsCode(err, CodeDecode) {
		t.Fatalf("expected malformed window payload decode error, got %v", err)
	}
	if decoded.Name != "ops" || decoded.Source != "popup-1" || decoded.Payload.Kind != "" {
		t.Fatalf("unexpected malformed decode window envelope result: %+v", decoded)
	}

	var windowHandler func(WindowEnvelope, error)
	window := WindowChannel{
		name: func() string { return "ops" },
		subscribe: func(handler func(WindowEnvelope, error)) (Subscription, error) {
			windowHandler = handler
			return Subscription{}, nil
		},
		publish: func(any) error { return nil },
	}

	callCount := 0
	var lastDecoded DecodedWindowEnvelope[SurfaceSignal]
	var lastErr error
	if _, err := SubscribeSurfaceSignals(window, func(message DecodedWindowEnvelope[SurfaceSignal], err error) {
		callCount++
		lastDecoded = message
		lastErr = err
	}); err != nil {
		t.Fatalf("subscribe surface signals: %v", err)
	}

	windowHandler(WindowEnvelope{Name: "ops", Payload: "bad"}, nil)
	if !IsCode(lastErr, CodeDecode) || lastDecoded.Name != "ops" {
		t.Fatalf("expected surface signal decode error callback, got msg=%+v err=%v", lastDecoded, lastErr)
	}

	windowHandler(WindowEnvelope{}, context.Canceled)
	if lastErr != context.Canceled {
		t.Fatalf("expected surface signal transport error passthrough, got %v", lastErr)
	}
	if callCount != 2 {
		t.Fatalf("expected two surface signal callback invocations, got %d", callCount)
	}

	checks := []SurfaceSignal{
		{Kind: ""},
		{Kind: SurfaceSignalSession},
		{Kind: SurfaceSignalRoute, Route: &SurfaceRouteSignal{}},
		{Kind: SurfaceSignalSelection, Selection: &SurfaceSelectionSignal{}},
		{Kind: SurfaceSignalIntent, Intent: &SurfaceIntentSignal{}},
	}
	for _, signal := range checks {
		if err := PublishSurfaceSignal(window, signal); !IsCode(err, CodeInvalid) {
			t.Fatalf("expected invalid surface signal error for %+v, got %v", signal, err)
		}
	}
}
