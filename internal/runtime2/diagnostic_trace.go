package runtime2

import (
	"fmt"
	"strings"
)

// DiagnosticTraceMetadata stores debug-only trace IDs that correlate scheduler, worker, and commit attempts.
type DiagnosticTraceMetadata struct {
	IsDebug            bool   `json:"is_debug,omitempty"`
	TraceID            string `json:"trace_id,omitempty"`
	SchedulerAttemptID uint64 `json:"scheduler_attempt_id,omitempty"`
	WorkerAttemptID    uint64 `json:"worker_attempt_id,omitempty"`
	CommitAttemptID    uint64 `json:"commit_attempt_id,omitempty"`
}

// ValidateDiagnosticTraceMetadata verifies debug-only trace metadata is complete and non-ambiguous.
func ValidateDiagnosticTraceMetadata(parseMetadata DiagnosticTraceMetadata) error {
	if !parseMetadata.IsDebug {
		return fmt.Errorf("runtime2: diagnostic trace metadata is debug-only and requires is_debug=true")
	}
	parseTraceID := strings.TrimSpace(parseMetadata.TraceID)
	if parseTraceID == "" {
		return fmt.Errorf("runtime2: diagnostic trace ID is required")
	}
	if parseTraceID != parseMetadata.TraceID {
		return fmt.Errorf("runtime2: diagnostic trace ID must not contain surrounding whitespace")
	}
	if parseMetadata.SchedulerAttemptID == 0 {
		return fmt.Errorf("runtime2: diagnostic trace scheduler attempt ID is required")
	}
	if parseMetadata.WorkerAttemptID == 0 {
		return fmt.Errorf("runtime2: diagnostic trace worker attempt ID is required")
	}
	if parseMetadata.CommitAttemptID == 0 {
		return fmt.Errorf("runtime2: diagnostic trace commit attempt ID is required")
	}
	return nil
}
