//go:build !js || !wasm
// +build !js !wasm

package interop

import (
	"context"
	"testing"
	"time"
)

func TestClipboardTimerMediaAndWindowEnvWrappers(parseT *testing.T) {
	parseCtx := context.TODO()

	parseEnv := WindowEnv{}
	if _, parseOk := parseEnv.Lookup("x"); parseOk {
		parseT.Fatalf("expected missing lookup when env lookup is nil")
	}
	if _, parseOk2 := parseEnv.LookupString("x"); parseOk2 {
		parseT.Fatalf("expected missing lookup string for nil env")
	}
	if parseGot := parseEnv.String("x", "fallback"); parseGot != "fallback" {
		parseT.Fatalf("expected fallback string, got %q", parseGot)
	}

	parseClipboard := Clipboard{}
	if parseErr := parseClipboard.WriteText(parseCtx, "hello"); !IsCode(parseErr, CodeUnavailable) {
		parseT.Fatalf("expected unavailable clipboard write, got %v", parseErr)
	}
	if _, parseErr2 := parseClipboard.ReadText(parseCtx); !IsCode(parseErr2, CodeUnavailable) {
		parseT.Fatalf("expected unavailable clipboard read, got %v", parseErr2)
	}

	parseClipboard = Clipboard{
		writeText: func(parseCtx2 context.Context, parseText2 string) error {
			if parseCtx2 == nil || parseText2 != "hello" {
				parseT.Fatalf("unexpected write params: ctx=%v text=%q", parseCtx2, parseText2)
			}
			return nil
		},
		readText: func(parseCtx3 context.Context) (string, error) {
			if parseCtx3 == nil {
				parseT.Fatalf("expected context fallback")
			}
			return "hello", nil
		},
	}
	if parseErr3 := parseClipboard.WriteText(parseCtx, "hello"); parseErr3 != nil {
		parseT.Fatalf("clipboard write: %v", parseErr3)
	}
	if parseText, parseErr4 := parseClipboard.ReadText(parseCtx); parseErr4 != nil || parseText != "hello" {
		parseT.Fatalf("clipboard read = %q err=%v", parseText, parseErr4)
	}

	parseTimer := Timer{}
	if parseErr5 := parseTimer.Cancel(); !IsCode(parseErr5, CodeUnavailable) {
		parseT.Fatalf("expected unavailable timer cancel, got %v", parseErr5)
	}
	parseTimer = Timer{cancel: func() error { return nil }}
	if parseErr6 := parseTimer.Cancel(); parseErr6 != nil {
		parseT.Fatalf("timer cancel: %v", parseErr6)
	}

	parseMedia := MediaQueryList{}
	if parseMedia.Matches() {
		parseT.Fatalf("expected default media query matches=false")
	}
	if parseMedia.Media() != "" {
		parseT.Fatalf("expected default media query media empty")
	}
	if _, parseErr7 := parseMedia.Subscribe(func(MediaQueryEvent) {}); !IsCode(parseErr7, CodeUnavailable) {
		parseT.Fatalf("expected unavailable media subscribe, got %v", parseErr7)
	}
	parseMedia = MediaQueryList{
		matches: func() bool { return true },
		media:   func() string { return "(min-width: 1200px)" },
		subscribe: func(handler func(MediaQueryEvent)) (Subscription, error) {
			handler(MediaQueryEvent{Matches: true, Media: "(min-width: 1200px)"})
			return Subscription{}, nil
		},
	}
	if !parseMedia.Matches() || parseMedia.Media() == "" {
		parseT.Fatalf("expected media query accessors")
	}
	if _, parseErr8 := parseMedia.Subscribe(func(MediaQueryEvent) {}); parseErr8 != nil {
		parseT.Fatalf("media subscribe: %v", parseErr8)
	}
}

