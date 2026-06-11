//go:build js && wasm
// +build js,wasm

package fetch

import (
	"strings"
	"syscall/js"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

func TestUseWebSocketReceivesBoundedMessagesAndSends(parseT *testing.T) {
	installFetchHookContext(parseT)
	parseSockets, parseRestoreSocket := installRealtimeMockConstructor(parseT, "WebSocket", true)
	defer parseRestoreSocket()

	parseFiber := runtime.GetCurrentFiber()
	parseSocket := UseWebSocket("wss://example.test/live", WebSocketOptions{
		MaxMessages: 2,
		Protocols:   []string{"json"},
	})
	parseEffects := getFetchTestFiberEffects(parseT, parseFiber)
	if len(parseEffects) != 1 {
		parseT.Fatalf("expected one WebSocket effect, got %d", len(parseEffects))
	}
	parseEffects[0].Fn()
	if len(*parseSockets) != 1 {
		parseT.Fatalf("expected one WebSocket instance, got %d", len(*parseSockets))
	}
	parseRaw := (*parseSockets)[0]
	parseRaw.Call("__emit", "open", js.Global().Get("Object").New())
	if parseState := parseSocket.Get(); parseState.Status != RealtimeOpen || !parseState.Open || parseState.Connecting {
		parseT.Fatalf("expected open WebSocket state, got %+v", parseState)
	}

	for _, parsePayload := range []string{"one", "two", "three"} {
		parseEvent := js.Global().Get("Object").New()
		parseEvent.Set("type", "message")
		parseEvent.Set("data", parsePayload)
		parseRaw.Call("__emit", "message", parseEvent)
	}
	parseState := parseSocket.Get()
	if len(parseState.Messages) != 2 || parseState.Messages[0].Data != "two" || parseState.Messages[1].Data != "three" || parseState.LastMessage.Data != "three" {
		parseT.Fatalf("expected bounded WebSocket messages, got %+v", parseState)
	}

	if parseErr := parseSocket.Send("client-ping"); parseErr != nil {
		parseT.Fatalf("expected WebSocket send to succeed, got %v", parseErr)
	}
	parseSent := parseRaw.Get("__sent")
	if parseSent.Length() != 1 || parseSent.Index(0).String() != "client-ping" {
		parseT.Fatalf("expected sent WebSocket message, got length=%d", parseSent.Length())
	}
	parseSocket.Close()
	if parseState2 := parseSocket.Get(); parseState2.Status != RealtimeClosed || parseState2.Open || !parseState2.Closed {
		parseT.Fatalf("expected closed WebSocket state, got %+v", parseState2)
	}
}

func TestUseWebSocketReconnectsWithBoundedBackoff(parseT *testing.T) {
	installFetchHookContext(parseT)
	parseSockets, parseRestoreSocket := installRealtimeMockConstructor(parseT, "WebSocket", true)
	defer parseRestoreSocket()

	parseFiber := runtime.GetCurrentFiber()
	parseSocket := UseWebSocket("wss://example.test/retry", WebSocketOptions{
		MaxReconnects:  1,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
	})
	parseEffects := getFetchTestFiberEffects(parseT, parseFiber)
	parseEffects[0].Fn()
	if len(*parseSockets) != 1 {
		parseT.Fatalf("expected one initial socket, got %d", len(*parseSockets))
	}
	(*parseSockets)[0].Call("__emit", "open", js.Global().Get("Object").New())
	(*parseSockets)[0].Call("__emit", "close", js.Global().Get("Object").New())
	waitRealtimeWasmCondition(parseT, time.Second, func() bool {
		return len(*parseSockets) == 2
	})
	parseState := parseSocket.Get()
	if parseState.Reconnects != 1 || parseState.ConnectAttempts != 2 {
		parseT.Fatalf("expected one reconnect attempt and two connects, got %+v", parseState)
	}
}

func TestUseWebSocketHeartbeatSendsConfiguredMessage(parseT *testing.T) {
	installFetchHookContext(parseT)
	parseSockets, parseRestoreSocket := installRealtimeMockConstructor(parseT, "WebSocket", true)
	defer parseRestoreSocket()

	parseFiber := runtime.GetCurrentFiber()
	parseSocket := UseWebSocket("wss://example.test/heartbeat", WebSocketOptions{
		HeartbeatInterval: time.Millisecond,
		HeartbeatMessage:  "hb",
	})
	parseEffects := getFetchTestFiberEffects(parseT, parseFiber)
	parseEffects[0].Fn()
	parseRaw := (*parseSockets)[0]
	parseRaw.Call("__emit", "open", js.Global().Get("Object").New())
	waitRealtimeWasmCondition(parseT, time.Second, func() bool {
		return parseRaw.Get("__sent").Length() > 0 && !parseSocket.Get().LastHeartbeatAt.IsZero()
	})
	if parseSent := parseRaw.Get("__sent").Index(0).String(); parseSent != "hb" {
		parseT.Fatalf("expected configured heartbeat message, got %q", parseSent)
	}
	parseSocket.Close()
}

func TestUseEventSourceReceivesMessagesAndReconnectsOnError(parseT *testing.T) {
	installFetchHookContext(parseT)
	parseSources, parseRestoreSource := installRealtimeMockConstructor(parseT, "EventSource", false)
	defer parseRestoreSource()

	parseFiber := runtime.GetCurrentFiber()
	parseEvents := UseEventSource("/api/events", EventSourceOptions{
		MaxReconnects:  1,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
		MaxMessages:    1,
	})
	parseEffects := getFetchTestFiberEffects(parseT, parseFiber)
	parseEffects[0].Fn()
	if len(*parseSources) != 1 {
		parseT.Fatalf("expected one EventSource instance, got %d", len(*parseSources))
	}
	parseRaw := (*parseSources)[0]
	parseRaw.Call("__emit", "open", js.Global().Get("Object").New())
	parseMessage := js.Global().Get("Object").New()
	parseMessage.Set("type", "message")
	parseMessage.Set("data", "ready")
	parseMessage.Set("lastEventId", "evt-1")
	parseRaw.Call("__emit", "message", parseMessage)
	if parseState := parseEvents.Get(); len(parseState.Messages) != 1 || parseState.LastMessage.Data != "ready" || parseState.LastMessage.LastEventID != "evt-1" {
		parseT.Fatalf("expected EventSource message state, got %+v", parseState)
	}

	parseError := js.Global().Get("Object").New()
	parseError.Set("message", "network dropped")
	parseRaw.Call("__emit", "error", parseError)
	waitRealtimeWasmCondition(parseT, time.Second, func() bool {
		return len(*parseSources) == 2
	})
	parseState := parseEvents.Get()
	if parseState.Reconnects != 1 || parseState.Error == nil || !strings.Contains(parseState.Error.Error(), "network dropped") {
		parseT.Fatalf("expected EventSource reconnect state with error, got %+v", parseState)
	}
}

func installRealtimeMockConstructor(parseT *testing.T, parseName string, shouldSupportSend bool) (*[]js.Value, func()) {
	parseT.Helper()
	parseInstances := []js.Value{}
	parseFuncs := []js.Func{}
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseListeners := map[string][]js.Value{}
		parseRaw.Set("__url", parseArgs[0])
		if len(parseArgs) > 1 {
			parseRaw.Set("__init", parseArgs[1])
		}
		parseRaw.Set("__sent", js.Global().Get("Array").New())
		parseRaw.Set("__closed", false)

		parseAdd := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			parseEvent := parseArgs2[0].String()
			parseListeners[parseEvent] = append(parseListeners[parseEvent], parseArgs2[1])
			return nil
		})
		parseRemove := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			parseEvent := parseArgs3[0].String()
			parseCallback := parseArgs3[1]
			parseCurrent := parseListeners[parseEvent]
			parseNext := parseCurrent[:0]
			for _, parseListener := range parseCurrent {
				if !parseListener.Equal(parseCallback) {
					parseNext = append(parseNext, parseListener)
				}
			}
			parseListeners[parseEvent] = parseNext
			return nil
		})
		parseEmit := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseEvent := parseArgs4[0].String()
			parsePayload := js.Global().Get("Object").New()
			if len(parseArgs4) > 1 {
				parsePayload = parseArgs4[1]
			}
			for _, parseListener := range append([]js.Value(nil), parseListeners[parseEvent]...) {
				parseListener.Invoke(parsePayload)
			}
			parseProperty := parseRaw.Get("on" + parseEvent)
			if parseProperty.Type() == js.TypeFunction {
				parseProperty.Invoke(parsePayload)
			}
			return nil
		})
		parseClose := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			_ = parseArgs5
			parseRaw.Set("__closed", true)
			return nil
		})
		parseFuncs = append(parseFuncs, parseAdd, parseRemove, parseEmit, parseClose)
		parseRaw.Set("addEventListener", parseAdd)
		parseRaw.Set("removeEventListener", parseRemove)
		parseRaw.Set("__emit", parseEmit)
		parseRaw.Set("close", parseClose)
		if shouldSupportSend {
			parseSend := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
				parseRaw.Get("__sent").Call("push", parseArgs6[0])
				return nil
			})
			parseFuncs = append(parseFuncs, parseSend)
			parseRaw.Set("send", parseSend)
		}
		parseInstances = append(parseInstances, parseRaw)
		return parseRaw
	})
	parseFuncs = append(parseFuncs, parseCtor)
	parseRestore := setGlobalJSValue(parseName, parseCtor)
	return &parseInstances, func() {
		parseRestore()
		for _, parseFn := range parseFuncs {
			parseFn.Release()
		}
	}
}

func waitRealtimeWasmCondition(parseT *testing.T, parseTimeout time.Duration, parseCheck func() bool) {
	parseT.Helper()
	parseDeadline := time.Now().Add(parseTimeout)
	for time.Now().Before(parseDeadline) {
		if parseCheck() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	parseT.Fatal("timed out waiting for realtime wasm test condition")
}
