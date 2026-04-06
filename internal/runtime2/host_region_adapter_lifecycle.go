package runtime2

import (
	"crypto/sha256"
	"fmt"
	"time"
)

func BuildHostRegionAdapter(parseRegionInstanceID RegionInstanceID, parseSchedulerShardIDs []SchedulerShardID) (*HostRegionAdapter, error) {
	getRegionInstanceID, parseRegionInstanceIDErr := ParseRegionInstanceID(string(parseRegionInstanceID))
	if parseRegionInstanceIDErr != nil {
		return nil, parseRegionInstanceIDErr
	}
	getSchedulerShardIDs, parseSchedulerShardIDsErr := parseSchedulerShardList(parseSchedulerShardIDs)
	if parseSchedulerShardIDsErr != nil {
		return nil, fmt.Errorf("runtime2: host region adapter requires valid scheduler shard IDs: %w", parseSchedulerShardIDsErr)
	}
	return &HostRegionAdapter{
		storeRegionInstanceID:           getRegionInstanceID,
		storeCoordinator:                BuildCoordinator(),
		storeScheduler:                  BuildScheduler(getSchedulerShardIDs),
		storeRecoveryCoordinator:        BuildRecoveryCoordinator(),
		storeRegionDOMIndexHandle:       BuildRegionDOMIndex(),
		storeHostRegionTransportTier:    TransportTierStructuredClone,
		storeHostRegionSnapshotTier:     TransportTierStructuredClone,
		storeHostRegionPatchTier:        TransportTierStructuredClone,
		storeHostRegionPatchIdempotency: BuildPatchIdempotencyTracker(),
	}, nil
}

// GetHostRegionInstanceID reports the adapter-owned region instance identity.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionInstanceID() RegionInstanceID {
	if parseHostRegionAdapter == nil {
		return ""
	}
	return parseHostRegionAdapter.storeRegionInstanceID
}

// GetHostRegionCoordinator reports the adapter-owned coordinator handle.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionCoordinator() *Coordinator {
	if parseHostRegionAdapter == nil {
		return nil
	}
	return parseHostRegionAdapter.storeCoordinator
}

// GetHostRegionScheduler reports the adapter-owned scheduler handle.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionScheduler() *Scheduler {
	if parseHostRegionAdapter == nil {
		return nil
	}
	return parseHostRegionAdapter.storeScheduler
}

// GetHostRegionRecoveryCoordinator reports the adapter-owned recovery coordinator handle.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionRecoveryCoordinator() *RecoveryCoordinator {
	if parseHostRegionAdapter == nil {
		return nil
	}
	return parseHostRegionAdapter.storeRecoveryCoordinator
}

// GetHostRegionDOMIndex reports the adapter-owned region DOM index handle.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionDOMIndex() *RegionDOMIndex {
	if parseHostRegionAdapter == nil {
		return nil
	}
	return parseHostRegionAdapter.storeRegionDOMIndexHandle
}

// SetHostRegionRoundTripTimingEnabled enables or disables dispatch-to-patch-ready and commit timing capture for one host region adapter.
func (parseHostRegionAdapter *HostRegionAdapter) SetHostRegionRoundTripTimingEnabled(parseIsEnabled bool) {
	if parseHostRegionAdapter == nil {
		return
	}
	parseHostRegionAdapter.isHostRegionRoundTripTimingEnabled = parseIsEnabled
	if parseIsEnabled {
		return
	}
	parseHostRegionAdapter.storeHostRegionDispatchAt = time.Time{}
	parseHostRegionAdapter.storeHostRegionPatchReadyAt = time.Time{}
	parseHostRegionAdapter.storeHostRegionCommitAt = time.Time{}
	parseHostRegionAdapter.storeHostRegionDispatchToPatchNS = 0
	parseHostRegionAdapter.storeHostRegionDispatchToCommitNS = 0
	parseHostRegionAdapter.storeHostRegionPatchToCommitNS = 0
}

