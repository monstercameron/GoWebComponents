package runtime2

import "fmt"

// HostRegionAdapter owns host-side runtime2 handles for one live region instance.
type HostRegionAdapter struct {
	storeRegionInstanceID              RegionInstanceID
	storeCoordinator                   *Coordinator
	storeScheduler                     *Scheduler
	storeRecoveryCoordinator           *RecoveryCoordinator
	storeRegionDOMIndexHandle          *RegionDOMIndex
	storeHostRegionSourceLookup        HostRegionSourceLookup
	storeHostRegionSnapshotFingerprint string
	storeHostRegionDeferredDispatch    *hostRegionDeferredDispatch
	storeHostRegionRepairRemountEpoch  uint64
	storeHostRegionRepairVersionFloor  uint64
	storeHostRegionLatestValidVersion  uint64
	isHostRegionRemoved                bool
	isHostRegionFallbackPending        bool
	isHostRegionFallbackActive         bool
	isHostRegionRepairPending          bool
	isHostRegionHydrationComplete      bool
	hasHostRegionPostHydrationAttached bool
	isHostRegionLocalShellOwned        bool
}

type hostRegionDeferredDispatch struct {
	getInputVersion        uint64
	getSnapshotEnvelope    SnapshotEnvelope
	getSnapshotFingerprint string
}

// HostRegionMountResult reports host-side mount outputs for one region mount attempt.
type HostRegionMountResult struct {
	GetSchedulerJob     SchedulerJob
	GetCoordinatorEntry CoordinatorEntry
}

// HostRegionUpdateResult reports host-side update outputs for one region update dispatch.
type HostRegionUpdateResult struct {
	GetSchedulerJob      SchedulerJob
	GetDispatchedVersion uint64
}

// HostRegionDisposeResult reports host-side dispose outputs for one mounted region.
type HostRegionDisposeResult struct {
	HasCoordinatorDisposed bool
	HasSchedulerDisposed   bool
	GetClearedDOMNodeCount int
}

// HostRegionSourceLookup resolves declared source values and versions from shipped runtime state.
type HostRegionSourceLookup func(parseSourceIDs []string) (map[string]any, map[string]uint64, error)

// HostRegionSourceSnapshot stores source values and source versions returned by host-side source lookup.
type HostRegionSourceSnapshot struct {
	GetSourceValues   map[string]any
	GetSourceVersions map[string]uint64
}

// HostRegionSnapshotFingerprintResult reports one snapshot fingerprint decision for no-change detection.
type HostRegionSnapshotFingerprintResult struct {
	GetSnapshotFingerprint string
	HasNoChange            bool
}

// HostRegionDispatchPriority identifies runtime2 host update dispatch priority classification.
type HostRegionDispatchPriority string

const (
	// HostRegionDispatchPriorityUrgent marks an update that should dispatch immediately.
	HostRegionDispatchPriorityUrgent HostRegionDispatchPriority = "urgent"
	// HostRegionDispatchPriorityDeferred marks an update that may dispatch later under deferred policy.
	HostRegionDispatchPriorityDeferred HostRegionDispatchPriority = "deferred"
)

// ParseHostRegionDispatchPriority validates one host update dispatch priority value.
func ParseHostRegionDispatchPriority(parsePriority HostRegionDispatchPriority) (HostRegionDispatchPriority, error) {
	switch parsePriority {
	case HostRegionDispatchPriorityUrgent, HostRegionDispatchPriorityDeferred:
		return parsePriority, nil
	default:
		return "", fmt.Errorf("runtime2: host dispatch priority %q is unsupported", parsePriority)
	}
}

func parseHostRegionMaxVersion(parseVersions ...uint64) uint64 {
	var parseMaxVersion uint64
	for _, parseVersion := range parseVersions {
		if parseVersion > parseMaxVersion {
			parseMaxVersion = parseVersion
		}
	}
	return parseMaxVersion
}

