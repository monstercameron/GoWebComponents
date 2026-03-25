//go:build !js || !wasm
// +build !js !wasm

package ui

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

type correlationIDKeyType struct{}
type w3cTraceContextKeyType struct{}

var (
	correlationIDKey   = correlationIDKeyType{}
	w3cTraceContextKey = w3cTraceContextKeyType{}
)

// W3CTraceContext holds a parsed W3C Trace Context propagation envelope
// (https://www.w3.org/TR/trace-context/).
//
// TraceID is the 128-bit trace identifier (32 lowercase hex chars).
// ParentSpanID is the 64-bit span identifier received from the upstream caller (16 hex chars).
// SpanID is the 64-bit span identifier for this server-side operation (16 hex chars, newly generated).
// Flags is the two-hex-char trace flags field ("01" = sampled, "00" = not sampled).
// TraceState is the raw, opaque tracestate header value forwarded without modification.
//
// When stored by SSRCorrelationMiddleware, TraceID equals the value returned by
// CorrelationIDFromContext so the two surfaces remain consistent.
type W3CTraceContext struct {
	TraceID      string
	ParentSpanID string
	SpanID       string
	Flags        string
	TraceState   string
}

// IsValid reports whether the trace context carries a usable 128-bit trace ID.
func (tc W3CTraceContext) IsValid() bool {
	return len(tc.TraceID) == 32 && tc.TraceID != "00000000000000000000000000000000"
}

// Traceparent formats the context as a W3C traceparent header value using SpanID
// as the current operation identifier so downstream callers can continue the trace.
func (tc W3CTraceContext) Traceparent() string {
	if !tc.IsValid() {
		return ""
	}
	flags := tc.Flags
	if flags == "" {
		flags = "00"
	}
	spanID := tc.SpanID
	if spanID == "" {
		spanID = "0000000000000000"
	}
	return fmt.Sprintf("00-%s-%s-%s", tc.TraceID, spanID, flags)
}

// WithCorrelationID stores a correlation ID in the request context.
// Retrieve it later with CorrelationIDFromContext.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, strings.TrimSpace(id))
}

// CorrelationIDFromContext returns the correlation ID stored by WithCorrelationID
// or SSRCorrelationMiddleware, or an empty string if none is present.
func CorrelationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}

// WithW3CTraceContext stores a W3CTraceContext in ctx.
// SSRCorrelationMiddleware calls this automatically when a valid traceparent is present.
func WithW3CTraceContext(ctx context.Context, tc W3CTraceContext) context.Context {
	return context.WithValue(ctx, w3cTraceContextKey, tc)
}

// TraceContextFromContext returns the W3C trace context stored by
// SSRCorrelationMiddleware, and whether one was found.
func TraceContextFromContext(ctx context.Context) (W3CTraceContext, bool) {
	if tc, ok := ctx.Value(w3cTraceContextKey).(W3CTraceContext); ok {
		return tc, true
	}
	return W3CTraceContext{}, false
}

// SSRObservabilityOptionsFromContext builds SSRObservabilityOptions using the
// correlation ID stored in ctx by SSRCorrelationMiddleware or WithCorrelationID.
//
//	opts := ui.SSRObservabilityOptionsFromContext(r.Context())
//	markup, err := ui.RenderToStringObserved(root, opts)
func SSRObservabilityOptionsFromContext(ctx context.Context) SSRObservabilityOptions {
	return SSRObservabilityOptions{
		CorrelationID: CorrelationIDFromContext(ctx),
	}
}

// SetBootstrapCorrelationID embeds the correlation ID from ctx into the bootstrap
// payload so the client can auto-adopt it during hydration without manual threading.
//
//	ui.SetBootstrapCorrelationID(r.Context(), &bootstrap)
func SetBootstrapCorrelationID(ctx context.Context, bootstrap *SSRBootstrap) {
	if bootstrap == nil {
		return
	}
	if id := CorrelationIDFromContext(ctx); id != "" {
		bootstrap.CorrelationID = id
	}
}

