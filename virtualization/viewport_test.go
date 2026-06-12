package virtualization

import "testing"

func TestComputeViewportStateEmptyList(parseT *testing.T) {
	parseState, parseErr := ComputeViewportState(ViewportConfig{
		TotalItems: 0,
		RowHeight:  32,
		Overscan:   2,
	}, 0, 320)
	if parseErr != nil {
		parseT.Fatalf("ComputeViewportState() error = %v", parseErr)
	}
	if parseState.TotalHeight != 0 {
		parseT.Fatalf("TotalHeight = %v, want 0", parseState.TotalHeight)
	}
	if parseState.Visible.Len() != 0 {
		parseT.Fatalf("Visible.Len() = %d, want 0", parseState.Visible.Len())
	}
	if parseState.Rendered.Len() != 0 {
		parseT.Fatalf("Rendered.Len() = %d, want 0", parseState.Rendered.Len())
	}
}

func TestComputeViewportStateVisibleRangeRegressions(parseT *testing.T) {
	parseTests := []struct {
		name           string
		config         ViewportConfig
		scrollTop      float64
		viewportHeight float64
		wantVisible    Range
		wantRendered   Range
	}{
		{
			name: "tiny list clamps rendered range to item count",
			config: ViewportConfig{
				TotalItems: 3,
				RowHeight:  40,
				Overscan:   4,
			},
			scrollTop:      0,
			viewportHeight: 200,
			wantVisible:    Range{Start: 0, End: 3},
			wantRendered:   Range{Start: 0, End: 3},
		},
		{
			name: "start of large list keeps overscan before at zero",
			config: ViewportConfig{
				TotalItems: 1000,
				RowHeight:  24,
				Overscan:   3,
			},
			scrollTop:      0,
			viewportHeight: 96,
			wantVisible:    Range{Start: 0, End: 4},
			wantRendered:   Range{Start: 0, End: 7},
		},
		{
			name: "overscan boundary in middle of list",
			config: ViewportConfig{
				TotalItems: 50,
				RowHeight:  30,
				Overscan:   1,
			},
			scrollTop:      150,
			viewportHeight: 90,
			wantVisible:    Range{Start: 5, End: 8},
			wantRendered:   Range{Start: 4, End: 9},
		},
		{
			name: "resize driven recalculation expands visible range",
			config: ViewportConfig{
				TotalItems: 80,
				RowHeight:  20,
				Overscan:   2,
			},
			scrollTop:      200,
			viewportHeight: 200,
			wantVisible:    Range{Start: 10, End: 20},
			wantRendered:   Range{Start: 8, End: 22},
		},
		{
			name: "negative scroll is clamped to zero",
			config: ViewportConfig{
				TotalItems: 20,
				RowHeight:  25,
				Overscan:   2,
			},
			scrollTop:      -50,
			viewportHeight: 100,
			wantVisible:    Range{Start: 0, End: 4},
			wantRendered:   Range{Start: 0, End: 6},
		},
		{
			name: "out of range scroll produces empty end range",
			config: ViewportConfig{
				TotalItems: 10,
				RowHeight:  20,
			},
			scrollTop:      5000,
			viewportHeight: 100,
			wantVisible:    Range{Start: 10, End: 10},
			wantRendered:   Range{Start: 10, End: 10},
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			parseState, parseErr := ComputeViewportState(parseTest.config, parseTest.scrollTop, parseTest.viewportHeight)
			if parseErr != nil {
				parseT2.Fatalf("ComputeViewportState() error = %v", parseErr)
			}
			if parseState.Visible != parseTest.wantVisible {
				parseT2.Fatalf("Visible = %+v, want %+v", parseState.Visible, parseTest.wantVisible)
			}
			if parseState.Rendered != parseTest.wantRendered {
				parseT2.Fatalf("Rendered = %+v, want %+v", parseState.Rendered, parseTest.wantRendered)
			}
		})
	}
}

func TestComputeViewportStateUsesVisibleAndOverscanRanges(parseT *testing.T) {
	parseState, parseErr := ComputeViewportState(ViewportConfig{
		TotalItems: 100,
		RowHeight:  20,
		Overscan:   2,
	}, 45, 60)
	if parseErr != nil {
		parseT.Fatalf("ComputeViewportState() error = %v", parseErr)
	}
	if parseState.Visible != (Range{Start: 2, End: 6}) {
		parseT.Fatalf("Visible = %+v, want {Start:2 End:6}", parseState.Visible)
	}
	if parseState.Rendered != (Range{Start: 0, End: 8}) {
		parseT.Fatalf("Rendered = %+v, want {Start:0 End:8}", parseState.Rendered)
	}
}

