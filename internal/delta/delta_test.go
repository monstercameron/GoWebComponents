package delta

import (
	"fmt"
	"testing"
)

// v5 P3.15a — the delta publication engine.
//
// The criterion is that delta ops are O(change). Most of these tests are
// therefore about op COUNT, not just correctness: an engine that produced the
// right final state with N ops for a one-row edit would pass every correctness
// test and fail the only requirement that matters.

func buildRows(parseCount int) []Row {
	parseRows := make([]Row, 0, parseCount)
	for parseIndex := range parseCount {
		parseRows = append(parseRows, Row{
			Key:     Key(fmt.Sprintf("k%04d", parseIndex)),
			Version: 1,
			Payload: []byte(fmt.Sprintf("payload-%d", parseIndex)),
		})
	}
	return parseRows
}

// applyOps replays ops against a model of the consumer, so the tests can assert
// that the ops actually produce the intended state rather than merely counting
// them. A delta engine whose ops are minimal but wrong is worse than a slow one.
func applyOps(parseState []Row, parseOps []Op) ([]Row, error) {
	parsePayloadByKey := map[Key][]byte{}
	parseOrder := make([]Key, 0, len(parseState))
	for _, parseRow := range parseState {
		parsePayloadByKey[parseRow.Key] = parseRow.Payload
		parseOrder = append(parseOrder, parseRow.Key)
	}

	removeKey := func(parseKey Key) {
		for parsePosition, parseExisting := range parseOrder {
			if parseExisting == parseKey {
				parseOrder = append(parseOrder[:parsePosition], parseOrder[parsePosition+1:]...)
				return
			}
		}
	}
	insertAfter := func(parseKey Key, parseAfterKey Key) error {
		if parseAfterKey == "" {
			parseOrder = append([]Key{parseKey}, parseOrder...)
			return nil
		}
		for parsePosition, parseExisting := range parseOrder {
			if parseExisting == parseAfterKey {
				parseOrder = append(parseOrder, "")
				copy(parseOrder[parsePosition+2:], parseOrder[parsePosition+1:])
				parseOrder[parsePosition+1] = parseKey
				return nil
			}
		}
		return fmt.Errorf("anchor %q not present when placing %q", parseAfterKey, parseKey)
	}

	for _, parseOp := range parseOps {
		switch parseOp.Kind {
		case OpRemove:
			removeKey(parseOp.Key)
			delete(parsePayloadByKey, parseOp.Key)
		case OpInsert:
			if parseErr := insertAfter(parseOp.Key, parseOp.AfterKey); parseErr != nil {
				return nil, parseErr
			}
			parsePayloadByKey[parseOp.Key] = parseOp.Payload
		case OpMove:
			removeKey(parseOp.Key)
			if parseErr := insertAfter(parseOp.Key, parseOp.AfterKey); parseErr != nil {
				return nil, parseErr
			}
		case OpUpdate:
			if _, hasKey := parsePayloadByKey[parseOp.Key]; !hasKey {
				return nil, fmt.Errorf("update for absent key %q", parseOp.Key)
			}
			parsePayloadByKey[parseOp.Key] = parseOp.Payload
		}
	}

	parseResult := make([]Row, 0, len(parseOrder))
	for _, parseKey := range parseOrder {
		parseResult = append(parseResult, Row{Key: parseKey, Payload: parsePayloadByKey[parseKey]})
	}
	return parseResult, nil
}

func assertConsumerMatches(parseT *testing.T, parseLabel string, parseConsumer []Row, parseWant []Row) {
	parseT.Helper()
	if len(parseConsumer) != len(parseWant) {
		parseT.Fatalf("%s: consumer has %d rows, want %d", parseLabel, len(parseConsumer), len(parseWant))
	}
	for parseIndex := range parseWant {
		if parseConsumer[parseIndex].Key != parseWant[parseIndex].Key {
			parseT.Fatalf("%s: row %d is %q, want %q", parseLabel, parseIndex, parseConsumer[parseIndex].Key, parseWant[parseIndex].Key)
		}
		if string(parseConsumer[parseIndex].Payload) != string(parseWant[parseIndex].Payload) {
			parseT.Fatalf("%s: row %q payload = %q, want %q", parseLabel,
				parseWant[parseIndex].Key, parseConsumer[parseIndex].Payload, parseWant[parseIndex].Payload)
		}
	}
}