func TestChannelAndWorkerScopeZeroCoverageBranches(parseT *testing.T) {
	parseCrossTab := CrossTabChannel{}
	if parseErr := parseCrossTab.Close(); !IsCode(parseErr, CodeUnavailable) {
		parseT.Fatalf("expected unavailable cross-tab close, got %v", parseErr)
	}
	parseCrossTab = CrossTabChannel{close: func() error { return nil }}
	if parseErr2 := parseCrossTab.Close(); parseErr2 != nil {
		parseT.Fatalf("cross-tab close: %v", parseErr2)
	}

	parseWindow := WindowChannel{}
	if parseGot := parseWindow.TargetOrigin(); parseGot != "" {
		parseT.Fatalf("expected empty target origin, got %q", parseGot)
	}
	if parseErr3 := parseWindow.Focus(); !IsCode(parseErr3, CodeUnavailable) {
		parseT.Fatalf("expected unavailable window focus, got %v", parseErr3)
	}
	if parseErr4 := parseWindow.Close(); !IsCode(parseErr4, CodeUnavailable) {
		parseT.Fatalf("expected unavailable window close, got %v", parseErr4)
	}
	if parseWindow.Closed() {
		parseT.Fatalf("expected closed=false when closed callback is nil")
	}

	parseWindow = WindowChannel{
		targetOrigin: func() string { return "https://atlas.local" },
		focus:        func() error { return nil },
		close:        func() error { return nil },
		closed:       func() bool { return true },
	}
	if parseGot2 := parseWindow.TargetOrigin(); parseGot2 != "https://atlas.local" {
		parseT.Fatalf("target origin = %q", parseGot2)
	}
	if parseErr5 := parseWindow.Focus(); parseErr5 != nil {
		parseT.Fatalf("window focus: %v", parseErr5)
	}
	if parseErr6 := parseWindow.Close(); parseErr6 != nil {
		parseT.Fatalf("window close: %v", parseErr6)
	}
	if !parseWindow.Closed() {
		parseT.Fatalf("expected closed=true")
	}

	parseScope := WorkerScope{}
	if parseErr7 := parseScope.Post(WorkerMessage{}); !IsCode(parseErr7, CodeUnavailable) {
		parseT.Fatalf("expected unavailable worker scope post, got %v", parseErr7)
	}
	if _, parseErr8 := parseScope.Subscribe(func(WorkerMessage, error) {}); !IsCode(parseErr8, CodeUnavailable) {
		parseT.Fatalf("expected unavailable worker scope subscribe, got %v", parseErr8)
	}

	parsePosted := make([]WorkerMessage, 0, 6)
	parseScope = WorkerScope{
		post: func(parseMessage WorkerMessage) error {
			parsePosted = append(parsePosted, parseMessage)
			return nil
		},
		subscribe: func(handler func(WorkerMessage, error)) (Subscription, error) {
			handler(WorkerMessage{Phase: "message", Name: "task", Payload: map[string]any{"ok": true}}, nil)
			return Subscription{}, nil
		},
	}
	if _, parseErr9 := parseScope.Subscribe(func(WorkerMessage, error) {}); parseErr9 != nil {
		parseT.Fatalf("worker scope subscribe: %v", parseErr9)
	}
	_ = parseScope.Ready("task")
	_ = parseScope.Message("task", map[string]any{"step": 1})
	_ = parseScope.Progress("id-1", "task", map[string]any{"progress": 50})
	_ = parseScope.Result("id-1", "task", map[string]any{"done": true})
	_ = parseScope.Error("id-1", "task", "failed", map[string]any{"reason": "timeout"})
	if len(parsePosted) != 5 {
		parseT.Fatalf("expected worker scope helper posts, got %d", len(parsePosted))
	}
}

