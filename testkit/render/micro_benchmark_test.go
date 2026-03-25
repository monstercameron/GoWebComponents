package render

import "testing"

func BenchmarkNewResourceControllerMicro(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NewResourceController[int]()
	}
}
