package runtime2_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
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
