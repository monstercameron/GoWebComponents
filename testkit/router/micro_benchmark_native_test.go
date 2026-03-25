//go:build !js || !wasm
// +build !js !wasm

package routertest

import "testing"

func BenchmarkFixtureStubMethodsMicro(b *testing.B) {
	var fixture Fixture

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fixture.Path()
		_ = fixture.Inspect()
		_ = fixture.Query()
		_ = fixture.Params()
		fixture.Cleanup()
	}
}