// HostRegionUpdateDispatchResult reports one host-side update dispatch decision including short-circuit behavior.
type HostRegionUpdateDispatchResult struct {
	HasScheduled           bool
	HasNoChange            bool
	HasDeferredQueued      bool
	HasDeferredSuperseded  bool
	HasDeferredCanceled    bool
	GetDispatchPriority    HostRegionDispatchPriority
	GetSchedulerJob        SchedulerJob
	GetSnapshotEnvelope    SnapshotEnvelope
	GetSnapshotFingerprint string
}

// HostRegionOwnerInvalidateResult reports cleanup actions applied after owner-side invalidation.
type HostRegionOwnerInvalidateResult struct {
	HasSchedulerCanceled bool
	HasDeferredCleared   bool
}

// HostRegionWorkerOutputResult reports whether one worker output attempt committed or was ignored.
type HostRegionWorkerOutputResult struct {
	HasCommitted bool
	HasIgnored   bool
}

// HostRegionOwnerRemoveResult reports owner-removal handling for one mounted region.
type HostRegionOwnerRemoveResult struct {
	HasDisposed                   bool
	HasLateWorkerOutputSuppressed bool
}

// HostRegionStructuralRemountResult reports renderer or shell-ownership changes and remount decisions.
type HostRegionStructuralRemountResult struct {
	HasRendererChanged       bool
	HasShellOwnershipChanged bool
	HasRemounted             bool
	GetRemountEpoch          uint64
}

// HostRegionFallbackMirrorResult reports fallback mirroring into coordinator and scheduler state.
type HostRegionFallbackMirrorResult struct {
	HasCoordinatorFallback bool
	HasSchedulerFallback   bool
}

// HostRegionPatchReadyResult reports whether one patch-ready result may continue to commit processing.
type HostRegionPatchReadyResult struct {
	HasAccepted     bool
	HasIgnored      bool
	GetIgnoreReason string
}

// HostRegionWorkerDeathResult reports host-side worker death handling outcomes.
type HostRegionWorkerDeathResult struct {
	HasReassigned      bool
	HasFallbackEntered bool
	IsRepairPending    bool
	GetRemountEpoch    uint64
	GetVersionFloor    uint64
}

// HostRegionRepairRemountResult reports repair-driven remount handshake outcomes.
type HostRegionRepairRemountResult struct {
	HasRemounted       bool
	HasFallbackCleared bool
	GetRemountEpoch    uint64
	GetVersionFloor    uint64
}

// HostRegionHydrationAttachResult reports post-hydration worker attach handling outcomes.
type HostRegionHydrationAttachResult struct {
	HasAttached bool
	HasBlocked  bool
}

