//go:build js && wasm

package hooks

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v7/ui"
)

func BenchmarkRenderHookCurrentMicroWasm(parseB *testing.B) {
	parseHarness := RenderHook(parseB, func() int {
		return 42
	})
	defer parseHarness.Cleanup()

	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = parseHarness.Current()
	}
}

func benchStateRerenderHook() int {
	parseCounter := ui.UseState(0)
	return parseCounter.Get()
}

func BenchmarkRenderHookStateRerenderMicroWasm(parseB *testing.B) {
	parseHarness := RenderHook(parseB, benchStateRerenderHook)
	defer parseHarness.Cleanup()

	parseB.ReportAllocs()
	for parseB.Loop() {
		parseHarness.Rerender()
	}
}
