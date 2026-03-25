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
func (rt *Runtime) SetNextHydrationObserver(correlationID string, notify func(HydrationMetrics)) {
	if rt == nil {
		return
	}
	rt.nextHydrationObserver = notify
	rt.nextHydrationCorrelationID = strings.TrimSpace(correlationID)
}

func (rt *Runtime) beginHydrationMetrics(existingChildren int, strict bool) {
	if rt == nil {
		return
	}
	rt.hydrationMetricsActive = true
	rt.hydrationMetrics = HydrationMetrics{
		CorrelationID:        rt.nextHydrationCorrelationID,
		StartedAt:            time.Now().UTC(),
		ExistingDOMNodeCount: existingChildren,
		Strict:               strict,
	}
}

func (rt *Runtime) recordHydrationFallback() {
	if rt == nil || !rt.hydrationMetricsActive {
		return
	}
	rt.hydrationMetrics.FallbackCount++
}

func (rt *Runtime) recordHydrationMismatch() {
	if rt == nil || !rt.hydrationMetricsActive {
		return
	}
	rt.hydrationMetrics.MismatchCount++
}

func (rt *Runtime) recordHydrationDiscarded(count int) {
	if rt == nil || !rt.hydrationMetricsActive || count <= 0 {
		return
	}
	rt.hydrationMetrics.DiscardedNodeCount += count
}

func (rt *Runtime) finishHydrationMetrics(failed bool, failure string) {
	if rt == nil || !rt.hydrationMetricsActive {
		return
	}
	metrics := rt.hydrationMetrics
	metrics.FinishedAt = time.Now().UTC()
	metrics.Duration = metrics.FinishedAt.Sub(metrics.StartedAt)
	metrics.DurationNs = metrics.Duration.Nanoseconds()
	metrics.Failed = failed
	metrics.Failure = strings.TrimSpace(failure)

	notify := rt.nextHydrationObserver
	rt.hydrationMetrics = HydrationMetrics{}
	rt.hydrationMetricsActive = false
	rt.nextHydrationObserver = nil
	rt.nextHydrationCorrelationID = ""
	rt.profiling.hydrationDurationNs = metrics.DurationNs

	phase := "finish"
	if metrics.Failed {
		phase = "error"
	}
	rt.RecordProfilingEvent(ProfilingEvent{
		Domain:        "runtime",
		Name:          "hydration",
		Phase:         phase,
		Target:        "root",
		CorrelationID: metrics.CorrelationID,
		DurationNs:    metrics.DurationNs,
		Fields: map[string]string{
			"strict":   strconv.FormatBool(metrics.Strict),
			"failed":   strconv.FormatBool(metrics.Failed),
			"fallback": strconv.Itoa(metrics.FallbackCount),
		},
	})

	if notify != nil {
		notify(metrics)
	}
}