// BuildHostRegionAdapter creates one host-side region adapter with coordinator, scheduler, recovery, and DOM-index handles.
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
		storeRegionInstanceID:     getRegionInstanceID,
		storeCoordinator:          BuildCoordinator(),
		storeScheduler:            BuildScheduler(getSchedulerShardIDs),
		storeRecoveryCoordinator:  BuildRecoveryCoordinator(),
		storeRegionDOMIndexHandle: BuildRegionDOMIndex(),
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
	parseHostRegionAdapter.storeHostRegionRepairRemountEpoch = 0
	parseHostRegionAdapter.storeHostRegionRepairVersionFloor = 0
	parseHostRegionAdapter.storeHostRegionSnapshotFingerprint = ""
	parseHostRegionAdapter.storeHostRegionDeferredDispatch = nil
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
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return HostRegionUpdateResult{}, fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	if getCoordinatorEntry.IsFallback {
		return HostRegionUpdateResult{}, fmt.Errorf("runtime2: host region %q is in fallback mode", parseHostRegionAdapter.storeRegionInstanceID)
	}
	if parseVersionErr := ValidateMonotonicInputVersion(getCoordinatorEntry.LastDispatchedVersion, parseInputVersion); parseVersionErr != nil {
		return HostRegionUpdateResult{}, parseVersionErr
	}
	getSchedulerJob, parseSchedulerErr := parseHostRegionAdapter.storeScheduler.HandleSchedulerUpdate(string(parseHostRegionAdapter.storeRegionInstanceID))
	if parseSchedulerErr != nil {
		return HostRegionUpdateResult{}, parseSchedulerErr
	}
	if parseUpdateErr := parseHostRegionAdapter.storeCoordinator.UpdateRegion(parseHostRegionAdapter.storeRegionInstanceID, parseInputVersion); parseUpdateErr != nil {
		return HostRegionUpdateResult{}, parseUpdateErr
	}
	getCoordinatorEntry, _ = parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID)
	parseHostRegionAdapter.storeHostRegionLatestValidVersion = parseHostRegionMaxVersion(
		parseHostRegionAdapter.storeHostRegionLatestValidVersion,
		parseInputVersion,
	)
	return HostRegionUpdateResult{
		GetSchedulerJob:      getSchedulerJob,
		GetDispatchedVersion: getCoordinatorEntry.LastDispatchedVersion,
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
	parseHostRegionAdapter.storeHostRegionDeferredDispatch = nil
	parseHostRegionAdapter.isHostRegionFallbackPending = false
	parseHostRegionAdapter.isHostRegionFallbackActive = false
	parseHostRegionAdapter.isHostRegionRepairPending = false
	parseHostRegionAdapter.isHostRegionHydrationComplete = false
	parseHostRegionAdapter.hasHostRegionPostHydrationAttached = false
	parseHostRegionAdapter.storeHostRegionRepairRemountEpoch = 0
	parseHostRegionAdapter.storeHostRegionRepairVersionFloor = 0
	return HostRegionDisposeResult{
		HasCoordinatorDisposed: true,
		HasSchedulerDisposed:   hasSchedulerDisposed,
		GetClearedDOMNodeCount: getDOMClearResult.GetClearedNodeCount,
	}, nil
}

// SetHostRegionSourceLookup sets the host-side bridge used to look up declared source values and versions.
func (parseHostRegionAdapter *HostRegionAdapter) SetHostRegionSourceLookup(parseSourceLookup HostRegionSourceLookup) error {
	if parseHostRegionAdapter == nil {
		return fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseSourceLookup == nil {
		return fmt.Errorf("runtime2: host source lookup is required")
	}
	parseHostRegionAdapter.storeHostRegionSourceLookup = parseSourceLookup
	return nil
}

// HandleHostRegionDeclaredSourceLookup resolves canonical declared source IDs through the configured host source lookup bridge.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionDeclaredSourceLookup(parseSourceIDs []string) (HostRegionSourceSnapshot, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionSourceSnapshot{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	getSourceIDs, parseSourceIDsErr := NormalizeSourceIDs(parseSourceIDs)
	if parseSourceIDsErr != nil {
		return HostRegionSourceSnapshot{}, parseSourceIDsErr
	}
	if len(getSourceIDs) == 0 {
		return HostRegionSourceSnapshot{}, nil
	}
	if parseHostRegionAdapter.storeHostRegionSourceLookup == nil {
		return HostRegionSourceSnapshot{}, fmt.Errorf("runtime2: host source lookup is not configured")
	}
	getSourceValues, getSourceVersions, parseSourceLookupErr := parseHostRegionAdapter.storeHostRegionSourceLookup(getSourceIDs)
	if parseSourceLookupErr != nil {
		return HostRegionSourceSnapshot{}, parseSourceLookupErr
	}
	buildSourceValues, parseSourceValuesErr := BuildSourceSnapshot(getSourceIDs, getSourceValues)
	if parseSourceValuesErr != nil {
		return HostRegionSourceSnapshot{}, parseSourceValuesErr
	}
	buildSourceVersions := make(map[string]uint64, len(getSourceVersions))
	for getSourceID, getSourceVersion := range getSourceVersions {
		buildSourceVersions[getSourceID] = getSourceVersion
	}
	if buildSourceValues == nil {
		buildSourceValues = make(map[string]any)
	}
	for getSourceID, getSourceValue := range buildSourceValues {
		buildSourceValues[getSourceID] = getSourceValue
	}
	return HostRegionSourceSnapshot{
		GetSourceValues:   buildSourceValues,
		GetSourceVersions: buildSourceVersions,
	}, nil
}

