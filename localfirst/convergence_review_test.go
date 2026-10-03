package localfirst

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

// cloneText deep-copies a Text through its exported state so replicas can be merged in many
// different orders without aliasing each other's element pointers.
func cloneText(parseT *Text, parseReplica string) *Text {
	return RestoreText(parseT.Export(), parseReplica)
}

// buildRandomTexts creates n replicas that each apply a random burst of inserts and deletes after
// an optional shared prefix, with random pairwise merges in between (so clocks skew apart).
func buildRandomTexts(parseRand *rand.Rand, parseReplicas int) []*Text {
	parseTexts := make([]*Text, parseReplicas)
	for parseIndex := range parseTexts {
		parseTexts[parseIndex] = NewText(fmt.Sprintf("r%d", parseIndex))
	}
	for parseStep := 0; parseStep < 60; parseStep++ {
		parseText := parseTexts[parseRand.Intn(parseReplicas)]
		switch parseRand.Intn(5) {
		case 0, 1, 2:
			parseText.Insert(parseRand.Intn(parseText.Len()+2)-1, string(rune('a'+parseRand.Intn(26))))
		case 3:
			parseText.Delete(parseRand.Intn(parseText.Len()+1), 1+parseRand.Intn(2))
		case 4:
			parseOther := parseTexts[parseRand.Intn(parseReplicas)]
			parseText.Merge(parseOther)
		}
	}
	return parseTexts
}

// TestTextMergeConvergesUnderAnyOrderAndDuplicates is a property test: for random concurrent
// histories, merging all replicas into a fresh replica in any permutation, with every delivery
// duplicated, yields the identical string (commutativity, associativity, idempotence).
func TestTextMergeConvergesUnderAnyOrderAndDuplicates(parseT *testing.T) {
	for parseSeed := int64(1); parseSeed <= 200; parseSeed++ {
		parseRand := rand.New(rand.NewSource(parseSeed))
		parseTexts := buildRandomTexts(parseRand, 4)

		var parseWant string
		for parseTrial := 0; parseTrial < 6; parseTrial++ {
			parseOrder := parseRand.Perm(len(parseTexts))
			parseFresh := NewText("observer")
			for _, parseIndex := range parseOrder {
				parseFresh.Merge(parseTexts[parseIndex])
				if parseRand.Intn(2) == 0 {
					parseFresh.Merge(parseTexts[parseIndex]) // duplicate delivery
				}
			}
			// Re-merge everything once more; must be a no-op.
			before := parseFresh.Value()
			for _, parseText := range parseTexts {
				parseFresh.Merge(parseText)
			}
			if parseFresh.Value() != before {
				parseT.Fatalf("seed %d: re-merge changed value %q -> %q", parseSeed, before, parseFresh.Value())
			}
			if parseTrial == 0 {
				parseWant = parseFresh.Value()
				continue
			}
			if parseFresh.Value() != parseWant {
				parseT.Fatalf("seed %d trial %d: order %v gave %q, want %q", parseSeed, parseTrial, parseOrder, parseFresh.Value(), parseWant)
			}
		}
	}
}

// TestTextAssociativity checks (a.merge(b)).merge(c) == a.merge(b.merge(c)).
func TestTextAssociativity(parseT *testing.T) {
	for parseSeed := int64(1); parseSeed <= 200; parseSeed++ {
		parseRand := rand.New(rand.NewSource(parseSeed))
		parseTexts := buildRandomTexts(parseRand, 3)
		parseA, parseB, parseC := parseTexts[0], parseTexts[1], parseTexts[2]

		parseLeft := cloneText(parseA, "x")
		parseLeft.Merge(parseB)
		parseLeft.Merge(parseC)

		parseBC := cloneText(parseB, "y")
		parseBC.Merge(parseC)
		parseRight := cloneText(parseA, "x")
		parseRight.Merge(parseBC)

		if parseLeft.Value() != parseRight.Value() {
			parseT.Fatalf("seed %d: (a+b)+c=%q but a+(b+c)=%q", parseSeed, parseLeft.Value(), parseRight.Value())
		}
		if !reflect.DeepEqual(parseLeft.Export().Ops, parseRight.Export().Ops) {
			parseT.Fatalf("seed %d: exported op sets differ", parseSeed)
		}
	}
}

// TestTextRestoreKeepsTombstoneOnDuplicateOps pins that a persisted/transported op list that
// repeats an element (duplicate delivery) keeps the tombstone: delete must stay monotonic.
func TestTextRestoreKeepsTombstoneOnDuplicateOps(parseT *testing.T) {
	parseOps := []TextOp{
		{Replica: "a", Seq: 1, Char: "x", Deleted: true},
		{Replica: "a", Seq: 1, Char: "x"}, // stale duplicate delivered after the delete
	}
	parseText := RestoreText(TextState{Ops: parseOps}, "b")
	if parseText.Value() != "" {
		parseT.Fatalf("deleted element resurrected by a duplicate op: %q", parseText.Value())
	}
}

// TestTextLocalInsertsAfterMergeKeepCausalOrder pins that after merging a remote edit, a local
// insert mints an id above everything seen so it sorts after concurrent older siblings.
func TestTextLocalInsertsAfterMergeKeepCausalOrder(parseT *testing.T) {
	parseA := NewText("a")
	parseA.Insert(0, "hello")
	parseB := NewText("b")
	parseB.Merge(parseA)
	parseB.Insert(5, "!")
	parseA.Merge(parseB)
	if parseA.Value() != "hello!" || parseB.Value() != "hello!" {
		parseT.Fatalf("a=%q b=%q, want hello!", parseA.Value(), parseB.Value())
	}
}

