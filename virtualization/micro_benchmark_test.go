package virtualization

import "testing"

func BenchmarkComputeViewportStateMicro(b *testing.B) {
	config := ViewportConfig{
		TotalItems: 20000,
		RowHeight:  32,
		Overscan:   4,
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		scrollTop := float64((i % 6000) * 12)
		_, err := ComputeViewportState(config, scrollTop, 640)
		if err != nil {
			b.Fatal(err)
		}
	}
}