// HandleHostRegionUpdateSnapshot captures one update snapshot envelope from normalized props plus declared source lookup.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionUpdateSnapshot(parseSpec ParallelRegionSpec, parseInputVersion uint64) (SnapshotEnvelope, error) {
	if parseHostRegionAdapter == nil {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseInputVersion == 0 {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: host update snapshot input version is required")
	}
	getSpec, parseSpecErr := NormalizeParallelRegionSpec(parseSpec)
	if parseSpecErr != nil {
		return SnapshotEnvelope{}, parseSpecErr
	}
	if getSpec.RegionInstanceID != parseHostRegionAdapter.storeRegionInstanceID {
		return SnapshotEnvelope{}, fmt.Errorf(
			"runtime2: host region adapter mounted for %q cannot capture snapshot for region %q",
			parseHostRegionAdapter.storeRegionInstanceID,
			getSpec.RegionInstanceID,
		)
	}
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(getSpec.RegionInstanceID)
	if !hasCoordinatorEntry {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: host region %q is not mounted", getSpec.RegionInstanceID)
	}
	if getCoordinatorEntry.RendererID != getSpec.RendererID {
		return SnapshotEnvelope{}, fmt.Errorf(
			"runtime2: mounted renderer ID %q does not match snapshot renderer ID %q",
			getCoordinatorEntry.RendererID,
			getSpec.RendererID,
		)
	}
	getSourceSnapshot, parseSourceSnapshotErr := parseHostRegionAdapter.HandleHostRegionDeclaredSourceLookup(getSpec.SourceIDs)
	if parseSourceSnapshotErr != nil {
		return SnapshotEnvelope{}, parseSourceSnapshotErr
	}
	return BuildSnapshotEnvelope(
		getSpec.RegionInstanceID,
		getCoordinatorEntry.Epoch,
		parseInputVersion,
		getSpec.Props,
		getSpec.SourceIDs,
		getSourceSnapshot.GetSourceValues,
		getSourceSnapshot.GetSourceVersions,
	)
}

// HandleHostRegionSnapshotFingerprint computes and stores one stable snapshot fingerprint for host-side no-change detection.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionSnapshotFingerprint(parseSnapshotEnvelope SnapshotEnvelope) (HostRegionSnapshotFingerprintResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionSnapshotFingerprintResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseSnapshotEnvelope.RegionInstanceID != parseHostRegionAdapter.storeRegionInstanceID {
		return HostRegionSnapshotFingerprintResult{}, fmt.Errorf(
			"runtime2: host region adapter mounted for %q cannot fingerprint snapshot for region %q",
			parseHostRegionAdapter.storeRegionInstanceID,
			parseSnapshotEnvelope.RegionInstanceID,
		)
	}
	if parseSnapshotErr := ValidateSnapshotEnvelope(parseSnapshotEnvelope); parseSnapshotErr != nil {
		return HostRegionSnapshotFingerprintResult{}, parseSnapshotErr
	}
	buildFingerprintEnvelope := parseSnapshotEnvelope
	buildFingerprintEnvelope.InputVersion = 1
	getSnapshotFingerprint, parseSnapshotFingerprintErr := GetSnapshotFingerprint(buildFingerprintEnvelope)
	if parseSnapshotFingerprintErr != nil {
		return HostRegionSnapshotFingerprintResult{}, parseSnapshotFingerprintErr
	}
	hasNoChange := getSnapshotFingerprint == parseHostRegionAdapter.storeHostRegionSnapshotFingerprint && getSnapshotFingerprint != ""
	parseHostRegionAdapter.storeHostRegionSnapshotFingerprint = getSnapshotFingerprint
	return HostRegionSnapshotFingerprintResult{
		GetSnapshotFingerprint: getSnapshotFingerprint,
		HasNoChange:            hasNoChange,
	}, nil
}

