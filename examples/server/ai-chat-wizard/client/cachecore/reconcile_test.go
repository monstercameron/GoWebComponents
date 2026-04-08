package cachecore

import (
	"context"
	"testing"
	"time"
)

// TestApplyAckReconcile verifies queue-ack removal, authoritative merge, and optimistic cleanup behavior.
func TestApplyAckReconcile(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := BuildMemoryStorage()
	parseScopeKey := BuildScopeKey(ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1", Locale: "en-US", RouteKey: "/app", ThreadKey: "thread-1"})
	parseResourceKey := "resource.thread"
	parseQueueKey := "queue.send"
	_, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []OutboxRecordEnvelope) ([]OutboxRecordEnvelope, error) {
		return []OutboxRecordEnvelope{
			BuildOutboxRecordEnvelope(parseQueueKey, parseResourceKey, "op-1", time.Now().UTC(), time.Now().UTC().Add(30*time.Second), []byte(`{"optimistic":1}`), OutboxStatusQueued, "ack-1", ""),
			BuildOutboxRecordEnvelope(parseQueueKey, parseResourceKey, "op-2", time.Now().UTC(), time.Now().UTC().Add(30*time.Second), []byte(`{"optimistic":2}`), OutboxStatusQueued, "ack-2", ""),
		}, nil
	})
	if parseErr != nil {
		parseT.Fatalf("UpdateQueueAtomic(seed): %v", parseErr)
	}
	parseOptimisticScopedResourceKey := BuildScopedResourceKey(parseScopeKey, parseResourceKey+"#optimistic")
	if parseErr = parseStorage.Set(parseCtx, BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey+"#optimistic", "v0", "", time.Now().UTC(), 0, time.Minute, []byte(`{"pending":true}`), CacheStatusReady, "")); parseErr != nil {
		parseT.Fatalf("Set(optimistic): %v", parseErr)
	}
	parseAuthoritativeRecord := BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "v2", "etag-2", time.Now().UTC(), 30*time.Second, 120*time.Second, []byte(`{"authoritative":true}`), CacheStatusReady, "")
	parseResult, parseErr := ApplyAckReconcile(parseCtx, parseStorage, AckReconcileInput{
		QueueKey:            parseQueueKey,
		AckKey:              "ack-1",
		ScopeKey:            parseScopeKey,
		ResourceKey:         parseResourceKey,
		AuthoritativeRecord: parseAuthoritativeRecord,
		OptimisticScopedResources: []string{
			parseOptimisticScopedResourceKey,
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ApplyAckReconcile: %v", parseErr)
	}
	if parseResult.QueueRemovedCount != 1 || !parseResult.IsCacheUpdated {
		parseT.Fatalf("unexpected reconcile result: %+v", parseResult)
	}
	parseQueueAfter, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []OutboxRecordEnvelope) ([]OutboxRecordEnvelope, error) {
		return parseQueue, nil
	})
	if parseErr != nil {
		parseT.Fatalf("UpdateQueueAtomic(read): %v", parseErr)
	}
	if len(parseQueueAfter) != 1 || parseQueueAfter[0].AckKey != "ack-2" {
		parseT.Fatalf("unexpected queue after ack reconcile: %+v", parseQueueAfter)
	}
	if _, isParseOptimisticFound, _ := parseStorage.Get(parseCtx, parseOptimisticScopedResourceKey); isParseOptimisticFound {
		parseT.Fatal("expected optimistic placeholder to be removed")
	}
	parseAuthoritativeScopedResourceKey := BuildScopedResourceKey(parseScopeKey, parseResourceKey)
	parseLoadedAuthoritative, isParseAuthoritativeFound, parseErr := parseStorage.Get(parseCtx, parseAuthoritativeScopedResourceKey)
	if parseErr != nil || !isParseAuthoritativeFound {
		parseT.Fatalf("Get(authoritative): found=%v err=%v", isParseAuthoritativeFound, parseErr)
	}
	if parseLoadedAuthoritative.Version != "v2" || parseLoadedAuthoritative.ETagOrHash != "etag-2" {
		parseT.Fatalf("unexpected authoritative record after reconcile: %+v", parseLoadedAuthoritative)
	}
}
