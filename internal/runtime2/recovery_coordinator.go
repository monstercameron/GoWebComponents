package runtime2

import (
	"fmt"
	"strings"
)

// RegionFallbackState stores per-region fallback ownership and version metadata.
type RegionFallbackState struct {
	IsLocalOwnership bool
	GetReason        string
	GetEpoch         uint64
	GetInputVersion  uint64
}

// WorkerPatchDecision reports whether worker-originated output should be ignored.
type WorkerPatchDecision struct {
	HasIgnored bool
}

// WorkerDiagnosticDecision reports whether worker diagnostics should be ignored.
type WorkerDiagnosticDecision struct {
	HasIgnored bool
}

// WorkerDeathRecoveryResult reports reassignment or fallback decisions after worker death.
type WorkerDeathRecoveryResult struct {
	HasReassigned      bool
	GetRemountEpoch    uint64
	HasFallbackEntered bool
}

// TransportFailureKind identifies one decode-failure trigger that should enter fallback.
type TransportFailureKind string

const (
	transportFailureKindInvalid TransportFailureKind = ""

	TransportFailureKindMalformedControlPayload   TransportFailureKind = "malformed-control-payload"
	TransportFailureKindMalformedPatchPayload     TransportFailureKind = "malformed-patch-payload"
	TransportFailureKindMalformedSharedMemoryPage TransportFailureKind = "malformed-shared-memory-page"
)

// DOMCommitFailureKind identifies one commit-layer failure trigger that should enter fallback.
type DOMCommitFailureKind string

const (
	domCommitFailureKindInvalid DOMCommitFailureKind = ""

	DOMCommitFailureKindMissingParentAnchor DOMCommitFailureKind = "missing-parent-anchor"
	DOMCommitFailureKindMissingNodeLookup   DOMCommitFailureKind = "missing-node-lookup"
	DOMCommitFailureKindInvalidKeyedMove    DOMCommitFailureKind = "invalid-keyed-move"
)

// WorkerDeathFailureKind identifies one worker-death recovery failure reason.
type WorkerDeathFailureKind string

const (
	workerDeathFailureKindInvalid WorkerDeathFailureKind = ""

	WorkerDeathFailureKindRepairFailure WorkerDeathFailureKind = "worker-repair-failure"
	WorkerDeathFailureKindNoReassign    WorkerDeathFailureKind = "worker-death-no-reassign"
)

// RecoveryCoordinator tracks region fallback ownership and worker-output suppression rules.
type RecoveryCoordinator struct {
	storeFallbackByRegionID     map[string]RegionFallbackState
	storeLocalVersionByRegionID map[string]uint64
}

// BuildRecoveryCoordinator creates an empty recovery coordinator.
func BuildRecoveryCoordinator() *RecoveryCoordinator {
	return &RecoveryCoordinator{
		storeFallbackByRegionID:     make(map[string]RegionFallbackState),
		storeLocalVersionByRegionID: make(map[string]uint64),
	}
}

// EnterRegionLocalFallback marks one region as locally owned and records fallback metadata.
func (parseRecoveryCoordinator *RecoveryCoordinator) EnterRegionLocalFallback(parseRegionID string, parseReason string, parseEpoch uint64, parseInputVersion uint64) {
	if parseRecoveryCoordinator == nil {
		return
	}
	if strings.TrimSpace(parseRegionID) == "" {
		return
	}
	parseRecoveryCoordinator.storeFallbackByRegionID[parseRegionID] = RegionFallbackState{
		IsLocalOwnership: true,
		GetReason:        parseReason,
		GetEpoch:         parseEpoch,
		GetInputVersion:  parseInputVersion,
	}
	if parseInputVersion > parseRecoveryCoordinator.storeLocalVersionByRegionID[parseRegionID] {
		parseRecoveryCoordinator.storeLocalVersionByRegionID[parseRegionID] = parseInputVersion
	}
}

// IsRegionLocalOwnership reports whether one region is currently marked as locally owned.
func (parseRecoveryCoordinator *RecoveryCoordinator) IsRegionLocalOwnership(parseRegionID string) bool {
	if parseRecoveryCoordinator == nil {
		return false
	}
	parseFallbackState, hasFallbackState := parseRecoveryCoordinator.storeFallbackByRegionID[parseRegionID]
	return hasFallbackState && parseFallbackState.IsLocalOwnership
}

// HandleWorkerPatch decides whether to ignore one worker patch based on region fallback ownership.
func (parseRecoveryCoordinator *RecoveryCoordinator) HandleWorkerPatch(parseRegionID string, parseEpoch uint64, parseInputVersion uint64) WorkerPatchDecision {
	if parseRecoveryCoordinator == nil {
		return WorkerPatchDecision{}
	}
	if parseRecoveryCoordinator.IsRegionLocalOwnership(parseRegionID) {
		return WorkerPatchDecision{
			HasIgnored: true,
		}
	}
	return WorkerPatchDecision{}
}

// HandleWorkerDiagnostic decides whether to ignore worker diagnostics for locally owned fallback regions.
func (parseRecoveryCoordinator *RecoveryCoordinator) HandleWorkerDiagnostic(parseRegionID string, parseEpoch uint64, parseInputVersion uint64) WorkerDiagnosticDecision {
	if parseRecoveryCoordinator == nil {
		return WorkerDiagnosticDecision{}
	}
	if parseRecoveryCoordinator.IsRegionLocalOwnership(parseRegionID) {
		return WorkerDiagnosticDecision{
			HasIgnored: true,
		}
	}
	return WorkerDiagnosticDecision{}
}

