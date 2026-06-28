package ui

import (
	"fmt"
	"reflect"
	"sync"
)

// inspectSink receives UseInspect messages. It is swappable so the dev log can be routed
// (devtools, a test buffer, or silenced in production).
var (
	inspectMu   sync.RWMutex
	inspectSink = func(parseMessage string) { fmt.Println(parseMessage) }
)

// SetInspectSink redirects UseInspect output (e.g. to the devtools bridge or a test buffer)
// and returns a function that restores the previous sink. Pass a no-op to silence
// inspection in production builds.
func SetInspectSink(parseSink func(string)) func() {
	inspectMu.Lock()
	parsePrevious := inspectSink
	if parseSink != nil {
		inspectSink = parseSink
	}
	inspectMu.Unlock()
	return func() {
		inspectMu.Lock()
		inspectSink = parsePrevious
		inspectMu.Unlock()
	}
}

// emitInspect sends a message to the current sink.
func emitInspect(parseMessage string) {
	inspectMu.RLock()
	parseSink := inspectSink
	inspectMu.RUnlock()
	parseSink(parseMessage)
}

// inspectMessage formats the dev log for one observed value: the initial value on mount, a
// "old -> new" line when it changed, or "" (no log) when unchanged. Kept pure so the
// change-detection and formatting are unit-testable without a render lifecycle.
func inspectMessage[T any](parseLabel string, parsePrevious T, parseNext T, parseInit bool) (string, bool) {
	if parseInit {
		return fmt.Sprintf("[inspect] %s = %v (init)", parseLabel, parseNext), true
	}
	if reflect.DeepEqual(parsePrevious, parseNext) {
		return "", false
	}
	return fmt.Sprintf("[inspect] %s: %v -> %v", parseLabel, parsePrevious, parseNext), true
}

// UseInspect logs a labeled value whenever it changes between renders — GoWebComponents'
// answer to Svelte's $inspect. It records the initial value on mount and, on each
// subsequent render, logs an "old -> new" line when the value changed (by structural
// equality), routed through the current inspect sink. A pure dev aid: leave it in during
// development and silence it in production either at runtime with
// SetInspectSink(func(string){}) or, at zero per-call cost, by building with the gwcsilent
// tag (`go build -tags gwcsilent`), which wires the sink to a no-op.
//
//	count := ui.UseState(0)
//	ui.UseInspect("count", count.Get()) // logs each time count changes
func UseInspect[T any](parseLabel string, parseValue T) {
	parsePrevious := UseRef[*T](nil)
	parsePrev := parsePrevious.Get()

	parseMessage, parseChanged := inspectMessage(parseLabel, deref(parsePrev), parseValue, parsePrev == nil)
	if parseChanged {
		emitInspect(parseMessage)
		parseStored := parseValue
		parsePrevious.Set(&parseStored)
	}
}

// deref returns the pointed-to value, or the zero value for a nil pointer.
func deref[T any](parsePtr *T) T {
	if parsePtr == nil {
		var parseZero T
		return parseZero
	}
	return *parsePtr
}
