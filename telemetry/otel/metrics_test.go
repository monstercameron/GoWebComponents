//go:build !js || !wasm

package otel

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/monstercameron/GoWebComponents/v7/internal/diagnostics"
	"github.com/monstercameron/GoWebComponents/v7/ui"
)

// buildMetricsBridge installs a Bridge on an in-memory SDK and returns the reader.
func buildMetricsBridge(parseT *testing.T) (*Bridge, *sdkmetric.ManualReader, *tracetest.InMemoryExporter) {
	parseT.Helper()
	parseReader := sdkmetric.NewManualReader()
	parseMeters := sdkmetric.NewMeterProvider(sdkmetric.WithReader(parseReader))
	parseSpans := tracetest.NewInMemoryExporter()
	parseTraces := sdktrace.NewTracerProvider(sdktrace.WithSyncer(parseSpans))
	parseBridge, parseShutdown := Install(Options{TracerProvider: parseTraces, MeterProvider: parseMeters})
	parseT.Cleanup(func() { _ = parseShutdown(context.Background()) })
	return parseBridge, parseReader, parseSpans
}

// collectMetrics returns every exported metric keyed by instrument name.
func collectMetrics(parseT *testing.T, parseReader *sdkmetric.ManualReader) map[string]metricdata.Metrics {
	parseT.Helper()
	var parseData metricdata.ResourceMetrics
	if parseErr := parseReader.Collect(context.Background(), &parseData); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseOut := map[string]metricdata.Metrics{}
	for _, parseScope := range parseData.ScopeMetrics {
		for _, parseMetric := range parseScope.Metrics {
			parseOut[parseMetric.Name] = parseMetric
		}
	}
	return parseOut
}

func TestMetricsRecordedFromMiddlewareAndServerFn(parseT *testing.T) {
	parseBridge, parseReader, _ := buildMetricsBridge(parseT)
	parseSrv := buildTestServer(parseT, parseBridge)
	getPage(parseT, parseSrv.URL, "")

	parseAll := collectMetrics(parseT, parseReader)

	parseSSR, parseOk := parseAll[MetricSSRRenderDuration]
	if !parseOk {
		parseT.Fatalf("missing %s in %v", MetricSSRRenderDuration, parseAll)
	}
	if parseSSR.Unit != "s" {
		parseT.Fatalf("ssr unit = %q", parseSSR.Unit)
	}
	parseHist := parseSSR.Data.(metricdata.Histogram[float64])
	if len(parseHist.DataPoints) != 1 || parseHist.DataPoints[0].Count != 1 {
		parseT.Fatalf("ssr datapoints = %+v", parseHist.DataPoints)
	}
	parseAttrs := parseHist.DataPoints[0].Attributes
	if parseRoute, _ := parseAttrs.Value("http.route"); parseRoute.AsString() != "/page" {
		parseT.Fatalf("http.route = %v", parseAttrs)
	}
	if parseStatus, _ := parseAttrs.Value("http.response.status_code"); parseStatus.AsInt64() != 200 {
		parseT.Fatalf("status = %v", parseAttrs)
	}

	parseFn, parseOk := parseAll[MetricServerFnDuration]
	if !parseOk || parseFn.Unit != "s" {
		parseT.Fatalf("serverfn metric = %+v ok=%v", parseFn, parseOk)
	}
	parseFnHist := parseFn.Data.(metricdata.Histogram[float64])
	parseMethods := map[string]uint64{}
	for _, parsePoint := range parseFnHist.DataPoints {
		parseMethod, _ := parsePoint.Attributes.Value("rpc.method")
		parseMethods[parseMethod.AsString()] = parsePoint.Count
	}
	if parseMethods["echo"] != 1 || parseMethods["boom"] != 1 || len(parseMethods) != 2 {
		parseT.Fatalf("rpc.method counts = %v", parseMethods)
	}
}

func TestPanicAndHydrationCounters(parseT *testing.T) {
	parseBridge, parseReader, _ := buildMetricsBridge(parseT)
	diagnostics.NotifyPanic("GWC-FRAMEWORK-USAGE")
	diagnostics.NotifyPanic("GWC-FRAMEWORK-USAGE")
	parseBridge.ObserveSSR(context.Background(), ui.SSRObservation{Name: "hydrate", Hydration: &ui.SSRHydrationMetrics{MismatchCount: 3}})
	parseBridge.RecordHydrationMismatch(context.Background(), 0)

	parseAll := collectMetrics(parseT, parseReader)
	parsePanic := parseAll[MetricPanic]
	if parsePanic.Unit != "{panic}" {
		parseT.Fatalf("panic unit = %q", parsePanic.Unit)
	}
	parseSum := parsePanic.Data.(metricdata.Sum[int64])
	if len(parseSum.DataPoints) != 1 || parseSum.DataPoints[0].Value != 2 {
		parseT.Fatalf("panic points = %+v", parseSum.DataPoints)
	}
	if parseCode, _ := parseSum.DataPoints[0].Attributes.Value("gwc.error.code"); parseCode.AsString() != "GWC-FRAMEWORK-USAGE" {
		parseT.Fatalf("panic attrs = %v", parseSum.DataPoints[0].Attributes)
	}
	parseMismatch := parseAll[MetricHydrationMismatch].Data.(metricdata.Sum[int64])
	if len(parseMismatch.DataPoints) != 1 || parseMismatch.DataPoints[0].Value != 3 {
		parseT.Fatalf("mismatch points = %+v", parseMismatch.DataPoints)
	}
}

