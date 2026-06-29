package devtools

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/logging"
)

var traceReplayState struct {
	mu      sync.RWMutex
	capture TraceCapture
	active  bool
}

// CaptureTrace captures the current live devtools snapshot with a label and timestamp.
func CaptureTrace(parseLabel string) TraceCapture {
	return TraceCapture{
		Label:      strings.TrimSpace(parseLabel),
		CapturedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Snapshot:   snapshotNowLive(),
	}
}

// ExportTraceCaptureJSON serializes one trace capture into stable JSON.
func ExportTraceCaptureJSON(parseCapture TraceCapture) ([]byte, error) {
	parseEncoded, parseErr := json.Marshal(parseCapture)
	if parseErr != nil {
		return nil, parseErr
	}
	var parseGeneric any
	if parseErr2 := json.Unmarshal(parseEncoded, &parseGeneric); parseErr2 != nil {
		return nil, parseErr2
	}
	return json.Marshal(logging.RedactTelemetryValue("devtools.trace", parseGeneric))
}

// ImportTraceCaptureJSON deserializes one trace capture from JSON.
func ImportTraceCaptureJSON(parseData []byte) (TraceCapture, error) {
	if len(parseData) == 0 {
		return TraceCapture{}, nil
	}
	var parseCapture TraceCapture
	if parseErr := json.Unmarshal(parseData, &parseCapture); parseErr != nil {
		return TraceCapture{}, parseErr
	}
	parseCapture.Label = strings.TrimSpace(parseCapture.Label)
	parseCapture.CapturedAt = strings.TrimSpace(parseCapture.CapturedAt)
	return parseCapture, nil
}

// SetTraceReplay activates replay mode for the supplied captured trace.
func SetTraceReplay(parseCapture TraceCapture) {
	traceReplayState.mu.Lock()
	defer traceReplayState.mu.Unlock()
	traceReplayState.capture = cloneTraceCapture(parseCapture)
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

func cloneTraceCapture(parseCapture TraceCapture) TraceCapture {
	parseEncoded, parseErr := json.Marshal(parseCapture)
	if parseErr != nil {
		return TraceCapture{}
	}
	var parseCloned TraceCapture
	if parseErr2 := json.Unmarshal(parseEncoded, &parseCloned); parseErr2 != nil {
		return TraceCapture{}
	}
	return parseCloned
}
