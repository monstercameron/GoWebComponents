package runtime2

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// HostRegionAdapter owns host-side runtime2 handles for one live region instance.
type HostRegionAdapter struct {
	storeRegionInstanceID              RegionInstanceID
	storeCoordinator                   *Coordinator
	storeScheduler                     *Scheduler
	storeRecoveryCoordinator           *RecoveryCoordinator
	storeRegionDOMIndexHandle          *RegionDOMIndex
	storeHostRegionSourceLookup        HostRegionSourceLookup
	storeHostRegionSnapshotFingerprint string
	storeHostRegionSnapshotHash        [sha256.Size]byte
	storeHostRegionDeferredDispatch    *hostRegionDeferredDispatch
	storeHostRegionRepairRemountEpoch  uint64
	storeHostRegionRepairVersionFloor  uint64
	storeHostRegionLatestValidVersion  uint64
	storeHostRegionLastPatchVersion    uint64
	storeHostRegionTransportTier       TransportTier
	storeHostRegionSnapshotTier        TransportTier
	storeHostRegionPatchTier           TransportTier
	storeHostRegionFallbackReason      string
	storeHostRegionDispatchAt          time.Time
	storeHostRegionPatchReadyAt        time.Time
	storeHostRegionCommitAt            time.Time
	storeHostRegionDispatchToPatchNS   uint64
	storeHostRegionDispatchToCommitNS  uint64
	storeHostRegionPatchToCommitNS     uint64
	storeHostRegionPatchIdempotency    *PatchIdempotencyTracker
	storeHostRegionDiagnosticRing      []ControlEnvelope
	storeHostRegionSnapshotDowngrade   DiagnosticDowngradeReason
	storeHostRegionPatchDowngrade      DiagnosticDowngradeReason
	isHostRegionRemoved                bool
	isHostRegionFallbackPending        bool
	isHostRegionFallbackActive         bool
	isHostRegionRepairPending          bool
	isHostRegionHydrationComplete      bool
	hasHostRegionPostHydrationAttached bool
	hasHostRegionHydratedShellAnchor   bool
	hasHostRegionSnapshotHash          bool
	hasHostRegionSnapshotDowngrade     bool
	hasHostRegionPatchDowngrade        bool
	isHostRegionLocalShellOwned        bool
}

const getHostRegionDiagnosticRingLimit = 32

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