// HandleHostRegionMount validates one parallel region spec and mounts scheduler plus coordinator ownership for the adapter region.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionMount(parseSpec ParallelRegionSpec, parseEpoch uint64) (HostRegionMountResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionMountResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	getSpec, parseSpecErr := NormalizeParallelRegionSpec(parseSpec)
	if parseSpecErr != nil {
		return HostRegionMountResult{}, parseSpecErr
	}
	if parseEpoch == 0 {
		return HostRegionMountResult{}, fmt.Errorf("runtime2: host mount epoch is required")
	}
	if getSpec.RegionInstanceID != parseHostRegionAdapter.storeRegionInstanceID {
		return HostRegionMountResult{}, fmt.Errorf(
			"runtime2: host region adapter mounted for %q cannot mount region %q",
			parseHostRegionAdapter.storeRegionInstanceID,
			getSpec.RegionInstanceID,
		)
	}
	if _, hasHostRegionEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(getSpec.RegionInstanceID); hasHostRegionEntry {
		return HostRegionMountResult{}, fmt.Errorf("runtime2: host region %q is already mounted", getSpec.RegionInstanceID)
	}
	getSchedulerJob, parseSchedulerErr := parseHostRegionAdapter.storeScheduler.HandleSchedulerMount(string(getSpec.RegionInstanceID))
	if parseSchedulerErr != nil {
		return HostRegionMountResult{}, parseSchedulerErr
	}
	parseMountErr := parseHostRegionAdapter.storeCoordinator.MountRegion(CoordinatorEntry{
		RegionInstanceID:    getSpec.RegionInstanceID,
		RendererID:          getSpec.RendererID,
		SourceIDs:           getSpec.SourceIDs,
		Epoch:               parseEpoch,
		AssignedWorkerShard: string(getSchedulerJob.GetSchedulerShardID),
	})
	if parseMountErr != nil {
		parseHostRegionAdapter.storeScheduler.HandleSchedulerDispose(string(getSpec.RegionInstanceID))
		return HostRegionMountResult{}, parseMountErr
	}
	parseHostRegionAdapter.isHostRegionLocalShellOwned = true
	parseHostRegionAdapter.isHostRegionRemoved = false
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
	parseHostRegionAdapter.storeHostRegionDiagnosticRing = nil
	parseHostRegionAdapter.hasHostRegionSnapshotDowngrade = false
	parseHostRegionAdapter.hasHostRegionPatchDowngrade = false
	parseHostRegionAdapter.storeHostRegionSnapshotFingerprint = ""
	parseHostRegionAdapter.storeHostRegionSnapshotHash = [sha256.Size]byte{}
	parseHostRegionAdapter.hasHostRegionSnapshotHash = false
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
		parseEpoch,
		getSpec.RendererID,
		getSpec.SourceIDs,
		false,
		0,
		0,
	)
	parseHostRegionAdapter.clearHostRegionSourceSnapshotCache()
	parseHostRegionAdapter.storeHostRegionDeferredDispatch = hostRegionDeferredDispatch{}
	parseHostRegionAdapter.hasHostRegionDeferredDispatch = false
	getCoordinatorEntry, _ := parseHostRegionAdapter.storeCoordinator.GetEntry(getSpec.RegionInstanceID)
	return HostRegionMountResult{
		GetSchedulerJob:     getSchedulerJob,
		GetCoordinatorEntry: getCoordinatorEntry,
	}, nil
}

