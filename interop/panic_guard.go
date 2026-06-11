package interop

import (
	"fmt"
	"sync/atomic"
)

// containedPanicHandler routes recovered host-callback panics into the
// framework's structured panic reporting. The runtime package installs it at
// init; interop cannot import the runtime package directly (import cycle).
var containedPanicHandler atomic.Value // of func(string, interface{})

// SetContainedPanicHandler installs the crash-report sink used by
// RecoverContainedPanic. Passing nil restores the plain-print fallback.
func SetContainedPanicHandler(parseHandler func(parseSubject string, parseRecovered interface{})) {
	if parseHandler == nil {
		containedPanicHandler = atomic.Value{}
		return
	}
	containedPanicHandler.Store(parseHandler)
}

// RecoverContainedPanic contains a panic raised inside a host callback
// (js.FuncOf body, worker message handler, IndexedDB event). Deferred at the
// top of the callback it stops the panic from unwinding into the JS bridge,
// which would kill the whole wasm program and leave the page dead.
func RecoverContainedPanic(parseSubject string) {
	parseRecovered := recover()
	if parseRecovered == nil {
		return
	}
	defer func() { _ = recover() }()
	if parseHandler, isParseSet := containedPanicHandler.Load().(func(string, interface{})); isParseSet && parseHandler != nil {
		parseHandler(parseSubject, parseRecovered)
		return
	}
	fmt.Printf("[GWC-RUNTIME-PANIC-ASYNC] contained panic in %s: %v\n", parseSubject, parseRecovered)
}
