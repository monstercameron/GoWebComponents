package fetch

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v4/query"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// DurableMutationRunner applies a mutation that is BOTH optimistic and durable: it updates the
// query cache immediately (so the UI reacts now) and enqueues the write to the persistent
// MutationQueue (so it survives a reload or offline gap). It returns the queued entry and any
// enqueue error.
type DurableMutationRunner[T any] func(parseOptimistic T, parseDraft MutationDraft, parseCommit func(context.Context, QueuedMutation) (T, error)) (QueuedMutation, error)

// UseDurableMutation bridges the optimistic query cache and the durable offline MutationQueue,
// which previously lived in separate packages with no link (the audit's FA2 gap). The returned
// runner: (1) durably enqueues the draft so the write is never lost; (2) applies the optimistic
// value to the cache and re-renders; (3) runs commit in the background — on success it clears the
// queue entry and commits the authoritative value, on failure it rolls the cache back and LEAVES
// the entry queued for later replay via MutationQueue.Replay. One call gives you optimistic UI,
// offline durability, and automatic reconciliation.
//
//	run := fetch.UseDurableMutation[int](appCache, "post/"+id+"/likes", queue)
//	onClick := func() {
//	    run(current+1, fetch.MutationDraft{Method: "POST", URL: "/api/like", Body: id},
//	        func(ctx context.Context, m fetch.QueuedMutation) (int, error) { return serverfn.Call(...) })
//	}
func UseDurableMutation[T any](parseCache *query.Cache, parseKey string, parseQueue MutationQueue) DurableMutationRunner[T] {
	parseRerender := ui.UseForceUpdate()
	return func(parseOptimistic T, parseDraft MutationDraft, parseCommit func(context.Context, QueuedMutation) (T, error)) (QueuedMutation, error) {
		return runDurableMutation(parseCache, parseKey, parseQueue, parseOptimistic, parseDraft, parseCommit, parseRerender)
	}
}

// runDurableMutation is the hook-free core of UseDurableMutation, so the durability + optimism
// bridge is unit-testable without a render context. onSettled is invoked once optimistically and
// again when the background commit settles.
func runDurableMutation[T any](
	parseCache *query.Cache,
	parseKey string,
	parseQueue MutationQueue,
	parseOptimistic T,
	parseDraft MutationDraft,
	parseCommit func(context.Context, QueuedMutation) (T, error),
	parseOnSettled func(),
) (QueuedMutation, error) {
	// Durability first: if the write cannot be persisted, do not show an optimism we cannot keep.
	parseQueued, parseErr := parseQueue.Enqueue(parseDraft)
	if parseErr != nil {
		return QueuedMutation{}, parseErr
	}

	query.MutateAsync(parseCache, parseKey, parseOptimistic, func() (T, error) {
		parseValue, parseCommitErr := parseCommit(context.Background(), parseQueued)
		if parseCommitErr == nil {
			// Committed authoritatively — the durable entry is no longer needed.
			_ = parseQueue.Remove(parseQueued.ID)
		}
		// On error the entry stays queued for MutationQueue.Replay to retry later.
		return parseValue, parseCommitErr
	}, func(query.Result[T]) {
		if parseOnSettled != nil {
			parseOnSettled()
		}
	})

	if parseOnSettled != nil {
		parseOnSettled() // optimistic value is live now
	}
	return parseQueued, nil
}
