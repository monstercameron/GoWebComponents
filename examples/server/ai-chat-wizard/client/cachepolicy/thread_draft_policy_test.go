package cachepolicy

import (
	"context"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/client/cachecore"
)

// TestStoreAndReadThreadDraft verifies thread drafts store and restore independently from unsent message queues.
func TestStoreAndReadThreadDraft(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseAPI := cachecore.BuildUIAPIWithSnapshot(parseStorage, parseSnapshot, cachecore.BuildSWRCoordinator(), cachecore.BuildInvalidationEngine(parseStorage, cachecore.BuildInvalidator()))
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})
	parseThreadKey := "thread-123"

	if parseErr := StoreThreadDraft(parseCtx, parseStorage, parseScopeKey, parseThreadKey, []byte(`{"text":"draft body"}`)); parseErr != nil {
		parseT.Fatalf("StoreThreadDraft: %v", parseErr)
	}
	parseStoredRecord, isParseStored, parseErr := parseStorage.Get(parseCtx, cachecore.BuildScopedResourceKey(parseScopeKey, BuildThreadDraftResourceKey(parseThreadKey)))
	if parseErr != nil || !isParseStored {
		parseT.Fatalf("Get(stored): found=%t err=%v", isParseStored, parseErr)
	}
	parseSnapshot.SetSync(parseStoredRecord)

	parseView, parseErr := ReadThreadDraft(parseCtx, parseAPI, parseScopeKey, parseThreadKey, nil)
	if parseErr != nil {
		parseT.Fatalf("ReadThreadDraft: %v", parseErr)
	}
	if !parseView.IsFound || !parseView.IsCached {
		parseT.Fatalf("expected cached thread draft view, got %+v", parseView)
	}
}

// TestApplyThreadDraftDiscardRules verifies send/clear discard paths remove draft cache state from storage and snapshot.
func TestApplyThreadDraftDiscardRules(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})
	parseThreadKey := "thread-123"
	parseResourceKey := BuildThreadDraftResourceKey(parseThreadKey)

	if parseErr := StoreThreadDraft(parseCtx, parseStorage, parseScopeKey, parseThreadKey, []byte(`{"text":"draft body"}`)); parseErr != nil {
		parseT.Fatalf("StoreThreadDraft(initial): %v", parseErr)
	}
	parseSeedRecord, isParseSeeded, parseErr := parseStorage.Get(parseCtx, cachecore.BuildScopedResourceKey(parseScopeKey, parseResourceKey))
	if parseErr != nil || !isParseSeeded {
		parseT.Fatalf("Get(seed): found=%t err=%v", isParseSeeded, parseErr)
	}
	parseSnapshot.SetSync(parseSeedRecord)

	parseRemovedAfterSend, parseErr := ApplyThreadDraftDiscardAfterSend(parseCtx, parseStorage, parseSnapshot, parseScopeKey, parseThreadKey)
	if parseErr != nil {
		parseT.Fatalf("ApplyThreadDraftDiscardAfterSend: %v", parseErr)
	}
	if parseRemovedAfterSend != 1 {
		parseT.Fatalf("expected send discard to remove one draft, got %d", parseRemovedAfterSend)
	}
	if _, isParseFound := parseSnapshot.GetSync(parseScopeKey, parseResourceKey); isParseFound {
		parseT.Fatalf("expected snapshot draft to be removed after send discard")
	}

	if parseErr = StoreThreadDraft(parseCtx, parseStorage, parseScopeKey, parseThreadKey, []byte(`{"text":"draft body again"}`)); parseErr != nil {
		parseT.Fatalf("StoreThreadDraft(second): %v", parseErr)
	}
	parseSeedRecord, _, _ = parseStorage.Get(parseCtx, cachecore.BuildScopedResourceKey(parseScopeKey, parseResourceKey))
	parseSnapshot.SetSync(parseSeedRecord)
	parseRemovedAfterClear, parseErr := ApplyThreadDraftDiscardByUserClear(parseCtx, parseStorage, parseSnapshot, parseScopeKey, parseThreadKey)
	if parseErr != nil {
		parseT.Fatalf("ApplyThreadDraftDiscardByUserClear: %v", parseErr)
	}
	if parseRemovedAfterClear != 1 {
		parseT.Fatalf("expected user-clear discard to remove one draft, got %d", parseRemovedAfterClear)
	}
	parseRecordAfterClear, isParseFound, parseErr := parseStorage.Get(parseCtx, cachecore.BuildScopedResourceKey(parseScopeKey, parseResourceKey))
	if parseErr != nil {
		parseT.Fatalf("Get(after-clear): %v", parseErr)
	}
	if isParseFound {
		parseT.Fatalf("expected no stored thread draft after clear, got %+v", parseRecordAfterClear)
	}
}
