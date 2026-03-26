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
// When stored by WrapSSRCorrelation, TraceID equals the value returned by
// CorrelationIDFromContext so the two surfaces remain consistent.
type W3CTraceContext struct {
	TraceID      string
	ParentSpanID string
	SpanID       string
	Flags        string
	TraceState   string
}

// IsValid reports whether the trace context carries a usable 128-bit trace ID.
func (parseTc W3CTraceContext) IsValid() bool {
	return len(parseTc.TraceID) == 32 && parseTc.TraceID != "00000000000000000000000000000000"
}

// Traceparent formats the context as a W3C traceparent header value using SpanID
// as the current operation identifier so downstream callers can continue the trace.
func (parseTc W3CTraceContext) Traceparent() string {
	if !parseTc.IsValid() {
		return ""
	}
	parseFlags := parseTc.Flags
	if parseFlags == "" {
		parseFlags = "00"
	}
	parseSpanID := parseTc.SpanID
	if parseSpanID == "" {
		parseSpanID = "0000000000000000"
	}
	return fmt.Sprintf("00-%s-%s-%s", parseTc.TraceID, parseSpanID, parseFlags)
}

// WithCorrelationID stores a correlation ID in the request context.
// Retrieve it later with CorrelationIDFromContext.
func WithCorrelationID(parseCtx context.Context, parseId string) context.Context {
	return context.WithValue(parseCtx, correlationIDKey, strings.TrimSpace(parseId))
}

// CorrelationIDFromContext returns the correlation ID stored by WithCorrelationID
// or WrapSSRCorrelation, or an empty string if none is present.
func CorrelationIDFromContext(parseCtx context.Context) string {
	if parseId, parseOk := parseCtx.Value(correlationIDKey).(string); parseOk {
		return parseId
	}
	return ""
}

// WithW3CTraceContext stores a W3CTraceContext in ctx.
// WrapSSRCorrelation calls this automatically when a valid traceparent is present.
func WithW3CTraceContext(parseCtx context.Context, parseTc W3CTraceContext) context.Context {
	return context.WithValue(parseCtx, w3cTraceContextKey, parseTc)
}

// TraceContextFromContext returns the W3C trace context stored by
// WrapSSRCorrelation, and whether one was found.
func TraceContextFromContext(parseCtx context.Context) (W3CTraceContext, bool) {
	if parseTc, parseOk := parseCtx.Value(w3cTraceContextKey).(W3CTraceContext); parseOk {
		return parseTc, true
	}
	return W3CTraceContext{}, false
}

// SSRObservabilityOptionsFromContext builds SSRObservabilityOptions using the
// correlation ID stored in ctx by WrapSSRCorrelation or WithCorrelationID.
//
//	opts := ui.SSRObservabilityOptionsFromContext(r.Context())
//	markup, err := ui.RenderToStringObserved(root, opts)
func SSRObservabilityOptionsFromContext(parseCtx context.Context) SSRObservabilityOptions {
	return SSRObservabilityOptions{
		CorrelationID: CorrelationIDFromContext(parseCtx),
	}
}

// ConfigureBootstrapCorrelation embeds the correlation ID from ctx into the bootstrap
// payload so the client can auto-adopt it during hydration without manual threading.
//
//	ui.ConfigureBootstrapCorrelation(r.Context(), &bootstrap)
func ConfigureBootstrapCorrelation(parseCtx context.Context, parseBootstrap *SSRBootstrap) {
	if parseBootstrap == nil {
		return
	}
	if parseId := CorrelationIDFromContext(parseCtx); parseId != "" {
		parseBootstrap.CorrelationID = parseId
	}
}

// GetSSRObservationAttributes maps one SSRObservation to OTel-compatible span attribute
// key-value pairs.
//
// The returned map uses stable "gwc.*" attribute keys following OpenTelemetry naming
// conventions. Pass them to your OTel SDK with SetAttributes or equivalent:
//
//	attrs := ui.GetSSRObservationAttributes(event)
//	for k, v := range attrs {
//	    span.SetAttributes(attribute.String(k, v))
//	}
//
// Keys are guaranteed stable across patch releases once the observability surface
// is marked stable.
func GetSSRObservationAttributes(parseObservation SSRObservation) map[string]string {
	parseAttrs := map[string]string{
		"gwc.ssr.event.name":      parseObservation.Name,
		"gwc.ssr.event.domain":    parseObservation.Domain,
		"gwc.ssr.event.phase":     parseObservation.Phase,
		"gwc.ssr.correlation_id":  parseObservation.CorrelationID,
		"gwc.ssr.event.timestamp": parseObservation.Timestamp.UTC().Format("2006-01-02T15:04:05.999999999Z"),
	}
	if parseR := parseObservation.Render; parseR != nil {
		parseAttrs["gwc.ssr.render.duration_ms"] = fmt.Sprintf("%.3f", float64(parseR.DurationNs)/1e6)
		parseAttrs["gwc.ssr.render.duration_ns"] = fmt.Sprintf("%d", parseR.DurationNs)
	}
	if parseB := parseObservation.Bootstrap; parseB != nil {
		parseAttrs["gwc.ssr.bootstrap.format"] = parseB.Format
		parseAttrs["gwc.ssr.bootstrap.payload_bytes"] = fmt.Sprintf("%d", parseB.PayloadBytes)
		parseAttrs["gwc.ssr.bootstrap.script_bytes"] = fmt.Sprintf("%d", parseB.ScriptBytes)
	}
	if parseH := parseObservation.Hydration; parseH != nil {
		parseAttrs["gwc.ssr.hydration.duration_ms"] = fmt.Sprintf("%.3f", float64(parseH.DurationNs)/1e6)
		parseAttrs["gwc.ssr.hydration.duration_ns"] = fmt.Sprintf("%d", parseH.DurationNs)
		parseAttrs["gwc.ssr.hydration.existing_dom_nodes"] = fmt.Sprintf("%d", parseH.ExistingDOMNodeCount)
		parseAttrs["gwc.ssr.hydration.fallback_count"] = fmt.Sprintf("%d", parseH.FallbackCount)
		parseAttrs["gwc.ssr.hydration.mismatch_count"] = fmt.Sprintf("%d", parseH.MismatchCount)
		parseAttrs["gwc.ssr.hydration.discarded_node_count"] = fmt.Sprintf("%d", parseH.DiscardedNodeCount)
		parseAttrs["gwc.ssr.hydration.strict"] = fmt.Sprintf("%t", parseH.Strict)
		parseAttrs["gwc.ssr.hydration.failed"] = fmt.Sprintf("%t", parseH.Failed)
		if parseH.Failure != "" {
			parseAttrs["gwc.ssr.hydration.failure"] = parseH.Failure
		}
	}
	return parseAttrs
}

