package runtime2_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestHandleHostControlEnvelopeDiagnosticStoresHostDiagnosticRing verifies diagnostic dispatch appends a durable host-region diagnostic ring entry.
func TestHandleHostControlEnvelopeDiagnosticStoresHostDiagnosticRing(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	parseEnvelope, parseEnvelopeErr := runtime2.BuildControlDiagnosticEnvelope("region-1", runtime2.ControlDiagnosticEnvelopeSpec{
		DiagnosticType: runtime2.DiagnosticEventKindUpdate,
		DiagnosticText: "token=secret-123",
	})
	if parseEnvelopeErr != nil {
		parseT.Fatalf("BuildControlDiagnosticEnvelope returned error: %v", parseEnvelopeErr)
	}
	parseDispatchResult, parseDispatchErr := runtime2.HandleHostControlEnvelope(buildHostRegionAdapter, parseEnvelope)
	if parseDispatchErr != nil {
		parseT.Fatalf("HandleHostControlEnvelope(diagnostic) returned error: %v", parseDispatchErr)
	}
	if !parseDispatchResult.HasDiagnosticResult {
		parseT.Fatal("expected diagnostic dispatch result")
	}
	getDiagnosticRing := buildHostRegionAdapter.GetHostRegionDiagnosticRing()
	if len(getDiagnosticRing) != 1 {
		parseT.Fatalf("expected one durable diagnostic ring entry, got %d", len(getDiagnosticRing))
	}
	if getDiagnosticRing[0].DiagnosticType != string(runtime2.DiagnosticEventKindUpdate) {
		parseT.Fatalf("expected stored diagnostic type %q, got %q", runtime2.DiagnosticEventKindUpdate, getDiagnosticRing[0].DiagnosticType)
	}
	if strings.Contains(getDiagnosticRing[0].DiagnosticText, "secret-123") {
		parseT.Fatalf("expected stored diagnostic text to redact sensitive values, got %q", getDiagnosticRing[0].DiagnosticText)
	}
	if !strings.Contains(getDiagnosticRing[0].DiagnosticText, "token=[redacted]") {
		parseT.Fatalf("expected stored diagnostic text to include redaction marker, got %q", getDiagnosticRing[0].DiagnosticText)
	}
}

// TestHandleHostRegionDisposeClearsHostDiagnosticRing verifies region disposal clears durable diagnostic ring state.
func TestHandleHostRegionDisposeClearsHostDiagnosticRing(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	parseEnvelope, parseEnvelopeErr := runtime2.BuildControlDiagnosticEnvelope("region-1", runtime2.ControlDiagnosticEnvelopeSpec{
		DiagnosticType: runtime2.DiagnosticEventKindFallback,
		DiagnosticText: "fallback entered",
	})
	if parseEnvelopeErr != nil {
		parseT.Fatalf("BuildControlDiagnosticEnvelope returned error: %v", parseEnvelopeErr)
	}
	if _, parseDispatchErr := runtime2.HandleHostControlEnvelope(buildHostRegionAdapter, parseEnvelope); parseDispatchErr != nil {
		parseT.Fatalf("HandleHostControlEnvelope(diagnostic) returned error: %v", parseDispatchErr)
	}
	if _, parseDisposeErr := buildHostRegionAdapter.HandleHostRegionDispose(); parseDisposeErr != nil {
		parseT.Fatalf("HandleHostRegionDispose returned error: %v", parseDisposeErr)
	}
	getDiagnosticRing := buildHostRegionAdapter.GetHostRegionDiagnosticRing()
	if len(getDiagnosticRing) != 0 {
		parseT.Fatalf("expected diagnostic ring to clear on dispose, got %d entries", len(getDiagnosticRing))
	}
}

