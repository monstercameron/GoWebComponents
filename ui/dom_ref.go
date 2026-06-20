package ui

import "github.com/monstercameron/GoWebComponents/internal/runtime"

// DOM element refs (G2). UseDOMRef returns a stable handle to a rendered DOM
// element. Spread it onto exactly one element with html.Ref / shorthand.Ref; the
// element's live node is available from the first effect that runs after mount.
// Before mount, after unmount, and on the native/SSR build, Node() is nil and
// Mounted() is false.
//
//	r := ui.UseDOMRef()
//	ui.UseEffect(func() func() { r.Focus(); return nil }, "")   // wasm
//	return shorthand.Input(shorthand.Ref(r))

// domRefBox is the sink stored in props. The runtime's commit phase writes the
// live node here on mount and nil on unmount; it is only ever touched from the
// single-threaded commit phase plus the owning component's effects.
type domRefBox struct {
	node runtime.DOMNode
}

// SetDOMNode implements runtime.DOMRefSink. The nil-receiver guard keeps a
// zero-value DOMRef (one not produced by UseDOMRef) safe.
func (parseB *domRefBox) SetDOMNode(parseNode runtime.DOMNode) {
	if parseB == nil {
		return
	}
	parseB.node = parseNode
}

// DOMRef is a handle to a rendered DOM element.
type DOMRef struct {
	box *domRefBox
}

// UseDOMRef creates a DOMRef that is stable across renders. Call it
// unconditionally at a stable hook position, like any other hook.
func UseDOMRef() DOMRef {
	parseBox := UseRef[*domRefBox](nil)
	if parseBox.Get() == nil {
		parseBox.Set(&domRefBox{})
	}
	return DOMRef{box: parseBox.Get()}
}

// Node returns the live DOM node, or nil before mount / after unmount / on native.
func (parseR DOMRef) Node() runtime.DOMNode {
	if parseR.box == nil {
		return nil
	}
	return parseR.box.node
}

// Mounted reports whether the ref currently points at a live DOM node.
func (parseR DOMRef) Mounted() bool {
	return !runtime.IsDOMNodeNull(parseR.Node())
}

// Sink returns the underlying runtime sink for the html.Ref PropOption. It is an
// internal wiring seam, not part of the everyday API.
func (parseR DOMRef) Sink() runtime.DOMRefSink {
	if parseR.box == nil {
		return nil
	}
	return parseR.box
}

// Focus moves keyboard focus to the referenced element if it is mounted and
// focusable. Cross-build: it routes through the node's optional Focuser
// capability (real focus on wasm, no-op on native/SSR), so callers need no build
// tags. Safe to call before mount / after unmount (no-op).
func (parseR DOMRef) Focus() {
	if parseFocuser, parseOk := parseR.Node().(runtime.Focuser); parseOk && parseFocuser != nil {
		parseFocuser.Focus()
	}
}
