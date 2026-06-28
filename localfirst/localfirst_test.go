package localfirst

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
)

// TestClockAfterDeterministicTiebreak proves the conflict order is total and deterministic:
// higher counter wins, and equal counters are broken by replica id (never a coin flip).
func TestClockAfterDeterministicTiebreak(parseT *testing.T) {
	if !(Clock{Counter: 2}).After(Clock{Counter: 1}) {
		parseT.Fatal("higher counter must win")
	}
	if (Clock{Counter: 1}).After(Clock{Counter: 2}) {
		parseT.Fatal("lower counter must lose")
	}
	if !(Clock{Counter: 1, ReplicaID: "B"}).After(Clock{Counter: 1, ReplicaID: "A"}) {
		parseT.Fatal("equal counters must break ties by replica id")
	}
	if (Clock{Counter: 1, ReplicaID: "A"}).After(Clock{Counter: 1, ReplicaID: "B"}) {
		parseT.Fatal("tie-break must be antisymmetric")
	}
}

// TestOptimisticLocalWriteThenSync proves a write is visible immediately (optimistic) and
// that a sync round-trip pushes it to the authority and clears the pending log.
func TestOptimisticLocalWriteThenSync(parseT *testing.T) {
	parseAuthority := NewAuthority()
	parseReplica := NewReplica("A")

	parseReplica.Set("greeting", "hello")
	if parseValue, parseOk := parseReplica.Get("greeting"); !parseOk || parseValue != "hello" {
		parseT.Fatal("write should be visible locally before any sync")
	}
	if len(parseReplica.Pending()) != 1 {
		parseT.Fatalf("expected 1 pending mutation, got %d", len(parseReplica.Pending()))
	}

	Sync(parseReplica, parseAuthority)

	if len(parseReplica.Pending()) != 0 {
		parseT.Fatalf("expected pending cleared after sync, got %d", len(parseReplica.Pending()))
	}
	if parseAuthority.Snapshot()["greeting"] != "hello" {
		parseT.Fatalf("authority should hold the synced value, got %v", parseAuthority.Snapshot())
	}
}

// TestOfflineConflictConverges is the marquee local-first guarantee: two replicas mutate
// the SAME key while offline, then reconnect in sequence — and every replica plus the
// authority converge to the one last-write-wins value, with no lost pending writes.
func TestOfflineConflictConverges(parseT *testing.T) {
	parseAuthority := NewAuthority()
	parseReplicaA := NewReplica("A")
	parseReplicaB := NewReplica("B")

	// Both edit "doc" while offline (no Sync yet). Equal counters (1); B wins the tie.
	parseReplicaA.Set("doc", "from-A")
	parseReplicaB.Set("doc", "from-B")

	// A reconnects first and converges with the authority.
	Sync(parseReplicaA, parseAuthority)
	if parseAuthority.Snapshot()["doc"] != "from-A" {
		parseT.Fatalf("after A syncs, authority should hold from-A, got %v", parseAuthority.Snapshot())
	}

	// B reconnects; its write has the same counter but a higher replica id, so it wins.
	Sync(parseReplicaB, parseAuthority)
	if parseAuthority.Snapshot()["doc"] != "from-B" {
		parseT.Fatalf("after B syncs, LWW should pick from-B, got %v", parseAuthority.Snapshot())
	}

	// A pulls again (empty pending) and must adopt the winner — full convergence.
	Sync(parseReplicaA, parseAuthority)

	parseWant := map[string]string{"doc": "from-B"}
	for parseName, parseSnapshot := range map[string]map[string]string{
		"authority": parseAuthority.Snapshot(),
		"replicaA":  parseReplicaA.Snapshot(),
		"replicaB":  parseReplicaB.Snapshot(),
	} {
		if !equalStringMaps(parseSnapshot, parseWant) {
			parseT.Fatalf("%s did not converge: got %v, want %v", parseName, parseSnapshot, parseWant)
		}
	}
	if len(parseReplicaA.Pending()) != 0 || len(parseReplicaB.Pending()) != 0 {
		parseT.Fatal("no pending mutations should remain after convergence")
	}
}

// TestReEditAfterPullWins proves Lamport monotonicity: a replica that pulls a remote winner
// and then edits again produces a clock that outranks it, so its new value sticks.
func TestReEditAfterPullWins(parseT *testing.T) {
	parseAuthority := NewAuthority()
	parseA := NewReplica("A")
	parseB := NewReplica("B")

	parseB.Set("k", "b-wins-tie")
	Sync(parseB, parseAuthority) // authority: k=b-wins-tie @ (1,B)
	Sync(parseA, parseAuthority) // A pulls k=(1,B), advances its counter past 1

	parseA.Set("k", "a-re-edit") // must mint a clock with counter > 1
	Sync(parseA, parseAuthority)

	if parseAuthority.Snapshot()["k"] != "a-re-edit" {
		parseT.Fatalf("a re-edit after pulling should win, got %v", parseAuthority.Snapshot())
	}
}

