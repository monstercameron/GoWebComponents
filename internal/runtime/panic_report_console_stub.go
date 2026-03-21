//go:build !js || !wasm
// +build !js !wasm

package runtime

func emitBrowserPanicReport(report PanicReport) bool {
	_ = report
	return false
}