func TestDecodedSubscribeAndPublishWindowHelpers(parseT *testing.T) {
	parseEnvelope := CrossTabEnvelope{
		Name:     "clients",
		Source:   "tab-1",
		Sequence: 7,
		SentAt:   time.Date(2026, 3, 24, 18, 0, 0, 0, time.UTC),
		Payload:  map[string]any{"count": 2},
	}
	parseDecoded, parseErr := DecodeCrossTabEnvelope[map[string]int](parseEnvelope)
	if parseErr != nil {
		parseT.Fatalf("decode cross-tab envelope: %v", parseErr)
	}
	if parseDecoded.Name != "clients" || parseDecoded.Payload["count"] != 2 {
		parseT.Fatalf("unexpected decoded cross-tab payload: %+v", parseDecoded)
	}
	if _, parseErr2 := DecodeCrossTabEnvelope[map[string]int](CrossTabEnvelope{Name: "clients", Payload: "bad"}); !IsCode(parseErr2, CodeDecode) {
		parseT.Fatalf("expected decode error for malformed cross-tab payload, got %v", parseErr2)
	}

	var parseCrossTabHandler func(CrossTabEnvelope, error)
	parseChannel := CrossTabChannel{
		name: func() string { return "clients" },
		publish: func(parsePayload any) error {
			_ = parsePayload
			return nil
		},
		subscribe: func(handler func(CrossTabEnvelope, error)) (Subscription, error) {
			parseCrossTabHandler = handler
			return Subscription{}, nil
		},
	}
	if _, parseErr3 := SubscribeDecodedCrossTab[map[string]int](parseChannel, nil); !IsCode(parseErr3, CodeInvalid) {
		parseT.Fatalf("expected nil handler error for decoded cross-tab subscribe, got %v", parseErr3)
	}
	var parseGotDecoded DecodedCrossTabEnvelope[map[string]int]
	var parseGotErr error
	if _, parseErr4 := SubscribeDecodedCrossTab(parseChannel, func(parseMsg DecodedCrossTabEnvelope[map[string]int], parseErr13 error) {
		parseGotDecoded = parseMsg
		parseGotErr = parseErr13
	}); parseErr4 != nil {
		parseT.Fatalf("subscribe decoded cross-tab: %v", parseErr4)
	}
	parseCrossTabHandler(parseEnvelope, nil)
	if parseGotErr != nil || parseGotDecoded.Payload["count"] != 2 {
		parseT.Fatalf("decoded cross-tab callback mismatch: msg=%+v err=%v", parseGotDecoded, parseGotErr)
	}
	parseCrossTabHandler(CrossTabEnvelope{}, context.DeadlineExceeded)
	if parseGotErr != context.DeadlineExceeded {
		parseT.Fatalf("expected subscribe decoded cross-tab to pass through transport error")
	}

	var parseWindowHandler func(WindowEnvelope, error)
	parseWindow := WindowChannel{
		name: func() string { return "atlas-window" },
		publish: func(parsePayload2 any) error {
			_ = parsePayload2
			return nil
		},
		subscribe: func(handler func(WindowEnvelope, error)) (Subscription, error) {
			parseWindowHandler = handler
			return Subscription{}, nil
		},
	}
	if _, parseErr5 := SubscribeDecodedWindow[map[string]int](parseWindow, nil); !IsCode(parseErr5, CodeInvalid) {
		parseT.Fatalf("expected nil handler error for decoded window subscribe, got %v", parseErr5)
	}
	if _, parseErr6 := SubscribeSurfaceSignals(parseWindow, nil); !IsCode(parseErr6, CodeInvalid) {
		parseT.Fatalf("expected nil handler error for surface signal subscribe, got %v", parseErr6)
	}

	var parseDecodedWindow DecodedWindowEnvelope[map[string]int]
	if _, parseErr7 := SubscribeDecodedWindow(parseWindow, func(parseMsg2 DecodedWindowEnvelope[map[string]int], parseErr14 error) {
		if parseErr14 == nil {
			parseDecodedWindow = parseMsg2
		}
	}); parseErr7 != nil {
		parseT.Fatalf("subscribe decoded window: %v", parseErr7)
	}
	parseWindowHandler(WindowEnvelope{Name: "atlas-window", Payload: map[string]any{"count": 4}}, nil)
	if parseDecodedWindow.Payload["count"] != 4 {
		parseT.Fatalf("expected decoded window payload")
	}

	parseSelf := ClientIdentity{ID: "ops-1", App: "atlas", Surface: "window", Role: "operator"}
	if parseErr8 := PublishClientHelloWindow(parseWindow, parseSelf); parseErr8 != nil {
		parseT.Fatalf("publish client hello window: %v", parseErr8)
	}
	if parseErr9 := PublishClientHelloWindowWithCapabilities(parseWindow, parseSelf, ClientCapabilities{ProtocolVersion: "v1"}); parseErr9 != nil {
		parseT.Fatalf("publish hello window with capabilities: %v", parseErr9)
	}
	if parseErr10 := PublishClientGoodbye(parseChannel, parseSelf); parseErr10 != nil {
		parseT.Fatalf("publish client goodbye: %v", parseErr10)
	}
	if parseErr11 := PublishClientGoodbyeWindow(parseWindow, parseSelf); parseErr11 != nil {
		parseT.Fatalf("publish client goodbye window: %v", parseErr11)
	}
	if parseErr12 := PublishClientEvent(parseChannel, "inventory:event", parseSelf, map[string]any{"sku": "SKU-1"}); parseErr12 != nil {
		parseT.Fatalf("publish client event: %v", parseErr12)
	}

	parseCaps := defaultWindowClientCapabilities(parseWindow)
	if parseCaps.ProtocolVersion != "v1" || len(parseCaps.Transports) != 1 || parseCaps.Transports[0] != "window-message" {
		parseT.Fatalf("unexpected default window capabilities: %+v", parseCaps)
	}
}

