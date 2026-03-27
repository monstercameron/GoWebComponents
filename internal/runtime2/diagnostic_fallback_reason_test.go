package runtime2

import (
	"strings"
	"testing"
)

// TestValidateDiagnosticFallbackReasonAcceptsTransportReason verifies transport fallback diagnostics accept known recovery reasons.
func TestValidateDiagnosticFallbackReasonAcceptsTransportReason(parseT *testing.T) {
	parseReason := DiagnosticFallbackReason{
		Domain: "transport",
		Reason: string(TransportFailureKindMalformedPatchPayload),
	}
	if parseErr := ValidateDiagnosticFallbackReason(parseReason); parseErr != nil {
		parseT.Fatalf("ValidateDiagnosticFallbackReason(transport) returned error: %v", parseErr)
	}
}

// TestValidateDiagnosticFallbackReasonAcceptsDOMReason verifies DOM fallback diagnostics accept known recovery reasons.
func TestValidateDiagnosticFallbackReasonAcceptsDOMReason(parseT *testing.T) {
	parseReason := DiagnosticFallbackReason{
		Domain: "dom",
		Reason: string(DOMCommitFailureKindMissingNodeLookup),
	}
	if parseErr := ValidateDiagnosticFallbackReason(parseReason); parseErr != nil {
		parseT.Fatalf("ValidateDiagnosticFallbackReason(dom) returned error: %v", parseErr)
	}
}

// TestValidateDiagnosticFallbackReasonAcceptsWorkerDeathReason verifies worker-death fallback diagnostics accept known recovery reasons.
func TestValidateDiagnosticFallbackReasonAcceptsWorkerDeathReason(parseT *testing.T) {
	parseReason := DiagnosticFallbackReason{
		Domain: "worker-death",
		Reason: string(WorkerDeathFailureKindNoReassign),
	}
	if parseErr := ValidateDiagnosticFallbackReason(parseReason); parseErr != nil {
		parseT.Fatalf("ValidateDiagnosticFallbackReason(worker-death) returned error: %v", parseErr)
	}
}

// TestValidateDiagnosticFallbackReasonRejectsUnsupportedDomain verifies fallback diagnostics reject unknown domains.
func TestValidateDiagnosticFallbackReasonRejectsUnsupportedDomain(parseT *testing.T) {
	parseErr := ValidateDiagnosticFallbackReason(DiagnosticFallbackReason{
		Domain: "network",
		Reason: string(TransportFailureKindMalformedControlPayload),
	})
	if parseErr == nil {
		parseT.Fatal("expected unsupported fallback domain to fail")
	}
	if !strings.Contains(parseErr.Error(), "domain") {
		parseT.Fatalf("expected unsupported-domain error details, got %v", parseErr)
	}
}

// TestValidateDiagnosticFallbackReasonRejectsUnsupportedReason verifies fallback diagnostics reject unknown reasons within one domain.
func TestValidateDiagnosticFallbackReasonRejectsUnsupportedReason(parseT *testing.T) {
	parseErr := ValidateDiagnosticFallbackReason(DiagnosticFallbackReason{
		Domain: "dom",
		Reason: string(TransportFailureKindMalformedPatchPayload),
	})
	if parseErr == nil {
		parseT.Fatal("expected unsupported fallback reason to fail")
	}
	if !strings.Contains(parseErr.Error(), "unsupported") {
		parseT.Fatalf("expected unsupported-reason error details, got %v", parseErr)
	}
}
