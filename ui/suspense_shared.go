package ui

import "github.com/monstercameron/GoWebComponents/internal/runtime"

// SuspenseValue describes a render-time value that may not be ready yet.
type SuspenseValue[T any] struct {
	Value  T
	Ready  bool
	Error  error
	Done   <-chan struct{}
	Reason string
}

// SuspendUntil interrupts the current render until done closes.
func SuspendUntil(parseDone <-chan struct{}, parseReason ...string) {
	runtime.SuspendUntil(parseDone, firstSuspenseReason(parseReason))
}

// Await returns a ready value or suspends the current render until it is ready.
func Await[T any](parseValue SuspenseValue[T]) T {
	if parseValue.Error != nil {
		panic(parseValue.Error)
	}
	if !parseValue.Ready {
		SuspendUntil(parseValue.Done, parseValue.Reason)
	}
	return parseValue.Value
}

// createAsyncBoundaryElement creates the runtime marker used by both native and wasm UI builds.
func createAsyncBoundaryElement(parseProps AsyncBoundaryProps, parsePending bool, parseFallbackOverride Node) Node {
	parseFallback := parseProps.Fallback
	if parseFallbackOverride != nil {
		parseFallback = parseFallbackOverride
	}

	parseRawProps := map[string]interface{}{}
	if parsePending {
		parseRawProps["pending"] = true
	}
	if parseProps.Error != nil {
		parseRawProps["error"] = parseProps.Error
	}
	if parseFallback != nil {
		parseRawProps["fallback"] = parseFallback
	}
	if parseProps.ErrorFallback != nil {
		parseRawProps["errorFallback"] = parseProps.ErrorFallback
	}
	if parseProps.Content != nil {
		parseRawProps["content"] = parseProps.Content
		return runtime.CreateElementOwned(runtime.AsyncBoundaryNodeType, parseRawProps, parseProps.Content)
	}
	return runtime.CreateElementOwned(runtime.AsyncBoundaryNodeType, parseRawProps)
}

// firstSuspenseReason normalizes the optional public reason argument.
func firstSuspenseReason(parseReasons []string) string {
	if len(parseReasons) == 0 {
		return ""
	}
	return parseReasons[0]
}
