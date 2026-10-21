//go:build !js || !wasm

package hooks

import "testing"

func BenchmarkHarnessStubMethodsMicro(parseB *testing.B) {
	var parseHarness Harness[int]

	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = parseHarness.Current()
		parseHarness.Rerender()
		parseHarness.Flush()
		parseHarness.Act(func() {})
		parseHarness.Cleanup()
	}
}
