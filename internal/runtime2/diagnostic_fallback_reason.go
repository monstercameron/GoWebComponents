package runtime2

import (
	"fmt"
	"strings"
)

// DiagnosticFallbackReason stores structured fallback diagnostics aligned with recovery-layer reason domains.
type DiagnosticFallbackReason struct {
	Domain string `json:"domain,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// ValidateDiagnosticFallbackReason verifies fallback diagnostics align with transport, DOM, and worker-death reason sets.
func ValidateDiagnosticFallbackReason(parseReason DiagnosticFallbackReason) error {
	parseDomain := strings.TrimSpace(parseReason.Domain)
	if parseDomain == "" {
		return fmt.Errorf("runtime2: diagnostic fallback domain is required")
	}
	if parseDomain != parseReason.Domain {
		return fmt.Errorf("runtime2: diagnostic fallback domain must not contain surrounding whitespace")
	}
	parseReasonCode := strings.TrimSpace(parseReason.Reason)
	if parseReasonCode == "" {
		return fmt.Errorf("runtime2: diagnostic fallback reason is required")
	}
	if parseReasonCode != parseReason.Reason {
		return fmt.Errorf("runtime2: diagnostic fallback reason must not contain surrounding whitespace")
	}
	switch parseDomain {
	case "transport":
		switch TransportFailureKind(parseReasonCode) {
		case TransportFailureKindMalformedControlPayload, TransportFailureKindMalformedPatchPayload, TransportFailureKindMalformedSharedMemoryPage:
			return nil
		default:
			return fmt.Errorf("runtime2: diagnostic fallback transport reason %q is unsupported", parseReasonCode)
		}
	case "dom":
		switch DOMCommitFailureKind(parseReasonCode) {
		case DOMCommitFailureKindMissingParentAnchor, DOMCommitFailureKindMissingNodeLookup, DOMCommitFailureKindInvalidKeyedMove:
			return nil
		default:
			return fmt.Errorf("runtime2: diagnostic fallback dom reason %q is unsupported", parseReasonCode)
		}
	case "worker-death":
		switch WorkerDeathFailureKind(parseReasonCode) {
		case WorkerDeathFailureKindRepairFailure, WorkerDeathFailureKindNoReassign:
			return nil
		default:
			return fmt.Errorf("runtime2: diagnostic fallback worker-death reason %q is unsupported", parseReasonCode)
		}
	default:
		return fmt.Errorf("runtime2: diagnostic fallback domain %q is unsupported", parseDomain)
	}
}
