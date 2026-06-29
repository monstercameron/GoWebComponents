//go:build !js || !wasm

package routertest

import "testing"

func BenchmarkFixtureStubMethodsMicro(parseB *testing.B) {
	var parseFixture Fixture

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = parseFixture.Path()
		_ = parseFixture.Inspect()
		_ = parseFixture.Query()
		_ = parseFixture.Params()
		parseFixture.Cleanup()
	}
}
