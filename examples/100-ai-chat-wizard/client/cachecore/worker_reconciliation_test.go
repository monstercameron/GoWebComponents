package cachecore

import (
	"context"
	"testing"
	"time"
)

// TestWorkerSnapshotReconcilerApplyEventAckAndUpdate verifies queue ack plus authoritative record merge behavior.
func TestWorkerSnapshotReconcilerApplyEventAckAndUpdate(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := BuildMemoryStorage()
	parseSnapshot := BuildSnapshotStore()
	parseReconciler := BuildWorkerSnapshotReconciler(parseStorage, parseSnapshot)

	parseScopeKey := BuildScopeKey(ScopeKey{AppVersion: "v1", SessionID: "s1", UserID: "u1", Workspace: "w1"})
	parseQueueKey := BuildScopedResourceKey(parseScopeKey, "outbox.unsent")
	_, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []OutboxRecordEnvelope) ([]OutboxRecordEnvelope, error) {
		return append(parseQueue,
			BuildOutboxRecordEnvelope(parseQueueKey, "chat.send", "op-1", time.Now().UTC(), time.Now().UTC(), []byte(`{"message":"one"}`), OutboxStatusQueued, "ack-1", ""),
			BuildOutboxRecordEnvelope(parseQueueKey, "chat.send", "op-2", time.Now().UTC(), time.Now().UTC(), []byte(`{"message":"two"}`), OutboxStatusQueued, "ack-2", ""),
		), nil
	})
	if parseErr != nil {
		parseT.Fatalf("UpdateQueueAtomic(seed): %v", parseErr)
	}

	parseRecord := BuildCacheRecordEnvelope(parseScopeKey, "thread.history", "v2", "h2", time.Now().UTC(), time.Minute, time.Hour, []byte(`{"messages":[1,2]}`), CacheStatusReady, "")
	parseResult, parseErr := parseReconciler.ApplyEvent(parseCtx, WorkerSnapshotReconcileInput{
		Event: WorkerEvent{
			EventType:   WorkerEventAcked,
			ScopeKey:    parseScopeKey,
			ResourceKey: "thread.history",
			UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
		Record:          parseRecord,
		AckQueueKey:     parseQueueKey,
		AckOperationKey: "op-1",
	})
	if parseErr != nil {
		parseT.Fatalf("ApplyEvent: %v", parseErr)
	}
	if !parseResult.IsApplied || !parseResult.IsRecordUpdated || parseResult.QueueRemovedCount != 1 {
		parseT.Fatalf("unexpected reconcile result: %+v", parseResult)
	}

	parseSnapshotRecord, isParseFound := parseSnapshot.GetSync(parseScopeKey, "thread.history")
	if !isParseFound {
		parseT.Fatalf("expected snapshot record after reconcile")
	}
	if parseSnapshotRecord.Version != "v2" {
		parseT.Fatalf("expected version v2, got %q", parseSnapshotRecord.Version)
	}

	parseQueue, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []OutboxRecordEnvelope) ([]OutboxRecordEnvelope, error) {
		return parseQueue, nil
	})
	if parseErr != nil {
		parseT.Fatalf("UpdateQueueAtomic(read): %v", parseErr)
	}
	if len(parseQueue) != 1 || parseQueue[0].OperationKey != "op-2" {
		parseT.Fatalf("expected only op-2 in queue, got %+v", parseQueue)
	}
}