func TestUnavailableBranchesAcrossInteropWrappers(parseT *testing.T) {
	parseCtx := context.TODO()

	var parseStorage Storage
	_, _, parseErr := parseStorage.GetItem("k")
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	requireInteropCode(parseT, parseStorage.SetItem("k", "v"), CodeUnavailable)
	_, parseErr = parseStorage.GetMany("k")
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	requireInteropCode(parseT, parseStorage.RemoveItem("k"), CodeUnavailable)
	requireInteropCode(parseT, parseStorage.Clear(), CodeUnavailable)
	_, parseErr = parseStorage.Len()
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	_, _, parseErr = parseStorage.Key(0)
	requireInteropCode(parseT, parseErr, CodeUnavailable)

	var parsePersistent PersistentStore
	_, _, parseErr = parsePersistent.GetItem(parseCtx, "k")
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	requireInteropCode(parseT, parsePersistent.SetItem(parseCtx, "k", "v"), CodeUnavailable)
	requireInteropCode(parseT, parsePersistent.RemoveItem(parseCtx, "k"), CodeUnavailable)
	requireInteropCode(parseT, parsePersistent.Clear(parseCtx), CodeUnavailable)
	_, parseErr = parsePersistent.Keys(parseCtx)
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	_, parseErr = parsePersistent.Len(parseCtx)
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	if parseErr2 := parsePersistent.Close(); parseErr2 != nil {
		parseT.Fatalf("expected nil close for zero-value persistent store, got %v", parseErr2)
	}

	var parseLocation Location
	requireInteropCode(parseT, parseLocation.Assign("/next"), CodeUnavailable)
	requireInteropCode(parseT, parseLocation.Replace("/next"), CodeUnavailable)
	requireInteropCode(parseT, parseLocation.Reload(), CodeUnavailable)
	if parseLocation.Href() != "" || parseLocation.Pathname() != "" || parseLocation.Search() != "" || parseLocation.Hash() != "" || parseLocation.Origin() != "" {
		parseT.Fatalf("expected zero-value location getters to return empty strings")
	}

	var parseHistory History
	_, parseErr = parseHistory.Len()
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	_, parseErr = parseHistory.State()
	requireInteropCode(parseT, parseErr, CodeUnavailable)
	requireInteropCode(parseT, parseHistory.Back(), CodeUnavailable)
	requireInteropCode(parseT, parseHistory.Forward(), CodeUnavailable)
	requireInteropCode(parseT, parseHistory.Go(1), CodeUnavailable)
	requireInteropCode(parseT, parseHistory.PushState(nil, "", "/"), CodeUnavailable)
	requireInteropCode(parseT, parseHistory.ReplaceState(nil, "", "/"), CodeUnavailable)

	var parseEvents EventTarget
	requireInteropCode(parseT, parseEvents.Dispatch("ready", nil), CodeUnavailable)
	if _, parseErr3 := parseEvents.Listen("ready", func(BrowserEvent) {}); !IsCode(parseErr3, CodeUnavailable) {
		parseT.Fatalf("expected unavailable event listen, got %v", parseErr3)
	}
	if _, parseErr4 := parseEvents.Subscribe("ready", func(CustomEvent) {}); !IsCode(parseErr4, CodeUnavailable) {
		parseT.Fatalf("expected unavailable event subscribe, got %v", parseErr4)
	}

	var parseElement Element
	if parseElement.TagName() != "" || parseElement.ID() != "" || parseElement.ClassName() != "" {
		parseT.Fatalf("expected empty zero-value element metadata")
	}
	requireInteropCode(parseT, parseElement.Focus(), CodeUnavailable)
	requireInteropCode(parseT, parseElement.Blur(), CodeUnavailable)
	requireInteropCode(parseT, parseElement.Click(), CodeUnavailable)
	requireInteropCode(parseT, parseElement.SetScrollTop(1), CodeUnavailable)
	requireInteropCode(parseT, parseElement.ScrollIntoView(), CodeUnavailable)
	if _, parseErr5 := parseElement.BoundingClientRect(); !IsCode(parseErr5, CodeUnavailable) {
		parseT.Fatalf("expected unavailable bounding rect, got %v", parseErr5)
	}
	if _, parseErr6 := parseElement.Events(); !IsCode(parseErr6, CodeUnavailable) {
		parseT.Fatalf("expected unavailable element events, got %v", parseErr6)
	}
	if _, parseErr7 := parseElement.Listen("ready", func(BrowserEvent) {}); !IsCode(parseErr7, CodeUnavailable) {
		parseT.Fatalf("expected unavailable element listen, got %v", parseErr7)
	}
	if _, parseErr8 := parseElement.Subscribe("ready", func(CustomEvent) {}); !IsCode(parseErr8, CodeUnavailable) {
		parseT.Fatalf("expected unavailable element subscribe, got %v", parseErr8)
	}
	if parseErr9 := parseElement.Dispatch("ready", nil); !IsCode(parseErr9, CodeUnavailable) {
		parseT.Fatalf("expected unavailable element dispatch, got %v", parseErr9)
	}
	if _, parseErr10 := parseElement.ObserveResize(func(ResizeEntry) {}); !IsCode(parseErr10, CodeUnavailable) {
		parseT.Fatalf("expected unavailable resize observe, got %v", parseErr10)
	}
	if _, parseErr11 := parseElement.ObserveIntersection(func(IntersectionEntry) {}); !IsCode(parseErr11, CodeUnavailable) {
		parseT.Fatalf("expected unavailable intersection observe, got %v", parseErr11)
	}
	if _, _, _, parseErr12 := parseElement.ScrollMetrics(); !IsCode(parseErr12, CodeUnavailable) {
		parseT.Fatalf("expected unavailable scroll metrics, got %v", parseErr12)
	}

	var parseDocument Document
	if _, _, parseErr13 := parseDocument.ElementByID("root"); !IsCode(parseErr13, CodeUnavailable) {
		parseT.Fatalf("expected unavailable ElementByID, got %v", parseErr13)
	}
	if _, _, parseErr14 := parseDocument.QuerySelector("#root"); !IsCode(parseErr14, CodeUnavailable) {
		parseT.Fatalf("expected unavailable QuerySelector, got %v", parseErr14)
	}
	if _, parseErr15 := parseDocument.ElementsByID("root"); !IsCode(parseErr15, CodeUnavailable) {
		parseT.Fatalf("expected unavailable ElementsByID, got %v", parseErr15)
	}

	var parseModule Module
	if _, parseErr16 := parseModule.Call(parseCtx, "x"); !IsCode(parseErr16, CodeUnavailable) {
		parseT.Fatalf("expected unavailable module call, got %v", parseErr16)
	}
	if _, parseErr17 := parseModule.CallDefault(parseCtx); !IsCode(parseErr17, CodeUnavailable) {
		parseT.Fatalf("expected unavailable module default call, got %v", parseErr17)
	}
	if _, parseErr18 := parseModule.Value(parseCtx, "x"); !IsCode(parseErr18, CodeUnavailable) {
		parseT.Fatalf("expected unavailable module value, got %v", parseErr18)
	}
	if parseErr19 := parseModule.Dispose(); !IsCode(parseErr19, CodeUnavailable) {
		parseT.Fatalf("expected unavailable module dispose, got %v", parseErr19)
	}

	var parseCrossTab CrossTabChannel
	if parseCrossTab.Name() != "" || parseCrossTab.Transport() != "" {
		parseT.Fatalf("expected empty zero-value cross-tab metadata")
	}
	if parseErr20 := parseCrossTab.Publish(nil); !IsCode(parseErr20, CodeUnavailable) {
		parseT.Fatalf("expected unavailable cross-tab publish, got %v", parseErr20)
	}
	if _, parseErr21 := parseCrossTab.Subscribe(func(CrossTabEnvelope, error) {}); !IsCode(parseErr21, CodeUnavailable) {
		parseT.Fatalf("expected unavailable cross-tab subscribe, got %v", parseErr21)
	}

	var parseWindow WindowChannel
	if parseWindow.Name() != "" {
		parseT.Fatalf("expected empty zero-value window name")
	}
	if parseErr22 := parseWindow.Publish(nil); !IsCode(parseErr22, CodeUnavailable) {
		parseT.Fatalf("expected unavailable window publish, got %v", parseErr22)
	}
	if _, parseErr23 := parseWindow.Subscribe(func(WindowEnvelope, error) {}); !IsCode(parseErr23, CodeUnavailable) {
		parseT.Fatalf("expected unavailable window subscribe, got %v", parseErr23)
	}

	var parseWorker Worker
	if parseErr24 := parseWorker.Post("msg"); !IsCode(parseErr24, CodeUnavailable) {
		parseT.Fatalf("expected unavailable worker post, got %v", parseErr24)
	}
	if parseErr24b := parseWorker.PostPorts("msg"); !IsCode(parseErr24b, CodeUnavailable) {
		parseT.Fatalf("expected unavailable worker post ports, got %v", parseErr24b)
	}
	if _, parseErr25 := parseWorker.Subscribe(func(WorkerMessage, error) {}); !IsCode(parseErr25, CodeUnavailable) {
		parseT.Fatalf("expected unavailable worker subscribe, got %v", parseErr25)
	}
	if _, parseErr26 := parseWorker.Request(parseCtx, "task", nil, nil); !IsCode(parseErr26, CodeUnavailable) {
		parseT.Fatalf("expected unavailable worker request, got %v", parseErr26)
	}
	if parseErr27 := parseWorker.Terminate(); !IsCode(parseErr27, CodeUnavailable) {
		parseT.Fatalf("expected unavailable worker terminate, got %v", parseErr27)
	}
	if parseErr28 := parseWorker.Restart(parseCtx); !IsCode(parseErr28, CodeUnavailable) {
		parseT.Fatalf("expected unavailable worker restart, got %v", parseErr28)
	}
	if _, parseErr29 := SubscribeDecodedWorker[map[string]int](parseWorker, nil); !IsCode(parseErr29, CodeInvalid) {
		parseT.Fatalf("expected invalid nil-handler decoded worker subscribe, got %v", parseErr29)
	}

	parseWorker = Worker{
		subscribe: func(handler func(WorkerMessage, error)) (Subscription, error) {
			handler(WorkerMessage{}, context.DeadlineExceeded)
			handler(WorkerMessage{Name: "task", Payload: "bad"}, nil)
			handler(WorkerMessage{Name: "task", Payload: map[string]any{"n": 2}}, nil)
			return Subscription{}, nil
		},
		request: func(_ context.Context, _ string, _ any, parseOnProgress func(WorkerMessage, error)) (WorkerMessage, error) {
			if parseOnProgress != nil {
				parseOnProgress(WorkerMessage{Name: "task", Payload: "bad"}, nil)
			}
			return WorkerMessage{Name: "task", Payload: map[string]any{"n": 9}}, nil
		},
	}
	parseCalls := 0
	if _, parseErr30 := SubscribeDecodedWorker(parseWorker, func(DecodedWorkerMessage[map[string]int], error) {
		parseCalls++
	}); parseErr30 != nil {
		parseT.Fatalf("subscribe decoded worker: %v", parseErr30)
	}
	if parseCalls != 3 {
		parseT.Fatalf("expected decoded worker callback invocations, got %d", parseCalls)
	}
	if parseResult, parseErr31 := RequestWorkerDecoded[map[string]int, map[string]int, map[string]int](parseCtx, parseWorker, "task", map[string]int{"x": 1}, func(DecodedWorkerMessage[map[string]int], error) {}); parseErr31 != nil || parseResult["n"] != 9 {
		parseT.Fatalf("request worker decoded result=%v err=%v", parseResult, parseErr31)
	}

	var parsePort MessagePort
	if parseErr31b := parsePort.Post("msg"); !IsCode(parseErr31b, CodeUnavailable) {
		parseT.Fatalf("expected unavailable message-port post, got %v", parseErr31b)
	}
	if parseErr31c := parsePort.PostPorts("msg"); !IsCode(parseErr31c, CodeUnavailable) {
		parseT.Fatalf("expected unavailable message-port post ports, got %v", parseErr31c)
	}
	if _, parseErr31d := parsePort.Subscribe(func(MessagePortMessage, error) {}); !IsCode(parseErr31d, CodeUnavailable) {
		parseT.Fatalf("expected unavailable message-port subscribe, got %v", parseErr31d)
	}
	if parseErr31e := parsePort.Close(); !IsCode(parseErr31e, CodeUnavailable) {
		parseT.Fatalf("expected unavailable message-port close, got %v", parseErr31e)
	}
	if _, parseErr31f := SubscribeDecodedMessagePort[map[string]int](parsePort, nil); !IsCode(parseErr31f, CodeInvalid) {
		parseT.Fatalf("expected invalid nil-handler decoded message-port subscribe, got %v", parseErr31f)
	}

	parsePort = MessagePort{
		subscribe: func(parseHandler func(MessagePortMessage, error)) (Subscription, error) {
			parseHandler(MessagePortMessage{}, context.DeadlineExceeded)
			parseHandler(MessagePortMessage{Payload: "bad"}, nil)
			parseHandler(MessagePortMessage{Payload: map[string]any{"n": 4}}, nil)
			return Subscription{}, nil
		},
	}
	parsePortCalls := 0
	if _, parseErr31g := SubscribeDecodedMessagePort(parsePort, func(DecodedMessagePortMessage[map[string]int], error) {
		parsePortCalls++
	}); parseErr31g != nil {
		parseT.Fatalf("subscribe decoded message port: %v", parseErr31g)
	}
	if parsePortCalls != 3 {
		parseT.Fatalf("expected decoded message-port callback invocations, got %d", parsePortCalls)
	}

	parseDecodedPort, parseErr31h := DecodeMessagePortMessage[map[string]int](MessagePortMessage{Payload: map[string]any{"n": 8}})
	if parseErr31h != nil || parseDecodedPort.Payload["n"] != 8 {
		parseT.Fatalf("decode message-port result=%v err=%v", parseDecodedPort, parseErr31h)
	}

	var parseShared SharedBuffer
	if parseShared.GetByteLength() != 0 || parseShared.GetInt32Length() != 0 {
		parseT.Fatalf("expected zero-value shared buffer lengths, got bytes=%d int32=%d", parseShared.GetByteLength(), parseShared.GetInt32Length())
	}
	if _, parseErr31i := parseShared.ReadBytes(0, make([]byte, 4)); !IsCode(parseErr31i, CodeUnavailable) {
		parseT.Fatalf("expected unavailable shared-buffer read, got %v", parseErr31i)
	}
	if _, parseErr31j := parseShared.WriteBytes(0, []byte{1, 2}); !IsCode(parseErr31j, CodeUnavailable) {
		parseT.Fatalf("expected unavailable shared-buffer write, got %v", parseErr31j)
	}
	if _, parseErr31k := parseShared.LoadInt32(0); !IsCode(parseErr31k, CodeUnavailable) {
		parseT.Fatalf("expected unavailable shared-buffer load, got %v", parseErr31k)
	}
	if parseErr31l := parseShared.StoreInt32(0, 1); !IsCode(parseErr31l, CodeUnavailable) {
		parseT.Fatalf("expected unavailable shared-buffer store, got %v", parseErr31l)
	}
	if _, parseErr31m := parseShared.AddInt32(0, 1); !IsCode(parseErr31m, CodeUnavailable) {
		parseT.Fatalf("expected unavailable shared-buffer add, got %v", parseErr31m)
	}
	if _, parseErr31n := parseShared.SubInt32(0, 1); !IsCode(parseErr31n, CodeUnavailable) {
		parseT.Fatalf("expected unavailable shared-buffer sub, got %v", parseErr31n)
	}
	if _, parseErr31o := parseShared.AndInt32(0, 1); !IsCode(parseErr31o, CodeUnavailable) {
		parseT.Fatalf("expected unavailable shared-buffer and, got %v", parseErr31o)
	}
	if _, parseErr31p := parseShared.OrInt32(0, 1); !IsCode(parseErr31p, CodeUnavailable) {
		parseT.Fatalf("expected unavailable shared-buffer or, got %v", parseErr31p)
	}
	if _, parseErr31q := parseShared.XorInt32(0, 1); !IsCode(parseErr31q, CodeUnavailable) {
		parseT.Fatalf("expected unavailable shared-buffer xor, got %v", parseErr31q)
	}
	if _, parseErr31r := parseShared.ExchangeInt32(0, 1); !IsCode(parseErr31r, CodeUnavailable) {
		parseT.Fatalf("expected unavailable shared-buffer exchange, got %v", parseErr31r)
	}
	if _, parseErr31s := parseShared.CompareExchangeInt32(0, 1, 2); !IsCode(parseErr31s, CodeUnavailable) {
		parseT.Fatalf("expected unavailable shared-buffer compare-exchange, got %v", parseErr31s)
	}
	if _, parseErr31t := parseShared.WaitInt32(0, 1, time.Millisecond); !IsCode(parseErr31t, CodeUnavailable) {
		parseT.Fatalf("expected unavailable shared-buffer wait, got %v", parseErr31t)
	}
	if _, parseErr31u := parseShared.NotifyInt32(0, 1); !IsCode(parseErr31u, CodeUnavailable) {
		parseT.Fatalf("expected unavailable shared-buffer notify, got %v", parseErr31u)
	}

	var parsePool WorkerPool
	if parsePool.GetSize() != 0 || parsePool.GetQueueLimit() != 0 {
		parseT.Fatalf("expected zero-value worker pool limits, got size=%d queue=%d", parsePool.GetSize(), parsePool.GetQueueLimit())
	}
	if _, parseErr31v := parsePool.Request(parseCtx, "task", nil, nil); !IsCode(parseErr31v, CodeUnavailable) {
		parseT.Fatalf("expected unavailable worker-pool request, got %v", parseErr31v)
	}
	if parseErr31w := parsePool.Close(); !IsCode(parseErr31w, CodeUnavailable) {
		parseT.Fatalf("expected unavailable worker-pool close, got %v", parseErr31w)
	}
	if parseErr31x := parsePool.Drain(parseCtx); !IsCode(parseErr31x, CodeUnavailable) {
		parseT.Fatalf("expected unavailable worker-pool drain, got %v", parseErr31x)
	}
}

