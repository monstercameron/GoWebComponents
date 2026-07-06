package virtualization

import (
	"strconv"
	"testing"
)

// TestRestorationStoreIsBoundedLRU pins that the process-global scroll-restoration
// store never grows past maxRestorationSnapshots and evicts least-recently-stored
// entries. Without the cap the map leaked one entry per unique (possibly dynamic)
// list ID for the lifetime of the process.
func TestRestorationStoreIsBoundedLRU(parseT *testing.T) {
	// Reset the global store for a deterministic test.
	restorationStore.mu.Lock()
	restorationStore.snapshots = map[string]restorationSnapshot{}
	restorationStore.order = nil
	restorationStore.mu.Unlock()

	total := maxRestorationSnapshots + 50
	for parseI := 0; parseI < total; parseI++ {
		storeRestorationSnapshot("list-"+strconv.Itoa(parseI), restorationSnapshot{ScrollTop: float64(parseI)})
	}

	restorationStore.mu.Lock()
	parseCount := len(restorationStore.snapshots)
	parseOrderLen := len(restorationStore.order)
	restorationStore.mu.Unlock()

	if parseCount != maxRestorationSnapshots {
		parseT.Fatalf("store size = %d, want capped at %d", parseCount, maxRestorationSnapshots)
	}
	if parseOrderLen != maxRestorationSnapshots {
		parseT.Fatalf("order length = %d, want %d (must stay in lockstep with the map)", parseOrderLen, maxRestorationSnapshots)
	}

	// The oldest 50 IDs must have been evicted; the newest maxRestorationSnapshots retained.
	if _, parseOk := loadRestorationSnapshotForTest("list-0"); parseOk {
		parseT.Fatal("expected the least-recently-stored snapshot to be evicted")
	}
	if _, parseOk := loadRestorationSnapshotForTest("list-" + strconv.Itoa(total-1)); !parseOk {
		parseT.Fatal("expected the most-recently-stored snapshot to be retained")
	}
}

// TestRestorationStoreReStoreDoesNotGrowOrder pins that re-storing an existing ID
// updates it in place (recency) rather than appending a duplicate to the order
// slice, so the order tracker can't grow unbounded under repeated scroll saves.
func TestRestorationStoreReStoreDoesNotGrowOrder(parseT *testing.T) {
	restorationStore.mu.Lock()
	restorationStore.snapshots = map[string]restorationSnapshot{}
	restorationStore.order = nil
	restorationStore.mu.Unlock()

	for parseI := 0; parseI < 500; parseI++ {
		storeRestorationSnapshot("same-list", restorationSnapshot{ScrollTop: float64(parseI)})
	}

	restorationStore.mu.Lock()
	parseCount := len(restorationStore.snapshots)
	parseOrderLen := len(restorationStore.order)
	restorationStore.mu.Unlock()

	if parseCount != 1 || parseOrderLen != 1 {
		parseT.Fatalf("re-storing one ID 500x should keep size=1/order=1, got size=%d order=%d", parseCount, parseOrderLen)
	}
}

// loadRestorationSnapshotForTest reads without triggering the browser-persistence
// fallback (which is a no-op natively anyway), keeping the assertion about the
// in-memory map alone.
func loadRestorationSnapshotForTest(parseId string) (restorationSnapshot, bool) {
	restorationStore.mu.Lock()
	defer restorationStore.mu.Unlock()
	parseSnapshot, parseOk := restorationStore.snapshots[parseId]
	return parseSnapshot, parseOk
}
