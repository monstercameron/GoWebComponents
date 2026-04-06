package runtime2

import (
	"crypto/sha256"
	"fmt"
	"time"
)

func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionStructuralRemount(
	parseSpec ParallelRegionSpec,
	parseIsLocalShellOwned bool,
) (HostRegionStructuralRemountResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionStructuralRemountResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	getSpec, parseSpecErr := NormalizeParallelRegionSpec(parseSpec)
	if parseSpecErr != nil {
		return HostRegionStructuralRemountResult{}, parseSpecErr
	}
	if getSpec.RegionInstanceID != parseHostRegionAdapter.storeRegionInstanceID {
		return HostRegionStructuralRemountResult{}, fmt.Errorf(
			"runtime2: host region adapter mounted for %q cannot remount region %q",
			parseHostRegionAdapter.storeRegionInstanceID,
			getSpec.RegionInstanceID,
		)
	}
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(getSpec.RegionInstanceID)
	if !hasCoordinatorEntry {
		return HostRegionStructuralRemountResult{}, fmt.Errorf("runtime2: host region %q is not mounted", getSpec.RegionInstanceID)
	}
	hasRendererChanged := getCoordinatorEntry.RendererID != getSpec.RendererID
	hasShellOwnershipChanged := parseHostRegionAdapter.isHostRegionLocalShellOwned != parseIsLocalShellOwned
	if !hasRendererChanged && !hasShellOwnershipChanged {
		return HostRegionStructuralRemountResult{
			HasRendererChanged:       false,
			HasShellOwnershipChanged: false,
		}, nil
	}
	getRemountEpoch := getCoordinatorEntry.Epoch + 1
	parseHostRegionAdapter.storeScheduler.HandleSchedulerDispose(string(getSpec.RegionInstanceID))
	if parseDisposeErr := parseHostRegionAdapter.storeCoordinator.DisposeRegion(getSpec.RegionInstanceID); parseDisposeErr != nil {
		return HostRegionStructuralRemountResult{}, parseDisposeErr
	}
	getSchedulerJob, parseSchedulerErr := parseHostRegionAdapter.storeScheduler.HandleSchedulerMount(string(getSpec.RegionInstanceID))
	if parseSchedulerErr != nil {
		return HostRegionStructuralRemountResult{}, parseSchedulerErr
	}
	if parseMountErr := parseHostRegionAdapter.storeCoordinator.MountRegion(CoordinatorEntry{
		RegionInstanceID:    getSpec.RegionInstanceID,
		RendererID:          getSpec.RendererID,
		SourceIDs:           getSpec.SourceIDs,
		Epoch:               getRemountEpoch,
		AssignedWorkerShard: string(getSchedulerJob.GetSchedulerShardID),
	}); parseMountErr != nil {
		return HostRegionStructuralRemountResult{}, parseMountErr
	}
	parseHostRegionAdapter.isHostRegionLocalShellOwned = parseIsLocalShellOwned
	parseHostRegionAdapter.storeHostRegionSnapshotFingerprint = ""
	parseHostRegionAdapter.storeHostRegionSnapshotHash = [sha256.Size]byte{}
	parseHostRegionAdapter.hasHostRegionSnapshotHash = false
	parseHostRegionAdapter.storeHostRegionSnapshotFastHash = 0
	parseHostRegionAdapter.hasHostRegionSnapshotFastHash = false
	parseHostRegionAdapter.storeHostRegionDispatchHash = [sha256.Size]byte{}
	parseHostRegionAdapter.hasHostRegionDispatchHash = false
	parseHostRegionAdapter.storeHostRegionDispatchBytes = nil
	parseHostRegionAdapter.hasHostRegionDispatchBytes = false
	parseHostRegionAdapter.storeHostRegionDispatchFastHash = 0
	parseHostRegionAdapter.hasHostRegionDispatchFastHash = false
	parseHostRegionAdapter.storeHostRegionDispatchRendererID = ""
	parseHostRegionAdapter.storeHostRegionDispatchEpoch = 0
	parseHostRegionAdapter.storeHostRegionDispatchInputVersion = 0
	parseHostRegionAdapter.storeHostRegionDispatchSourceVersion = 0
	parseHostRegionAdapter.storeHostRegionDispatchSourceVersionTuple = nil
	parseHostRegionAdapter.storeHostRegionDispatchSourceVersionScratch = nil
	parseHostRegionAdapter.clearHostRegionDispatchPropsLayout()
	parseHostRegionAdapter.hasHostRegionDispatchSourceVersionTuple = false
	parseHostRegionAdapter.hasHostRegionDispatchVersionVector = false
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheState(
		getRemountEpoch,
		getSpec.RendererID,
		getSpec.SourceIDs,
		false,
		0,
		0,
	)
	parseHostRegionAdapter.clearHostRegionSourceSnapshotCache()
	parseHostRegionAdapter.storeHostRegionDeferredDispatch = hostRegionDeferredDispatch{}
	parseHostRegionAdapter.hasHostRegionDeferredDispatch = false
	parseHostRegionAdapter.isHostRegionFallbackPending = false
	parseHostRegionAdapter.isHostRegionFallbackActive = false
	parseHostRegionAdapter.isHostRegionRepairPending = false
	parseHostRegionAdapter.isHostRegionHydrationComplete = false
	parseHostRegionAdapter.hasHostRegionPostHydrationAttached = false
	parseHostRegionAdapter.hasHostRegionHydratedShellAnchor = false
	parseHostRegionAdapter.storeHostRegionRepairRemountEpoch = 0
	parseHostRegionAdapter.storeHostRegionRepairVersionFloor = 0
	parseHostRegionAdapter.storeHostRegionLastPatchVersion = 0
	parseHostRegionAdapter.storeHostRegionTransportTier = TransportTierStructuredClone
	parseHostRegionAdapter.storeHostRegionSnapshotTier = TransportTierStructuredClone
	parseHostRegionAdapter.storeHostRegionPatchTier = TransportTierStructuredClone
	parseHostRegionAdapter.storeHostRegionFallbackReason = ""
	parseHostRegionAdapter.storeHostRegionSnapshotDowngrade = DiagnosticDowngradeReason{}
	parseHostRegionAdapter.storeHostRegionPatchDowngrade = DiagnosticDowngradeReason{}
	parseHostRegionAdapter.storeHostRegionDispatchAt = time.Time{}
	parseHostRegionAdapter.storeHostRegionPatchReadyAt = time.Time{}
	parseHostRegionAdapter.storeHostRegionCommitAt = time.Time{}
	parseHostRegionAdapter.storeHostRegionDispatchToPatchNS = 0
	parseHostRegionAdapter.storeHostRegionDispatchToCommitNS = 0
	parseHostRegionAdapter.storeHostRegionPatchToCommitNS = 0
	parseHostRegionAdapter.storeHostRegionPatchIdempotency = BuildPatchIdempotencyTracker()
	parseHostRegionAdapter.hasHostRegionSnapshotDowngrade = false
	parseHostRegionAdapter.hasHostRegionPatchDowngrade = false
	parseHostRegionAdapter.storeRecoveryCoordinator.ClearRegionLocalFallback(string(getSpec.RegionInstanceID))
	parseHostRegionAdapter.storeScheduler.ClearSchedulerFallbackOwnership(string(getSpec.RegionInstanceID))
	return HostRegionStructuralRemountResult{
		HasRendererChanged:       hasRendererChanged,
		HasShellOwnershipChanged: hasShellOwnershipChanged,
		HasRemounted:             true,
		GetRemountEpoch:          getRemountEpoch,
	}, nil
}