type hostRegionSnapshotHashResult struct {
	getSnapshotHash [sha256.Size]byte
	hasNoChange     bool
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

// HostRegionDiagnosticResult reports whether one worker diagnostic was stored or ignored.
type HostRegionDiagnosticResult struct {
	HasStored        bool
	HasIgnored       bool
	GetIgnoreReason  string
	GetDiagnostic    ControlEnvelope
	GetDiagnosticLen int
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

// HostRegionShellIdentityMismatchResult reports shell-marker identity mismatches for region or renderer IDs.
type HostRegionShellIdentityMismatchResult struct {
	HasMismatch           bool
	HasRegionIDMismatch   bool
	HasRendererIDMismatch bool
}

// HostRegionShellAnchorCheckResult reports whether one hydrated shell anchor is missing for the mounted region.
type HostRegionShellAnchorCheckResult struct {
	HasMissingAnchor bool
}

// HostRegionShellMismatchFallbackResult reports fallback ownership and remount-floor outcomes after shell mismatch.
type HostRegionShellMismatchFallbackResult struct {
	HasFallbackEntered  bool
	HasOutputSuppressed bool
	GetRemountEpoch     uint64
	GetVersionFloor     uint64
}

// HostRegionRuntimeMode identifies one runtime-status ownership mode for a host region.
type HostRegionRuntimeMode string

const (
	// HostRegionRuntimeModeLocalShell reports local-shell ownership before worker attach.
	HostRegionRuntimeModeLocalShell HostRegionRuntimeMode = "local-shell"
	// HostRegionRuntimeModeWorkerAttached reports active worker-backed attach.
	HostRegionRuntimeModeWorkerAttached HostRegionRuntimeMode = "worker-attached"
	// HostRegionRuntimeModeFallback reports locally-owned fallback mode.
	HostRegionRuntimeModeFallback HostRegionRuntimeMode = "fallback"
)

// HostRegionRuntimeStatus reports one host region runtime status snapshot for observability surfaces.
type HostRegionRuntimeStatus struct {
	GetRegionInstanceID            RegionInstanceID
	GetRegionMode                  HostRegionRuntimeMode
	GetAssignedWorkerShard         string
	GetRendererID                  RendererID
	GetEpoch                       uint64
	GetIsHydrationComplete         bool
	HasHydratedShellAnchor         bool
	HasPostHydrationAttached       bool
	GetLastSnapshotVersion         uint64
	GetLastDispatchedVersion       uint64
	GetLastCommittedVersion        uint64
	GetTransportTier               TransportTier
	HasSnapshotDowngrade           bool
	GetSnapshotDowngradePath       DiagnosticDowngradePath
	GetSnapshotDowngradeReason     string
	HasPatchDowngrade              bool
	GetPatchDowngradePath          DiagnosticDowngradePath
	GetPatchDowngradeReason        string
	GetDroppedStalePatchCount      uint64
	GetIgnoredStaleDiagnosticCount uint64
	GetFallbackReason              string
}

// HostRegionRoundTripTiming reports dispatch, patch-ready, and commit timing spans for one region.
type HostRegionRoundTripTiming struct {
	GetDispatchToPatchReadyNS uint64
	GetDispatchToCommitNS     uint64
	GetPatchReadyToCommitNS   uint64
}

// HostRegionTransportDowngradeStatus reports separate snapshot and patch transport downgrade accounting for one region.
type HostRegionTransportDowngradeStatus struct {
	GetSnapshotTransportTier TransportTier
	HasSnapshotDowngrade     bool
	GetSnapshotDowngrade     DiagnosticDowngradeReason
	GetPatchTransportTier    TransportTier
	HasPatchDowngrade        bool
	GetPatchDowngrade        DiagnosticDowngradeReason
}

// HostRegionDiagnosticsSnapshot reports one read-only host diagnostics snapshot for redacted events, downgrade accounting, and counters.
type HostRegionDiagnosticsSnapshot struct {
	GetRegionInstanceID            RegionInstanceID
	GetDiagnosticEvents            []ControlEnvelope
	GetTransportDowngradeStatus    HostRegionTransportDowngradeStatus
	GetDroppedStalePatchCount      uint64
	GetIgnoredStaleDiagnosticCount uint64
	GetRepairTriggeredRemountCount uint64
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
	return HostRegionDisposeResult{
		HasCoordinatorDisposed: true,
		HasSchedulerDisposed:   hasSchedulerDisposed,
		GetClearedDOMNodeCount: getDOMClearResult.GetClearedNodeCount,
	}, nil
}

// HandleHostRegionDiagnosticEnvelope records one validated diagnostic control envelope in durable host-region diagnostic state.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionDiagnosticEnvelope(parseEnvelope ControlEnvelope) (HostRegionDiagnosticResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionDiagnosticResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	parseEnvelope = RedactControlDiagnosticEnvelope(parseEnvelope)
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		return HostRegionDiagnosticResult{}, parseErr
	}
	if parseEnvelope.Kind != ControlKindDiagnostic {
		return HostRegionDiagnosticResult{}, fmt.Errorf("runtime2: control kind %q is not diagnostic", parseEnvelope.Kind)
	}
	if parseEnvelope.RegionInstanceID != parseHostRegionAdapter.storeRegionInstanceID {
		return HostRegionDiagnosticResult{}, fmt.Errorf(
			"runtime2: host region adapter mounted for %q cannot store diagnostic for region %q",
			parseHostRegionAdapter.storeRegionInstanceID,
			parseEnvelope.RegionInstanceID,
		)
	}
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return HostRegionDiagnosticResult{}, fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	if hasDiagnosticIgnored, getIgnoreReason := parseHostRegionAdapter.shouldHostRegionIgnoreDiagnostic(parseEnvelope, getCoordinatorEntry); hasDiagnosticIgnored {
		if shouldHostRegionCountStaleDiagnosticIgnore(getIgnoreReason) {
			if _, parseCountErr := parseHostRegionAdapter.storeCoordinator.IncrementRegionIgnoredStaleDiagnosticCount(parseHostRegionAdapter.storeRegionInstanceID); parseCountErr != nil {
				return HostRegionDiagnosticResult{}, parseCountErr
			}
		}
		return HostRegionDiagnosticResult{
			HasIgnored:       true,
			GetIgnoreReason:  getIgnoreReason,
			GetDiagnostic:    parseEnvelope,
			GetDiagnosticLen: len(parseHostRegionAdapter.storeHostRegionDiagnosticRing),
		}, nil
	}
	parseHostRegionAdapter.storeHostRegionDiagnosticDowngrade(parseEnvelope)
	parseHostRegionAdapter.storeHostRegionDiagnosticEnvelope(parseEnvelope)
	return HostRegionDiagnosticResult{
		HasStored:        true,
		GetDiagnostic:    parseEnvelope,
		GetDiagnosticLen: len(parseHostRegionAdapter.storeHostRegionDiagnosticRing),
	}, nil
}

