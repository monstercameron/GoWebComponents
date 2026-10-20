package telemetry

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v7/internal/telemetryredaction"
	"github.com/monstercameron/GoWebComponents/v7/interop"
)

const (
	testTraceID  = "0af7651916cd43dd8448eb211c80319c"
	testSpanID   = "b7ad6b7169203331"
	testParentID = "00f067aa0ba902b7"
)

func getGoldenEvents() []RUMEvent {
	parseWhen := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	return []RUMEvent{
		{Name: "commit", Type: "profile", Timestamp: parseWhen, TraceID: testTraceID, SpanID: testSpanID, DurationNs: 1200,
			Attributes: map[string]string{"gwc.domain": "runtime", "gwc.component": "App"}},
		{Name: "child.render", Type: "profile", Timestamp: parseWhen.Add(time.Millisecond), TraceID: testTraceID, ParentSpanID: testSpanID, DurationNs: 500,
			Attributes: map[string]string{"gwc.render.mode": "client"}},
		{Name: "runtime.hydration", Type: "ssr", Timestamp: parseWhen.Add(2 * time.Millisecond), TraceID: "correlation-1", DurationNs: 4200,
			Attributes: map[string]string{"gwc.ssr.hydration.failed": "true", "gwc.ssr.hydration.failure": "text mismatch"}},
		{Name: "fetch.users", Type: "fetch", Timestamp: parseWhen.Add(3 * time.Millisecond), TraceID: testTraceID, ParentSpanID: "0000000000000000", DurationNs: 90,
			Attributes: map[string]string{"gwc.target": "/api/users"}},
	}
}

func TestBuildOTLPJSONMatchesGolden(parseT *testing.T) {
	telemetryredaction.Configure(telemetryredaction.Policy{})
	parsePayload, parseErr := BuildOTLPJSON(getGoldenEvents(), ExportOptions{ServiceName: "demo", ServiceVersion: "1.2.3", Environment: "test", ScopeVersion: "6.0.0"})
	if parseErr != nil {
		parseT.Fatalf("BuildOTLPJSON: %v", parseErr)
	}
	var parseIndented bytes.Buffer
	if parseErr := json.Indent(&parseIndented, parsePayload, "", "  "); parseErr != nil {
		parseT.Fatalf("indent: %v", parseErr)
	}
	parseWant, parseErr := os.ReadFile(filepath.Join("testdata", "golden_otlp.json"))
	if parseErr != nil {
		parseT.Fatalf("read golden: %v\nactual:\n%s", parseErr, parseIndented.String())
	}
	parseGot := strings.TrimSpace(parseIndented.String())
	parseExpected := strings.TrimSpace(strings.ReplaceAll(string(parseWant), "\r\n", "\n"))
	if parseGot != parseExpected {
		parseT.Fatalf("golden mismatch\nactual:\n%s", parseGot)
	}
	parsePayload2, _ := BuildOTLPJSON(getGoldenEvents(), ExportOptions{ServiceName: "demo", ServiceVersion: "1.2.3", Environment: "test", ScopeVersion: "6.0.0"})
	if !bytes.Equal(parsePayload, parsePayload2) {
		parseT.Fatal("output is not deterministic")
	}
}

func parseFirstSpan(parseT *testing.T, parsePayload []byte) map[string]any {
	parseT.Helper()
	var parseDoc struct {
		ResourceSpans []struct {
			ScopeSpans []struct {
				Spans []map[string]any `json:"spans"`
			} `json:"scopeSpans"`
		} `json:"resourceSpans"`
	}
	if parseErr := json.Unmarshal(parsePayload, &parseDoc); parseErr != nil {
		parseT.Fatalf("unmarshal: %v", parseErr)
	}
	return parseDoc.ResourceSpans[0].ScopeSpans[0].Spans[0]
}

