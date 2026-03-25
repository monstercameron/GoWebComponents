package virtualization

import (
	"errors"
	"math"
)

// Range describes an index window with a start-inclusive, end-exclusive shape.
type Range struct {
	Start int
	End   int
}

// Len reports the number of items in the range.
func (r Range) Len() int {
	if r.End <= r.Start {
		return 0
	}
	return r.End - r.Start
}

// ViewportConfig describes the fixed-height list contract used by the first
// virtualization pass.
type ViewportConfig struct {
	TotalItems int
	RowHeight  float64
	Overscan   int
}

// ViewportState describes the current scroll offset, viewport height, and the
// visible plus rendered ranges for one fixed-height list.
type ViewportState struct {
	ScrollTop      float64
	ViewportHeight float64
	TotalItems     int
	TotalHeight    float64
	RowHeight      float64
	Overscan       int
	Visible        Range
	Rendered       Range
}

// ViewportDiagnostics exposes the current rendered and visible window in a
// shape that is convenient for examples, debugging, and future devtools.
type ViewportDiagnostics struct {
	ScrollTop             float64
	ViewportHeight        float64
	TotalItems            int
	TotalHeight           float64
	RowHeight             float64
	Overscan              int
	VisibleStart          int
	VisibleEnd            int
	VisibleCount          int
	RenderedStart         int
	RenderedEnd           int
	RenderedCount         int
	OverscanBeforeCount   int
	OverscanAfterCount    int
	MeasurementCount      int
	InvalidationCount     int
	ScrollCorrectionCount int
	RowMountCount         int
	RowUnmountCount       int
}

// ComputeViewportState converts scroll metrics into visible and rendered ranges
// for a fixed-height virtualized list.
func ComputeViewportState(config ViewportConfig, scrollTop, viewportHeight float64) (ViewportState, error) {
	normalized, err := normalizeConfig(config)
	if err != nil {
		return ViewportState{}, err
	}
	if scrollTop < 0 {
		scrollTop = 0
	}
	if viewportHeight < 0 {
		viewportHeight = 0
	}

	state := ViewportState{
		ScrollTop:      scrollTop,
		ViewportHeight: viewportHeight,
		TotalItems:     normalized.TotalItems,
		TotalHeight:    float64(normalized.TotalItems) * normalized.RowHeight,
		RowHeight:      normalized.RowHeight,
		Overscan:       normalized.Overscan,
	}
	if normalized.TotalItems == 0 || viewportHeight == 0 {
		return state, nil
	}

	visibleStart := clampIndex(int(math.Floor(scrollTop/normalized.RowHeight)), normalized.TotalItems)
	visibleEnd := clampIndex(int(math.Ceil((scrollTop+viewportHeight)/normalized.RowHeight)), normalized.TotalItems)
	if visibleEnd < visibleStart {
		visibleEnd = visibleStart
	}

	renderedStart := clampIndex(visibleStart-normalized.Overscan, normalized.TotalItems)
	renderedEnd := clampIndex(visibleEnd+normalized.Overscan, normalized.TotalItems)
	if renderedEnd < renderedStart {
		renderedEnd = renderedStart
	}

	state.Visible = Range{Start: visibleStart, End: visibleEnd}
	state.Rendered = Range{Start: renderedStart, End: renderedEnd}
	return state, nil
}

// Diagnostics converts a viewport state into an inspection-friendly summary.
func (s ViewportState) Diagnostics() ViewportDiagnostics {
	before := s.Visible.Start - s.Rendered.Start
	if before < 0 {
		before = 0
	}
	after := s.Rendered.End - s.Visible.End
	if after < 0 {
		after = 0
	}
	return ViewportDiagnostics{
		ScrollTop:             s.ScrollTop,
		ViewportHeight:        s.ViewportHeight,
		TotalItems:            s.TotalItems,
		TotalHeight:           s.TotalHeight,
		RowHeight:             s.RowHeight,
		Overscan:              s.Overscan,
		VisibleStart:          s.Visible.Start,
		VisibleEnd:            s.Visible.End,
		VisibleCount:          s.Visible.Len(),
		RenderedStart:         s.Rendered.Start,
		RenderedEnd:           s.Rendered.End,
		RenderedCount:         s.Rendered.Len(),
		OverscanBeforeCount:   before,
		OverscanAfterCount:    after,
		MeasurementCount:      0,
		InvalidationCount:     0,
		ScrollCorrectionCount: 0,
	}
}

// WithRowLifecycle annotates a diagnostics snapshot with row mount and unmount
// counters gathered by the list primitive.
func (d ViewportDiagnostics) WithRowLifecycle(mounts, unmounts int) ViewportDiagnostics {
	d.RowMountCount = mounts
	d.RowUnmountCount = unmounts
	return d
}

func normalizeConfig(config ViewportConfig) (ViewportConfig, error) {
	if config.TotalItems < 0 {
		return ViewportConfig{}, errors.New("virtualization: total items must be >= 0")
	}
	if config.RowHeight <= 0 {
		return ViewportConfig{}, errors.New("virtualization: row height must be > 0")
	}
	if config.Overscan < 0 {
		return ViewportConfig{}, errors.New("virtualization: overscan must be >= 0")
	}
	return config, nil
}

func clampIndex(value, total int) int {
	if value < 0 {
		return 0
	}
	if value > total {
		return total
	}
	return value
}
