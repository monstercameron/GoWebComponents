package services

import (
	"fmt"
	"strings"
)

// Ownership arbitration — extracted from runtime2's RecoveryCoordinator (P3.3).
//
// The problem it solves is not DOM-specific at all: a unit of work normally runs
// remotely, the remote path fails, and the local side takes over. The subtle
// part is what happens NEXT. A worker that dies, restarts, and resumes will ship
// output computed from pre-failure state, and accepting it would overwrite
// locally-computed authoritative state with something older. So a failure is not
// just an error to report — it transfers ownership, and ownership decides whose
// output counts until it is explicitly handed back.
//
// P3.8a's command failure model needs exactly this, and so does any service with
// a local fallback path. What kept it from being reusable is that it spoke in
// regions and patches.
//
// ORDERING, and why this is not a straight port: runtime2's coordinator accepts
// epoch and version arguments on its worker-output handlers and never reads
// them. That is safe there because the patch parse layer rejects a stale epoch
// before the coordinator is reached — the same arrangement the idempotency
// ledger relies on. A payload-agnostic arbiter has no layer above it, so it
// carries the ordering guard itself and those arguments become load-bearing.
// See TestArbiterRejectsStaleRemoteOutputAfterRestore.

// Owner names which side is authoritative for a unit of work.
type Owner uint8

const (
	// OwnerRemote is the normal state: the worker owns the unit.
	OwnerRemote Owner = iota
	// OwnerLocal means a failure moved authority to the local side. Remote
	// output is ignored until ownership is explicitly restored.
	OwnerLocal
)

func (parseOwner Owner) String() string {
	if parseOwner == OwnerLocal {
		return "local"
	}
	return "remote"
}

// FailureClass classifies why ownership moved.
//
// The class is kept separate from the free-text reason because recovery policy
// branches on it: a transport failure may be retryable where a panic is not.
// Callers define their own reasons; the classes are the closed set the arbiter
// understands.
type FailureClass string

const (
	// FailureTransport is a payload that could not be decoded or delivered.
	FailureTransport FailureClass = "transport"
	// FailureCommit is a failure applying an otherwise valid payload.
	FailureCommit FailureClass = "commit"
	// FailureWorkerDeath is the remote side terminating.
	FailureWorkerDeath FailureClass = "worker-death"
	// FailureTimeout is a remote side that did not answer in time. It is
	// distinguished from death because the worker may still be alive and may
	// still deliver — which is precisely the stale-output hazard.
	FailureTimeout FailureClass = "timeout"
	// FailurePanic is an unrecoverable fault inside the remote handler.
	FailurePanic FailureClass = "panic"
)

// IsKnownFailureClass reports whether a class is one the arbiter understands.
//
// Unknown classes are rejected rather than absorbed: a typo'd class that
// silently entered fallback would look identical to a real failure in
// diagnostics, and the unit would never be handed back.
func IsKnownFailureClass(parseClass FailureClass) bool {
	switch parseClass {
	case FailureTransport, FailureCommit, FailureWorkerDeath, FailureTimeout, FailurePanic:
		return true
	default:
		return false
	}
}

// FallbackState is one unit's ownership record.
type FallbackState struct {
	Owner Owner
	Class FailureClass
	// Reason is the caller's free-text detail, for diagnostics only.
	Reason string
	// Epoch is the remote epoch in effect when ownership moved.
	Epoch uint64
	// Version is the highest locally authoritative version at that moment.
	Version uint64
}

// RemoteVerdict is what an arbiter says about arriving remote output.
type RemoteVerdict uint8

const (
	// VerdictAccept means the output is current and authoritative.
	VerdictAccept RemoteVerdict = iota
	// VerdictRejectLocalOwned means the local side currently owns the unit.
	VerdictRejectLocalOwned
	// VerdictRejectStaleEpoch means the output predates the current remote
	// assignment — typically a worker that died and was replaced.
	VerdictRejectStaleEpoch
	// VerdictRejectStaleVersion means the output is behind what the local side
	// already computed authoritatively while it owned the unit.
	VerdictRejectStaleVersion
)

// Accepted reports whether a verdict means "use this output".
//
// Callers should branch on this rather than comparing against VerdictAccept, so
// a future rejection reason cannot silently become an acceptance.
func (parseVerdict RemoteVerdict) Accepted() bool {
	return parseVerdict == VerdictAccept
}

func (parseVerdict RemoteVerdict) String() string {
	switch parseVerdict {
	case VerdictAccept:
		return "accept"
	case VerdictRejectLocalOwned:
		return "reject-local-owned"
	case VerdictRejectStaleEpoch:
		return "reject-stale-epoch"
	case VerdictRejectStaleVersion:
		return "reject-stale-version"
	default:
		return fmt.Sprintf("unknown-verdict(%d)", uint8(parseVerdict))
	}
}

