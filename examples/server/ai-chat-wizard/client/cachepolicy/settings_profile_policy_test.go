package cachepolicy

import (
	"context"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/client/cachecore"
)

// TestStoreReadSettingsSnapshot verifies settings snapshots bootstrap quickly with snapshot-first reads and SWR refresh.
func TestStoreReadSettingsSnapshot(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseAPI := cachecore.BuildUIAPIWithSnapshot(parseStorage, parseSnapshot, cachecore.BuildSWRCoordinator(), cachecore.BuildInvalidationEngine(parseStorage, cachecore.BuildInvalidator()))
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})
	parseSection := settingsSnapshotSectionProfile
	parseResourceKey := BuildSettingsSnapshotResourceKey(parseSection)
	parsePolicy := BuildSettingsSnapshotPolicy()

	parseStaleRecord := cachecore.BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "v1", "hash-v1", time.Now().UTC().Add(-5*time.Minute), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, []byte(`{"display_name":"sam"}`), cachecore.CacheStatusReady, "")
	if parseErr := parseStorage.Set(parseCtx, parseStaleRecord); parseErr != nil {
		parseT.Fatalf("Set(seed): %v", parseErr)
	}
	parseSnapshot.SetSync(parseStaleRecord)
	parseRefreshDone := make(chan struct{}, 1)
	parseView, parseErr := ReadSettingsSnapshot(parseCtx, parseAPI, parseScopeKey, parseSection, func(parseCtx context.Context, parseScopeKey string, parseResourceKey string) (cachecore.CacheRecordEnvelope, error) {
		_ = parseCtx
		parseRefreshDone <- struct{}{}
		return cachecore.BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "v2", "hash-v2", time.Now().UTC(), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, []byte(`{"display_name":"sam updated"}`), cachecore.CacheStatusReady, ""), nil
	})
	if parseErr != nil {
		parseT.Fatalf("ReadSettingsSnapshot: %v", parseErr)
	}
	if !parseView.IsFound || !parseView.IsCached || !parseView.IsStale || !parseView.RefreshTriggered {
		parseT.Fatalf("unexpected settings snapshot view: %+v", parseView)
	}
	select {
	case <-parseRefreshDone:
	case <-time.After(2 * time.Second):
		parseT.Fatal("timed out waiting for settings snapshot background refresh")
	}
}

// TestApplySettingsSnapshotWriteSuccessAndFailure verifies write-success merge and write-failure stale/unavailable behavior.
func TestApplySettingsSnapshotWriteSuccessAndFailure(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})
	parseSection := settingsSnapshotSectionPrompt
	parseResourceKey := BuildSettingsSnapshotResourceKey(parseSection)
	parseScopedResourceKey := cachecore.BuildScopedResourceKey(parseScopeKey, parseResourceKey)

	if parseErr := ApplySettingsSnapshotWriteSuccess(parseCtx, parseStorage, parseSnapshot, parseScopeKey, parseSection, "v1", []byte(`{"prompt":"be concise"}`)); parseErr != nil {
		parseT.Fatalf("ApplySettingsSnapshotWriteSuccess: %v", parseErr)
	}
	parseRecordAfterSuccess, isParseFound, parseErr := parseStorage.Get(parseCtx, parseScopedResourceKey)
	if parseErr != nil || !isParseFound {
		parseT.Fatalf("Get(after success): found=%t err=%v", isParseFound, parseErr)
	}
	if parseRecordAfterSuccess.Status != cachecore.CacheStatusReady || parseRecordAfterSuccess.Version != "v1" {
		parseT.Fatalf("unexpected record after write success: %+v", parseRecordAfterSuccess)
	}

	if parseErr = ApplySettingsSnapshotWriteFailure(parseCtx, parseStorage, parseSnapshot, parseScopeKey, parseSection, "store unavailable"); parseErr != nil {
		parseT.Fatalf("ApplySettingsSnapshotWriteFailure: %v", parseErr)
	}
	parseRecordAfterFailure, isParseFound, parseErr := parseStorage.Get(parseCtx, parseScopedResourceKey)
	if parseErr != nil || !isParseFound {
		parseT.Fatalf("Get(after failure): found=%t err=%v", isParseFound, parseErr)
	}
	if parseRecordAfterFailure.Status != cachecore.CacheStatusStale || parseRecordAfterFailure.Error != "store unavailable" {
		parseT.Fatalf("unexpected record after write failure: %+v", parseRecordAfterFailure)
	}
	parseSnapshotRecord, isParseSnapshotFound := parseSnapshot.GetSync(parseScopeKey, parseResourceKey)
	if !isParseSnapshotFound || parseSnapshotRecord.Status != cachecore.CacheStatusStale {
		parseT.Fatalf("expected stale snapshot mirror after write failure, got found=%t record=%+v", isParseSnapshotFound, parseSnapshotRecord)
	}
}