func (parseHostRegionAdapter *HostRegionAdapter) handleHostRegionTransportDecodeFailure(
	parseFailureKind TransportFailureKind,
	parseInputVersion uint64,
) error {
	if parseHostRegionAdapter == nil {
		return fmt.Errorf("runtime2: host region adapter is nil")
	}
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	parseHostRegionAdapter.isHostRegionFallbackPending = true
	parseHostRegionAdapter.storeHostRegionRepairVersionFloor = parseHostRegionMaxVersion(
		parseHostRegionAdapter.storeHostRegionRepairVersionFloor,
		parseInputVersion,
		getCoordinatorEntry.LastCommittedVersion,
		getCoordinatorEntry.LastDispatchedVersion,
		parseHostRegionAdapter.storeHostRegionLatestValidVersion,
	)
	parseHostRegionAdapter.storeHostRegionFallbackReason = string(parseFailureKind)
	return parseHostRegionAdapter.storeRecoveryCoordinator.HandleTransportDecodeFailure(
		string(parseHostRegionAdapter.storeRegionInstanceID),
		parseFailureKind,
		getCoordinatorEntry.Epoch,
		parseInputVersion,
	)
}

// HandleHostRegionStructuredCloneDecodeFailure routes one structured-clone decode failure through recovery.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionStructuredCloneDecodeFailure(parseInputVersion uint64) error {
	return parseHostRegionAdapter.handleHostRegionTransportDecodeFailure(
		TransportFailureKindMalformedControlPayload,
		parseInputVersion,
	)
}

// HandleHostRegionBinaryDecodeFailure routes one binary decode failure through recovery.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionBinaryDecodeFailure(parseInputVersion uint64) error {
	return parseHostRegionAdapter.handleHostRegionTransportDecodeFailure(
		TransportFailureKindMalformedPatchPayload,
		parseInputVersion,
	)
}

// HandleHostRegionSharedPageDecodeFailure routes one shared-page decode failure through recovery.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionSharedPageDecodeFailure(parseInputVersion uint64) error {
	return parseHostRegionAdapter.handleHostRegionTransportDecodeFailure(
		TransportFailureKindMalformedSharedMemoryPage,
		parseInputVersion,
	)
}

// HandleHostRegionDOMPatchTransactionFailure routes one DOM patch-transaction failure through recovery.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionDOMPatchTransactionFailure(
	parseFailureKind DOMCommitFailureKind,
	parseInputVersion uint64,
) error {
	if parseHostRegionAdapter == nil {
		return fmt.Errorf("runtime2: host region adapter is nil")
	}
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	parseHostRegionAdapter.isHostRegionFallbackPending = true
	parseHostRegionAdapter.storeHostRegionRepairVersionFloor = parseHostRegionMaxVersion(
		parseHostRegionAdapter.storeHostRegionRepairVersionFloor,
		parseInputVersion,
		getCoordinatorEntry.LastCommittedVersion,
		getCoordinatorEntry.LastDispatchedVersion,
		parseHostRegionAdapter.storeHostRegionLatestValidVersion,
	)
	parseHostRegionAdapter.storeHostRegionFallbackReason = string(parseFailureKind)
	return parseHostRegionAdapter.storeRecoveryCoordinator.HandleDOMCommitFailure(
		string(parseHostRegionAdapter.storeRegionInstanceID),
		parseFailureKind,
		getCoordinatorEntry.Epoch,
		parseInputVersion,
	)
}

// HandleHostRegionFallbackOwnershipBegin mirrors recovery fallback ownership into coordinator and scheduler state.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionFallbackOwnershipBegin() (HostRegionFallbackMirrorResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionFallbackMirrorResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	getRegionID := string(parseHostRegionAdapter.storeRegionInstanceID)
	if !parseHostRegionAdapter.isHostRegionFallbackPending && !parseHostRegionAdapter.storeRecoveryCoordinator.IsRegionLocalOwnership(getRegionID) {
		return HostRegionFallbackMirrorResult{}, fmt.Errorf("runtime2: no pending recovery fallback for region %q", getRegionID)
	}
	if _, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID); !hasCoordinatorEntry {
		return HostRegionFallbackMirrorResult{}, fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	if parseFallbackErr := parseHostRegionAdapter.storeCoordinator.handleCoordinatorFallbackRegionTrusted(parseHostRegionAdapter.storeRegionInstanceID); parseFallbackErr != nil {
		return HostRegionFallbackMirrorResult{}, parseFallbackErr
	}
	hasSchedulerFallback := parseHostRegionAdapter.storeScheduler.HandleSchedulerFallback(getRegionID)
	parseHostRegionAdapter.isHostRegionFallbackPending = false
	parseHostRegionAdapter.isHostRegionFallbackActive = true
	if getFallbackState, hasFallbackState := parseHostRegionAdapter.storeRecoveryCoordinator.GetRegionFallbackState(getRegionID); hasFallbackState {
		parseHostRegionAdapter.storeHostRegionFallbackReason = getFallbackState.GetReason
	}
	return HostRegionFallbackMirrorResult{
		HasCoordinatorFallback: true,
		HasSchedulerFallback:   hasSchedulerFallback,
	}, nil
}

