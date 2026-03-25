//go:build !js || !wasm
// +build !js !wasm

package ui_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestWithCorrelationIDRoundTrip(t *testing.T) {
	ctx := ui.WithCorrelationID(context.Background(), "req-abc123")
	if got := ui.CorrelationIDFromContext(ctx); got != "req-abc123" {
		t.Fatalf("expected req-abc123, got %q", got)
	}
}

func TestWithCorrelationIDTrimsSpace(t *testing.T) {
	ctx := ui.WithCorrelationID(context.Background(), "  req-trim  ")
	if got := ui.CorrelationIDFromContext(ctx); got != "req-trim" {
		t.Fatalf("expected req-trim, got %q", got)
	}
}

func TestCorrelationIDFromContextEmpty(t *testing.T) {
	if got := ui.CorrelationIDFromContext(context.Background()); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestSSRObservabilityOptionsFromContext(t *testing.T) {
	ctx := ui.WithCorrelationID(context.Background(), "trace-xyz")
	opts := ui.SSRObservabilityOptionsFromContext(ctx)
	if opts.CorrelationID != "trace-xyz" {
		t.Fatalf("expected trace-xyz, got %q", opts.CorrelationID)
	}
}

func TestSetBootstrapCorrelationID(t *testing.T) {
	ctx := ui.WithCorrelationID(context.Background(), "boot-corr")
	var bootstrap ui.SSRBootstrap
	ui.SetBootstrapCorrelationID(ctx, &bootstrap)
	if bootstrap.CorrelationID != "boot-corr" {
		t.Fatalf("expected boot-corr, got %q", bootstrap.CorrelationID)
	}
}

func TestSetBootstrapCorrelationIDNilSafe(t *testing.T) {
	ctx := ui.WithCorrelationID(context.Background(), "should-not-panic")
	ui.SetBootstrapCorrelationID(ctx, nil) // must not panic
}

func TestSSRCorrelationMiddlewareReadsXCorrelationID(t *testing.T) {
	handler := ui.SSRCorrelationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(ui.CorrelationIDFromContext(r.Context())))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Correlation-ID", "incoming-corr-id")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "incoming-corr-id" {
		t.Fatalf("expected incoming-corr-id in context, got %q", rec.Body.String())
	}
	if rec.Header().Get("X-Correlation-ID") != "incoming-corr-id" {
		t.Fatalf("expected X-Correlation-ID response header to be incoming-corr-id, got %q", rec.Header().Get("X-Correlation-ID"))
	}
}

func TestSSRCorrelationMiddlewareFallsBackToXRequestID(t *testing.T) {
	handler := ui.SSRCorrelationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(ui.CorrelationIDFromContext(r.Context())))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "request-id-fallback")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "request-id-fallback" {
		t.Fatalf("expected request-id-fallback, got %q", rec.Body.String())
	}
}

func TestSSRCorrelationMiddlewareFallsBackToXTraceID(t *testing.T) {
	handler := ui.SSRCorrelationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(ui.CorrelationIDFromContext(r.Context())))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Trace-ID", "trace-id-fallback")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "trace-id-fallback" {
		t.Fatalf("expected trace-id-fallback, got %q", rec.Body.String())
	}
}

func TestSSRCorrelationMiddlewareGeneratesIDWhenMissing(t *testing.T) {
	var capturedID string
	handler := ui.SSRCorrelationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = ui.CorrelationIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if capturedID == "" {
		t.Fatal("expected generated correlation ID, got empty string")
	}
	if rec.Header().Get("X-Correlation-ID") != capturedID {
		t.Fatalf("response header X-Correlation-ID %q does not match context value %q", rec.Header().Get("X-Correlation-ID"), capturedID)
	}
}

func TestSSRCorrelationMiddlewarePrefersXCorrelationIDOverFallbacks(t *testing.T) {
	handler := ui.SSRCorrelationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(ui.CorrelationIDFromContext(r.Context())))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Correlation-ID", "primary-id")
	req.Header.Set("X-Request-ID", "fallback-id")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "primary-id" {
		t.Fatalf("expected X-Correlation-ID to take priority, got %q", rec.Body.String())
	}
}

// ─── W3C Trace Context ────────────────────────────────────────────────────────

func TestW3CTraceContextIsValid(t *testing.T) {
	valid := ui.W3CTraceContext{TraceID: "4bf92f3577b34da6a3ce929d0e0e4736"}
	if !valid.IsValid() {
		t.Fatal("expected valid trace context to report IsValid() == true")
	}
	zero := ui.W3CTraceContext{TraceID: "00000000000000000000000000000000"}
	if zero.IsValid() {
		t.Fatal("all-zero trace ID must not be considered valid")
	}
	short := ui.W3CTraceContext{TraceID: "abc123"}
	if short.IsValid() {
		t.Fatal("a short trace ID must not be considered valid")
	}
}

