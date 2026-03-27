//go:build !js || !wasm
// +build !js !wasm

package ui

// canParallelRegionUseRuntime2Lifecycle reports whether the current target can attach runtime2 browser lifecycle state.
func canParallelRegionUseRuntime2Lifecycle() bool {
	return false
}