// WrapSSRCorrelation is an HTTP middleware that propagates per-request
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
func WrapSSRCorrelation(parseNext http.Handler) http.Handler {
	return http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseTc := resolveTraceContext(parseR)
		parseW.Header().Set("X-Correlation-ID", parseTc.TraceID)
		if parseTp := parseTc.Traceparent(); parseTp != "" {
			parseW.Header().Set("Traceparent", parseTp)
		}
		if parseTc.TraceState != "" {
			parseW.Header().Set("Tracestate", parseTc.TraceState)
		}
		parseCtx := WithCorrelationID(parseR.Context(), parseTc.TraceID)
		parseCtx = WithW3CTraceContext(parseCtx, parseTc)
		parseNext.ServeHTTP(parseW, parseR.WithContext(parseCtx))
	})
}

// resolveTraceContext builds a W3CTraceContext for the incoming request.
func resolveTraceContext(parseR *http.Request) W3CTraceContext {
	// 1. W3C traceparent
	if parseTp := strings.TrimSpace(parseR.Header.Get("Traceparent")); parseTp != "" {
		if parseTc, parseOk := parseTraceparent(parseTp); parseOk {
			return W3CTraceContext{
				TraceID:      parseTc.TraceID,
				ParentSpanID: parseTc.ParentSpanID,
				SpanID:       generateSpanID(),
				Flags:        parseTc.Flags,
				TraceState:   strings.TrimSpace(parseR.Header.Get("Tracestate")),
			}
		}
	}
	// 2. Legacy correlation headers — keep as correlation ID, synthesize OTel context
	for _, parseHeader := range []string{"X-Correlation-ID", "X-Request-ID", "X-Trace-ID"} {
		if parseV := strings.TrimSpace(parseR.Header.Get(parseHeader)); parseV != "" {
			return W3CTraceContext{
				TraceID:      normalizeToTraceID(parseV),
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
func parseTraceparent(parseHeader string) (W3CTraceContext, bool) {
	parseParts := strings.Split(parseHeader, "-")
	if len(parseParts) != 4 {
		return W3CTraceContext{}, false
	}
	parseVersion, parseTraceID, parseParentSpanID, parseFlags := parseParts[0], parseParts[1], parseParts[2], parseParts[3]
	if len(parseVersion) != 2 || !isLowercaseHex(parseVersion) {
		return W3CTraceContext{}, false
	}
	if len(parseTraceID) != 32 || !isLowercaseHex(parseTraceID) || parseTraceID == "00000000000000000000000000000000" {
		return W3CTraceContext{}, false
	}
	if len(parseParentSpanID) != 16 || !isLowercaseHex(parseParentSpanID) || parseParentSpanID == "0000000000000000" {
		return W3CTraceContext{}, false
	}
	if len(parseFlags) != 2 || !isLowercaseHex(parseFlags) {
		return W3CTraceContext{}, false
	}
	return W3CTraceContext{
		TraceID:      parseTraceID,
		ParentSpanID: parseParentSpanID,
		Flags:        parseFlags,
	}, true
}

// normalizeToTraceID returns a 32-char hex trace ID when the input is a valid
// 16, 24, or 32-char hex string (left-padded with zeroes). Otherwise returns
// the input unchanged (Traceparent() will return empty for invalid trace IDs).
func normalizeToTraceID(parseV string) string {
	parseV = strings.ToLower(strings.TrimSpace(parseV))
	if len(parseV) == 32 && isLowercaseHex(parseV) {
		return parseV
	}
	if (len(parseV) == 16 || len(parseV) == 24) && isLowercaseHex(parseV) {
		return strings.Repeat("0", 32-len(parseV)) + parseV
	}
	return parseV
}

// isLowercaseHex is a core package helper.
func isLowercaseHex(parseS string) bool {
	for _, parseC := range parseS {
		if !((parseC >= '0' && parseC <= '9') || (parseC >= 'a' && parseC <= 'f')) {
			return false
		}
	}
	return true
}

// generateTraceID generates a random 128-bit (32 lowercase hex char) OTel-compatible trace ID.
func generateTraceID() string {
	var parseB [16]byte
	if _, parseErr := rand.Read(parseB[:]); parseErr != nil {
		return ""
	}
	return hex.EncodeToString(parseB[:])
}

// generateSpanID generates a random 64-bit (16 lowercase hex char) OTel-compatible span ID.
func generateSpanID() string {
	var parseB [8]byte
	if _, parseErr := rand.Read(parseB[:]); parseErr != nil {
		return ""
	}
	return hex.EncodeToString(parseB[:])
}