func TestClientDecodeScalarHelpers(parseT *testing.T) {
	parseFields := map[string]any{
		"name": "atlas",
		"kind": 7,
	}
	if parseGot := stringField(parseFields, "name"); parseGot != "atlas" {
		parseT.Fatalf("stringField string = %q", parseGot)
	}
	if parseGot2 := stringField(parseFields, "kind"); parseGot2 != "" {
		parseT.Fatalf("stringField non-string = %q", parseGot2)
	}
	if parseGot3 := stringField(parseFields, "missing"); parseGot3 != "" {
		parseT.Fatalf("stringField missing = %q", parseGot3)
	}

	if parseGot4 := intField(7); parseGot4 != 7 {
		parseT.Fatalf("intField int = %d", parseGot4)
	}
	if parseGot5 := intField(int64(8)); parseGot5 != 8 {
		parseT.Fatalf("intField int64 = %d", parseGot5)
	}
	if parseGot6 := intField(9.75); parseGot6 != 9 {
		parseT.Fatalf("intField float64 = %d", parseGot6)
	}
	if parseGot7 := intField("bad"); parseGot7 != 0 {
		parseT.Fatalf("intField default = %d", parseGot7)
	}

	if parseParsed, parseOk := clientTimeField(" "); parseOk || !parseParsed.IsZero() {
		parseT.Fatalf("clientTimeField blank = %v ok=%t", parseParsed, parseOk)
	}
	if parseParsed2, parseOk2 := clientTimeField("not-a-time"); parseOk2 || !parseParsed2.IsZero() {
		parseT.Fatalf("clientTimeField invalid = %v ok=%t", parseParsed2, parseOk2)
	}

	parseExpected := time.Date(2026, 3, 25, 13, 30, 0, 123456789, time.UTC)
	parseParsed3, parseOk3 := clientTimeField(parseExpected.Format(time.RFC3339Nano))
	if !parseOk3 || !parseParsed3.Equal(parseExpected) {
		parseT.Fatalf("clientTimeField valid = %v ok=%t", parseParsed3, parseOk3)
	}
}

