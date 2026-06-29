//go:build js && wasm

package runtime

import (
	"fmt"
	"syscall/js"
)

// writeInvokeHandlerPlatform handles WASM-only handler signatures:
// func(js.Value), func(js.Value) error, func(GoEvent), func(GoEvent) error.
// It is called by writeInvokeHandler when the common type-switch falls through
// to the default case. A zero js.Value (js.Undefined()) is supplied as the
// synthetic event so handlers that only read it defensively still work.
func writeInvokeHandlerPlatform(parseHandler any, parseEventName string, parseRef string) error {
	parseZeroJS := js.Undefined()
	parseZeroGoEvent := NewGoEvent(parseZeroJS)

	switch parseTyped := parseHandler.(type) {
	case func(js.Value):
		parseTyped(parseZeroJS)
		return nil
	case func(js.Value) error:
		if parseCallErr := parseTyped(parseZeroJS); parseCallErr != nil {
			return fmt.Errorf("handler %q on ref %q returned error: %w", parseEventName, parseRef, parseCallErr)
		}
		return nil
	case func(GoEvent):
		parseTyped(parseZeroGoEvent)
		return nil
	case func(GoEvent) error:
		if parseCallErr2 := parseTyped(parseZeroGoEvent); parseCallErr2 != nil {
			return fmt.Errorf("handler %q on ref %q returned error: %w", parseEventName, parseRef, parseCallErr2)
		}
		return nil
	default:
		return fmt.Errorf("handler %q on ref %q has unsupported type %T: %w",
			parseEventName, parseRef, parseHandler, ErrAgentNoHandler)
	}
}
