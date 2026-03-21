//go:build !js || !wasm
// +build !js !wasm

package interop

import (
	"context"
	"testing"
	"time"
)

func TestNativeInteropConstructorsReportUnavailable(t *testing.T) {
	checks := []struct {
		name string
		err  error
	}{
		{name: "LocalStorage", err: func() error { _, err := LocalStorage(); return err }()},
		{name: "SessionStorage", err: func() error { _, err := SessionStorage(); return err }()},
		{name: "WindowLocation", err: func() error { _, err := WindowLocation(); return err }()},
		{name: "WindowHistory", err: func() error { _, err := WindowHistory(); return err }()},
		{name: "NavigatorClipboard", err: func() error { _, err := NavigatorClipboard(); return err }()},
		{name: "WindowEvents", err: func() error { _, err := WindowEvents(); return err }()},
		{name: "DocumentEvents", err: func() error { _, err := DocumentEvents(); return err }()},
		{name: "CurrentDocument", err: func() error { _, err := CurrentDocument(); return err }()},
		{name: "MatchMedia", err: func() error { _, err := MatchMedia("(prefers-color-scheme: dark)"); return err }()},
		{name: "ImportModule", err: func() error { _, err := ImportModule(context.TODO(), "/demo.js"); return err }()},
		{name: "NewWorker", err: func() error { _, err := NewWorker(context.TODO(), WorkerOptions{URL: "/worker.js"}); return err }()},
		{name: "OpenCrossTabChannel", err: func() error { _, err := OpenCrossTabChannel(CrossTabChannelOptions{Name: "theme"}); return err }()},
		{name: "OpenSecondaryWindowChannel", err: func() error { _, err := OpenSecondaryWindowChannel(WindowChannelOptions{Name: "popup"}); return err }()},
		{name: "WindowOpenerChannel", err: func() error { _, err := WindowOpenerChannel(WindowChannelOptions{Name: "popup"}); return err }()},
	}
	for _, check := range checks {
		if !IsCode(check.err, CodeUnavailable) {
			t.Fatalf("%s: expected unavailable error, got %v", check.name, check.err)
		}
		interopErr, ok := AsError(check.err)
		if !ok || interopErr.Code != CodeUnavailable {
			t.Fatalf("%s: expected structured interop error, got %#v ok=%t", check.name, interopErr, ok)
		}
		code, ok := CodeOf(check.err)
		if !ok || code != CodeUnavailable {
			t.Fatalf("%s: expected code lookup to report unavailable, got %q ok=%t", check.name, code, ok)
		}
	}
}

func TestDecodeCustomEventProjectsTypedDetail(t *testing.T) {
	event := CustomEvent{
		Type: "asset-ready",
		Detail: map[string]any{
			"id":    "asset-42",
			"score": 9,
		},
	}
	decoded, err := DecodeCustomEvent[struct {
		ID    string `json:"id"`
		Score int    `json:"score"`
	}](event)
	if err != nil {
		t.Fatalf("expected typed custom event detail, got %v", err)
	}
	if decoded.Type != "asset-ready" || decoded.Detail.ID != "asset-42" || decoded.Detail.Score != 9 {
		t.Fatalf("unexpected decoded custom event: %+v", decoded)
	}
}

