//go:build !js || !wasm

package logging

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// resolveExternalContextDetails reads native framework metadata exposed by ui.
func resolveExternalContextDetails(parseCtx context.Context) logContextDetails {
	parseDetails := logContextDetails{
		correlationID: ui.CorrelationIDFromContext(parseCtx),
	}
	if parseTraceContext, parseOk := ui.TraceContextFromContext(parseCtx); parseOk {
		parseDetails.traceContext = TraceContext{
			TraceID:      parseTraceContext.TraceID,
			ParentSpanID: parseTraceContext.ParentSpanID,
			SpanID:       parseTraceContext.SpanID,
			Flags:        parseTraceContext.Flags,
			TraceState:   parseTraceContext.TraceState,
		}
	}
	return parseDetails
}