// HandleHostRegionUpdateDispatch captures one update snapshot, applies no-change short-circuit rules, and schedules worker update dispatch only when needed.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionUpdateDispatch(parseSpec ParallelRegionSpec, parseInputVersion uint64) (HostRegionUpdateDispatchResult, error) {
	return parseHostRegionAdapter.HandleHostRegionUpdateDispatchWithPriority(parseSpec, parseInputVersion, HostRegionDispatchPriorityUrgent)
}

// HandleHostRegionUpdateDispatchWithPriority captures one update snapshot and dispatches using the requested host update priority classification.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionUpdateDispatchWithPriority(
	parseSpec ParallelRegionSpec,
	parseInputVersion uint64,
	parseDispatchPriority HostRegionDispatchPriority,
) (HostRegionUpdateDispatchResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionUpdateDispatchResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	getDispatchPriority, parseDispatchPriorityErr := ParseHostRegionDispatchPriority(parseDispatchPriority)
	if parseDispatchPriorityErr != nil {
		return HostRegionUpdateDispatchResult{}, parseDispatchPriorityErr
	}
	if parseHostRegionAdapter.isHostRegionRepairPending {
		return HostRegionUpdateDispatchResult{}, fmt.Errorf("runtime2: host region %q repair remount is pending", parseHostRegionAdapter.storeRegionInstanceID)
	}
	getSnapshotEnvelope, parseSnapshotErr := parseHostRegionAdapter.HandleHostRegionUpdateSnapshot(parseSpec, parseInputVersion)
	if parseSnapshotErr != nil {
		return HostRegionUpdateDispatchResult{}, parseSnapshotErr
	}
	getFingerprintResult, parseFingerprintErr := parseHostRegionAdapter.HandleHostRegionSnapshotFingerprint(getSnapshotEnvelope)
	if parseFingerprintErr != nil {
		return HostRegionUpdateDispatchResult{}, parseFingerprintErr
	}
	if getFingerprintResult.HasNoChange {
		return HostRegionUpdateDispatchResult{
			HasScheduled:           false,
			HasNoChange:            true,
			GetDispatchPriority:    getDispatchPriority,
			GetSnapshotEnvelope:    getSnapshotEnvelope,
			GetSnapshotFingerprint: getFingerprintResult.GetSnapshotFingerprint,
		}, nil
	}
	if getDispatchPriority == HostRegionDispatchPriorityDeferred {
		if parseHostRegionAdapter.storeHostRegionDeferredDispatch != nil &&
			parseInputVersion <= parseHostRegionAdapter.storeHostRegionDeferredDispatch.getInputVersion {
			return HostRegionUpdateDispatchResult{}, fmt.Errorf(
				"runtime2: deferred input version %d must advance beyond queued version %d",
				parseInputVersion,
				parseHostRegionAdapter.storeHostRegionDeferredDispatch.getInputVersion,
			)
		}
		hasDeferredSuperseded := parseHostRegionAdapter.storeHostRegionDeferredDispatch != nil
		parseHostRegionAdapter.storeHostRegionDeferredDispatch = &hostRegionDeferredDispatch{
			getInputVersion:        parseInputVersion,
			getSnapshotEnvelope:    getSnapshotEnvelope,
			getSnapshotFingerprint: getFingerprintResult.GetSnapshotFingerprint,
		}
		return HostRegionUpdateDispatchResult{
			HasScheduled:           false,
			HasNoChange:            false,
			HasDeferredQueued:      true,
			HasDeferredSuperseded:  hasDeferredSuperseded,
			GetDispatchPriority:    getDispatchPriority,
			GetSnapshotEnvelope:    getSnapshotEnvelope,
			GetSnapshotFingerprint: getFingerprintResult.GetSnapshotFingerprint,
		}, nil
	}
	getUpdateResult, parseUpdateErr := parseHostRegionAdapter.HandleHostRegionUpdate(parseInputVersion)
	if parseUpdateErr != nil {
		return HostRegionUpdateDispatchResult{}, parseUpdateErr
	}
	hasDeferredCanceled := false
	if parseHostRegionAdapter.storeHostRegionDeferredDispatch != nil &&
		parseInputVersion > parseHostRegionAdapter.storeHostRegionDeferredDispatch.getInputVersion {
		parseHostRegionAdapter.storeHostRegionDeferredDispatch = nil
		hasDeferredCanceled = true
	}
	return HostRegionUpdateDispatchResult{
		HasScheduled:           true,
		HasNoChange:            false,
		HasDeferredCanceled:    hasDeferredCanceled,
		GetDispatchPriority:    getDispatchPriority,
		GetSchedulerJob:        getUpdateResult.GetSchedulerJob,
		GetSnapshotEnvelope:    getSnapshotEnvelope,
		GetSnapshotFingerprint: getFingerprintResult.GetSnapshotFingerprint,
	}, nil
}

