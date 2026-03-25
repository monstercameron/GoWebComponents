package hotreload

import "testing"

func BenchmarkEnabled(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = Enabled()
	}
}

func BenchmarkEnableDisableCycle(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		Enable()
		Disable()
	}
}
