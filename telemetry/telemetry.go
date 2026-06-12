package telemetry

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/devtools"
	"github.com/monstercameron/GoWebComponents/ui"
)

type RUMEvent struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Timestamp  time.Time         `json:"timestamp"`
	TraceID    string            `json:"traceId,omitempty"`
	SpanID     string            `json:"spanId,omitempty"`
	DurationNs int64             `json:"durationNs,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type ExportOptions struct {
	ServiceName string
}

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

// BuildOTLPJSON builds an OTLP/HTTP JSON trace export request body.
func BuildOTLPJSON(parseEvents []RUMEvent, parseOptions ExportOptions) ([]byte, error) {
	parseService := strings.TrimSpace(parseOptions.ServiceName)
	if parseService == "" {
		parseService = "gwc-browser"
	}
	parseSpans := make([]map[string]any, 0, len(parseEvents))
	for parseIndex, parseEvent := range parseEvents {
		parseSpans = append(parseSpans, map[string]any{
			"traceId":           traceID(parseEvent.TraceID, parseIndex),
			"spanId":            spanID(parseEvent.SpanID, parseIndex),
			"name":              parseEvent.Name,
			"kind":              3,
			"startTimeUnixNano": unixNanoString(resolveTimestamp(parseEvent.Timestamp)),
			"endTimeUnixNano":   unixNanoString(resolveTimestamp(parseEvent.Timestamp).Add(time.Duration(parseEvent.DurationNs))),
			"attributes":        otlpAttributes(parseEvent.Attributes),
		})
	}
	parseBody := map[string]any{
		"resourceSpans": []map[string]any{{
			"resource": map[string]any{
				"attributes": []map[string]any{{
					"key": "service.name",
					"value": map[string]string{
						"stringValue": parseService,
					},
				}},
			},
			"scopeSpans": []map[string]any{{
				"scope": map[string]string{
					"name": "github.com/monstercameron/GoWebComponents/telemetry",
				},
				"spans": parseSpans,
			}},
		}},
	}
	return json.Marshal(parseBody)
}

// ExportOTLPHTTP posts an OTLP/HTTP JSON payload to endpoint.
func ExportOTLPHTTP(parseClient *http.Client, parseEndpoint string, parseEvents []RUMEvent, parseOptions ExportOptions) error {
	parseBody, parseErr := BuildOTLPJSON(parseEvents, parseOptions)
	if parseErr != nil {
		return parseErr
	}
	if parseClient == nil {
		parseClient = http.DefaultClient
	}
	parseReq, parseErr := http.NewRequest(http.MethodPost, parseEndpoint, bytes.NewReader(parseBody))
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

func otlpAttributes(parseValues map[string]string) []map[string]any {
	parseAttrs := make([]map[string]any, 0, len(parseValues))
	for parseKey, parseValue := range parseValues {
		if strings.TrimSpace(parseKey) == "" {
			continue
		}
		parseAttrs = append(parseAttrs, map[string]any{
			"key": parseKey,
			"value": map[string]string{
				"stringValue": parseValue,
			},
		})
	}
	return parseAttrs
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

func traceID(parseValue string, parseIndex int) string {
	parseValue = strings.TrimSpace(parseValue)
	if len(parseValue) == 32 {
		return parseValue
	}
	return "000000000000000000000000" + leftPadHex(parseIndex+1, 8)
}

func spanID(parseValue string, parseIndex int) string {
	parseValue = strings.TrimSpace(parseValue)
	if len(parseValue) == 16 {
		return parseValue
	}
	return leftPadHex(parseIndex+1, 16)
}

func leftPadHex(parseValue int, parseWidth int) string {
	const parseDigits = "0123456789abcdef"
	parseOut := make([]byte, parseWidth)
	for parseIndex := parseWidth - 1; parseIndex >= 0; parseIndex-- {
		parseOut[parseIndex] = parseDigits[parseValue&15]
		parseValue >>= 4
	}
	return string(parseOut)
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