// ------------------------------------------------------- the O(change) claim

func TestFirstPublishInsertsEverything(parseT *testing.T) {
	parseEngine := New()
	parseRows := buildRows(100)

	parseOps, parseErr := parseEngine.Publish(parseRows)
	if parseErr != nil {
		parseT.Fatalf("Publish: %v", parseErr)
	}
	if len(parseOps) != 100 {
		parseT.Errorf("ops = %d, want 100 inserts on an empty projection", len(parseOps))
	}

	parseConsumer, parseApplyErr := applyOps(nil, parseOps)
	if parseApplyErr != nil {
		parseT.Fatalf("apply: %v", parseApplyErr)
	}
	assertConsumerMatches(parseT, "first publish", parseConsumer, parseRows)
}

// TestOneEditedRowInTwentyThousandPublishesOneOp is P3.15a's criterion, at the
// scale M11 is specified against.
func TestOneEditedRowInTwentyThousandPublishesOneOp(parseT *testing.T) {
	parseEngine := New()
	parseRows := buildRows(20000)
	if _, parseErr := parseEngine.Publish(parseRows); parseErr != nil {
		parseT.Fatalf("initial publish: %v", parseErr)
	}

	parseRows[9999].Version = 2
	parseRows[9999].Payload = []byte("edited")

	parseOps, parseErr := parseEngine.Publish(parseRows)
	if parseErr != nil {
		parseT.Fatalf("second publish: %v", parseErr)
	}
	if len(parseOps) != 1 {
		parseT.Fatalf("ops = %d, want exactly 1 for a one-row edit in 20,000", len(parseOps))
	}
	if parseOps[0].Kind != OpUpdate || parseOps[0].Key != "k9999" {
		parseT.Errorf("op = %+v, want an update of k9999", parseOps[0])
	}
}

func TestUnchangedRepublishPublishesNothing(parseT *testing.T) {
	parseEngine := New()
	parseRows := buildRows(500)
	if _, parseErr := parseEngine.Publish(parseRows); parseErr != nil {
		parseT.Fatalf("initial publish: %v", parseErr)
	}

	parseOps, parseErr := parseEngine.Publish(parseRows)
	if parseErr != nil {
		parseT.Fatalf("republish: %v", parseErr)
	}
	if len(parseOps) != 0 {
		parseT.Errorf("ops = %d, want 0 — republishing identical state must cost nothing", len(parseOps))
	}
}

// TestMovingOneRowToTheTopPublishesOneMove is why move minimization exists. A
// naive engine emits a move per displaced row, so dragging one row to the top of
// a 20,000-row table costs 20,000 ops.
func TestMovingOneRowToTheTopPublishesOneMove(parseT *testing.T) {
	parseEngine := New()
	parseRows := buildRows(2000)
	if _, parseErr := parseEngine.Publish(parseRows); parseErr != nil {
		parseT.Fatalf("initial publish: %v", parseErr)
	}

	parseMoved := append([]Row{parseRows[1500]}, append(append([]Row{}, parseRows[:1500]...), parseRows[1501:]...)...)

	parseOps, parseErr := parseEngine.Publish(parseMoved)
	if parseErr != nil {
		parseT.Fatalf("publish: %v", parseErr)
	}
	if len(parseOps) != 1 {
		parseT.Fatalf("ops = %d, want exactly 1 move", len(parseOps))
	}
	if parseOps[0].Kind != OpMove || parseOps[0].Key != "k1500" || parseOps[0].AfterKey != "" {
		parseT.Errorf("op = %+v, want k1500 moved to the head", parseOps[0])
	}

	parseConsumer, parseApplyErr := applyOps(parseRows, parseOps)
	if parseApplyErr != nil {
		parseT.Fatalf("apply: %v", parseApplyErr)
	}
	assertConsumerMatches(parseT, "single move", parseConsumer, parseMoved)
}

