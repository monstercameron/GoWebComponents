package services_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
	"github.com/monstercameron/GoWebComponents/v4/internal/services"
)

// v5 P3.3 — the extracted arbiter must decide ownership identically to the
// RecoveryCoordinator it came from, over the domain where both are defined.
//
// The shared domain is ownership itself: who is authoritative, whether remote
// output is admitted, and the reassign-or-fall-back policy after a death. The
// ordering guards (epoch floor, local-version check) are the documented
// difference and are covered by ownership_test.go instead — runtime2 delegates
// those to its patch parse layer, which a generic arbiter does not have.
//
// Epochs increase strictly throughout, which keeps the sequences inside the
// shared domain: runtime2 will reassign onto a reused epoch and the arbiter
// refuses to, deliberately.

// assertOwnershipAgrees compares both implementations' view of one unit.
func assertOwnershipAgrees(parseT *testing.T, parseLabel string, parseCoordinator *runtime2.RecoveryCoordinator, parseArbiter *services.Arbiter, parseUnitID string) {
	parseT.Helper()

	parseCoordinatorLocal := parseCoordinator.IsRegionLocalOwnership(parseUnitID)
	parseArbiterLocal := parseArbiter.Owner(parseUnitID) == services.OwnerLocal
	if parseCoordinatorLocal != parseArbiterLocal {
		parseT.Fatalf("%s: runtime2 localOwnership=%v but services owner=%s — the extracted policy diverged from its source",
			parseLabel, parseCoordinatorLocal, parseArbiter.Owner(parseUnitID))
	}

	// Worker-output suppression must follow ownership in lockstep. Versions are
	// held at the high-water mark so only the ownership rule is under test.
	parseCoordinatorIgnored := parseCoordinator.HandleWorkerPatch(parseUnitID, 1, 1).HasIgnored
	parseArbiterIgnored := !parseArbiter.AdmitRemote(parseUnitID, ^uint64(0), ^uint64(0)).Accepted()
	if parseCoordinatorIgnored != parseArbiterIgnored {
		parseT.Fatalf("%s: runtime2 ignored=%v but services ignored=%v",
			parseLabel, parseCoordinatorIgnored, parseArbiterIgnored)
	}
}

// TestArbiterMatchesRuntime2OnFailureEntry pins that every failure trigger the
// source treats as fallback-entering is treated the same way here.
func TestArbiterMatchesRuntime2OnFailureEntry(parseT *testing.T) {
	for _, parseCase := range []struct {
		label   string
		enter   func(*runtime2.RecoveryCoordinator, string) error
		class   services.FailureClass
		unitID  string
		wantErr bool
	}{
		{
			label: "transport decode failure",
			enter: func(parseCoordinator *runtime2.RecoveryCoordinator, parseUnitID string) error {
				return parseCoordinator.HandleTransportDecodeFailure(parseUnitID,
					runtime2.TransportFailureKindMalformedPatchPayload, 1, 4)
			},
			class:  services.FailureTransport,
			unitID: "unit-transport",
		},
		{
			label: "shared memory page failure",
			enter: func(parseCoordinator *runtime2.RecoveryCoordinator, parseUnitID string) error {
				return parseCoordinator.HandleTransportDecodeFailure(parseUnitID,
					runtime2.TransportFailureKindMalformedSharedMemoryPage, 1, 4)
			},
			class:  services.FailureTransport,
			unitID: "unit-shared",
		},
		{
			label: "dom commit failure",
			enter: func(parseCoordinator *runtime2.RecoveryCoordinator, parseUnitID string) error {
				return parseCoordinator.HandleDOMCommitFailure(parseUnitID,
					runtime2.DOMCommitFailureKindMissingParentAnchor, 1, 4)
			},
			class:  services.FailureCommit,
			unitID: "unit-commit",
		},
		{
			label: "invalid keyed move",
			enter: func(parseCoordinator *runtime2.RecoveryCoordinator, parseUnitID string) error {
				return parseCoordinator.HandleDOMCommitFailure(parseUnitID,
					runtime2.DOMCommitFailureKindInvalidKeyedMove, 1, 4)
			},
			class:  services.FailureCommit,
			unitID: "unit-keyed",
		},
	} {
		parseCoordinator := runtime2.BuildRecoveryCoordinator()
		parseArbiter := services.NewArbiter()

		assertOwnershipAgrees(parseT, parseCase.label+" (before)", parseCoordinator, parseArbiter, parseCase.unitID)

		if parseErr := parseCase.enter(parseCoordinator, parseCase.unitID); parseErr != nil {
			parseT.Fatalf("%s: runtime2 entry: %v", parseCase.label, parseErr)
		}
		if parseErr := parseArbiter.Fallback(parseCase.unitID, parseCase.class, parseCase.label, 1, 4); parseErr != nil {
			parseT.Fatalf("%s: services entry: %v", parseCase.label, parseErr)
		}

		assertOwnershipAgrees(parseT, parseCase.label+" (after)", parseCoordinator, parseArbiter, parseCase.unitID)
	}
}

