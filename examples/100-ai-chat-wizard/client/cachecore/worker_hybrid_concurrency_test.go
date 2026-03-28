package cachecore

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestWorkerHybridConcurrencyNoDoubleApply verifies concurrent worker events cannot double-apply acks or resurrect evicted drafts.
func TestWorkerHybridConcurrencyNoDoubleApply(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := BuildMemoryStorage()
	parseSnapshot := BuildSnapshotStore()
	parseReconciler := BuildWorkerSnapshotReconciler(parseStorage, parseSnapshot)

	parseScopeKey := BuildScopeKey(ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1", Locale: "en-US", RouteKey: "/app/chat", ThreadKey: "thread-1"})
	parseDraftResourceKey := "draft.thread-1"
	parseThreadResourceKey := "thread.history.thread-1"
	parseDraftScopedKey := BuildScopedResourceKey(parseScopeKey, parseDraftResourceKey)
	parseQueueKey := BuildScopedResourceKey(parseScopeKey, "outbox.unsent_messages")

	parseLocalDraft := BuildCacheRecordEnvelope(parseScopeKey, parseDraftResourceKey, "local-v1", "hash-local-v1", time.Now().UTC(), time.Minute, time.Hour, []byte(`{"draft":"hello"}`), CacheStatusReady, "")
	if parseErr := parseStorage.Set(parseCtx, parseLocalDraft); parseErr != nil {
		parseT.Fatalf("Set(local draft): %v", parseErr)
	}
	parseSnapshot.SetSync(parseLocalDraft)
	_, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []OutboxRecordEnvelope) ([]OutboxRecordEnvelope, error) {
		return append(parseQueue, BuildOutboxRecordEnvelope(parseQueueKey, "chat.send", "send-op-1", time.Now().UTC(), time.Now().UTC(), []byte(`{"text":"hello"}`), OutboxStatusQueued, "ack-send-op-1", "")), nil
	})
	if parseErr != nil {
		parseT.Fatalf("UpdateQueueAtomic(seed): %v", parseErr)
	}

	parseBase := time.Now().UTC()
	parseRefreshAt := parseBase.Add(2 * time.Second)
	parseEvictAt := parseBase.Add(3 * time.Second)
	parseAckAt := parseBase.Add(4 * time.Second)

	parseRefreshRecord := BuildCacheRecordEnvelope(parseScopeKey, parseDraftResourceKey, "server-v2", "hash-server-v2", parseRefreshAt, time.Minute, time.Hour, []byte(`{"draft":"hello world"}`), CacheStatusReady, "")
	parseAckedThreadRecord := BuildCacheRecordEnvelope(parseScopeKey, parseThreadResourceKey, "thread-v1", "hash-thread-v1", parseAckAt, time.Minute, time.Hour, []byte(`{"messages":[{"id":"m1"}]}`), CacheStatusReady, "")

	parseResults := make(chan WorkerSnapshotReconcileResult, 4)
	parseRun := func(parseInput WorkerSnapshotReconcileInput) {
		parseResult, parseErr := parseReconciler.ApplyEvent(parseCtx, parseInput)
		if parseErr != nil {
			parseT.Errorf("ApplyEvent error: %v", parseErr)
			return
		}
		parseResults <- parseResult
	}

	parseWaitGroup := sync.WaitGroup{}
	parseWaitGroup.Add(4)
	go func() {
		defer parseWaitGroup.Done()
		parseRun(WorkerSnapshotReconcileInput{
			Event: WorkerEvent{
				EventType:   WorkerEventUpdated,
				ScopeKey:    parseScopeKey,
				ResourceKey: parseDraftResourceKey,
				UpdatedAt:   parseRefreshAt.Format(time.RFC3339),
			},
			Record: parseRefreshRecord,
		})
	}()
	go func() {
		defer parseWaitGroup.Done()
		parseRun(WorkerSnapshotReconcileInput{
			Event: WorkerEvent{
				EventType:   WorkerEventEvicted,
				ScopeKey:    parseScopeKey,
				ResourceKey: parseDraftResourceKey,
				UpdatedAt:   parseEvictAt.Format(time.RFC3339),
			},
			InvalidatedScopedRecords: []string{parseDraftScopedKey},
		})
	}()
	go func() {
		defer parseWaitGroup.Done()
		parseRun(WorkerSnapshotReconcileInput{
			Event: WorkerEvent{
				EventType:   WorkerEventAcked,
				ScopeKey:    parseScopeKey,
				ResourceKey: parseThreadResourceKey,
				UpdatedAt:   parseAckAt.Format(time.RFC3339),
			},
			Record:          parseAckedThreadRecord,
			AckQueueKey:     parseQueueKey,
			AckOperationKey: "send-op-1",
		})
	}()
	go func() {
		defer parseWaitGroup.Done()
		parseRun(WorkerSnapshotReconcileInput{
			Event: WorkerEvent{
				EventType:   WorkerEventAcked,
				ScopeKey:    parseScopeKey,
				ResourceKey: parseThreadResourceKey,
				UpdatedAt:   parseAckAt.Format(time.RFC3339),
			},
			Record:          parseAckedThreadRecord,
			AckQueueKey:     parseQueueKey,
			AckOperationKey: "send-op-1",
		})
	}()
	parseWaitGroup.Wait()
	close(parseResults)

	parseQueueRemovedTotal := 0
	parseDuplicateCount := 0
	for parseResult := range parseResults {
		parseQueueRemovedTotal += parseResult.QueueRemovedCount
		if parseResult.IsDuplicate {
			parseDuplicateCount++
		}
	}
	if parseQueueRemovedTotal > 1 {
		parseT.Fatalf("expected at most one queue removal for duplicated ack events, got %d", parseQueueRemovedTotal)
	}
	if parseDuplicateCount == 0 {
		parseT.Fatalf("expected one duplicate ack outcome under concurrent identical events")
	}

	parseDraftRecord, isParseDraftFound, parseErr := parseStorage.Get(parseCtx, parseDraftScopedKey)
	if parseErr != nil {
		parseT.Fatalf("Get(draft): %v", parseErr)
	}
	if isParseDraftFound {
		parseT.Fatalf("expected draft to remain evicted after concurrent events, got %+v", parseDraftRecord)
	}
	if _, isParseDraftInSnapshot := parseSnapshot.GetSync(parseScopeKey, parseDraftResourceKey); isParseDraftInSnapshot {
		parseT.Fatalf("expected draft snapshot to stay evicted")
	}

	parseQueue, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []OutboxRecordEnvelope) ([]OutboxRecordEnvelope, error) {
		return parseQueue, nil
	})
	if parseErr != nil {
		parseT.Fatalf("UpdateQueueAtomic(read): %v", parseErr)
	}
	if len(parseQueue) != 0 {
		parseT.Fatalf("expected unsent queue to be empty after ack dedupe, got %+v", parseQueue)
	}

	parseThreadRecord, isParseThreadFound, parseErr := parseStorage.Get(parseCtx, BuildScopedResourceKey(parseScopeKey, parseThreadResourceKey))
	if parseErr != nil || !isParseThreadFound {
		parseT.Fatalf("Get(thread): found=%t err=%v", isParseThreadFound, parseErr)
	}
	if parseThreadRecord.Version != "thread-v1" {
		parseT.Fatalf("expected acked thread snapshot version thread-v1, got %+v", parseThreadRecord)
	}
}
