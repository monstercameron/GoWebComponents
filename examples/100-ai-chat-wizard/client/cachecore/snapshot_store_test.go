package cachecore

import (
	"context"
	"testing"
	"time"
)

// TestSnapshotStoreSyncReadWrite verifies synchronous snapshot store read/write/delete behavior.
func TestSnapshotStoreSyncReadWrite(parseT *testing.T) {
	parseStore := BuildSnapshotStore()
	parseScopeKey := BuildScopeKey(ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1", Locale: "en-US", RouteKey: "/app", ThreadKey: "thread-1"})
	parseRecord := BuildCacheRecordEnvelope(parseScopeKey, "resource.profile", "v1", "etag-1", time.Now().UTC(), 30*time.Second, 120*time.Second, []byte(`{"name":"sam"}`), CacheStatusReady, "")
	parseStore.SetSync(parseRecord)
	parseLoadedRecord, isParseFound := parseStore.GetSync(parseScopeKey, "resource.profile")
	if !isParseFound {
		parseT.Fatal("expected mirrored snapshot record")
	}
	if parseLoadedRecord.Version != "v1" {
		parseT.Fatalf("unexpected snapshot record payload: %+v", parseLoadedRecord)
	}
	parseStore.DeleteSync(parseScopeKey, "resource.profile")
	if _, isParseFound = parseStore.GetSync(parseScopeKey, "resource.profile"); isParseFound {
		parseT.Fatal("expected snapshot record deletion")
	}
}

// TestSnapshotStoreApplyFromStorage verifies mirror hydration from storage.
func TestSnapshotStoreApplyFromStorage(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := BuildMemoryStorage()
	parseSnapshotStore := BuildSnapshotStore()
	parseScopeKey := BuildScopeKey(ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1", Locale: "en-US", RouteKey: "/app", ThreadKey: "thread-1"})
	parseRecord := BuildCacheRecordEnvelope(parseScopeKey, "resource.settings", "v1", "etag-1", time.Now().UTC(), 30*time.Second, 120*time.Second, []byte(`{"theme":"dark"}`), CacheStatusReady, "")
	if parseErr := parseStorage.Set(parseCtx, parseRecord); parseErr != nil {
		parseT.Fatalf("Set(storage seed): %v", parseErr)
	}
	if parseErr := parseSnapshotStore.ApplyFromStorage(parseCtx, parseStorage, parseScopeKey); parseErr != nil {
		parseT.Fatalf("ApplyFromStorage: %v", parseErr)
	}
	parseMirroredRecords := parseSnapshotStore.ListSync(parseScopeKey)
	if len(parseMirroredRecords) != 1 {
		parseT.Fatalf("expected one mirrored record, got %d", len(parseMirroredRecords))
	}
}
