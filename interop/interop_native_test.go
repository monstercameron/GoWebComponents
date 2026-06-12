//go:build !js || !wasm

package interop

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNativeInteropJSONWireShapePreservesZeroTimeFields(parseT *testing.T) {
	parseCases := []struct {
		name     string
		value    any
		expected []string
	}{
		{
			name:     "cross tab envelope",
			value:    CrossTabEnvelope{},
			expected: []string{`"sentAt":"0001-01-01T00:00:00Z"`},
		},
		{
			name:     "window envelope",
			value:    WindowEnvelope{},
			expected: []string{`"sentAt":"0001-01-01T00:00:00Z"`},
		},
		{
			name:     "client message",
			value:    ClientMessage{},
			expected: []string{`"sentAt":"0001-01-01T00:00:00Z"`},
		},
		{
			name:     "surface session",
			value:    SurfaceSessionSignal{},
			expected: []string{`"expiresAt":"0001-01-01T00:00:00Z"`},
		},
	}
	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT2 *testing.T) {
			parseEncoded, parseErr := json.Marshal(parseCase.value)
			if parseErr != nil {
				parseT2.Fatalf("marshal %s: %v", parseCase.name, parseErr)
			}
			parseText := string(parseEncoded)
			for _, parseExpected := range parseCase.expected {
				if !strings.Contains(parseText, parseExpected) {
					parseT2.Fatalf("expected %s to preserve %s, got %s", parseCase.name, parseExpected, parseText)
				}
			}
		})
	}
}

func TestNativeInteropConstructorsReportUnavailable(parseT *testing.T) {
	parseChecks := []struct {
		name string
		err  error
	}{
		{name: "GetLocalStorage", err: func() error { _, parseErr := GetLocalStorage(); return parseErr }()},
		{name: "GetSessionStorage", err: func() error { _, parseErr2 := GetSessionStorage(); return parseErr2 }()},
		{name: "OpenPersistentStore", err: func() error {
			_, parseErr3 := OpenPersistentStore(context.TODO(), PersistentStoreOptions{Name: "cache"})
			return parseErr3
		}()},
		{name: "GetWindowLocation", err: func() error { _, parseErr4 := GetWindowLocation(); return parseErr4 }()},
		{name: "GetWindowHistory", err: func() error { _, parseErr5 := GetWindowHistory(); return parseErr5 }()},
		{name: "GetClipboard", err: func() error { _, parseErr6 := GetClipboard(); return parseErr6 }()},
		{name: "GetWindowEvents", err: func() error { _, parseErr7 := GetWindowEvents(); return parseErr7 }()},
		{name: "GetDocumentEvents", err: func() error { _, parseErr8 := GetDocumentEvents(); return parseErr8 }()},
		{name: "GetDocument", err: func() error { _, parseErr9 := GetDocument(); return parseErr9 }()},
		{name: "GetMediaQuery", err: func() error { _, parseErr10 := GetMediaQuery("(prefers-color-scheme: dark)"); return parseErr10 }()},
		{name: "RequestPersistentStorage", err: func() error { _, parseErrRP := RequestPersistentStorage(context.TODO()); return parseErrRP }()},
		{name: "IsStoragePersisted", err: func() error { _, parseErrIP := IsStoragePersisted(context.TODO()); return parseErrIP }()},
		{name: "ImportModule", err: func() error { _, parseErr11 := ImportModule(context.TODO(), "/demo.js"); return parseErr11 }()},
		{name: "OpenWorker", err: func() error {
			_, parseErr12 := OpenWorker(context.TODO(), WorkerOptions{URL: "/worker.js"})
			return parseErr12
		}()},
		{name: "OpenMessageChannel", err: func() error {
			_, parseErr13 := OpenMessageChannel()
			return parseErr13
		}()},
		{name: "GetSharedMemorySupport", err: func() error {
			_, parseErr14 := GetSharedMemorySupport()
			return parseErr14
		}()},
		{name: "OpenSharedBuffer", err: func() error {
			_, parseErr15 := OpenSharedBuffer(16)
			return parseErr15
		}()},
		{name: "OpenCrossTabChannel", err: func() error {
			_, parseErr16 := OpenCrossTabChannel(CrossTabChannelOptions{Name: "theme"})
			return parseErr16
		}()},
		{name: "OpenSecondaryWindowChannel", err: func() error {
			_, parseErr17 := OpenSecondaryWindowChannel(WindowChannelOptions{Name: "popup"})
			return parseErr17
		}()},
		{name: "OpenWindowOpenerChannel", err: func() error {
			_, parseErr18 := OpenWindowOpenerChannel(WindowChannelOptions{Name: "popup"})
			return parseErr18
		}()},
	}
	for _, parseCheck := range parseChecks {
		if !IsCode(parseCheck.err, CodeUnavailable) {
			parseT.Fatalf("%s: expected unavailable error, got %v", parseCheck.name, parseCheck.err)
		}
		parseInteropErr, parseOk := AsError(parseCheck.err)
		if !parseOk || parseInteropErr.Code != CodeUnavailable {
			parseT.Fatalf("%s: expected structured interop error, got %#v ok=%t", parseCheck.name, parseInteropErr, parseOk)
		}
		parseCode, parseOk := CodeOf(parseCheck.err)
		if !parseOk || parseCode != CodeUnavailable {
			parseT.Fatalf("%s: expected code lookup to report unavailable, got %q ok=%t", parseCheck.name, parseCode, parseOk)
		}
	}
}

