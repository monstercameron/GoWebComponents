package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestGetHostRegionRuntimeStatusReportsLocalShellMode verifies mounted local-first regions report local-shell runtime status before worker attach.
func TestGetHostRegionRuntimeStatusReportsLocalShellMode(parseT *testing.T) {
	buildHostRegionAdapter, parseBuildErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	if _, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			SourceIDs:        []string{"status", "count", "status"},
		},
		1,
	); parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	getRuntimeStatus, hasRuntimeStatus := buildHostRegionAdapter.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		parseT.Fatal("expected runtime status for mounted region")
	}
	if getRuntimeStatus.GetRegionMode != runtime2.HostRegionRuntimeModeLocalShell {
		parseT.Fatalf("expected local-shell runtime mode, got %q", getRuntimeStatus.GetRegionMode)
	}
	if getRuntimeStatus.GetRendererID != runtime2.RendererID("dashboard.hot-panel") {
		parseT.Fatalf("expected renderer dashboard.hot-panel, got %q", getRuntimeStatus.GetRendererID)
	}
	if getRuntimeStatus.GetEpoch != 1 {
		parseT.Fatalf("expected epoch 1, got %d", getRuntimeStatus.GetEpoch)
	}
	if getRuntimeStatus.GetIsHydrationComplete {
		parseT.Fatal("expected hydration-complete state false before hydration flow")
	}
	if getRuntimeStatus.HasHydratedShellAnchor {
		parseT.Fatal("expected hydrated shell anchor false before hydration flow")
	}
	if getRuntimeStatus.HasPostHydrationAttached {
		parseT.Fatal("expected post-hydration attached false before hydration flow")
	}
	if getRuntimeStatus.GetTransportTier != runtime2.TransportTierStructuredClone {
		parseT.Fatalf("expected default transport tier structured-clone, got %q", getRuntimeStatus.GetTransportTier)
	}
	if getRuntimeStatus.HasSnapshotDowngrade || getRuntimeStatus.HasPatchDowngrade {
		parseT.Fatal("expected no downgrade accounting in initial local-shell status")
	}
	if getRuntimeStatus.GetDroppedStalePatchCount != 0 || getRuntimeStatus.GetIgnoredStaleDiagnosticCount != 0 {
		parseT.Fatalf(
			"expected zero stale counters in initial local-shell status, got patch=%d diagnostic=%d",
			getRuntimeStatus.GetDroppedStalePatchCount,
			getRuntimeStatus.GetIgnoredStaleDiagnosticCount,
		)
	}
	if getRuntimeStatus.GetFallbackReason != "" {
		parseT.Fatalf("expected empty fallback reason outside fallback mode, got %q", getRuntimeStatus.GetFallbackReason)
	}
}

