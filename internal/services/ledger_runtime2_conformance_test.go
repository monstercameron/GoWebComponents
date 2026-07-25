package services_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
	"github.com/monstercameron/GoWebComponents/v4/internal/services"
)

// v5 P3.3 — the extracted ledger must decide identically to the tracker it came
// from, over the domain where both are defined.
//
// This is the same discipline as transport_runtime2_conformance_test.go and for
// the same reason: an extraction that merely LOOKS like its source is a fork
// waiting to happen. Divergence here is the quiet kind — both implementations
// work, they simply disagree, and a stream deduplicated by one is replayed by
// the other.
//
// The shared domain is: one stream, one epoch, versions inside the retention
// window. The two documented differences (epoch ordering, bounded retention)
// are deliberately outside it and are covered by ledger_test.go instead.

// runDifferentialSequence drives one sequence of (version, identity) arrivals
// through both implementations and requires them to agree at every step.
func runDifferentialSequence(parseT *testing.T, parseLabel string, parseArrivals [][2]interface{}) {
	parseT.Helper()

	parseTracker := runtime2.BuildPatchIdempotencyTracker()
	parseLedger := services.NewLedger()

	const parseRegionID = "differential-region"
	const parseEpoch = uint64(1)

	for parseStep, parseArrival := range parseArrivals {
		parseVersion := parseArrival[0].(uint64)
		parseIdentity := parseArrival[1].(string)

		parseTrackerApply, parseTrackerErr := parseTracker.HandlePatchIdempotency(
			parseRegionID, parseEpoch, parseVersion, parseIdentity)
		parseLedgerDecision, parseLedgerErr := parseLedger.Apply(
			parseRegionID, parseEpoch, parseVersion, parseIdentity)

		if (parseTrackerErr == nil) != (parseLedgerErr == nil) {
			parseT.Fatalf("%s step %d (v=%d id=%q): runtime2 err=%v but services err=%v",
				parseLabel, parseStep, parseVersion, parseIdentity, parseTrackerErr, parseLedgerErr)
		}
		if parseTrackerApply != parseLedgerDecision.ShouldApply() {
			parseT.Fatalf("%s step %d (v=%d id=%q): runtime2 apply=%v but services said %s — the extracted policy diverged from its source",
				parseLabel, parseStep, parseVersion, parseIdentity, parseTrackerApply, parseLedgerDecision)
		}
	}
}

func buildArrival(parseVersion uint64, parseIdentity string) [2]interface{} {
	return [2]interface{}{parseVersion, parseIdentity}
}

// TestLedgerMatchesRuntime2OnScriptedSequences covers the named cases by hand,
// so a failure names the situation rather than a step index in noise.
func TestLedgerMatchesRuntime2OnScriptedSequences(parseT *testing.T) {
	for _, parseCase := range []struct {
		label    string
		arrivals [][2]interface{}
	}{
		{
			label: "strictly in order",
			arrivals: [][2]interface{}{
				buildArrival(1, "a"), buildArrival(2, "b"), buildArrival(3, "c"),
			},
		},
		{
			label: "exact duplicates interleaved",
			arrivals: [][2]interface{}{
				buildArrival(1, "a"), buildArrival(1, "a"), buildArrival(2, "b"), buildArrival(1, "a"), buildArrival(2, "b"),
			},
		},
		{
			label: "reordered deliveries below the high-water mark",
			arrivals: [][2]interface{}{
				buildArrival(10, "j"), buildArrival(4, "d"), buildArrival(9, "i"), buildArrival(11, "k"), buildArrival(1, "a"),
			},
		},
		{
			label: "conflict then continue",
			arrivals: [][2]interface{}{
				buildArrival(1, "a"), buildArrival(1, "different"), buildArrival(2, "b"),
			},
		},
		{
			label: "gap in versions",
			arrivals: [][2]interface{}{
				buildArrival(1, "a"), buildArrival(50, "z"), buildArrival(51, "z1"), buildArrival(2, "b"),
			},
		},
		{
			label: "repeated high-water mark",
			arrivals: [][2]interface{}{
				buildArrival(7, "g"), buildArrival(7, "g"), buildArrival(7, "g"), buildArrival(8, "h"),
			},
		},
	} {
		runDifferentialSequence(parseT, parseCase.label, parseCase.arrivals)
	}
}

// TestLedgerMatchesRuntime2OnGeneratedSequences sweeps a much larger space than
// hand-written cases reach, deterministically.
//
// An LCG rather than math/rand: the sequence must be identical on every machine
// and every run, or a divergence found in CI cannot be reproduced locally.
func TestLedgerMatchesRuntime2OnGeneratedSequences(parseT *testing.T) {
	const parseVersionSpace = 24
	const parseIdentitySpace = 3

	for parseSeed := uint64(1); parseSeed <= 64; parseSeed++ {
		parseRandomState := parseSeed
		parseArrivals := make([][2]interface{}, 0, 200)

		for parseStep := 0; parseStep < 200; parseStep++ {
			// Numerical Recipes LCG constants: deterministic and adequate here.
			parseRandomState = parseRandomState*6364136223846793005 + 1442695040888963407
			parseDraw := parseRandomState >> 33

			parseVersion := (parseDraw % parseVersionSpace) + 1
			// Identity usually follows the version (a well-behaved producer) and
			// occasionally does not, which is the conflict case.
			parseIdentity := fmt.Sprintf("id-%d", parseVersion)
			if (parseDraw/parseVersionSpace)%17 == 0 {
				parseIdentity = fmt.Sprintf("id-%d-variant-%d", parseVersion, (parseDraw/97)%parseIdentitySpace)
			}
			parseArrivals = append(parseArrivals, buildArrival(parseVersion, parseIdentity))
		}

		runDifferentialSequence(parseT, fmt.Sprintf("generated seed %d", parseSeed), parseArrivals)
	}
}

// TestLedgerMatchesRuntime2AcrossForwardEpochs pins the epoch behavior the two
// DO share: a forward epoch resets versioning in both.
//
// Backward epochs are the documented difference — runtime2 relies on its parse
// layer to reject them, the ledger guards them itself — and are covered by
// TestLedgerSkipsStaleEpoch rather than here.
func TestLedgerMatchesRuntime2AcrossForwardEpochs(parseT *testing.T) {
	parseTracker := runtime2.BuildPatchIdempotencyTracker()
	parseLedger := services.NewLedger()

	const parseRegionID = "epoch-region"

	for parseEpoch := uint64(1); parseEpoch <= 4; parseEpoch++ {
		for _, parseVersion := range []uint64{1, 2, 2, 1, 3} {
			parseIdentity := fmt.Sprintf("e%d-v%d", parseEpoch, parseVersion)

			parseTrackerApply, parseTrackerErr := parseTracker.HandlePatchIdempotency(
				parseRegionID, parseEpoch, parseVersion, parseIdentity)
			parseLedgerDecision, parseLedgerErr := parseLedger.Apply(
				parseRegionID, parseEpoch, parseVersion, parseIdentity)

			if (parseTrackerErr == nil) != (parseLedgerErr == nil) {
				parseT.Fatalf("epoch %d version %d: runtime2 err=%v but services err=%v",
					parseEpoch, parseVersion, parseTrackerErr, parseLedgerErr)
			}
			if parseTrackerApply != parseLedgerDecision.ShouldApply() {
				parseT.Fatalf("epoch %d version %d: runtime2 apply=%v but services said %s",
					parseEpoch, parseVersion, parseTrackerApply, parseLedgerDecision)
			}
		}
	}
}