func TestNativeWindowEnvAndNilContextStubs(parseT *testing.T) {
	parseNativeEnv, _ := GetWindowEnv()
	if parseValue, parseOk := parseNativeEnv.Lookup("demo"); parseOk || parseValue.Present() {
		parseT.Fatalf("expected native window env lookup to be empty, got value=%+v ok=%t", parseValue, parseOk)
	}
	parseCompatEnv, _ := SharedWindowEnv()
	if parseValue2, parseOk2 := parseCompatEnv.Lookup("demo"); parseOk2 || parseValue2.Present() {
		parseT.Fatalf("expected shared window env wrapper lookup to be empty, got value=%+v ok=%t", parseValue2, parseOk2)
	}

	parseChecks := []struct {
		name string
		err  error
	}{
		{name: "OpenPersistentStore", err: func() error {
			_, parseErr := OpenPersistentStore(context.TODO(), PersistentStoreOptions{Name: "cache"})
			return parseErr
		}()},
		{name: "ImportModule", err: func() error {
			_, parseErr2 := ImportModule(context.TODO(), "/demo.js")
			return parseErr2
		}()},
		{name: "OpenWorker", err: func() error {
			_, parseErr3 := OpenWorker(context.TODO(), WorkerOptions{URL: "/worker.js"})
			return parseErr3
		}()},
		{name: "OpenGoWASMWorker", err: func() error {
			_, parseErr4 := OpenGoWASMWorker(context.TODO(), GoWASMWorkerOptions{WASMURL: "/worker.wasm"})
			return parseErr4
		}()},
		{name: "GetWorkerScope", err: func() error {
			_, parseErr5 := GetWorkerScope()
			return parseErr5
		}()},
		{name: "OpenMessageChannel", err: func() error {
			_, parseErr6 := OpenMessageChannel()
			return parseErr6
		}()},
		{name: "GetSharedMemorySupport", err: func() error {
			_, parseErr7 := GetSharedMemorySupport()
			return parseErr7
		}()},
		{name: "OpenSharedBuffer", err: func() error {
			_, parseErr8 := OpenSharedBuffer(16)
			return parseErr8
		}()},
	}

	for _, parseCheck := range parseChecks {
		if !IsCode(parseCheck.err, CodeUnavailable) {
			parseT.Fatalf("%s: expected unavailable error, got %v", parseCheck.name, parseCheck.err)
		}
	}
}

func TestDecodeCustomEventProjectsTypedDetail(parseT *testing.T) {
	parseEvent := CustomEvent{
		Type: "asset-ready",
		Detail: map[string]any{
			"id":    "asset-42",
			"score": 9,
		},
	}
	parseDecoded, parseErr := DecodeCustomEvent[struct {
		ID    string `json:"id"`
		Score int    `json:"score"`
	}](parseEvent)
	if parseErr != nil {
		parseT.Fatalf("expected typed custom event detail, got %v", parseErr)
	}
	if parseDecoded.Type != "asset-ready" || parseDecoded.Detail.ID != "asset-42" || parseDecoded.Detail.Score != 9 {
		parseT.Fatalf("unexpected decoded custom event: %+v", parseDecoded)
	}
}

func TestDecodeReportsStructuredEncodeAndDecodeErrors(parseT *testing.T) {
	var parseTarget map[string]string
	if parseErr := Decode(func() {}, &parseTarget); !IsCode(parseErr, CodeEncode) {
		parseT.Fatalf("expected encode failure to report CodeEncode, got %v", parseErr)
	}

	var parseTyped struct {
		Count int `json:"count"`
	}
	if parseErr2 := Decode(map[string]any{"count": "not-a-number"}, &parseTyped); !IsCode(parseErr2, CodeDecode) {
		parseT.Fatalf("expected decode failure to report CodeDecode, got %v", parseErr2)
	}
}

