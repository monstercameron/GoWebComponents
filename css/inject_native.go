//go:build !(js && wasm)

package css

// injectStyleElement (native) routes injected CSS into the buffer sink, keyed by
// id, so SSR (StyleBlock/Harvest) and native tests can read it back. The registry
// dedup makes it idempotent per id and a different body under the same id is ignored without a diagnostic.
func injectStyleElement(parseID string, parseCSS string) {
	registerInjectedOnce("inject:"+parseID, parseCSS)
}