// storeHostRegionDiagnosticEnvelope appends one diagnostic envelope while keeping only the newest bounded ring entries.
func (parseHostRegionAdapter *HostRegionAdapter) storeHostRegionDiagnosticEnvelope(parseEnvelope ControlEnvelope) {
	if parseHostRegionAdapter == nil {
		return
	}
	parseHostRegionAdapter.storeHostRegionDiagnosticRing = append(parseHostRegionAdapter.storeHostRegionDiagnosticRing, parseEnvelope)
	if len(parseHostRegionAdapter.storeHostRegionDiagnosticRing) <= getHostRegionDiagnosticRingLimit {
		return
	}
	parseDropCount := len(parseHostRegionAdapter.storeHostRegionDiagnosticRing) - getHostRegionDiagnosticRingLimit
	parseHostRegionAdapter.storeHostRegionDiagnosticRing = append(
		[]ControlEnvelope(nil),
		parseHostRegionAdapter.storeHostRegionDiagnosticRing[parseDropCount:]...,
	)
}

// shouldHostRegionCountStaleDiagnosticIgnore reports whether one diagnostic ignore reason should increment stale-diagnostic counters.
func shouldHostRegionCountStaleDiagnosticIgnore(parseIgnoreReason string) bool {
	return strings.HasPrefix(parseIgnoreReason, "stale-")
}

// storeHostRegionDiagnosticDowngrade records one diagnostic downgrade into snapshot or patch accounting based on diagnostic event kind.
func (parseHostRegionAdapter *HostRegionAdapter) storeHostRegionDiagnosticDowngrade(parseEnvelope ControlEnvelope) {
	if parseHostRegionAdapter == nil || parseEnvelope.DiagnosticDowngrade == nil {
		return
	}
	parseDiagnosticKind, parseKindErr := ParseDiagnosticEventKind(parseEnvelope.DiagnosticType)
	if parseKindErr != nil {
		return
	}
	parseDowngrade := *parseEnvelope.DiagnosticDowngrade
	switch parseDiagnosticKind {
	case DiagnosticEventKindMount, DiagnosticEventKindUpdate:
		parseHostRegionAdapter.storeHostRegionSnapshotDowngrade = parseDowngrade
		parseHostRegionAdapter.hasHostRegionSnapshotDowngrade = true
		if parseEnvelope.TransportTier != "" {
			parseHostRegionAdapter.storeHostRegionSnapshotTier = parseEnvelope.TransportTier
		}
	case DiagnosticEventKindPatchReady:
		parseHostRegionAdapter.storeHostRegionPatchDowngrade = parseDowngrade
		parseHostRegionAdapter.hasHostRegionPatchDowngrade = true
		if parseEnvelope.TransportTier != "" {
			parseHostRegionAdapter.storeHostRegionPatchTier = parseEnvelope.TransportTier
			parseHostRegionAdapter.storeHostRegionTransportTier = parseEnvelope.TransportTier
		}
	}
}