// HandleHostRegionUpdate dispatches one mounted region update through scheduler and coordinator version tracking.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionUpdate(parseInputVersion uint64) (HostRegionUpdateResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionUpdateResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseInputVersion == 0 {
		return HostRegionUpdateResult{}, fmt.Errorf("runtime2: host update input version is required")
	}
	if parseHostRegionAdapter.isHostRegionRepairPending {
		return HostRegionUpdateResult{}, fmt.Errorf("runtime2: host region %q repair remount is pending", parseHostRegionAdapter.storeRegionInstanceID)
	}
	if parseInputVersion < parseHostRegionAdapter.storeHostRegionRepairVersionFloor {
		return HostRegionUpdateResult{}, fmt.Errorf(
			"runtime2: host update input version %d is below repair floor %d",
			parseInputVersion,
			parseHostRegionAdapter.storeHostRegionRepairVersionFloor,
		)
	}
	getEntryIsFallback,
		_,
		getEntryLastDispatchedVersion,
		hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntryDispatchValidation(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return HostRegionUpdateResult{}, fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	if getEntryIsFallback {
		return HostRegionUpdateResult{}, fmt.Errorf("runtime2: host region %q is in fallback mode", parseHostRegionAdapter.storeRegionInstanceID)
	}
	if parseVersionErr := ValidateMonotonicInputVersion(getEntryLastDispatchedVersion, parseInputVersion); parseVersionErr != nil {
		return HostRegionUpdateResult{}, parseVersionErr
	}
	getSchedulerJob, parseSchedulerErr := parseHostRegionAdapter.storeScheduler.HandleSchedulerUpdate(string(parseHostRegionAdapter.storeRegionInstanceID))
	if parseSchedulerErr != nil {
		return HostRegionUpdateResult{}, parseSchedulerErr
	}
	if parseCoordinatorUpdateErr := parseHostRegionAdapter.storeCoordinator.applyRegionDispatchedVersionTrusted(
		parseHostRegionAdapter.storeRegionInstanceID,
		parseInputVersion,
	); parseCoordinatorUpdateErr != nil {
		return HostRegionUpdateResult{}, parseCoordinatorUpdateErr
	}
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheDispatchedVersion(parseInputVersion)
	parseHostRegionAdapter.storeHostRegionLatestValidVersion = parseHostRegionMaxVersion(
		parseHostRegionAdapter.storeHostRegionLatestValidVersion,
		parseInputVersion,
	)
	return HostRegionUpdateResult{
		GetSchedulerJob:      getSchedulerJob,
		GetDispatchedVersion: parseInputVersion,
	}, nil
}

// HandleHostRegionDispose disposes one mounted host region from coordinator, scheduler, and DOM-index state.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionDispose() (HostRegionDisposeResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionDisposeResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	getRegionInstanceID := parseHostRegionAdapter.storeRegionInstanceID
	if _, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(getRegionInstanceID); !hasCoordinatorEntry {
		return HostRegionDisposeResult{}, fmt.Errorf("runtime2: host region %q is not mounted", getRegionInstanceID)
	}
	hasSchedulerDisposed := parseHostRegionAdapter.storeScheduler.HandleSchedulerDispose(string(getRegionInstanceID))
	getDOMClearResult := parseHostRegionAdapter.storeRegionDOMIndexHandle.ClearRegionDOMNodes(string(getRegionInstanceID), nil)
	if parseDisposeErr := parseHostRegionAdapter.storeCoordinator.DisposeRegion(getRegionInstanceID); parseDisposeErr != nil {
		return HostRegionDisposeResult{}, parseDisposeErr
	}
	parseHostRegionAdapter.isHostRegionLocalShellOwned = false
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
	parseHostRegionAdapter.storeHostRegionDiagnosticRing = nil
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
	parseHostRegionAdapter.clearHostRegionCoordinatorCacheState()
	parseHostRegionAdapter.clearHostRegionSourceSnapshotCache()
	return HostRegionDisposeResult{
		HasCoordinatorDisposed: true,
		HasSchedulerDisposed:   hasSchedulerDisposed,
		GetClearedDOMNodeCount: getDOMClearResult.GetClearedNodeCount,
	}, nil
}

// HandleHostRegionDiagnosticEnvelope records one validated diagnostic control envelope in durable host-region diagnostic state.
