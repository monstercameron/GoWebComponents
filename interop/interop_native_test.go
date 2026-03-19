//go:build !js || !wasm
// +build !js !wasm

package interop

import (
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
		{name: "ImportModule", err: func() error { _, err := ImportModule(nil, "/demo.js"); return err }()},
		{name: "NewWorker", err: func() error { _, err := NewWorker(nil, WorkerOptions{URL: "/worker.js"}); return err }()},
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
