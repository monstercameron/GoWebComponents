package devtools

import (
	"encoding/json"
	"strings"
)

const currentBugCaptureBundleVersion = 1

// CaptureBugBundle captures one local debugging bundle from the current live snapshot.
func CaptureBugBundle(label string) BugCaptureBundle {
	trace := CaptureTrace(label)
	return BugCaptureBundle{
		Version:    currentBugCaptureBundleVersion,
		Label:      trace.Label,
		CapturedAt: trace.CapturedAt,
		Trace:      trace,
	}
}

// ExportBugCaptureBundleJSON serializes one local debugging bundle into stable JSON.
func ExportBugCaptureBundleJSON(bundle BugCaptureBundle) ([]byte, error) {
	return json.Marshal(bundle)
}

// ImportBugCaptureBundleJSON deserializes one local debugging bundle from JSON.
func ImportBugCaptureBundleJSON(data []byte) (BugCaptureBundle, error) {
	if len(data) == 0 {
		return BugCaptureBundle{}, nil
	}
	var bundle BugCaptureBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return BugCaptureBundle{}, err
	}
	if bundle.Version == 0 {
		bundle.Version = currentBugCaptureBundleVersion
	}
	bundle.Label = strings.TrimSpace(bundle.Label)
	bundle.CapturedAt = strings.TrimSpace(bundle.CapturedAt)
	bundle.Trace.Label = strings.TrimSpace(bundle.Trace.Label)
	bundle.Trace.CapturedAt = strings.TrimSpace(bundle.Trace.CapturedAt)
	if bundle.Label == "" {
		bundle.Label = bundle.Trace.Label
	}
	if bundle.CapturedAt == "" {
		bundle.CapturedAt = bundle.Trace.CapturedAt
	}
	return bundle, nil
}

// ReplayBugCaptureBundle replays the bug-capture bundle through the trace replay path.
func ReplayBugCaptureBundle(bundle BugCaptureBundle) {
	trace := bundle.Trace
	if trace.Label == "" {
		trace.Label = strings.TrimSpace(bundle.Label)
	}
	if trace.CapturedAt == "" {
		trace.CapturedAt = strings.TrimSpace(bundle.CapturedAt)
	}
	SetTraceReplay(trace)
}