// HandleHostRegionFallbackMirror mirrors recovery fallback ownership into coordinator and scheduler state.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionFallbackMirror() (HostRegionFallbackMirrorResult, error) {
	return parseHostRegionAdapter.HandleHostRegionFallbackOwnershipBegin()
}

// HandleHostRegionPatchReady validates one patch-ready version against fallback and repair ownership gates.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionPatchReady(parseInputVersion uint64) (HostRegionPatchReadyResult, error) {
	return parseHostRegionAdapter.HandleHostRegionPatchReadyWithVersion(parseInputVersion, parseInputVersion)
}

// HandleHostRegionPatchReadyWithVersion validates one patch-ready patch and input version pair against fallback and repair ownership gates.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionPatchReadyWithVersion(parsePatchVersion uint64, parseInputVersion uint64) (HostRegionPatchReadyResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionPatchReadyResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parsePatchVersion == 0 {
		return HostRegionPatchReadyResult{}, fmt.Errorf("runtime2: patch-ready patch version is required")
	}
	if parseInputVersion == 0 {
		return HostRegionPatchReadyResult{}, fmt.Errorf("runtime2: patch-ready input version is required")
	}
	if parseHostRegionAdapter.isHostRegionRemoved {
		return HostRegionPatchReadyResult{HasIgnored: true, GetIgnoreReason: "owner-removed"}, nil
	}
	if parseHostRegionAdapter.isHostRegionFallbackPending {
		return HostRegionPatchReadyResult{HasIgnored: true, GetIgnoreReason: "fallback-pending"}, nil
	}
	if parseHostRegionAdapter.isHostRegionFallbackActive ||
		parseHostRegionAdapter.storeRecoveryCoordinator.IsRegionLocalOwnership(string(parseHostRegionAdapter.storeRegionInstanceID)) {
		return HostRegionPatchReadyResult{HasIgnored: true, GetIgnoreReason: "fallback-active"}, nil
	}
	if parseHostRegionAdapter.isHostRegionRepairPending {
		return HostRegionPatchReadyResult{HasIgnored: true, GetIgnoreReason: "repair-pending"}, nil
	}
	if _, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID); !hasCoordinatorEntry {
		return HostRegionPatchReadyResult{HasIgnored: true, GetIgnoreReason: "not-mounted"}, nil
	}
	if parseInputVersion < parseHostRegionAdapter.storeHostRegionRepairVersionFloor {
		if _, parseDropErr := parseHostRegionAdapter.storeCoordinator.handleCoordinatorIncrementRegionDroppedStalePatchCountTrusted(parseHostRegionAdapter.storeRegionInstanceID); parseDropErr != nil {
			return HostRegionPatchReadyResult{}, parseDropErr
		}
		return HostRegionPatchReadyResult{
			HasIgnored:      true,
			GetIgnoreReason: "stale-before-repair-floor",
		}, nil
	}
	if parsePatchVersion <= parseHostRegionAdapter.storeHostRegionLastPatchVersion {
		if _, parseDropErr := parseHostRegionAdapter.storeCoordinator.handleCoordinatorIncrementRegionDroppedStalePatchCountTrusted(parseHostRegionAdapter.storeRegionInstanceID); parseDropErr != nil {
			return HostRegionPatchReadyResult{}, parseDropErr
		}
		return HostRegionPatchReadyResult{
			HasIgnored:      true,
			GetIgnoreReason: "stale-patch-version",
		}, nil
	}
	if parseHostRegionAdapter.isHostRegionRoundTripTimingEnabled && parseHostRegionAdapter.storeHostRegionPatchReadyAt.IsZero() {
		parseHostRegionAdapter.storeHostRegionPatchReadyAt = time.Now()
		if !parseHostRegionAdapter.storeHostRegionDispatchAt.IsZero() {
			parseDispatchToPatchNS := parseHostRegionAdapter.storeHostRegionPatchReadyAt.Sub(parseHostRegionAdapter.storeHostRegionDispatchAt).Nanoseconds()
			if parseDispatchToPatchNS > 0 {
				parseHostRegionAdapter.storeHostRegionDispatchToPatchNS = uint64(parseDispatchToPatchNS)
			}
		}
	}
	return HostRegionPatchReadyResult{
		HasAccepted: true,
	}, nil
}

