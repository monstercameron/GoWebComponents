package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestGetCapabilityReportDefaultsWithoutWorkerSupport verifies the package-level report stays disabled by default.
func TestGetCapabilityReportDefaultsWithoutWorkerSupport(parseT *testing.T) {
	parseReport := runtime2.GetCapabilityReport()
	if parseReport.HasWorkerSupport {
		parseT.Fatal("expected worker support to default to false")
	}
	if parseReport.HasStructuredCloneSupport {
		parseT.Fatal("expected structured-clone support to default to false")
	}
	if parseReport.HasSharedMemoryTransportSupport {
		parseT.Fatal("expected shared-memory transport support to default to false")
	}
}

// TestBuildCapabilityReportKeepsStructuredCloneIndependentFromSharedMemory verifies the report does not collapse transport tiers together.
func TestBuildCapabilityReportKeepsStructuredCloneIndependentFromSharedMemory(parseT *testing.T) {
	parseReport := runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
		HasWorkerSupport:          true,
		HasStructuredCloneSupport: true,
	})
	if !parseReport.HasWorkerSupport {
		parseT.Fatal("expected worker support")
	}
	if !parseReport.HasStructuredCloneSupport {
		parseT.Fatal("expected structured-clone support")
	}
	if parseReport.HasSharedMemoryTransportSupport {
		parseT.Fatal("expected shared-memory transport support to remain false")
	}
}

// TestBuildCapabilityReportDisablesDependentFeaturesWithoutWorkerSupport verifies impossible combinations normalize safely.
func TestBuildCapabilityReportDisablesDependentFeaturesWithoutWorkerSupport(parseT *testing.T) {
	parseReport := runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasBinaryTransportSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
	if parseReport.HasMessagePortSupport {
		parseT.Fatal("expected message-port support to stay disabled without workers")
	}
	if parseReport.HasStructuredCloneSupport {
		parseT.Fatal("expected structured-clone support to stay disabled without workers")
	}
	if parseReport.HasSharedBufferSupport {
		parseT.Fatal("expected shared-buffer support to stay disabled without workers")
	}
}