func TestPublishClientHelpers(parseT *testing.T) {
	parseSelf := ClientIdentity{ID: "storefront-1", App: "atlas", Surface: "tab", Role: "storefront", Version: "v1"}

	var parseCrossTabPublished any
	parseCrossTab := CrossTabChannel{
		name: func() string { return "atlas-clients" },
		publish: func(parsePayload any) error {
			parseCrossTabPublished = parsePayload
			return nil
		},
	}
	if parseErr := PublishClientHello(parseCrossTab, parseSelf); parseErr != nil {
		parseT.Fatalf("expected hello publish to succeed, got %v", parseErr)
	}
	parseHello, parseOk := parseCrossTabPublished.(ClientMessage)
	if !parseOk {
		parseT.Fatalf("expected client hello payload, got %T", parseCrossTabPublished)
	}
	if parseHello.Kind != ClientHello || parseHello.Topic != ClientPresenceTopic || parseHello.Source.ID != "storefront-1" || parseHello.SentAt.IsZero() {
		parseT.Fatalf("unexpected hello payload: %+v", parseHello)
	}
	if parseHello.Capabilities == nil || parseHello.Capabilities.ProtocolVersion != "v1" || len(parseHello.Capabilities.Encodings) == 0 || parseHello.Capabilities.Encodings[0] != "json" {
		parseT.Fatalf("expected default cross-tab hello capabilities, got %+v", parseHello.Capabilities)
	}

	if parseErr2 := PublishClientInvalidation(parseCrossTab, "cache:inventory", parseSelf, "rev-9"); parseErr2 != nil {
		parseT.Fatalf("expected invalidation publish to succeed, got %v", parseErr2)
	}
	parseInvalidation, parseOk := parseCrossTabPublished.(ClientMessage)
	if !parseOk {
		parseT.Fatalf("expected invalidation payload, got %T", parseCrossTabPublished)
	}
	if parseInvalidation.Kind != ClientInvalidate || parseInvalidation.Topic != "cache:inventory" || parseInvalidation.Revision != "rev-9" {
		parseT.Fatalf("unexpected invalidation payload: %+v", parseInvalidation)
	}

	var parseWindowPublished any
	parseWindowChannel := WindowChannel{
		name: func() string { return "atlas-window" },
		publish: func(parsePayload2 any) error {
			parseWindowPublished = parsePayload2
			return nil
		},
	}
	if parseErr3 := PublishClientIntent(parseWindowChannel, "selection:inspect", parseSelf, "inspector-1", map[string]any{"id": "SKU-44"}); parseErr3 != nil {
		parseT.Fatalf("expected intent publish to succeed, got %v", parseErr3)
	}
	parseIntent, parseOk := parseWindowPublished.(ClientMessage)
	if !parseOk {
		parseT.Fatalf("expected intent payload, got %T", parseWindowPublished)
	}
	if parseIntent.Kind != ClientIntent || parseIntent.Topic != "selection:inspect" || parseIntent.Target != "inspector-1" {
		parseT.Fatalf("unexpected intent payload: %+v", parseIntent)
	}

	if parseErr4 := PublishClientIntent(parseWindowChannel, "selection:inspect", parseSelf, "", nil); !IsCode(parseErr4, CodeInvalid) {
		parseT.Fatalf("expected invalid target error, got %v", parseErr4)
	}
	if parseErr5 := PublishClientInvalidation(parseCrossTab, "cache:inventory", parseSelf, ""); !IsCode(parseErr5, CodeInvalid) {
		parseT.Fatalf("expected invalid revision error, got %v", parseErr5)
	}
	if parseErr6 := PublishClientHello(parseCrossTab, ClientIdentity{}); !IsCode(parseErr6, CodeInvalid) {
		parseT.Fatalf("expected invalid identity error, got %v", parseErr6)
	}
	if parseErr7 := PublishClientMessage(parseCrossTab, ClientMessage{Kind: ClientEvent, Topic: "ops", Source: parseSelf}); parseErr7 != nil {
		parseT.Fatalf("expected generic publish helper to succeed, got %v", parseErr7)
	}
	if parseErr8 := PublishClientWindowMessage(parseWindowChannel, ClientMessage{Kind: ClientResult, Topic: "clients", Source: parseSelf, Target: "inspector-1"}); parseErr8 != nil {
		parseT.Fatalf("expected generic window publish helper to succeed, got %v", parseErr8)
	}

	if parseErr9 := PublishClientBinaryCrossTab(parseCrossTab, "asset:preview", parseSelf, ClientBinaryPayload{ContentType: "application/octet-stream", Bytes: []byte{1, 2, 3}}); parseErr9 != nil {
		parseT.Fatalf("expected binary cross-tab publish helper to succeed, got %v", parseErr9)
	}
	parseBinaryCrossTab, parseOk := parseCrossTabPublished.(ClientMessage)
	if !parseOk {
		parseT.Fatalf("expected binary cross-tab payload, got %T", parseCrossTabPublished)
	}
	parseBinaryBytes, parseOk := parseBinaryCrossTab.Payload.([]byte)
	if !parseOk || len(parseBinaryBytes) != 3 || parseBinaryCrossTab.Encoding != ClientPayloadBinary || parseBinaryCrossTab.ContentType != "application/octet-stream" {
		parseT.Fatalf("unexpected binary cross-tab payload: %+v", parseBinaryCrossTab)
	}

	if parseErr10 := PublishClientBinaryWindow(parseWindowChannel, "asset:preview", parseSelf, "inspector-1", ClientBinaryPayload{ContentType: "application/octet-stream", Bytes: []byte{4, 5, 6}}); parseErr10 != nil {
		parseT.Fatalf("expected binary window publish helper to succeed, got %v", parseErr10)
	}
	parseBinaryWindow, parseOk := parseWindowPublished.(ClientMessage)
	if !parseOk {
		parseT.Fatalf("expected binary window payload, got %T", parseWindowPublished)
	}
	parseWindowBytes, parseOk := parseBinaryWindow.Payload.([]byte)
	if !parseOk || len(parseWindowBytes) != 3 || parseBinaryWindow.Encoding != ClientPayloadBinary || parseBinaryWindow.ContentType != "application/octet-stream" || parseBinaryWindow.Target != "inspector-1" {
		parseT.Fatalf("unexpected binary window payload: %+v", parseBinaryWindow)
	}

	if parseErr11 := PublishClientBinaryCrossTab(parseCrossTab, "asset:preview", parseSelf, ClientBinaryPayload{Bytes: []byte{1}}); !IsCode(parseErr11, CodeInvalid) {
		parseT.Fatalf("expected invalid content-type error for binary cross-tab publish, got %v", parseErr11)
	}
	if parseErr12 := PublishClientBinaryWindow(parseWindowChannel, "asset:preview", parseSelf, "", ClientBinaryPayload{ContentType: "application/octet-stream", Bytes: []byte{1}}); !IsCode(parseErr12, CodeInvalid) {
		parseT.Fatalf("expected invalid target error for binary window publish, got %v", parseErr12)
	}

	parseCustomCaps := ClientCapabilities{ProtocolVersion: "v2", Encodings: []string{"json", "binary"}, Topics: []string{"clients", "asset:preview"}}
	if parseErr13 := PublishClientHelloWithCapabilities(parseCrossTab, parseSelf, parseCustomCaps); parseErr13 != nil {
		parseT.Fatalf("expected hello-with-capabilities publish to succeed, got %v", parseErr13)
	}
	parseCustomHello, parseOk := parseCrossTabPublished.(ClientMessage)
	if !parseOk || parseCustomHello.Capabilities == nil || parseCustomHello.Capabilities.ProtocolVersion != "v2" || len(parseCustomHello.Capabilities.Topics) != 2 {
		parseT.Fatalf("unexpected custom hello capabilities payload: %+v", parseCustomHello)
	}
}

