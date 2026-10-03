package serverfn

import (
	"context"
	"net/http"
	"sync"
)

// TracePropagator extracts and injects distributed trace context on HTTP headers.
// It mirrors ui.TracePropagator so a single adapter (see telemetry/otel) satisfies
// both. serverfn imports no tracing SDK; the hook is the only coupling.
type TracePropagator interface {
	// Extract returns a context carrying the trace context found in parseHeader.
	Extract(parseCtx context.Context, parseHeader http.Header) context.Context
	// Inject writes the trace context carried by parseCtx into parseHeader.
	Inject(parseCtx context.Context, parseHeader http.Header)
}

// ServerSpanFunc starts a server-side span for one server function call. It
// returns the context the function runs with and a finish callback that receives
// the HTTP status written to the client and the error the function returned (nil
// on success). The callback is always called exactly once.
type ServerSpanFunc func(parseCtx context.Context, parseName string, parseHeader http.Header) (context.Context, func(parseStatus int, parseErr error))

var (
	traceHookMu     sync.RWMutex
	tracePropagator TracePropagator
	serverSpanHook  ServerSpanFunc
)

// SetTracePropagator installs the propagator Call uses to inject trace context
// into outgoing requests. A nil value removes it, which is the default and leaves
// requests unchanged.
func SetTracePropagator(parsePropagator TracePropagator) {
	traceHookMu.Lock()
	tracePropagator = parsePropagator
	traceHookMu.Unlock()
}

// SetServerSpanHook installs the function Handle uses to wrap each server
// function call in a server span. A nil value removes it, which is the default.
func SetServerSpanHook(parseHook ServerSpanFunc) {
	traceHookMu.Lock()
	serverSpanHook = parseHook
	traceHookMu.Unlock()
}

// injectTraceHeaders writes trace context into parseHeader when a propagator is installed.
func injectTraceHeaders(parseCtx context.Context, parseHeader http.Header) {
	traceHookMu.RLock()
	parsePropagator := tracePropagator
	traceHookMu.RUnlock()
	if parsePropagator != nil {
		parsePropagator.Inject(parseCtx, parseHeader)
	}
}

// serverSpan tracks one in-flight server function span.
type serverSpan struct {
	writer *statusWriter
	finish func(parseStatus int, parseErr error)
	err    error
}

// statusWriter records the status code written to the client.
type statusWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader records parseStatus and forwards it.
func (parseW *statusWriter) WriteHeader(parseStatus int) {
	if parseW.status == 0 {
		parseW.status = parseStatus
	}
	parseW.ResponseWriter.WriteHeader(parseStatus)
}

// Write records an implicit 200 and forwards the bytes.
func (parseW *statusWriter) Write(parseData []byte) (int, error) {
	if parseW.status == 0 {
		parseW.status = http.StatusOK
	}
	return parseW.ResponseWriter.Write(parseData)
}

// beginServerSpan starts a span through the installed hook and swaps the writer
// and request for traced versions. It returns nil when no hook is installed.
func beginServerSpan(parseW *http.ResponseWriter, parseR **http.Request, parseName string) *serverSpan {
	traceHookMu.RLock()
	parseHook := serverSpanHook
	traceHookMu.RUnlock()
	if parseHook == nil {
		return nil
	}
	parseCtx, parseFinish := parseHook((*parseR).Context(), parseName, (*parseR).Header)
	if parseFinish == nil {
		return nil
	}
	parseWriter := &statusWriter{ResponseWriter: *parseW}
	*parseW = parseWriter
	*parseR = (*parseR).WithContext(parseCtx)
	return &serverSpan{writer: parseWriter, finish: parseFinish}
}

// noteError remembers the error the function returned. It is nil-safe.
func (parseSpan *serverSpan) noteError(parseErr error) {
	if parseSpan != nil {
		parseSpan.err = parseErr
	}
}

// end finishes the span with the recorded status. It is nil-safe.
func (parseSpan *serverSpan) end() {
	if parseSpan == nil {
		return
	}
	parseStatus := parseSpan.writer.status
	if parseStatus == 0 {
		parseStatus = http.StatusOK
	}
	parseSpan.finish(parseStatus, parseSpan.err)
}
