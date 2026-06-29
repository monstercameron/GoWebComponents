//go:build !(js && wasm)

package ui

import "github.com/monstercameron/GoWebComponents/v4/interop"

// Element-scoped hooks are no-ops on non-browser builds (no live DOM). The
// reactive implementations live in the js/wasm build.

// UseElementGeometry returns the zero Rect on native/SSR.
func UseElementGeometry(parseRef DOMRef) interop.Rect { return interop.Rect{} }

// UseIntersection returns false on native/SSR.
func UseIntersection(parseRef DOMRef, parseOptions ...interop.IntersectionObserverOptions) bool {
	return false
}

// UseAnimationRestart is a no-op on native/SSR.
func UseAnimationRestart(parseRef DOMRef, parseClassName string, parseDeps ...any) {}

// UsePointerEvents is a no-op on native/SSR.
func UsePointerEvents(parseRef DOMRef, parseHandlers PointerHandlers) {}

// UseWheel is a no-op on native/SSR.
func UseWheel(parseRef DOMRef, parseHandler func(Event)) {}
