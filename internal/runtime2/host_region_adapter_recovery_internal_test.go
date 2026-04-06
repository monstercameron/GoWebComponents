package runtime2

import "testing"

// buildMountedHostRegionAdapterForRecoveryInternalTest creates one mounted host region adapter for direct recovery helper coverage.
func buildMountedHostRegionAdapterForRecoveryInternalTest(parseT *testing.T) *HostRegionAdapter {
	parseT.Helper()
	parseHostRegionAdapter, parseAdapterErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseAdapterErr)
	}
	if _, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
	}, 2); parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	return parseHostRegionAdapter
}

// TestHostRegionRecoveryStateGettersAndDiagnosticCopies verifies recovery-state getters reflect adapter flags and diagnostic getters return defensive copies.
func TestHostRegionRecoveryStateGettersAndDiagnosticCopies(parseT *testing.T) {
	var parseNilHostRegionAdapter *HostRegionAdapter
	if parseNilHostRegionAdapter.GetHostRegionIsFallbackPending() {
		parseT.Fatal("expected nil fallback-pending getter to be false")
	}
	if parseNilHostRegionAdapter.GetHostRegionIsFallbackActive() {
		parseT.Fatal("expected nil fallback-active getter to be false")
	}
	if parseNilHostRegionAdapter.GetHostRegionIsRepairPending() {
		parseT.Fatal("expected nil repair-pending getter to be false")
	}
	if parseNilHostRegionAdapter.IsHostRegionLocalShellOwnership() {
		parseT.Fatal("expected nil local-shell-ownership getter to be false")
	}
	if parseNilHostRegionAdapter.GetHostRegionIsHydrationComplete() {
		parseT.Fatal("expected nil hydration-complete getter to be false")
	}
	if parseNilHostRegionAdapter.HasHostRegionPostHydrationAttached() {
		parseT.Fatal("expected nil post-hydration-attached getter to be false")
	}
	if parseNilHostRegionAdapter.GetHostRegionRepairRemountEpoch() != 0 {
		parseT.Fatal("expected nil repair-remount epoch getter to be zero")
	}
	if parseNilHostRegionAdapter.GetHostRegionRepairVersionFloor() != 0 {
		parseT.Fatal("expected nil repair-version floor getter to be zero")
	}
	if parseNilHostRegionAdapter.GetHostRegionLatestValidVersion() != 0 {
		parseT.Fatal("expected nil latest-valid-version getter to be zero")
	}
	if parseNilHostRegionAdapter.GetHostRegionDiagnosticRing() != nil {
		parseT.Fatal("expected nil diagnostic ring getter to be nil")
	}
	if parseNilHostRegionAdapter.HasHostRegionHydratedShellAnchor() {
		parseT.Fatal("expected nil hydrated-shell-anchor getter to be false")
	}

	parseHostRegionAdapter := &HostRegionAdapter{
		isHostRegionFallbackPending:        true,
		isHostRegionFallbackActive:         true,
		isHostRegionRepairPending:          true,
		isHostRegionLocalShellOwned:        true,
		isHostRegionHydrationComplete:      true,
		hasHostRegionPostHydrationAttached: true,
		storeHostRegionRepairRemountEpoch:  9,
		storeHostRegionRepairVersionFloor:  11,
		storeHostRegionLatestValidVersion:  13,
		storeHostRegionSnapshotTier:        TransportTierBinary,
		hasHostRegionSnapshotDowngrade:     true,
		storeHostRegionSnapshotDowngrade: DiagnosticDowngradeReason{
			Path:   DiagnosticDowngradePathBinary,
			Reason: "decode-failed",
		},
		storeHostRegionPatchTier:    TransportTierSharedBuffer,
		hasHostRegionPatchDowngrade: true,
		storeHostRegionPatchDowngrade: DiagnosticDowngradeReason{
			Path:   DiagnosticDowngradePathSharedMemory,
			Reason: "invalid-shared-page",
		},
		storeHostRegionDiagnosticRing: []ControlEnvelope{{
			DiagnosticTiming: &DiagnosticTimingMetrics{QueueNanos: 1},
			DiagnosticSize:   &DiagnosticSizeMetrics{SnapshotBytes: 2},
			DiagnosticFallback: &DiagnosticFallbackReason{
				Domain: "transport",
				Reason: string(TransportFailureKindMalformedPatchPayload),
			},
			DiagnosticTrace: &DiagnosticTraceMetadata{
				IsDebug:            true,
				TraceID:            "trace-1",
				SchedulerAttemptID: 1,
				WorkerAttemptID:    2,
				CommitAttemptID:    3,
			},
			DiagnosticDowngrade: &DiagnosticDowngradeReason{
				Path:   DiagnosticDowngradePathBinary,
				Reason: "decode-failed",
			},
		}},
		hasHostRegionHydratedShellAnchor: true,
	}
	if !parseHostRegionAdapter.GetHostRegionIsFallbackPending() || !parseHostRegionAdapter.GetHostRegionIsFallbackActive() || !parseHostRegionAdapter.GetHostRegionIsRepairPending() || !parseHostRegionAdapter.IsHostRegionLocalShellOwnership() {
		parseT.Fatalf("expected recovery-state getters to reflect active flags, got %+v", parseHostRegionAdapter)
	}
	if !parseHostRegionAdapter.GetHostRegionIsHydrationComplete() || !parseHostRegionAdapter.HasHostRegionPostHydrationAttached() {
		parseT.Fatalf("expected hydration-state getters to reflect stored flags, got %+v", parseHostRegionAdapter)
	}
	if parseHostRegionAdapter.GetHostRegionRepairRemountEpoch() != 9 || parseHostRegionAdapter.GetHostRegionRepairVersionFloor() != 11 || parseHostRegionAdapter.GetHostRegionLatestValidVersion() != 13 {
		parseT.Fatalf("expected repair/version getters to reflect stored counters, got %+v", parseHostRegionAdapter)
	}
	if !parseHostRegionAdapter.HasHostRegionHydratedShellAnchor() {
		parseT.Fatal("expected hydrated-shell-anchor getter to reflect stored flag")
	}
	parseDowngradeStatus := parseHostRegionAdapter.GetHostRegionTransportDowngradeStatus()
	if parseDowngradeStatus.GetSnapshotTransportTier != TransportTierBinary || !parseDowngradeStatus.HasSnapshotDowngrade || parseDowngradeStatus.GetPatchTransportTier != TransportTierSharedBuffer || !parseDowngradeStatus.HasPatchDowngrade {
		parseT.Fatalf("unexpected downgrade status: %+v", parseDowngradeStatus)
	}

	parseDiagnosticRing := parseHostRegionAdapter.GetHostRegionDiagnosticRing()
	if len(parseDiagnosticRing) != 1 || parseDiagnosticRing[0].DiagnosticTiming == nil || parseDiagnosticRing[0].DiagnosticSize == nil || parseDiagnosticRing[0].DiagnosticFallback == nil || parseDiagnosticRing[0].DiagnosticTrace == nil || parseDiagnosticRing[0].DiagnosticDowngrade == nil {
		parseT.Fatalf("expected diagnostic ring copy with all nested payloads present, got %+v", parseDiagnosticRing)
	}
	parseDiagnosticRing[0].DiagnosticTiming.QueueNanos = 99
	parseDiagnosticRing[0].DiagnosticSize.SnapshotBytes = 88
	parseDiagnosticRing[0].DiagnosticFallback.Reason = "mutated"
	parseDiagnosticRing[0].DiagnosticTrace.TraceID = "mutated"
	parseDiagnosticRing[0].DiagnosticDowngrade.Reason = "mutated"
	if parseHostRegionAdapter.storeHostRegionDiagnosticRing[0].DiagnosticTiming.QueueNanos != 1 ||
		parseHostRegionAdapter.storeHostRegionDiagnosticRing[0].DiagnosticSize.SnapshotBytes != 2 ||
		parseHostRegionAdapter.storeHostRegionDiagnosticRing[0].DiagnosticFallback.Reason != string(TransportFailureKindMalformedPatchPayload) ||
		parseHostRegionAdapter.storeHostRegionDiagnosticRing[0].DiagnosticTrace.TraceID != "trace-1" ||
		parseHostRegionAdapter.storeHostRegionDiagnosticRing[0].DiagnosticDowngrade.Reason != "decode-failed" {
		parseT.Fatalf("expected diagnostic ring getter to return deep copies, got %+v", parseHostRegionAdapter.storeHostRegionDiagnosticRing[0])
	}
}