// GetHostRegionDeferredInputVersion reports the currently queued deferred input version, or zero when no deferred update is queued.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionDeferredInputVersion() uint64 {
	if parseHostRegionAdapter == nil || parseHostRegionAdapter.storeHostRegionDeferredDispatch == nil {
		return 0
	}
	return parseHostRegionAdapter.storeHostRegionDeferredDispatch.getInputVersion
}

// HandleHostRegionOwnerInvalidate clears queued region work after owner-side invalidation.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionOwnerInvalidate() (HostRegionOwnerInvalidateResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionOwnerInvalidateResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if _, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID); !hasCoordinatorEntry {
		return HostRegionOwnerInvalidateResult{}, fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	hasSchedulerCanceled := parseHostRegionAdapter.storeScheduler.HandleSchedulerCancel(string(parseHostRegionAdapter.storeRegionInstanceID))
	hasDeferredCleared := parseHostRegionAdapter.storeHostRegionDeferredDispatch != nil
	parseHostRegionAdapter.storeHostRegionDeferredDispatch = nil
	return HostRegionOwnerInvalidateResult{
		HasSchedulerCanceled: hasSchedulerCanceled,
		HasDeferredCleared:   hasDeferredCleared,
	}, nil
}

// HandleHostRegionOwnerRerender records owner-rerender committed precedence for one mounted region.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionOwnerRerender(parseInputVersion uint64) error {
	if parseHostRegionAdapter == nil {
		return fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseInputVersion == 0 {
		return fmt.Errorf("runtime2: owner rerender input version is required")
	}
	if _, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID); !hasCoordinatorEntry {
		return fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	if parseCommitErr := parseHostRegionAdapter.storeCoordinator.CommitRegion(parseHostRegionAdapter.storeRegionInstanceID, parseInputVersion); parseCommitErr != nil {
		return parseCommitErr
	}
	parseHostRegionAdapter.storeHostRegionLatestValidVersion = parseHostRegionMaxVersion(
		parseHostRegionAdapter.storeHostRegionLatestValidVersion,
		parseInputVersion,
	)
	return nil
}

