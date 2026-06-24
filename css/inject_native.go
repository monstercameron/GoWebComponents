//go:build !(js && wasm)

package css

// injectStyleElement (native) routes injected CSS into the buffer sink, keyed by
// id, so SSR (StyleBlock/Harvest) and native tests can read it back. The registry
// dedup makes it idempotent per id (first call wins).
func injectStyleElement(parseID string, parseCSS string) {
	registerAndEmit("inject:"+parseID, parseCSS)
}
