//go:build !js || !wasm

package otel

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/monstercameron/GoWebComponents/v7/ui"
)

// PropagatorAdapter adapts an OpenTelemetry TextMapPropagator to the header-based
// hooks of ui and serverfn.
type PropagatorAdapter struct {
	propagator propagation.TextMapPropagator
}

// Extract reads trace context from parseHeader with the OpenTelemetry propagator
// and mirrors a valid remote parent into the ui W3CTraceContext so
// ui.WrapSSRCorrelation continues the same trace.
func (parseAdapter *PropagatorAdapter) Extract(parseCtx context.Context, parseHeader http.Header) context.Context {
	parseCtx = parseAdapter.propagator.Extract(parseCtx, propagation.HeaderCarrier(parseHeader))
	parseSC := trace.SpanContextFromContext(parseCtx)
	if parseSC.IsValid() && parseSC.IsRemote() {
		parseCtx = ui.WithW3CTraceContext(parseCtx, ui.W3CTraceContext{
			TraceID:      parseSC.TraceID().String(),
			ParentSpanID: parseSC.SpanID().String(),
			Flags:        formatTraceFlags(parseSC),
			TraceState:   parseSC.TraceState().String(),
		})
	}
	return parseCtx
}

// Inject writes the active span's context into parseHeader. When the context has
// no span but carries a ui W3CTraceContext, that context is written instead so a
// request wrapped only by ui.WrapSSRCorrelation still propagates.
func (parseAdapter *PropagatorAdapter) Inject(parseCtx context.Context, parseHeader http.Header) {
	parseAdapter.propagator.Inject(parseCtx, propagation.HeaderCarrier(parseHeader))
	if parseHeader.Get("Traceparent") != "" {
		return
	}
	if parseTc, parseOk := ui.TraceContextFromContext(parseCtx); parseOk {
		if parseTp := parseTc.Traceparent(); parseTp != "" {
			parseHeader.Set("Traceparent", parseTp)
			if parseTc.TraceState != "" {
				parseHeader.Set("Tracestate", parseTc.TraceState)
			}
		}
	}
}

// formatTraceFlags renders the sampled bit as a traceparent flags field.
func formatTraceFlags(parseSC trace.SpanContext) string {
	if parseSC.TraceFlags().IsSampled() {
		return "01"
	}
	return "00"
}