// TestReversalIsBoundedByTheChange: a full reversal genuinely changes almost
// everything, so a large op count is correct here. What must NOT happen is more
// than one op per row.
func TestReversalIsBoundedByTheChange(parseT *testing.T) {
	parseEngine := New()
	parseRows := buildRows(200)
	if _, parseErr := parseEngine.Publish(parseRows); parseErr != nil {
		parseT.Fatalf("initial publish: %v", parseErr)
	}

	parseReversed := make([]Row, 0, len(parseRows))
	for parseIndex := len(parseRows) - 1; parseIndex >= 0; parseIndex-- {
		parseReversed = append(parseReversed, parseRows[parseIndex])
	}

	parseOps, parseErr := parseEngine.Publish(parseReversed)
	if parseErr != nil {
		parseT.Fatalf("publish: %v", parseErr)
	}
	if len(parseOps) > len(parseRows) {
		parseT.Errorf("ops = %d for a %d-row reversal, want at most one per row", len(parseOps), len(parseRows))
	}

	parseConsumer, parseApplyErr := applyOps(parseRows, parseOps)
	if parseApplyErr != nil {
		parseT.Fatalf("apply: %v", parseApplyErr)
	}
	assertConsumerMatches(parseT, "reversal", parseConsumer, parseReversed)
}

// -------------------------------------------------------------- correctness

func TestMixedInsertUpdateRemoveAndMove(parseT *testing.T) {
	parseEngine := New()
	parseInitial := []Row{
		{Key: "a", Version: 1, Payload: []byte("A")},
		{Key: "b", Version: 1, Payload: []byte("B")},
		{Key: "c", Version: 1, Payload: []byte("C")},
		{Key: "d", Version: 1, Payload: []byte("D")},
	}
	if _, parseErr := parseEngine.Publish(parseInitial); parseErr != nil {
		parseT.Fatalf("initial: %v", parseErr)
	}

	parseNext := []Row{
		{Key: "c", Version: 1, Payload: []byte("C")},       // moved to the front
		{Key: "a", Version: 2, Payload: []byte("A-prime")}, // updated
		{Key: "e", Version: 1, Payload: []byte("E")},       // inserted
		{Key: "d", Version: 1, Payload: []byte("D")},       // unchanged
		// "b" removed
	}

	parseOps, parseErr := parseEngine.Publish(parseNext)
	if parseErr != nil {
		parseT.Fatalf("publish: %v", parseErr)
	}

	parseConsumer, parseApplyErr := applyOps(parseInitial, parseOps)
	if parseApplyErr != nil {
		parseT.Fatalf("apply: %v", parseApplyErr)
	}
	assertConsumerMatches(parseT, "mixed", parseConsumer, parseNext)
}

// TestMovedAndEditedRowGetsBothOps: a move carries no payload by design, so an
// engine that emitted only the move would leave the consumer showing stale data
// in the right place — the kind of bug that survives a correctness test written
// only against ordering.
func TestMovedAndEditedRowGetsBothOps(parseT *testing.T) {
	parseEngine := New()
	parseInitial := []Row{
		{Key: "a", Version: 1, Payload: []byte("A")},
		{Key: "b", Version: 1, Payload: []byte("B")},
		{Key: "c", Version: 1, Payload: []byte("C")},
	}
	if _, parseErr := parseEngine.Publish(parseInitial); parseErr != nil {
		parseT.Fatalf("initial: %v", parseErr)
	}

	parseNext := []Row{
		{Key: "c", Version: 2, Payload: []byte("C-prime")},
		{Key: "a", Version: 1, Payload: []byte("A")},
		{Key: "b", Version: 1, Payload: []byte("B")},
	}
	parseOps, parseErr := parseEngine.Publish(parseNext)
	if parseErr != nil {
		parseT.Fatalf("publish: %v", parseErr)
	}

	hasMove, hasUpdate := false, false
	for _, parseOp := range parseOps {
		if parseOp.Key == "c" && parseOp.Kind == OpMove {
			hasMove = true
		}
		if parseOp.Key == "c" && parseOp.Kind == OpUpdate {
			hasUpdate = true
		}
	}
	if !hasMove || !hasUpdate {
		parseT.Errorf("ops = %+v, want both a move and an update for c", parseOps)
	}

	parseConsumer, parseApplyErr := applyOps(parseInitial, parseOps)
	if parseApplyErr != nil {
		parseT.Fatalf("apply: %v", parseApplyErr)
	}
	assertConsumerMatches(parseT, "moved and edited", parseConsumer, parseNext)
}