// HandleTransportDecodeFailure enters region-local fallback for supported transport decode failures.
func (parseRecoveryCoordinator *RecoveryCoordinator) HandleTransportDecodeFailure(parseRegionID string, parseFailureKind TransportFailureKind, parseEpoch uint64, parseInputVersion uint64) error {
	if parseRecoveryCoordinator == nil {
		return fmt.Errorf("runtime2: recovery coordinator is nil")
	}
	switch parseFailureKind {
	case TransportFailureKindMalformedControlPayload, TransportFailureKindMalformedPatchPayload, TransportFailureKindMalformedSharedMemoryPage:
		parseRecoveryCoordinator.EnterRegionLocalFallback(parseRegionID, string(parseFailureKind), parseEpoch, parseInputVersion)
		return nil
	case transportFailureKindInvalid:
		return fmt.Errorf("runtime2: transport failure kind is required")
	default:
		return fmt.Errorf("runtime2: transport failure kind %q is unsupported", parseFailureKind)
	}
}

// HandleDOMCommitFailure enters region-local fallback for supported invalid DOM commit states.
func (parseRecoveryCoordinator *RecoveryCoordinator) HandleDOMCommitFailure(parseRegionID string, parseFailureKind DOMCommitFailureKind, parseEpoch uint64, parseInputVersion uint64) error {
	if parseRecoveryCoordinator == nil {
		return fmt.Errorf("runtime2: recovery coordinator is nil")
	}
	switch parseFailureKind {
	case DOMCommitFailureKindMissingParentAnchor, DOMCommitFailureKindMissingNodeLookup, DOMCommitFailureKindInvalidKeyedMove:
		parseRecoveryCoordinator.EnterRegionLocalFallback(parseRegionID, string(parseFailureKind), parseEpoch, parseInputVersion)
		return nil
	case domCommitFailureKindInvalid:
		return fmt.Errorf("runtime2: dom commit failure kind is required")
	default:
		return fmt.Errorf("runtime2: dom commit failure kind %q is unsupported", parseFailureKind)
	}
}

// HandleWorkerDeath applies reassignment-or-fallback policy when a worker dies.
func (parseRecoveryCoordinator *RecoveryCoordinator) HandleWorkerDeath(parseRegionID string, hasReassignSupport bool, parseRemountEpoch uint64, parseInputVersion uint64) (WorkerDeathRecoveryResult, error) {
	if parseRecoveryCoordinator == nil {
		return WorkerDeathRecoveryResult{}, fmt.Errorf("runtime2: recovery coordinator is nil")
	}
	if strings.TrimSpace(parseRegionID) == "" {
		return WorkerDeathRecoveryResult{}, fmt.Errorf("runtime2: region ID is required")
	}
	if hasReassignSupport {
		if parseRemountEpoch == 0 {
			parseRecoveryCoordinator.EnterRegionLocalFallback(parseRegionID, string(WorkerDeathFailureKindRepairFailure), 0, parseInputVersion)
			return WorkerDeathRecoveryResult{
				HasFallbackEntered: true,
			}, fmt.Errorf("runtime2: worker reassignment for region %q requires a fresh remount epoch", parseRegionID)
		}
		delete(parseRecoveryCoordinator.storeFallbackByRegionID, parseRegionID)
		return WorkerDeathRecoveryResult{
			HasReassigned:   true,
			GetRemountEpoch: parseRemountEpoch,
		}, nil
	}
	parseRecoveryCoordinator.EnterRegionLocalFallback(parseRegionID, string(WorkerDeathFailureKindNoReassign), 0, parseInputVersion)
	return WorkerDeathRecoveryResult{
		HasFallbackEntered: true,
	}, nil
}

// SetRegionLocalVersion records locally authoritative state progress for one region.
func (parseRecoveryCoordinator *RecoveryCoordinator) SetRegionLocalVersion(parseRegionID string, parseInputVersion uint64) {
	if parseRecoveryCoordinator == nil {
		return
	}
	if strings.TrimSpace(parseRegionID) == "" {
		return
	}
	if parseInputVersion > parseRecoveryCoordinator.storeLocalVersionByRegionID[parseRegionID] {
		parseRecoveryCoordinator.storeLocalVersionByRegionID[parseRegionID] = parseInputVersion
	}
}

// GetRegionLocalVersion reports the locally authoritative state version for one region.
func (parseRecoveryCoordinator *RecoveryCoordinator) GetRegionLocalVersion(parseRegionID string) uint64 {
	if parseRecoveryCoordinator == nil {
		return 0
	}
	return parseRecoveryCoordinator.storeLocalVersionByRegionID[parseRegionID]
}

// ClearRegionLocalFallback clears one region's local-fallback ownership marker.
func (parseRecoveryCoordinator *RecoveryCoordinator) ClearRegionLocalFallback(parseRegionID string) bool {
	if parseRecoveryCoordinator == nil {
		return false
	}
	if strings.TrimSpace(parseRegionID) == "" {
		return false
	}
	if !parseRecoveryCoordinator.IsRegionLocalOwnership(parseRegionID) {
		return false
	}
	delete(parseRecoveryCoordinator.storeFallbackByRegionID, parseRegionID)
	return true
}
