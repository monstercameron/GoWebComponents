//go:build !js || !wasm

package ui_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func TestWithCorrelationIDRoundTrip(parseT *testing.T) {
	parseCtx := ui.WithCorrelationID(context.Background(), "req-abc123")
	if parseGot := ui.CorrelationIDFromContext(parseCtx); parseGot != "req-abc123" {
		parseT.Fatalf("expected req-abc123, got %q", parseGot)
	}
}

func TestWithCorrelationIDTrimsSpace(parseT *testing.T) {
	parseCtx := ui.WithCorrelationID(context.Background(), "  req-trim  ")
	if parseGot := ui.CorrelationIDFromContext(parseCtx); parseGot != "req-trim" {
		parseT.Fatalf("expected req-trim, got %q", parseGot)
	}
}

func TestCorrelationIDFromContextEmpty(parseT *testing.T) {
	if parseGot := ui.CorrelationIDFromContext(context.Background()); parseGot != "" {
		parseT.Fatalf("expected empty string, got %q", parseGot)
	}
}

func TestSSRObservabilityOptionsFromContext(parseT *testing.T) {
	parseCtx := ui.WithCorrelationID(context.Background(), "trace-xyz")
	parseOpts := ui.SSRObservabilityOptionsFromContext(parseCtx)
	if parseOpts.CorrelationID != "trace-xyz" {
		parseT.Fatalf("expected trace-xyz, got %q", parseOpts.CorrelationID)
	}
}

func TestSetBootstrapCorrelationID(parseT *testing.T) {
	parseCtx := ui.WithCorrelationID(context.Background(), "boot-corr")
	var parseBootstrap ui.SSRBootstrap
	ui.ConfigureBootstrapCorrelation(parseCtx, &parseBootstrap)
	if parseBootstrap.CorrelationID != "boot-corr" {
		parseT.Fatalf("expected boot-corr, got %q", parseBootstrap.CorrelationID)
	}
}

func TestSetBootstrapCorrelationIDNilSafe(parseT *testing.T) {
	parseCtx := ui.WithCorrelationID(context.Background(), "should-not-panic")
	ui.ConfigureBootstrapCorrelation(parseCtx, nil) // must not panic
}

func TestSSRCorrelationMiddlewareReadsXCorrelationID(parseT *testing.T) {
	parseHandler := ui.WrapSSRCorrelation(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.WriteHeader(http.StatusOK)
		_, _ = parseW.Write([]byte(ui.CorrelationIDFromContext(parseR.Context())))
	}))

	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseReq.Header.Set("X-Correlation-ID", "incoming-corr-id")
	parseRec := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRec, parseReq)

	if parseRec.Body.String() != "incoming-corr-id" {
		parseT.Fatalf("expected incoming-corr-id in context, got %q", parseRec.Body.String())
	}
	if parseRec.Header().Get("X-Correlation-ID") != "incoming-corr-id" {
		parseT.Fatalf("expected X-Correlation-ID response header to be incoming-corr-id, got %q", parseRec.Header().Get("X-Correlation-ID"))
	}
}

func TestSSRCorrelationMiddlewareFallsBackToXRequestID(parseT *testing.T) {
	parseHandler := ui.WrapSSRCorrelation(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.WriteHeader(http.StatusOK)
		_, _ = parseW.Write([]byte(ui.CorrelationIDFromContext(parseR.Context())))
	}))

	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseReq.Header.Set("X-Request-ID", "request-id-fallback")
	parseRec := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRec, parseReq)

	if parseRec.Body.String() != "request-id-fallback" {
		parseT.Fatalf("expected request-id-fallback, got %q", parseRec.Body.String())
	}
}

func TestSSRCorrelationMiddlewareFallsBackToXTraceID(parseT *testing.T) {
	parseHandler := ui.WrapSSRCorrelation(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.WriteHeader(http.StatusOK)
		_, _ = parseW.Write([]byte(ui.CorrelationIDFromContext(parseR.Context())))
	}))

	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseReq.Header.Set("X-Trace-ID", "trace-id-fallback")
	parseRec := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRec, parseReq)

	if parseRec.Body.String() != "trace-id-fallback" {
		parseT.Fatalf("expected trace-id-fallback, got %q", parseRec.Body.String())
	}
}

