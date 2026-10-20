//go:build !js || !wasm

package otel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/monstercameron/GoWebComponents/v7/internal/gwcattrs"
	"github.com/monstercameron/GoWebComponents/v7/interop"
	"github.com/monstercameron/GoWebComponents/v7/telemetry"
	gwctrace "github.com/monstercameron/GoWebComponents/v7/trace"
)

// ResolveTraceID keeps a valid W3C trace id and otherwise derives a deterministic
// one from the supplied correlation text, so events sharing a correlation id
// share a trace. It repeats the algorithm of the unexported telemetry helper used
// by the OTLP JSON exporter, so both exporters agree on the trace id of an event.
func ResolveTraceID(parseValue string, parseIndex int, parseName string) string {
	parseValue = strings.TrimSpace(parseValue)
	if interop.IsValidTraceID(parseValue) {
		return parseValue
	}
	parseSeed := "trace|" + parseValue
	if parseValue == "" {
		parseSeed = "trace||" + strconv.Itoa(parseIndex) + "|" + parseName
	}
	parseSum := sha256.Sum256([]byte(parseSeed))
	return hex.EncodeToString(parseSum[:16])
}

// buildVirtualParent derives the deterministic non-zero span id used as the
// parent of replayed root spans, so they carry the intended trace id.
func buildVirtualParent(parseTraceID string) string {
	parseSum := sha256.Sum256([]byte("root|" + parseTraceID))
	if parseSum[0] == 0 && parseSum[1] == 0 {
		parseSum[7] = 1
	}
	return hex.EncodeToString(parseSum[:8])
}

// buildParentContext returns a sampled remote span context for parseTraceID and
// parseSpanID. The sampled flag makes parent-based samplers keep replayed spans,
// which were already recorded once.
func buildParentContext(parseTraceID string, parseSpanID string) (trace.SpanContext, bool) {
	parseTID, parseErr := trace.TraceIDFromHex(parseTraceID)
	if parseErr != nil {
		return trace.SpanContext{}, false
	}
	parseSID, parseErr := trace.SpanIDFromHex(parseSpanID)
	if parseErr != nil {
		return trace.SpanContext{}, false
	}
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: parseTID, SpanID: parseSID, TraceFlags: trace.FlagsSampled, Remote: true,
	}), true
}

// replayItem describes one finished span or event to replay.
type replayItem struct {
	name      string
	traceID   string
	selfID    string
	parentID  string
	start     time.Time
	end       time.Time
	kind      trace.SpanKind
	attrs     map[string]string
	failed    bool
	failedMsg string
}

// orderParentsFirst returns item indexes so that an item whose parent id matches
// another item's self id comes after it. Cycles and missing parents fall through
// in input order.
func orderParentsFirst(parseItems []replayItem) []int {
	parseSelf := make(map[string]struct{}, len(parseItems))
	for _, parseItem := range parseItems {
		if parseItem.selfID != "" {
			parseSelf[parseItem.selfID] = struct{}{}
		}
	}
	parseDone := map[string]struct{}{}
	parseOrder := make([]int, 0, len(parseItems))
	parseEmitted := make([]bool, len(parseItems))
	for parseProgress := true; parseProgress; {
		parseProgress = false
		for parseIndex, parseItem := range parseItems {
			if parseEmitted[parseIndex] {
				continue
			}
			_, parseHasParent := parseSelf[parseItem.parentID]
			_, parseParentDone := parseDone[parseItem.parentID]
			if parseItem.parentID == "" || !parseHasParent || parseParentDone {
				parseEmitted[parseIndex] = true
				parseOrder = append(parseOrder, parseIndex)
				if parseItem.selfID != "" {
					parseDone[parseItem.selfID] = struct{}{}
				}
				parseProgress = true
			}
		}
	}
	for parseIndex := range parseItems {
		if !parseEmitted[parseIndex] {
			parseOrder = append(parseOrder, parseIndex)
		}
	}
	return parseOrder
}

// replayItems starts and ends one OpenTelemetry span per item with explicit
// timestamps, parenting items on each other where ids match.
func (parseBridge *Bridge) replayItems(parseCtx context.Context, parseItems []replayItem) error {
	parseExported := map[string]trace.SpanContext{}
	for _, parseIndex := range orderParentsFirst(parseItems) {
		if parseErr := parseCtx.Err(); parseErr != nil {
			return parseErr
		}
		parseItem := parseItems[parseIndex]
		parseParentSC, parseLinked := parseExported[parseItem.parentID]
		if !parseLinked || parseItem.parentID == "" {
			parseParentID := buildVirtualParent(parseItem.traceID)
			if interop.IsValidSpanID(parseItem.parentID) {
				parseParentID = parseItem.parentID
			}
			parseParentSC, _ = buildParentContext(parseItem.traceID, parseParentID)
		}
		parseSpanCtx := trace.ContextWithSpanContext(parseCtx, parseParentSC)
		_, parseSpan := parseBridge.tracer.Start(parseSpanCtx, parseItem.name,
			trace.WithTimestamp(parseItem.start),
			trace.WithSpanKind(parseItem.kind),
			trace.WithAttributes(convertAttributes(gwcattrs.NormalizeErrorAttrs(parseItem.attrs, parseItem.failedMsg))...))
		if parseItem.failed {
			parseMessage, _ := redactString(gwcattrs.KeyExceptionMessage, parseItem.failedMsg)
			parseSpan.SetStatus(codes.Error, parseMessage)
		}
		parseSpan.End(trace.WithTimestamp(parseItem.end))
		if parseItem.selfID != "" {
			parseExported[parseItem.selfID] = parseSpan.SpanContext()
		}
	}
	return nil
}

