package runtime2

import (
	"fmt"
	"strings"
	"time"
)

func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionUpdateDispatch(parseSpec ParallelRegionSpec, parseInputVersion uint64) (HostRegionUpdateDispatchResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionUpdateDispatchResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	return parseHostRegionAdapter.handleHostRegionUpdateDispatchWithKnownPriority(parseSpec, parseInputVersion, HostRegionDispatchPriorityUrgent)
}

// HandleHostRegionUpdateDispatchWithTransition captures one update snapshot and maps transition updates onto deferred dispatch priority.
// It routes directly to the inner dispatch path with a known-valid priority to avoid ParseHostRegionDispatchPriority overhead on every call.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionUpdateDispatchWithTransition(
	parseSpec ParallelRegionSpec,
	parseInputVersion uint64,
	parseIsTransition bool,
) (HostRegionUpdateDispatchResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionUpdateDispatchResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseIsTransition {
		return parseHostRegionAdapter.handleHostRegionUpdateDispatchWithKnownPriority(parseSpec, parseInputVersion, HostRegionDispatchPriorityDeferred)
	}
	return parseHostRegionAdapter.handleHostRegionUpdateDispatchWithKnownPriority(parseSpec, parseInputVersion, HostRegionDispatchPriorityUrgent)
}

// HandleHostRegionUpdateDispatchWithPriority captures one update snapshot and dispatches using the requested host update priority classification.
// External callers may pass any HostRegionDispatchPriority value; validation runs before delegating to the inner dispatch path.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionUpdateDispatchWithPriority(
	parseSpec ParallelRegionSpec,
	parseInputVersion uint64,
	parseDispatchPriority HostRegionDispatchPriority,
) (HostRegionUpdateDispatchResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionUpdateDispatchResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	switch parseDispatchPriority {
	case HostRegionDispatchPriorityUrgent, HostRegionDispatchPriorityDeferred:
		return parseHostRegionAdapter.handleHostRegionUpdateDispatchWithKnownPriority(
			parseSpec,
			parseInputVersion,
			parseDispatchPriority,
		)
	default:
		return HostRegionUpdateDispatchResult{}, fmt.Errorf("runtime2: host dispatch priority %q is unsupported", parseDispatchPriority)
	}
}