// HandleHostRegionWorkerOutput applies one worker output commit attempt under owner-rerender precedence rules.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionWorkerOutput(parseInputVersion uint64) (HostRegionWorkerOutputResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionWorkerOutputResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	getPatchReadyResult, parsePatchReadyErr := parseHostRegionAdapter.HandleHostRegionPatchReady(parseInputVersion)
	if parsePatchReadyErr != nil {
		return HostRegionWorkerOutputResult{}, parsePatchReadyErr
	}
	if getPatchReadyResult.HasIgnored {
		return HostRegionWorkerOutputResult{
			HasIgnored: true,
		}, nil
	}
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return HostRegionWorkerOutputResult{
			HasIgnored: true,
		}, nil
	}
	if parseInputVersion <= getCoordinatorEntry.LastCommittedVersion {
		return HostRegionWorkerOutputResult{
			HasIgnored: true,
		}, nil
	}
	if parseCommitErr := parseHostRegionAdapter.storeCoordinator.CommitRegion(parseHostRegionAdapter.storeRegionInstanceID, parseInputVersion); parseCommitErr != nil {
		return HostRegionWorkerOutputResult{}, parseCommitErr
	}
	parseHostRegionAdapter.storeHostRegionLatestValidVersion = parseHostRegionMaxVersion(
		parseHostRegionAdapter.storeHostRegionLatestValidVersion,
		parseInputVersion,
	)
	return HostRegionWorkerOutputResult{
		HasCommitted: true,
	}, nil
}

// HandleHostRegionOwnerRemove disposes one mounted region and suppresses late worker output.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionOwnerRemove() (HostRegionOwnerRemoveResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionOwnerRemoveResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if _, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID); !hasCoordinatorEntry {
		parseHostRegionAdapter.isHostRegionRemoved = true
		return HostRegionOwnerRemoveResult{
			HasLateWorkerOutputSuppressed: true,
		}, nil
	}
	if _, parseDisposeErr := parseHostRegionAdapter.HandleHostRegionDispose(); parseDisposeErr != nil {
		return HostRegionOwnerRemoveResult{}, parseDisposeErr
	}
	parseHostRegionAdapter.isHostRegionRemoved = true
	return HostRegionOwnerRemoveResult{
		HasDisposed:                   true,
		HasLateWorkerOutputSuppressed: true,
	}, nil
}

// HandleHostRegionStructuralRemount remounts one region when renderer identity or shell ownership changed.
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
		Epoch:               getRemountEpoch,
		AssignedWorkerShard: string(getSchedulerJob.GetSchedulerShardID),
	}); parseMountErr != nil {
		return HostRegionStructuralRemountResult{}, parseMountErr
	}
	parseHostRegionAdapter.isHostRegionLocalShellOwned = parseIsLocalShellOwned
	parseHostRegionAdapter.storeHostRegionSnapshotFingerprint = ""
	parseHostRegionAdapter.storeHostRegionDeferredDispatch = nil
	parseHostRegionAdapter.isHostRegionFallbackPending = false
	parseHostRegionAdapter.isHostRegionFallbackActive = false
	parseHostRegionAdapter.isHostRegionRepairPending = false
	parseHostRegionAdapter.isHostRegionHydrationComplete = false
	parseHostRegionAdapter.hasHostRegionPostHydrationAttached = false
	parseHostRegionAdapter.storeHostRegionRepairRemountEpoch = 0
	parseHostRegionAdapter.storeHostRegionRepairVersionFloor = 0
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
	if parseFallbackErr := parseHostRegionAdapter.storeCoordinator.FallbackRegion(parseHostRegionAdapter.storeRegionInstanceID); parseFallbackErr != nil {
		return HostRegionFallbackMirrorResult{}, parseFallbackErr
	}
	hasSchedulerFallback := parseHostRegionAdapter.storeScheduler.HandleSchedulerFallback(getRegionID)
	parseHostRegionAdapter.isHostRegionFallbackPending = false
	parseHostRegionAdapter.isHostRegionFallbackActive = true
	return HostRegionFallbackMirrorResult{
		HasCoordinatorFallback: true,
		HasSchedulerFallback:   hasSchedulerFallback,
	}, nil
}

