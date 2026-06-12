package logging

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"
)

type logContextDetails struct {
	correlationID string
	traceContext  TraceContext
}

type logLevelDetails struct {
	canonicalLevel string
	severityText   string
	severityNumber int
	consoleMethod  string
}

var reservedRecordKeys = map[string]struct{}{
	"attributes":      {},
	"correlation_id":  {},
	"level":           {},
	"message":         {},
	"scope":           {},
	"severity_number": {},
	"severity_text":   {},
	"span_id":         {},
	"timestamp":       {},
	"trace_id":        {},
	"traceparent":     {},
	"tracestate":      {},
}

// buildLogFields normalizes logger arguments into one structured field map.
func buildLogFields(parseLogArgs []any) map[string]any {
	var parseFields map[string]any
	for parseIndex := 0; parseIndex < len(parseLogArgs); parseIndex++ {
		parseArg := parseLogArgs[parseIndex]
		if parseArg == nil {
			continue
		}

		switch parseTyped := parseArg.(type) {
		case map[string]any:
			for parseFieldKey, parseFieldValue := range parseTyped {
				parseFields = storeLogFieldValue(parseFields, parseFieldKey, parseFieldValue)
			}
			continue
		case map[string]string:
			for parseFieldKey, parseFieldValue := range parseTyped {
				parseFields = storeLogFieldValue(parseFields, parseFieldKey, parseFieldValue)
			}
			continue
		case slog.Attr:
			parseFields = buildLogAttrFields(parseFields, parseTyped)
			continue
		case []slog.Attr:
			for _, parseAttr := range parseTyped {
				parseFields = buildLogAttrFields(parseFields, parseAttr)
			}
			continue
		case string:
			parseFieldKey := strings.TrimSpace(parseTyped)
			if parseFieldKey != "" && parseIndex+1 < len(parseLogArgs) {
				parseFields = storeLogFieldValue(parseFields, parseFieldKey, parseLogArgs[parseIndex+1])
				parseIndex++
				continue
			}
		case error:
			parseFields = storeLogFieldValue(parseFields, "error", parseTyped.Error())
			continue
		}

		parseFields = storeLogFieldValue(parseFields, buildArgumentFieldKey(parseIndex), parseArg)
	}
	return parseFields
}

// buildLogAttrFields stores one slog attribute inside the structured field map.
func buildLogAttrFields(parseFields map[string]any, parseAttr slog.Attr) map[string]any {
	parseFieldKey := strings.TrimSpace(parseAttr.Key)
	if parseFieldKey == "" {
		return parseFields
	}
	return storeLogFieldValue(parseFields, parseFieldKey, buildSlogValue(parseAttr.Value))
}

// storeLogFieldValue writes one normalized field value into the structured field map.
func storeLogFieldValue(parseFields map[string]any, parseFieldKey string, parseFieldValue any) map[string]any {
	parseStoredKey := strings.TrimSpace(parseFieldKey)
	if parseStoredKey == "" {
		return parseFields
	}
	if parseFields == nil {
		parseFields = make(map[string]any)
	}
	parseFields[parseStoredKey] = normalizeLogValue(parseFieldValue)
	return parseFields
}

// buildArgumentFieldKey returns one stable key for one positional logging argument.
func buildArgumentFieldKey(parseIndex int) string {
	return fmt.Sprintf("arg_%d", parseIndex)
}

// protectLogFieldKey prevents user-supplied fields from clobbering core record keys.
func protectLogFieldKey(parseFieldKey string) string {
	parseTrimmedKey := strings.TrimSpace(parseFieldKey)
	if parseTrimmedKey == "" {
		return ""
	}
	if _, parseReserved := reservedRecordKeys[parseTrimmedKey]; parseReserved {
		return "field_" + parseTrimmedKey
	}
	return parseTrimmedKey
}

// buildLogRecord creates one structured log record enriched with context-backed metadata.
func buildLogRecord(parseCtx context.Context, parseLogLevel, parseLogScope, parseLogMessage string, parseLogFields map[string]any) map[string]any {
	parseLevelDetails := buildLogLevelDetails(parseLogLevel)
	parseContextDetails := resolveLogContextDetails(parseCtx)
	parseAttributes := buildLogAttributes(parseLogFields)

	parseRecord := map[string]any{
		"attributes":      parseAttributes,
		"level":           parseLevelDetails.canonicalLevel,
		"message":         strings.TrimSpace(parseLogMessage),
		"scope":           strings.TrimSpace(parseLogScope),
		"severity_number": parseLevelDetails.severityNumber,
		"severity_text":   parseLevelDetails.severityText,
		"timestamp":       time.Now().UTC().Format(time.RFC3339Nano),
	}
	if parseContextDetails.correlationID != "" {
		parseRecord["correlation_id"] = parseContextDetails.correlationID
	}
	if parseTraceID := strings.TrimSpace(parseContextDetails.traceContext.TraceID); parseTraceID != "" {
		parseRecord["trace_id"] = parseTraceID
	}
	if parseSpanID := strings.TrimSpace(parseContextDetails.traceContext.SpanID); parseSpanID != "" {
		parseRecord["span_id"] = parseSpanID
	}
	if parseTraceparent := parseContextDetails.traceContext.Traceparent(); parseTraceparent != "" {
		parseRecord["traceparent"] = parseTraceparent
	}
	if parseTracestate := strings.TrimSpace(parseContextDetails.traceContext.TraceState); parseTracestate != "" {
		parseRecord["tracestate"] = parseTracestate
	}
	for parseFieldKey, parseFieldValue := range parseAttributes {
		parseStoredKey := protectLogFieldKey(parseFieldKey)
		if parseStoredKey == "" {
			continue
		}
		parseRecord[parseStoredKey] = parseFieldValue
	}
	return parseRecord
}