func TestSSRCorrelationMiddlewareGeneratesIDWhenMissing(parseT *testing.T) {
	var parseCapturedID string
	parseHandler := ui.WrapSSRCorrelation(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseCapturedID = ui.CorrelationIDFromContext(parseR.Context())
		parseW.WriteHeader(http.StatusOK)
	}))

	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseRec := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRec, parseReq)

	if parseCapturedID == "" {
		parseT.Fatal("expected generated correlation ID, got empty string")
	}
	if parseRec.Header().Get("X-Correlation-ID") != parseCapturedID {
		parseT.Fatalf("response header X-Correlation-ID %q does not match context value %q", parseRec.Header().Get("X-Correlation-ID"), parseCapturedID)
	}
}

func TestSSRCorrelationMiddlewarePrefersXCorrelationIDOverFallbacks(parseT *testing.T) {
	parseHandler := ui.WrapSSRCorrelation(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.WriteHeader(http.StatusOK)
		_, _ = parseW.Write([]byte(ui.CorrelationIDFromContext(parseR.Context())))
	}))

	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseReq.Header.Set("X-Correlation-ID", "primary-id")
	parseReq.Header.Set("X-Request-ID", "fallback-id")
	parseRec := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRec, parseReq)

	if parseRec.Body.String() != "primary-id" {
		parseT.Fatalf("expected X-Correlation-ID to take priority, got %q", parseRec.Body.String())
	}
}

// ─── W3C Trace Context ────────────────────────────────────────────────────────

func TestW3CTraceContextIsValid(parseT *testing.T) {
	parseValid := ui.W3CTraceContext{TraceID: "4bf92f3577b34da6a3ce929d0e0e4736"}
	if !parseValid.IsValid() {
		parseT.Fatal("expected valid trace context to report IsValid() == true")
	}
	parseZero := ui.W3CTraceContext{TraceID: "00000000000000000000000000000000"}
	if parseZero.IsValid() {
		parseT.Fatal("all-zero trace ID must not be considered valid")
	}
	parseShort := ui.W3CTraceContext{TraceID: "abc123"}
	if parseShort.IsValid() {
		parseT.Fatal("a short trace ID must not be considered valid")
	}
}

func TestW3CTraceContextTraceparentFormat(parseT *testing.T) {
	parseTc := ui.W3CTraceContext{
		TraceID: "4bf92f3577b34da6a3ce929d0e0e4736",
		SpanID:  "00f067aa0ba902b7",
		Flags:   "01",
	}
	parseGot := parseTc.Traceparent()
	parseWant := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	if parseGot != parseWant {
		parseT.Fatalf("Traceparent() = %q; want %q", parseGot, parseWant)
	}
}

func TestW3CTraceContextTraceparentEmptyForInvalidTraceID(parseT *testing.T) {
	parseTc := ui.W3CTraceContext{TraceID: "tooshort", SpanID: "00f067aa0ba902b7", Flags: "01"}
	if parseGot := parseTc.Traceparent(); parseGot != "" {
		parseT.Fatalf("expected empty Traceparent() for invalid trace ID, got %q", parseGot)
	}
}

// ─── traceparent middleware ───────────────────────────────────────────────────

func TestSSRCorrelationMiddlewarePrefersTraceparentOverXCorrelationID(parseT *testing.T) {
	const (
		traceID     = "4bf92f3577b34da6a3ce929d0e0e4736"
		traceparent = "00-" + traceID + "-00f067aa0ba902b7-01"
	)
	var parseCapturedID string
	parseHandler := ui.WrapSSRCorrelation(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseCapturedID = ui.CorrelationIDFromContext(parseR.Context())
		parseW.WriteHeader(http.StatusOK)
	}))

	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseReq.Header.Set("Traceparent", traceparent)
	parseReq.Header.Set("X-Correlation-ID", "legacy-id")
	parseRec := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRec, parseReq)

	if parseCapturedID != traceID {
		parseT.Fatalf("expected traceparent trace-id %q as correlation ID, got %q", traceID, parseCapturedID)
	}
}

func TestSSRCorrelationMiddlewareWritesTraceparentResponse(parseT *testing.T) {
	const (
		traceID     = "4bf92f3577b34da6a3ce929d0e0e4736"
		parentSpan  = "00f067aa0ba902b7"
		traceparent = "00-" + traceID + "-" + parentSpan + "-01"
	)
	parseHandler := ui.WrapSSRCorrelation(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.WriteHeader(http.StatusOK)
	}))

	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseReq.Header.Set("Traceparent", traceparent)
	parseRec := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRec, parseReq)

	parseResp := parseRec.Header().Get("Traceparent")
	if parseResp == "" {
		parseT.Fatal("expected Traceparent response header, got empty string")
	}
	// Must start with "00-{same traceID}-" and end with "-01"
	parsePrefix := "00-" + traceID + "-"
	if len(parseResp) < len(parsePrefix) || parseResp[:len(parsePrefix)] != parsePrefix {
		parseT.Fatalf("Traceparent response %q does not start with %q", parseResp, parsePrefix)
	}
	if parseResp[len(parseResp)-3:] != "-01" {
		parseT.Fatalf("Traceparent response %q does not preserve sampled flag", parseResp)
	}
}

