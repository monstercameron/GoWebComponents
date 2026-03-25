package virtualization

import "testing"

func BenchmarkComputeViewportState(b *testing.B) {
	b.ReportAllocs()
	config := ViewportConfig{
		TotalItems: 10000,
		RowHeight:  28,
		Overscan:   8,
	}
	for b.Loop() {
		state, err := ComputeViewportState(config, 10240, 900)
		if err != nil {
			b.Fatalf("ComputeViewportState: %v", err)
		}
		if state.Rendered.Len() == 0 {
			b.Fatal("ComputeViewportState returned empty rendered range")
		}
	}
}

func BenchmarkViewportDiagnosticsWithRowLifecycle(b *testing.B) {
	b.ReportAllocs()
	state, err := ComputeViewportState(ViewportConfig{
		TotalItems: 5000,
		RowHeight:  32,
		Overscan:   4,
	}, 4096, 768)
	if err != nil {
		b.Fatalf("ComputeViewportState: %v", err)
	}
	for b.Loop() {
		diagnostics := state.Diagnostics().WithRowLifecycle(12, 9)
		if diagnostics.VisibleCount == 0 {
			b.Fatal("Diagnostics returned zero visible rows")
		}
	}
}
