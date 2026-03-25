//go:build !js || !wasm
// +build !js !wasm

package runtime

// emitBrowserPanicReport is a core package helper.
func emitBrowserPanicReport(parsePanicReport PanicReport) bool {
	_ = parsePanicReport
	return false
}
