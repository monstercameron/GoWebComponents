//go:build production && (!js || !wasm)

package runtime

// Keep inspection semantics available to production-tagged native/SSR builds;
// only the browser renderer omits this development metadata.
const hookSignatureRecordingEnabled = true
