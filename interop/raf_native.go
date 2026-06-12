//go:build !js || !wasm

package interop

// RequestAnimationFrame is a non-browser stub. On native builds there is no
// display rendering pipeline, so the callback is never invoked. The returned
// cancel func is a no-op but is always non-nil so callers can call it
// unconditionally without a nil check.
//
// Browser (GOOS=js GOARCH=wasm) builds use the real requestAnimationFrame API
// instead; see raf_wasm.go.
func RequestAnimationFrame(parseCallback func(parseTimestampMillis float64)) (parseCancel func()) {
	_ = parseCallback
	return func() {}
}
