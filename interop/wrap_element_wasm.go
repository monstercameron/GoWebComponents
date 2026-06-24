//go:build js && wasm

package interop

import "syscall/js"

// WrapElement wraps a live DOM element js.Value in the interop Element surface,
// exposing BoundingClientRect, ObserveResize, ObserveIntersection, Listen, etc.
// It is the bridge used by element-scoped UI hooks (geometry, intersection,
// pointer) that already hold a DOM node from a ref. Returns the zero Element for
// a null/undefined value.
func WrapElement(parseValue js.Value) Element {
	return elementFromJSValue(parseValue)
}
