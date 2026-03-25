package hotreload

import "testing"

func BenchmarkEnableDisableCycleMicro(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Enable()
		Disable()
		_ = IsEnabled()
	}
}
