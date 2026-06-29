package telemetry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/devtools"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestEventsFromSnapshotAndBuildOTLPJSON(parseT *testing.T) {
	parseEvents := EventsFromSnapshot(devtools.Snapshot{
		Profiling: devtools.Profiling{RecentEvents: []devtools.ProfilingEvent{{
			Domain:        "runtime",
			Name:          "commit",
			Phase:         "finish",
			Target:        "root",
			CorrelationID: "11111111111111111111111111111111",
			DurationNs:    1200,
			Timestamp:     "2026-06-12T12:00:00Z",
		}}},
		Diagnostics: []devtools.Diagnostic{{Source: "runtime", Severity: devtools.SeverityWarning, Code: "GWC-1", Message: "warn"}},
	})
	if len(parseEvents) != 2 {
		parseT.Fatalf("event count = %d, want 2", len(parseEvents))
	}
	parsePayload, parseErr := BuildOTLPJSON(parseEvents, ExportOptions{ServiceName: "app"})
	if parseErr != nil {
		parseT.Fatalf("BuildOTLPJSON returned error: %v", parseErr)
	}
	if !json.Valid(parsePayload) {
		parseT.Fatalf("payload is not valid JSON: %s", string(parsePayload))
	}
	if !strings.Contains(string(parsePayload), `"service.name"`) || !strings.Contains(string(parsePayload), `"commit"`) {
		parseT.Fatalf("payload missing expected OTLP content: %s", string(parsePayload))
	}
}

func TestExportOTLPHTTPPostsJSON(parseT *testing.T) {
	parsePosted := false
	parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parsePosted = true
		if parseR.Method != http.MethodPost {
			parseT.Fatalf("method = %s", parseR.Method)
		}
		if parseR.Header.Get("Content-Type") != "application/json" {
			parseT.Fatalf("content-type = %q", parseR.Header.Get("Content-Type"))
		}
		parseW.WriteHeader(http.StatusAccepted)
	}))
	defer parseServer.Close()

	parseErr := ExportOTLPHTTP(context.Background(), parseServer.Client(), parseServer.URL, []RUMEvent{{Name: "load", Type: "profile"}}, ExportOptions{})
	if parseErr != nil {
		parseT.Fatalf("ExportOTLPHTTP returned error: %v", parseErr)
	}
	if !parsePosted {
		parseT.Fatal("server did not receive export")
	}
}

func TestEventFromSSRObservationPreservesHydrationDurationAndAttributes(parseT *testing.T) {
	parseWhen := time.Date(2026, 6, 13, 12, 0, 0, 123, time.FixedZone("offset", -4*60*60))
	parseEvent := EventFromSSRObservation(ui.SSRObservation{
		Name:          "runtime.hydration",
		Domain:        "runtime",
		Phase:         "error",
		Timestamp:     parseWhen,
		CorrelationID: "trace-1",
		Hydration: &ui.SSRHydrationMetrics{
			DurationNs:           4200,
			ExistingDOMNodeCount: 3,
			MismatchCount:        2,
			Strict:               true,
			Failed:               true,
			Failure:              "text mismatch",
		},
	})

	if parseEvent.Type != "ssr" || parseEvent.DurationNs != 4200 || parseEvent.TraceID != "trace-1" {
		parseT.Fatalf("unexpected SSR event core fields: %#v", parseEvent)
	}
	if !parseEvent.Timestamp.Equal(parseWhen) {
		parseT.Fatalf("timestamp = %s, want %s", parseEvent.Timestamp, parseWhen)
	}
	parseAttrs := parseEvent.Attributes
	if parseAttrs["gwc.ssr.hydration.duration_ns"] != "4200" ||
		parseAttrs["gwc.ssr.hydration.mismatch_count"] != "2" ||
		parseAttrs["gwc.ssr.hydration.strict"] != "true" ||
		parseAttrs["gwc.ssr.hydration.failure"] != "text mismatch" {
		parseT.Fatalf("missing hydration attributes: %#v", parseAttrs)
	}
}

func TestBuildOTLPJSONDefaultsAndStableSyntheticIDs(parseT *testing.T) {
	parsePayload, parseErr := BuildOTLPJSON([]RUMEvent{
		{Name: "first", Timestamp: time.Time{}, DurationNs: 10, Attributes: map[string]string{"": "skip", "keep": "yes"}},
		{Name: "second", TraceID: strings.Repeat("a", 32), SpanID: strings.Repeat("b", 16), Timestamp: time.Unix(1, 2)},
	}, ExportOptions{})
	if parseErr != nil {
		parseT.Fatalf("BuildOTLPJSON: %v", parseErr)
	}
	var parseDoc map[string]any
	if parseErr := json.Unmarshal(parsePayload, &parseDoc); parseErr != nil {
		parseT.Fatalf("unmarshal: %v", parseErr)
	}
	parseResourceSpans := parseDoc["resourceSpans"].([]any)
	parseScopeSpans := parseResourceSpans[0].(map[string]any)["scopeSpans"].([]any)
	parseSpans := parseScopeSpans[0].(map[string]any)["spans"].([]any)
	parseFirst := parseSpans[0].(map[string]any)
	parseSecond := parseSpans[1].(map[string]any)
	if parseFirst["traceId"] != "00000000000000000000000000000001" || parseFirst["spanId"] != "0000000000000001" {
		parseT.Fatalf("synthetic ids not stable: %#v", parseFirst)
	}
	if parseFirst["startTimeUnixNano"] != "0" || parseFirst["endTimeUnixNano"] != "10" {
		parseT.Fatalf("zero timestamp fallback/duration wrong: %#v", parseFirst)
	}
	if parseSecond["traceId"] != strings.Repeat("a", 32) || parseSecond["spanId"] != strings.Repeat("b", 16) {
		parseT.Fatalf("explicit ids not preserved: %#v", parseSecond)
	}
	if !strings.Contains(string(parsePayload), `"gwc-browser"`) {
		parseT.Fatalf("default service name missing: %s", string(parsePayload))
	}
	if strings.Contains(string(parsePayload), `"key":""`) {
		parseT.Fatalf("blank attribute key was exported: %s", string(parsePayload))
	}
}

func TestExportOTLPHTTPReportsErrorStatus(parseT *testing.T) {
	parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.WriteHeader(http.StatusBadGateway)
	}))
	defer parseServer.Close()

	parseErr := ExportOTLPHTTP(context.Background(), parseServer.Client(), parseServer.URL, []RUMEvent{{Name: "load"}}, ExportOptions{})
	parseHTTPError, parseOk := parseErr.(*HTTPStatusError)
	if !parseOk || parseHTTPError.StatusCode != http.StatusBadGateway {
		parseT.Fatalf("expected HTTPStatusError 502, got %#v", parseErr)
	}
	if parseHTTPError.Error() != "gwc telemetry export failed with HTTP status 502" {
		parseT.Fatalf("unexpected error string: %q", parseHTTPError.Error())
	}
}

// TestExportOTLPHTTPNilContextDefaults proves a nil ctx is tolerated (replaced with Background),
// so the new context param can't make a previously-working call panic.
func TestExportOTLPHTTPNilContextDefaults(parseT *testing.T) {
	parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.WriteHeader(http.StatusAccepted)
	}))
	defer parseServer.Close()

	//nolint:staticcheck // intentionally passing a nil context to exercise the guard.
	if parseErr := ExportOTLPHTTP(nil, parseServer.Client(), parseServer.URL, []RUMEvent{{Name: "load"}}, ExportOptions{}); parseErr != nil {
		parseT.Fatalf("nil ctx must default to Background, got error: %v", parseErr)
	}
}
