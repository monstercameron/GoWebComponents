//go:build !js || !wasm

package otel

import (
	"context"
	"math"
	"net/http"
	"runtime/metrics"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"

	"github.com/monstercameron/GoWebComponents/v7/internal/gwcattrs"
	"github.com/monstercameron/GoWebComponents/v7/ui"
)

// Metric instrument names, exactly as exported.
const (
	MetricSSRRenderDuration = "gwc.ssr.render.duration"
	MetricServerFnDuration  = "gwc.serverfn.duration"
	MetricPanic             = "gwc.panic"
	MetricHydrationMismatch = "gwc.hydration.mismatch"
	MetricSchedGoroutines   = "gwc.sched.goroutines"
	MetricSchedLatencyP99   = "gwc.sched.latency"
	MetricGCHeapLive        = "gwc.gc.heap.live"
	attrHTTPRoute           = "http.route"
	attrHTTPStatusCode      = "http.response.status_code"
	attrRPCMethod           = "rpc.method"
	routeUnmatched          = "_unmatched"
	runtimeSchedGoroutines  = "/sched/goroutines:goroutines"
	runtimeSchedLatencies   = "/sched/latencies:seconds"
	runtimeGCHeapLive       = "/gc/heap/live:bytes"
	schedLatencyQuantile    = 0.99
)

// bridgeMetrics holds the instruments of one Bridge and the bounded attribute
// sets that keep their cardinality finite.
type bridgeMetrics struct {
	ssrDuration  otelmetric.Float64Histogram
	fnDuration   otelmetric.Float64Histogram
	panics       otelmetric.Int64Counter
	mismatch     otelmetric.Int64Counter
	registration otelmetric.Registration
	routes       *BoundedSet
	methods      *BoundedSet
	codes        *BoundedSet
}

// buildBridgeMetrics creates every instrument on parseMeter. Instrument creation
// errors are returned so a caller can see a misconfigured provider; the bridge
// still works, with the failed instrument left as a no-op.
func buildBridgeMetrics(parseMeter otelmetric.Meter) (*bridgeMetrics, error) {
	parseM := &bridgeMetrics{routes: NewBoundedSet(0), methods: NewBoundedSet(0), codes: NewBoundedSet(0)}
	var parseFirst error
	parseKeep := func(parseErr error) {
		if parseErr != nil && parseFirst == nil {
			parseFirst = parseErr
		}
	}
	parseBuckets := otelmetric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10)
	var parseErr error
	parseM.ssrDuration, parseErr = parseMeter.Float64Histogram(MetricSSRRenderDuration,
		otelmetric.WithUnit("s"), otelmetric.WithDescription("Duration of SSR requests."), parseBuckets)
	parseKeep(parseErr)
	parseM.fnDuration, parseErr = parseMeter.Float64Histogram(MetricServerFnDuration,
		otelmetric.WithUnit("s"), otelmetric.WithDescription("Duration of server function calls."), parseBuckets)
	parseKeep(parseErr)
	parseM.panics, parseErr = parseMeter.Int64Counter(MetricPanic,
		otelmetric.WithUnit("{panic}"), otelmetric.WithDescription("Reported panics by gwc error code."))
	parseKeep(parseErr)
	parseM.mismatch, parseErr = parseMeter.Int64Counter(MetricHydrationMismatch,
		otelmetric.WithUnit("{mismatch}"), otelmetric.WithDescription("Hydration mismatches observed during SSR hydration."))
	parseKeep(parseErr)

	parseGoroutines, parseErr := parseMeter.Int64ObservableGauge(MetricSchedGoroutines,
		otelmetric.WithUnit("{goroutine}"), otelmetric.WithDescription("Live goroutines."))
	parseKeep(parseErr)
	parseLatency, parseErr := parseMeter.Float64ObservableGauge(MetricSchedLatencyP99,
		otelmetric.WithUnit("s"), otelmetric.WithDescription("99th percentile goroutine scheduling latency."))
	parseKeep(parseErr)
	parseHeap, parseErr := parseMeter.Int64ObservableGauge(MetricGCHeapLive,
		otelmetric.WithUnit("By"), otelmetric.WithDescription("Heap memory occupied by live objects as of the last GC."))
	parseKeep(parseErr)

	parseSamples := []metrics.Sample{{Name: runtimeSchedGoroutines}, {Name: runtimeSchedLatencies}, {Name: runtimeGCHeapLive}}
	parseM.registration, parseErr = parseMeter.RegisterCallback(func(parseCtx context.Context, parseObserver otelmetric.Observer) error {
		metrics.Read(parseSamples)
		if parseSamples[0].Value.Kind() == metrics.KindUint64 {
			parseObserver.ObserveInt64(parseGoroutines, clampInt64(parseSamples[0].Value.Uint64()))
		}
		if parseSamples[1].Value.Kind() == metrics.KindFloat64Histogram {
			if parseP99, parseOk := getHistogramQuantile(parseSamples[1].Value.Float64Histogram(), schedLatencyQuantile); parseOk {
				parseObserver.ObserveFloat64(parseLatency, parseP99)
			}
		}
		if parseSamples[2].Value.Kind() == metrics.KindUint64 {
			parseObserver.ObserveInt64(parseHeap, clampInt64(parseSamples[2].Value.Uint64()))
		}
		return nil
	}, parseGoroutines, parseLatency, parseHeap)
	parseKeep(parseErr)
	return parseM, parseFirst
}

