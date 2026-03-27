package runtime2

import "fmt"

// DiagnosticSizeMetrics stores structured size diagnostics for one worker-backed lifecycle event.
type DiagnosticSizeMetrics struct {
	SnapshotBytes   uint64 `json:"snapshot_bytes,omitempty"`
	IRBytes         uint64 `json:"ir_bytes,omitempty"`
	PatchBytes      uint64 `json:"patch_bytes,omitempty"`
	SharedPageBytes uint64 `json:"shared_page_bytes,omitempty"`
}

// ValidateDiagnosticSizeMetrics verifies size diagnostics include at least one non-zero payload size.
func ValidateDiagnosticSizeMetrics(parseMetrics DiagnosticSizeMetrics) error {
	if parseMetrics.SnapshotBytes == 0 &&
		parseMetrics.IRBytes == 0 &&
		parseMetrics.PatchBytes == 0 &&
		parseMetrics.SharedPageBytes == 0 {
		return fmt.Errorf("runtime2: diagnostic size payload requires at least one non-zero metric")
	}
	return nil
}
