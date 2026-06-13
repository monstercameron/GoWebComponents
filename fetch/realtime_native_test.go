//go:build !js || !wasm

package fetch

import (
	"strings"
	"testing"
	"time"
)

func TestRealtimeNativeHooksReturnUnsupportedState(parseT *testing.T) {
	installFetchTestHookContext(parseT)

	parseSocket := UseWebSocket("wss://example.test/events")
	parseSocketState := parseSocket.Get()
	if parseSocketState.Status != RealtimeUnsupported || parseSocketState.Supported || !parseSocketState.Closed {
		parseT.Fatalf("unexpected native WebSocket state: %+v", parseSocketState)
	}
	if parseSocketState.Error == nil || !strings.Contains(parseSocketState.Error.Error(), "WebSocket API unavailable") {
		parseT.Fatalf("expected native WebSocket unsupported error, got %v", parseSocketState.Error)
	}
	if parseErr := parseSocket.Send("ping"); parseErr == nil || !strings.Contains(parseErr.Error(), "not open") {
		parseT.Fatalf("expected unopened native WebSocket send error, got %v", parseErr)
	}
	parseSocket.Open()
	parseSocket.Close()

	parseEvents := UseEventSource("/api/events")
	parseEventState := parseEvents.Get()
	if parseEventState.Status != RealtimeUnsupported || parseEventState.Supported || !parseEventState.Closed {
		parseT.Fatalf("unexpected native EventSource state: %+v", parseEventState)
	}
	if parseEventState.Error == nil || !strings.Contains(parseEventState.Error.Error(), "EventSource API unavailable") {
		parseT.Fatalf("expected native EventSource unsupported error, got %v", parseEventState.Error)
	}
	parseEvents.Open()
	parseEvents.Close()
}

func TestRealtimeNativeTransportsReportUnsupported(parseT *testing.T) {
	parseWSState := buildWebSocketInitialState()
	if parseWSState.Status != RealtimeUnsupported || parseWSState.Supported || !parseWSState.Closed || !isRealtimeUnsupportedError(parseWSState.Error) {
		parseT.Fatalf("unexpected websocket initial state: %+v", parseWSState)
	}
	parseESState := buildEventSourceInitialState()
	if parseESState.Status != RealtimeUnsupported || parseESState.Supported || !parseESState.Closed || !isRealtimeUnsupportedError(parseESState.Error) {
		parseT.Fatalf("unexpected eventsource initial state: %+v", parseESState)
	}
	if parseTransport, parseErr := openWebSocketTransport("wss://example.test", realtimeResolvedOptions{}, realtimeTransportCallbacks{}); parseTransport != nil || !isRealtimeUnsupportedError(parseErr) {
		parseT.Fatalf("openWebSocketTransport = (%#v, %v), want nil unsupported", parseTransport, parseErr)
	}
	if parseTransport, parseErr := openEventSourceTransport("/events", realtimeResolvedOptions{}, realtimeTransportCallbacks{}); parseTransport != nil || !isRealtimeUnsupportedError(parseErr) {
		parseT.Fatalf("openEventSourceTransport = (%#v, %v), want nil unsupported", parseTransport, parseErr)
	}
}

func TestRealtimeOptionNormalizationAndBounds(parseT *testing.T) {
	parseNow := time.Date(2026, time.June, 11, 12, 0, 0, 0, time.UTC)
	parseOptions := resolveWebSocketOptions([]WebSocketOptions{{
		Protocols:      []string{"json", "v2"},
		MaxReconnects:  2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     40 * time.Millisecond,
		BackoffFactor:  3,
		MaxMessages:    2,
		MaxErrors:      1,
		Now:            func() time.Time { return parseNow },
	}})
	if parseOptions.now() != parseNow || parseOptions.maxReconnects != 2 || parseOptions.maxMessages != 2 || parseOptions.maxErrors != 1 {
		parseT.Fatalf("unexpected resolved WebSocket options: %+v", parseOptions)
	}
	if parseBackoff := calculateRealtimeBackoff(3, parseOptions.initialBackoff, parseOptions.maxBackoff, parseOptions.backoffFactor); parseBackoff != 40*time.Millisecond {
		parseT.Fatalf("expected capped backoff, got %s", parseBackoff)
	}

	parseMessages := appendRealtimeMessages(nil, RealtimeMessage{Data: "one"}, 2)
	parseMessages = appendRealtimeMessages(parseMessages, RealtimeMessage{Data: "two"}, 2)
	parseMessages = appendRealtimeMessages(parseMessages, RealtimeMessage{Data: "three"}, 2)
	if len(parseMessages) != 2 || parseMessages[0].Data != "two" || parseMessages[1].Data != "three" {
		parseT.Fatalf("expected bounded messages to keep newest entries, got %+v", parseMessages)
	}

	parseErrors := appendRealtimeErrors(nil, RealtimeError{Message: "first"}, 1)
	parseErrors = appendRealtimeErrors(parseErrors, RealtimeError{Message: "second"}, 1)
	if len(parseErrors) != 1 || parseErrors[0].Message != "second" {
		parseT.Fatalf("expected bounded errors to keep newest entry, got %+v", parseErrors)
	}
}
