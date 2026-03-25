//go:build js && wasm
// +build js,wasm

package hooks

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func BenchmarkRenderHookCurrentMicroWasm(b *testing.B) {
	harness := RenderHook(b, func() int {
		return 42
	})
	defer harness.Cleanup()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = harness.Current()
	}
}

func BenchmarkRenderHookStateRerenderMicroWasm(b *testing.B) {
	harness := RenderHook(b, func() int {
		counter := ui.UseState(0)
		return counter.Get()
	})
	defer harness.Cleanup()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		harness.Rerender()
	}
}