// buildLogLevelDetails normalizes one free-form log level into record metadata.
func buildLogLevelDetails(parseLogLevel string) logLevelDetails {
	parseNormalizedLevel := strings.ToLower(strings.TrimSpace(parseLogLevel))
	switch parseNormalizedLevel {
	case "trace":
		return logLevelDetails{canonicalLevel: "trace", severityText: "TRACE", severityNumber: 1, consoleMethod: "trace"}
	case "debug":
		return logLevelDetails{canonicalLevel: "debug", severityText: "DEBUG", severityNumber: 5, consoleMethod: "debug"}
	case "", "info", "log":
		return logLevelDetails{canonicalLevel: "info", severityText: "INFO", severityNumber: 9, consoleMethod: "info"}
	case "warn", "warning":
		return logLevelDetails{canonicalLevel: "warn", severityText: "WARN", severityNumber: 13, consoleMethod: "warn"}
	case "error":
		return logLevelDetails{canonicalLevel: "error", severityText: "ERROR", severityNumber: 17, consoleMethod: "error"}
	default:
		return logLevelDetails{
			canonicalLevel: parseNormalizedLevel,
			severityText:   strings.ToUpper(parseNormalizedLevel),
			severityNumber: 9,
			consoleMethod:  "log",
		}
	}
}

// resolveLogContextDetails merges logging-owned and framework-provided context metadata.
func resolveLogContextDetails(parseCtx context.Context) logContextDetails {
	if parseCtx == nil {
		return logContextDetails{}
	}

	parseDetails := logContextDetails{
		correlationID: GetCorrelationID(parseCtx),
	}
	if parseTraceContext, parseOk := GetTraceContext(parseCtx); parseOk {
		parseDetails.traceContext = parseTraceContext
	}

	parseExternalDetails := resolveExternalContextDetails(parseCtx)
	if parseDetails.correlationID == "" {
		parseDetails.correlationID = parseExternalDetails.correlationID
	}
	parseDetails.traceContext = buildMergedTraceContext(parseDetails.traceContext, parseExternalDetails.traceContext)
	if parseDetails.correlationID == "" {
		parseDetails.correlationID = strings.TrimSpace(parseDetails.traceContext.TraceID)
	}
	return parseDetails
}

// buildMergedTraceContext fills missing trace fields from one fallback context.
func buildMergedTraceContext(parseTraceContext TraceContext, parseFallback TraceContext) TraceContext {
	if strings.TrimSpace(parseTraceContext.TraceID) == "" {
		parseTraceContext.TraceID = strings.TrimSpace(parseFallback.TraceID)
	}
	if strings.TrimSpace(parseTraceContext.ParentSpanID) == "" {
		parseTraceContext.ParentSpanID = strings.TrimSpace(parseFallback.ParentSpanID)
	}
	if strings.TrimSpace(parseTraceContext.SpanID) == "" {
		parseTraceContext.SpanID = strings.TrimSpace(parseFallback.SpanID)
	}
	if strings.TrimSpace(parseTraceContext.Flags) == "" {
		parseTraceContext.Flags = strings.TrimSpace(parseFallback.Flags)
	}
	if strings.TrimSpace(parseTraceContext.TraceState) == "" {
		parseTraceContext.TraceState = strings.TrimSpace(parseFallback.TraceState)
	}
	return parseTraceContext
}

// buildLogAttributes normalizes user-supplied logging fields into one JSON-safe attribute map.
func buildLogAttributes(parseLogFields map[string]any) map[string]any {
	parseAttributes := make(map[string]any, len(parseLogFields))
	for parseFieldKey, parseFieldValue := range parseLogFields {
		parseStoredKey := strings.TrimSpace(parseFieldKey)
		if parseStoredKey == "" {
			continue
		}
		parseAttributes[parseStoredKey] = normalizeLogValue(parseFieldValue)
	}
	return parseAttributes
}