// TestHandleHostControlEnvelopeDiagnosticRingTrimKeepsNewestDeterministicOrder verifies ring trim keeps newest diagnostics in deterministic order.
func TestHandleHostControlEnvelopeDiagnosticRingTrimKeepsNewestDeterministicOrder(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	const parseTotalDiagnostics = 100
	for parseDiagnosticIndex := range parseTotalDiagnostics {
		parseDiagnosticSpec := runtime2.ControlDiagnosticEnvelopeSpec{
			DiagnosticText: fmt.Sprintf("event-seq=%d", parseDiagnosticIndex),
		}
		switch parseDiagnosticIndex % 3 {
		case 0:
			parseDiagnosticSpec.DiagnosticType = runtime2.DiagnosticEventKindFallback
		case 1:
			parseDiagnosticSpec.DiagnosticType = runtime2.DiagnosticEventKindRepair
		default:
			parseDiagnosticSpec.DiagnosticType = runtime2.DiagnosticEventKindPatchReady
			parseDiagnosticSpec.TransportTier = runtime2.TransportTierStructuredClone
			parseDiagnosticSpec.DiagnosticDowngrade = &runtime2.DiagnosticDowngradeReason{
				Path:   runtime2.DiagnosticDowngradePathSharedMemory,
				Reason: string(runtime2.SharedPatchDowngradeReasonInvalidSharedPage),
			}
		}
		parseEnvelope, parseEnvelopeErr := runtime2.BuildControlDiagnosticEnvelope("region-1", parseDiagnosticSpec)
		if parseEnvelopeErr != nil {
			parseT.Fatalf("BuildControlDiagnosticEnvelope(%d) returned error: %v", parseDiagnosticIndex, parseEnvelopeErr)
		}
		if _, parseDispatchErr := runtime2.HandleHostControlEnvelope(buildHostRegionAdapter, parseEnvelope); parseDispatchErr != nil {
			parseT.Fatalf("HandleHostControlEnvelope(%d) returned error: %v", parseDiagnosticIndex, parseDispatchErr)
		}
	}
	getDiagnosticRing := buildHostRegionAdapter.GetHostRegionDiagnosticRing()
	if len(getDiagnosticRing) == 0 {
		parseT.Fatal("expected diagnostics ring entries after diagnostic dispatch")
	}
	if len(getDiagnosticRing) >= parseTotalDiagnostics {
		parseT.Fatalf("expected diagnostics ring trim after %d events, got %d entries", parseTotalDiagnostics, len(getDiagnosticRing))
	}
	parseStartSequence := parseTotalDiagnostics - len(getDiagnosticRing)
	parseHasFallback := false
	parseHasRepair := false
	parseHasPatchDowngrade := false
	for parseDiagnosticIndex := range getDiagnosticRing {
		parseExpectedSequence := parseStartSequence + parseDiagnosticIndex
		parseExpectedText := fmt.Sprintf("event-seq=%d", parseExpectedSequence)
		if getDiagnosticRing[parseDiagnosticIndex].DiagnosticText != parseExpectedText {
			parseT.Fatalf(
				"expected ring entry %d diagnostic text %q, got %q",
				parseDiagnosticIndex,
				parseExpectedText,
				getDiagnosticRing[parseDiagnosticIndex].DiagnosticText,
			)
		}
		switch getDiagnosticRing[parseDiagnosticIndex].DiagnosticType {
		case string(runtime2.DiagnosticEventKindFallback):
			parseHasFallback = true
		case string(runtime2.DiagnosticEventKindRepair):
			parseHasRepair = true
		case string(runtime2.DiagnosticEventKindPatchReady):
			if getDiagnosticRing[parseDiagnosticIndex].DiagnosticDowngrade != nil &&
				getDiagnosticRing[parseDiagnosticIndex].DiagnosticDowngrade.Reason == string(runtime2.SharedPatchDowngradeReasonInvalidSharedPage) {
				parseHasPatchDowngrade = true
			}
		}
	}
	if !parseHasFallback || !parseHasRepair || !parseHasPatchDowngrade {
		parseT.Fatalf(
			"expected trimmed ring to retain fallback, repair, and patch-downgrade events, got fallback=%t repair=%t patch-downgrade=%t",
			parseHasFallback,
			parseHasRepair,
			parseHasPatchDowngrade,
		)
	}
}