// TestGetHostRegionRuntimeStatusReportsWorkerAttachedMode verifies post-hydration attach and transport selection are reflected in runtime status.
func TestGetHostRegionRuntimeStatusReportsWorkerAttachedMode(parseT *testing.T) {
	buildHostRegionAdapter, parseBuildErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	if _, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	); parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	parseCapabilityReport := runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
		HasWorkerSupport:                true,
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasBinaryTransportSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
	buildSharedSnapshotPage, parseSharedPageErr := runtime2.BuildSharedSnapshotPage(4096)
	if parseSharedPageErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseSharedPageErr)
	}
	if _, parseDispatchErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatchWithTransport(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
		},
		2,
		parseCapabilityReport,
		buildSharedSnapshotPage,
	); parseDispatchErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatchWithTransport returned error: %v", parseDispatchErr)
	}
	if parseHydrationErr := buildHostRegionAdapter.HandleHostRegionHydrationComplete(); parseHydrationErr != nil {
		parseT.Fatalf("HandleHostRegionHydrationComplete returned error: %v", parseHydrationErr)
	}
	if parseAnchorErr := buildHostRegionAdapter.HandleHostRegionRegisterHydratedShellAnchor(1, "div"); parseAnchorErr != nil {
		parseT.Fatalf("HandleHostRegionRegisterHydratedShellAnchor returned error: %v", parseAnchorErr)
	}
	if _, parseAttachErr := buildHostRegionAdapter.HandleHostRegionPostHydrationAttach(); parseAttachErr != nil {
		parseT.Fatalf("HandleHostRegionPostHydrationAttach returned error: %v", parseAttachErr)
	}
	getRuntimeStatus, hasRuntimeStatus := buildHostRegionAdapter.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		parseT.Fatal("expected runtime status for mounted region")
	}
	if getRuntimeStatus.GetRegionMode != runtime2.HostRegionRuntimeModeWorkerAttached {
		parseT.Fatalf("expected worker-attached runtime mode, got %q", getRuntimeStatus.GetRegionMode)
	}
	if getRuntimeStatus.GetTransportTier != runtime2.TransportTierSharedBuffer {
		parseT.Fatalf("expected shared-buffer transport tier, got %q", getRuntimeStatus.GetTransportTier)
	}
	if !getRuntimeStatus.GetIsHydrationComplete {
		parseT.Fatal("expected hydration-complete state true after hydration flow")
	}
	if !getRuntimeStatus.HasHydratedShellAnchor {
		parseT.Fatal("expected hydrated shell anchor true after anchor registration")
	}
	if !getRuntimeStatus.HasPostHydrationAttached {
		parseT.Fatal("expected post-hydration attached true after attach flow")
	}
	if getRuntimeStatus.GetLastSnapshotVersion != 2 {
		parseT.Fatalf("expected last snapshot version 2, got %d", getRuntimeStatus.GetLastSnapshotVersion)
	}
	if getRuntimeStatus.HasSnapshotDowngrade || getRuntimeStatus.HasPatchDowngrade {
		parseT.Fatal("expected no downgrade accounting when shared transport succeeds")
	}
	if getRuntimeStatus.GetFallbackReason != "" {
		parseT.Fatalf("expected empty fallback reason outside fallback mode, got %q", getRuntimeStatus.GetFallbackReason)
	}
}

// TestGetHostRegionRuntimeStatusReportsFallbackMode verifies fallback runtime mode includes fallback reason context.
func TestGetHostRegionRuntimeStatusReportsFallbackMode(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if parseBinaryErr := buildHostRegionAdapter.HandleHostRegionBinaryDecodeFailure(6); parseBinaryErr != nil {
		parseT.Fatalf("HandleHostRegionBinaryDecodeFailure returned error: %v", parseBinaryErr)
	}
	if _, parseFallbackErr := buildHostRegionAdapter.HandleHostRegionFallbackOwnershipBegin(); parseFallbackErr != nil {
		parseT.Fatalf("HandleHostRegionFallbackOwnershipBegin returned error: %v", parseFallbackErr)
	}
	getRuntimeStatus, hasRuntimeStatus := buildHostRegionAdapter.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		parseT.Fatal("expected runtime status for mounted region")
	}
	if getRuntimeStatus.GetRegionMode != runtime2.HostRegionRuntimeModeFallback {
		parseT.Fatalf("expected fallback runtime mode, got %q", getRuntimeStatus.GetRegionMode)
	}
	if getRuntimeStatus.GetFallbackReason != string(runtime2.TransportFailureKindMalformedPatchPayload) {
		parseT.Fatalf(
			"expected fallback reason %q, got %q",
			runtime2.TransportFailureKindMalformedPatchPayload,
			getRuntimeStatus.GetFallbackReason,
		)
	}
}

