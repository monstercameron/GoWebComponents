//go:build !js || !wasm

package routertest

import "testing"

func BenchmarkFixtureStubMethodsMicro(parseB *testing.B) {
	var parseFixture Fixture

	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = parseFixture.Path()
		_ = parseFixture.Inspect()
		_ = parseFixture.Query()
		_ = parseFixture.Params()
		parseFixture.Cleanup()
	}
}