// stripZeroTallies drops zero-valued replica entries: Inc(replica, 0) records an explicit zero
// that Merge never copies (0 is not greater than absent), which is semantically identical.
func stripZeroTallies(parseState CounterState) CounterState {
	parseOut := CounterState{Pos: map[string]uint64{}, Neg: map[string]uint64{}}
	for parseKey, parseValue := range parseState.Pos {
		if parseValue != 0 {
			parseOut.Pos[parseKey] = parseValue
		}
	}
	for parseKey, parseValue := range parseState.Neg {
		if parseValue != 0 {
			parseOut.Neg[parseKey] = parseValue
		}
	}
	return parseOut
}

// buildRandomCounters builds n counters mutated independently.
func buildRandomCounters(parseRand *rand.Rand, parseCount int) []*Counter {
	parseCounters := make([]*Counter, parseCount)
	for parseIndex := range parseCounters {
		parseCounters[parseIndex] = NewCounter()
		parseReplica := fmt.Sprintf("r%d", parseIndex)
		for range parseRand.Intn(8) {
			if parseRand.Intn(2) == 0 {
				parseCounters[parseIndex].Inc(parseReplica, uint64(parseRand.Intn(10)))
			} else {
				parseCounters[parseIndex].Dec(parseReplica, uint64(parseRand.Intn(10)))
			}
		}
	}
	return parseCounters
}

// TestCounterMergeProperties checks commutativity, associativity and idempotence of PN-Counter
// merge over random states, including repeated delivery.
func TestCounterMergeProperties(parseT *testing.T) {
	for parseSeed := int64(1); parseSeed <= 300; parseSeed++ {
		parseRand := rand.New(rand.NewSource(parseSeed))
		parseCounters := buildRandomCounters(parseRand, 3)
		parseA, parseB, parseC := parseCounters[0], parseCounters[1], parseCounters[2]

		parseAB := RestoreCounter(parseA.Export())
		parseAB.Merge(parseB)
		parseBA := RestoreCounter(parseB.Export())
		parseBA.Merge(parseA)
		if !reflect.DeepEqual(stripZeroTallies(parseAB.Export()), stripZeroTallies(parseBA.Export())) {
			parseT.Fatalf("seed %d: merge not commutative", parseSeed)
		}

		parseLeft := RestoreCounter(parseAB.Export())
		parseLeft.Merge(parseC)
		parseBC := RestoreCounter(parseB.Export())
		parseBC.Merge(parseC)
		parseRight := RestoreCounter(parseA.Export())
		parseRight.Merge(parseBC)
		if !reflect.DeepEqual(stripZeroTallies(parseLeft.Export()), stripZeroTallies(parseRight.Export())) {
			parseT.Fatalf("seed %d: merge not associative", parseSeed)
		}

		parseBefore := parseLeft.Value()
		parseLeft.Merge(parseA)
		parseLeft.Merge(parseC)
		parseLeft.Merge(parseLeft)
		if parseLeft.Value() != parseBefore {
			parseT.Fatalf("seed %d: merge not idempotent: %d -> %d", parseSeed, parseBefore, parseLeft.Value())
		}
	}
}

// TestReplicaAuthorityConvergesUnderRandomSyncOrder drives several replicas through random
// offline writes, deletes and syncs (with skewed Lamport counters) and asserts every replica and
// the authority agree after a final pull, whatever the sync interleaving.
func TestReplicaAuthorityConvergesUnderRandomSyncOrder(parseT *testing.T) {
	for parseSeed := int64(1); parseSeed <= 200; parseSeed++ {
		parseRand := rand.New(rand.NewSource(parseSeed))
		parseAuthority := NewAuthority()
		parseReplicas := []*Replica{NewReplica("a"), NewReplica("b"), NewReplica("c")}
		for parseStep := 0; parseStep < 80; parseStep++ {
			parseReplica := parseReplicas[parseRand.Intn(len(parseReplicas))]
			parseKey := fmt.Sprintf("k%d", parseRand.Intn(4))
			switch parseRand.Intn(5) {
			case 0, 1:
				parseReplica.Set(parseKey, fmt.Sprintf("v%d", parseRand.Intn(100)))
			case 2:
				parseReplica.Delete(parseKey)
			case 3, 4:
				Sync(parseReplica, parseAuthority)
			}
		}
		for parseRound := 0; parseRound < 2; parseRound++ {
			for _, parseReplica := range parseReplicas {
				Sync(parseReplica, parseAuthority)
			}
		}
		parseWant := parseAuthority.Snapshot()
		for parseIndex, parseReplica := range parseReplicas {
			if !reflect.DeepEqual(parseReplica.Snapshot(), parseWant) {
				parseT.Fatalf("seed %d: replica %d = %v, authority = %v", parseSeed, parseIndex, parseReplica.Snapshot(), parseWant)
			}
			if len(parseReplica.Pending()) != 0 {
				parseT.Fatalf("seed %d: replica %d still has %d pending after final sync", parseSeed, parseIndex, len(parseReplica.Pending()))
			}
		}
	}
}

// TestReplicaSurvivesHugeRemoteCounter pins that a remote clock near the uint64 ceiling does not
// wrap the local Lamport counter to zero (which would make every later local write lose to it).
func TestReplicaSurvivesHugeRemoteCounter(parseT *testing.T) {
	parseReplica := NewReplica("a")
	parseReplica.Merge([]Record{{Key: "k", Value: "remote", Clock: Clock{Counter: ^uint64(0), ReplicaID: "z"}}})
	parseRecord := parseReplica.Set("k", "local")
	if parseRecord.Clock.Counter == 0 {
		parseT.Fatalf("local counter wrapped to zero after observing a max remote clock")
	}
}