// buildSlogValue resolves one slog value into one JSON-safe logging value.
func buildSlogValue(parseValue slog.Value) any {
	parseResolvedValue := parseValue.Resolve()
	switch parseResolvedValue.Kind() {
	case slog.KindBool:
		return parseResolvedValue.Bool()
	case slog.KindDuration:
		return parseResolvedValue.Duration().String()
	case slog.KindFloat64:
		return parseResolvedValue.Float64()
	case slog.KindInt64:
		return parseResolvedValue.Int64()
	case slog.KindString:
		return parseResolvedValue.String()
	case slog.KindTime:
		return parseResolvedValue.Time().UTC().Format(time.RFC3339Nano)
	case slog.KindUint64:
		return parseResolvedValue.Uint64()
	case slog.KindGroup:
		parseGroupFields := make(map[string]any, len(parseResolvedValue.Group()))
		for _, parseAttr := range parseResolvedValue.Group() {
			parseFieldKey := strings.TrimSpace(parseAttr.Key)
			if parseFieldKey == "" {
				continue
			}
			parseGroupFields[parseFieldKey] = buildSlogValue(parseAttr.Value)
		}
		return parseGroupFields
	case slog.KindAny:
		return normalizeLogValue(parseResolvedValue.Any())
	default:
		return normalizeLogValue(parseResolvedValue.Any())
	}
}

// normalizeLogValue converts one arbitrary value into one JSON-safe logging value.
func normalizeLogValue(parseValue any) any {
	switch parseTyped := parseValue.(type) {
	case nil:
		return nil
	case string:
		return parseTyped
	case bool:
		return parseTyped
	case int:
		return parseTyped
	case int8:
		return parseTyped
	case int16:
		return parseTyped
	case int32:
		return parseTyped
	case int64:
		return parseTyped
	case uint:
		return parseTyped
	case uint8:
		return parseTyped
	case uint16:
		return parseTyped
	case uint32:
		return parseTyped
	case uint64:
		return parseTyped
	case float32:
		return parseTyped
	case float64:
		return parseTyped
	case time.Time:
		return parseTyped.UTC().Format(time.RFC3339Nano)
	case time.Duration:
		return parseTyped.String()
	case error:
		return parseTyped.Error()
	// slog.Attr implements fmt.Stringer, so it must be matched before the
	// Stringer case or Attrs collapse to their string form.
	case slog.Attr:
		return map[string]any{strings.TrimSpace(parseTyped.Key): buildSlogValue(parseTyped.Value)}
	case fmt.Stringer:
		return parseTyped.String()
	case []any:
		parseSlice := make([]any, 0, len(parseTyped))
		for _, parseItem := range parseTyped {
			parseSlice = append(parseSlice, normalizeLogValue(parseItem))
		}
		return parseSlice
	case map[string]any:
		parseMap := make(map[string]any, len(parseTyped))
		for parseFieldKey, parseFieldValue := range parseTyped {
			parseMap[strings.TrimSpace(parseFieldKey)] = normalizeLogValue(parseFieldValue)
		}
		return parseMap
	case map[string]string:
		parseMap := make(map[string]any, len(parseTyped))
		for parseFieldKey, parseFieldValue := range parseTyped {
			parseMap[strings.TrimSpace(parseFieldKey)] = parseFieldValue
		}
		return parseMap
	}

	parseValueReflect := reflect.ValueOf(parseValue)
	if !parseValueReflect.IsValid() {
		return nil
	}
	switch parseValueReflect.Kind() {
	case reflect.Interface, reflect.Pointer:
		if parseValueReflect.IsNil() {
			return nil
		}
		return normalizeLogValue(parseValueReflect.Elem().Interface())
	case reflect.Map:
		return buildMapValue(parseValueReflect)
	case reflect.Slice, reflect.Array:
		return buildSliceValue(parseValueReflect)
	default:
		return fmt.Sprint(parseValue)
	}
}

// buildMapValue converts one reflected map into one JSON-safe logging object.
func buildMapValue(parseValue reflect.Value) map[string]any {
	if parseValue.Kind() != reflect.Map || (parseValue.Kind() == reflect.Map && parseValue.IsNil()) {
		return nil
	}
	parseMap := make(map[string]any, parseValue.Len())
	for _, parseMapKey := range parseValue.MapKeys() {
		parseFieldKey := strings.TrimSpace(fmt.Sprint(parseMapKey.Interface()))
		if parseFieldKey == "" {
			continue
		}
		parseMap[parseFieldKey] = normalizeLogValue(parseValue.MapIndex(parseMapKey).Interface())
	}
	return parseMap
}

// buildSliceValue converts one reflected slice or array into one JSON-safe logging array.
func buildSliceValue(parseValue reflect.Value) []any {
	if (parseValue.Kind() != reflect.Slice && parseValue.Kind() != reflect.Array) || (parseValue.Kind() == reflect.Slice && parseValue.IsNil()) {
		return nil
	}
	parseSlice := make([]any, 0, parseValue.Len())
	for parseIndex := 0; parseIndex < parseValue.Len(); parseIndex++ {
		parseSlice = append(parseSlice, normalizeLogValue(parseValue.Index(parseIndex).Interface()))
	}
	return parseSlice
}
