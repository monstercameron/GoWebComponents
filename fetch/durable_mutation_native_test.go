//go:build !js || !wasm

package fetch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/interop"
	"github.com/monstercameron/GoWebComponents/v5/query"
)

// openDurableTestQueue opens a mutation queue backed by the in-memory native store.
func openDurableTestQueue(parseT *testing.T) MutationQueue {
	parseT.Helper()
	parseStore, _ := buildFetchTestPersistentStore(parseT)
	parseNow := time.Date(2026, time.June, 27, 12, 0, 0, 0, time.UTC)
	parseQueue, parseErr := OpenMutationQueue(MutationQueueOptions{
		StorageKey:  "durable-test",
		MaxAttempts: 3,
		BaseDelay:   time.Second,
		MaxDelay:    2 * time.Second,
		StoreResolver: func(context.Context) (interop.PersistentStore, error) {
			return parseStore, nil
		},
		Now: func() time.Time { return parseNow },
	})
	if parseErr != nil {
		parseT.Fatalf("OpenMutationQueue: %v", parseErr)
	}
	return parseQueue
}

// TestDurableMutationSuccessClearsQueue proves the FA2 bridge: a successful durable mutation
// applies the optimistic value to the query cache, persists the write, and — once the commit
// succeeds — clears the queue entry and commits the authoritative value.
func TestDurableMutationSuccessClearsQueue(parseT *testing.T) {
	parseQueue := openDurableTestQueue(parseT)
	parseCache := query.New()

	parseSettled := make(chan struct{}, 2)
	parseQueued, parseErr := runDurableMutation[int](parseCache, "likes", parseQueue, 5,
		MutationDraft{Method: "POST", URL: "/api/like", Kind: "like"},
		func(context.Context, QueuedMutation) (int, error) { return 7, nil },
		func() { parseSettled <- struct{}{} },
	)
	if parseErr != nil {
		parseT.Fatalf("runDurableMutation: %v", parseErr)
	}
	if parseQueued.ID == "" {
		parseT.Fatal("expected a durable queue entry id")
	}
	// Optimistic value is live immediately.
	if parseSnap := query.Snapshot[int](parseCache, "likes"); parseSnap.Data != 5 {
		parseT.Fatalf("expected optimistic 5 in cache, got %d", parseSnap.Data)
	}
	// Entry was durably enqueued.
	if parseList, _ := parseQueue.List(); len(parseList) != 1 {
		parseT.Fatalf("expected 1 durable entry, got %d", len(parseList))
	}

	waitDurableSettle(parseT, parseSettled)
	// Authoritative value committed and the durable entry cleared.
	if parseSnap := query.Snapshot[int](parseCache, "likes"); parseSnap.Data != 7 {
		parseT.Fatalf("expected committed 7 in cache, got %d", parseSnap.Data)
	}
	if parseList, _ := parseQueue.List(); len(parseList) != 0 {
		parseT.Fatalf("successful commit should clear the queue, %d remain", len(parseList))
	}
}

// TestDurableMutationFailureKeepsQueued proves a failed commit rolls the cache back but LEAVES
// the entry in the durable queue for later replay (offline durability).
func TestDurableMutationFailureKeepsQueued(parseT *testing.T) {
	parseQueue := openDurableTestQueue(parseT)
	parseCache := query.New()
	parseCache.Set("likes", 1) // prior authoritative value

	parseSettled := make(chan struct{}, 2)
	parseBoom := errors.New("offline")
	_, parseErr := runDurableMutation[int](parseCache, "likes", parseQueue, 2,
		MutationDraft{Method: "POST", URL: "/api/like", Kind: "like"},
		func(context.Context, QueuedMutation) (int, error) { return 0, parseBoom },
		func() { parseSettled <- struct{}{} },
	)
	if parseErr != nil {
		parseT.Fatalf("runDurableMutation: %v", parseErr)
	}

	waitDurableSettle(parseT, parseSettled)
	// Cache rolled back to the prior value.
	if parseSnap := query.Snapshot[int](parseCache, "likes"); parseSnap.Data != 1 {
		parseT.Fatalf("expected rollback to 1, got %d", parseSnap.Data)
	}
	// The write survives in the durable queue for replay.
	if parseList, _ := parseQueue.List(); len(parseList) != 1 {
		parseT.Fatalf("a failed commit must keep the entry queued for replay, got %d", len(parseList))
	}
}

// waitDurableSettle waits for the background commit's settle callback (the second signal — the
// first is the synchronous optimistic settle).
func waitDurableSettle(parseT *testing.T, parseSettled chan struct{}) {
	parseT.Helper()
	parseReceived := 0
	parseDeadline := time.After(2 * time.Second)
	for parseReceived < 2 {
		select {
		case <-parseSettled:
			parseReceived++
		case <-parseDeadline:
			parseT.Fatalf("timed out waiting for durable mutation to settle (got %d/2 signals)", parseReceived)
		}
	}
}
