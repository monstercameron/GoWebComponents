package runtime

import (
	"strconv"
	"strings"
	"time"
)

// HydrationMetrics summarizes one hydration pass for observability hooks.
type HydrationMetrics struct {
	CorrelationID        string
	StartedAt            time.Time
	FinishedAt           time.Time
	Duration             time.Duration
	DurationNs           int64
	ExistingDOMNodeCount int
	FallbackCount        int
	MismatchCount        int
	DiscardedNodeCount   int
	Strict               bool
	Failed               bool
	Failure              string
}

// SetNextHydrationObserver registers a one-shot observer for the next hydration attempt.
func (parseRt *Runtime) SetNextHydrationObserver(parseCorrelationID string, parseNotify func(HydrationMetrics)) {
	if parseRt == nil {
		return
	}
	parseRt.nextHydrationObserver = parseNotify
	parseRt.nextHydrationCorrelationID = strings.TrimSpace(parseCorrelationID)
}

// beginHydrationMetrics is a core package helper.
func (parseRt *Runtime) beginHydrationMetrics(parseExistingChildren int, isStrict bool) {
	if parseRt == nil {
		return
	}
	parseRt.hydrationMetricsActive = true
	parseRt.hydrationMetrics = HydrationMetrics{
		CorrelationID:        parseRt.nextHydrationCorrelationID,
		StartedAt:            time.Now().UTC(),
		ExistingDOMNodeCount: parseExistingChildren,
		Strict:               isStrict,
	}
}

// recordHydrationFallback is a core package helper.
func (parseRt *Runtime) recordHydrationFallback() {
	if parseRt == nil || !parseRt.hydrationMetricsActive {
		return
	}
	parseRt.hydrationMetrics.FallbackCount++
}

// recordHydrationMismatch is a core package helper.
func (parseRt *Runtime) recordHydrationMismatch() {
	if parseRt == nil || !parseRt.hydrationMetricsActive {
		return
	}
	parseRt.hydrationMetrics.MismatchCount++
}

// recordHydrationDiscarded is a core package helper.
func (parseRt *Runtime) recordHydrationDiscarded(parseCount int) {
	if parseRt == nil || !parseRt.hydrationMetricsActive || parseCount <= 0 {
		return
	}
	parseRt.hydrationMetrics.DiscardedNodeCount += parseCount
}

// finishHydrationMetrics is a core package helper.
func (parseRt *Runtime) finishHydrationMetrics(isFailed bool, parseFailure string) {
	if parseRt == nil || !parseRt.hydrationMetricsActive {
		return
	}
	parseMetrics := parseRt.hydrationMetrics
	parseMetrics.FinishedAt = time.Now().UTC()
	parseMetrics.Duration = parseMetrics.FinishedAt.Sub(parseMetrics.StartedAt)
	parseMetrics.DurationNs = parseMetrics.Duration.Nanoseconds()
	parseMetrics.Failed = isFailed
	parseMetrics.Failure = strings.TrimSpace(parseFailure)

	parseNotify := parseRt.nextHydrationObserver
	parseRt.lastHydrationMetrics = parseMetrics
	parseRt.hydrationMetrics = HydrationMetrics{}
	parseRt.hydrationMetricsActive = false
	parseRt.nextHydrationObserver = nil
	parseRt.nextHydrationCorrelationID = ""
	parseRt.profiling.hydrationDurationNs = parseMetrics.DurationNs

	parsePhase := "finish"
	if parseMetrics.Failed {
		parsePhase = "error"
	}
	parseRt.RecordProfilingEvent(ProfilingEvent{
		Domain:        "runtime",
		Name:          "hydration",
		Phase:         parsePhase,
		Target:        "root",
		CorrelationID: parseMetrics.CorrelationID,
		DurationNs:    parseMetrics.DurationNs,
		Fields: map[string]string{
			"strict":   strconv.FormatBool(parseMetrics.Strict),
			"failed":   strconv.FormatBool(parseMetrics.Failed),
			"fallback": strconv.Itoa(parseMetrics.FallbackCount),
		},
	})

	if parseNotify != nil {
		parseNotify(parseMetrics)
	}
}
