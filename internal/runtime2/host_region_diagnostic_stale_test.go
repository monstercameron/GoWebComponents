package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleHostControlEnvelopeDiagnosticIgnoresStaleEpoch verifies older-epoch diagnostics are ignored before ring storage.
func TestHandleHostControlEnvelopeDiagnosticIgnoresStaleEpoch(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	parseEnvelope := runtime2.ControlEnvelope{
		ProtocolVersion:  runtime2.ProtocolVersionParallelV1,
		Kind:             runtime2.ControlKindDiagnostic,
		RegionInstanceID: "region-1",
		Epoch:            1,
		InputVersion:     1,
		DiagnosticType:   string(runtime2.DiagnosticEventKindUpdate),
		DiagnosticText:   "stale epoch diagnostic",
	}
	parseDispatchResult, parseDispatchErr := runtime2.HandleHostControlEnvelope(buildHostRegionAdapter, parseEnvelope)
	if parseDispatchErr != nil {
		parseT.Fatalf("HandleHostControlEnvelope(stale diagnostic epoch) returned error: %v", parseDispatchErr)
	}
	if !parseDispatchResult.HasDiagnosticResult {
		parseT.Fatal("expected diagnostic dispatch result for stale diagnostic")
	}
	if !parseDispatchResult.HasDiagnosticIgnored {
		parseT.Fatal("expected stale epoch diagnostic to be ignored")
	}
	if parseDispatchResult.GetDiagnosticIgnore != "stale-epoch" {
		parseT.Fatalf("expected stale-epoch ignore reason, got %q", parseDispatchResult.GetDiagnosticIgnore)
	}
	getDiagnosticRing := buildHostRegionAdapter.GetHostRegionDiagnosticRing()
	if len(getDiagnosticRing) != 0 {
		parseT.Fatalf("expected stale epoch diagnostic to skip ring storage, got %d entries", len(getDiagnosticRing))
	}
	parseEntry, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if parseEntry.IgnoredStaleDiagnosticCount != 1 {
		parseT.Fatalf("expected stale-diagnostic ignore counter 1, got %d", parseEntry.IgnoredStaleDiagnosticCount)
	}
}

// TestHandleHostControlEnvelopeDiagnosticIgnoresStaleVersion verifies lower-version diagnostics are ignored after newer dispatch progress.
func TestHandleHostControlEnvelopeDiagnosticIgnoresStaleVersion(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if _, parseUpdateErr := buildHostRegionAdapter.HandleHostRegionUpdate(5); parseUpdateErr != nil {
		parseT.Fatalf("HandleHostRegionUpdate returned error: %v", parseUpdateErr)
	}
	parseStaleEnvelope := runtime2.ControlEnvelope{
		ProtocolVersion:  runtime2.ProtocolVersionParallelV1,
		Kind:             runtime2.ControlKindDiagnostic,
		RegionInstanceID: "region-1",
		Epoch:            2,
		InputVersion:     4,
		DiagnosticType:   string(runtime2.DiagnosticEventKindUpdate),
		DiagnosticText:   "stale version diagnostic",
	}
	parseStaleResult, parseStaleErr := runtime2.HandleHostControlEnvelope(buildHostRegionAdapter, parseStaleEnvelope)
	if parseStaleErr != nil {
		parseT.Fatalf("HandleHostControlEnvelope(stale diagnostic version) returned error: %v", parseStaleErr)
	}
	if !parseStaleResult.HasDiagnosticIgnored {
		parseT.Fatal("expected stale version diagnostic to be ignored")
	}
	if parseStaleResult.GetDiagnosticIgnore != "stale-version" {
		parseT.Fatalf("expected stale-version ignore reason, got %q", parseStaleResult.GetDiagnosticIgnore)
	}
	parseFreshEnvelope := parseStaleEnvelope
	parseFreshEnvelope.InputVersion = 5
	parseFreshEnvelope.DiagnosticText = "fresh version diagnostic"
	parseFreshResult, parseFreshErr := runtime2.HandleHostControlEnvelope(buildHostRegionAdapter, parseFreshEnvelope)
	if parseFreshErr != nil {
		parseT.Fatalf("HandleHostControlEnvelope(fresh diagnostic version) returned error: %v", parseFreshErr)
	}
	if parseFreshResult.HasDiagnosticIgnored {
		parseT.Fatalf("expected fresh diagnostic to store, got ignore reason %q", parseFreshResult.GetDiagnosticIgnore)
	}
	getDiagnosticRing := buildHostRegionAdapter.GetHostRegionDiagnosticRing()
	if len(getDiagnosticRing) != 1 {
		parseT.Fatalf("expected one fresh diagnostic ring entry, got %d", len(getDiagnosticRing))
	}
}

// TestHandleHostControlEnvelopeDiagnosticIgnoresFallbackOwnedDiagnostic verifies fallback-owned regions ignore delayed diagnostics.
func TestHandleHostControlEnvelopeDiagnosticIgnoresFallbackOwnedDiagnostic(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	buildHostRegionAdapter.GetHostRegionRecoveryCoordinator().EnterRegionLocalFallback("region-1", "worker-death-no-reassign", 2, 6)
	parseEnvelope := runtime2.ControlEnvelope{
		ProtocolVersion:  runtime2.ProtocolVersionParallelV1,
		Kind:             runtime2.ControlKindDiagnostic,
		RegionInstanceID: "region-1",
		Epoch:            2,
		InputVersion:     6,
		DiagnosticType:   string(runtime2.DiagnosticEventKindFallback),
		DiagnosticText:   "delayed fallback diagnostic",
	}
	parseDispatchResult, parseDispatchErr := runtime2.HandleHostControlEnvelope(buildHostRegionAdapter, parseEnvelope)
	if parseDispatchErr != nil {
		parseT.Fatalf("HandleHostControlEnvelope(fallback-owned diagnostic) returned error: %v", parseDispatchErr)
	}
	if !parseDispatchResult.HasDiagnosticIgnored {
		parseT.Fatal("expected fallback-owned diagnostic to be ignored")
	}
	if parseDispatchResult.GetDiagnosticIgnore != "fallback-active" {
		parseT.Fatalf("expected fallback-active ignore reason, got %q", parseDispatchResult.GetDiagnosticIgnore)
	}
	getDiagnosticRing := buildHostRegionAdapter.GetHostRegionDiagnosticRing()
	if len(getDiagnosticRing) != 0 {
		parseT.Fatalf("expected fallback-owned diagnostic to skip ring storage, got %d entries", len(getDiagnosticRing))
	}
}
