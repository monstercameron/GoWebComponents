//go:build !js || !wasm

package ui

import (
	"context"
	"net/http"
	"strings"
	"sync"
)

// TracePropagator extracts and injects distributed trace context on HTTP headers.
// It is a hook only: the ui package imports no tracing SDK, so an adapter (for
// example telemetry/otel) supplies the implementation.
//
// The hook uses net/http and is therefore native-only; the browser build never
// links it.
type TracePropagator interface {
	// Extract returns a context derived from parseCtx that carries the trace
	// context found in parseHeader. Implementations store a valid W3CTraceContext
	// with WithW3CTraceContext when a remote parent is present.
	Extract(parseCtx context.Context, parseHeader http.Header) context.Context
	// Inject writes the trace context carried by parseCtx into parseHeader.
	Inject(parseCtx context.Context, parseHeader http.Header)
}

var (
	tracePropagatorMu sync.RWMutex
	tracePropagator   TracePropagator = defaultTracePropagator{}
)

// SetTracePropagator installs the process-wide propagator used by
// WrapSSRCorrelation and InjectTraceContext. A nil value restores the default
// propagator, which reads and writes only the W3C traceparent and tracestate
// headers and keeps the legacy correlation header fallbacks.
func SetTracePropagator(parsePropagator TracePropagator) {
	tracePropagatorMu.Lock()
	defer tracePropagatorMu.Unlock()
	if parsePropagator == nil {
		tracePropagator = defaultTracePropagator{}
		return
	}
	tracePropagator = parsePropagator
}

// GetTracePropagator returns the active propagator. It never returns nil.
func GetTracePropagator() TracePropagator {
	tracePropagatorMu.RLock()
	defer tracePropagatorMu.RUnlock()
	return tracePropagator
}

// IsTracePropagatorInstalled reports whether a non-default propagator is active.
func IsTracePropagatorInstalled() bool {
	_, parseIsDefault := GetTracePropagator().(defaultTracePropagator)
	return !parseIsDefault
}

// InjectTraceContext writes the trace context carried by parseCtx into
// parseHeader through the installed propagator. It does nothing unless a
// non-default propagator is installed, so outgoing requests are unchanged by
// default.
func InjectTraceContext(parseCtx context.Context, parseHeader http.Header) {
	if parseCtx == nil || parseHeader == nil || !IsTracePropagatorInstalled() {
		return
	}
	GetTracePropagator().Inject(parseCtx, parseHeader)
}

// defaultTracePropagator preserves the behavior that predates the hook: it parses
// a valid W3C traceparent into a W3CTraceContext and writes the stored context
// back out.
type defaultTracePropagator struct{}

// Extract parses a W3C traceparent header into parseCtx.
func (defaultTracePropagator) Extract(parseCtx context.Context, parseHeader http.Header) context.Context {
	parseTp := strings.TrimSpace(parseHeader.Get("Traceparent"))
	if parseTp == "" {
		return parseCtx
	}
	parseTc, parseOk := parseTraceparent(parseTp)
	if !parseOk {
		return parseCtx
	}
	parseTc.TraceState = strings.TrimSpace(parseHeader.Get("Tracestate"))
	return WithW3CTraceContext(parseCtx, parseTc)
}

// Inject writes the stored W3CTraceContext as a traceparent header.
func (defaultTracePropagator) Inject(parseCtx context.Context, parseHeader http.Header) {
	parseTc, parseOk := TraceContextFromContext(parseCtx)
	if !parseOk {
		return
	}
	if parseTp := parseTc.Traceparent(); parseTp != "" {
		parseHeader.Set("Traceparent", parseTp)
		if parseTc.TraceState != "" {
			parseHeader.Set("Tracestate", parseTc.TraceState)
		}
	}
}
