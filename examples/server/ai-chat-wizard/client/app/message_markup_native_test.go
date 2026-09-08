//go:build !js || !wasm

package app

import "testing"

// setMessageMarkupTestRuntime keeps native SSR independent of a browser adapter.
func setMessageMarkupTestRuntime(parseT *testing.T) {
	parseT.Helper()
}
