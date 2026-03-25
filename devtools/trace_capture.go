package devtools

import (
	"encoding/json"
	"strings"
	"sync"
	"time"
)

var traceReplayState struct {
	mu      sync.RWMutex
	capture TraceCapture
	active  bool
}

// CaptureTrace captures the current live devtools snapshot with a label and timestamp.
func CaptureTrace(label string) TraceCapture {
	return TraceCapture{
		Label:      strings.TrimSpace(label),
		CapturedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Snapshot:   snapshotNowLive(),
	}
}

// ExportTraceCaptureJSON serializes one trace capture into stable JSON.
func ExportTraceCaptureJSON(capture TraceCapture) ([]byte, error) {
	return json.Marshal(capture)
}

// ImportTraceCaptureJSON deserializes one trace capture from JSON.
func ImportTraceCaptureJSON(data []byte) (TraceCapture, error) {
	if len(data) == 0 {
		return TraceCapture{}, nil
	}
	var capture TraceCapture
	if err := json.Unmarshal(data, &capture); err != nil {
		return TraceCapture{}, err
	}
	capture.Label = strings.TrimSpace(capture.Label)
	capture.CapturedAt = strings.TrimSpace(capture.CapturedAt)
	return capture, nil
}

// SetTraceReplay activates replay mode for the supplied captured trace.
func SetTraceReplay(capture TraceCapture) {
	traceReplayState.mu.Lock()
	defer traceReplayState.mu.Unlock()
	traceReplayState.capture = cloneTraceCapture(capture)
	traceReplayState.active = true
}

// ClearTraceReplay disables the current replayed trace.
func ClearTraceReplay() {
	traceReplayState.mu.Lock()
	defer traceReplayState.mu.Unlock()
	traceReplayState.capture = TraceCapture{}
	traceReplayState.active = false
}

// CurrentTraceReplay returns the current replayed trace, if any.
func CurrentTraceReplay() (TraceCapture, bool) {
	traceReplayState.mu.RLock()
	defer traceReplayState.mu.RUnlock()
	if !traceReplayState.active {
		return TraceCapture{}, false
	}
	return cloneTraceCapture(traceReplayState.capture), true
}

func cloneTraceCapture(capture TraceCapture) TraceCapture {
	encoded, err := json.Marshal(capture)
	if err != nil {
		return TraceCapture{}
	}
	var cloned TraceCapture
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		return TraceCapture{}
	}
	return cloned
}