// TestWorkerSnapshotReconcilerApplyEventInvalidationAndEviction verifies deterministic delete merges.
func TestWorkerSnapshotReconcilerApplyEventInvalidationAndEviction(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := BuildMemoryStorage()
	parseSnapshot := BuildSnapshotStore()
	parseReconciler := BuildWorkerSnapshotReconciler(parseStorage, parseSnapshot)

	parseScopeKey := BuildScopeKey(ScopeKey{AppVersion: "v1", SessionID: "s1", UserID: "u1", Workspace: "w1"})
	parseKeepRecord := BuildCacheRecordEnvelope(parseScopeKey, "resource.keep", "v1", "h1", time.Now().UTC(), time.Minute, time.Hour, []byte(`{"ok":1}`), CacheStatusReady, "")
	parseEvictRecord := BuildCacheRecordEnvelope(parseScopeKey, "resource.evict", "v1", "h2", time.Now().UTC(), time.Minute, time.Hour, []byte(`{"evict":1}`), CacheStatusReady, "")
	if parseErr := parseStorage.Set(parseCtx, parseKeepRecord); parseErr != nil {
		parseT.Fatalf("Set(keep): %v", parseErr)
	}
	if parseErr := parseStorage.Set(parseCtx, parseEvictRecord); parseErr != nil {
		parseT.Fatalf("Set(evict): %v", parseErr)
	}
	parseSnapshot.SetSync(parseKeepRecord)
	parseSnapshot.SetSync(parseEvictRecord)

	parseInvalidateScopedKey := BuildScopedResourceKey(parseScopeKey, "resource.keep")
	parseResult, parseErr := parseReconciler.ApplyEvent(parseCtx, WorkerSnapshotReconcileInput{
		Event: WorkerEvent{
			EventType:   WorkerEventEvicted,
			ScopeKey:    parseScopeKey,
			ResourceKey: "resource.evict",
			UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
		InvalidatedScopedRecords: []string{parseInvalidateScopedKey},
	})
	if parseErr != nil {
		parseT.Fatalf("ApplyEvent(evict): %v", parseErr)
	}
	if parseResult.DeletedCount != 2 {
		parseT.Fatalf("expected 2 deletes, got %+v", parseResult)
	}
	if _, isParseFound := parseSnapshot.GetSync(parseScopeKey, "resource.keep"); isParseFound {
		parseT.Fatalf("expected invalidated keep record to be removed")
	}
	if _, isParseFound := parseSnapshot.GetSync(parseScopeKey, "resource.evict"); isParseFound {
		parseT.Fatalf("expected evicted record to be removed")
	}
}

// TestWorkerSnapshotReconcilerApplyEventDedupAndOrder verifies duplicate and stale ordering guardrails.
func TestWorkerSnapshotReconcilerApplyEventDedupAndOrder(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := BuildMemoryStorage()
	parseSnapshot := BuildSnapshotStore()
	parseReconciler := BuildWorkerSnapshotReconciler(parseStorage, parseSnapshot)

	parseScopeKey := BuildScopeKey(ScopeKey{AppVersion: "v1", SessionID: "s1", UserID: "u1", Workspace: "w1"})
	parseResourceKey := "thread.history"
	parseNewerAt := time.Now().UTC()
	parseOlderAt := parseNewerAt.Add(-time.Minute)

	parseNewerRecord := BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "v2", "hash-v2", parseNewerAt, time.Minute, time.Hour, []byte(`{"v":2}`), CacheStatusReady, "")
	parseApplyResult, parseErr := parseReconciler.ApplyEvent(parseCtx, WorkerSnapshotReconcileInput{
		Event: WorkerEvent{
			EventType:   WorkerEventUpdated,
			ScopeKey:    parseScopeKey,
			ResourceKey: parseResourceKey,
			UpdatedAt:   parseNewerAt.Format(time.RFC3339),
		},
		Record: parseNewerRecord,
	})
	if parseErr != nil {
		parseT.Fatalf("ApplyEvent(newer): %v", parseErr)
	}
	if !parseApplyResult.IsApplied || !parseApplyResult.IsRecordUpdated {
		parseT.Fatalf("expected newer event apply, got %+v", parseApplyResult)
	}

	parseDuplicateResult, parseErr := parseReconciler.ApplyEvent(parseCtx, WorkerSnapshotReconcileInput{
		Event: WorkerEvent{
			EventType:   WorkerEventUpdated,
			ScopeKey:    parseScopeKey,
			ResourceKey: parseResourceKey,
			UpdatedAt:   parseNewerAt.Format(time.RFC3339),
		},
		Record: parseNewerRecord,
	})
	if parseErr != nil {
		parseT.Fatalf("ApplyEvent(duplicate): %v", parseErr)
	}
	if !parseDuplicateResult.IsDuplicate || parseDuplicateResult.IsApplied {
		parseT.Fatalf("expected duplicate outcome, got %+v", parseDuplicateResult)
	}

	parseOlderRecord := BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "v1", "hash-v1", parseOlderAt, time.Minute, time.Hour, []byte(`{"v":1}`), CacheStatusReady, "")
	parseOutOfOrderResult, parseErr := parseReconciler.ApplyEvent(parseCtx, WorkerSnapshotReconcileInput{
		Event: WorkerEvent{
			EventType:   WorkerEventUpdated,
			ScopeKey:    parseScopeKey,
			ResourceKey: parseResourceKey,
			UpdatedAt:   parseOlderAt.Format(time.RFC3339),
		},
		Record: parseOlderRecord,
	})
	if parseErr != nil {
		parseT.Fatalf("ApplyEvent(out-of-order): %v", parseErr)
	}
	if !parseOutOfOrderResult.IsOutOfOrder || parseOutOfOrderResult.IsApplied {
		parseT.Fatalf("expected out-of-order outcome, got %+v", parseOutOfOrderResult)
	}
	parseSnapshotRecord, isParseFound := parseSnapshot.GetSync(parseScopeKey, parseResourceKey)
	if !isParseFound || parseSnapshotRecord.Version != "v2" {
		parseT.Fatalf("expected snapshot to keep v2 after out-of-order drop, got found=%t record=%+v", isParseFound, parseSnapshotRecord)
	}
}