// shouldHostRegionIgnoreDiagnostic reports whether one diagnostic envelope is stale for the region's latest fallback or repair state.
func (parseHostRegionAdapter *HostRegionAdapter) shouldHostRegionIgnoreDiagnostic(parseEnvelope ControlEnvelope, parseCoordinatorEntry CoordinatorEntry) (bool, string) {
	if parseHostRegionAdapter == nil {
		return false, ""
	}
	getRegionID := string(parseHostRegionAdapter.storeRegionInstanceID)
	getFallbackState, hasFallbackState := parseHostRegionAdapter.storeRecoveryCoordinator.GetRegionFallbackState(getRegionID)
	if parseEnvelope.Epoch > 0 {
		if parseEnvelope.Epoch < parseCoordinatorEntry.Epoch {
			return true, "stale-epoch"
		}
		if parseHostRegionAdapter.isHostRegionRepairPending &&
			parseHostRegionAdapter.storeHostRegionRepairRemountEpoch > 0 &&
			parseEnvelope.Epoch < parseHostRegionAdapter.storeHostRegionRepairRemountEpoch {
			return true, "stale-repair-epoch"
		}
		if hasFallbackState && getFallbackState.GetEpoch > 0 && parseEnvelope.Epoch < getFallbackState.GetEpoch {
			return true, "stale-fallback-epoch"
		}
	}
	getVersionFloor := parseHostRegionAdapter.getHostRegionDiagnosticVersionFloor(parseCoordinatorEntry)
	if parseEnvelope.InputVersion > 0 {
		if parseEnvelope.InputVersion < getVersionFloor {
			return true, "stale-version"
		}
		if hasFallbackState && getFallbackState.GetInputVersion > 0 && parseEnvelope.InputVersion < getFallbackState.GetInputVersion {
			return true, "stale-fallback-version"
		}
	}
	if parseHostRegionAdapter.storeRecoveryCoordinator.HandleWorkerDiagnostic(
		getRegionID,
		parseEnvelope.Epoch,
		parseEnvelope.InputVersion,
	).HasIgnored {
		return true, "fallback-active"
	}
	return false, ""
}

// getHostRegionDiagnosticVersionFloor reports the oldest diagnostic input version still considered current.
func (parseHostRegionAdapter *HostRegionAdapter) getHostRegionDiagnosticVersionFloor(parseCoordinatorEntry CoordinatorEntry) uint64 {
	if parseHostRegionAdapter == nil {
		return 0
	}
	return parseHostRegionMaxVersion(
		parseCoordinatorEntry.LastCommittedVersion,
		parseCoordinatorEntry.LastDispatchedVersion,
		parseHostRegionAdapter.storeHostRegionLatestValidVersion,
		parseHostRegionAdapter.storeHostRegionRepairVersionFloor,
	)
}

// SetHostRegionPatchTransportTier stores one latest patch transport tier selected by worker patch-ready signaling.
func (parseHostRegionAdapter *HostRegionAdapter) SetHostRegionPatchTransportTier(parseTransportTier TransportTier) error {
	if parseHostRegionAdapter == nil {
		return fmt.Errorf("runtime2: host region adapter is nil")
	}
	getTransportTier, parseTierErr := ParseTransportTier(string(parseTransportTier))
	if parseTierErr != nil {
		return parseTierErr
	}
	parseHostRegionAdapter.storeHostRegionPatchTier = getTransportTier
	parseHostRegionAdapter.storeHostRegionTransportTier = getTransportTier
	return nil
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
	return parseHostRegionAdapter.handleHostRegionDeclaredSourceLookupNormalized(getSourceIDs)
}

