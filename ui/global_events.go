package ui

// Global (document/window) event hooks (G9).
//
// Subscribing to document/window events from a component previously dropped to
// raw syscall/js addEventListener with a hand-managed (often leaked) js.Func.
// These hooks own that lifetime: the listener is attached on mount and removed —
// and its js.Func released — on unmount or when the deps change, via the effect
// cleanup. This is the managed answer to the "global shortcuts / outside-click /
// idle-activity" patterns (G9, G13, G25).
//
//	ui.UseGlobalKey(func(e ui.KeyboardEvent) {
//	    if e.GetKey() == "Escape" { onClose() }
//	})
//
// Handlers fire only while the owning component is mounted. On the native/SSR
// build there is no DOM, so binding is a safe no-op.

type globalScope uint8

const (
	scopeDocument globalScope = iota
	scopeWindow
)

// bindGlobalEvent is the platform seam: attach handler to the global target and
// return an unbind func (nil if nothing was bound). It is a package var so tests
// can substitute a recorder; production uses the build-specific default.
var bindGlobalEvent = defaultBindGlobalEvent

// UseDocumentEvent subscribes the calling component to a document-level event for
// as long as it is mounted.
func UseDocumentEvent(eventType string, handler func(Event), deps ...any) {
	useGlobalEvent(scopeDocument, eventType, handler, deps...)
}

// UseWindowEvent subscribes the calling component to a window-level event (resize,
// scroll, hashchange, …) for as long as it is mounted.
func UseWindowEvent(eventType string, handler func(Event), deps ...any) {
	useGlobalEvent(scopeWindow, eventType, handler, deps...)
}

// UseGlobalKey subscribes to document keydown — app-wide keyboard shortcuts. To
// ignore keystrokes while the user is typing, check the event target in the
// handler (e.g. skip when document.activeElement is an input/textarea).
func UseGlobalKey(handler func(KeyboardEvent), deps ...any) {
	UseDocumentEvent("keydown", handler, deps...)
}

func useGlobalEvent(scope globalScope, eventType string, handler func(Event), deps ...any) {
	UseEffect(func() func() {
		if handler == nil {
			return nil
		}
		return bindGlobalEvent(scope, eventType, handler)
	}, globalEventDeps(scope, eventType, deps)...)
}

// globalEventDeps returns a stable, non-empty dep set when the caller passes none
// — so the listener binds once on mount. (UseEffect with zero deps runs every
// render, which would re-attach and leak a listener per render.) Explicit deps
// are passed through so the listener re-binds when they change.
func globalEventDeps(scope globalScope, eventType string, deps []any) []any {
	if len(deps) == 0 {
		return []any{globalEventSentinel, scope, eventType}
	}
	return deps
}

const globalEventSentinel = "gwc-global-event"
