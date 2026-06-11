//go:build js && wasm
// +build js,wasm

package fetch

import (
	"fmt"
	"syscall/js"
	"time"
)

// buildWebSocketInitialState returns the browser WebSocket initial state.
func buildWebSocketInitialState() RealtimeState {
	return RealtimeState{
		Status:    RealtimeIdle,
		Supported: true,
		Closed:    true,
	}
}

// buildEventSourceInitialState returns the browser EventSource initial state.
func buildEventSourceInitialState() RealtimeState {
	return RealtimeState{
		Status:    RealtimeIdle,
		Supported: true,
		Closed:    true,
	}
}

type realtimeBrowserListener struct {
	name       string
	property   string
	fn         js.Func
	isProperty bool
}

type realtimeBrowserTransport struct {
	raw       js.Value
	listeners []realtimeBrowserListener
	isClosed  bool
}

// openWebSocketTransport opens one browser WebSocket transport.
func openWebSocketTransport(parseURL string, parseOptions realtimeResolvedOptions, parseCallbacks realtimeTransportCallbacks) (parseTransport realtimeTransport, parseErr error) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseTransport = nil
			parseErr = fmt.Errorf("open WebSocket: %v", parseRecovered)
		}
	}()

	parseCtor := js.Global().Get("WebSocket")
	if parseCtor.Type() != js.TypeFunction {
		return nil, realtimeUnsupportedError{api: "WebSocket"}
	}

	var parseRaw js.Value
	switch len(parseOptions.protocols) {
	case 0:
		parseRaw = parseCtor.New(parseURL)
	case 1:
		parseRaw = parseCtor.New(parseURL, parseOptions.protocols[0])
	default:
		parseProtocols := js.Global().Get("Array").New()
		for _, parseProtocol := range parseOptions.protocols {
			parseProtocols.Call("push", parseProtocol)
		}
		parseRaw = parseCtor.New(parseURL, parseProtocols)
	}

	parseTransport2 := &realtimeBrowserTransport{raw: parseRaw}
	parseTransport2.addListener("open", func(parseEvent js.Value) {
		_ = parseEvent
		parseCallbacks.handleOpen()
	})
	parseTransport2.addListener("message", func(parseEvent js.Value) {
		parseCallbacks.handleMessage(buildRealtimeMessageFromJS(parseEvent, parseOptions.now()))
	})
	parseTransport2.addListener("error", func(parseEvent js.Value) {
		parseCallbacks.handleError(buildRealtimeErrorFromJS("WebSocket error", parseEvent))
	})
	parseTransport2.addListener("close", func(parseEvent js.Value) {
		_ = parseEvent
		parseCallbacks.handleClose()
	})
	return parseTransport2, nil
}

// openEventSourceTransport opens one browser EventSource transport.
func openEventSourceTransport(parseURL string, parseOptions realtimeResolvedOptions, parseCallbacks realtimeTransportCallbacks) (parseTransport realtimeTransport, parseErr error) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseTransport = nil
			parseErr = fmt.Errorf("open EventSource: %v", parseRecovered)
		}
	}()

	parseCtor := js.Global().Get("EventSource")
	if parseCtor.Type() != js.TypeFunction {
		return nil, realtimeUnsupportedError{api: "EventSource"}
	}

	var parseRaw js.Value
	if parseOptions.withCredentials {
		parseInit := js.Global().Get("Object").New()
		parseInit.Set("withCredentials", true)
		parseRaw = parseCtor.New(parseURL, parseInit)
	} else {
		parseRaw = parseCtor.New(parseURL)
	}

	parseTransport2 := &realtimeBrowserTransport{raw: parseRaw}
	parseTransport2.addListener("open", func(parseEvent js.Value) {
		_ = parseEvent
		parseCallbacks.handleOpen()
	})
	parseTransport2.addListener("message", func(parseEvent js.Value) {
		parseCallbacks.handleMessage(buildRealtimeMessageFromJS(parseEvent, parseOptions.now()))
	})
	parseTransport2.addListener("error", func(parseEvent js.Value) {
		parseCallbacks.handleError(buildRealtimeErrorFromJS("EventSource error", parseEvent))
		parseCallbacks.handleClose()
	})
	return parseTransport2, nil
}

// send sends one text frame when the browser transport supports send.
func (parseT *realtimeBrowserTransport) send(parseMessage string) (parseErr error) {
	if parseT == nil || parseT.isClosed {
		return fmt.Errorf("realtime transport is closed")
	}
	parseSend := parseT.raw.Get("send")
	if parseSend.Type() != js.TypeFunction {
		return fmt.Errorf("realtime transport cannot send messages")
	}
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseErr = fmt.Errorf("send realtime message: %v", parseRecovered)
		}
	}()
	parseT.raw.Call("send", parseMessage)
	return nil
}