// unitOwnershipState is one unit's arbitration state.
type unitOwnershipState struct {
	fallback FallbackState
	// localVersion is the highest version the local side computed authoritatively.
	localVersion uint64
	// remoteEpochFloor is the lowest remote epoch whose output is still current.
	// A restore raises it, which is what invalidates a dead worker's in-flight
	// output without needing to know anything about that output.
	remoteEpochFloor uint64
}

// Arbiter tracks ownership and remote-output admissibility across named units.
//
// Not safe for concurrent use; it belongs to one coordinating thread.
type Arbiter struct {
	stateByUnitID map[string]unitOwnershipState
}

// NewArbiter creates an arbiter with every unit remotely owned by default.
func NewArbiter() *Arbiter {
	return &Arbiter{stateByUnitID: make(map[string]unitOwnershipState)}
}

// Fallback moves a unit to local ownership.
//
// Idempotent for repeated failures on an already-local unit: the class and
// reason update to the most recent cause, and the version high-water mark only
// ever rises. Losing the original cause is the right trade — the most recent
// failure is what a caller diagnosing a stuck unit needs.
func (parseArbiter *Arbiter) Fallback(parseUnitID string, parseClass FailureClass, parseReason string, parseEpoch uint64, parseVersion uint64) error {
	if parseArbiter == nil {
		return fmt.Errorf("services: arbiter is nil")
	}
	if strings.TrimSpace(parseUnitID) == "" {
		return fmt.Errorf("services: arbiter unit id is required")
	}
	if !IsKnownFailureClass(parseClass) {
		return fmt.Errorf("services: unknown failure class %q", parseClass)
	}

	getState := parseArbiter.stateByUnitID[parseUnitID]
	if parseVersion > getState.localVersion {
		getState.localVersion = parseVersion
	}
	getState.fallback = FallbackState{
		Owner:   OwnerLocal,
		Class:   parseClass,
		Reason:  parseReason,
		Epoch:   parseEpoch,
		Version: getState.localVersion,
	}
	parseArbiter.stateByUnitID[parseUnitID] = getState
	return nil
}

// Owner reports which side is authoritative for a unit. Unknown units are
// remotely owned: nothing has failed.
func (parseArbiter *Arbiter) Owner(parseUnitID string) Owner {
	if parseArbiter == nil {
		return OwnerRemote
	}
	return parseArbiter.stateByUnitID[parseUnitID].fallback.Owner
}

// State reports a unit's full ownership record, and whether one exists.
func (parseArbiter *Arbiter) State(parseUnitID string) (FallbackState, bool) {
	if parseArbiter == nil {
		return FallbackState{}, false
	}
	getState, hasState := parseArbiter.stateByUnitID[parseUnitID]
	if !hasState {
		return FallbackState{}, false
	}
	return getState.fallback, true
}

// AdmitRemote decides whether arriving remote output should be used.
//
// This is the call that must not be simplified to an ownership check. Ownership
// covers the during-fallback case; the epoch floor covers output that was
// in flight when a worker was replaced; the version check covers output computed
// before the local side advanced the unit. All three are the same hazard —
// older state overwriting newer — arriving by different routes.
func (parseArbiter *Arbiter) AdmitRemote(parseUnitID string, parseEpoch uint64, parseVersion uint64) RemoteVerdict {
	if parseArbiter == nil {
		return VerdictAccept
	}
	getState, hasState := parseArbiter.stateByUnitID[parseUnitID]
	if !hasState {
		return VerdictAccept
	}
	if getState.fallback.Owner == OwnerLocal {
		return VerdictRejectLocalOwned
	}
	if parseEpoch < getState.remoteEpochFloor {
		return VerdictRejectStaleEpoch
	}
	if parseVersion < getState.localVersion {
		return VerdictRejectStaleVersion
	}
	return VerdictAccept
}

// RecordLocalProgress records that the local side computed a unit up to a
// version while owning it. The high-water mark only rises.
func (parseArbiter *Arbiter) RecordLocalProgress(parseUnitID string, parseVersion uint64) {
	if parseArbiter == nil || strings.TrimSpace(parseUnitID) == "" {
		return
	}
	getState := parseArbiter.stateByUnitID[parseUnitID]
	if parseVersion > getState.localVersion {
		getState.localVersion = parseVersion
		parseArbiter.stateByUnitID[parseUnitID] = getState
	}
}

// LocalVersion reports the highest locally authoritative version for a unit.
func (parseArbiter *Arbiter) LocalVersion(parseUnitID string) uint64 {
	if parseArbiter == nil {
		return 0
	}
	return parseArbiter.stateByUnitID[parseUnitID].localVersion
}