func TestPublishToEmptyRemovesEverything(parseT *testing.T) {
	parseEngine := New()
	parseRows := buildRows(50)
	if _, parseErr := parseEngine.Publish(parseRows); parseErr != nil {
		parseT.Fatalf("initial: %v", parseErr)
	}

	parseOps, parseErr := parseEngine.Publish(nil)
	if parseErr != nil {
		parseT.Fatalf("publish empty: %v", parseErr)
	}
	if len(parseOps) != 50 {
		parseT.Errorf("ops = %d, want 50 removes", len(parseOps))
	}
	if parseEngine.Len() != 0 {
		parseT.Errorf("engine holds %d rows after publishing empty", parseEngine.Len())
	}
}

// TestRandomizedPublishSequencesConverge sweeps far more shapes than hand-written
// cases reach. Deterministic by construction so a failure reproduces.
func TestRandomizedPublishSequencesConverge(parseT *testing.T) {
	for parseSeed := uint64(1); parseSeed <= 50; parseSeed++ {
		parseEngine := New()
		var parseConsumer []Row
		parseRandom := parseSeed

		for parseRound := range 8 {
			parseRandom = parseRandom*6364136223846793005 + 1442695040888963407

			// Build a new state by sampling keys from a small space, so inserts,
			// removes, reorders, and edits all occur naturally.
			parseSize := int((parseRandom>>33)%12) + 1
			parseNext := make([]Row, 0, parseSize)
			parseUsed := map[Key]bool{}
			for range parseSize {
				parseRandom = parseRandom*6364136223846793005 + 1442695040888963407
				parseKey := Key(fmt.Sprintf("k%d", (parseRandom>>33)%10))
				if parseUsed[parseKey] {
					continue
				}
				parseUsed[parseKey] = true
				parseVersion := (parseRandom >> 20) % 3
				parseNext = append(parseNext, Row{
					Key:     parseKey,
					Version: parseVersion,
					Payload: []byte(fmt.Sprintf("%s-v%d", parseKey, parseVersion)),
				})
			}

			parseOps, parseErr := parseEngine.Publish(parseNext)
			if parseErr != nil {
				parseT.Fatalf("seed %d round %d: publish: %v", parseSeed, parseRound, parseErr)
			}
			parseApplied, parseApplyErr := applyOps(parseConsumer, parseOps)
			if parseApplyErr != nil {
				parseT.Fatalf("seed %d round %d: apply: %v", parseSeed, parseRound, parseApplyErr)
			}
			assertConsumerMatches(parseT, fmt.Sprintf("seed %d round %d", parseSeed, parseRound), parseApplied, parseNext)
			parseConsumer = parseApplied
		}
	}
}

// ------------------------------------------------------------- change-driven

func TestApplyIsOChangeAndTouchesNothingElse(parseT *testing.T) {
	parseEngine := New()
	parseRows := buildRows(20000)
	if _, parseErr := parseEngine.Publish(parseRows); parseErr != nil {
		parseT.Fatalf("initial: %v", parseErr)
	}

	parseOps, parseErr := parseEngine.Apply([]Change{
		{Kind: ChangeUpsert, Row: Row{Key: "k0500", Version: 2, Payload: []byte("edited")}},
	})
	if parseErr != nil {
		parseT.Fatalf("Apply: %v", parseErr)
	}
	if len(parseOps) != 1 || parseOps[0].Kind != OpUpdate || parseOps[0].Key != "k0500" {
		parseT.Errorf("ops = %+v, want exactly one update of k0500", parseOps)
	}
	if parseEngine.Len() != 20000 {
		parseT.Errorf("engine holds %d rows, want 20000 untouched", parseEngine.Len())
	}
}