// HandleHostRegionPatchReady validates one patch-ready version against fallback and repair ownership gates.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionPatchReady(parseInputVersion uint64) (HostRegionPatchReadyResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionPatchReadyResult{}, fmt.Errorf("runtime2: host region adapter is nil")
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
		return HostRegionPatchReadyResult{
			HasIgnored:      true,
			GetIgnoreReason: "stale-before-repair-floor",
		}, nil
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
		getCoordinatorEntry.IsFallback = true
		getCoordinatorEntry.CurrentState = CoordinatorStateFallback
		if parseStoreErr := parseHostRegionAdapter.storeCoordinator.storeMutableEntry(getCoordinatorEntry); parseStoreErr != nil {
			return HostRegionWorkerDeathResult{}, parseStoreErr
		}
		parseHostRegionAdapter.storeScheduler.HandleSchedulerFallback(string(parseHostRegionAdapter.storeRegionInstanceID))
		parseHostRegionAdapter.storeRecoveryCoordinator.EnterRegionLocalFallback(
			string(parseHostRegionAdapter.storeRegionInstanceID),
			"worker-repair-pending",
			getRecoveryResult.GetRemountEpoch,
			getVersionFloor,
		)
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
	if parseFallbackErr := parseHostRegionAdapter.storeCoordinator.FallbackRegion(parseHostRegionAdapter.storeRegionInstanceID); parseFallbackErr != nil {
		return HostRegionWorkerDeathResult{}, parseFallbackErr
	}
	parseHostRegionAdapter.storeScheduler.HandleSchedulerFallback(string(parseHostRegionAdapter.storeRegionInstanceID))
	parseHostRegionAdapter.storeHostRegionRepairVersionFloor = parseHostRegionMaxVersion(
		parseHostRegionAdapter.storeHostRegionRepairVersionFloor,
		getVersionFloor,
	)
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
	getCoordinatorEntry.AssignedWorkerShard = string(getSchedulerJob.GetSchedulerShardID)
	getCoordinatorEntry.LastDispatchedVersion = parseHostRegionMaxVersion(
		getCoordinatorEntry.LastDispatchedVersion,
		parseHostRegionAdapter.storeHostRegionRepairVersionFloor,
	)
	getCoordinatorEntry.LastCommittedVersion = parseHostRegionMaxVersion(
		getCoordinatorEntry.LastCommittedVersion,
		parseHostRegionAdapter.storeHostRegionRepairVersionFloor,
	)
	getCoordinatorEntry.IsFallback = false
	getCoordinatorEntry.CurrentState = CoordinatorStateMounted
	if parseStoreErr := parseHostRegionAdapter.storeCoordinator.storeMutableEntry(getCoordinatorEntry); parseStoreErr != nil {
		return HostRegionRepairRemountResult{}, parseStoreErr
	}
	parseHostRegionAdapter.storeScheduler.ClearSchedulerFallbackOwnership(string(getSpec.RegionInstanceID))
	parseHostRegionAdapter.storeRecoveryCoordinator.ClearRegionLocalFallback(string(getSpec.RegionInstanceID))
	parseHostRegionAdapter.isHostRegionRepairPending = false
	parseHostRegionAdapter.isHostRegionFallbackActive = false
	parseHostRegionAdapter.isHostRegionFallbackPending = false
	parseHostRegionAdapter.storeHostRegionRepairRemountEpoch = 0
	parseHostRegionAdapter.storeHostRegionSnapshotFingerprint = ""
	parseHostRegionAdapter.storeHostRegionDeferredDispatch = nil
	parseHostRegionAdapter.hasHostRegionPostHydrationAttached = false
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
	parseHostRegionAdapter.hasHostRegionPostHydrationAttached = true
	return HostRegionHydrationAttachResult{
		HasAttached: true,
	}, nil
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