// TestArbiterMatchesRuntime2OnUnknownFailureKinds: both must refuse a failure
// kind they do not understand, and neither may transfer ownership on refusal.
func TestArbiterMatchesRuntime2OnUnknownFailureKinds(parseT *testing.T) {
	parseCoordinator := runtime2.BuildRecoveryCoordinator()
	parseArbiter := services.NewArbiter()

	if parseErr := parseCoordinator.HandleTransportDecodeFailure("unit", runtime2.TransportFailureKind("invented"), 1, 1); parseErr == nil {
		parseT.Error("runtime2 accepted an unknown transport failure kind")
	}
	if parseErr := parseArbiter.Fallback("unit", services.FailureClass("invented"), "x", 1, 1); parseErr == nil {
		parseT.Error("services accepted an unknown failure class")
	}
	assertOwnershipAgrees(parseT, "after refused entry", parseCoordinator, parseArbiter, "unit")

	// The empty kind is required-field territory in both.
	if parseErr := parseCoordinator.HandleTransportDecodeFailure("unit", runtime2.TransportFailureKind(""), 1, 1); parseErr == nil {
		parseT.Error("runtime2 accepted an empty transport failure kind")
	}
	if parseErr := parseArbiter.Fallback("unit", services.FailureClass(""), "x", 1, 1); parseErr == nil {
		parseT.Error("services accepted an empty failure class")
	}
}

// TestArbiterMatchesRuntime2OnWorkerDeathPolicy walks the reassign-or-fall-back
// policy over strictly increasing epochs.
func TestArbiterMatchesRuntime2OnWorkerDeathPolicy(parseT *testing.T) {
	for _, parseCase := range []struct {
		label            string
		hasReassign      bool
		reassignEpoch    uint64
		wantReassigned   bool
		wantFellBack     bool
		wantErrorFromAPI bool
	}{
		{"reassignable", true, 2, true, false, false},
		{"no reassignment support", false, 0, false, true, false},
		{"claims support, supplies no epoch", true, 0, false, true, true},
	} {
		parseCoordinator := runtime2.BuildRecoveryCoordinator()
		parseArbiter := services.NewArbiter()
		const parseUnitID = "unit-death"

		parseCoordinatorResult, parseCoordinatorErr := parseCoordinator.HandleWorkerDeath(
			parseUnitID, parseCase.hasReassign, parseCase.reassignEpoch, 3)
		parseArbiterOutcome, parseArbiterErr := parseArbiter.HandleRemoteDeath(
			parseUnitID, parseCase.hasReassign, parseCase.reassignEpoch, 3)

		if (parseCoordinatorErr == nil) != (parseArbiterErr == nil) {
			parseT.Fatalf("%s: runtime2 err=%v but services err=%v", parseCase.label, parseCoordinatorErr, parseArbiterErr)
		}
		if (parseCoordinatorErr != nil) != parseCase.wantErrorFromAPI {
			parseT.Fatalf("%s: error=%v, want error=%v", parseCase.label, parseCoordinatorErr, parseCase.wantErrorFromAPI)
		}
		if parseCoordinatorResult.HasReassigned != parseArbiterOutcome.Reassigned {
			parseT.Fatalf("%s: runtime2 reassigned=%v but services reassigned=%v",
				parseCase.label, parseCoordinatorResult.HasReassigned, parseArbiterOutcome.Reassigned)
		}
		if parseCoordinatorResult.HasFallbackEntered != parseArbiterOutcome.FellBack {
			parseT.Fatalf("%s: runtime2 fellBack=%v but services fellBack=%v",
				parseCase.label, parseCoordinatorResult.HasFallbackEntered, parseArbiterOutcome.FellBack)
		}
		if parseArbiterOutcome.Reassigned != parseCase.wantReassigned || parseArbiterOutcome.FellBack != parseCase.wantFellBack {
			parseT.Fatalf("%s: outcome = %+v, want reassigned=%v fellBack=%v",
				parseCase.label, parseArbiterOutcome, parseCase.wantReassigned, parseCase.wantFellBack)
		}

		assertOwnershipAgrees(parseT, parseCase.label, parseCoordinator, parseArbiter, parseUnitID)
	}
}

