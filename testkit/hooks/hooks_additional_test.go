//go:build !js || !wasm
// +build !js !wasm

package hooks

import "testing"

// TestNativeHarnessNoOpMethods verifies the host-build hook harness stubs stay safe to call.
func TestNativeHarnessNoOpMethods(parseT *testing.T) {
	var parseHarness Harness[int]
	if parseHarness.Current() != 0 {
		parseT.Fatalf("expected zero-value current hook state, got %d", parseHarness.Current())
	}

	isParseCalled := false
	parseHarness.Rerender()
	parseHarness.Flush()
	parseHarness.Act(func() {
		isParseCalled = true
	})
	parseHarness.Cleanup()

	if isParseCalled {
		parseT.Fatal("expected native Act stub to remain a no-op")
	}
}