// handleHostRegionUpdateDispatchWithKnownPriority is the inner dispatch path used by callers that have already validated the priority constant.
func (parseHostRegionAdapter *HostRegionAdapter) handleHostRegionUpdateDispatchWithKnownPriority(
	parseSpec ParallelRegionSpec,
	parseInputVersion uint64,
	getDispatchPriority HostRegionDispatchPriority,
) (HostRegionUpdateDispatchResult, error) {
	if parseHostRegionAdapter.isHostRegionRepairPending {
		return HostRegionUpdateDispatchResult{}, fmt.Errorf("runtime2: host region %q repair remount is pending", parseHostRegionAdapter.storeRegionInstanceID)
	}
	getSnapshotEnvelope,
		hasSnapshotExactSourceIDs,
		getSnapshotSourceIDs,
		getDispatchIsFallback,
		getDispatchLastSnapshotVersion,
		getDispatchLastDispatchedVersion,
		parseSnapshotErr := parseHostRegionAdapter.handleHostRegionUpdateSnapshot(parseSpec, parseInputVersion, false)
	if parseSnapshotErr != nil {
		return HostRegionUpdateDispatchResult{}, parseSnapshotErr
	}
	hasDispatchNoChange, parseDispatchHashErr := parseHostRegionAdapter.handleHostRegionDispatchHash(
		getSnapshotEnvelope,
		getSnapshotSourceIDs,
		parseSpec.RendererID,
	)
	if parseDispatchHashErr != nil {
		return HostRegionUpdateDispatchResult{}, parseDispatchHashErr
	}
	if hasDispatchNoChange {
		if parseSnapshotStateErr := parseHostRegionAdapter.storeCoordinator.applyRegionSnapshotState(
			parseHostRegionAdapter.storeRegionInstanceID,
			parseInputVersion,
			getSnapshotSourceIDs,
			!hasSnapshotExactSourceIDs,
		); parseSnapshotStateErr != nil {
			return HostRegionUpdateDispatchResult{}, parseSnapshotStateErr
		}
		parseHostRegionAdapter.storeHostRegionCoordinatorCacheSnapshotState(
			parseInputVersion,
			getSnapshotSourceIDs,
			!hasSnapshotExactSourceIDs,
		)
		return HostRegionUpdateDispatchResult{
			HasScheduled:           false,
			HasNoChange:            true,
			GetDispatchPriority:    getDispatchPriority,
			GetSnapshotEnvelope:    getSnapshotEnvelope,
			GetSnapshotFingerprint: "",
		}, nil
	}
	if getDispatchPriority == HostRegionDispatchPriorityDeferred {
		if parseSnapshotStateErr := parseHostRegionAdapter.storeCoordinator.applyRegionSnapshotState(
			parseHostRegionAdapter.storeRegionInstanceID,
			parseInputVersion,
			getSnapshotSourceIDs,
			!hasSnapshotExactSourceIDs,
		); parseSnapshotStateErr != nil {
			return HostRegionUpdateDispatchResult{}, parseSnapshotStateErr
		}
		parseHostRegionAdapter.storeHostRegionCoordinatorCacheSnapshotState(
			parseInputVersion,
			getSnapshotSourceIDs,
			!hasSnapshotExactSourceIDs,
		)
		if parseHostRegionAdapter.hasHostRegionDeferredDispatch &&
			parseInputVersion <= parseHostRegionAdapter.storeHostRegionDeferredDispatch.getInputVersion {
			return HostRegionUpdateDispatchResult{}, fmt.Errorf(
				"runtime2: deferred input version %d must advance beyond queued version %d",
				parseInputVersion,
				parseHostRegionAdapter.storeHostRegionDeferredDispatch.getInputVersion,
			)
		}
		hasDeferredSuperseded := parseHostRegionAdapter.hasHostRegionDeferredDispatch
		parseHostRegionAdapter.storeHostRegionDeferredDispatch = hostRegionDeferredDispatch{
			getInputVersion:        parseInputVersion,
			getSnapshotEnvelope:    getSnapshotEnvelope,
			getSnapshotFingerprint: "",
		}
		parseHostRegionAdapter.hasHostRegionDeferredDispatch = true
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
	if getDispatchIsFallback {
		return HostRegionUpdateDispatchResult{}, fmt.Errorf("runtime2: host region %q is in fallback mode", parseHostRegionAdapter.storeRegionInstanceID)
	}
	if parseVersionErr := ValidateMonotonicInputVersion(getDispatchLastSnapshotVersion, parseInputVersion); parseVersionErr != nil {
		return HostRegionUpdateDispatchResult{}, parseVersionErr
	}
	if parseVersionErr := ValidateMonotonicInputVersion(getDispatchLastDispatchedVersion, parseInputVersion); parseVersionErr != nil {
		return HostRegionUpdateDispatchResult{}, parseVersionErr
	}
	getSchedulerJob, parseSchedulerErr := parseHostRegionAdapter.storeScheduler.HandleSchedulerUpdate(string(parseHostRegionAdapter.storeRegionInstanceID))
	if parseSchedulerErr != nil {
		return HostRegionUpdateDispatchResult{}, parseSchedulerErr
	}
	if parseCoordinatorStoreErr := parseHostRegionAdapter.storeCoordinator.applyRegionSnapshotDispatchState(
		parseHostRegionAdapter.storeRegionInstanceID,
		parseInputVersion,
		parseInputVersion,
		getSnapshotSourceIDs,
		!hasSnapshotExactSourceIDs,
	); parseCoordinatorStoreErr != nil {
		return HostRegionUpdateDispatchResult{}, parseCoordinatorStoreErr
	}
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheSnapshotDispatchState(
		parseInputVersion,
		parseInputVersion,
		getSnapshotSourceIDs,
		!hasSnapshotExactSourceIDs,
	)
	hasDeferredCanceled := false
	if parseHostRegionAdapter.hasHostRegionDeferredDispatch &&
		parseInputVersion > parseHostRegionAdapter.storeHostRegionDeferredDispatch.getInputVersion {
		parseHostRegionAdapter.storeHostRegionDeferredDispatch = hostRegionDeferredDispatch{}
		parseHostRegionAdapter.hasHostRegionDeferredDispatch = false
		hasDeferredCanceled = true
	}
	if parseHostRegionAdapter.isHostRegionRoundTripTimingEnabled {
		parseHostRegionAdapter.storeHostRegionDispatchAt = time.Now()
		parseHostRegionAdapter.storeHostRegionPatchReadyAt = time.Time{}
		parseHostRegionAdapter.storeHostRegionCommitAt = time.Time{}
		parseHostRegionAdapter.storeHostRegionDispatchToPatchNS = 0
		parseHostRegionAdapter.storeHostRegionDispatchToCommitNS = 0
		parseHostRegionAdapter.storeHostRegionPatchToCommitNS = 0
	}
	parseHostRegionAdapter.storeHostRegionLatestValidVersion = parseHostRegionMaxVersion(
		parseHostRegionAdapter.storeHostRegionLatestValidVersion,
		parseInputVersion,
	)
	return HostRegionUpdateDispatchResult{
		HasScheduled:           true,
		HasNoChange:            false,
		HasDeferredCanceled:    hasDeferredCanceled,
		GetDispatchPriority:    getDispatchPriority,
		GetSchedulerJob:        getSchedulerJob,
		GetSnapshotEnvelope:    getSnapshotEnvelope,
		GetSnapshotFingerprint: "",
	}, nil
}

// GetHostRegionDeferredInputVersion reports the currently queued deferred input version, or zero when no deferred update is queued.
func (parseHostRegionAdapter *HostRegionAdapter) GetHostRegionDeferredInputVersion() uint64 {
	if parseHostRegionAdapter == nil || !parseHostRegionAdapter.hasHostRegionDeferredDispatch {
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
	hasDeferredCleared := parseHostRegionAdapter.hasHostRegionDeferredDispatch
	parseHostRegionAdapter.storeHostRegionDeferredDispatch = hostRegionDeferredDispatch{}
	parseHostRegionAdapter.hasHostRegionDeferredDispatch = false
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
	if parseCommitErr := parseHostRegionAdapter.storeCoordinator.handleCoordinatorCommitRegionTrusted(parseHostRegionAdapter.storeRegionInstanceID, parseInputVersion); parseCommitErr != nil {
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
		if _, parseDropErr := parseHostRegionAdapter.storeCoordinator.handleCoordinatorIncrementRegionDroppedStalePatchCountTrusted(parseHostRegionAdapter.storeRegionInstanceID); parseDropErr != nil {
			return HostRegionWorkerOutputResult{}, parseDropErr
		}
		return HostRegionWorkerOutputResult{
			HasIgnored: true,
		}, nil
	}
	if parseCommitErr := parseHostRegionAdapter.storeCoordinator.handleCoordinatorCommitRegionTrusted(parseHostRegionAdapter.storeRegionInstanceID, parseInputVersion); parseCommitErr != nil {
		return HostRegionWorkerOutputResult{}, parseCommitErr
	}
	if parseHostRegionAdapter.isHostRegionRoundTripTimingEnabled {
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
	hasPatchKeyedMoveOp := parseHasPatchKeyedMoveOp(parsePatch.GetOps)
	var buildKnownNodeIDs map[uint64]struct{}
	var buildSiblingCountByParent map[uint64]uint32
	if hasPatchKeyedMoveOp {
		buildKnownNodeIDs, buildSiblingCountByParent = BuildRegionDOMPatchLookupMaps(parseHostRegionAdapter.storeRegionDOMIndexHandle, buildRegionID)
	} else {
		buildKnownNodeIDs = BuildKnownNodeIDsForRegionDOMIndex(parseHostRegionAdapter.storeRegionDOMIndexHandle, buildRegionID)
	}
	parsePatchResult, hasPatchApply, parsePatchErr := ParsePatchStreamTransactionWithKeyedMoveHint(
		parsePatch,
		buildRegionID,
		getCoordinatorEntry.Epoch,
		buildKnownNodeIDs,
		buildSiblingCountByParent,
		parseHostRegionAdapter.storeHostRegionPatchIdempotency,
		hasPatchKeyedMoveOp,
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
		parseHostRegionAdapter.clearHostRegionCoordinatorCacheState()
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
