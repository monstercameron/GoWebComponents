package virtualization

import "testing"

func BenchmarkComputeViewportState(parseB *testing.B) {
	parseB.ReportAllocs()
	parseConfig := ViewportConfig{
		TotalItems: 10000,
		RowHeight:  28,
		Overscan:   8,
	}
	for parseB.Loop() {
		parseState, parseErr := ComputeViewportState(parseConfig, 10240, 900)
		if parseErr != nil {
			parseB.Fatalf("ComputeViewportState: %v", parseErr)
		}
		if parseState.Rendered.Len() == 0 {
			parseB.Fatal("ComputeViewportState returned empty rendered range")
		}
	}
}

func BenchmarkViewportDiagnosticsWithRowLifecycle(parseB *testing.B) {
	parseB.ReportAllocs()
	parseState, parseErr := ComputeViewportState(ViewportConfig{
		TotalItems: 5000,
		RowHeight:  32,
		Overscan:   4,
	}, 4096, 768)
	if parseErr != nil {
		parseB.Fatalf("ComputeViewportState: %v", parseErr)
	}
	for parseB.Loop() {
		parseDiagnostics := parseState.Diagnostics().WithRowLifecycle(12, 9)
		if parseDiagnostics.VisibleCount == 0 {
			parseB.Fatal("Diagnostics returned zero visible rows")
		}
	}
}
