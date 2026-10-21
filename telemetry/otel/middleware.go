//go:build !js || !wasm

package otel

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/monstercameron/GoWebComponents/v7/internal/errcode"
	"github.com/monstercameron/GoWebComponents/v7/internal/gwcattrs"
	"github.com/monstercameron/GoWebComponents/v7/ui"
)

// SSRSpanName is the name of the server span started around an SSR request.
const SSRSpanName = "gwc.ssr.render"

// ServerFnSpanPrefix prefixes the name of every server function span.
const ServerFnSpanPrefix = "gwc.serverfn."

// statusRecorder captures the status code a handler writes.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader records the first status written.
func (parseW *statusRecorder) WriteHeader(parseStatus int) {
	if parseW.status == 0 {
		parseW.status = parseStatus
	}
	parseW.ResponseWriter.WriteHeader(parseStatus)
}

// Write records an implicit 200.
func (parseW *statusRecorder) Write(parseData []byte) (int, error) {
	if parseW.status == 0 {
		parseW.status = http.StatusOK
	}
	return parseW.ResponseWriter.Write(parseData)
}

// Unwrap exposes the underlying writer to http.ResponseController.
func (parseW *statusRecorder) Unwrap() http.ResponseWriter { return parseW.ResponseWriter }

// Middleware wraps parseNext in a gwc.ssr.render server span with
// ui.WrapSSRCorrelation semantics: the span continues an incoming trace,
// X-Correlation-ID is the span trace id, and Traceparent/Tracestate are echoed
// with the span's own id and sampled flag. When the tracer yields no valid span
// context (for example a no-op provider with no incoming trace) it falls back to
// ui.WrapSSRCorrelation.
func (parseBridge *Bridge) Middleware(parseNext http.Handler) http.Handler {
	parseFallback := ui.WrapSSRCorrelation(parseNext)
	return http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseCtx := parseBridge.propagator.Extract(parseR.Context(), propagation.HeaderCarrier(parseR.Header))
		parseParent := trace.SpanContextFromContext(parseCtx)
		parseCtx, parseSpan := parseBridge.tracer.Start(parseCtx, SSRSpanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(convertAttributes(map[string]string{
				"http.request.method": parseR.Method,
				"url.path":            parseR.URL.Path,
			})...))
		parseSC := parseSpan.SpanContext()
		if !parseSC.IsValid() {
			parseSpan.End()
			parseFallback.ServeHTTP(parseW, parseR)
			return
		}
		if parseLegacy := parseLegacyCorrelationID(parseR); parseLegacy != "" && !parseParent.IsValid() {
			// No W3C parent: keep the caller's own id visible on the span.
			parseSpan.SetAttributes(convertAttributes(map[string]string{gwcattrs.KeySSRCorrelationID: parseLegacy})...)
		}
		parseTc := ui.W3CTraceContext{
			TraceID:    parseSC.TraceID().String(),
			SpanID:     parseSC.SpanID().String(),
			Flags:      formatTraceFlags(parseSC),
			TraceState: parseSC.TraceState().String(),
		}
		if parseParent.IsValid() {
			parseTc.ParentSpanID = parseParent.SpanID().String()
		}
		parseW.Header().Set("X-Correlation-ID", parseTc.TraceID)
		if parseTp := parseTc.Traceparent(); parseTp != "" {
			parseW.Header().Set("Traceparent", parseTp)
		}
		if parseTc.TraceState != "" {
			parseW.Header().Set("Tracestate", parseTc.TraceState)
		}
		parseCtx = ui.WithCorrelationID(parseCtx, parseTc.TraceID)
		parseCtx = ui.WithW3CTraceContext(parseCtx, parseTc)

		parseRec := &statusRecorder{ResponseWriter: parseW}
		parseStarted := time.Now()
		parseServed := parseR.WithContext(parseCtx)
		defer func() {
			if parseVal := recover(); parseVal != nil {
				recordSpanError(parseSpan, fmt.Errorf("panic: %v", parseVal))
				parseSpan.SetStatus(codes.Error, "panic")
				parseSpan.End()
				panic(parseVal)
			}
			finishHTTPSpan(parseSpan, parseRec.status, nil)
			parseBridge.recordSSRDuration(parseCtx, parseServed, parseRec.status, time.Since(parseStarted))
		}()
		parseNext.ServeHTTP(parseRec, parseServed)
	})
}

// RecordSSRObservation adds the observation's gwc.ssr.* attributes to the span in
// parseCtx and marks the span failed when the observation reports an error phase
// or a failed hydration. It does nothing when parseCtx carries no recording span.
func RecordSSRObservation(parseCtx context.Context, parseObservation ui.SSRObservation) {
	parseSpan := trace.SpanFromContext(parseCtx)
	if !parseSpan.IsRecording() {
		return
	}
	parseSpan.SetAttributes(convertAttributes(ui.GetSSRObservationAttributes(parseObservation))...)
	if parseObservation.Phase == "error" || (parseObservation.Hydration != nil && parseObservation.Hydration.Failed) {
		parseSpan.SetStatus(codes.Error, "ssr observation reported failure")
	}
}

// startServerFnSpan is the serverfn.ServerSpanFunc: it continues the incoming
// trace and starts a gwc.serverfn.<name> server span.
func (parseBridge *Bridge) startServerFnSpan(parseCtx context.Context, parseName string, parseHeader http.Header) (context.Context, func(int, error)) {
	parseCtx = parseBridge.propagator.Extract(parseCtx, propagation.HeaderCarrier(parseHeader))
	parseCtx, parseSpan := parseBridge.tracer.Start(parseCtx, ServerFnSpanPrefix+parseName,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(attribute.String("gwc.serverfn.name", parseName)))
	parseStarted := time.Now()
	return parseCtx, func(parseStatus int, parseErr error) {
		finishHTTPSpan(parseSpan, parseStatus, parseErr)
		parseBridge.recordServerFnDuration(parseCtx, parseName, time.Since(parseStarted))
	}
}

// finishHTTPSpan records the response status and any error, then ends the span.
// Only 5xx responses mark a server span as failed.
func finishHTTPSpan(parseSpan trace.Span, parseStatus int, parseErr error) {
	if parseStatus != 0 {
		parseSpan.SetAttributes(attribute.String("http.response.status_code", strconv.Itoa(parseStatus)))
	}
	if parseErr != nil {
		recordSpanError(parseSpan, parseErr)
	}
	if parseStatus >= http.StatusInternalServerError {
		parseSpan.SetStatus(codes.Error, http.StatusText(parseStatus))
	}
	parseSpan.End()
}

// recordSpanError adds an exception event carrying the redacted message and the
// gwc.error.code when the error has one.
func recordSpanError(parseSpan trace.Span, parseErr error) {
	parseAttrs := map[string]string{
		gwcattrs.KeyExceptionType:    fmt.Sprintf("%T", parseErr),
		gwcattrs.KeyExceptionMessage: parseErr.Error(),
	}
	if parseCode := errcode.CodeOf(parseErr); parseCode != "" {
		parseAttrs[gwcattrs.KeyErrorCode] = string(parseCode)
	}
	parseSpan.AddEvent("exception", trace.WithAttributes(convertAttributes(parseAttrs)...))
}

// parseLegacyCorrelationID returns the first legacy correlation header value.
func parseLegacyCorrelationID(parseR *http.Request) string {
	for _, parseName := range []string{"X-Correlation-ID", "X-Request-ID", "X-Trace-ID"} {
		if parseValue := parseR.Header.Get(parseName); parseValue != "" {
			return parseValue
		}
	}
	return ""
}