// HandleHostRegionWorkerDeath routes dead-worker handling into recovery and sets repair or fallback host ownership state.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionWorkerDeath(parseHasReassignSupport bool) (HostRegionWorkerDeathResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionWorkerDeathResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return HostRegionWorkerDeathResult{}, fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	getVersionFloor := parseHostRegionMaxVersion(
		getCoordinatorEntry.LastCommittedVersion,
		getCoordinatorEntry.LastDispatchedVersion,
		parseHostRegionAdapter.storeHostRegionRepairVersionFloor,
		parseHostRegionAdapter.storeHostRegionLatestValidVersion,
		parseHostRegionAdapter.storeRecoveryCoordinator.GetRegionLocalVersion(string(parseHostRegionAdapter.storeRegionInstanceID)),
	)
	if getVersionFloor == 0 {
		getVersionFloor = 1
	}
	getRemountEpoch := getCoordinatorEntry.Epoch + 1
	getRecoveryResult, parseRecoveryErr := parseHostRegionAdapter.storeRecoveryCoordinator.HandleWorkerDeath(
		string(parseHostRegionAdapter.storeRegionInstanceID),
		parseHasReassignSupport,
		getRemountEpoch,
		getVersionFloor,
	)
	if getRecoveryResult.HasReassigned {
		getCoordinatorEntry.Epoch = getRecoveryResult.GetRemountEpoch
		getCoordinatorEntry.LastDispatchedVersion = getVersionFloor
		getCoordinatorEntry.LastCommittedVersion = getVersionFloor
		getCoordinatorEntry.IsAttached = false
		getCoordinatorEntry.IsFallback = true
		getCoordinatorEntry.CurrentState = CoordinatorStateFallback
		if parseStoreErr := parseHostRegionAdapter.storeCoordinator.storeMutableEntry(getCoordinatorEntry); parseStoreErr != nil {
			return HostRegionWorkerDeathResult{}, parseStoreErr
		}
		parseHostRegionAdapter.storeHostRegionCoordinatorCacheState(
			getCoordinatorEntry.Epoch,
			getCoordinatorEntry.RendererID,
			getCoordinatorEntry.SourceIDs,
			getCoordinatorEntry.IsFallback,
			getCoordinatorEntry.LastSnapshotVersion,
			getCoordinatorEntry.LastDispatchedVersion,
		)
		parseHostRegionAdapter.storeScheduler.HandleSchedulerFallback(string(parseHostRegionAdapter.storeRegionInstanceID))
		parseHostRegionAdapter.storeRecoveryCoordinator.EnterRegionLocalFallback(
			string(parseHostRegionAdapter.storeRegionInstanceID),
			"worker-repair-pending",
			getRecoveryResult.GetRemountEpoch,
			getVersionFloor,
		)
		parseHostRegionAdapter.storeHostRegionFallbackReason = "worker-repair-pending"
		parseHostRegionAdapter.storeHostRegionRepairRemountEpoch = getRecoveryResult.GetRemountEpoch
		parseHostRegionAdapter.storeHostRegionRepairVersionFloor = getVersionFloor
		parseHostRegionAdapter.storeHostRegionLatestValidVersion = parseHostRegionMaxVersion(
			parseHostRegionAdapter.storeHostRegionLatestValidVersion,
			getVersionFloor,
		)
		parseHostRegionAdapter.isHostRegionRepairPending = true
		parseHostRegionAdapter.isHostRegionFallbackActive = true
		parseHostRegionAdapter.isHostRegionFallbackPending = false
		return HostRegionWorkerDeathResult{
			HasReassigned:   true,
			IsRepairPending: true,
			GetRemountEpoch: getRecoveryResult.GetRemountEpoch,
			GetVersionFloor: getVersionFloor,
		}, nil
	}
	if parseRecoveryErr != nil && !getRecoveryResult.HasFallbackEntered {
		return HostRegionWorkerDeathResult{}, parseRecoveryErr
	}
	if parseFallbackErr := parseHostRegionAdapter.storeCoordinator.handleCoordinatorFallbackRegionTrusted(parseHostRegionAdapter.storeRegionInstanceID); parseFallbackErr != nil {
		return HostRegionWorkerDeathResult{}, parseFallbackErr
	}
	parseHostRegionAdapter.storeScheduler.HandleSchedulerFallback(string(parseHostRegionAdapter.storeRegionInstanceID))
	parseHostRegionAdapter.storeHostRegionRepairVersionFloor = parseHostRegionMaxVersion(
		parseHostRegionAdapter.storeHostRegionRepairVersionFloor,
		getVersionFloor,
	)
	parseHostRegionAdapter.storeHostRegionFallbackReason = string(WorkerDeathFailureKindNoReassign)
	if getFallbackState, hasFallbackState := parseHostRegionAdapter.storeRecoveryCoordinator.GetRegionFallbackState(string(parseHostRegionAdapter.storeRegionInstanceID)); hasFallbackState {
		parseHostRegionAdapter.storeHostRegionFallbackReason = getFallbackState.GetReason
	}
	parseHostRegionAdapter.isHostRegionFallbackActive = true
	parseHostRegionAdapter.isHostRegionFallbackPending = false
	parseHostRegionAdapter.isHostRegionRepairPending = false
	return HostRegionWorkerDeathResult{
		HasFallbackEntered: true,
		GetVersionFloor:    getVersionFloor,
	}, parseRecoveryErr
}