func TestClientCapabilityDefaultsAndProtocolHelpers(parseT *testing.T) {
	parseBroadcastCaps := defaultCrossTabClientCapabilities(CrossTabChannel{
		transport: func() string { return "broadcast-channel" },
	})
	if parseBroadcastCaps.ProtocolVersion != "v1" || len(parseBroadcastCaps.Transports) != 1 || parseBroadcastCaps.Transports[0] != "broadcast-channel" {
		parseT.Fatalf("unexpected broadcast default capabilities: %+v", parseBroadcastCaps)
	}
	if !ClientSupportsEncoding(parseBroadcastCaps, ClientPayloadBinary) {
		parseT.Fatalf("expected broadcast channel defaults to include binary encoding")
	}

	parseFallbackCaps := defaultCrossTabClientCapabilities(CrossTabChannel{
		transport: func() string { return "  " },
	})
	if len(parseFallbackCaps.Transports) != 0 {
		parseT.Fatalf("expected blank transport to be omitted, got %+v", parseFallbackCaps.Transports)
	}
	if !ClientSupportsEncoding(parseFallbackCaps, "") {
		parseT.Fatalf("expected blank encoding requests to default to json support")
	}

	if protocolMajor("") != "" {
		parseT.Fatalf("expected empty protocol major for blank input")
	}
	if protocolMajor("v2") != "2" {
		parseT.Fatalf("expected protocol major without dot to preserve full version")
	}
	if !ClientProtocolCompatible(ClientCapabilities{ProtocolVersion: " v2 "}, ClientCapabilities{ProtocolVersion: "2.4"}) {
		parseT.Fatalf("expected normalized protocol versions with matching majors to be compatible")
	}
	if ClientProtocolCompatible(ClientCapabilities{}, ClientCapabilities{ProtocolVersion: "v1"}) {
		parseT.Fatalf("expected missing local protocol version to be incompatible")
	}

	if ClientSupportsTopic(ClientCapabilities{}, "") {
		parseT.Fatalf("expected blank topic to be unsupported")
	}
}