func TestPublishClientHelpers(t *testing.T) {
	self := ClientIdentity{ID: "storefront-1", App: "atlas", Surface: "tab", Role: "storefront", Version: "v1"}

	var crossTabPublished any
	crossTab := CrossTabChannel{
		name: func() string { return "atlas-clients" },
		publish: func(payload any) error {
			crossTabPublished = payload
			return nil
		},
	}
	if err := PublishClientHello(crossTab, self); err != nil {
		t.Fatalf("expected hello publish to succeed, got %v", err)
	}
	hello, ok := crossTabPublished.(ClientMessage)
	if !ok {
		t.Fatalf("expected client hello payload, got %T", crossTabPublished)
	}
	if hello.Kind != ClientHello || hello.Topic != ClientPresenceTopic || hello.Source.ID != "storefront-1" || hello.SentAt.IsZero() {
		t.Fatalf("unexpected hello payload: %+v", hello)
	}
	if hello.Capabilities == nil || hello.Capabilities.ProtocolVersion != "v1" || len(hello.Capabilities.Encodings) == 0 || hello.Capabilities.Encodings[0] != "json" {
		t.Fatalf("expected default cross-tab hello capabilities, got %+v", hello.Capabilities)
	}

	if err := PublishClientInvalidation(crossTab, "cache:inventory", self, "rev-9"); err != nil {
		t.Fatalf("expected invalidation publish to succeed, got %v", err)
	}
	invalidation, ok := crossTabPublished.(ClientMessage)
	if !ok {
		t.Fatalf("expected invalidation payload, got %T", crossTabPublished)
	}
	if invalidation.Kind != ClientInvalidate || invalidation.Topic != "cache:inventory" || invalidation.Revision != "rev-9" {
		t.Fatalf("unexpected invalidation payload: %+v", invalidation)
	}

	var windowPublished any
	windowChannel := WindowChannel{
		name: func() string { return "atlas-window" },
		publish: func(payload any) error {
			windowPublished = payload
			return nil
		},
	}
	if err := PublishClientIntent(windowChannel, "selection:inspect", self, "inspector-1", map[string]any{"id": "SKU-44"}); err != nil {
		t.Fatalf("expected intent publish to succeed, got %v", err)
	}
	intent, ok := windowPublished.(ClientMessage)
	if !ok {
		t.Fatalf("expected intent payload, got %T", windowPublished)
	}
	if intent.Kind != ClientIntent || intent.Topic != "selection:inspect" || intent.Target != "inspector-1" {
		t.Fatalf("unexpected intent payload: %+v", intent)
	}

	if err := PublishClientIntent(windowChannel, "selection:inspect", self, "", nil); !IsCode(err, CodeInvalid) {
		t.Fatalf("expected invalid target error, got %v", err)
	}
	if err := PublishClientInvalidation(crossTab, "cache:inventory", self, ""); !IsCode(err, CodeInvalid) {
		t.Fatalf("expected invalid revision error, got %v", err)
	}
	if err := PublishClientHello(crossTab, ClientIdentity{}); !IsCode(err, CodeInvalid) {
		t.Fatalf("expected invalid identity error, got %v", err)
	}
	if err := PublishClientMessage(crossTab, ClientMessage{Kind: ClientEvent, Topic: "ops", Source: self}); err != nil {
		t.Fatalf("expected generic publish helper to succeed, got %v", err)
	}
	if err := PublishClientWindowMessage(windowChannel, ClientMessage{Kind: ClientResult, Topic: "clients", Source: self, Target: "inspector-1"}); err != nil {
		t.Fatalf("expected generic window publish helper to succeed, got %v", err)
	}

	if err := PublishClientBinaryCrossTab(crossTab, "asset:preview", self, ClientBinaryPayload{ContentType: "application/octet-stream", Bytes: []byte{1, 2, 3}}); err != nil {
		t.Fatalf("expected binary cross-tab publish helper to succeed, got %v", err)
	}
	binaryCrossTab, ok := crossTabPublished.(ClientMessage)
	if !ok {
		t.Fatalf("expected binary cross-tab payload, got %T", crossTabPublished)
	}
	binaryBytes, ok := binaryCrossTab.Payload.([]byte)
	if !ok || len(binaryBytes) != 3 || binaryCrossTab.Encoding != ClientPayloadBinary || binaryCrossTab.ContentType != "application/octet-stream" {
		t.Fatalf("unexpected binary cross-tab payload: %+v", binaryCrossTab)
	}

	if err := PublishClientBinaryWindow(windowChannel, "asset:preview", self, "inspector-1", ClientBinaryPayload{ContentType: "application/octet-stream", Bytes: []byte{4, 5, 6}}); err != nil {
		t.Fatalf("expected binary window publish helper to succeed, got %v", err)
	}
	binaryWindow, ok := windowPublished.(ClientMessage)
	if !ok {
		t.Fatalf("expected binary window payload, got %T", windowPublished)
	}
	windowBytes, ok := binaryWindow.Payload.([]byte)
	if !ok || len(windowBytes) != 3 || binaryWindow.Encoding != ClientPayloadBinary || binaryWindow.ContentType != "application/octet-stream" || binaryWindow.Target != "inspector-1" {
		t.Fatalf("unexpected binary window payload: %+v", binaryWindow)
	}

	if err := PublishClientBinaryCrossTab(crossTab, "asset:preview", self, ClientBinaryPayload{Bytes: []byte{1}}); !IsCode(err, CodeInvalid) {
		t.Fatalf("expected invalid content-type error for binary cross-tab publish, got %v", err)
	}
	if err := PublishClientBinaryWindow(windowChannel, "asset:preview", self, "", ClientBinaryPayload{ContentType: "application/octet-stream", Bytes: []byte{1}}); !IsCode(err, CodeInvalid) {
		t.Fatalf("expected invalid target error for binary window publish, got %v", err)
	}

	customCaps := ClientCapabilities{ProtocolVersion: "v2", Encodings: []string{"json", "binary"}, Topics: []string{"clients", "asset:preview"}}
	if err := PublishClientHelloWithCapabilities(crossTab, self, customCaps); err != nil {
		t.Fatalf("expected hello-with-capabilities publish to succeed, got %v", err)
	}
	customHello, ok := crossTabPublished.(ClientMessage)
	if !ok || customHello.Capabilities == nil || customHello.Capabilities.ProtocolVersion != "v2" || len(customHello.Capabilities.Topics) != 2 {
		t.Fatalf("unexpected custom hello capabilities payload: %+v", customHello)
	}
}