// Restore hands a unit back to remote ownership at a fresh epoch.
//
// The epoch must be strictly newer than anything seen, and that requirement is
// the whole mechanism: raising the floor invalidates the previous worker's
// in-flight output without the arbiter needing to know anything about it.
// Restoring at a reused epoch would readmit exactly the output the fallback
// existed to exclude, so it is refused rather than accepted with a warning.
func (parseArbiter *Arbiter) Restore(parseUnitID string, parseEpoch uint64) error {
	if parseArbiter == nil {
		return fmt.Errorf("services: arbiter is nil")
	}
	if strings.TrimSpace(parseUnitID) == "" {
		return fmt.Errorf("services: arbiter unit id is required")
	}
	if parseEpoch == 0 {
		return fmt.Errorf("services: restoring unit %q requires a fresh non-zero epoch", parseUnitID)
	}

	getState, hasState := parseArbiter.stateByUnitID[parseUnitID]
	if hasState {
		if parseEpoch <= getState.fallback.Epoch || parseEpoch < getState.remoteEpochFloor {
			return fmt.Errorf(
				"services: restoring unit %q requires an epoch newer than %d, got %d",
				parseUnitID, maxUint64(getState.fallback.Epoch, getState.remoteEpochFloor), parseEpoch)
		}
	}

	getState.remoteEpochFloor = parseEpoch
	getState.fallback = FallbackState{
		Owner: OwnerRemote,
		Epoch: parseEpoch,
	}
	parseArbiter.stateByUnitID[parseUnitID] = getState
	return nil
}

// DeathOutcome reports what happened to a unit when its remote side died.
type DeathOutcome struct {
	// Reassigned means the unit is remotely owned again at RestoredEpoch.
	Reassigned    bool
	RestoredEpoch uint64
	// FellBack means the unit moved to local ownership instead.
	FellBack bool
}

// HandleRemoteDeath applies the reassign-or-fall-back policy for a dead remote.
//
// The two "no reassignment happened" cases stay distinct rather than collapsing
// into one, because they mean opposite things about the system. A caller that
// does not support reassignment falling back is the design working; a caller
// that claims support and then supplies no epoch is a bug, and it still falls
// back — losing availability quietly would be worse — but it also returns an
// error so the bug is visible rather than absorbed as a normal recovery.
func (parseArbiter *Arbiter) HandleRemoteDeath(parseUnitID string, hasReassignSupport bool, parseReassignEpoch uint64, parseVersion uint64) (DeathOutcome, error) {
	if parseArbiter == nil {
		return DeathOutcome{}, fmt.Errorf("services: arbiter is nil")
	}
	if strings.TrimSpace(parseUnitID) == "" {
		return DeathOutcome{}, fmt.Errorf("services: arbiter unit id is required")
	}

	if hasReassignSupport {
		if parseReassignEpoch == 0 {
			_ = parseArbiter.Fallback(parseUnitID, FailureWorkerDeath, "reassignment supplied no epoch", 0, parseVersion)
			return DeathOutcome{FellBack: true}, fmt.Errorf(
				"services: reassigning unit %q requires a fresh remount epoch", parseUnitID)
		}
		if parseRestoreErr := parseArbiter.Restore(parseUnitID, parseReassignEpoch); parseRestoreErr != nil {
			// A reused epoch would readmit the dead worker's in-flight output, so
			// fall back rather than reassign onto it.
			_ = parseArbiter.Fallback(parseUnitID, FailureWorkerDeath, parseRestoreErr.Error(), parseReassignEpoch, parseVersion)
			return DeathOutcome{FellBack: true}, parseRestoreErr
		}
		return DeathOutcome{Reassigned: true, RestoredEpoch: parseReassignEpoch}, nil
	}

	if parseFallbackErr := parseArbiter.Fallback(parseUnitID, FailureWorkerDeath, "no reassignment available", 0, parseVersion); parseFallbackErr != nil {
		return DeathOutcome{}, parseFallbackErr
	}
	return DeathOutcome{FellBack: true}, nil
}

// Forget drops a unit's arbitration state entirely, for units that genuinely
// end. It clears the local version and epoch floor along with ownership, so a
// unit that later resumes would readmit stale output — use Restore to hand a
// live unit back.
func (parseArbiter *Arbiter) Forget(parseUnitID string) {
	if parseArbiter == nil {
		return
	}
	delete(parseArbiter.stateByUnitID, parseUnitID)
}

// LocalOwnedUnits lists every unit currently in local fallback.
//
// The source had no way to enumerate these, which made "why is the app slow"
// unanswerable when units had quietly fallen back one at a time. Order is
// unspecified; callers that display it should sort.
func (parseArbiter *Arbiter) LocalOwnedUnits() []string {
	if parseArbiter == nil {
		return nil
	}
	var parseUnits []string
	for parseUnitID, parseState := range parseArbiter.stateByUnitID {
		if parseState.fallback.Owner == OwnerLocal {
			parseUnits = append(parseUnits, parseUnitID)
		}
	}
	return parseUnits
}

func maxUint64(parseLeft uint64, parseRight uint64) uint64 {
	if parseLeft > parseRight {
		return parseLeft
	}
	return parseRight
}