// TestHostRegionRecoveryHydrationAndShellGuards verifies hydration attach, post-render attach, and shell-anchor helpers enforce mounted-region and sequencing rules.
func TestHostRegionRecoveryHydrationAndShellGuards(parseT *testing.T) {
	var parseNilHostRegionAdapter *HostRegionAdapter
	if parseErr := parseNilHostRegionAdapter.HandleHostRegionHydrationComplete(); parseErr == nil {
		parseT.Fatal("expected nil hydration-complete call to fail")
	}
	if _, parseErr := parseNilHostRegionAdapter.HandleHostRegionPostHydrationAttach(); parseErr == nil {
		parseT.Fatal("expected nil post-hydration attach to fail")
	}
	if parseErr := parseNilHostRegionAdapter.HandleHostRegionPostRenderAttach(); parseErr == nil {
		parseT.Fatal("expected nil post-render attach to fail")
	}
	if parseErr := parseNilHostRegionAdapter.HandleHostRegionRegisterHydratedShellAnchor(1, "div"); parseErr == nil {
		parseT.Fatal("expected nil hydrated shell anchor registration to fail")
	}
	if _, parseErr := parseNilHostRegionAdapter.HandleHostRegionShellMissingAnchorDetection(); parseErr == nil {
		parseT.Fatal("expected nil shell-missing-anchor detection to fail")
	}

	parseHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryInternalTest(parseT)
	parseShellAnchorCheck, parseShellAnchorErr := parseHostRegionAdapter.HandleHostRegionShellMissingAnchorDetection()
	if parseShellAnchorErr != nil {
		parseT.Fatalf("HandleHostRegionShellMissingAnchorDetection returned error: %v", parseShellAnchorErr)
	}
	if !parseShellAnchorCheck.HasMissingAnchor {
		parseT.Fatalf("expected shell anchor to be missing before registration, got %+v", parseShellAnchorCheck)
	}
	parseHydrationAttachResult, parseHydrationAttachErr := parseHostRegionAdapter.HandleHostRegionPostHydrationAttach()
	if parseHydrationAttachErr == nil || !parseHydrationAttachResult.HasBlocked {
		parseT.Fatalf("expected post-hydration attach before hydration complete to block, got (%+v, %v)", parseHydrationAttachResult, parseHydrationAttachErr)
	}
	if parseHydrationErr := parseHostRegionAdapter.HandleHostRegionHydrationComplete(); parseHydrationErr != nil {
		parseT.Fatalf("HandleHostRegionHydrationComplete returned error: %v", parseHydrationErr)
	}
	if parseHostRegionAdapter.HandleHostRegionRegisterHydratedShellAnchor(0, "div") == nil {
		parseT.Fatal("expected zero node ID hydrated shell anchor registration to fail")
	}
	if parseHostRegionAdapter.HandleHostRegionRegisterHydratedShellAnchor(1, "") == nil {
		parseT.Fatal("expected empty tag hydrated shell anchor registration to fail")
	}
	parseHydrationAttachResult, parseHydrationAttachErr = parseHostRegionAdapter.HandleHostRegionPostHydrationAttach()
	if parseHydrationAttachErr == nil || !parseHydrationAttachResult.HasBlocked {
		parseT.Fatalf("expected post-hydration attach before shell anchor registration to block, got (%+v, %v)", parseHydrationAttachResult, parseHydrationAttachErr)
	}
	if parseAnchorErr := parseHostRegionAdapter.HandleHostRegionRegisterHydratedShellAnchor(1, "div"); parseAnchorErr != nil {
		parseT.Fatalf("HandleHostRegionRegisterHydratedShellAnchor returned error: %v", parseAnchorErr)
	}
	parseShellAnchorCheck, parseShellAnchorErr = parseHostRegionAdapter.HandleHostRegionShellMissingAnchorDetection()
	if parseShellAnchorErr != nil {
		parseT.Fatalf("HandleHostRegionShellMissingAnchorDetection(after anchor) returned error: %v", parseShellAnchorErr)
	}
	if parseShellAnchorCheck.HasMissingAnchor {
		parseT.Fatalf("expected shell anchor registration to clear missing-anchor flag, got %+v", parseShellAnchorCheck)
	}
	parseHydrationAttachResult, parseHydrationAttachErr = parseHostRegionAdapter.HandleHostRegionPostHydrationAttach()
	if parseHydrationAttachErr != nil {
		parseT.Fatalf("HandleHostRegionPostHydrationAttach returned error: %v", parseHydrationAttachErr)
	}
	if !parseHydrationAttachResult.HasAttached || !parseHostRegionAdapter.HasHostRegionPostHydrationAttached() || !parseHostRegionAdapter.GetHostRegionIsHydrationComplete() {
		parseT.Fatalf("expected successful post-hydration attach state, got (%+v, %+v)", parseHydrationAttachResult, parseHostRegionAdapter)
	}
	if parsePostRenderErr := parseHostRegionAdapter.HandleHostRegionPostRenderAttach(); parsePostRenderErr != nil {
		parseT.Fatalf("HandleHostRegionPostRenderAttach returned error: %v", parsePostRenderErr)
	}
	parseIdentityResult, parseIdentityErr := parseHostRegionAdapter.HandleHostRegionShellIdentityMismatchDetection(SSRShellMarker{
		Version:          SSRShellMarkerVersionV1,
		RegionInstanceID: RegionInstanceID("region-2"),
		RendererID:       RendererID("dashboard.hot-panel"),
	})
	if parseIdentityErr != nil {
		parseT.Fatalf("HandleHostRegionShellIdentityMismatchDetection returned error: %v", parseIdentityErr)
	}
	if !parseIdentityResult.HasMismatch || !parseIdentityResult.HasRegionIDMismatch || parseIdentityResult.HasRendererIDMismatch {
		parseT.Fatalf("expected region-ID mismatch only, got %+v", parseIdentityResult)
	}
}

