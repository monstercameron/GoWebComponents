//go:build !js || !wasm
// +build !js !wasm

package logging

import (
	"context"
	"strings"

	"github.com/monstercameron/GoWebComponents/ui"
)

// resolveExternalContextDetails returns correlation and trace details exposed by native framework helpers.
func resolveExternalContextDetails(parseCtx context.Context) logContextDetails {
	if parseCtx == nil {
		return logContextDetails{}
	}

	parseDetails := logContextDetails{
		correlationID: strings.TrimSpace(ui.CorrelationIDFromContext(parseCtx)),
	}
	if parseTraceContext, parseOk := ui.TraceContextFromContext(parseCtx); parseOk {
		parseDetails.traceContext = TraceContext{
			TraceID:      strings.TrimSpace(parseTraceContext.TraceID),
			ParentSpanID: strings.TrimSpace(parseTraceContext.ParentSpanID),
			SpanID:       strings.TrimSpace(parseTraceContext.SpanID),
			Flags:        strings.TrimSpace(parseTraceContext.Flags),
			TraceState:   strings.TrimSpace(parseTraceContext.TraceState),
		}
		if parseDetails.correlationID == "" {
			parseDetails.correlationID = parseDetails.traceContext.TraceID
		}
	}
	return parseDetails
}