func TestSubscribeClientMessagesDecodesPayload(t *testing.T) {
	var crossTabHandler func(CrossTabEnvelope, error)
	crossTab := CrossTabChannel{
		name: func() string { return "atlas-clients" },
		subscribe: func(handler func(CrossTabEnvelope, error)) (Subscription, error) {
			crossTabHandler = handler
			return Subscription{}, nil
		},
	}

	var received ClientMessage
	subscription, err := SubscribeClientMessages(crossTab, func(message ClientMessage, err error) {
		if err != nil {
			t.Fatalf("expected decoded client message, got %v", err)
		}
		received = message
	})
	if err != nil {
		t.Fatalf("expected cross-tab client subscription to succeed, got %v", err)
	}
	defer subscription.Cancel()

	crossTabHandler(CrossTabEnvelope{
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

	if received.Kind != ClientResult || received.Topic != "clients" || received.Target != "storefront-1" {
		t.Fatalf("unexpected decoded cross-tab client message: %+v", received)
	}

	var windowHandler func(WindowEnvelope, error)
	windowChannel := WindowChannel{
		name: func() string { return "atlas-window" },
		subscribe: func(handler func(WindowEnvelope, error)) (Subscription, error) {
			windowHandler = handler
			return Subscription{}, nil
		},
	}

	var windowReceived ClientMessage
	windowSubscription, err := SubscribeClientWindowMessages(windowChannel, func(message ClientMessage, err error) {
		if err != nil {
			t.Fatalf("expected decoded client window message, got %v", err)
		}
		windowReceived = message
	})
	if err != nil {
		t.Fatalf("expected window client subscription to succeed, got %v", err)
	}
	defer windowSubscription.Cancel()

	windowHandler(WindowEnvelope{
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

	if windowReceived.Kind != ClientHello || windowReceived.Source.ID != "inspector-1" || windowReceived.Topic != ClientPresenceTopic {
		t.Fatalf("unexpected decoded window client message: %+v", windowReceived)
	}

	decoded, err := DecodeClientMessage(map[string]any{
		"kind":  "query",
		"topic": "clients",
		"source": map[string]any{
			"id":      "storefront-1",
			"app":     "atlas",
			"surface": "tab",
		},
	})
	if err != nil {
		t.Fatalf("expected DecodeClientMessage to succeed, got %v", err)
	}
	if decoded.Kind != ClientQuery || decoded.Source.Surface != "tab" {
		t.Fatalf("unexpected decoded client message: %+v", decoded)
	}

	if _, err := DecodeClientMessage(map[string]any{
		"kind":  "mystery",
		"topic": "clients",
		"source": map[string]any{
			"id":      "storefront-1",
			"app":     "atlas",
			"surface": "tab",
		},
	}); !IsCode(err, CodeInvalid) {
		t.Fatalf("expected invalid kind error, got %v", err)
	}

	if _, err := SubscribeClientMessages(crossTab, nil); !IsCode(err, CodeInvalid) {
		t.Fatalf("expected nil handler error for cross-tab subscribe, got %v", err)
	}
	if _, err := SubscribeClientWindowMessages(windowChannel, nil); !IsCode(err, CodeInvalid) {
		t.Fatalf("expected nil handler error for window subscribe, got %v", err)
	}
	publishChannel := CrossTabChannel{
		name: func() string { return "atlas-clients" },
		publish: func(payload any) error {
			return nil
		},
	}
	if err := PublishClientQuery(publishChannel, "clients", ClientIdentity{ID: "storefront-1", App: "atlas", Surface: "tab"}); err != nil {
		t.Fatalf("expected query publish helper to validate, got %v", err)
	}
	if err := PublishClientResult(publishChannel, "clients", ClientIdentity{ID: "ops-1", App: "atlas", Surface: "popup"}, "storefront-1", map[string]any{"peers": 2}); err != nil {
		t.Fatalf("expected result publish helper to validate, got %v", err)
	}
}

func TestMultiClientQueryReliabilitySemantics(t *testing.T) {
	type pendingRequest struct {
		deadline time.Time
		terminal bool
		timedOut bool
	}

	type mockBus struct {
		nextSequence int64
		handlers     []func(CrossTabEnvelope, error)
	}

	bus := &mockBus{}
	newChannel := func(name string, source string) CrossTabChannel {
		return CrossTabChannel{
			name: func() string { return name },
			publish: func(payload any) error {
				bus.nextSequence++
				envelope := CrossTabEnvelope{
					Name:     name,
					Payload:  payload,
					Source:   source,
					Sequence: bus.nextSequence,
					SentAt:   time.Now().UTC(),
				}
				for _, handler := range bus.handlers {
					if handler != nil {
						handler(envelope, nil)
					}
				}
				return nil
			},
			subscribe: func(handler func(CrossTabEnvelope, error)) (Subscription, error) {
				bus.handlers = append(bus.handlers, handler)
				index := len(bus.handlers) - 1
				return Subscription{cancel: func() {
					bus.handlers[index] = nil
				}}, nil
			},
		}
	}

	alphaSelf := ClientIdentity{ID: "alpha-1", App: "atlas", Surface: "tab-a"}
	betaSelf := ClientIdentity{ID: "beta-1", App: "atlas", Surface: "tab-b"}
	alpha := newChannel("clients", "alpha-transport")
	beta := newChannel("clients", "beta-transport")

	betaPending := map[string]*pendingRequest{
		"req-fast":      {deadline: time.Date(2026, 3, 19, 11, 0, 1, 0, time.UTC)},
		"req-1":         {deadline: time.Date(2026, 3, 19, 11, 0, 1, 0, time.UTC)},
		"req-2":         {deadline: time.Date(2026, 3, 19, 11, 0, 1, 0, time.UTC)},
		"req-slow":      {deadline: time.Date(2026, 3, 19, 11, 0, 1, 0, time.UTC)},
		"beta-to-alpha": {deadline: time.Date(2026, 3, 19, 11, 0, 1, 0, time.UTC)},
	}
	alphaPending := map[string]*pendingRequest{
		"alpha-to-beta": {deadline: time.Date(2026, 3, 19, 11, 0, 1, 0, time.UTC)},
	}
	betaAccepted := map[string]string{}
	alphaAccepted := map[string]string{}
	betaDuplicateDrops := 0
	betaLateDrops := 0
	betaTimeouts := 0
	alphaTimeouts := 0

	expirePending := func(now time.Time, pending map[string]*pendingRequest) int {
		expired := 0
		for _, request := range pending {
			if request == nil || request.terminal || request.timedOut {
				continue
			}
			if now.After(request.deadline) {
				request.timedOut = true
				expired++
			}
		}
		return expired
	}

	alphaSubscription, err := SubscribeClientMessages(alpha, func(message ClientMessage, err error) {
		if err != nil {
			t.Fatalf("expected alpha reliability subscription to decode messages, got %v", err)
		}
		switch message.Kind {
		case ClientQuery:
			if message.Source.ID != betaSelf.ID {
				return
			}
			switch message.ID {
			case "req-fast":
				response := ClientMessage{ID: message.ID, Kind: ClientResult, Topic: message.Topic, Source: alphaSelf, Target: betaSelf.ID, Payload: map[string]any{"slot": "fast"}}
				if publishErr := PublishClientMessage(alpha, response); publishErr != nil {
					t.Fatalf("expected alpha fast result publish to succeed, got %v", publishErr)
				}
				if publishErr := PublishClientMessage(alpha, response); publishErr != nil {
					t.Fatalf("expected alpha duplicate fast result publish to succeed, got %v", publishErr)
				}
			case "req-1":
				if publishErr := PublishClientMessage(alpha, ClientMessage{ID: message.ID, Kind: ClientResult, Topic: message.Topic, Source: alphaSelf, Target: betaSelf.ID, Payload: map[string]any{"slot": "one"}}); publishErr != nil {
					t.Fatalf("expected alpha req-1 result publish to succeed, got %v", publishErr)
				}
			case "req-2":
				if publishErr := PublishClientMessage(alpha, ClientMessage{ID: message.ID, Kind: ClientResult, Topic: message.Topic, Source: alphaSelf, Target: betaSelf.ID, Payload: map[string]any{"slot": "two"}}); publishErr != nil {
					t.Fatalf("expected alpha req-2 result publish to succeed, got %v", publishErr)
				}
			case "beta-to-alpha":
				if publishErr := PublishClientMessage(alpha, ClientMessage{ID: message.ID, Kind: ClientResult, Topic: message.Topic, Source: alphaSelf, Target: betaSelf.ID, Payload: map[string]any{"slot": "bidirectional-beta"}}); publishErr != nil {
					t.Fatalf("expected alpha bidirectional result publish to succeed, got %v", publishErr)
				}
			}
		case ClientResult:
			if message.Target != alphaSelf.ID {
				return
			}
			request := alphaPending[message.ID]
			if request == nil || request.timedOut || request.terminal {
				return
			}
			payload, _ := message.Payload.(map[string]any)
			alphaAccepted[message.ID], _ = payload["slot"].(string)
			request.terminal = true
		}
	})
	if err != nil {
		t.Fatalf("expected alpha reliability subscription to succeed, got %v", err)
	}
	defer alphaSubscription.Cancel()

	betaSubscription, err := SubscribeClientMessages(beta, func(message ClientMessage, err error) {
		if err != nil {
			t.Fatalf("expected beta reliability subscription to decode messages, got %v", err)
		}
		switch message.Kind {
		case ClientQuery:
			if message.Source.ID != alphaSelf.ID || message.ID != "alpha-to-beta" {
				return
			}
			if publishErr := PublishClientMessage(beta, ClientMessage{ID: message.ID, Kind: ClientResult, Topic: message.Topic, Source: betaSelf, Target: alphaSelf.ID, Payload: map[string]any{"slot": "bidirectional-alpha"}}); publishErr != nil {
				t.Fatalf("expected beta bidirectional result publish to succeed, got %v", publishErr)
			}
		case ClientResult:
			if message.Target != betaSelf.ID {
				return
			}
			request := betaPending[message.ID]
			if request == nil {
				return
			}
			if request.timedOut {
				betaLateDrops++
				return
			}
			if request.terminal {
				betaDuplicateDrops++
				return
			}
			payload, _ := message.Payload.(map[string]any)
			betaAccepted[message.ID], _ = payload["slot"].(string)
			request.terminal = true
		}
	})
	if err != nil {
		t.Fatalf("expected beta reliability subscription to succeed, got %v", err)
	}
	defer betaSubscription.Cancel()

	queryTopic := "inventory:query"
	queries := []ClientMessage{
		{ID: "req-fast", Kind: ClientQuery, Topic: queryTopic, Source: betaSelf},
		{ID: "req-1", Kind: ClientQuery, Topic: queryTopic, Source: betaSelf},
		{ID: "req-2", Kind: ClientQuery, Topic: queryTopic, Source: betaSelf},
		{ID: "req-slow", Kind: ClientQuery, Topic: queryTopic, Source: betaSelf},
		{ID: "alpha-to-beta", Kind: ClientQuery, Topic: queryTopic, Source: alphaSelf},
		{ID: "beta-to-alpha", Kind: ClientQuery, Topic: queryTopic, Source: betaSelf},
	}
	for _, query := range queries {
		channel := beta
		if query.Source.ID == alphaSelf.ID {
			channel = alpha
		}
		if err := PublishClientMessage(channel, query); err != nil {
			t.Fatalf("expected query %q to publish, got %v", query.ID, err)
		}
	}

	betaTimeouts += expirePending(time.Date(2026, 3, 19, 11, 0, 2, 0, time.UTC), betaPending)
	alphaTimeouts += expirePending(time.Date(2026, 3, 19, 11, 0, 2, 0, time.UTC), alphaPending)

	if err := PublishClientMessage(alpha, ClientMessage{ID: "req-slow", Kind: ClientResult, Topic: queryTopic, Source: alphaSelf, Target: betaSelf.ID, Payload: map[string]any{"slot": "late"}}); err != nil {
		t.Fatalf("expected late result publish to succeed, got %v", err)
	}

	if betaAccepted["req-fast"] != "fast" {
		t.Fatalf("expected request correlation to retain the fast result, got %+v", betaAccepted)
	}
	if betaAccepted["req-1"] != "one" || betaAccepted["req-2"] != "two" {
		t.Fatalf("expected concurrent query results to correlate by request ID, got %+v", betaAccepted)
	}
	if betaAccepted["beta-to-alpha"] != "bidirectional-beta" {
		t.Fatalf("expected simultaneous beta->alpha request to receive a reply, got %+v", betaAccepted)
	}
	if alphaAccepted["alpha-to-beta"] != "bidirectional-alpha" {
		t.Fatalf("expected simultaneous alpha->beta request to receive a reply, got %+v", alphaAccepted)
	}
	if betaDuplicateDrops != 1 {
		t.Fatalf("expected duplicate result tolerance to ignore the second fast result, got %d", betaDuplicateDrops)
	}
	if betaTimeouts != 1 || alphaTimeouts != 0 {
		t.Fatalf("expected only the slow beta request to time out, beta=%d alpha=%d", betaTimeouts, alphaTimeouts)
	}
	if betaLateDrops != 1 {
		t.Fatalf("expected late result discard after timeout, got %d", betaLateDrops)
	}
	if _, ok := betaAccepted["req-slow"]; ok {
		t.Fatalf("expected timed-out request to reject late result, got %+v", betaAccepted)
	}
	if !betaPending["req-fast"].terminal || !betaPending["req-1"].terminal || !betaPending["req-2"].terminal || !betaPending["beta-to-alpha"].terminal || !alphaPending["alpha-to-beta"].terminal {
		t.Fatalf("expected accepted requests to be terminal, beta=%+v alpha=%+v", betaPending, alphaPending)
	}
}

func TestMultiClientSecurityAuthorizationGuards(t *testing.T) {
	storefront := ClientIdentity{ID: "storefront-1", App: "atlas", Surface: "tab", Role: "storefront"}
	operator := ClientIdentity{ID: "ops-1", App: "atlas", Surface: "popup", Role: "operator"}

	crossTab := CrossTabChannel{
		name: func() string { return "atlas-clients" },
		publish: func(payload any) error {
			return nil
		},
	}
	windowChannel := WindowChannel{
		name: func() string { return "atlas-window" },
		publish: func(payload any) error {
			return nil
		},
	}

	if err := PublishClientMessage(crossTab, ClientMessage{Kind: ClientEvent, Topic: "operator:inventory", Source: storefront, Payload: map[string]any{"sku": "SKU-44"}}); !IsCode(err, CodeUnauthorized) {
		t.Fatalf("expected unauthorized topic publish to fail, got %v", err)
	}
	if err := PublishClientIntent(windowChannel, "session:logout", storefront, "popup-1", map[string]any{"reason": "peer-requested"}); !IsCode(err, CodeUnauthorized) {
		t.Fatalf("expected privileged intent rejection for non-operator role, got %v", err)
	}
	if err := PublishClientIntent(windowChannel, "operator:inventory", operator, "popup-1", map[string]any{"sku": "SKU-44"}); err != nil {
		t.Fatalf("expected operator role to publish privileged intent, got %v", err)
	}

	if _, err := DecodeClientMessage(map[string]any{
		"kind":  "event",
		"topic": "operator:inventory",
		"source": map[string]any{
			"id":      "storefront-1",
			"app":     "atlas",
			"surface": "tab",
			"role":    "storefront",
		},
	}); !IsCode(err, CodeUnauthorized) {
		t.Fatalf("expected unauthorized privileged topic decode to fail, got %v", err)
	}

	if _, err := DecodeClientMessage(map[string]any{
		"kind":  "hello",
		"topic": "clients",
		"source": map[string]any{
			"id":      "storefront-1",
			"surface": "tab",
		},
	}); !IsCode(err, CodeInvalid) {
		t.Fatalf("expected malformed peer identity to fail validation, got %v", err)
	}
}

func TestClientCapabilityNegotiation(t *testing.T) {
	local := ClientCapabilities{ProtocolVersion: "v1.2", Encodings: []string{"json", "binary"}, Topics: []string{"clients", "asset:preview"}}
	peer := ClientCapabilities{ProtocolVersion: "1.9", Encodings: []string{"json", "binary"}, Topics: []string{"asset:preview"}}
	if !ClientProtocolCompatible(local, peer) {
		t.Fatal("expected matching major protocol versions to be compatible")
	}
	if !ClientSupportsEncoding(peer, ClientPayloadBinary) {
		t.Fatal("expected peer to advertise binary encoding support")
	}
	if !ClientSupportsTopic(peer, "asset:preview") {
		t.Fatal("expected peer topic support to include asset:preview")
	}
	if !ClientCanExchange(local, peer, "asset:preview", ClientPayloadBinary) {
		t.Fatal("expected local and peer capabilities to allow binary exchange on asset:preview")
	}

	missingBinary := ClientCapabilities{ProtocolVersion: "v1.4", Encodings: []string{"json"}, Topics: []string{"asset:preview"}}
	if ClientCanExchange(local, missingBinary, "asset:preview", ClientPayloadBinary) {
		t.Fatal("expected binary exchange to fail when one peer is json-only")
	}

	wrongTopic := ClientCapabilities{ProtocolVersion: "v1.4", Encodings: []string{"json", "binary"}, Topics: []string{"clients"}}
	if ClientCanExchange(local, wrongTopic, "asset:preview", ClientPayloadJSON) {
		t.Fatal("expected exchange to fail when the peer does not advertise the topic")
	}

	incompatible := ClientCapabilities{ProtocolVersion: "v2.0", Encodings: []string{"json", "binary"}, Topics: []string{"asset:preview"}}
	if ClientProtocolCompatible(local, incompatible) {
		t.Fatal("expected different major protocol versions to be incompatible")
	}
	if ClientCanExchange(local, incompatible, "asset:preview", ClientPayloadJSON) {
		t.Fatal("expected exchange to fail on incompatible protocol versions")
	}

	stagedRollout := ClientCapabilities{ProtocolVersion: "v1.5", Encodings: []string{"json", "binary"}}
	if !ClientCanExchange(local, stagedRollout, "asset:preview", ClientPayloadJSON) {
		t.Fatal("expected topic-unspecified peers to allow additive staged rollout negotiation")
	}

	decoded, err := DecodeClientMessage(map[string]any{
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
	if err != nil {
		t.Fatalf("expected hello message with capabilities to decode, got %v", err)
	} else if decoded.Capabilities == nil || !ClientSupportsEncoding(*decoded.Capabilities, ClientPayloadBinary) || !ClientSupportsTopic(*decoded.Capabilities, "asset:preview") {
		t.Fatalf("unexpected decoded capabilities payload: %+v", decoded.Capabilities)
	}
}

func TestPublishSurfaceSignalHelpers(t *testing.T) {
	var published any
	channel := WindowChannel{
		name: func() string { return "ops" },
		publish: func(payload any) error {
			published = payload
			return nil
		},
	}

	expiresAt := time.Date(2026, 3, 18, 19, 0, 0, 0, time.UTC)
	if err := PublishSessionExpired(channel, "Session expired in another surface.", "/login", expiresAt); err != nil {
		t.Fatalf("expected session-expired helper to publish, got %v", err)
	}
	signal, ok := published.(SurfaceSignal)
	if !ok {
		t.Fatalf("expected surface signal payload, got %T", published)
	}
	if signal.Kind != SurfaceSignalSession || signal.Session == nil || signal.Session.Status != "expired" || signal.Session.ReturnTo != "/login" || !signal.Session.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("unexpected session signal payload: %+v", signal)
	}

	if err := PublishRouteFocus(channel, "/orders/42", "tab=activity", "order-heading"); err != nil {
		t.Fatalf("expected route-focus helper to publish, got %v", err)
	}
	signal, ok = published.(SurfaceSignal)
	if !ok || signal.Route == nil || signal.Route.Path != "/orders/42" || signal.Route.Query != "tab=activity" || signal.Route.FocusID != "order-heading" {
		t.Fatalf("unexpected route signal payload: %+v", published)
	}

	if err := PublishSelection(channel, "invoice", "INV-204", "rev-12"); err != nil {
		t.Fatalf("expected selection helper to publish, got %v", err)
	}
	signal, ok = published.(SurfaceSignal)
	if !ok || signal.Selection == nil || signal.Selection.Scope != "invoice" || signal.Selection.ID != "INV-204" || signal.Selection.Revision != "rev-12" {
		t.Fatalf("unexpected selection signal payload: %+v", published)
	}

	if err := PublishIntent(channel, SurfaceIntentFocusPanel, "audit-log", map[string]string{"tab": "alerts"}); err != nil {
		t.Fatalf("expected intent helper to publish, got %v", err)
	}
	signal, ok = published.(SurfaceSignal)
	if !ok || signal.Intent == nil || signal.Intent.Action != SurfaceIntentFocusPanel || signal.Intent.Target != "audit-log" || signal.Intent.Params["tab"] != "alerts" {
		t.Fatalf("unexpected intent signal payload: %+v", published)
	}
}

func TestDecodeSurfaceSignalProjectsTypedPayload(t *testing.T) {
	message := WindowEnvelope{
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
	decoded, err := DecodeSurfaceSignal(message)
	if err != nil {
		t.Fatalf("expected typed surface signal, got %v", err)
	}
	if decoded.Name != "ops" || decoded.Source != "popup-1" {
		t.Fatalf("unexpected envelope metadata: %+v", decoded)
	}
	if decoded.Payload.Kind != SurfaceSignalSelection || decoded.Payload.Selection == nil || decoded.Payload.Selection.ID != "SKU-42" {
		t.Fatalf("unexpected decoded surface payload: %+v", decoded.Payload)
	}
}