// TestHostRegionRecoveryFailureRoutesRejectNilAndUnmountedAdapters verifies failure-routing helpers reject nil or unmounted adapters before mutating recovery state.
func TestHostRegionRecoveryFailureRoutesRejectNilAndUnmountedAdapters(parseT *testing.T) {
	var parseNilHostRegionAdapter *HostRegionAdapter
	if parseErr := parseNilHostRegionAdapter.HandleHostRegionStructuredCloneDecodeFailure(3); parseErr == nil {
		parseT.Fatal("expected nil structured-clone decode failure route to fail")
	}
	if parseErr := parseNilHostRegionAdapter.HandleHostRegionBinaryDecodeFailure(3); parseErr == nil {
		parseT.Fatal("expected nil binary decode failure route to fail")
	}
	if parseErr := parseNilHostRegionAdapter.HandleHostRegionSharedPageDecodeFailure(3); parseErr == nil {
		parseT.Fatal("expected nil shared-page decode failure route to fail")
	}
	if parseErr := parseNilHostRegionAdapter.HandleHostRegionDOMPatchTransactionFailure(DOMCommitFailureKindMissingNodeLookup, 3); parseErr == nil {
		parseT.Fatal("expected nil DOM patch transaction failure route to fail")
	}
	if _, parseErr := parseNilHostRegionAdapter.HandleHostRegionFallbackOwnershipBegin(); parseErr == nil {
		parseT.Fatal("expected nil fallback ownership begin to fail")
	}

	parseHostRegionAdapter, parseAdapterErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseAdapterErr)
	}
	if parseErr := parseHostRegionAdapter.HandleHostRegionStructuredCloneDecodeFailure(3); parseErr == nil {
		parseT.Fatal("expected unmounted structured-clone decode failure route to fail")
	}
	if parseErr := parseHostRegionAdapter.HandleHostRegionDOMPatchTransactionFailure(DOMCommitFailureKindMissingNodeLookup, 3); parseErr == nil {
		parseT.Fatal("expected unmounted DOM patch transaction failure route to fail")
	}
	if _, parseErr := parseHostRegionAdapter.HandleHostRegionFallbackOwnershipBegin(); parseErr == nil {
		parseT.Fatal("expected fallback ownership begin without pending fallback to fail")
	}
}
