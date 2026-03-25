package virtualization

import "testing"

func TestComputeViewportStateEmptyList(t *testing.T) {
	state, err := ComputeViewportState(ViewportConfig{
		TotalItems: 0,
		RowHeight:  32,
		Overscan:   2,
	}, 0, 320)
	if err != nil {
		t.Fatalf("ComputeViewportState() error = %v", err)
	}
	if state.TotalHeight != 0 {
		t.Fatalf("TotalHeight = %v, want 0", state.TotalHeight)
	}
	if state.Visible.Len() != 0 {
		t.Fatalf("Visible.Len() = %d, want 0", state.Visible.Len())
	}
	if state.Rendered.Len() != 0 {
		t.Fatalf("Rendered.Len() = %d, want 0", state.Rendered.Len())
	}
}

func TestComputeViewportStateVisibleRangeRegressions(t *testing.T) {
	tests := []struct {
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, err := ComputeViewportState(test.config, test.scrollTop, test.viewportHeight)
			if err != nil {
				t.Fatalf("ComputeViewportState() error = %v", err)
			}
			if state.Visible != test.wantVisible {
				t.Fatalf("Visible = %+v, want %+v", state.Visible, test.wantVisible)
			}
			if state.Rendered != test.wantRendered {
				t.Fatalf("Rendered = %+v, want %+v", state.Rendered, test.wantRendered)
			}
		})
	}
}

func TestComputeViewportStateUsesVisibleAndOverscanRanges(t *testing.T) {
	state, err := ComputeViewportState(ViewportConfig{
		TotalItems: 100,
		RowHeight:  20,
		Overscan:   2,
	}, 45, 60)
	if err != nil {
		t.Fatalf("ComputeViewportState() error = %v", err)
	}
	if state.Visible != (Range{Start: 2, End: 6}) {
		t.Fatalf("Visible = %+v, want {Start:2 End:6}", state.Visible)
	}
	if state.Rendered != (Range{Start: 0, End: 8}) {
		t.Fatalf("Rendered = %+v, want {Start:0 End:8}", state.Rendered)
	}
}

func TestComputeViewportStateClampsAtEnd(t *testing.T) {
	state, err := ComputeViewportState(ViewportConfig{
		TotalItems: 10,
		RowHeight:  50,
		Overscan:   3,
	}, 480, 120)
	if err != nil {
		t.Fatalf("ComputeViewportState() error = %v", err)
	}
	if state.Visible != (Range{Start: 9, End: 10}) {
		t.Fatalf("Visible = %+v, want {Start:9 End:10}", state.Visible)
	}
	if state.Rendered != (Range{Start: 6, End: 10}) {
		t.Fatalf("Rendered = %+v, want {Start:6 End:10}", state.Rendered)
	}
}

func TestComputeViewportStateRejectsInvalidConfig(t *testing.T) {
	_, err := ComputeViewportState(ViewportConfig{TotalItems: -1, RowHeight: 10}, 0, 100)
	if err == nil {
		t.Fatal("ComputeViewportState() error = nil, want invalid config error")
	}
	_, err = ComputeViewportState(ViewportConfig{TotalItems: 1, RowHeight: 0}, 0, 100)
	if err == nil {
		t.Fatal("ComputeViewportState() error = nil, want invalid row height error")
	}
	_, err = ComputeViewportState(ViewportConfig{TotalItems: 1, RowHeight: 10, Overscan: -1}, 0, 100)
	if err == nil {
		t.Fatal("ComputeViewportState() error = nil, want invalid overscan error")
	}
}

func TestComputeViewportStateClampsNegativeViewportHeightToZero(t *testing.T) {
	state, err := ComputeViewportState(ViewportConfig{
		TotalItems: 10,
		RowHeight:  20,
		Overscan:   2,
	}, 40, -100)
	if err != nil {
		t.Fatalf("ComputeViewportState() error = %v", err)
	}
	if state.ViewportHeight != 0 {
		t.Fatalf("expected viewport height to clamp at zero, got %v", state.ViewportHeight)
	}
	if state.Visible.Len() != 0 || state.Rendered.Len() != 0 {
		t.Fatalf("expected no visible/rendered ranges when viewport height is zero, got visible=%+v rendered=%+v", state.Visible, state.Rendered)
	}
}

func TestViewportStateDiagnostics(t *testing.T) {
	state := ViewportState{
		ScrollTop:      40,
		ViewportHeight: 120,
		TotalItems:     100,
		TotalHeight:    2000,
		RowHeight:      20,
		Overscan:       2,
		Visible:        Range{Start: 2, End: 8},
		Rendered:       Range{Start: 0, End: 10},
	}
	diagnostics := state.Diagnostics()
	if diagnostics.VisibleCount != 6 {
		t.Fatalf("VisibleCount = %d, want 6", diagnostics.VisibleCount)
	}
	if diagnostics.RenderedCount != 10 {
		t.Fatalf("RenderedCount = %d, want 10", diagnostics.RenderedCount)
	}
	if diagnostics.OverscanBeforeCount != 2 || diagnostics.OverscanAfterCount != 2 {
		t.Fatalf("overscan counts = (%d, %d), want (2, 2)", diagnostics.OverscanBeforeCount, diagnostics.OverscanAfterCount)
	}
	if diagnostics.MeasurementCount != 0 || diagnostics.InvalidationCount != 0 || diagnostics.ScrollCorrectionCount != 0 {
		t.Fatalf("fixed-height diagnostics should report zero measurement churn, got %+v", diagnostics)
	}
	if diagnostics.RowMountCount != 0 || diagnostics.RowUnmountCount != 0 {
		t.Fatalf("default diagnostics should report zero row churn, got %+v", diagnostics)
	}
}

func TestViewportDiagnosticsWithRowLifecycle(t *testing.T) {
	diagnostics := (ViewportState{}).Diagnostics().WithRowLifecycle(7, 5)
	if diagnostics.RowMountCount != 7 || diagnostics.RowUnmountCount != 5 {
		t.Fatalf("row lifecycle counts = (%d, %d), want (7, 5)", diagnostics.RowMountCount, diagnostics.RowUnmountCount)
	}
}

func TestViewportDiagnosticsClampsNegativeOverscanGaps(t *testing.T) {
	diagnostics := (ViewportState{
		Visible:  Range{Start: 5, End: 7},
		Rendered: Range{Start: 6, End: 6},
	}).Diagnostics()
	if diagnostics.OverscanBeforeCount != 0 || diagnostics.OverscanAfterCount != 0 {
		t.Fatalf("expected negative overscan gap values to clamp at zero, got before=%d after=%d", diagnostics.OverscanBeforeCount, diagnostics.OverscanAfterCount)
	}
}