// handleHostRegionDeclaredSourceLookupNormalized resolves one normalized declared source ID set through the host source lookup bridge.
func (parseHostRegionAdapter *HostRegionAdapter) handleHostRegionDeclaredSourceLookupNormalized(parseSourceIDs []string) (HostRegionSourceSnapshot, error) {
	if len(parseSourceIDs) == 0 {
		return HostRegionSourceSnapshot{}, nil
	}
	if parseHostRegionAdapter.storeHostRegionSourceLookup == nil {
		return HostRegionSourceSnapshot{}, fmt.Errorf("runtime2: host source lookup is not configured")
	}
	getSourceValues, getSourceVersions, parseSourceLookupErr := parseHostRegionAdapter.storeHostRegionSourceLookup(parseSourceIDs)
	if parseSourceLookupErr != nil {
		return HostRegionSourceSnapshot{}, parseSourceLookupErr
	}
	buildSourceValues, parseSourceValuesErr := buildSnapshotSourceValues(parseSourceIDs, getSourceValues)
	if parseSourceValuesErr != nil {
		return HostRegionSourceSnapshot{}, parseSourceValuesErr
	}
	buildSourceVersions := make(map[string]uint64, len(parseSourceIDs))
	for _, getSourceID := range parseSourceIDs {
		getSourceVersion, hasSourceVersion := getSourceVersions[getSourceID]
		if !hasSourceVersion {
			continue
		}
		buildSourceVersions[getSourceID] = getSourceVersion
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
	getSourceSnapshot, parseSourceSnapshotErr := parseHostRegionAdapter.handleHostRegionDeclaredSourceLookupNormalized(getSpec.SourceIDs)
	if parseSourceSnapshotErr != nil {
		return SnapshotEnvelope{}, parseSourceSnapshotErr
	}
	getSnapshotEnvelope, parseSnapshotEnvelopeErr := buildSnapshotEnvelopeFromNormalizedSourceIDs(
		getSpec.RegionInstanceID,
		getCoordinatorEntry.Epoch,
		parseInputVersion,
		getSpec.Props,
		getSpec.SourceIDs,
		getSourceSnapshot.GetSourceValues,
		getSourceSnapshot.GetSourceVersions,
	)
	if parseSnapshotEnvelopeErr != nil {
		return SnapshotEnvelope{}, parseSnapshotEnvelopeErr
	}
	if parseSourceIDsErr := parseHostRegionAdapter.storeCoordinator.SetRegionSourceIDs(getSpec.RegionInstanceID, getSpec.SourceIDs); parseSourceIDsErr != nil {
		return SnapshotEnvelope{}, parseSourceIDsErr
	}
	if parseSnapshotVersionErr := parseHostRegionAdapter.storeCoordinator.SetRegionLastSnapshotVersion(getSpec.RegionInstanceID, parseInputVersion); parseSnapshotVersionErr != nil {
		return SnapshotEnvelope{}, parseSnapshotVersionErr
	}
	return getSnapshotEnvelope, nil
}

// HandleHostRegionSnapshotFingerprint computes and stores one stable snapshot fingerprint for host-side no-change detection.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionSnapshotFingerprint(parseSnapshotEnvelope SnapshotEnvelope) (HostRegionSnapshotFingerprintResult, error) {
	getSnapshotHashResult, parseSnapshotHashErr := parseHostRegionAdapter.handleHostRegionSnapshotHash(parseSnapshotEnvelope)
	if parseSnapshotHashErr != nil {
		return HostRegionSnapshotFingerprintResult{}, parseSnapshotHashErr
	}
	getSnapshotFingerprint := hex.EncodeToString(getSnapshotHashResult.getSnapshotHash[:])
	parseHostRegionAdapter.storeHostRegionSnapshotFingerprint = getSnapshotFingerprint
	return HostRegionSnapshotFingerprintResult{
		GetSnapshotFingerprint: getSnapshotFingerprint,
		HasNoChange:            getSnapshotHashResult.hasNoChange,
	}, nil
}

// handleHostRegionSnapshotHash computes and stores one snapshot SHA-256 digest for host-side no-change detection.
func (parseHostRegionAdapter *HostRegionAdapter) handleHostRegionSnapshotHash(parseSnapshotEnvelope SnapshotEnvelope) (hostRegionSnapshotHashResult, error) {
	if parseHostRegionAdapter == nil {
		return hostRegionSnapshotHashResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseSnapshotEnvelope.RegionInstanceID != parseHostRegionAdapter.storeRegionInstanceID {
		return hostRegionSnapshotHashResult{}, fmt.Errorf(
			"runtime2: host region adapter mounted for %q cannot fingerprint snapshot for region %q",
			parseHostRegionAdapter.storeRegionInstanceID,
			parseSnapshotEnvelope.RegionInstanceID,
		)
	}
	buildFingerprintEnvelope := parseSnapshotEnvelope
	buildFingerprintEnvelope.InputVersion = 1
	getSnapshotHash, parseSnapshotHashErr := GetSnapshotFingerprintHash(buildFingerprintEnvelope)
	if parseSnapshotHashErr != nil {
		return hostRegionSnapshotHashResult{}, parseSnapshotHashErr
	}
	hasNoChange := parseHostRegionAdapter.hasHostRegionSnapshotHash && getSnapshotHash == parseHostRegionAdapter.storeHostRegionSnapshotHash
	parseHostRegionAdapter.storeHostRegionSnapshotHash = getSnapshotHash
	parseHostRegionAdapter.hasHostRegionSnapshotHash = true
	return hostRegionSnapshotHashResult{
		getSnapshotHash: getSnapshotHash,
		hasNoChange:     hasNoChange,
	}, nil
}

// HandleHostRegionUpdateDispatch captures one update snapshot, applies no-change short-circuit rules, and schedules worker update dispatch only when needed.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionUpdateDispatch(parseSpec ParallelRegionSpec, parseInputVersion uint64) (HostRegionUpdateDispatchResult, error) {
	return parseHostRegionAdapter.HandleHostRegionUpdateDispatchWithPriority(parseSpec, parseInputVersion, HostRegionDispatchPriorityUrgent)
}

