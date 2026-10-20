package telemetry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v7/devtools"
	"github.com/monstercameron/GoWebComponents/v7/internal/gwcattrs"
	"github.com/monstercameron/GoWebComponents/v7/internal/telemetryredaction"
	"github.com/monstercameron/GoWebComponents/v7/interop"
	"github.com/monstercameron/GoWebComponents/v7/ui"
)

type RUMEvent struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	TraceID   string    `json:"traceId,omitempty"`
	SpanID    string    `json:"spanId,omitempty"`
	// ParentSpanID is the span id of the parent span; it is exported only when valid.
	ParentSpanID string            `json:"parentSpanId,omitempty"`
	DurationNs   int64             `json:"durationNs,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
}

// ExportOptions configures the OTLP resource and scope attached to an export.
type ExportOptions struct {
	ServiceName string
	// ServiceVersion is exported as service.version when non-empty.
	ServiceVersion string
	// Environment is exported as deployment.environment when non-empty.
	Environment string
	// ScopeVersion is exported as the instrumentation scope version when non-empty.
	ScopeVersion string
}

const (
	scopeName        = "github.com/monstercameron/GoWebComponents/v7/telemetry"
	otlpKindInternal = 1
	otlpKindServer   = 2
	otlpKindClient   = 3
	otlpStatusError  = 2
)

// EventsFromSnapshot extracts browser RUM events from a devtools snapshot.
func EventsFromSnapshot(parseSnapshot devtools.Snapshot) []RUMEvent {
	parseEvents := make([]RUMEvent, 0, len(parseSnapshot.Profiling.RecentEvents)+len(parseSnapshot.Diagnostics)+len(parseSnapshot.Logs))
	for _, parseEvent := range parseSnapshot.Profiling.RecentEvents {
		parseEvents = append(parseEvents, RUMEvent{
			Name:       parseEvent.Name,
			Type:       "profile",
			Timestamp:  parseTime(parseEvent.Timestamp),
			TraceID:    parseEvent.CorrelationID,
			DurationNs: parseEvent.DurationNs,
			Attributes: mergeAttrs(map[string]string{
				"gwc.domain": parseEvent.Domain,
				"gwc.phase":  parseEvent.Phase,
				"gwc.target": parseEvent.Target,
			}, parseEvent.Fields),
		})
	}
	for _, parseDiagnostic := range parseSnapshot.Diagnostics {
		parseEvents = append(parseEvents, RUMEvent{
			Name: "diagnostic." + parseDiagnostic.Code,
			Type: "diagnostic",
			Attributes: map[string]string{
				"gwc.source":   parseDiagnostic.Source,
				"gwc.severity": string(parseDiagnostic.Severity),
				"gwc.message":  parseDiagnostic.Message,
			},
		})
	}
	for _, parseLog := range parseSnapshot.Logs {
		parseEvents = append(parseEvents, RUMEvent{
			Name:      "log." + parseLog.Domain,
			Type:      "log",
			Timestamp: parseTime(parseLog.Timestamp),
			TraceID:   parseLog.CorrelationID,
			Attributes: mergeAttrs(map[string]string{
				"gwc.level":   string(parseLog.Level),
				"gwc.message": parseLog.Message,
			}, parseLog.Fields),
		})
	}
	return parseEvents
}

// EventFromSSRObservation converts SSR/hydration observations into the same RUM stream.
func EventFromSSRObservation(parseObservation ui.SSRObservation) RUMEvent {
	parseAttrs := ui.GetSSRObservationAttributes(parseObservation)
	return RUMEvent{
		Name:       parseObservation.Name,
		Type:       "ssr",
		Timestamp:  parseObservation.Timestamp,
		TraceID:    parseObservation.CorrelationID,
		DurationNs: durationFromSSR(parseObservation),
		Attributes: parseAttrs,
	}
}

// BuildOTLPJSON builds an OTLP/HTTP JSON trace export request body. Ids that are not valid
// W3C ids are replaced by deterministic hash-derived ids so the output is stable and always
// accepted by collectors. Attribute values pass through the telemetry redaction policy.
func BuildOTLPJSON(parseEvents []RUMEvent, parseOptions ExportOptions) ([]byte, error) {
	parseService := strings.TrimSpace(parseOptions.ServiceName)
	if parseService == "" {
		parseService = "gwc-browser"
	}
	parseSpans := make([]map[string]any, 0, len(parseEvents))
	for parseIndex, parseEvent := range parseEvents {
		parseAttrs := buildEventAttributes(parseEvent)
		parseTrace := resolveTraceID(parseEvent.TraceID, parseIndex, parseEvent.Name)
		parseSpan := map[string]any{
			"traceId":           parseTrace,
			"spanId":            resolveSpanID(parseEvent.SpanID, parseTrace, parseIndex, parseEvent.Name),
			"name":              parseEvent.Name,
			"kind":              getSpanKind(parseEvent),
			"startTimeUnixNano": unixNanoString(resolveTimestamp(parseEvent.Timestamp)),
			"endTimeUnixNano":   unixNanoString(resolveTimestamp(parseEvent.Timestamp).Add(time.Duration(parseEvent.DurationNs))),
			"attributes":        otlpAttributes(parseAttrs),
		}
		if interop.IsValidSpanID(parseEvent.ParentSpanID) {
			parseSpan["parentSpanId"] = parseEvent.ParentSpanID
		}
		if isErrorEvent(parseEvent) {
			parseSpan["status"] = map[string]any{"code": otlpStatusError, "message": getErrorMessage(parseEvent, parseAttrs)}
		}
		parseSpans = append(parseSpans, parseSpan)
	}
	parseResource := []map[string]any{
		stringAttr("service.name", parseService),
		stringAttr("telemetry.sdk.language", "go"),
		stringAttr("telemetry.sdk.name", "gwc"),
	}
	if parseVersion := strings.TrimSpace(parseOptions.ServiceVersion); parseVersion != "" {
		parseResource = append(parseResource, stringAttr("service.version", parseVersion))
	}
	if parseEnvironment := strings.TrimSpace(parseOptions.Environment); parseEnvironment != "" {
		parseResource = append(parseResource, stringAttr("deployment.environment", parseEnvironment))
	}
	parseScope := map[string]string{"name": scopeName}
	if parseScopeVersion := strings.TrimSpace(parseOptions.ScopeVersion); parseScopeVersion != "" {
		parseScope["version"] = parseScopeVersion
	}
	parseBody := map[string]any{
		"resourceSpans": []map[string]any{{
			"resource": map[string]any{"attributes": parseResource},
			"scopeSpans": []map[string]any{{
				"scope": parseScope,
				"spans": parseSpans,
			}},
		}},
	}
	return json.Marshal(parseBody)
}

// ExportOTLPHTTP posts an OTLP/HTTP JSON payload to endpoint. The context bounds the POST — a stuck
// collector is cancelled via ctx (deadline/cancel) instead of blocking the caller forever.
func ExportOTLPHTTP(parseCtx context.Context, parseClient *http.Client, parseEndpoint string, parseEvents []RUMEvent, parseOptions ExportOptions) error {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	parseBody, parseErr := BuildOTLPJSON(parseEvents, parseOptions)
	if parseErr != nil {
		return parseErr
	}
	if parseClient == nil {
		parseClient = http.DefaultClient
	}
	parseReq, parseErr := http.NewRequestWithContext(parseCtx, http.MethodPost, parseEndpoint, bytes.NewReader(parseBody))
	if parseErr != nil {
		return parseErr
	}
	parseReq.Header.Set("Content-Type", "application/json")
	parseResp, parseErr := parseClient.Do(parseReq)
	if parseErr != nil {
		return parseErr
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode >= 300 {
		return &HTTPStatusError{StatusCode: parseResp.StatusCode}
	}
	return nil
}

type HTTPStatusError struct {
	StatusCode int
}

func (parseErr *HTTPStatusError) Error() string {
	return "gwc telemetry export failed with HTTP status " + intString(parseErr.StatusCode)
}

// buildEventAttributes is the single place event attributes are prepared for export: it
// applies the telemetry redaction policy, which drops any attribute whose redactor fails.
func buildEventAttributes(parseEvent RUMEvent) map[string]string {
	parseAttrs := parseEvent.Attributes
	if isErrorEvent(parseEvent) {
		parseAttrs = gwcattrs.NormalizeErrorAttrs(parseAttrs, getErrorMessage(parseEvent, parseAttrs))
	}
	return telemetryredaction.StringMap("attributes", parseAttrs)
}

// otlpAttributes converts already-redacted attributes to sorted OTLP key-values.
func otlpAttributes(parseValues map[string]string) []map[string]any {
	parseKeys := make([]string, 0, len(parseValues))
	for parseKey := range parseValues {
		if strings.TrimSpace(parseKey) != "" {
			parseKeys = append(parseKeys, parseKey)
		}
	}
	sort.Strings(parseKeys)
	parseAttrs := make([]map[string]any, 0, len(parseKeys))
	for _, parseKey := range parseKeys {
		parseAttrs = append(parseAttrs, stringAttr(parseKey, parseValues[parseKey]))
	}
	return parseAttrs
}

func stringAttr(parseKey string, parseValue string) map[string]any {
	return map[string]any{"key": parseKey, "value": map[string]string{"stringValue": parseValue}}
}

// getSpanKind maps an event to an OTLP span kind: ssr is server, fetch is client and
// render, commit, hydrate and everything else is internal.
func getSpanKind(parseEvent RUMEvent) int {
	switch {
	case parseEvent.Type == "ssr":
		return otlpKindServer
	case parseEvent.Type == "fetch" || strings.HasPrefix(parseEvent.Name, "fetch"):
		return otlpKindClient
	default:
		return otlpKindInternal
	}
}

// isErrorEvent reports whether an event represents a failure that needs an error status.
func isErrorEvent(parseEvent RUMEvent) bool {
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

// getErrorMessage picks the status message from the redacted attributes so redaction applies.
func getErrorMessage(parseEvent RUMEvent, parseRedacted map[string]string) string {
	for _, parseKey := range []string{gwcattrs.KeyExceptionMessage, "gwc.message", gwcattrs.KeySSRHydrationFailure} {
		if parseValue := parseRedacted[parseKey]; parseValue != "" {
			return parseValue
		}
	}
	return parseEvent.Name
}

func mergeAttrs(parseBase map[string]string, parseExtra map[string]string) map[string]string {
	parseOut := make(map[string]string, len(parseBase)+len(parseExtra))
	for parseKey, parseValue := range parseBase {
		parseOut[parseKey] = parseValue
	}
	for parseKey, parseValue := range parseExtra {
		parseOut[parseKey] = parseValue
	}
	return parseOut
}

func parseTime(parseValue string) time.Time {
	parseValue = strings.TrimSpace(parseValue)
	if parseValue == "" {
		return time.Time{}
	}
	if parseT, parseErr := time.Parse(time.RFC3339Nano, parseValue); parseErr == nil {
		return parseT
	}
	return time.Time{}
}

func resolveTimestamp(parseValue time.Time) time.Time {
	if parseValue.IsZero() {
		return time.Unix(0, 0).UTC()
	}
	return parseValue.UTC()
}

func durationFromSSR(parseObservation ui.SSRObservation) int64 {
	if parseObservation.Render != nil {
		return parseObservation.Render.DurationNs
	}
	if parseObservation.Bootstrap != nil {
		return 0
	}
	if parseObservation.Hydration != nil {
		return parseObservation.Hydration.DurationNs
	}
	return 0
}

func unixNanoString(parseValue time.Time) string {
	return int64String(parseValue.UnixNano())
}

// resolveTraceID keeps a valid trace id and otherwise derives a deterministic one from the
// supplied correlation text (so events sharing a correlation id share a trace) or, when none
// was supplied, from the event position and name.
func resolveTraceID(parseValue string, parseIndex int, parseName string) string {
	parseValue = strings.TrimSpace(parseValue)
	if interop.IsValidTraceID(parseValue) {
		return parseValue
	}
	parseSeed := "trace|" + parseValue
	if parseValue == "" {
		parseSeed = "trace||" + intString(parseIndex) + "|" + parseName
	}
	parseSum := sha256.Sum256([]byte(parseSeed))
	return hex.EncodeToString(parseSum[:16])
}

// resolveSpanID keeps a valid span id and otherwise derives a non-zero deterministic one
// from the trace id, event position and name.
func resolveSpanID(parseValue string, parseTrace string, parseIndex int, parseName string) string {
	parseValue = strings.TrimSpace(parseValue)
	if interop.IsValidSpanID(parseValue) {
		return parseValue
	}
	parseSum := sha256.Sum256([]byte("span|" + parseTrace + "|" + intString(parseIndex) + "|" + parseName))
	if isAllZeroBytes(parseSum[:8]) {
		parseSum[7] = 1
	}
	return hex.EncodeToString(parseSum[:8])
}

func isAllZeroBytes(parseValue []byte) bool {
	for _, parseByte := range parseValue {
		if parseByte != 0 {
			return false
		}
	}
	return true
}

func intString(parseValue int) string {
	return int64String(int64(parseValue))
}

func int64String(parseValue int64) string {
	if parseValue == 0 {
		return "0"
	}
	parseNegative := parseValue < 0
	if parseNegative {
		parseValue = -parseValue
	}
	parseDigits := make([]byte, 0, 20)
	for parseValue > 0 {
		parseDigits = append(parseDigits, byte('0'+parseValue%10))
		parseValue /= 10
	}
	if parseNegative {
		parseDigits = append(parseDigits, '-')
	}
	for parseLeft, parseRight := 0, len(parseDigits)-1; parseLeft < parseRight; parseLeft, parseRight = parseLeft+1, parseRight-1 {
		parseDigits[parseLeft], parseDigits[parseRight] = parseDigits[parseRight], parseDigits[parseLeft]
	}
	return string(parseDigits)
}