func TestApplyUpsertInsertsAtAnAnchor(parseT *testing.T) {
	parseEngine := New()
	parseInitial := []Row{
		{Key: "a", Version: 1, Payload: []byte("A")},
		{Key: "b", Version: 1, Payload: []byte("B")},
	}
	if _, parseErr := parseEngine.Publish(parseInitial); parseErr != nil {
		parseT.Fatalf("initial: %v", parseErr)
	}

	parseOps, parseErr := parseEngine.Apply([]Change{
		{Kind: ChangeUpsert, AfterKey: "a", Row: Row{Key: "mid", Version: 1, Payload: []byte("M")}},
		{Kind: ChangeUpsert, AfterKey: "", Row: Row{Key: "head", Version: 1, Payload: []byte("H")}},
	})
	if parseErr != nil {
		parseT.Fatalf("Apply: %v", parseErr)
	}

	parseConsumer, parseApplyErr := applyOps(parseInitial, parseOps)
	if parseApplyErr != nil {
		parseT.Fatalf("apply: %v", parseApplyErr)
	}
	parseWantKeys := []Key{"head", "a", "mid", "b"}
	for parseIndex, parseWant := range parseWantKeys {
		if parseConsumer[parseIndex].Key != parseWant {
			parseT.Fatalf("order = %v, want %v", parseConsumer, parseWantKeys)
		}
	}
	// The engine's own order must agree, or the next Publish computes its diff
	// against a state the consumer does not have.
	parseEngineKeys := parseEngine.Keys()
	for parseIndex, parseWant := range parseWantKeys {
		if parseEngineKeys[parseIndex] != parseWant {
			parseT.Fatalf("engine order = %v, want %v", parseEngineKeys, parseWantKeys)
		}
	}
}

func TestApplyUpsertAtTheSameVersionPublishesNothing(parseT *testing.T) {
	parseEngine := New()
	if _, parseErr := parseEngine.Publish([]Row{{Key: "a", Version: 3, Payload: []byte("A")}}); parseErr != nil {
		parseT.Fatalf("initial: %v", parseErr)
	}

	parseOps, parseErr := parseEngine.Apply([]Change{
		{Kind: ChangeUpsert, Row: Row{Key: "a", Version: 3, Payload: []byte("A")}},
	})
	if parseErr != nil {
		parseT.Fatalf("Apply: %v", parseErr)
	}
	if len(parseOps) != 0 {
		parseT.Errorf("ops = %+v, want none — same version means same data", parseOps)
	}
}

// TestApplyRemoveOfAnAbsentKeyConverges: a producer replaying a change log
// should converge, not fail.
func TestApplyRemoveOfAnAbsentKeyConverges(parseT *testing.T) {
	parseEngine := New()
	parseOps, parseErr := parseEngine.Apply([]Change{
		{Kind: ChangeRemove, Row: Row{Key: "ghost"}},
	})
	if parseErr != nil {
		parseT.Fatalf("Apply: %v", parseErr)
	}
	if len(parseOps) != 0 {
		parseT.Errorf("ops = %+v, want none", parseOps)
	}
}

// TestApplyRejectsAnUnpublishedAnchor: guessing a position would put the row
// somewhere the producer did not ask for, and the consumer would never learn the
// order was wrong.
func TestApplyRejectsAnUnpublishedAnchor(parseT *testing.T) {
	parseEngine := New()
	if _, parseErr := parseEngine.Apply([]Change{
		{Kind: ChangeUpsert, AfterKey: "nope", Row: Row{Key: "a", Version: 1}},
	}); parseErr == nil {
		parseT.Error("an anchor that is not published must be rejected")
	}
}

// TestPublishAfterApplyDiffsAgainstTheAppliedState is the composition that would
// break if Apply updated its ops but not its own bookkeeping.
func TestPublishAfterApplyDiffsAgainstTheAppliedState(parseT *testing.T) {
	parseEngine := New()
	parseInitial := []Row{{Key: "a", Version: 1, Payload: []byte("A")}}
	if _, parseErr := parseEngine.Publish(parseInitial); parseErr != nil {
		parseT.Fatalf("initial: %v", parseErr)
	}
	if _, parseErr := parseEngine.Apply([]Change{
		{Kind: ChangeUpsert, AfterKey: "a", Row: Row{Key: "b", Version: 1, Payload: []byte("B")}},
	}); parseErr != nil {
		parseT.Fatalf("Apply: %v", parseErr)
	}

	parseOps, parseErr := parseEngine.Publish([]Row{
		{Key: "a", Version: 1, Payload: []byte("A")},
		{Key: "b", Version: 1, Payload: []byte("B")},
	})
	if parseErr != nil {
		parseT.Fatalf("Publish: %v", parseErr)
	}
	if len(parseOps) != 0 {
		parseT.Errorf("ops = %+v, want none — the publish state matches what Apply left", parseOps)
	}
}

// ------------------------------------------------------------------- guards