// TestArbiterMatchesRuntime2OverALifecycle runs several fall-back / hand-back
// cycles and compares ownership after every transition, which is where a subtle
// divergence would show rather than in any single call.
func TestArbiterMatchesRuntime2OverALifecycle(parseT *testing.T) {
	parseCoordinator := runtime2.BuildRecoveryCoordinator()
	parseArbiter := services.NewArbiter()
	const parseUnitID = "unit-lifecycle"

	parseEpoch := uint64(1)
	for parseCycle := range 5 {
		parseEpoch++
		if parseErr := parseCoordinator.HandleTransportDecodeFailure(parseUnitID,
			runtime2.TransportFailureKindMalformedControlPayload, parseEpoch, uint64(parseCycle)); parseErr != nil {
			parseT.Fatalf("cycle %d: runtime2 fallback: %v", parseCycle, parseErr)
		}
		if parseErr := parseArbiter.Fallback(parseUnitID, services.FailureTransport,
			"malformed control payload", parseEpoch, uint64(parseCycle)); parseErr != nil {
			parseT.Fatalf("cycle %d: services fallback: %v", parseCycle, parseErr)
		}
		assertOwnershipAgrees(parseT, fmt.Sprintf("cycle %d after fallback", parseCycle), parseCoordinator, parseArbiter, parseUnitID)

		// A repeated failure while already locally owned must be a no-op for
		// ownership in both.
		if parseErr := parseCoordinator.HandleDOMCommitFailure(parseUnitID,
			runtime2.DOMCommitFailureKindMissingNodeLookup, parseEpoch, uint64(parseCycle)); parseErr != nil {
			parseT.Fatalf("cycle %d: runtime2 repeat failure: %v", parseCycle, parseErr)
		}
		if parseErr := parseArbiter.Fallback(parseUnitID, services.FailureCommit,
			"missing node lookup", parseEpoch, uint64(parseCycle)); parseErr != nil {
			parseT.Fatalf("cycle %d: services repeat failure: %v", parseCycle, parseErr)
		}
		assertOwnershipAgrees(parseT, fmt.Sprintf("cycle %d after repeat", parseCycle), parseCoordinator, parseArbiter, parseUnitID)

		parseEpoch++
		if !parseCoordinator.ClearRegionLocalFallback(parseUnitID) {
			parseT.Fatalf("cycle %d: runtime2 clear reported no change", parseCycle)
		}
		if parseErr := parseArbiter.Restore(parseUnitID, parseEpoch); parseErr != nil {
			parseT.Fatalf("cycle %d: services restore: %v", parseCycle, parseErr)
		}
		assertOwnershipAgrees(parseT, fmt.Sprintf("cycle %d after restore", parseCycle), parseCoordinator, parseArbiter, parseUnitID)
	}
}