// TestGetHostRegionRuntimeStatusReportsDowngradeReasonsAndStaleCounters verifies runtime status includes downgrade reasons and stale counter snapshots.
func TestGetHostRegionRuntimeStatusReportsDowngradeReasonsAndStaleCounters(parseT *testing.T) {
	buildHostRegionAdapter, parseBuildErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	if _, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	); parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	parseCapabilityReport := runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
		HasWorkerSupport:                true,
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasBinaryTransportSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
	if _, parseDispatchErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatchWithTransport(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
		},
		2,
		parseCapabilityReport,
		nil,
	); parseDispatchErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatchWithTransport returned error: %v", parseDispatchErr)
	}
	parseDiagnosticEnvelope, parseDiagnosticErr := runtime2.BuildControlDiagnosticEnvelope("region-1", runtime2.ControlDiagnosticEnvelopeSpec{
		DiagnosticType: runtime2.DiagnosticEventKindPatchReady,
		TransportTier:  runtime2.TransportTierStructuredClone,
		DiagnosticDowngrade: &runtime2.DiagnosticDowngradeReason{
			Path:   runtime2.DiagnosticDowngradePathSharedMemory,
			Reason: string(runtime2.SharedPatchDowngradeReasonInvalidSharedPage),
		},
		DiagnosticText: "patch downgrade",
	})
	if parseDiagnosticErr != nil {
		parseT.Fatalf("BuildControlDiagnosticEnvelope returned error: %v", parseDiagnosticErr)
	}
	if _, parseControlDispatchErr := runtime2.HandleHostControlEnvelope(buildHostRegionAdapter, parseDiagnosticEnvelope); parseControlDispatchErr != nil {
		parseT.Fatalf("HandleHostControlEnvelope(diagnostic) returned error: %v", parseControlDispatchErr)
	}
	if _, parseCounterErr := buildHostRegionAdapter.GetHostRegionCoordinator().IncrementRegionDroppedStalePatchCount(runtime2.RegionInstanceID("region-1")); parseCounterErr != nil {
		parseT.Fatalf("IncrementRegionDroppedStalePatchCount returned error: %v", parseCounterErr)
	}
	if _, parseCounterErr := buildHostRegionAdapter.GetHostRegionCoordinator().IncrementRegionIgnoredStaleDiagnosticCount(runtime2.RegionInstanceID("region-1")); parseCounterErr != nil {
		parseT.Fatalf("IncrementRegionIgnoredStaleDiagnosticCount returned error: %v", parseCounterErr)
	}
	getRuntimeStatus, hasRuntimeStatus := buildHostRegionAdapter.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		parseT.Fatal("expected runtime status for mounted region")
	}
	if !getRuntimeStatus.HasSnapshotDowngrade {
		parseT.Fatal("expected snapshot downgrade state in runtime status")
	}
	if getRuntimeStatus.GetSnapshotDowngradePath != runtime2.DiagnosticDowngradePathSharedMemory {
		parseT.Fatalf("expected snapshot downgrade path %q, got %q", runtime2.DiagnosticDowngradePathSharedMemory, getRuntimeStatus.GetSnapshotDowngradePath)
	}
	if getRuntimeStatus.GetSnapshotDowngradeReason != string(runtime2.SharedSnapshotDowngradeReasonSharedMemoryUnavailable) {
		parseT.Fatalf(
			"expected snapshot downgrade reason %q, got %q",
			runtime2.SharedSnapshotDowngradeReasonSharedMemoryUnavailable,
			getRuntimeStatus.GetSnapshotDowngradeReason,
		)
	}
	if !getRuntimeStatus.HasPatchDowngrade {
		parseT.Fatal("expected patch downgrade state in runtime status")
	}
	if getRuntimeStatus.GetPatchDowngradePath != runtime2.DiagnosticDowngradePathSharedMemory {
		parseT.Fatalf("expected patch downgrade path %q, got %q", runtime2.DiagnosticDowngradePathSharedMemory, getRuntimeStatus.GetPatchDowngradePath)
	}
	if getRuntimeStatus.GetPatchDowngradeReason != string(runtime2.SharedPatchDowngradeReasonInvalidSharedPage) {
		parseT.Fatalf(
			"expected patch downgrade reason %q, got %q",
			runtime2.SharedPatchDowngradeReasonInvalidSharedPage,
			getRuntimeStatus.GetPatchDowngradeReason,
		)
	}
	if getRuntimeStatus.GetDroppedStalePatchCount != 1 {
		parseT.Fatalf("expected dropped stale patch counter 1, got %d", getRuntimeStatus.GetDroppedStalePatchCount)
	}
	if getRuntimeStatus.GetIgnoredStaleDiagnosticCount != 1 {
		parseT.Fatalf("expected ignored stale diagnostic counter 1, got %d", getRuntimeStatus.GetIgnoredStaleDiagnosticCount)
	}
}