// clampInt64 converts a runtime counter to int64 without overflow.
func clampInt64(parseValue uint64) int64 {
	if parseValue > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(parseValue)
}

// getHistogramQuantile returns the upper bound of the bucket holding the
// parseQuantile-th observation. The bool is false for an empty histogram. An
// infinite top bucket falls back to its finite lower bound.
func getHistogramQuantile(parseHist *metrics.Float64Histogram, parseQuantile float64) (float64, bool) {
	if parseHist == nil || len(parseHist.Counts) == 0 {
		return 0, false
	}
	var parseTotal uint64
	for _, parseCount := range parseHist.Counts {
		parseTotal += parseCount
	}
	if parseTotal == 0 {
		return 0, false
	}
	parseTarget := uint64(math.Ceil(float64(parseTotal) * parseQuantile))
	var parseSeen uint64
	for parseIndex, parseCount := range parseHist.Counts {
		parseSeen += parseCount
		if parseSeen >= parseTarget {
			parseUpper := parseHist.Buckets[parseIndex+1]
			if math.IsInf(parseUpper, 1) {
				parseUpper = parseHist.Buckets[parseIndex]
			}
			if math.IsInf(parseUpper, 0) {
				return 0, false
			}
			return parseUpper, true
		}
	}
	return 0, false
}

// getRouteLabel reduces a ServeMux pattern ("GET /users/{id}") to its path so
// the label is the route template, never the concrete URL.
func getRouteLabel(parsePattern string) string {
	parsePattern = strings.TrimSpace(parsePattern)
	if parsePattern == "" {
		return routeUnmatched
	}
	if parseIndex := strings.IndexByte(parsePattern, ' '); parseIndex >= 0 {
		parsePattern = strings.TrimSpace(parsePattern[parseIndex+1:])
	}
	if parsePattern == "" {
		return routeUnmatched
	}
	return parsePattern
}

// recordSSRDuration records one SSR request on the render duration histogram.
func (parseBridge *Bridge) recordSSRDuration(parseCtx context.Context, parseRequest *http.Request, parseStatus int, parseElapsed time.Duration) {
	if parseBridge.metrics == nil {
		return
	}
	if parseStatus == 0 {
		parseStatus = http.StatusOK
	}
	parseBridge.metrics.ssrDuration.Record(parseCtx, parseElapsed.Seconds(), otelmetric.WithAttributes(
		attribute.String(attrHTTPRoute, parseBridge.metrics.routes.Bound(getRouteLabel(parseRequest.Pattern))),
		attribute.Int(attrHTTPStatusCode, parseStatus)))
}

// recordServerFnDuration records one server function call on its histogram.
func (parseBridge *Bridge) recordServerFnDuration(parseCtx context.Context, parseName string, parseElapsed time.Duration) {
	if parseBridge.metrics == nil {
		return
	}
	parseBridge.metrics.fnDuration.Record(parseCtx, parseElapsed.Seconds(), otelmetric.WithAttributes(
		attribute.String(attrRPCMethod, parseBridge.metrics.methods.Bound(parseName))))
}

// RecordPanic counts one panic under its gwc error code. It is the function
// Install registers with the diagnostics panic observer.
func (parseBridge *Bridge) RecordPanic(parseCode string) {
	if parseBridge.metrics == nil {
		return
	}
	parseBridge.metrics.panics.Add(context.Background(), 1, otelmetric.WithAttributes(
		attribute.String(gwcattrs.KeyErrorCode, parseBridge.metrics.codes.Bound(parseCode))))
}

// RecordHydrationMismatch adds parseCount hydration mismatches to the counter.
// Non-positive counts are ignored.
func (parseBridge *Bridge) RecordHydrationMismatch(parseCtx context.Context, parseCount int) {
	if parseBridge.metrics == nil || parseCount <= 0 {
		return
	}
	parseBridge.metrics.mismatch.Add(parseCtx, int64(parseCount))
}

// closeMetrics unregisters the runtime gauge callback.
func (parseBridge *Bridge) closeMetrics() {
	if parseBridge.metrics != nil && parseBridge.metrics.registration != nil {
		_ = parseBridge.metrics.registration.Unregister()
	}
}

// ObserveSSR is RecordSSRObservation plus the bridge's metrics: it adds the
// observation's hydration mismatch count to gwc.hydration.mismatch.
func (parseBridge *Bridge) ObserveSSR(parseCtx context.Context, parseObservation ui.SSRObservation) {
	RecordSSRObservation(parseCtx, parseObservation)
	if parseObservation.Hydration != nil {
		parseBridge.RecordHydrationMismatch(parseCtx, parseObservation.Hydration.MismatchCount)
	}
}