// TestDeleteTombstoneConverges proves a delete propagates and removes the key everywhere.
func TestDeleteTombstoneConverges(parseT *testing.T) {
	parseAuthority := NewAuthority()
	parseA := NewReplica("A")
	parseB := NewReplica("B")

	parseA.Set("item", "value")
	Sync(parseA, parseAuthority)
	Sync(parseB, parseAuthority)
	if _, parseOk := parseB.Get("item"); !parseOk {
		parseT.Fatal("B should have received the item before deletion")
	}

	parseA.Delete("item")
	Sync(parseA, parseAuthority)
	Sync(parseB, parseAuthority)

	if _, parseOk := parseB.Get("item"); parseOk {
		parseT.Fatal("delete tombstone should have removed the item on B")
	}
	if len(parseAuthority.Snapshot()) != 0 {
		parseT.Fatalf("authority should have no live records, got %v", parseAuthority.Snapshot())
	}
}

// TestConcurrentWritesAreSafe exercises the replica mutex under concurrent writers and
// asserts every write is queued (no lost updates, no panic).
func TestConcurrentWritesAreSafe(parseT *testing.T) {
	parseReplica := NewReplica("A")
	var parseWG sync.WaitGroup
	for parseI := range 50 {
		parseWG.Add(1)
		go func(parseN int) {
			defer parseWG.Done()
			parseReplica.Set(fmt.Sprintf("k%d", parseN), "v")
		}(parseI)
	}
	parseWG.Wait()
	if len(parseReplica.Pending()) != 50 {
		parseT.Fatalf("expected 50 pending mutations, got %d", len(parseReplica.Pending()))
	}
}

// TestDurableOfflineQueueSurvivesReload proves the offline queue is durable: a replica with
// unsynced offline writes is exported, round-tripped through JSON (as a reload would do via
// storage), restored, and then converges — no offline write is lost across the "reload".
func TestDurableOfflineQueueSurvivesReload(parseT *testing.T) {
	parseAuthority := NewAuthority()

	// Edit offline (never synced), then "close the tab": serialize the whole replica.
	parseBefore := NewReplica("A")
	parseBefore.Set("draft", "unsynced-work")
	parseBefore.Set("note", "also-offline")
	if len(parseBefore.Pending()) != 2 {
		parseT.Fatalf("expected 2 pending offline writes, got %d", len(parseBefore.Pending()))
	}

	parseBlob, parseErr := json.Marshal(parseBefore.Export())
	if parseErr != nil {
		parseT.Fatalf("marshal replica state: %v", parseErr)
	}

	// "Reopen the tab": decode persisted state and restore the replica.
	var parseState ReplicaState
	if parseErr := json.Unmarshal(parseBlob, &parseState); parseErr != nil {
		parseT.Fatalf("unmarshal replica state: %v", parseErr)
	}
	parseAfter := RestoreReplica(parseState)

	if len(parseAfter.Pending()) != 2 {
		parseT.Fatalf("offline queue lost across reload: %d pending, want 2", len(parseAfter.Pending()))
	}
	if parseValue, parseOk := parseAfter.Get("draft"); !parseOk || parseValue != "unsynced-work" {
		parseT.Fatalf("restored replica lost local value, got %q", parseValue)
	}

	// Reconnect: the restored offline writes reach the authority and converge.
	Sync(parseAfter, parseAuthority)
	if parseAuthority.Snapshot()["draft"] != "unsynced-work" || parseAuthority.Snapshot()["note"] != "also-offline" {
		parseT.Fatalf("restored offline writes did not converge: %v", parseAuthority.Snapshot())
	}
	if len(parseAfter.Pending()) != 0 {
		parseT.Fatal("pending should clear after the restored writes sync")
	}
}

// TestAuthorityReceiveResolvesConflictDirectly proves Authority.Receive applies last-write-
// wins in isolation (not only via Sync): a later batch with the same key but a winning clock
// overrides the earlier record, and a losing clock is rejected — and the returned changes
// reflect the resolved state.
func TestAuthorityReceiveResolvesConflictDirectly(parseT *testing.T) {
	parseAuthority := NewAuthority()

	// First write wins for now.
	parseChanges := parseAuthority.Receive([]Mutation{
		{Record: Record{Key: "doc", Value: "from-A", Clock: Clock{Counter: 1, ReplicaID: "A"}}},
	})
	if len(parseChanges) != 1 || parseChanges[0].Value != "from-A" {
		parseT.Fatalf("expected from-A accepted, got %+v", parseChanges)
	}

	// A higher-clock write overrides it.
	parseChanges = parseAuthority.Receive([]Mutation{
		{Record: Record{Key: "doc", Value: "from-B", Clock: Clock{Counter: 2, ReplicaID: "B"}}},
	})
	if len(parseChanges) != 1 || parseChanges[0].Value != "from-B" {
		parseT.Fatalf("expected higher-clock from-B to win, got %+v", parseChanges)
	}

	// A stale write (lower clock) is rejected; Receive echoes the current winner.
	parseChanges = parseAuthority.Receive([]Mutation{
		{Record: Record{Key: "doc", Value: "stale-A", Clock: Clock{Counter: 1, ReplicaID: "A"}}},
	})
	if len(parseChanges) != 1 || parseChanges[0].Value != "from-B" {
		parseT.Fatalf("a stale write must not override; expected from-B, got %+v", parseChanges)
	}
	if parseAuthority.Snapshot()["doc"] != "from-B" {
		parseT.Fatalf("authority should hold from-B, got %v", parseAuthority.Snapshot())
	}
}

// equalStringMaps reports whether two string maps are equal.
func equalStringMaps(parseA, parseB map[string]string) bool {
	if len(parseA) != len(parseB) {
		return false
	}
	for parseKey, parseValue := range parseA {
		if parseB[parseKey] != parseValue {
			return false
		}
	}
	return true
}
