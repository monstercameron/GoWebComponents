package logging

import (
	"context"
	"fmt"
	"strings"
)

type correlationIDKeyType struct{}
type traceContextKeyType struct{}

var (
	correlationIDKey = correlationIDKeyType{}
	traceContextKey  = traceContextKeyType{}
)

// TraceContext holds W3C trace-context identifiers for one logging scope.
type TraceContext struct {
	TraceID      string
	ParentSpanID string
	SpanID       string
	Flags        string
	TraceState   string
}

// IsValid reports whether the trace context carries one usable 128-bit trace ID.
func (parseTraceContext TraceContext) IsValid() bool {
	return len(parseTraceContext.TraceID) == 32 && parseTraceContext.TraceID != "00000000000000000000000000000000"
}

// Traceparent formats the trace context as one W3C traceparent header value.
func (parseTraceContext TraceContext) Traceparent() string {
	if !parseTraceContext.IsValid() {
		return ""
	}
	parseFlags := strings.TrimSpace(parseTraceContext.Flags)
	if parseFlags == "" {
		parseFlags = "00"
	}
	parseSpanID := strings.TrimSpace(parseTraceContext.SpanID)
	if parseSpanID == "" {
		parseSpanID = "0000000000000000"
	}
	return fmt.Sprintf("00-%s-%s-%s", parseTraceContext.TraceID, parseSpanID, parseFlags)
}

// StoreCorrelationID stores one correlation ID in ctx for later log enrichment.
func StoreCorrelationID(parseCtx context.Context, parseCorrelationID string) context.Context {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return context.WithValue(parseCtx, correlationIDKey, strings.TrimSpace(parseCorrelationID))
}

// GetCorrelationID returns the correlation ID stored by StoreCorrelationID.
func GetCorrelationID(parseCtx context.Context) string {
	if parseCtx == nil {
		return ""
	}
	if parseCorrelationID, parseOk := parseCtx.Value(correlationIDKey).(string); parseOk {
		return strings.TrimSpace(parseCorrelationID)
	}
	return ""
}

// StoreTraceContext stores one trace context in ctx for later log enrichment.
func StoreTraceContext(parseCtx context.Context, parseTraceContext TraceContext) context.Context {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	parseTraceContext.TraceID = strings.TrimSpace(parseTraceContext.TraceID)
	parseTraceContext.ParentSpanID = strings.TrimSpace(parseTraceContext.ParentSpanID)
	parseTraceContext.SpanID = strings.TrimSpace(parseTraceContext.SpanID)
	parseTraceContext.Flags = strings.TrimSpace(parseTraceContext.Flags)
	parseTraceContext.TraceState = strings.TrimSpace(parseTraceContext.TraceState)
	return context.WithValue(parseCtx, traceContextKey, parseTraceContext)
}

// GetTraceContext returns the trace context stored by StoreTraceContext.
func GetTraceContext(parseCtx context.Context) (TraceContext, bool) {
	if parseCtx == nil {
		return TraceContext{}, false
	}
	if parseTraceContext, parseOk := parseCtx.Value(traceContextKey).(TraceContext); parseOk {
		return parseTraceContext, true
	}
	return TraceContext{}, false
}