// HandleHostRegionRepairRemount applies one healthy remount handshake and clears fallback ownership.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionRepairRemount(parseSpec ParallelRegionSpec) (HostRegionRepairRemountResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionRepairRemountResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if !parseHostRegionAdapter.isHostRegionRepairPending {
		return HostRegionRepairRemountResult{}, fmt.Errorf("runtime2: host region %q has no pending repair remount", parseHostRegionAdapter.storeRegionInstanceID)
	}
	getSpec, parseSpecErr := NormalizeParallelRegionSpec(parseSpec)
	if parseSpecErr != nil {
		return HostRegionRepairRemountResult{}, parseSpecErr
	}
	if getSpec.RegionInstanceID != parseHostRegionAdapter.storeRegionInstanceID {
		return HostRegionRepairRemountResult{}, fmt.Errorf(
			"runtime2: host region adapter mounted for %q cannot remount region %q",
			parseHostRegionAdapter.storeRegionInstanceID,
			getSpec.RegionInstanceID,
		)
	}
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(getSpec.RegionInstanceID)
	if !hasCoordinatorEntry {
		return HostRegionRepairRemountResult{}, fmt.Errorf("runtime2: host region %q is not mounted", getSpec.RegionInstanceID)
	}
	getRemountEpoch := parseHostRegionMaxVersion(
		parseHostRegionAdapter.storeHostRegionRepairRemountEpoch,
		getCoordinatorEntry.Epoch,
	)
	parseHostRegionAdapter.storeScheduler.HandleSchedulerDispose(string(getSpec.RegionInstanceID))
	getSchedulerJob, parseSchedulerErr := parseHostRegionAdapter.storeScheduler.HandleSchedulerMount(string(getSpec.RegionInstanceID))
	if parseSchedulerErr != nil {
		return HostRegionRepairRemountResult{}, parseSchedulerErr
	}
	getCoordinatorEntry.Epoch = getRemountEpoch
	getCoordinatorEntry.RendererID = getSpec.RendererID
	getCoordinatorEntry.SourceIDs = getSpec.SourceIDs
	getCoordinatorEntry.AssignedWorkerShard = string(getSchedulerJob.GetSchedulerShardID)
	getCoordinatorEntry.LastDispatchedVersion = parseHostRegionMaxVersion(
		getCoordinatorEntry.LastDispatchedVersion,
		parseHostRegionAdapter.storeHostRegionRepairVersionFloor,
	)
	getCoordinatorEntry.LastCommittedVersion = parseHostRegionMaxVersion(
		getCoordinatorEntry.LastCommittedVersion,
		parseHostRegionAdapter.storeHostRegionRepairVersionFloor,
	)
	getCoordinatorEntry.IsAttached = false
	getCoordinatorEntry.IsFallback = false
	getCoordinatorEntry.CurrentState = CoordinatorStateMounted
	if parseStoreErr := parseHostRegionAdapter.storeCoordinator.storeMutableEntry(getCoordinatorEntry); parseStoreErr != nil {
		return HostRegionRepairRemountResult{}, parseStoreErr
	}
	if _, parseCounterErr := parseHostRegionAdapter.storeCoordinator.handleCoordinatorIncrementRegionRepairRemountCountTrusted(getSpec.RegionInstanceID); parseCounterErr != nil {
		return HostRegionRepairRemountResult{}, parseCounterErr
	}
	parseHostRegionAdapter.storeScheduler.ClearSchedulerFallbackOwnership(string(getSpec.RegionInstanceID))
	parseHostRegionAdapter.storeRecoveryCoordinator.ClearRegionLocalFallback(string(getSpec.RegionInstanceID))
	parseHostRegionAdapter.isHostRegionRepairPending = false
	parseHostRegionAdapter.isHostRegionFallbackActive = false
	parseHostRegionAdapter.isHostRegionFallbackPending = false
	parseHostRegionAdapter.storeHostRegionRepairRemountEpoch = 0
	parseHostRegionAdapter.storeHostRegionLastPatchVersion = 0
	parseHostRegionAdapter.storeHostRegionTransportTier = TransportTierStructuredClone
	parseHostRegionAdapter.storeHostRegionSnapshotTier = TransportTierStructuredClone
	parseHostRegionAdapter.storeHostRegionPatchTier = TransportTierStructuredClone
	parseHostRegionAdapter.storeHostRegionFallbackReason = ""
	parseHostRegionAdapter.storeHostRegionSnapshotDowngrade = DiagnosticDowngradeReason{}
	parseHostRegionAdapter.storeHostRegionPatchDowngrade = DiagnosticDowngradeReason{}
	parseHostRegionAdapter.storeHostRegionDispatchAt = time.Time{}
	parseHostRegionAdapter.storeHostRegionPatchReadyAt = time.Time{}
	parseHostRegionAdapter.storeHostRegionCommitAt = time.Time{}
	parseHostRegionAdapter.storeHostRegionDispatchToPatchNS = 0
	parseHostRegionAdapter.storeHostRegionDispatchToCommitNS = 0
	parseHostRegionAdapter.storeHostRegionPatchToCommitNS = 0
	parseHostRegionAdapter.storeHostRegionPatchIdempotency = BuildPatchIdempotencyTracker()
	parseHostRegionAdapter.hasHostRegionSnapshotDowngrade = false
	parseHostRegionAdapter.hasHostRegionPatchDowngrade = false
	parseHostRegionAdapter.storeHostRegionSnapshotFingerprint = ""
	parseHostRegionAdapter.storeHostRegionSnapshotHash = [sha256.Size]byte{}
	parseHostRegionAdapter.hasHostRegionSnapshotHash = false
	parseHostRegionAdapter.storeHostRegionSnapshotFastHash = 0
	parseHostRegionAdapter.hasHostRegionSnapshotFastHash = false
	parseHostRegionAdapter.storeHostRegionDispatchHash = [sha256.Size]byte{}
	parseHostRegionAdapter.hasHostRegionDispatchHash = false
	parseHostRegionAdapter.storeHostRegionDispatchBytes = nil
	parseHostRegionAdapter.hasHostRegionDispatchBytes = false
	parseHostRegionAdapter.storeHostRegionDispatchFastHash = 0
	parseHostRegionAdapter.hasHostRegionDispatchFastHash = false
	parseHostRegionAdapter.storeHostRegionDispatchRendererID = ""
	parseHostRegionAdapter.storeHostRegionDispatchEpoch = 0
	parseHostRegionAdapter.storeHostRegionDispatchInputVersion = 0
	parseHostRegionAdapter.storeHostRegionDispatchSourceVersion = 0
	parseHostRegionAdapter.storeHostRegionDispatchSourceVersionTuple = nil
	parseHostRegionAdapter.storeHostRegionDispatchSourceVersionScratch = nil
	parseHostRegionAdapter.clearHostRegionDispatchPropsLayout()
	parseHostRegionAdapter.hasHostRegionDispatchSourceVersionTuple = false
	parseHostRegionAdapter.hasHostRegionDispatchVersionVector = false
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheState(
		getCoordinatorEntry.Epoch,
		getCoordinatorEntry.RendererID,
		getCoordinatorEntry.SourceIDs,
		false,
		getCoordinatorEntry.LastSnapshotVersion,
		getCoordinatorEntry.LastDispatchedVersion,
	)
	parseHostRegionAdapter.clearHostRegionSourceSnapshotCache()
	parseHostRegionAdapter.storeHostRegionDeferredDispatch = hostRegionDeferredDispatch{}
	parseHostRegionAdapter.hasHostRegionDeferredDispatch = false
	parseHostRegionAdapter.hasHostRegionPostHydrationAttached = false
	parseHostRegionAdapter.hasHostRegionHydratedShellAnchor = false
	return HostRegionRepairRemountResult{
		HasRemounted:       true,
		HasFallbackCleared: true,
		GetRemountEpoch:    getCoordinatorEntry.Epoch,
		GetVersionFloor:    parseHostRegionAdapter.storeHostRegionRepairVersionFloor,
	}, nil
}

// GetHostRegionIsFallbackPending reports whether fallback ownership handoff has started but not yet mirrored.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionIsFallbackPending() bool {
	if parseHostRegionAdapter == nil {
		return false
	}
	return parseHostRegionAdapter.isHostRegionFallbackPending
}

// GetHostRegionIsFallbackActive reports whether fallback ownership is currently active.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionIsFallbackActive() bool {
	if parseHostRegionAdapter == nil {
		return false
	}
	return parseHostRegionAdapter.isHostRegionFallbackActive
}

// GetHostRegionIsRepairPending reports whether repair remount is currently pending.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionIsRepairPending() bool {
	if parseHostRegionAdapter == nil {
		return false
	}
	return parseHostRegionAdapter.isHostRegionRepairPending
}