func TestRuntimeGaugesObserved(parseT *testing.T) {
	_, parseReader, _ := buildMetricsBridge(parseT)
	parseAll := collectMetrics(parseT, parseReader)
	parseGoroutines, parseOk := parseAll[MetricSchedGoroutines]
	if !parseOk || parseGoroutines.Unit != "{goroutine}" {
		parseT.Fatalf("goroutines metric = %+v ok=%v", parseGoroutines, parseOk)
	}
	if parseGauge := parseGoroutines.Data.(metricdata.Gauge[int64]); len(parseGauge.DataPoints) != 1 || parseGauge.DataPoints[0].Value < 1 {
		parseT.Fatalf("goroutine points = %+v", parseGauge.DataPoints)
	}
	parseHeap, parseOk := parseAll[MetricGCHeapLive]
	if !parseOk || parseHeap.Unit != "By" {
		parseT.Fatalf("heap metric = %+v ok=%v", parseHeap, parseOk)
	}
	if _, parseOk := parseAll[MetricSchedLatencyP99]; !parseOk {
		parseT.Log("sched latency p99 not observed (empty histogram); allowed on an idle runtime")
	}
}

func TestShutdownUnregistersGaugesAndPanicObserver(parseT *testing.T) {
	parseReader := sdkmetric.NewManualReader()
	parseMeters := sdkmetric.NewMeterProvider(sdkmetric.WithReader(parseReader))
	_, parseShutdown := Install(Options{MeterProvider: parseMeters})
	_ = parseShutdown(context.Background())
	if _, parseOk := collectMetrics(parseT, parseReader)[MetricSchedGoroutines]; parseOk {
		if parseGauge := collectMetrics(parseT, parseReader)[MetricSchedGoroutines].Data.(metricdata.Gauge[int64]); len(parseGauge.DataPoints) != 0 {
			parseT.Fatalf("gauge still observed after shutdown: %+v", parseGauge.DataPoints)
		}
	}
	diagnostics.NotifyPanic("GWC-X")
	if parseMetric, parseOk := collectMetrics(parseT, parseReader)[MetricPanic]; parseOk {
		parseT.Fatalf("panic counted after shutdown: %+v", parseMetric)
	}
}

func TestQuantileFromRuntimeHistogramBuckets(parseT *testing.T) {
	if _, parseOk := getHistogramQuantile(nil, 0.99); parseOk {
		parseT.Fatal("nil histogram yielded a value")
	}
	parseHist := &metricsHistogram{Counts: []uint64{90, 9, 1}, Buckets: []float64{0, 1, 2, 4}}
	if parseGot, _ := getHistogramQuantile(parseHist, 0.99); parseGot != 2 {
		parseT.Fatalf("p99 = %v, want 2", parseGot)
	}
	parseInf := &metricsHistogram{Counts: []uint64{0, 5}, Buckets: []float64{0, 1, infinity()}}
	if parseGot, _ := getHistogramQuantile(parseInf, 0.99); parseGot != 1 {
		parseT.Fatalf("p99 with +Inf bucket = %v, want 1", parseGot)
	}
}

func TestRouteLabelStripsMethodAndEmpty(parseT *testing.T) {
	for parseIn, parseWant := range map[string]string{
		"GET /users/{id}": "/users/{id}", "/plain": "/plain", "": routeUnmatched, "  ": routeUnmatched,
	} {
		if parseGot := getRouteLabel(parseIn); parseGot != parseWant {
			parseT.Errorf("getRouteLabel(%q) = %q, want %q", parseIn, parseGot, parseWant)
		}
	}
}

func TestRouteCardinalityIsBounded(parseT *testing.T) {
	parseBridge, parseReader, _ := buildMetricsBridge(parseT)
	for parseIndex := 0; parseIndex < DefaultAttributeCap+25; parseIndex++ {
		parseRequest := newPatternRequest(fmt.Sprintf("/r/%d", parseIndex))
		parseBridge.recordSSRDuration(context.Background(), parseRequest, 200, time.Millisecond)
	}
	parseHist := collectMetrics(parseT, parseReader)[MetricSSRRenderDuration].Data.(metricdata.Histogram[float64])
	if len(parseHist.DataPoints) != DefaultAttributeCap+1 {
		parseT.Fatalf("series = %d, want %d", len(parseHist.DataPoints), DefaultAttributeCap+1)
	}
	parseOther := attribute.NewSet(attribute.String("http.route", OverflowValue), attribute.Int("http.response.status_code", 200))
	for _, parsePoint := range parseHist.DataPoints {
		if parsePoint.Attributes.Equals(&parseOther) && parsePoint.Count != 25 {
			parseT.Fatalf("overflow count = %d, want 25", parsePoint.Count)
		}
	}
}
