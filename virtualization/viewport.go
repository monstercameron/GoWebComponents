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
func (parseR Range) Len() int {
	if parseR.End <= parseR.Start {
		return 0
	}
	return parseR.End - parseR.Start
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
func ComputeViewportState(parseConfig ViewportConfig, parseScrollTop, parseViewportHeight float64) (ViewportState, error) {
	parseNormalized, parseErr := normalizeConfig(parseConfig)
	if parseErr != nil {
		return ViewportState{}, parseErr
	}
	// NaN and negative offsets mean "top"; +Inf is capped below by the index clamp.
	// The negated comparisons are what catch NaN, which fails every ordering test.
	if !(parseScrollTop > 0) {
		parseScrollTop = 0
	}
	if !(parseViewportHeight > 0) {
		parseViewportHeight = 0
	}

	parseTotalHeight := float64(parseNormalized.TotalItems) * parseNormalized.RowHeight

	parseState := ViewportState{
		ScrollTop:      parseScrollTop,
		ViewportHeight: parseViewportHeight,
		TotalItems:     parseNormalized.TotalItems,
		TotalHeight:    parseTotalHeight,
		RowHeight:      parseNormalized.RowHeight,
		Overscan:       parseNormalized.Overscan,
	}
	if parseNormalized.TotalItems == 0 || parseViewportHeight == 0 {
		return parseState, nil
	}

	parseVisibleStart := clampFloatIndex(math.Floor(parseScrollTop/parseNormalized.RowHeight), parseNormalized.TotalItems)
	parseVisibleEnd := max(clampFloatIndex(math.Ceil((parseScrollTop+parseViewportHeight)/parseNormalized.RowHeight), parseNormalized.TotalItems), parseVisibleStart)

	// Overscan beyond the list size adds nothing, and an unclamped huge value would
	// overflow the int addition below and wrap the rendered window to empty.
	parseOverscan := min(parseNormalized.Overscan, parseNormalized.TotalItems)
	parseRenderedStart := clampIndex(parseVisibleStart-parseOverscan, parseNormalized.TotalItems)
	parseRenderedEnd := max(clampIndex(parseVisibleEnd+parseOverscan, parseNormalized.TotalItems), parseRenderedStart)

	parseState.Visible = Range{Start: parseVisibleStart, End: parseVisibleEnd}
	parseState.Rendered = Range{Start: parseRenderedStart, End: parseRenderedEnd}
	return parseState, nil
}

// Diagnostics converts a viewport state into an inspection-friendly summary.
func (parseS ViewportState) Diagnostics() ViewportDiagnostics {
	parseBefore := max(parseS.Visible.Start-parseS.Rendered.Start, 0)
	parseAfter := max(parseS.Rendered.End-parseS.Visible.End, 0)
	return ViewportDiagnostics{
		ScrollTop:             parseS.ScrollTop,
		ViewportHeight:        parseS.ViewportHeight,
		TotalItems:            parseS.TotalItems,
		TotalHeight:           parseS.TotalHeight,
		RowHeight:             parseS.RowHeight,
		Overscan:              parseS.Overscan,
		VisibleStart:          parseS.Visible.Start,
		VisibleEnd:            parseS.Visible.End,
		VisibleCount:          parseS.Visible.Len(),
		RenderedStart:         parseS.Rendered.Start,
		RenderedEnd:           parseS.Rendered.End,
		RenderedCount:         parseS.Rendered.Len(),
		OverscanBeforeCount:   parseBefore,
		OverscanAfterCount:    parseAfter,
		MeasurementCount:      0,
		InvalidationCount:     0,
		ScrollCorrectionCount: 0,
	}
}

// WithRowLifecycle annotates a diagnostics snapshot with row mount and unmount
// counters gathered by the list primitive.
func (parseD ViewportDiagnostics) WithRowLifecycle(parseMounts, parseUnmounts int) ViewportDiagnostics {
	parseD.RowMountCount = parseMounts
	parseD.RowUnmountCount = parseUnmounts
	return parseD
}

func normalizeConfig(parseConfig ViewportConfig) (ViewportConfig, error) {
	if parseConfig.TotalItems < 0 {
		return ViewportConfig{}, errors.New("virtualization: total items must be >= 0")
	}
	if parseConfig.RowHeight <= 0 {
		return ViewportConfig{}, errors.New("virtualization: row height must be > 0")
	}
	if parseConfig.Overscan < 0 {
		return ViewportConfig{}, errors.New("virtualization: overscan must be >= 0")
	}
	return parseConfig, nil
}

// clampFloatIndex clamps a float row index into [0, parseTotal] BEFORE converting
// to int, so +Inf or a value beyond the int range cannot hit the
// implementation-defined float-to-int conversion.
func clampFloatIndex(parseValue float64, parseTotal int) int {
	if !(parseValue > 0) {
		return 0
	}
	if parseValue >= float64(parseTotal) {
		return parseTotal
	}
	return int(parseValue)
}

func clampIndex(parseValue, parseTotal int) int {
	if parseValue < 0 {
		return 0
	}
	if parseValue > parseTotal {
		return parseTotal
	}
	return parseValue
}