func TestSSRCorrelationMiddlewareForwardsTracestate(parseT *testing.T) {
	const tracestate = "vendor=opaquevalue,rojo=00f067aa0ba902b7"
	parseHandler := ui.WrapSSRCorrelation(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.WriteHeader(http.StatusOK)
	}))

	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseReq.Header.Set("Traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	parseReq.Header.Set("Tracestate", tracestate)
	parseRec := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRec, parseReq)

	if parseGot := parseRec.Header().Get("Tracestate"); parseGot != tracestate {
		parseT.Fatalf("expected Tracestate %q forwarded to response, got %q", tracestate, parseGot)
	}
}

func TestTraceContextFromContext(parseT *testing.T) {
	const (
		traceID     = "4bf92f3577b34da6a3ce929d0e0e4736"
		parentSpan  = "00f067aa0ba902b7"
		traceparent = "00-" + traceID + "-" + parentSpan + "-01"
	)
	var parseCapturedTC ui.W3CTraceContext
	var isFound bool
	parseHandler := ui.WrapSSRCorrelation(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseCapturedTC, isFound = ui.TraceContextFromContext(parseR.Context())
		parseW.WriteHeader(http.StatusOK)
	}))

	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseReq.Header.Set("Traceparent", traceparent)
	parseRec := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRec, parseReq)

	if !isFound {
		parseT.Fatal("expected W3CTraceContext in context, got not found")
	}
	if parseCapturedTC.TraceID != traceID {
		parseT.Fatalf("TraceID = %q; want %q", parseCapturedTC.TraceID, traceID)
	}
	if parseCapturedTC.ParentSpanID != parentSpan {
		parseT.Fatalf("ParentSpanID = %q; want %q", parseCapturedTC.ParentSpanID, parentSpan)
	}
	if parseCapturedTC.Flags != "01" {
		parseT.Fatalf("Flags = %q; want 01", parseCapturedTC.Flags)
	}
	if len(parseCapturedTC.SpanID) != 16 {
		parseT.Fatalf("SpanID = %q; want 16-char hex", parseCapturedTC.SpanID)
	}
}

func TestSSRCorrelationMiddlewareGeneratesOTelCompatibleIDs(parseT *testing.T) {
	var parseCapturedID string
	var parseCapturedTC ui.W3CTraceContext
	parseHandler := ui.WrapSSRCorrelation(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseCapturedID = ui.CorrelationIDFromContext(parseR.Context())
		parseCapturedTC, _ = ui.TraceContextFromContext(parseR.Context())
		parseW.WriteHeader(http.StatusOK)
	}))

	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseRec := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRec, parseReq)

	if len(parseCapturedID) != 32 {
		parseT.Fatalf("generated correlation ID must be 32 hex chars (OTel 128-bit trace ID), got %q (len %d)", parseCapturedID, len(parseCapturedID))
	}
	if len(parseCapturedTC.SpanID) != 16 {
		parseT.Fatalf("generated span ID must be 16 hex chars (OTel 64-bit span ID), got %q (len %d)", parseCapturedTC.SpanID, len(parseCapturedTC.SpanID))
	}
}

// ─── SSRObservationAttributes ─────────────────────────────────────────────────

func TestSSRObservationAttributesBaseFields(parseT *testing.T) {
	parseObs := ui.SSRObservation{
		Name:          "ssr.render.finish",
		Domain:        "render",
		Phase:         "finish",
		CorrelationID: "trace-abc",
	}
	parseAttrs := ui.GetSSRObservationAttributes(parseObs)
	parseChecks := map[string]string{
		"gwc.ssr.event.name":     "ssr.render.finish",
		"gwc.ssr.event.domain":   "render",
		"gwc.ssr.event.phase":    "finish",
		"gwc.ssr.correlation_id": "trace-abc",
	}
	for parseK, parseWant := range parseChecks {
		if parseGot := parseAttrs[parseK]; parseGot != parseWant {
			parseT.Errorf("attrs[%q] = %q; want %q", parseK, parseGot, parseWant)
		}
	}
}