func TestComputeViewportStateClampsAtEnd(parseT *testing.T) {
	parseState, parseErr := ComputeViewportState(ViewportConfig{
		TotalItems: 10,
		RowHeight:  50,
		Overscan:   3,
	}, 480, 120)
	if parseErr != nil {
		parseT.Fatalf("ComputeViewportState() error = %v", parseErr)
	}
	if parseState.Visible != (Range{Start: 9, End: 10}) {
		parseT.Fatalf("Visible = %+v, want {Start:9 End:10}", parseState.Visible)
	}
	if parseState.Rendered != (Range{Start: 6, End: 10}) {
		parseT.Fatalf("Rendered = %+v, want {Start:6 End:10}", parseState.Rendered)
	}
}

func TestComputeViewportStateRejectsInvalidConfig(parseT *testing.T) {
	_, parseErr := ComputeViewportState(ViewportConfig{TotalItems: -1, RowHeight: 10}, 0, 100)
	if parseErr == nil {
		parseT.Fatal("ComputeViewportState() error = nil, want invalid config error")
	}
	_, parseErr = ComputeViewportState(ViewportConfig{TotalItems: 1, RowHeight: 0}, 0, 100)
	if parseErr == nil {
		parseT.Fatal("ComputeViewportState() error = nil, want invalid row height error")
	}
	_, parseErr = ComputeViewportState(ViewportConfig{TotalItems: 1, RowHeight: 10, Overscan: -1}, 0, 100)
	if parseErr == nil {
		parseT.Fatal("ComputeViewportState() error = nil, want invalid overscan error")
	}
}

func TestComputeViewportStateClampsNegativeViewportHeightToZero(parseT *testing.T) {
	parseState, parseErr := ComputeViewportState(ViewportConfig{
		TotalItems: 10,
		RowHeight:  20,
		Overscan:   2,
	}, 40, -100)
	if parseErr != nil {
		parseT.Fatalf("ComputeViewportState() error = %v", parseErr)
	}
	if parseState.ViewportHeight != 0 {
		parseT.Fatalf("expected viewport height to clamp at zero, got %v", parseState.ViewportHeight)
	}
	if parseState.Visible.Len() != 0 || parseState.Rendered.Len() != 0 {
		parseT.Fatalf("expected no visible/rendered ranges when viewport height is zero, got visible=%+v rendered=%+v", parseState.Visible, parseState.Rendered)
	}
}

func TestViewportStateDiagnostics(parseT *testing.T) {
	parseState := ViewportState{
		ScrollTop:      40,
		ViewportHeight: 120,
		TotalItems:     100,
		TotalHeight:    2000,
		RowHeight:      20,
		Overscan:       2,
		Visible:        Range{Start: 2, End: 8},
		Rendered:       Range{Start: 0, End: 10},
	}
	parseDiagnostics := parseState.Diagnostics()
	if parseDiagnostics.VisibleCount != 6 {
		parseT.Fatalf("VisibleCount = %d, want 6", parseDiagnostics.VisibleCount)
	}
	if parseDiagnostics.RenderedCount != 10 {
		parseT.Fatalf("RenderedCount = %d, want 10", parseDiagnostics.RenderedCount)
	}
	if parseDiagnostics.OverscanBeforeCount != 2 || parseDiagnostics.OverscanAfterCount != 2 {
		parseT.Fatalf("overscan counts = (%d, %d), want (2, 2)", parseDiagnostics.OverscanBeforeCount, parseDiagnostics.OverscanAfterCount)
	}
	if parseDiagnostics.MeasurementCount != 0 || parseDiagnostics.InvalidationCount != 0 || parseDiagnostics.ScrollCorrectionCount != 0 {
		parseT.Fatalf("fixed-height diagnostics should report zero measurement churn, got %+v", parseDiagnostics)
	}
	if parseDiagnostics.RowMountCount != 0 || parseDiagnostics.RowUnmountCount != 0 {
		parseT.Fatalf("default diagnostics should report zero row churn, got %+v", parseDiagnostics)
	}
}

func TestViewportDiagnosticsWithRowLifecycle(parseT *testing.T) {
	parseDiagnostics := (ViewportState{}).Diagnostics().WithRowLifecycle(7, 5)
	if parseDiagnostics.RowMountCount != 7 || parseDiagnostics.RowUnmountCount != 5 {
		parseT.Fatalf("row lifecycle counts = (%d, %d), want (7, 5)", parseDiagnostics.RowMountCount, parseDiagnostics.RowUnmountCount)
	}
}

func TestViewportDiagnosticsClampsNegativeOverscanGaps(parseT *testing.T) {
	parseDiagnostics := (ViewportState{
		Visible:  Range{Start: 5, End: 7},
		Rendered: Range{Start: 6, End: 6},
	}).Diagnostics()
	if parseDiagnostics.OverscanBeforeCount != 0 || parseDiagnostics.OverscanAfterCount != 0 {
		parseT.Fatalf("expected negative overscan gap values to clamp at zero, got before=%d after=%d", parseDiagnostics.OverscanBeforeCount, parseDiagnostics.OverscanAfterCount)
	}
}