func TestSurfaceSignalDecodeAndValidationBranches(parseT *testing.T) {
	parseDecoded, parseErr := DecodeWindowEnvelope[SurfaceSignal](WindowEnvelope{
		Name:    "ops",
		Source:  "popup-1",
		SentAt:  time.Date(2026, 3, 25, 15, 0, 0, 0, time.UTC),
		Payload: "bad",
	})
	if !IsCode(parseErr, CodeDecode) {
		parseT.Fatalf("expected malformed window payload decode error, got %v", parseErr)
	}
	if parseDecoded.Name != "ops" || parseDecoded.Source != "popup-1" || parseDecoded.Payload.Kind != "" {
		parseT.Fatalf("unexpected malformed decode window envelope result: %+v", parseDecoded)
	}

	var parseWindowHandler func(WindowEnvelope, error)
	parseWindow := WindowChannel{
		name: func() string { return "ops" },
		subscribe: func(handler func(WindowEnvelope, error)) (Subscription, error) {
			parseWindowHandler = handler
			return Subscription{}, nil
		},
		publish: func(any) error { return nil },
	}

	parseCallCount := 0
	var parseLastDecoded DecodedWindowEnvelope[SurfaceSignal]
	var parseLastErr error
	if _, parseErr2 := SubscribeSurfaceSignals(parseWindow, func(parseMessage DecodedWindowEnvelope[SurfaceSignal], parseErr4 error) {
		parseCallCount++
		parseLastDecoded = parseMessage
		parseLastErr = parseErr4
	}); parseErr2 != nil {
		parseT.Fatalf("subscribe surface signals: %v", parseErr2)
	}

	parseWindowHandler(WindowEnvelope{Name: "ops", Payload: "bad"}, nil)
	if !IsCode(parseLastErr, CodeDecode) || parseLastDecoded.Name != "ops" {
		parseT.Fatalf("expected surface signal decode error callback, got msg=%+v err=%v", parseLastDecoded, parseLastErr)
	}

	parseWindowHandler(WindowEnvelope{}, context.Canceled)
	if parseLastErr != context.Canceled {
		parseT.Fatalf("expected surface signal transport error passthrough, got %v", parseLastErr)
	}
	if parseCallCount != 2 {
		parseT.Fatalf("expected two surface signal callback invocations, got %d", parseCallCount)
	}

	parseChecks := []SurfaceSignal{
		{Kind: ""},
		{Kind: SurfaceSignalSession},
		{Kind: SurfaceSignalRoute, Route: &SurfaceRouteSignal{}},
		{Kind: SurfaceSignalSelection, Selection: &SurfaceSelectionSignal{}},
		{Kind: SurfaceSignalIntent, Intent: &SurfaceIntentSignal{}},
	}
	for _, parseSignal := range parseChecks {
		if parseErr3 := PublishSurfaceSignal(parseWindow, parseSignal); !IsCode(parseErr3, CodeInvalid) {
			parseT.Fatalf("expected invalid surface signal error for %+v, got %v", parseSignal, parseErr3)
		}
	}
}