func TestSSRObservationAttributesRenderMetrics(parseT *testing.T) {
	parseObs := ui.SSRObservation{
		Name:   "ssr.render.finish",
		Render: &ui.SSRRenderMetrics{DurationNs: 5_000_000},
	}
	parseAttrs := ui.GetSSRObservationAttributes(parseObs)
	if parseAttrs["gwc.ssr.render.duration_ns"] != "5000000" {
		parseT.Errorf("render.duration_ns = %q; want 5000000", parseAttrs["gwc.ssr.render.duration_ns"])
	}
	if parseAttrs["gwc.ssr.render.duration_ms"] != "5.000" {
		parseT.Errorf("render.duration_ms = %q; want 5.000", parseAttrs["gwc.ssr.render.duration_ms"])
	}
}

func TestSSRObservationAttributesHydrationMetrics(parseT *testing.T) {
	parseObs := ui.SSRObservation{
		Name: "ssr.hydration.finish",
		Hydration: &ui.SSRHydrationMetrics{
			DurationNs:           2_000_000,
			ExistingDOMNodeCount: 42,
			FallbackCount:        1,
			MismatchCount:        0,
			DiscardedNodeCount:   3,
			Strict:               true,
			Failed:               false,
		},
	}
	parseAttrs := ui.GetSSRObservationAttributes(parseObs)
	if parseAttrs["gwc.ssr.hydration.duration_ns"] != "2000000" {
		parseT.Errorf("hydration.duration_ns = %q; want 2000000", parseAttrs["gwc.ssr.hydration.duration_ns"])
	}
	if parseAttrs["gwc.ssr.hydration.existing_dom_nodes"] != "42" {
		parseT.Errorf("hydration.existing_dom_nodes = %q; want 42", parseAttrs["gwc.ssr.hydration.existing_dom_nodes"])
	}
	if parseAttrs["gwc.ssr.hydration.strict"] != "true" {
		parseT.Errorf("hydration.strict = %q; want true", parseAttrs["gwc.ssr.hydration.strict"])
	}
	if parseAttrs["gwc.ssr.hydration.failed"] != "false" {
		parseT.Errorf("hydration.failed = %q; want false", parseAttrs["gwc.ssr.hydration.failed"])
	}
	if _, hasFailure := parseAttrs["gwc.ssr.hydration.failure"]; hasFailure {
		parseT.Error("gwc.ssr.hydration.failure must not be present when Failure is empty string")
	}
}

func TestSSRObservationAttributesHydrationFailure(parseT *testing.T) {
	parseObs := ui.SSRObservation{
		Name: "ssr.hydration.error",
		Hydration: &ui.SSRHydrationMetrics{
			Failed:  true,
			Failure: "root mismatch: expected div, got span",
		},
	}
	parseAttrs := ui.GetSSRObservationAttributes(parseObs)
	if parseAttrs["gwc.ssr.hydration.failed"] != "true" {
		parseT.Errorf("hydration.failed = %q; want true", parseAttrs["gwc.ssr.hydration.failed"])
	}
	if parseAttrs["gwc.ssr.hydration.failure"] != "root mismatch: expected div, got span" {
		parseT.Errorf("hydration.failure = %q; want failure message", parseAttrs["gwc.ssr.hydration.failure"])
	}
}

func TestSSRObservationAttributesBootstrapMetrics(parseT *testing.T) {
	parseObs := ui.SSRObservation{
		Name: "ssr.bootstrap.json",
		Bootstrap: &ui.SSRBootstrapMetrics{
			Format:       "json",
			PayloadBytes: 1024,
			ScriptBytes:  256,
		},
	}
	parseAttrs := ui.GetSSRObservationAttributes(parseObs)
	if parseAttrs["gwc.ssr.bootstrap.format"] != "json" {
		parseT.Errorf("bootstrap.format = %q; want json", parseAttrs["gwc.ssr.bootstrap.format"])
	}
	if parseAttrs["gwc.ssr.bootstrap.payload_bytes"] != "1024" {
		parseT.Errorf("bootstrap.payload_bytes = %q; want 1024", parseAttrs["gwc.ssr.bootstrap.payload_bytes"])
	}
	if parseAttrs["gwc.ssr.bootstrap.script_bytes"] != "256" {
		parseT.Errorf("bootstrap.script_bytes = %q; want 256", parseAttrs["gwc.ssr.bootstrap.script_bytes"])
	}
}
