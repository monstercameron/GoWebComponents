//go:build js && wasm

package css

import "syscall/js"

// injectStyleElement (wasm) creates a <style id="..."> element in <head> exactly
// once. If an element with the id already exists (this call or SSR), it is a
// no-op, making Inject idempotent by id. A no-op when no document is present
// (worker / node test host) so it degrades gracefully.
func injectStyleElement(parseID string, parseCSS string) {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return
	}
	if existing := document.Call("getElementById", parseID); existing.Truthy() {
		return
	}
	style := document.Call("createElement", "style")
	style.Set("id", parseID)
	style.Set("textContent", parseCSS)
	head := document.Get("head")
	if !head.Truthy() {
		head = document.Get("documentElement")
	}
	if head.Truthy() {
		head.Call("appendChild", style)
	}
}