func TestW3CTraceContextTraceparentFormat(t *testing.T) {
	tc := ui.W3CTraceContext{
		TraceID: "4bf92f3577b34da6a3ce929d0e0e4736",
		SpanID:  "00f067aa0ba902b7",
		Flags:   "01",
	}
	got := tc.Traceparent()
	want := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	if got != want {
		t.Fatalf("Traceparent() = %q; want %q", got, want)
	}
}

func TestW3CTraceContextTraceparentEmptyForInvalidTraceID(t *testing.T) {
	tc := ui.W3CTraceContext{TraceID: "tooshort", SpanID: "00f067aa0ba902b7", Flags: "01"}
	if got := tc.Traceparent(); got != "" {
		t.Fatalf("expected empty Traceparent() for invalid trace ID, got %q", got)
	}
}

// ─── traceparent middleware ───────────────────────────────────────────────────

func TestSSRCorrelationMiddlewarePrefersTraceparentOverXCorrelationID(t *testing.T) {
	const (
		traceID    = "4bf92f3577b34da6a3ce929d0e0e4736"
		traceparent = "00-" + traceID + "-00f067aa0ba902b7-01"
	)
	var capturedID string
	handler := ui.SSRCorrelationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = ui.CorrelationIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Traceparent", traceparent)
	req.Header.Set("X-Correlation-ID", "legacy-id")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if capturedID != traceID {
		t.Fatalf("expected traceparent trace-id %q as correlation ID, got %q", traceID, capturedID)
	}
}

func TestSSRCorrelationMiddlewareWritesTraceparentResponse(t *testing.T) {
	const (
		traceID    = "4bf92f3577b34da6a3ce929d0e0e4736"
		parentSpan = "00f067aa0ba902b7"
		traceparent = "00-" + traceID + "-" + parentSpan + "-01"
	)
	handler := ui.SSRCorrelationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Traceparent", traceparent)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Header().Get("Traceparent")
	if resp == "" {
		t.Fatal("expected Traceparent response header, got empty string")
	}
	// Must start with "00-{same traceID}-" and end with "-01"
	prefix := "00-" + traceID + "-"
	if len(resp) < len(prefix) || resp[:len(prefix)] != prefix {
		t.Fatalf("Traceparent response %q does not start with %q", resp, prefix)
	}
	if resp[len(resp)-3:] != "-01" {
		t.Fatalf("Traceparent response %q does not preserve sampled flag", resp)
	}
}

func TestSSRCorrelationMiddlewareForwardsTracestate(t *testing.T) {
	const tracestate = "vendor=opaquevalue,rojo=00f067aa0ba902b7"
	handler := ui.SSRCorrelationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	req.Header.Set("Tracestate", tracestate)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Tracestate"); got != tracestate {
		t.Fatalf("expected Tracestate %q forwarded to response, got %q", tracestate, got)
	}
}

func TestTraceContextFromContext(t *testing.T) {
	const (
		traceID    = "4bf92f3577b34da6a3ce929d0e0e4736"
		parentSpan = "00f067aa0ba902b7"
		traceparent = "00-" + traceID + "-" + parentSpan + "-01"
	)
	var capturedTC ui.W3CTraceContext
	var found bool
	handler := ui.SSRCorrelationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTC, found = ui.TraceContextFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Traceparent", traceparent)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !found {
		t.Fatal("expected W3CTraceContext in context, got not found")
	}
	if capturedTC.TraceID != traceID {
		t.Fatalf("TraceID = %q; want %q", capturedTC.TraceID, traceID)
	}
	if capturedTC.ParentSpanID != parentSpan {
		t.Fatalf("ParentSpanID = %q; want %q", capturedTC.ParentSpanID, parentSpan)
	}
	if capturedTC.Flags != "01" {
		t.Fatalf("Flags = %q; want 01", capturedTC.Flags)
	}
	if len(capturedTC.SpanID) != 16 {
		t.Fatalf("SpanID = %q; want 16-char hex", capturedTC.SpanID)
	}
}

func TestSSRCorrelationMiddlewareGeneratesOTelCompatibleIDs(t *testing.T) {
	var capturedID string
	var capturedTC ui.W3CTraceContext
	handler := ui.SSRCorrelationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = ui.CorrelationIDFromContext(r.Context())
		capturedTC, _ = ui.TraceContextFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if len(capturedID) != 32 {
		t.Fatalf("generated correlation ID must be 32 hex chars (OTel 128-bit trace ID), got %q (len %d)", capturedID, len(capturedID))
	}
	if len(capturedTC.SpanID) != 16 {
		t.Fatalf("generated span ID must be 16 hex chars (OTel 64-bit span ID), got %q (len %d)", capturedTC.SpanID, len(capturedTC.SpanID))
	}
}

