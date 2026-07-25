package cachepolicy

import (
	"context"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/client/cachecore"
)

// TestApplyReconnectConflictResolution verifies reconnect conflict handling across unsent queue dedupe, settings merge, preference ack, and thread reconcile.
func TestApplyReconnectConflictResolution(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseReconciler := cachecore.BuildWorkerSnapshotReconciler(parseStorage, parseSnapshot)
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})

	if parseErr := StoreUnsentMessage(parseCtx, parseStorage, parseScopeKey, "", "send-op-1", []byte(`{"text":"from new chat"}`)); parseErr != nil {
		parseT.Fatalf("StoreUnsentMessage(new-chat op1): %v", parseErr)
	}
	if parseErr := StoreUnsentMessage(parseCtx, parseStorage, parseScopeKey, "", "send-op-2", []byte(`{"text":"from new chat op2"}`)); parseErr != nil {
		parseT.Fatalf("StoreUnsentMessage(new-chat op2): %v", parseErr)
	}
	if parseErr := StoreUnsentMessage(parseCtx, parseStorage, parseScopeKey, "thread-123", "send-op-2", []byte(`{"text":"thread existing op2"}`)); parseErr != nil {
		parseT.Fatalf("StoreUnsentMessage(thread op2): %v", parseErr)
	}

	parseSettingsScopedResourceKey := cachecore.BuildScopedResourceKey(parseScopeKey, BuildSettingsSnapshotResourceKey(settingsSnapshotSectionPreferences))
	parseStaleSettingsRecord := cachecore.BuildCacheRecordEnvelope(parseScopeKey, BuildSettingsSnapshotResourceKey(settingsSnapshotSectionPreferences), "v1", "", time.Now().UTC(), 0, BuildSettingsSnapshotPolicy().ExpiresAfter, []byte(`{"remembered":["old"]}`), cachecore.CacheStatusStale, "unavailable")
	if parseErr := parseStorage.Set(parseCtx, parseStaleSettingsRecord); parseErr != nil {
		parseT.Fatalf("Set(stale settings): %v", parseErr)
	}
	parseSnapshot.SetSync(parseStaleSettingsRecord)

	parsePreferenceQueueKey := cachecore.BuildScopedResourceKey(parseScopeKey, rememberedPreferenceQueueResourceKey)
	if _, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parsePreferenceQueueKey, func(parseQueue []cachecore.OutboxRecordEnvelope) ([]cachecore.OutboxRecordEnvelope, error) {
		return []cachecore.OutboxRecordEnvelope{
			cachecore.BuildOutboxRecordEnvelope(parsePreferenceQueueKey, "settings.preferences", "pref-op-1", time.Now().UTC(), time.Now().UTC(), []byte(`{"k":"tone"}`), cachecore.OutboxStatusQueued, "", ""),
		}, nil
	}); parseErr != nil {
		parseT.Fatalf("UpdateQueueAtomic(preference seed): %v", parseErr)
	}

	parseThreadRecord := cachecore.BuildCacheRecordEnvelope(parseScopeKey, BuildThreadHistoryResourceKey("thread-123"), "thread-v2", "hash-v2", time.Now().UTC(), BuildThreadHistoryPolicy().StaleAfter, BuildThreadHistoryPolicy().ExpiresAfter, []byte(`{"messages":[{"id":"m1"}]}`), cachecore.CacheStatusReady, "")
	parseResult, parseErr := ApplyReconnectConflictResolution(parseCtx, parseStorage, parseSnapshot, parseReconciler, ReconnectConflictResolutionInput{
		ScopeKey:                        parseScopeKey,
		CanonicalThreadRoutePublicID:    "thread-123",
		UnsentSourceThreadRoutePublicID: "",
		ThreadUpdatedAt:                 time.Now().UTC(),
		AuthoritativeThreadRecord:       parseThreadRecord,
		AuthoritativeSettingsBySection: map[string]cachecore.CacheRecordEnvelope{
			settingsSnapshotSectionPreferences: cachecore.BuildCacheRecordEnvelope(parseScopeKey, BuildSettingsSnapshotResourceKey(settingsSnapshotSectionPreferences), "v2", "", time.Now().UTC(), BuildSettingsSnapshotPolicy().StaleAfter, BuildSettingsSnapshotPolicy().ExpiresAfter, []byte(`{"remembered":["new"]}`), cachecore.CacheStatusReady, ""),
		},
		AckedPreferenceOperationKeys: []string{"pref-op-1"},
	})
	if parseErr != nil {
		parseT.Fatalf("ApplyReconnectConflictResolution: %v", parseErr)
	}
	if parseResult.UnsentMovedCount != 1 || parseResult.SettingsUpdatedCount != 1 || parseResult.PreferenceAckedCount != 1 || !parseResult.IsThreadUpdated {
		parseT.Fatalf("unexpected conflict-resolution result: %+v", parseResult)
	}

	parseNewChatQueue, parseErr := GetUnsentMessageQueue(parseCtx, parseStorage, parseScopeKey, "")
	if parseErr != nil {
		parseT.Fatalf("GetUnsentMessageQueue(new-chat): %v", parseErr)
	}
	if len(parseNewChatQueue) != 0 {
		parseT.Fatalf("expected new-chat queue drained after route normalization, got %+v", parseNewChatQueue)
	}
	parseThreadQueue, parseErr := GetUnsentMessageQueue(parseCtx, parseStorage, parseScopeKey, "thread-123")
	if parseErr != nil {
		parseT.Fatalf("GetUnsentMessageQueue(thread): %v", parseErr)
	}
	if len(parseThreadQueue) != 2 {
		parseT.Fatalf("expected deduped thread queue with two rows, got %+v", parseThreadQueue)
	}

	parseSettingsRecord, isParseSettingsFound, parseErr := parseStorage.Get(parseCtx, parseSettingsScopedResourceKey)
	if parseErr != nil || !isParseSettingsFound {
		parseT.Fatalf("Get(settings): found=%t err=%v", isParseSettingsFound, parseErr)
	}
	if parseSettingsRecord.Status != cachecore.CacheStatusReady || parseSettingsRecord.Version != "v2" {
		parseT.Fatalf("expected authoritative settings merge, got %+v", parseSettingsRecord)
	}

	parsePreferenceQueue, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parsePreferenceQueueKey, func(parseQueue []cachecore.OutboxRecordEnvelope) ([]cachecore.OutboxRecordEnvelope, error) {
		return parseQueue, nil
	})
	if parseErr != nil {
		parseT.Fatalf("UpdateQueueAtomic(preference read): %v", parseErr)
	}
	if len(parsePreferenceQueue) != 0 {
		parseT.Fatalf("expected preference queue to be ack-cleared, got %+v", parsePreferenceQueue)
	}

	parseThreadRecordStored, isParseThreadFound, parseErr := parseStorage.Get(parseCtx, cachecore.BuildScopedResourceKey(parseScopeKey, BuildThreadHistoryResourceKey("thread-123")))
	if parseErr != nil || !isParseThreadFound {
		parseT.Fatalf("Get(thread): found=%t err=%v", isParseThreadFound, parseErr)
	}
	if parseThreadRecordStored.Version != "thread-v2" {
		parseT.Fatalf("expected authoritative thread version, got %+v", parseThreadRecordStored)
	}
}
