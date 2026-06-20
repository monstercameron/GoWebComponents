//go:build !(js && wasm)

package ui

// defaultBindGlobalEvent is a no-op on the native/SSR build (there is no
// document/window to bind to). Returning nil means UseEffect has no cleanup.
func defaultBindGlobalEvent(parseScope globalScope, parseEventType string, parseHandler func(Event)) func() {
	return nil
}
