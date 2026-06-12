package telemetry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/devtools"
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

	parseErr := ExportOTLPHTTP(parseServer.Client(), parseServer.URL, []RUMEvent{{Name: "load", Type: "profile"}}, ExportOptions{})
	if parseErr != nil {
		parseT.Fatalf("ExportOTLPHTTP returned error: %v", parseErr)
	}
	if !parsePosted {
		parseT.Fatal("server did not receive export")
	}
}
