//go:build !js || !wasm
// +build !js !wasm

package hooks

import "testing"

func BenchmarkHarnessStubMethodsMicro(b *testing.B) {
	var harness Harness[int]

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = harness.Current()
		harness.Rerender()
		harness.Flush()
		harness.Act(func() {})
		harness.Cleanup()
	}
}