func TestBuildOTLPJSONRejectsInvalidIDs(parseT *testing.T) {
	telemetryredaction.Configure(telemetryredaction.Policy{})
	for _, parseCase := range []struct{ name, trace, span string }{
		{"zero trace", strings.Repeat("0", 32), strings.Repeat("0", 16)},
		{"non hex", strings.Repeat("z", 32), strings.Repeat("g", 16)},
		{"uppercase", strings.ToUpper(testTraceID), strings.ToUpper(testSpanID)},
		{"wrong length", "abc123", "abc"},
		{"empty", "", ""},
	} {
		parsePayload, parseErr := BuildOTLPJSON([]RUMEvent{{Name: "x", TraceID: parseCase.trace, SpanID: parseCase.span}}, ExportOptions{})
		if parseErr != nil {
			parseT.Fatalf("%s: %v", parseCase.name, parseErr)
		}
		parseSpan := parseFirstSpan(parseT, parsePayload)
		parseTrace, _ := parseSpan["traceId"].(string)
		parseSpanID, _ := parseSpan["spanId"].(string)
		if !interop.IsValidTraceID(parseTrace) || !interop.IsValidSpanID(parseSpanID) {
			parseT.Fatalf("%s: invalid ids emitted %q %q", parseCase.name, parseTrace, parseSpanID)
		}
		if parseTrace == parseCase.trace || parseSpanID == parseCase.span {
			parseT.Fatalf("%s: invalid input id was passed through", parseCase.name)
		}
	}
}

func TestBuildOTLPJSONSharedCorrelationSharesTrace(parseT *testing.T) {
	parsePayload, _ := BuildOTLPJSON([]RUMEvent{{Name: "a", TraceID: "req-7"}, {Name: "b", TraceID: "req-7"}}, ExportOptions{})
	var parseDoc map[string]any
	_ = json.Unmarshal(parsePayload, &parseDoc)
	parseSpans := parseDoc["resourceSpans"].([]any)[0].(map[string]any)["scopeSpans"].([]any)[0].(map[string]any)["spans"].([]any)
	parseA, parseB := parseSpans[0].(map[string]any), parseSpans[1].(map[string]any)
	if parseA["traceId"] != parseB["traceId"] || parseA["spanId"] == parseB["spanId"] {
		parseT.Fatalf("shared correlation must share trace with distinct spans: %v %v", parseA, parseB)
	}
}

func TestBuildOTLPJSONErrorStatus(parseT *testing.T) {
	telemetryredaction.Configure(telemetryredaction.Policy{})
	parsePayload, _ := BuildOTLPJSON([]RUMEvent{{Name: "boom", Type: "error", Attributes: map[string]string{"exception.message": "kaput"}}}, ExportOptions{})
	parseStatus, _ := parseFirstSpan(parseT, parsePayload)["status"].(map[string]any)
	if parseStatus["code"] != float64(2) || parseStatus["message"] != "kaput" {
		parseT.Fatalf("status = %#v", parseStatus)
	}
	parsePayload, _ = BuildOTLPJSON([]RUMEvent{{Name: "fine", Type: "profile"}}, ExportOptions{})
	if _, parseHas := parseFirstSpan(parseT, parsePayload)["status"]; parseHas {
		parseT.Fatal("non-error span carries a status")
	}
}

func TestBuildOTLPJSONRedactsAttributesAndStatusMessage(parseT *testing.T) {
	telemetryredaction.Configure(telemetryredaction.Policy{Fields: []string{"gwc.secret", "exception.message", "gwc.broken"}, Redact: func(parseCtx telemetryredaction.Context) (any, error) {
		if parseCtx.Field == "gwc.broken" {
			return nil, errors.New("redactor failed")
		}
		return "[masked]", nil
	}})
	parseT.Cleanup(func() { telemetryredaction.Configure(telemetryredaction.Policy{}) })
	parsePayload, _ := BuildOTLPJSON([]RUMEvent{{Name: "boom", Type: "error", Attributes: map[string]string{
		"gwc.secret": "hunter2", "exception.message": "token=abc", "gwc.broken": "leak", "gwc.public": "ok",
	}}}, ExportOptions{})
	parseText := string(parsePayload)
	for _, parseLeak := range []string{"hunter2", "token=abc", "leak"} {
		if strings.Contains(parseText, parseLeak) {
			parseT.Fatalf("payload leaked %q: %s", parseLeak, parseText)
		}
	}
	if !strings.Contains(parseText, "[masked]") || !strings.Contains(parseText, `"gwc.public"`) {
		parseT.Fatalf("expected masked value and public attribute: %s", parseText)
	}
	if strings.Contains(parseText, `"gwc.broken"`) {
		parseT.Fatalf("attribute with failing redactor was not dropped: %s", parseText)
	}
}