// ExportRecorderSpans replays the finished spans of parseRecorder into the
// bridge's TracerProvider. Recorder timestamps are on the recording thread's own
// clock, so parseOrigin is the wall-clock time of that clock's zero; when it is
// zero the origin is chosen so the newest span ends now. Open spans are skipped.
// Spans sharing a correlation id share a trace; a span whose parent was dropped
// from the recorder becomes a root of that trace.
func (parseBridge *Bridge) ExportRecorderSpans(parseCtx context.Context, parseRecorder *gwctrace.Recorder, parseOrigin time.Time) error {
	parseSpans := parseRecorder.Spans()
	var parseMaxEnd int64
	for _, parseSpan := range parseSpans {
		parseMaxEnd = max(parseMaxEnd, parseSpan.EndNanos)
	}
	if parseOrigin.IsZero() {
		parseOrigin = time.Now().Add(-time.Duration(parseMaxEnd))
	}
	parseItems := make([]replayItem, 0, len(parseSpans))
	for parseIndex, parseSpan := range parseSpans {
		if parseSpan.Open() {
			continue
		}
		parseTrace := ResolveTraceID(string(parseSpan.Correlation), 0, "")
		if parseSpan.Correlation == "" {
			parseTrace = ResolveTraceID("", parseIndex, parseSpan.Name)
		}
		parseAttrs := map[string]string{"gwc.thread": string(parseSpan.Thread)}
		if parseSpan.Correlation != "" {
			parseAttrs[gwcattrs.KeySSRCorrelationID] = string(parseSpan.Correlation)
		}
		parseItems = append(parseItems, replayItem{
			name:      parseSpan.Name,
			traceID:   parseTrace,
			selfID:    string(parseSpan.ID),
			parentID:  string(parseSpan.Parent),
			start:     parseOrigin.Add(time.Duration(parseSpan.StartNanos)),
			end:       parseOrigin.Add(time.Duration(parseSpan.EndNanos)),
			kind:      trace.SpanKindInternal,
			attrs:     parseAttrs,
			failed:    parseSpan.Err != "",
			failedMsg: parseSpan.Err,
		})
	}
	return parseBridge.replayItems(parseCtx, parseItems)
}

// ExportRUMEvents replays browser RUM events into the bridge's TracerProvider.
// Event ids that are not valid W3C ids are normalized exactly as the OTLP JSON
// exporter does, so the same event lands in the same trace through either path.
// The SDK assigns span ids itself, so an event's own valid span id is kept in the
// gwc.event.span_id attribute and used to link replayed children to parents.
func (parseBridge *Bridge) ExportRUMEvents(parseCtx context.Context, parseEvents []telemetry.RUMEvent) error {
	parseItems := make([]replayItem, 0, len(parseEvents))
	for parseIndex, parseEvent := range parseEvents {
		parseStart := parseEvent.Timestamp
		if parseStart.IsZero() {
			parseStart = time.Now()
		}
		parseAttrs := make(map[string]string, len(parseEvent.Attributes)+1)
		for parseKey, parseValue := range parseEvent.Attributes {
			parseAttrs[parseKey] = parseValue
		}
		parseSelf := ""
		if interop.IsValidSpanID(parseEvent.SpanID) {
			parseSelf = parseEvent.SpanID
			parseAttrs["gwc.event.span_id"] = parseEvent.SpanID
		}
		parseFailed := isErrorEvent(parseEvent)
		parseMessage := ""
		if parseFailed {
			parseMessage = getErrorMessage(parseEvent)
		}
		parseItems = append(parseItems, replayItem{
			name:      parseEvent.Name,
			traceID:   ResolveTraceID(parseEvent.TraceID, parseIndex, parseEvent.Name),
			selfID:    parseSelf,
			parentID:  parseEvent.ParentSpanID,
			start:     parseStart,
			end:       parseStart.Add(time.Duration(parseEvent.DurationNs)),
			kind:      getEventKind(parseEvent),
			attrs:     parseAttrs,
			failed:    parseFailed,
			failedMsg: parseMessage,
		})
	}
	return parseBridge.replayItems(parseCtx, parseItems)
}

// getEventKind maps an event to a span kind like the OTLP JSON exporter: ssr is
// server, fetch is client and everything else is internal.
func getEventKind(parseEvent telemetry.RUMEvent) trace.SpanKind {
	switch {
	case parseEvent.Type == "ssr":
		return trace.SpanKindServer
	case parseEvent.Type == "fetch" || strings.HasPrefix(parseEvent.Name, "fetch"):
		return trace.SpanKindClient
	default:
		return trace.SpanKindInternal
	}
}

// isErrorEvent reports whether an event represents a failure.
func isErrorEvent(parseEvent telemetry.RUMEvent) bool {
	if parseEvent.Type == "error" {
		return true
	}
	parseAttrs := parseEvent.Attributes
	return parseAttrs["gwc.severity"] == "error" ||
		parseAttrs[gwcattrs.KeySSREventPhase] == "error" ||
		parseAttrs[gwcattrs.KeySSRHydrationFailed] == "true" ||
		parseAttrs[gwcattrs.KeyExceptionType] != "" ||
		parseAttrs[gwcattrs.KeyErrorCode] != ""
}

// getErrorMessage picks the raw status message; callers redact it before export.
func getErrorMessage(parseEvent telemetry.RUMEvent) string {
	for _, parseKey := range []string{gwcattrs.KeyExceptionMessage, "gwc.message", gwcattrs.KeySSRHydrationFailure} {
		if parseValue := parseEvent.Attributes[parseKey]; parseValue != "" {
			return parseValue
		}
	}
	return parseEvent.Name
}
