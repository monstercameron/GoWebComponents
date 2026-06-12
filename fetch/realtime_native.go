//go:build !js || !wasm

package fetch

// buildWebSocketInitialState returns the non-browser WebSocket fallback state.
func buildWebSocketInitialState() RealtimeState {
	return RealtimeState{
		Status:    RealtimeUnsupported,
		Supported: false,
		Closed:    true,
		Error:     realtimeUnsupportedError{api: "WebSocket"},
	}
}

// buildEventSourceInitialState returns the non-browser EventSource fallback state.
func buildEventSourceInitialState() RealtimeState {
	return RealtimeState{
		Status:    RealtimeUnsupported,
		Supported: false,
		Closed:    true,
		Error:     realtimeUnsupportedError{api: "EventSource"},
	}
}

// openWebSocketTransport reports that WebSocket is unavailable off js/wasm.
func openWebSocketTransport(parseURL string, parseOptions realtimeResolvedOptions, parseCallbacks realtimeTransportCallbacks) (realtimeTransport, error) {
	_ = parseURL
	_ = parseOptions
	_ = parseCallbacks
	return nil, realtimeUnsupportedError{api: "WebSocket"}
}

// openEventSourceTransport reports that EventSource is unavailable off js/wasm.
func openEventSourceTransport(parseURL string, parseOptions realtimeResolvedOptions, parseCallbacks realtimeTransportCallbacks) (realtimeTransport, error) {
	_ = parseURL
	_ = parseOptions
	_ = parseCallbacks
	return nil, realtimeUnsupportedError{api: "EventSource"}
}
