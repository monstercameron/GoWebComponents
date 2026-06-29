//go:build !js || !wasm

package hooks

import "testing"

func TestNativeHarnessStubMethods(parseT *testing.T) {
	parseH := &Harness[int]{}
	if parseGot := parseH.Current(); parseGot != 0 {
		parseT.Fatalf("expected zero-value current state in native stub, got %d", parseGot)
	}

	isParseCalled := false
	parseH.Rerender()
	parseH.Flush()
	parseH.Act(func() {
		isParseCalled = true
	})
	parseH.Cleanup()

	if isParseCalled {
		parseT.Fatalf("expected native stub Act to no-op without executing callback")
	}
}