// close closes the browser transport and releases registered callbacks.
func (parseT *realtimeBrowserTransport) close() (parseErr error) {
	if parseT == nil || parseT.isClosed {
		return nil
	}
	parseT.isClosed = true
	parseRemove := parseT.raw.Get("removeEventListener")
	for _, parseListener := range parseT.listeners {
		if parseListener.isProperty {
			parseT.raw.Set(parseListener.property, js.Null())
		} else if parseRemove.Type() == js.TypeFunction {
			parseT.raw.Call("removeEventListener", parseListener.name, parseListener.fn)
		}
	}
	parseClose := parseT.raw.Get("close")
	if parseClose.Type() == js.TypeFunction {
		defer func() {
			if parseRecovered := recover(); parseRecovered != nil {
				parseErr = fmt.Errorf("close realtime transport: %v", parseRecovered)
			}
			parseT.releaseListeners()
		}()
		parseT.raw.Call("close")
		return nil
	}
	parseT.releaseListeners()
	return nil
}

// addListener registers one browser event callback.
func (parseT *realtimeBrowserTransport) addListener(parseName string, parseHandler func(js.Value)) {
	parseFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseEvent := js.Undefined()
		if len(parseArgs) > 0 {
			parseEvent = parseArgs[0]
		}
		parseHandler(parseEvent)
		return nil
	})
	parseAdd := parseT.raw.Get("addEventListener")
	if parseAdd.Type() == js.TypeFunction {
		parseT.raw.Call("addEventListener", parseName, parseFn)
		parseT.listeners = append(parseT.listeners, realtimeBrowserListener{name: parseName, fn: parseFn})
		return
	}
	parseProperty := "on" + parseName
	parseT.raw.Set(parseProperty, parseFn)
	parseT.listeners = append(parseT.listeners, realtimeBrowserListener{name: parseName, property: parseProperty, fn: parseFn, isProperty: true})
}

// releaseListeners releases all Go callbacks registered with the browser.
func (parseT *realtimeBrowserTransport) releaseListeners() {
	for _, parseListener := range parseT.listeners {
		parseListener.fn.Release()
	}
	parseT.listeners = nil
}

// buildRealtimeMessageFromJS converts one browser MessageEvent to a public message.
func buildRealtimeMessageFromJS(parseEvent js.Value, parseNow time.Time) RealtimeMessage {
	parseMessage := RealtimeMessage{ReceivedAt: parseNow}
	if parseEvent.IsUndefined() || parseEvent.IsNull() {
		return parseMessage
	}
	parseType := parseEvent.Get("type")
	if parseType.Type() == js.TypeString {
		parseMessage.Type = parseType.String()
	}
	parseLastEventID := parseEvent.Get("lastEventId")
	if parseLastEventID.Type() == js.TypeString {
		parseMessage.LastEventID = parseLastEventID.String()
	}
	parseData := parseEvent.Get("data")
	switch parseData.Type() {
	case js.TypeString:
		parseMessage.Data = parseData.String()
	case js.TypeNumber, js.TypeBoolean:
		parseMessage.Data = fmt.Sprint(parseData)
	case js.TypeObject:
		if parseLength := parseData.Get("byteLength"); parseLength.Type() == js.TypeNumber {
			parseMessage.Data = fmt.Sprintf("[binary:%d]", parseLength.Int())
		} else {
			parseMessage.Data = parseData.String()
		}
	}
	return parseMessage
}

// buildRealtimeErrorFromJS converts one browser error event to a Go error.
func buildRealtimeErrorFromJS(parseFallback string, parseEvent js.Value) error {
	if parseEvent.IsUndefined() || parseEvent.IsNull() {
		return fmt.Errorf("%s", parseFallback)
	}
	if parseMessage := parseEvent.Get("message"); parseMessage.Type() == js.TypeString && parseMessage.String() != "" {
		return fmt.Errorf("%s", parseMessage.String())
	}
	if parseError := parseEvent.Get("error"); !parseError.IsUndefined() && !parseError.IsNull() {
		if parseErrorMessage := parseError.Get("message"); parseErrorMessage.Type() == js.TypeString && parseErrorMessage.String() != "" {
			return fmt.Errorf("%s", parseErrorMessage.String())
		}
		return fmt.Errorf("%s: %s", parseFallback, parseError.String())
	}
	return fmt.Errorf("%s", parseFallback)
}