func TestSubscribeClientMessagesDecodesPayload(parseT *testing.T) {
	var parseCrossTabHandler func(CrossTabEnvelope, error)
	parseCrossTab := CrossTabChannel{
		name: func() string { return "atlas-clients" },
		subscribe: func(handler func(CrossTabEnvelope, error)) (Subscription, error) {
			parseCrossTabHandler = handler
			return Subscription{}, nil
		},
	}

	var parseReceived ClientMessage
	parseSubscription, parseErr := SubscribeClientMessages(parseCrossTab, func(parseMessage ClientMessage, parseErr7 error) {
		if parseErr7 != nil {
			parseT.Fatalf("expected decoded client message, got %v", parseErr7)
		}
		parseReceived = parseMessage
	})
	if parseErr != nil {
		parseT.Fatalf("expected cross-tab client subscription to succeed, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	parseCrossTabHandler(CrossTabEnvelope{
		Name: "atlas-clients",
		Payload: map[string]any{
			"kind":  "result",
			"topic": "clients",
			"source": map[string]any{
				"id":      "ops-1",
				"app":     "atlas",
				"surface": "popup",
			},
			"target": "storefront-1",
			"payload": map[string]any{
				"peers": 3,
			},
		},
	}, nil)

	if parseReceived.Kind != ClientResult || parseReceived.Topic != "clients" || parseReceived.Target != "storefront-1" {
		parseT.Fatalf("unexpected decoded cross-tab client message: %+v", parseReceived)
	}

	var parseWindowHandler func(WindowEnvelope, error)
	parseWindowChannel := WindowChannel{
		name: func() string { return "atlas-window" },
		subscribe: func(handler func(WindowEnvelope, error)) (Subscription, error) {
			parseWindowHandler = handler
			return Subscription{}, nil
		},
	}

	var parseWindowReceived ClientMessage
	parseWindowSubscription, parseErr := SubscribeClientWindowMessages(parseWindowChannel, func(parseMessage2 ClientMessage, parseErr8 error) {
		if parseErr8 != nil {
			parseT.Fatalf("expected decoded client window message, got %v", parseErr8)
		}
		parseWindowReceived = parseMessage2
	})
	if parseErr != nil {
		parseT.Fatalf("expected window client subscription to succeed, got %v", parseErr)
	}
	defer parseWindowSubscription.Cancel()

	parseWindowHandler(WindowEnvelope{
		Name: "atlas-window",
		Payload: map[string]any{
			"kind":  "hello",
			"topic": "clients",
			"source": map[string]any{
				"id":      "inspector-1",
				"app":     "atlas",
				"surface": "popup",
			},
		},
	}, nil)

	if parseWindowReceived.Kind != ClientHello || parseWindowReceived.Source.ID != "inspector-1" || parseWindowReceived.Topic != ClientPresenceTopic {
		parseT.Fatalf("unexpected decoded window client message: %+v", parseWindowReceived)
	}

	parseDecoded, parseErr := DecodeClientMessage(map[string]any{
		"kind":  "query",
		"topic": "clients",
		"source": map[string]any{
			"id":      "storefront-1",
			"app":     "atlas",
			"surface": "tab",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("expected DecodeClientMessage to succeed, got %v", parseErr)
	}
	if parseDecoded.Kind != ClientQuery || parseDecoded.Source.Surface != "tab" {
		parseT.Fatalf("unexpected decoded client message: %+v", parseDecoded)
	}

	if _, parseErr2 := DecodeClientMessage(map[string]any{
		"kind":  "mystery",
		"topic": "clients",
		"source": map[string]any{
			"id":      "storefront-1",
			"app":     "atlas",
			"surface": "tab",
		},
	}); !IsCode(parseErr2, CodeInvalid) {
		parseT.Fatalf("expected invalid kind error, got %v", parseErr2)
	}

	if _, parseErr3 := SubscribeClientMessages(parseCrossTab, nil); !IsCode(parseErr3, CodeInvalid) {
		parseT.Fatalf("expected nil handler error for cross-tab subscribe, got %v", parseErr3)
	}
	if _, parseErr4 := SubscribeClientWindowMessages(parseWindowChannel, nil); !IsCode(parseErr4, CodeInvalid) {
		parseT.Fatalf("expected nil handler error for window subscribe, got %v", parseErr4)
	}
	parsePublishChannel := CrossTabChannel{
		name: func() string { return "atlas-clients" },
		publish: func(parsePayload any) error {
			return nil
		},
	}
	if parseErr5 := PublishClientQuery(parsePublishChannel, "clients", ClientIdentity{ID: "storefront-1", App: "atlas", Surface: "tab"}); parseErr5 != nil {
		parseT.Fatalf("expected query publish helper to validate, got %v", parseErr5)
	}
	if parseErr6 := PublishClientResult(parsePublishChannel, "clients", ClientIdentity{ID: "ops-1", App: "atlas", Surface: "popup"}, "storefront-1", map[string]any{"peers": 2}); parseErr6 != nil {
		parseT.Fatalf("expected result publish helper to validate, got %v", parseErr6)
	}
}

func TestMultiClientQueryReliabilitySemantics(parseT *testing.T) {
	type pendingRequest struct {
		deadline time.Time
		terminal bool
		timedOut bool
	}

	type mockBus struct {
		nextSequence int64
		handlers     []func(CrossTabEnvelope, error)
	}

	parseBus := &mockBus{}
	parseNewChannel := func(parseName string, parseSource string) CrossTabChannel {
		return CrossTabChannel{
			name: func() string { return parseName },
			publish: func(parsePayload3 any) error {
				parseBus.nextSequence++
				parseEnvelope := CrossTabEnvelope{
					Name:     parseName,
					Payload:  parsePayload3,
					Source:   parseSource,
					Sequence: parseBus.nextSequence,
					SentAt:   time.Now().UTC(),
				}
				for _, parseHandler := range parseBus.handlers {
					if parseHandler != nil {
						parseHandler(parseEnvelope, nil)
					}
				}
				return nil
			},
			subscribe: func(handler func(CrossTabEnvelope, error)) (Subscription, error) {
				parseBus.handlers = append(parseBus.handlers, handler)
				parseIndex := len(parseBus.handlers) - 1
				return Subscription{cancel: func() {
					parseBus.handlers[parseIndex] = nil
				}}, nil
			},
		}
	}

	parseAlphaSelf := ClientIdentity{ID: "alpha-1", App: "atlas", Surface: "tab-a"}
	parseBetaSelf := ClientIdentity{ID: "beta-1", App: "atlas", Surface: "tab-b"}
	parseAlpha := parseNewChannel("clients", "alpha-transport")
	parseBeta := parseNewChannel("clients", "beta-transport")

	parseBetaPending := map[string]*pendingRequest{
		"req-fast":      {deadline: time.Date(2026, 3, 19, 11, 0, 1, 0, time.UTC)},
		"req-1":         {deadline: time.Date(2026, 3, 19, 11, 0, 1, 0, time.UTC)},
		"req-2":         {deadline: time.Date(2026, 3, 19, 11, 0, 1, 0, time.UTC)},
		"req-slow":      {deadline: time.Date(2026, 3, 19, 11, 0, 1, 0, time.UTC)},
		"beta-to-alpha": {deadline: time.Date(2026, 3, 19, 11, 0, 1, 0, time.UTC)},
	}
	parseAlphaPending := map[string]*pendingRequest{
		"alpha-to-beta": {deadline: time.Date(2026, 3, 19, 11, 0, 1, 0, time.UTC)},
	}
	parseBetaAccepted := map[string]string{}
	parseAlphaAccepted := map[string]string{}
	parseBetaDuplicateDrops := 0
	parseBetaLateDrops := 0
	parseBetaTimeouts := 0
	parseAlphaTimeouts := 0

	parseExpirePending := func(parseNow time.Time, parsePending map[string]*pendingRequest) int {
		parseExpired := 0
		for _, parseRequest := range parsePending {
			if parseRequest == nil || parseRequest.terminal || parseRequest.timedOut {
				continue
			}
			if parseNow.After(parseRequest.deadline) {
				parseRequest.timedOut = true
				parseExpired++
			}
		}
		return parseExpired
	}

	parseAlphaSubscription, parseErr := SubscribeClientMessages(parseAlpha, func(parseMessage ClientMessage, parseErr4 error) {
		if parseErr4 != nil {
			parseT.Fatalf("expected alpha reliability subscription to decode messages, got %v", parseErr4)
		}
		switch parseMessage.Kind {
		case ClientQuery:
			if parseMessage.Source.ID != parseBetaSelf.ID {
				return
			}
			switch parseMessage.ID {
			case "req-fast":
				parseResponse := ClientMessage{ID: parseMessage.ID, Kind: ClientResult, Topic: parseMessage.Topic, Source: parseAlphaSelf, Target: parseBetaSelf.ID, Payload: map[string]any{"slot": "fast"}}
				if parsePublishErr := PublishClientMessage(parseAlpha, parseResponse); parsePublishErr != nil {
					parseT.Fatalf("expected alpha fast result publish to succeed, got %v", parsePublishErr)
				}
				if parsePublishErr2 := PublishClientMessage(parseAlpha, parseResponse); parsePublishErr2 != nil {
					parseT.Fatalf("expected alpha duplicate fast result publish to succeed, got %v", parsePublishErr2)
				}
			case "req-1":
				if parsePublishErr3 := PublishClientMessage(parseAlpha, ClientMessage{ID: parseMessage.ID, Kind: ClientResult, Topic: parseMessage.Topic, Source: parseAlphaSelf, Target: parseBetaSelf.ID, Payload: map[string]any{"slot": "one"}}); parsePublishErr3 != nil {
					parseT.Fatalf("expected alpha req-1 result publish to succeed, got %v", parsePublishErr3)
				}
			case "req-2":
				if parsePublishErr4 := PublishClientMessage(parseAlpha, ClientMessage{ID: parseMessage.ID, Kind: ClientResult, Topic: parseMessage.Topic, Source: parseAlphaSelf, Target: parseBetaSelf.ID, Payload: map[string]any{"slot": "two"}}); parsePublishErr4 != nil {
					parseT.Fatalf("expected alpha req-2 result publish to succeed, got %v", parsePublishErr4)
				}
			case "beta-to-alpha":
				if parsePublishErr5 := PublishClientMessage(parseAlpha, ClientMessage{ID: parseMessage.ID, Kind: ClientResult, Topic: parseMessage.Topic, Source: parseAlphaSelf, Target: parseBetaSelf.ID, Payload: map[string]any{"slot": "bidirectional-beta"}}); parsePublishErr5 != nil {
					parseT.Fatalf("expected alpha bidirectional result publish to succeed, got %v", parsePublishErr5)
				}
			}
		case ClientResult:
			if parseMessage.Target != parseAlphaSelf.ID {
				return
			}
			parseRequest2 := parseAlphaPending[parseMessage.ID]
			if parseRequest2 == nil || parseRequest2.timedOut || parseRequest2.terminal {
				return
			}
			parsePayload, _ := parseMessage.Payload.(map[string]any)
			parseAlphaAccepted[parseMessage.ID], _ = parsePayload["slot"].(string)
			parseRequest2.terminal = true
		}
	})
	if parseErr != nil {
		parseT.Fatalf("expected alpha reliability subscription to succeed, got %v", parseErr)
	}
	defer parseAlphaSubscription.Cancel()

	parseBetaSubscription, parseErr := SubscribeClientMessages(parseBeta, func(parseMessage2 ClientMessage, parseErr5 error) {
		if parseErr5 != nil {
			parseT.Fatalf("expected beta reliability subscription to decode messages, got %v", parseErr5)
		}
		switch parseMessage2.Kind {
		case ClientQuery:
			if parseMessage2.Source.ID != parseAlphaSelf.ID || parseMessage2.ID != "alpha-to-beta" {
				return
			}
			if parsePublishErr6 := PublishClientMessage(parseBeta, ClientMessage{ID: parseMessage2.ID, Kind: ClientResult, Topic: parseMessage2.Topic, Source: parseBetaSelf, Target: parseAlphaSelf.ID, Payload: map[string]any{"slot": "bidirectional-alpha"}}); parsePublishErr6 != nil {
				parseT.Fatalf("expected beta bidirectional result publish to succeed, got %v", parsePublishErr6)
			}
		case ClientResult:
			if parseMessage2.Target != parseBetaSelf.ID {
				return
			}
			parseRequest3 := parseBetaPending[parseMessage2.ID]
			if parseRequest3 == nil {
				return
			}
			if parseRequest3.timedOut {
				parseBetaLateDrops++
				return
			}
			if parseRequest3.terminal {
				parseBetaDuplicateDrops++
				return
			}
			parsePayload2, _ := parseMessage2.Payload.(map[string]any)
			parseBetaAccepted[parseMessage2.ID], _ = parsePayload2["slot"].(string)
			parseRequest3.terminal = true
		}
	})
	if parseErr != nil {
		parseT.Fatalf("expected beta reliability subscription to succeed, got %v", parseErr)
	}
	defer parseBetaSubscription.Cancel()

	parseQueryTopic := "inventory:query"
	parseQueries := []ClientMessage{
		{ID: "req-fast", Kind: ClientQuery, Topic: parseQueryTopic, Source: parseBetaSelf},
		{ID: "req-1", Kind: ClientQuery, Topic: parseQueryTopic, Source: parseBetaSelf},
		{ID: "req-2", Kind: ClientQuery, Topic: parseQueryTopic, Source: parseBetaSelf},
		{ID: "req-slow", Kind: ClientQuery, Topic: parseQueryTopic, Source: parseBetaSelf},
		{ID: "alpha-to-beta", Kind: ClientQuery, Topic: parseQueryTopic, Source: parseAlphaSelf},
		{ID: "beta-to-alpha", Kind: ClientQuery, Topic: parseQueryTopic, Source: parseBetaSelf},
	}
	for _, parseQuery := range parseQueries {
		parseChannel := parseBeta
		if parseQuery.Source.ID == parseAlphaSelf.ID {
			parseChannel = parseAlpha
		}
		if parseErr2 := PublishClientMessage(parseChannel, parseQuery); parseErr2 != nil {
			parseT.Fatalf("expected query %q to publish, got %v", parseQuery.ID, parseErr2)
		}
	}

	parseBetaTimeouts += parseExpirePending(time.Date(2026, 3, 19, 11, 0, 2, 0, time.UTC), parseBetaPending)
	parseAlphaTimeouts += parseExpirePending(time.Date(2026, 3, 19, 11, 0, 2, 0, time.UTC), parseAlphaPending)

	if parseErr3 := PublishClientMessage(parseAlpha, ClientMessage{ID: "req-slow", Kind: ClientResult, Topic: parseQueryTopic, Source: parseAlphaSelf, Target: parseBetaSelf.ID, Payload: map[string]any{"slot": "late"}}); parseErr3 != nil {
		parseT.Fatalf("expected late result publish to succeed, got %v", parseErr3)
	}

	if parseBetaAccepted["req-fast"] != "fast" {
		parseT.Fatalf("expected request correlation to retain the fast result, got %+v", parseBetaAccepted)
	}
	if parseBetaAccepted["req-1"] != "one" || parseBetaAccepted["req-2"] != "two" {
		parseT.Fatalf("expected concurrent query results to correlate by request ID, got %+v", parseBetaAccepted)
	}
	if parseBetaAccepted["beta-to-alpha"] != "bidirectional-beta" {
		parseT.Fatalf("expected simultaneous beta->alpha request to receive a reply, got %+v", parseBetaAccepted)
	}
	if parseAlphaAccepted["alpha-to-beta"] != "bidirectional-alpha" {
		parseT.Fatalf("expected simultaneous alpha->beta request to receive a reply, got %+v", parseAlphaAccepted)
	}
	if parseBetaDuplicateDrops != 1 {
		parseT.Fatalf("expected duplicate result tolerance to ignore the second fast result, got %d", parseBetaDuplicateDrops)
	}
	if parseBetaTimeouts != 1 || parseAlphaTimeouts != 0 {
		parseT.Fatalf("expected only the slow beta request to time out, beta=%d alpha=%d", parseBetaTimeouts, parseAlphaTimeouts)
	}
	if parseBetaLateDrops != 1 {
		parseT.Fatalf("expected late result discard after timeout, got %d", parseBetaLateDrops)
	}
	if _, parseOk := parseBetaAccepted["req-slow"]; parseOk {
		parseT.Fatalf("expected timed-out request to reject late result, got %+v", parseBetaAccepted)
	}
	if !parseBetaPending["req-fast"].terminal || !parseBetaPending["req-1"].terminal || !parseBetaPending["req-2"].terminal || !parseBetaPending["beta-to-alpha"].terminal || !parseAlphaPending["alpha-to-beta"].terminal {
		parseT.Fatalf("expected accepted requests to be terminal, beta=%+v alpha=%+v", parseBetaPending, parseAlphaPending)
	}
}

func TestMultiClientSecurityAuthorizationGuards(parseT *testing.T) {
	parseStorefront := ClientIdentity{ID: "storefront-1", App: "atlas", Surface: "tab", Role: "storefront"}
	parseOperator := ClientIdentity{ID: "ops-1", App: "atlas", Surface: "popup", Role: "operator"}

	parseCrossTab := CrossTabChannel{
		name: func() string { return "atlas-clients" },
		publish: func(parsePayload any) error {
			return nil
		},
	}
	parseWindowChannel := WindowChannel{
		name: func() string { return "atlas-window" },
		publish: func(parsePayload2 any) error {
			return nil
		},
	}

	if parseErr := PublishClientMessage(parseCrossTab, ClientMessage{Kind: ClientEvent, Topic: "operator:inventory", Source: parseStorefront, Payload: map[string]any{"sku": "SKU-44"}}); !IsCode(parseErr, CodeUnauthorized) {
		parseT.Fatalf("expected unauthorized topic publish to fail, got %v", parseErr)
	}
	if parseErr2 := PublishClientIntent(parseWindowChannel, "session:logout", parseStorefront, "popup-1", map[string]any{"reason": "peer-requested"}); !IsCode(parseErr2, CodeUnauthorized) {
		parseT.Fatalf("expected privileged intent rejection for non-operator role, got %v", parseErr2)
	}
	if parseErr3 := PublishClientIntent(parseWindowChannel, "operator:inventory", parseOperator, "popup-1", map[string]any{"sku": "SKU-44"}); parseErr3 != nil {
		parseT.Fatalf("expected operator role to publish privileged intent, got %v", parseErr3)
	}

	if _, parseErr4 := DecodeClientMessage(map[string]any{
		"kind":  "event",
		"topic": "operator:inventory",
		"source": map[string]any{
			"id":      "storefront-1",
			"app":     "atlas",
			"surface": "tab",
			"role":    "storefront",
		},
	}); !IsCode(parseErr4, CodeUnauthorized) {
		parseT.Fatalf("expected unauthorized privileged topic decode to fail, got %v", parseErr4)
	}

	if _, parseErr5 := DecodeClientMessage(map[string]any{
		"kind":  "hello",
		"topic": "clients",
		"source": map[string]any{
			"id":      "storefront-1",
			"surface": "tab",
		},
	}); !IsCode(parseErr5, CodeInvalid) {
		parseT.Fatalf("expected malformed peer identity to fail validation, got %v", parseErr5)
	}
}

func TestClientCapabilityNegotiation(parseT *testing.T) {
	parseLocal := ClientCapabilities{ProtocolVersion: "v1.2", Encodings: []string{"json", "binary"}, Topics: []string{"clients", "asset:preview"}}
	parsePeer := ClientCapabilities{ProtocolVersion: "1.9", Encodings: []string{"json", "binary"}, Topics: []string{"asset:preview"}}
	if !ClientProtocolCompatible(parseLocal, parsePeer) {
		parseT.Fatal("expected matching major protocol versions to be compatible")
	}
	if !ClientSupportsEncoding(parsePeer, ClientPayloadBinary) {
		parseT.Fatal("expected peer to advertise binary encoding support")
	}
	if !ClientSupportsTopic(parsePeer, "asset:preview") {
		parseT.Fatal("expected peer topic support to include asset:preview")
	}
	if !ClientCanExchange(parseLocal, parsePeer, "asset:preview", ClientPayloadBinary) {
		parseT.Fatal("expected local and peer capabilities to allow binary exchange on asset:preview")
	}

	parseMissingBinary := ClientCapabilities{ProtocolVersion: "v1.4", Encodings: []string{"json"}, Topics: []string{"asset:preview"}}
	if ClientCanExchange(parseLocal, parseMissingBinary, "asset:preview", ClientPayloadBinary) {
		parseT.Fatal("expected binary exchange to fail when one peer is json-only")
	}

	parseWrongTopic := ClientCapabilities{ProtocolVersion: "v1.4", Encodings: []string{"json", "binary"}, Topics: []string{"clients"}}
	if ClientCanExchange(parseLocal, parseWrongTopic, "asset:preview", ClientPayloadJSON) {
		parseT.Fatal("expected exchange to fail when the peer does not advertise the topic")
	}

	parseIncompatible := ClientCapabilities{ProtocolVersion: "v2.0", Encodings: []string{"json", "binary"}, Topics: []string{"asset:preview"}}
	if ClientProtocolCompatible(parseLocal, parseIncompatible) {
		parseT.Fatal("expected different major protocol versions to be incompatible")
	}
	if ClientCanExchange(parseLocal, parseIncompatible, "asset:preview", ClientPayloadJSON) {
		parseT.Fatal("expected exchange to fail on incompatible protocol versions")
	}

	parseStagedRollout := ClientCapabilities{ProtocolVersion: "v1.5", Encodings: []string{"json", "binary"}}
	if !ClientCanExchange(parseLocal, parseStagedRollout, "asset:preview", ClientPayloadJSON) {
		parseT.Fatal("expected topic-unspecified peers to allow additive staged rollout negotiation")
	}

	parseDecoded, parseErr := DecodeClientMessage(map[string]any{
		"kind":  "hello",
		"topic": "clients",
		"source": map[string]any{
			"id":      "storefront-1",
			"app":     "atlas",
			"surface": "tab",
		},
		"capabilities": map[string]any{
			"protocolVersion": "v1",
			"encodings":       []any{"json", "binary"},
			"topics":          []any{"clients", "asset:preview"},
		},
	})
	if parseErr != nil {
		parseT.Fatalf("expected hello message with capabilities to decode, got %v", parseErr)
	} else if parseDecoded.Capabilities == nil || !ClientSupportsEncoding(*parseDecoded.Capabilities, ClientPayloadBinary) || !ClientSupportsTopic(*parseDecoded.Capabilities, "asset:preview") {
		parseT.Fatalf("unexpected decoded capabilities payload: %+v", parseDecoded.Capabilities)
	}
}

func TestPublishSurfaceSignalHelpers(parseT *testing.T) {
	var parsePublished any
	parseChannel := WindowChannel{
		name: func() string { return "ops" },
		publish: func(parsePayload any) error {
			parsePublished = parsePayload
			return nil
		},
	}

	parseExpiresAt := time.Date(2026, 3, 18, 19, 0, 0, 0, time.UTC)
	if parseErr := PublishSessionExpired(parseChannel, "Session expired in another surface.", "/login", parseExpiresAt); parseErr != nil {
		parseT.Fatalf("expected session-expired helper to publish, got %v", parseErr)
	}
	parseSignal, parseOk := parsePublished.(SurfaceSignal)
	if !parseOk {
		parseT.Fatalf("expected surface signal payload, got %T", parsePublished)
	}
	if parseSignal.Kind != SurfaceSignalSession || parseSignal.Session == nil || parseSignal.Session.Status != "expired" || parseSignal.Session.ReturnTo != "/login" || !parseSignal.Session.ExpiresAt.Equal(parseExpiresAt) {
		parseT.Fatalf("unexpected session signal payload: %+v", parseSignal)
	}

	if parseErr2 := PublishRouteFocus(parseChannel, "/orders/42", "tab=activity", "order-heading"); parseErr2 != nil {
		parseT.Fatalf("expected route-focus helper to publish, got %v", parseErr2)
	}
	parseSignal, parseOk = parsePublished.(SurfaceSignal)
	if !parseOk || parseSignal.Route == nil || parseSignal.Route.Path != "/orders/42" || parseSignal.Route.Query != "tab=activity" || parseSignal.Route.FocusID != "order-heading" {
		parseT.Fatalf("unexpected route signal payload: %+v", parsePublished)
	}

	if parseErr3 := PublishSelection(parseChannel, "invoice", "INV-204", "rev-12"); parseErr3 != nil {
		parseT.Fatalf("expected selection helper to publish, got %v", parseErr3)
	}
	parseSignal, parseOk = parsePublished.(SurfaceSignal)
	if !parseOk || parseSignal.Selection == nil || parseSignal.Selection.Scope != "invoice" || parseSignal.Selection.ID != "INV-204" || parseSignal.Selection.Revision != "rev-12" {
		parseT.Fatalf("unexpected selection signal payload: %+v", parsePublished)
	}

	if parseErr4 := PublishIntent(parseChannel, SurfaceIntentFocusPanel, "audit-log", map[string]string{"tab": "alerts"}); parseErr4 != nil {
		parseT.Fatalf("expected intent helper to publish, got %v", parseErr4)
	}
	parseSignal, parseOk = parsePublished.(SurfaceSignal)
	if !parseOk || parseSignal.Intent == nil || parseSignal.Intent.Action != SurfaceIntentFocusPanel || parseSignal.Intent.Target != "audit-log" || parseSignal.Intent.Params["tab"] != "alerts" {
		parseT.Fatalf("unexpected intent signal payload: %+v", parsePublished)
	}
}

func TestDecodeSurfaceSignalProjectsTypedPayload(parseT *testing.T) {
	parseMessage := WindowEnvelope{
		Name:   "ops",
		Source: "popup-1",
		Payload: map[string]any{
			"kind": "selection",
			"selection": map[string]any{
				"scope":    "inventory-item",
				"id":       "SKU-42",
				"revision": "rev-7",
			},
		},
	}
	parseDecoded, parseErr := DecodeSurfaceSignal(parseMessage)
	if parseErr != nil {
		parseT.Fatalf("expected typed surface signal, got %v", parseErr)
	}
	if parseDecoded.Name != "ops" || parseDecoded.Source != "popup-1" {
		parseT.Fatalf("unexpected envelope metadata: %+v", parseDecoded)
	}
	if parseDecoded.Payload.Kind != SurfaceSignalSelection || parseDecoded.Payload.Selection == nil || parseDecoded.Payload.Selection.ID != "SKU-42" {
		parseT.Fatalf("unexpected decoded surface payload: %+v", parseDecoded.Payload)
	}
}