// SSRObservationAttributes maps one SSRObservation to OTel-compatible span attribute
// key-value pairs.
//
// The returned map uses stable "gwc.*" attribute keys following OpenTelemetry naming
// conventions. Pass them to your OTel SDK with SetAttributes or equivalent:
//
//	attrs := ui.SSRObservationAttributes(event)
//	for k, v := range attrs {
//	    span.SetAttributes(attribute.String(k, v))
//	}
//
// Keys are guaranteed stable across patch releases once the observability surface
// is marked stable.
func SSRObservationAttributes(observation SSRObservation) map[string]string {
	attrs := map[string]string{
		"gwc.ssr.event.name":      observation.Name,
		"gwc.ssr.event.domain":    observation.Domain,
		"gwc.ssr.event.phase":     observation.Phase,
		"gwc.ssr.correlation_id":  observation.CorrelationID,
		"gwc.ssr.event.timestamp": observation.Timestamp.UTC().Format("2006-01-02T15:04:05.999999999Z"),
	}
	if r := observation.Render; r != nil {
		attrs["gwc.ssr.render.duration_ms"] = fmt.Sprintf("%.3f", float64(r.DurationNs)/1e6)
		attrs["gwc.ssr.render.duration_ns"] = fmt.Sprintf("%d", r.DurationNs)
	}
	if b := observation.Bootstrap; b != nil {
		attrs["gwc.ssr.bootstrap.format"] = b.Format
		attrs["gwc.ssr.bootstrap.payload_bytes"] = fmt.Sprintf("%d", b.PayloadBytes)
		attrs["gwc.ssr.bootstrap.script_bytes"] = fmt.Sprintf("%d", b.ScriptBytes)
	}
	if h := observation.Hydration; h != nil {
		attrs["gwc.ssr.hydration.duration_ms"] = fmt.Sprintf("%.3f", float64(h.DurationNs)/1e6)
		attrs["gwc.ssr.hydration.duration_ns"] = fmt.Sprintf("%d", h.DurationNs)
		attrs["gwc.ssr.hydration.existing_dom_nodes"] = fmt.Sprintf("%d", h.ExistingDOMNodeCount)
		attrs["gwc.ssr.hydration.fallback_count"] = fmt.Sprintf("%d", h.FallbackCount)
		attrs["gwc.ssr.hydration.mismatch_count"] = fmt.Sprintf("%d", h.MismatchCount)
		attrs["gwc.ssr.hydration.discarded_node_count"] = fmt.Sprintf("%d", h.DiscardedNodeCount)
		attrs["gwc.ssr.hydration.strict"] = fmt.Sprintf("%t", h.Strict)
		attrs["gwc.ssr.hydration.failed"] = fmt.Sprintf("%t", h.Failed)
		if h.Failure != "" {
			attrs["gwc.ssr.hydration.failure"] = h.Failure
		}
	}
	return attrs
}

// SSRCorrelationMiddleware is an HTTP middleware that propagates per-request
// trace context following W3C Trace Context (https://www.w3.org/TR/trace-context/)
// and common correlation ID conventions.
//
// Resolution order:
//  1. W3C traceparent header — trace-id used as the correlation ID; a new span ID
//     is generated for this server operation; traceparent is written to the response.
//  2. X-Correlation-ID, X-Request-ID, X-Trace-ID — used as-is; a synthetic
//     W3CTraceContext is generated using the value as the trace ID.
//  3. Generated — a fresh OTel-compatible 128-bit trace ID and 64-bit span ID
//     are generated using crypto/rand.
//
// For every request the middleware:
//   - stores the correlation ID in ctx (retrieve with CorrelationIDFromContext)
//   - stores W3CTraceContext in ctx (retrieve with TraceContextFromContext)
//   - writes X-Correlation-ID and Traceparent to the response
//   - forwards the incoming Tracestate header unchanged
//
// Security note: correlation IDs must not encode user IDs, session tokens, or
// sensitive metadata. The middleware does not validate incoming header values
// beyond structural format — callers are responsible for ensuring IDs are opaque.
func SSRCorrelationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc := resolveTraceContext(r)
		w.Header().Set("X-Correlation-ID", tc.TraceID)
		if tp := tc.Traceparent(); tp != "" {
			w.Header().Set("Traceparent", tp)
		}
		if tc.TraceState != "" {
			w.Header().Set("Tracestate", tc.TraceState)
		}
		ctx := WithCorrelationID(r.Context(), tc.TraceID)
		ctx = WithW3CTraceContext(ctx, tc)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// resolveTraceContext builds a W3CTraceContext for the incoming request.