// HandleHostRegionUpdateDispatchWithTransition captures one update snapshot and maps transition updates onto deferred dispatch priority.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionUpdateDispatchWithTransition(
	parseSpec ParallelRegionSpec,
	parseInputVersion uint64,
	parseIsTransition bool,
) (HostRegionUpdateDispatchResult, error) {
	if parseIsTransition {
		return parseHostRegionAdapter.HandleHostRegionUpdateDispatchWithPriority(
			parseSpec,
			parseInputVersion,
			HostRegionDispatchPriorityDeferred,
		)
	}
	return parseHostRegionAdapter.HandleHostRegionUpdateDispatchWithPriority(
		parseSpec,
		parseInputVersion,
		HostRegionDispatchPriorityUrgent,
	)
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
	getSnapshotHashResult, parseSnapshotHashErr := parseHostRegionAdapter.handleHostRegionSnapshotHash(getSnapshotEnvelope)
	if parseSnapshotHashErr != nil {
		return HostRegionUpdateDispatchResult{}, parseSnapshotHashErr
	}
	if getSnapshotHashResult.hasNoChange {
		return HostRegionUpdateDispatchResult{
			HasScheduled:           false,
			HasNoChange:            true,
			GetDispatchPriority:    getDispatchPriority,
			GetSnapshotEnvelope:    getSnapshotEnvelope,
			GetSnapshotFingerprint: "",
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
			getSnapshotFingerprint: "",
		}
		return HostRegionUpdateDispatchResult{
			HasScheduled:           false,
			HasNoChange:            false,
			HasDeferredQueued:      true,
			HasDeferredSuperseded:  hasDeferredSuperseded,
			GetDispatchPriority:    getDispatchPriority,
			GetSnapshotEnvelope:    getSnapshotEnvelope,
			GetSnapshotFingerprint: "",
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
	parseHostRegionAdapter.storeHostRegionDispatchAt = time.Now()
	parseHostRegionAdapter.storeHostRegionPatchReadyAt = time.Time{}
	parseHostRegionAdapter.storeHostRegionCommitAt = time.Time{}
	parseHostRegionAdapter.storeHostRegionDispatchToPatchNS = 0
	parseHostRegionAdapter.storeHostRegionDispatchToCommitNS = 0
	parseHostRegionAdapter.storeHostRegionPatchToCommitNS = 0
	return HostRegionUpdateDispatchResult{
		HasScheduled:           true,
		HasNoChange:            false,
		HasDeferredCanceled:    hasDeferredCanceled,
		GetDispatchPriority:    getDispatchPriority,
		GetSchedulerJob:        getUpdateResult.GetSchedulerJob,
		GetSnapshotEnvelope:    getSnapshotEnvelope,
		GetSnapshotFingerprint: "",
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
	return parseHostRegionAdapter.parseHandleHostRegionWorkerOutputCommit(parseInputVersion)
}

// parseHandleHostRegionWorkerOutputCommit applies one committed worker output version without patch-ready gating.
func (parseHostRegionAdapter *HostRegionAdapter) parseHandleHostRegionWorkerOutputCommit(parseInputVersion uint64) (HostRegionWorkerOutputResult, error) {
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return HostRegionWorkerOutputResult{
			HasIgnored: true,
		}, nil
	}
	if parseInputVersion <= getCoordinatorEntry.LastCommittedVersion {
		if _, parseDropErr := parseHostRegionAdapter.storeCoordinator.IncrementRegionDroppedStalePatchCount(parseHostRegionAdapter.storeRegionInstanceID); parseDropErr != nil {
			return HostRegionWorkerOutputResult{}, parseDropErr
		}
		return HostRegionWorkerOutputResult{
			HasIgnored: true,
		}, nil
	}
	if parseCommitErr := parseHostRegionAdapter.storeCoordinator.CommitRegion(parseHostRegionAdapter.storeRegionInstanceID, parseInputVersion); parseCommitErr != nil {
		return HostRegionWorkerOutputResult{}, parseCommitErr
	}
	parseHostRegionAdapter.storeHostRegionCommitAt = time.Now()
	if !parseHostRegionAdapter.storeHostRegionDispatchAt.IsZero() {
		parseDispatchToCommitNS := parseHostRegionAdapter.storeHostRegionCommitAt.Sub(parseHostRegionAdapter.storeHostRegionDispatchAt).Nanoseconds()
		if parseDispatchToCommitNS > 0 {
			parseHostRegionAdapter.storeHostRegionDispatchToCommitNS = uint64(parseDispatchToCommitNS)
		}
	}
	if !parseHostRegionAdapter.storeHostRegionPatchReadyAt.IsZero() {
		parsePatchToCommitNS := parseHostRegionAdapter.storeHostRegionCommitAt.Sub(parseHostRegionAdapter.storeHostRegionPatchReadyAt).Nanoseconds()
		if parsePatchToCommitNS > 0 {
			parseHostRegionAdapter.storeHostRegionPatchToCommitNS = uint64(parsePatchToCommitNS)
		}
	}
	parseHostRegionAdapter.storeHostRegionLatestValidVersion = parseHostRegionMaxVersion(
		parseHostRegionAdapter.storeHostRegionLatestValidVersion,
		parseInputVersion,
	)
	return HostRegionWorkerOutputResult{
		HasCommitted: true,
	}, nil
}

// HandleHostRegionPatchCommit parses and commits one typed patch stream into DOM transaction boundaries.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionPatchCommit(parsePatch PatchStreamRaw, parseDOMCommitter *DOMCommitter) (HostRegionWorkerOutputResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionWorkerOutputResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseDOMCommitter == nil {
		parseDOMCommitter = BuildDOMCommitter(parseHostRegionAdapter.storeRegionDOMIndexHandle)
	}
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return HostRegionWorkerOutputResult{}, fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	if parsePatch.GetHeader.PatchVersion > 0 && parsePatch.GetHeader.PatchVersion <= parseHostRegionAdapter.storeHostRegionLastPatchVersion {
		return HostRegionWorkerOutputResult{
			HasIgnored: true,
		}, nil
	}
	buildRegionID := string(parseHostRegionAdapter.storeRegionInstanceID)
	buildKnownNodeIDs, buildSiblingCountByParent := BuildRegionDOMPatchLookupMaps(parseHostRegionAdapter.storeRegionDOMIndexHandle, buildRegionID)
	parsePatchResult, hasPatchApply, parsePatchErr := ParsePatchStreamTransaction(
		parsePatch,
		buildRegionID,
		getCoordinatorEntry.Epoch,
		buildKnownNodeIDs,
		buildSiblingCountByParent,
		parseHostRegionAdapter.storeHostRegionPatchIdempotency,
	)
	if parsePatchErr != nil {
		return HostRegionWorkerOutputResult{}, parsePatchErr
	}
	if !hasPatchApply {
		return HostRegionWorkerOutputResult{
			HasIgnored: true,
		}, nil
	}
	if parsePatchResult.GetHeader.PatchVersion <= parseHostRegionAdapter.storeHostRegionLastPatchVersion {
		return HostRegionWorkerOutputResult{
			HasIgnored: true,
		}, nil
	}
	getPatchReadyResult, parsePatchReadyErr := parseHostRegionAdapter.HandleHostRegionPatchReadyWithVersion(
		parsePatchResult.GetHeader.PatchVersion,
		parsePatchResult.GetHeader.InputVersion,
	)
	if parsePatchReadyErr != nil {
		return HostRegionWorkerOutputResult{}, parsePatchReadyErr
	}
	if getPatchReadyResult.HasIgnored {
		return HostRegionWorkerOutputResult{
			HasIgnored: true,
		}, nil
	}
	_, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(parsePatchResult.GetTransaction)
	if parseTransactionErr != nil {
		parseFailureKind := parseGetHostRegionDOMFailureKind(parseTransactionErr)
		_ = parseHostRegionAdapter.HandleHostRegionDOMPatchTransactionFailure(parseFailureKind, parsePatchResult.GetHeader.InputVersion)
		return HostRegionWorkerOutputResult{}, parseTransactionErr
	}
	parseWorkerOutputResult, parseWorkerOutputErr := parseHostRegionAdapter.parseHandleHostRegionWorkerOutputCommit(parsePatchResult.GetHeader.InputVersion)
	if parseWorkerOutputErr != nil {
		return HostRegionWorkerOutputResult{}, parseWorkerOutputErr
	}
	if parseWorkerOutputResult.HasCommitted {
		parseHostRegionAdapter.storeHostRegionLastPatchVersion = parsePatchResult.GetHeader.PatchVersion
	}
	return parseWorkerOutputResult, nil
}