// IsHostRegionLocalShellOwnership reports whether the adapter currently marks the region shell as locally owned.
func (parseHostRegionAdapter *HostRegionAdapter) IsHostRegionLocalShellOwnership() bool {
	if parseHostRegionAdapter == nil {
		return false
	}
	return parseHostRegionAdapter.isHostRegionLocalShellOwned
}

// HandleHostRegionHydrationComplete marks one mounted host region as hydration-complete.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionHydrationComplete() error {
	if parseHostRegionAdapter == nil {
		return fmt.Errorf("runtime2: host region adapter is nil")
	}
	if _, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID); !hasCoordinatorEntry {
		return fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	parseHostRegionAdapter.isHostRegionHydrationComplete = true
	return nil
}

// HandleHostRegionPostHydrationAttach enables worker attach only after hydration completes for one mounted region.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionPostHydrationAttach() (HostRegionHydrationAttachResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionHydrationAttachResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if _, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID); !hasCoordinatorEntry {
		return HostRegionHydrationAttachResult{}, fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	if !parseHostRegionAdapter.isHostRegionHydrationComplete {
		return HostRegionHydrationAttachResult{
			HasBlocked: true,
		}, fmt.Errorf("runtime2: post-hydration attach is blocked before hydration completes")
	}
	if !parseHostRegionAdapter.hasHostRegionHydratedShellAnchor {
		return HostRegionHydrationAttachResult{
			HasBlocked: true,
		}, fmt.Errorf("runtime2: post-hydration attach requires hydrated shell anchor registration")
	}
	parseHostRegionAdapter.hasHostRegionPostHydrationAttached = true
	if parseAttachErr := parseHostRegionAdapter.storeCoordinator.handleCoordinatorSetRegionAttachedTrusted(parseHostRegionAdapter.storeRegionInstanceID, true); parseAttachErr != nil {
		return HostRegionHydrationAttachResult{}, parseAttachErr
	}
	return HostRegionHydrationAttachResult{
		HasAttached: true,
	}, nil
}

// HandleHostRegionPostRenderAttach enables worker attach after client render without hydration-complete requirements.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionPostRenderAttach() error {
	if parseHostRegionAdapter == nil {
		return fmt.Errorf("runtime2: host region adapter is nil")
	}
	if _, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID); !hasCoordinatorEntry {
		return fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	return parseHostRegionAdapter.storeCoordinator.handleCoordinatorSetRegionAttachedTrusted(parseHostRegionAdapter.storeRegionInstanceID, true)
}

// GetHostRegionIsHydrationComplete reports whether hydration has completed for the region.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionIsHydrationComplete() bool {
	if parseHostRegionAdapter == nil {
		return false
	}
	return parseHostRegionAdapter.isHostRegionHydrationComplete
}

// HasHostRegionPostHydrationAttached reports whether post-hydration worker attach has been enabled.
func (parseHostRegionAdapter *HostRegionAdapter) HasHostRegionPostHydrationAttached() bool {
	if parseHostRegionAdapter == nil {
		return false
	}
	return parseHostRegionAdapter.hasHostRegionPostHydrationAttached
}

// GetHostRegionRepairRemountEpoch reports the pending repair remount epoch floor.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionRepairRemountEpoch() uint64 {
	if parseHostRegionAdapter == nil {
		return 0
	}
	return parseHostRegionAdapter.storeHostRegionRepairRemountEpoch
}

// GetHostRegionRepairVersionFloor reports the pending repair version floor.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionRepairVersionFloor() uint64 {
	if parseHostRegionAdapter == nil {
		return 0
	}
	return parseHostRegionAdapter.storeHostRegionRepairVersionFloor
}

// GetHostRegionLatestValidVersion reports the highest valid host-side input version observed.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionLatestValidVersion() uint64 {
	if parseHostRegionAdapter == nil {
		return 0
	}
	return parseHostRegionAdapter.storeHostRegionLatestValidVersion
}

// GetHostRegionRuntimeStatus reports one host region runtime status snapshot for observability.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionRuntimeStatus() (HostRegionRuntimeStatus, bool) {
	if parseHostRegionAdapter == nil {
		return HostRegionRuntimeStatus{}, false
	}
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return HostRegionRuntimeStatus{}, false
	}
	getRegionMode := HostRegionRuntimeModeLocalShell
	getRegionID := string(parseHostRegionAdapter.storeRegionInstanceID)
	if parseHostRegionAdapter.isHostRegionFallbackPending ||
		parseHostRegionAdapter.isHostRegionFallbackActive ||
		parseHostRegionAdapter.isHostRegionRepairPending ||
		getCoordinatorEntry.IsFallback ||
		parseHostRegionAdapter.storeRecoveryCoordinator.IsRegionLocalOwnership(getRegionID) {
		getRegionMode = HostRegionRuntimeModeFallback
	} else if getCoordinatorEntry.IsAttached || parseHostRegionAdapter.hasHostRegionPostHydrationAttached {
		getRegionMode = HostRegionRuntimeModeWorkerAttached
	}
	getTransportTier := parseHostRegionAdapter.storeHostRegionTransportTier
	if getTransportTier == "" {
		getTransportTier = TransportTierStructuredClone
	}
	getDowngradeStatus := parseHostRegionAdapter.GetHostRegionTransportDowngradeStatus()
	getFallbackReason := parseHostRegionAdapter.storeHostRegionFallbackReason
	if getFallbackState, hasFallbackState := parseHostRegionAdapter.storeRecoveryCoordinator.GetRegionFallbackState(getRegionID); hasFallbackState {
		getFallbackReason = getFallbackState.GetReason
	}
	if getRegionMode != HostRegionRuntimeModeFallback {
		getFallbackReason = ""
	}
	return HostRegionRuntimeStatus{
		GetRegionInstanceID:            parseHostRegionAdapter.storeRegionInstanceID,
		GetRegionMode:                  getRegionMode,
		GetAssignedWorkerShard:         getCoordinatorEntry.AssignedWorkerShard,
		GetRendererID:                  getCoordinatorEntry.RendererID,
		GetEpoch:                       getCoordinatorEntry.Epoch,
		GetIsHydrationComplete:         parseHostRegionAdapter.isHostRegionHydrationComplete,
		HasHydratedShellAnchor:         parseHostRegionAdapter.hasHostRegionHydratedShellAnchor,
		HasPostHydrationAttached:       parseHostRegionAdapter.hasHostRegionPostHydrationAttached,
		GetLastSnapshotVersion:         getCoordinatorEntry.LastSnapshotVersion,
		GetLastDispatchedVersion:       getCoordinatorEntry.LastDispatchedVersion,
		GetLastCommittedVersion:        getCoordinatorEntry.LastCommittedVersion,
		GetTransportTier:               getTransportTier,
		HasSnapshotDowngrade:           getDowngradeStatus.HasSnapshotDowngrade,
		GetSnapshotDowngradePath:       getDowngradeStatus.GetSnapshotDowngrade.Path,
		GetSnapshotDowngradeReason:     getDowngradeStatus.GetSnapshotDowngrade.Reason,
		HasPatchDowngrade:              getDowngradeStatus.HasPatchDowngrade,
		GetPatchDowngradePath:          getDowngradeStatus.GetPatchDowngrade.Path,
		GetPatchDowngradeReason:        getDowngradeStatus.GetPatchDowngrade.Reason,
		GetDroppedStalePatchCount:      getCoordinatorEntry.DroppedStalePatchCount,
		GetIgnoredStaleDiagnosticCount: getCoordinatorEntry.IgnoredStaleDiagnosticCount,
		GetFallbackReason:              getFallbackReason,
	}, true
}