// TestInitCapabilityReportStoresDetectedCapabilities verifies package-level capability initialization stores one detected report consumed by GetCapabilityReport.
func TestInitCapabilityReportStoresDetectedCapabilities(parseT *testing.T) {
	runtime2.ResetCapabilityReport()
	parseT.Cleanup(runtime2.ResetCapabilityReport)
	parseReport, parseInitErr := runtime2.InitCapabilityReport(runtime2.CapabilitySource{
		HasWorkerSupport:                true,
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasBinaryTransportSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
	if parseInitErr != nil {
		parseT.Fatalf("InitCapabilityReport returned error: %v", parseInitErr)
	}
	if !parseReport.HasSharedMemoryTransportSupport {
		parseT.Fatal("expected initialized capability report to preserve shared-memory transport support")
	}
	parseStoredReport := runtime2.GetCapabilityReport()
	if !parseStoredReport.HasWorkerSupport || !parseStoredReport.HasBinaryTransportSupport || !parseStoredReport.HasSharedMemoryTransportSupport {
		parseT.Fatalf("expected initialized package report to match detected support, got %+v", parseStoredReport)
	}
}

// TestSetCapabilityReportOverrideAndReset verifies override hooks update package-level capability state and reset restores default disabled reports.
func TestSetCapabilityReportOverrideAndReset(parseT *testing.T) {
	runtime2.ResetCapabilityReport()
	parseT.Cleanup(runtime2.ResetCapabilityReport)
	parseOverrideErr := runtime2.SetCapabilityReportOverride(runtime2.CapabilityReport{
		HasWorkerSupport:          true,
		HasMessagePortSupport:     true,
		HasStructuredCloneSupport: true,
		HasBinaryTransportSupport: true,
	})
	if parseOverrideErr != nil {
		parseT.Fatalf("SetCapabilityReportOverride returned error: %v", parseOverrideErr)
	}
	parseOverriddenReport := runtime2.GetCapabilityReport()
	if !parseOverriddenReport.HasWorkerSupport || !parseOverriddenReport.HasBinaryTransportSupport {
		parseT.Fatalf("expected overridden capability report to remain active, got %+v", parseOverriddenReport)
	}
	runtime2.ResetCapabilityReport()
	parseResetReport := runtime2.GetCapabilityReport()
	if parseResetReport.HasWorkerSupport || parseResetReport.HasStructuredCloneSupport || parseResetReport.HasBinaryTransportSupport {
		parseT.Fatalf("expected reset capability report to restore disabled defaults, got %+v", parseResetReport)
	}
}

// TestDetectCapabilitySourceDefaultsOutsideBrowserWASM verifies non-js builds report no live worker capability support.
func TestDetectCapabilitySourceDefaultsOutsideBrowserWASM(parseT *testing.T) {
	parseSource := runtime2.DetectCapabilitySource()
	if parseSource.HasWorkerSupport || parseSource.HasMessagePortSupport || parseSource.HasStructuredCloneSupport ||
		parseSource.HasBinaryTransportSupport || parseSource.HasSharedBufferSupport || parseSource.HasSharedMemoryTransportSupport {
		parseT.Fatalf("expected non-browser runtime capability detection to remain disabled, got %+v", parseSource)
	}
}

// TestInitCapabilityReportFromRuntimeUsesDetectedSource verifies runtime initialization consumes detected capability source state.
func TestInitCapabilityReportFromRuntimeUsesDetectedSource(parseT *testing.T) {
	runtime2.ResetCapabilityReport()
	parseT.Cleanup(runtime2.ResetCapabilityReport)
	parseReport, parseInitErr := runtime2.InitCapabilityReportFromRuntime()
	if parseInitErr != nil {
		parseT.Fatalf("InitCapabilityReportFromRuntime returned error: %v", parseInitErr)
	}
	parseExpectedReport := runtime2.BuildCapabilityReport(runtime2.DetectCapabilitySource())
	if parseReport != parseExpectedReport {
		parseT.Fatalf("expected runtime-initialized report %+v, got %+v", parseExpectedReport, parseReport)
	}
	if runtime2.GetCapabilityReport() != parseExpectedReport {
		parseT.Fatalf("expected stored capability report %+v after runtime init, got %+v", parseExpectedReport, runtime2.GetCapabilityReport())
	}
}

// TestSetCapabilityReportOverrideAffectsTransportSelectionDeterministically verifies package-level capability overrides drive deterministic snapshot and patch transport-tier selection.
func TestSetCapabilityReportOverrideAffectsTransportSelectionDeterministically(parseT *testing.T) {
	runtime2.ResetCapabilityReport()
	parseT.Cleanup(runtime2.ResetCapabilityReport)

	parseSharedOverrideErr := runtime2.SetCapabilityReportOverride(runtime2.CapabilityReport{
		HasWorkerSupport:                true,
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasBinaryTransportSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
	if parseSharedOverrideErr != nil {
		parseT.Fatalf("SetCapabilityReportOverride(shared) returned error: %v", parseSharedOverrideErr)
	}
	parseSharedSnapshotTier, parseSharedSnapshotTierErr := runtime2.SelectSnapshotTransportTier(runtime2.GetCapabilityReport())
	if parseSharedSnapshotTierErr != nil {
		parseT.Fatalf("SelectSnapshotTransportTier(shared-capabilities) returned error: %v", parseSharedSnapshotTierErr)
	}
	if parseSharedSnapshotTier != runtime2.TransportTierBinary {
		parseT.Fatalf("expected binary snapshot transport tier, got %q", parseSharedSnapshotTier)
	}
	parseSharedPatchTier, parseSharedPatchTierErr := runtime2.SelectPatchTransportTier(runtime2.GetCapabilityReport(), true)
	if parseSharedPatchTierErr != nil {
		parseT.Fatalf("SelectPatchTransportTier(shared) returned error: %v", parseSharedPatchTierErr)
	}
	if parseSharedPatchTier != runtime2.TransportTierSharedBuffer {
		parseT.Fatalf("expected shared patch transport tier, got %q", parseSharedPatchTier)
	}

	parseStructuredOverrideErr := runtime2.SetCapabilityReportOverride(runtime2.CapabilityReport{
		HasWorkerSupport:          true,
		HasMessagePortSupport:     true,
		HasStructuredCloneSupport: true,
	})
	if parseStructuredOverrideErr != nil {
		parseT.Fatalf("SetCapabilityReportOverride(structured) returned error: %v", parseStructuredOverrideErr)
	}
	parseStructuredSnapshotTier, parseStructuredSnapshotTierErr := runtime2.SelectSnapshotTransportTier(runtime2.GetCapabilityReport())
	if parseStructuredSnapshotTierErr != nil {
		parseT.Fatalf("SelectSnapshotTransportTier(structured) returned error: %v", parseStructuredSnapshotTierErr)
	}
	if parseStructuredSnapshotTier != runtime2.TransportTierStructuredClone {
		parseT.Fatalf("expected structured snapshot transport tier, got %q", parseStructuredSnapshotTier)
	}
	parseStructuredPatchTier, parseStructuredPatchTierErr := runtime2.SelectPatchTransportTier(runtime2.GetCapabilityReport(), false)
	if parseStructuredPatchTierErr != nil {
		parseT.Fatalf("SelectPatchTransportTier(structured) returned error: %v", parseStructuredPatchTierErr)
	}
	if parseStructuredPatchTier != runtime2.TransportTierStructuredClone {
		parseT.Fatalf("expected structured patch transport tier, got %q", parseStructuredPatchTier)
	}

	runtime2.ResetCapabilityReport()
	if _, parseResetSnapshotTierErr := runtime2.SelectSnapshotTransportTier(runtime2.GetCapabilityReport()); parseResetSnapshotTierErr == nil {
		parseT.Fatal("expected reset capability report to reject snapshot transport selection")
	}
	if _, parseResetPatchTierErr := runtime2.SelectPatchTransportTier(runtime2.GetCapabilityReport(), false); parseResetPatchTierErr == nil {
		parseT.Fatal("expected reset capability report to reject patch transport selection")
	}
}