func TestPublishRejectsDuplicateKeys(parseT *testing.T) {
	parseEngine := New()
	if _, parseErr := parseEngine.Publish([]Row{
		{Key: "a", Version: 1},
		{Key: "a", Version: 2},
	}); parseErr == nil {
		parseT.Error("two rows claiming one key have no correct delta and must be rejected")
	}
}

func TestPublishRejectsAnEmptyKey(parseT *testing.T) {
	parseEngine := New()
	if _, parseErr := parseEngine.Publish([]Row{{Version: 1}}); parseErr == nil {
		parseT.Error("a row without a key must be rejected")
	}
	if _, parseErr := parseEngine.Apply([]Change{{Kind: ChangeUpsert, Row: Row{Version: 1}}}); parseErr == nil {
		parseT.Error("a change without a key must be rejected")
	}
}

func TestResetForcesAFullRepublish(parseT *testing.T) {
	parseEngine := New()
	parseRows := buildRows(10)
	if _, parseErr := parseEngine.Publish(parseRows); parseErr != nil {
		parseT.Fatalf("initial: %v", parseErr)
	}

	parseEngine.Reset()
	parseOps, parseErr := parseEngine.Publish(parseRows)
	if parseErr != nil {
		parseT.Fatalf("publish after reset: %v", parseErr)
	}
	if len(parseOps) != 10 {
		parseT.Errorf("ops = %d, want 10 inserts after a reset", len(parseOps))
	}
}

func TestNilEngineIsSafe(parseT *testing.T) {
	var parseEngine *Engine
	if _, parseErr := parseEngine.Publish(nil); parseErr == nil {
		parseT.Error("a nil engine must error rather than panic")
	}
	if _, parseErr := parseEngine.Apply(nil); parseErr == nil {
		parseT.Error("a nil engine must error rather than panic")
	}
	if parseEngine.Len() != 0 || parseEngine.Keys() != nil {
		parseT.Error("a nil engine reads as empty")
	}
	parseEngine.Reset()
}

func TestUnknownChangeKindIsRejected(parseT *testing.T) {
	parseEngine := New()
	if _, parseErr := parseEngine.Apply([]Change{{Kind: ChangeKind(99), Row: Row{Key: "a"}}}); parseErr == nil {
		parseT.Error("an unknown change kind must be rejected")
	}
}

func TestOpKindLabelsAreDistinct(parseT *testing.T) {
	parseSeen := map[string]bool{}
	for _, parseKind := range []OpKind{OpInsert, OpUpdate, OpRemove, OpMove} {
		if parseSeen[parseKind.String()] {
			parseT.Errorf("op label %q is not distinct", parseKind)
		}
		parseSeen[parseKind.String()] = true
	}
	if OpKind(99).String() == "" {
		parseT.Error("an unknown op kind must still render for diagnostics")
	}
}

// TestLongestIncreasingSubsequence covers the move-minimization core directly,
// because a subtle bug there shows up only as extra ops — correct output, wrong
// cost, which the correctness tests would not catch.
func TestLongestIncreasingSubsequence(parseT *testing.T) {
	for _, parseCase := range []struct {
		label  string
		input  []int
		length int
	}{
		{"empty", nil, 0},
		{"single", []int{5}, 1},
		{"already increasing", []int{1, 2, 3, 4}, 4},
		{"strictly decreasing", []int{4, 3, 2, 1}, 1},
		{"one out of place", []int{1, 2, 9, 3, 4}, 4},
		{"classic", []int{10, 9, 2, 5, 3, 7, 101, 18}, 4},
	} {
		parseOffsets := longestIncreasingSubsequence(parseCase.input)
		if len(parseOffsets) != parseCase.length {
			parseT.Errorf("%s: length = %d, want %d", parseCase.label, len(parseOffsets), parseCase.length)
			continue
		}
		// The offsets must be increasing and select increasing values, or the
		// "these rows can stay put" claim is false.
		for parseIndex := 1; parseIndex < len(parseOffsets); parseIndex++ {
			if parseOffsets[parseIndex] <= parseOffsets[parseIndex-1] {
				parseT.Errorf("%s: offsets %v are not increasing", parseCase.label, parseOffsets)
				break
			}
			if parseCase.input[parseOffsets[parseIndex]] <= parseCase.input[parseOffsets[parseIndex-1]] {
				parseT.Errorf("%s: selected values are not increasing", parseCase.label)
				break
			}
		}
	}
}