// GetHostRegionRoundTripTiming reports dispatch-to-patch-ready and dispatch-to-commit timing spans.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionRoundTripTiming() HostRegionRoundTripTiming {
	if parseHostRegionAdapter == nil || !parseHostRegionAdapter.isHostRegionRoundTripTimingEnabled {
		return HostRegionRoundTripTiming{}
	}
	return HostRegionRoundTripTiming{
		GetDispatchToPatchReadyNS: parseHostRegionAdapter.storeHostRegionDispatchToPatchNS,
		GetDispatchToCommitNS:     parseHostRegionAdapter.storeHostRegionDispatchToCommitNS,
		GetPatchReadyToCommitNS:   parseHostRegionAdapter.storeHostRegionPatchToCommitNS,
	}
}

// GetHostRegionTransportDowngradeStatus reports separate snapshot and patch transport tiers plus latest downgrade metadata.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionTransportDowngradeStatus() HostRegionTransportDowngradeStatus {
	if parseHostRegionAdapter == nil {
		return HostRegionTransportDowngradeStatus{}
	}
	getSnapshotTier := parseHostRegionAdapter.storeHostRegionSnapshotTier
	if getSnapshotTier == "" {
		getSnapshotTier = TransportTierStructuredClone
	}
	getPatchTier := parseHostRegionAdapter.storeHostRegionPatchTier
	if getPatchTier == "" {
		getPatchTier = TransportTierStructuredClone
	}
	return HostRegionTransportDowngradeStatus{
		GetSnapshotTransportTier: getSnapshotTier,
		HasSnapshotDowngrade:     parseHostRegionAdapter.hasHostRegionSnapshotDowngrade,
		GetSnapshotDowngrade:     parseHostRegionAdapter.storeHostRegionSnapshotDowngrade,
		GetPatchTransportTier:    getPatchTier,
		HasPatchDowngrade:        parseHostRegionAdapter.hasHostRegionPatchDowngrade,
		GetPatchDowngrade:        parseHostRegionAdapter.storeHostRegionPatchDowngrade,
	}
}

// GetHostRegionDiagnosticRing reports the durable diagnostic envelope ring for the mounted region.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionDiagnosticRing() []ControlEnvelope {
	if parseHostRegionAdapter == nil || len(parseHostRegionAdapter.storeHostRegionDiagnosticRing) == 0 {
		return nil
	}
	getDiagnosticRing := make([]ControlEnvelope, len(parseHostRegionAdapter.storeHostRegionDiagnosticRing))
	for parseDiagnosticIndex := range parseHostRegionAdapter.storeHostRegionDiagnosticRing {
		getDiagnosticRing[parseDiagnosticIndex] = copyHostRegionDiagnosticEnvelope(parseHostRegionAdapter.storeHostRegionDiagnosticRing[parseDiagnosticIndex])
	}
	return getDiagnosticRing
}

// GetHostRegionDiagnosticsSnapshot reports one read-only diagnostics snapshot for the mounted region.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionDiagnosticsSnapshot() (HostRegionDiagnosticsSnapshot, bool) {
	if parseHostRegionAdapter == nil {
		return HostRegionDiagnosticsSnapshot{}, false
	}
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return HostRegionDiagnosticsSnapshot{}, false
	}
	getDiagnosticEvents := parseHostRegionAdapter.GetHostRegionDiagnosticRing()
	for parseDiagnosticIndex := range getDiagnosticEvents {
		getDiagnosticEvents[parseDiagnosticIndex] = RedactControlDiagnosticEnvelope(getDiagnosticEvents[parseDiagnosticIndex])
	}
	return HostRegionDiagnosticsSnapshot{
		GetRegionInstanceID:            parseHostRegionAdapter.storeRegionInstanceID,
		GetDiagnosticEvents:            getDiagnosticEvents,
		GetTransportDowngradeStatus:    parseHostRegionAdapter.GetHostRegionTransportDowngradeStatus(),
		GetDroppedStalePatchCount:      getCoordinatorEntry.DroppedStalePatchCount,
		GetIgnoredStaleDiagnosticCount: getCoordinatorEntry.IgnoredStaleDiagnosticCount,
		GetRepairTriggeredRemountCount: getCoordinatorEntry.RepairRemountCount,
	}, true
}

// copyHostRegionDiagnosticEnvelope clones one diagnostic envelope so read-only getters do not expose mutable pointer payloads.
func copyHostRegionDiagnosticEnvelope(parseEnvelope ControlEnvelope) ControlEnvelope {
	getEnvelope := parseEnvelope
	if parseEnvelope.DiagnosticTiming != nil {
		getTiming := *parseEnvelope.DiagnosticTiming
		getEnvelope.DiagnosticTiming = &getTiming
	}
	if parseEnvelope.DiagnosticSize != nil {
		getSize := *parseEnvelope.DiagnosticSize
		getEnvelope.DiagnosticSize = &getSize
	}
	if parseEnvelope.DiagnosticFallback != nil {
		getFallback := *parseEnvelope.DiagnosticFallback
		getEnvelope.DiagnosticFallback = &getFallback
	}
	if parseEnvelope.DiagnosticTrace != nil {
		getTrace := *parseEnvelope.DiagnosticTrace
		getEnvelope.DiagnosticTrace = &getTrace
	}
	if parseEnvelope.DiagnosticDowngrade != nil {
		getDowngrade := *parseEnvelope.DiagnosticDowngrade
		getEnvelope.DiagnosticDowngrade = &getDowngrade
	}
	return getEnvelope
}