// ─── SSRObservationAttributes ─────────────────────────────────────────────────

func TestSSRObservationAttributesBaseFields(t *testing.T) {
	obs := ui.SSRObservation{
		Name:          "ssr.render.finish",
		Domain:        "render",
		Phase:         "finish",
		CorrelationID: "trace-abc",
	}
	attrs := ui.SSRObservationAttributes(obs)
	checks := map[string]string{
		"gwc.ssr.event.name":     "ssr.render.finish",
		"gwc.ssr.event.domain":   "render",
		"gwc.ssr.event.phase":    "finish",
		"gwc.ssr.correlation_id": "trace-abc",
	}
	for k, want := range checks {
		if got := attrs[k]; got != want {
			t.Errorf("attrs[%q] = %q; want %q", k, got, want)
		}
	}
}

func TestSSRObservationAttributesRenderMetrics(t *testing.T) {
	obs := ui.SSRObservation{
		Name:   "ssr.render.finish",
		Render: &ui.SSRRenderMetrics{DurationNs: 5_000_000},
	}
	attrs := ui.SSRObservationAttributes(obs)
	if attrs["gwc.ssr.render.duration_ns"] != "5000000" {
		t.Errorf("render.duration_ns = %q; want 5000000", attrs["gwc.ssr.render.duration_ns"])
	}
	if attrs["gwc.ssr.render.duration_ms"] != "5.000" {
		t.Errorf("render.duration_ms = %q; want 5.000", attrs["gwc.ssr.render.duration_ms"])
	}
}

func TestSSRObservationAttributesHydrationMetrics(t *testing.T) {
	obs := ui.SSRObservation{
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
	attrs := ui.SSRObservationAttributes(obs)
	if attrs["gwc.ssr.hydration.duration_ns"] != "2000000" {
		t.Errorf("hydration.duration_ns = %q; want 2000000", attrs["gwc.ssr.hydration.duration_ns"])
	}
	if attrs["gwc.ssr.hydration.existing_dom_nodes"] != "42" {
		t.Errorf("hydration.existing_dom_nodes = %q; want 42", attrs["gwc.ssr.hydration.existing_dom_nodes"])
	}
	if attrs["gwc.ssr.hydration.strict"] != "true" {
		t.Errorf("hydration.strict = %q; want true", attrs["gwc.ssr.hydration.strict"])
	}
	if attrs["gwc.ssr.hydration.failed"] != "false" {
		t.Errorf("hydration.failed = %q; want false", attrs["gwc.ssr.hydration.failed"])
	}
	if _, hasFailure := attrs["gwc.ssr.hydration.failure"]; hasFailure {
		t.Error("gwc.ssr.hydration.failure must not be present when Failure is empty string")
	}
}

func TestSSRObservationAttributesHydrationFailure(t *testing.T) {
	obs := ui.SSRObservation{
		Name: "ssr.hydration.error",
		Hydration: &ui.SSRHydrationMetrics{
			Failed:  true,
			Failure: "root mismatch: expected div, got span",
		},
	}
	attrs := ui.SSRObservationAttributes(obs)
	if attrs["gwc.ssr.hydration.failed"] != "true" {
		t.Errorf("hydration.failed = %q; want true", attrs["gwc.ssr.hydration.failed"])
	}
	if attrs["gwc.ssr.hydration.failure"] != "root mismatch: expected div, got span" {
		t.Errorf("hydration.failure = %q; want failure message", attrs["gwc.ssr.hydration.failure"])
	}
}

func TestSSRObservationAttributesBootstrapMetrics(t *testing.T) {
	obs := ui.SSRObservation{
		Name: "ssr.bootstrap.json",
		Bootstrap: &ui.SSRBootstrapMetrics{
			Format:       "json",
			PayloadBytes: 1024,
			ScriptBytes:  256,
		},
	}
	attrs := ui.SSRObservationAttributes(obs)
	if attrs["gwc.ssr.bootstrap.format"] != "json" {
		t.Errorf("bootstrap.format = %q; want json", attrs["gwc.ssr.bootstrap.format"])
	}
	if attrs["gwc.ssr.bootstrap.payload_bytes"] != "1024" {
		t.Errorf("bootstrap.payload_bytes = %q; want 1024", attrs["gwc.ssr.bootstrap.payload_bytes"])
	}
	if attrs["gwc.ssr.bootstrap.script_bytes"] != "256" {
		t.Errorf("bootstrap.script_bytes = %q; want 256", attrs["gwc.ssr.bootstrap.script_bytes"])
	}
}
