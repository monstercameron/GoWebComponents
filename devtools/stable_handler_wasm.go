//go:build js && wasm && !production

package devtools

import (
	"sync"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// Stable event handlers for devtools' own UI.
//
// ui.WrapHandler mints a js.Func through syscall/js and nothing releases it.
// That is fine for its documented use — forwarding an already-prepared handler
// value, where the Kind check skips wrapping entirely — and a leak everywhere
// else, because a js.Func is only reclaimed by an explicit Release().
//
// Devtools was the worst possible caller: renderNode wraps a fresh inline
// closure per NODE, recursively, on every devtools render, so inspecting a
// 500-node tree leaked 500 JS callbacks per repaint, forever. The framework's
// own inspector was the largest handler leak in the repository.
//
// The fix is the pattern the runtime already uses for GoUseFunc (funcHandlerCell
// in internal/runtime): wrap ONCE per stable key and keep a cell holding the
// latest closure, so the wrapper is reused while still dispatching to the
// current state. A component would get this from ui.UseEvent, but these are
// plain recursive render functions with no hook slots to sit in — the node set
// changes shape between renders, so hook order could not be stable anyway.
//
// The cache is keyed by call site plus identity (a node path, an action label),
// which bounds it by the size of the inspected tree rather than by the number of
// renders: a bounded working set instead of an unbounded leak. Devtools is
// dev-only and single-threaded in wasm; the mutex is for correctness under the
// native test build of neighbouring code, not for contention.
var (
	stableHandlerMu    sync.Mutex
	stableHandlerCells = map[string]*stableHandlerCell{}
)

type stableHandlerCell struct {
	fn      func()
	handler ui.Handler
}

// stableHandler returns a handler that is wrapped once per key and thereafter
// reused, dispatching to the most recently supplied closure.
//
// Callers must pass a key that identifies the LOGICAL control, not the render:
// keying by render index would defeat the whole point and quietly restore the
// leak.
func stableHandler(parseKey string, parseFn func()) ui.Handler {
	stableHandlerMu.Lock()
	parseCell, hasCell := stableHandlerCells[parseKey]
	if !hasCell {
		parseCell = &stableHandlerCell{}
		// The wrapped closure reads through the cell, so the js.Func created
		// here stays valid no matter how many times the caller re-renders with a
		// different closure.
		parseCell.handler = ui.WrapHandler(func() {
			stableHandlerMu.Lock()
			parseCurrent := parseCell.fn
			stableHandlerMu.Unlock()
			if parseCurrent != nil {
				parseCurrent()
			}
		})
		stableHandlerCells[parseKey] = parseCell
	}
	parseCell.fn = parseFn
	stableHandlerMu.Unlock()
	return parseCell.handler
}
