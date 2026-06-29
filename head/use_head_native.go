//go:build !js || !wasm

package head

// UseHead is a no-op on native/SSR builds: there is no live document to mutate. Server-side head
// output is produced by RenderToString. The wasm build (use_head_wasm.go) applies the Document to
// the live document head on the client.
func UseHead(parseDoc Document) {}