// HandleHostRegionRegisterHydratedShellAnchor registers one hydrated shell anchor into the region DOM index.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionRegisterHydratedShellAnchor(parseNodeID uint64, parseTag string) error {
	if parseHostRegionAdapter == nil {
		return fmt.Errorf("runtime2: host region adapter is nil")
	}
	if _, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID); !hasCoordinatorEntry {
		return fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	if parseNodeID == 0 {
		return fmt.Errorf("runtime2: hydrated shell anchor node ID is required")
	}
	if parseTag == "" {
		return fmt.Errorf("runtime2: hydrated shell anchor tag is required")
	}
	if parseSetErr := parseHostRegionAdapter.storeRegionDOMIndexHandle.SetRegionDOMNode(
		string(parseHostRegionAdapter.storeRegionInstanceID),
		parseNodeID,
		&RegionDOMNode{
			GetNodeID: parseNodeID,
			GetTag:    parseTag,
		},
	); parseSetErr != nil {
		return parseSetErr
	}
	parseHostRegionAdapter.hasHostRegionHydratedShellAnchor = true
	return nil
}

// HasHostRegionHydratedShellAnchor reports whether one hydrated shell anchor has been registered.
func (parseHostRegionAdapter *HostRegionAdapter) HasHostRegionHydratedShellAnchor() bool {
	if parseHostRegionAdapter == nil {
		return false
	}
	return parseHostRegionAdapter.hasHostRegionHydratedShellAnchor
}

// HandleHostRegionShellIdentityMismatchDetection detects region-ID and renderer-ID mismatches for one hydrated shell marker.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionShellIdentityMismatchDetection(parseMarker SSRShellMarker) (HostRegionShellIdentityMismatchResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionShellIdentityMismatchResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseMarkerErr := ValidateSSRShellMarker(parseMarker); parseMarkerErr != nil {
		return HostRegionShellIdentityMismatchResult{}, parseMarkerErr
	}
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return HostRegionShellIdentityMismatchResult{}, fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	hasRegionIDMismatch := parseMarker.RegionInstanceID != parseHostRegionAdapter.storeRegionInstanceID
	hasRendererIDMismatch := parseMarker.RendererID != getCoordinatorEntry.RendererID
	return HostRegionShellIdentityMismatchResult{
		HasMismatch:           hasRegionIDMismatch || hasRendererIDMismatch,
		HasRegionIDMismatch:   hasRegionIDMismatch,
		HasRendererIDMismatch: hasRendererIDMismatch,
	}, nil
}

// HandleHostRegionShellMissingAnchorDetection reports whether one mounted region is missing hydrated shell anchor registration.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionShellMissingAnchorDetection() (HostRegionShellAnchorCheckResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionShellAnchorCheckResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if _, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID); !hasCoordinatorEntry {
		return HostRegionShellAnchorCheckResult{}, fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	return HostRegionShellAnchorCheckResult{
		HasMissingAnchor: !parseHostRegionAdapter.hasHostRegionHydratedShellAnchor,
	}, nil
}

// HandleHostRegionShellMismatchFallback enters local fallback ownership after hydration shell identity or anchor mismatch.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionShellMismatchFallback(parseInputVersion uint64) (HostRegionShellMismatchFallbackResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionShellMismatchFallbackResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return HostRegionShellMismatchFallbackResult{}, fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	getVersionFloor := parseHostRegionMaxVersion(
		parseInputVersion,
		getCoordinatorEntry.LastCommittedVersion,
		getCoordinatorEntry.LastDispatchedVersion,
		parseHostRegionAdapter.storeHostRegionLatestValidVersion,
		parseHostRegionAdapter.storeHostRegionRepairVersionFloor,
	)
	if getVersionFloor == 0 {
		getVersionFloor = 1
	}
	parseHostRegionAdapter.storeHostRegionRepairVersionFloor = parseHostRegionMaxVersion(
		parseHostRegionAdapter.storeHostRegionRepairVersionFloor,
		getVersionFloor,
	)
	parseHostRegionAdapter.storeHostRegionRepairRemountEpoch = parseHostRegionMaxVersion(
		parseHostRegionAdapter.storeHostRegionRepairRemountEpoch,
		getCoordinatorEntry.Epoch+1,
	)
	parseHostRegionAdapter.storeHostRegionFallbackReason = "hydration-shell-mismatch"
	parseHostRegionAdapter.storeRecoveryCoordinator.EnterRegionLocalFallback(
		string(parseHostRegionAdapter.storeRegionInstanceID),
		"hydration-shell-mismatch",
		getCoordinatorEntry.Epoch,
		getVersionFloor,
	)
	if parseFallbackErr := parseHostRegionAdapter.storeCoordinator.handleCoordinatorFallbackRegionTrusted(parseHostRegionAdapter.storeRegionInstanceID); parseFallbackErr != nil {
		return HostRegionShellMismatchFallbackResult{}, parseFallbackErr
	}
	parseHostRegionAdapter.storeScheduler.HandleSchedulerFallback(string(parseHostRegionAdapter.storeRegionInstanceID))
	parseHostRegionAdapter.isHostRegionFallbackPending = false
	parseHostRegionAdapter.isHostRegionFallbackActive = true
	parseHostRegionAdapter.isHostRegionRepairPending = true
	parseHostRegionAdapter.hasHostRegionPostHydrationAttached = false
	return HostRegionShellMismatchFallbackResult{
		HasFallbackEntered:  true,
		HasOutputSuppressed: true,
		GetRemountEpoch:     parseHostRegionAdapter.storeHostRegionRepairRemountEpoch,
		GetVersionFloor:     parseHostRegionAdapter.storeHostRegionRepairVersionFloor,
	}, nil
}