func resolveTraceContext(r *http.Request) W3CTraceContext {
	// 1. W3C traceparent
	if tp := strings.TrimSpace(r.Header.Get("Traceparent")); tp != "" {
		if tc, ok := parseTraceparent(tp); ok {
			return W3CTraceContext{
				TraceID:      tc.TraceID,
				ParentSpanID: tc.ParentSpanID,
				SpanID:       generateSpanID(),
				Flags:        tc.Flags,
				TraceState:   strings.TrimSpace(r.Header.Get("Tracestate")),
			}
		}
	}
	// 2. Legacy correlation headers — keep as correlation ID, synthesize OTel context
	for _, header := range []string{"X-Correlation-ID", "X-Request-ID", "X-Trace-ID"} {
		if v := strings.TrimSpace(r.Header.Get(header)); v != "" {
			return W3CTraceContext{
				TraceID:      normalizeToTraceID(v),
				ParentSpanID: "",
				SpanID:       generateSpanID(),
				Flags:        "00",
			}
		}
	}
	// 3. Generate fresh OTel-compatible IDs
	return W3CTraceContext{
		TraceID:      generateTraceID(),
		ParentSpanID: "",
		SpanID:       generateSpanID(),
		Flags:        "00",
	}
}

// parseTraceparent parses a W3C traceparent header value.
// Returns the parsed context and true on success.
func parseTraceparent(header string) (W3CTraceContext, bool) {
	parts := strings.Split(header, "-")
	if len(parts) != 4 {
		return W3CTraceContext{}, false
	}
	version, traceID, parentSpanID, flags := parts[0], parts[1], parts[2], parts[3]
	if len(version) != 2 || !isLowercaseHex(version) {
		return W3CTraceContext{}, false
	}
	if len(traceID) != 32 || !isLowercaseHex(traceID) || traceID == "00000000000000000000000000000000" {
		return W3CTraceContext{}, false
	}
	if len(parentSpanID) != 16 || !isLowercaseHex(parentSpanID) || parentSpanID == "0000000000000000" {
		return W3CTraceContext{}, false
	}
	if len(flags) != 2 || !isLowercaseHex(flags) {
		return W3CTraceContext{}, false
	}
	return W3CTraceContext{
		TraceID:      traceID,
		ParentSpanID: parentSpanID,
		Flags:        flags,
	}, true
}

// normalizeToTraceID returns a 32-char hex trace ID when the input is a valid
// 16, 24, or 32-char hex string (left-padded with zeroes). Otherwise returns
// the input unchanged (Traceparent() will return empty for invalid trace IDs).
func normalizeToTraceID(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if len(v) == 32 && isLowercaseHex(v) {
		return v
	}
	if (len(v) == 16 || len(v) == 24) && isLowercaseHex(v) {
		return strings.Repeat("0", 32-len(v)) + v
	}
	return v
}

func isLowercaseHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// generateTraceID generates a random 128-bit (32 lowercase hex char) OTel-compatible trace ID.
func generateTraceID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(b[:])
}

// generateSpanID generates a random 64-bit (16 lowercase hex char) OTel-compatible span ID.
func generateSpanID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(b[:])
}

// generateCorrelationID generates a fresh OTel-compatible 128-bit trace ID.
// Retained for callers that generate IDs independently of the middleware.
func generateCorrelationID() string {
	return generateTraceID()
}
