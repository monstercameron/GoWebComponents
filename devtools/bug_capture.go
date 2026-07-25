package devtools

import (
	"encoding/json"
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/logging"
)

const currentBugCaptureBundleVersion = 1

// CaptureBugBundle captures one local debugging bundle from the current live snapshot.
func CaptureBugBundle(parseLabel string) BugCaptureBundle {
	parseTrace := CaptureTrace(parseLabel)
	return BugCaptureBundle{
		Version:    currentBugCaptureBundleVersion,
		Label:      parseTrace.Label,
		CapturedAt: parseTrace.CapturedAt,
		Trace:      parseTrace,
	}
}

// ExportBugCaptureBundleJSON serializes one local debugging bundle into stable JSON.
func ExportBugCaptureBundleJSON(parseBundle BugCaptureBundle) ([]byte, error) {
	parseEncoded, parseErr := json.Marshal(parseBundle)
	if parseErr != nil {
		return nil, parseErr
	}
	var parseGeneric any
	if parseErr2 := json.Unmarshal(parseEncoded, &parseGeneric); parseErr2 != nil {
		return nil, parseErr2
	}
	return json.Marshal(logging.RedactTelemetryValue("devtools.bug", parseGeneric))
}

// ImportBugCaptureBundleJSON deserializes one local debugging bundle from JSON.
func ImportBugCaptureBundleJSON(parseData []byte) (BugCaptureBundle, error) {
	if len(parseData) == 0 {
		return BugCaptureBundle{}, nil
	}
	var parseBundle BugCaptureBundle
	if parseErr := json.Unmarshal(parseData, &parseBundle); parseErr != nil {
		return BugCaptureBundle{}, parseErr
	}
	if parseBundle.Version == 0 {
		parseBundle.Version = currentBugCaptureBundleVersion
	}
	parseBundle.Label = strings.TrimSpace(parseBundle.Label)
	parseBundle.CapturedAt = strings.TrimSpace(parseBundle.CapturedAt)
	parseBundle.Trace.Label = strings.TrimSpace(parseBundle.Trace.Label)
	parseBundle.Trace.CapturedAt = strings.TrimSpace(parseBundle.Trace.CapturedAt)
	if parseBundle.Label == "" {
		parseBundle.Label = parseBundle.Trace.Label
	}
	if parseBundle.CapturedAt == "" {
		parseBundle.CapturedAt = parseBundle.Trace.CapturedAt
	}
	return parseBundle, nil
}

// ReplayBugCaptureBundle replays the bug-capture bundle through the trace replay path.
func ReplayBugCaptureBundle(parseBundle BugCaptureBundle) {
	parseTrace := parseBundle.Trace
	if parseTrace.Label == "" {
		parseTrace.Label = strings.TrimSpace(parseBundle.Label)
	}
	if parseTrace.CapturedAt == "" {
		parseTrace.CapturedAt = strings.TrimSpace(parseBundle.CapturedAt)
	}
	SetTraceReplay(parseTrace)
}