// parseGetHostRegionDOMFailureKind maps one commit-layer error into one recovery failure kind.
func parseGetHostRegionDOMFailureKind(parseErr error) DOMCommitFailureKind {
	if parseErr == nil {
		return DOMCommitFailureKindMissingNodeLookup
	}
	parseErrorText := strings.ToLower(parseErr.Error())
	switch {
	case strings.Contains(parseErrorText, "parent") || strings.Contains(parseErrorText, "anchor"):
		return DOMCommitFailureKindMissingParentAnchor
	case strings.Contains(parseErrorText, "move") || strings.Contains(parseErrorText, "destination"):
		return DOMCommitFailureKindInvalidKeyedMove
	default:
		return DOMCommitFailureKindMissingNodeLookup
	}
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
		SourceIDs:           getSpec.SourceIDs,
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
	if parseFallbackErr := parseHostRegionAdapter.storeCoordinator.FallbackRegion(parseHostRegionAdapter.storeRegionInstanceID); parseFallbackErr != nil {
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
		if _, parseDropErr := parseHostRegionAdapter.storeCoordinator.IncrementRegionDroppedStalePatchCount(parseHostRegionAdapter.storeRegionInstanceID); parseDropErr != nil {
			return HostRegionPatchReadyResult{}, parseDropErr
		}
		return HostRegionPatchReadyResult{
			HasIgnored:      true,
			GetIgnoreReason: "stale-before-repair-floor",
		}, nil
	}
	if parsePatchVersion <= parseHostRegionAdapter.storeHostRegionLastPatchVersion {
		if _, parseDropErr := parseHostRegionAdapter.storeCoordinator.IncrementRegionDroppedStalePatchCount(parseHostRegionAdapter.storeRegionInstanceID); parseDropErr != nil {
			return HostRegionPatchReadyResult{}, parseDropErr
		}
		return HostRegionPatchReadyResult{
			HasIgnored:      true,
			GetIgnoreReason: "stale-patch-version",
		}, nil
	}
	if parseHostRegionAdapter.storeHostRegionPatchReadyAt.IsZero() {
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
	if parseFallbackErr := parseHostRegionAdapter.storeCoordinator.FallbackRegion(parseHostRegionAdapter.storeRegionInstanceID); parseFallbackErr != nil {
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
	if _, parseCounterErr := parseHostRegionAdapter.storeCoordinator.IncrementRegionRepairRemountCount(getSpec.RegionInstanceID); parseCounterErr != nil {
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
	parseHostRegionAdapter.storeHostRegionDeferredDispatch = nil
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
	if parseAttachErr := parseHostRegionAdapter.storeCoordinator.SetRegionAttached(parseHostRegionAdapter.storeRegionInstanceID, true); parseAttachErr != nil {
		return HostRegionHydrationAttachResult{}, parseAttachErr
	}
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
	if parseHostRegionAdapter == nil {
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
	if parseFallbackErr := parseHostRegionAdapter.storeCoordinator.FallbackRegion(parseHostRegionAdapter.storeRegionInstanceID); parseFallbackErr != nil {
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
