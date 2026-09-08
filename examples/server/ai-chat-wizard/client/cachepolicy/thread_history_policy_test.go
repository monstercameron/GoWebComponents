package cachepolicy

import (
	"context"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/client/cachecore"
)

// TestStoreReadThreadHistory verifies thread-history snapshots restore quickly via snapshot-first reads.
func TestStoreReadThreadHistory(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseAPI := cachecore.BuildUIAPIWithSnapshot(parseStorage, parseSnapshot, cachecore.BuildSWRCoordinator(), cachecore.BuildInvalidationEngine(parseStorage, cachecore.BuildInvalidator()))
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})
	parseThreadKey := "thread-123"
	parseResourceKey := BuildThreadHistoryResourceKey(parseThreadKey)
	parsePolicy := BuildThreadHistoryPolicy()

	parseStaleRecord := cachecore.BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "v1", "hash-v1", time.Now().UTC().Add(-3*time.Minute), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, []byte(`{"messages":[{"id":"m1"}]}`), cachecore.CacheStatusReady, "")
	if parseErr := parseStorage.Set(parseCtx, parseStaleRecord); parseErr != nil {
		parseT.Fatalf("Set(seed): %v", parseErr)
	}
	parseSnapshot.SetSync(parseStaleRecord)
	parseRefreshDone := make(chan struct{}, 1)

	parseView, parseErr := ReadThreadHistory(parseCtx, parseAPI, parseScopeKey, parseThreadKey, func(parseCtx context.Context, parseScopeKey string, parseResourceKey string) (cachecore.CacheRecordEnvelope, error) {
		_ = parseCtx
		parseRefreshDone <- struct{}{}
		return cachecore.BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "v2", "hash-v2", time.Now().UTC(), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, []byte(`{"messages":[{"id":"m1"},{"id":"m2"}]}`), cachecore.CacheStatusReady, ""), nil
	})
	if parseErr != nil {
		parseT.Fatalf("ReadThreadHistory: %v", parseErr)
	}
	if !parseView.IsFound || !parseView.IsCached || !parseView.IsStale || !parseView.RefreshTriggered {
		parseT.Fatalf("unexpected thread-history view: %+v", parseView)
	}
	select {
	case <-parseRefreshDone:
	case <-time.After(2 * time.Second):
		parseT.Fatal("timed out waiting for thread-history background refresh")
	}
}

// TestApplyThreadHistoryStreamPatchAndAuthoritativeReconcile verifies stream patch updates and reconnect authoritative ordering.
func TestApplyThreadHistoryStreamPatchAndAuthoritativeReconcile(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})
	parseThreadKey := "thread-123"

	if parseErr := StoreThreadHistorySnapshot(parseCtx, parseStorage, parseScopeKey, parseThreadKey, "v1", []byte(`{"delta":"A"}`)); parseErr != nil {
		parseT.Fatalf("StoreThreadHistorySnapshot: %v", parseErr)
	}
	parseSeedRecord, isParseFound, parseErr := parseStorage.Get(parseCtx, cachecore.BuildScopedResourceKey(parseScopeKey, BuildThreadHistoryResourceKey(parseThreadKey)))
	if parseErr != nil || !isParseFound {
		parseT.Fatalf("Get(seed): found=%t err=%v", isParseFound, parseErr)
	}
	parseSnapshot.SetSync(parseSeedRecord)

	if parseErr = ApplyThreadHistoryStreamPatch(parseCtx, parseStorage, parseSnapshot, parseScopeKey, parseThreadKey, func(parsePayload []byte) []byte {
		return append(parsePayload, []byte(`{"delta":"B"}`)...)
	}); parseErr != nil {
		parseT.Fatalf("ApplyThreadHistoryStreamPatch: %v", parseErr)
	}
	parseStreamPatchedRecord, isParseFound, parseErr := parseStorage.Get(parseCtx, cachecore.BuildScopedResourceKey(parseScopeKey, BuildThreadHistoryResourceKey(parseThreadKey)))
	if parseErr != nil || !isParseFound {
		parseT.Fatalf("Get(stream-patched): found=%t err=%v", isParseFound, parseErr)
	}
	if string(parseStreamPatchedRecord.Payload) != `{"delta":"A"}{"delta":"B"}` {
		parseT.Fatalf("unexpected stream-patched payload: %s", string(parseStreamPatchedRecord.Payload))
	}

	parseReconciler := cachecore.BuildWorkerSnapshotReconciler(parseStorage, parseSnapshot)
	parseBaseTime := time.Now().UTC()
	parseNewerRecord := cachecore.BuildCacheRecordEnvelope(parseScopeKey, BuildThreadHistoryResourceKey(parseThreadKey), "v3", "hash-v3", parseBaseTime.Add(2*time.Second), BuildThreadHistoryPolicy().StaleAfter, BuildThreadHistoryPolicy().ExpiresAfter, []byte(`{"authoritative":"new"}`), cachecore.CacheStatusReady, "")
	parseNewerResult, parseErr := ApplyThreadHistoryAuthoritativeReconcile(parseCtx, parseReconciler, parseScopeKey, parseThreadKey, parseBaseTime.Add(2*time.Second), parseNewerRecord)
	if parseErr != nil {
		parseT.Fatalf("ApplyThreadHistoryAuthoritativeReconcile(newer): %v", parseErr)
	}
	if !parseNewerResult.IsApplied || !parseNewerResult.IsRecordUpdated {
		parseT.Fatalf("expected newer authoritative reconcile apply, got %+v", parseNewerResult)
	}
	parseOlderRecord := cachecore.BuildCacheRecordEnvelope(parseScopeKey, BuildThreadHistoryResourceKey(parseThreadKey), "v2", "hash-v2", parseBaseTime.Add(time.Second), BuildThreadHistoryPolicy().StaleAfter, BuildThreadHistoryPolicy().ExpiresAfter, []byte(`{"authoritative":"old"}`), cachecore.CacheStatusReady, "")
	parseOlderResult, parseErr := ApplyThreadHistoryAuthoritativeReconcile(parseCtx, parseReconciler, parseScopeKey, parseThreadKey, parseBaseTime.Add(time.Second), parseOlderRecord)
	if parseErr != nil {
		parseT.Fatalf("ApplyThreadHistoryAuthoritativeReconcile(older): %v", parseErr)
	}
	if !parseOlderResult.IsOutOfOrder || parseOlderResult.IsApplied {
		parseT.Fatalf("expected out-of-order authoritative reconcile drop, got %+v", parseOlderResult)
	}
	parseFinalRecord, isParseFound, parseErr := parseStorage.Get(parseCtx, cachecore.BuildScopedResourceKey(parseScopeKey, BuildThreadHistoryResourceKey(parseThreadKey)))
	if parseErr != nil || !isParseFound {
		parseT.Fatalf("Get(final): found=%t err=%v", isParseFound, parseErr)
	}
	if parseFinalRecord.Version != "v3" || string(parseFinalRecord.Payload) != `{"authoritative":"new"}` {
		parseT.Fatalf("expected newer authoritative record to win, got %+v", parseFinalRecord)
	}
}
