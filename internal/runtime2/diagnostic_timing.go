package runtime2

import "fmt"

// DiagnosticTimingMetrics stores structured timing diagnostics for one worker-backed lifecycle event.
type DiagnosticTimingMetrics struct {
	QueueNanos     uint64 `json:"queue_ns,omitempty"`
	RenderNanos    uint64 `json:"render_ns,omitempty"`
	DiffNanos      uint64 `json:"diff_ns,omitempty"`
	EncodeNanos    uint64 `json:"encode_ns,omitempty"`
	TransportNanos uint64 `json:"transport_ns,omitempty"`
	CommitNanos    uint64 `json:"commit_ns,omitempty"`
}

// ValidateDiagnosticTimingMetrics verifies timing diagnostics include at least one non-zero stage duration.
func ValidateDiagnosticTimingMetrics(parseMetrics DiagnosticTimingMetrics) error {
	if parseMetrics.QueueNanos == 0 &&
		parseMetrics.RenderNanos == 0 &&
		parseMetrics.DiffNanos == 0 &&
		parseMetrics.EncodeNanos == 0 &&
		parseMetrics.TransportNanos == 0 &&
		parseMetrics.CommitNanos == 0 {
		return fmt.Errorf("runtime2: diagnostic timing payload requires at least one non-zero stage duration")
	}
	return nil
}
