package cachepolicy

import (
	"context"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/client/cachecore"
)

// TestStoreReadConversationListPage verifies sidebar page cache restores has-more and anchor metadata with snapshot-first reads.
func TestStoreReadConversationListPage(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseAPI := cachecore.BuildUIAPIWithSnapshot(parseStorage, parseSnapshot, cachecore.BuildSWRCoordinator(), cachecore.BuildInvalidationEngine(parseStorage, cachecore.BuildInvalidator()))
	parseInput := ConversationListPageInput{
		ScopeKey:   cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"}),
		PageCursor: "page-1",
		PageSize:   20,
	}

	if parseErr := StoreConversationListPage(parseCtx, parseStorage, parseInput, "bundle-v1", []byte(`{"conversations":[{"public_id":"c1"}]}`), true, "c1"); parseErr != nil {
		parseT.Fatalf("StoreConversationListPage: %v", parseErr)
	}
	parseStoredRecord, isParseFound, parseErr := parseStorage.Get(parseCtx, cachecore.BuildScopedResourceKey(parseInput.ScopeKey, BuildConversationListPageResourceKey(parseInput.PageCursor, parseInput.PageSize)))
	if parseErr != nil || !isParseFound {
		parseT.Fatalf("Get(stored page): found=%t err=%v", isParseFound, parseErr)
	}
	parsePayload, parseErr := ParseConversationListPagePayloadJSON(parseStoredRecord.Payload)
	if parseErr != nil {
		parseT.Fatalf("ParseConversationListPagePayloadJSON: %v", parseErr)
	}
	if !parsePayload.HasMore || parsePayload.AnchorConversationPublicID != "c1" {
		parseT.Fatalf("unexpected stored page payload metadata: %+v", parsePayload)
	}

	parsePolicy := BuildConversationListPagePolicy()
	parseStaleRecord := cachecore.BuildCacheRecordEnvelope(parseInput.ScopeKey, BuildConversationListPageResourceKey(parseInput.PageCursor, parseInput.PageSize), "bundle-v1", "", time.Now().UTC().Add(-10*time.Minute), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, parseStoredRecord.Payload, cachecore.CacheStatusReady, "")
	parseSnapshot.SetSync(parseStaleRecord)
	parseRefreshDone := make(chan struct{}, 1)
	parseView, parseErr := ReadConversationListPage(parseCtx, parseAPI, parseInput, func(parseCtx context.Context, parseInput ConversationListPageInput) (cachecore.CacheRecordEnvelope, error) {
		_ = parseCtx
		parseRefreshDone <- struct{}{}
		return cachecore.BuildCacheRecordEnvelope(parseInput.ScopeKey, BuildConversationListPageResourceKey(parseInput.PageCursor, parseInput.PageSize), "bundle-v2", "", time.Now().UTC(), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, []byte(`{"has_more":false}`), cachecore.CacheStatusReady, ""), nil
	})
	if parseErr != nil {
		parseT.Fatalf("ReadConversationListPage: %v", parseErr)
	}
	if !parseView.IsFound || !parseView.IsCached || !parseView.IsStale || !parseView.RefreshTriggered {
		parseT.Fatalf("unexpected page read view: %+v", parseView)
	}
	select {
	case <-parseRefreshDone:
	case <-time.After(2 * time.Second):
		parseT.Fatal("timed out waiting for conversation-list page background refresh")
	}
}

// TestStoreReadConversationListScrollAnchor verifies lightweight scroll-anchor restore behavior.
func TestStoreReadConversationListScrollAnchor(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseAPI := cachecore.BuildUIAPIWithSnapshot(parseStorage, parseSnapshot, cachecore.BuildSWRCoordinator(), cachecore.BuildInvalidationEngine(parseStorage, cachecore.BuildInvalidator()))
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})
	parseRouteKey := "/app/thread/c1"

	if parseErr := StoreConversationListScrollAnchor(parseCtx, parseStorage, parseScopeKey, parseRouteKey, []byte(`{"scroll_top":420,"anchor":"c1"}`)); parseErr != nil {
		parseT.Fatalf("StoreConversationListScrollAnchor: %v", parseErr)
	}
	parseStoredRecord, isParseFound, parseErr := parseStorage.Get(parseCtx, cachecore.BuildScopedResourceKey(parseScopeKey, BuildConversationListAnchorResourceKey(parseRouteKey)))
	if parseErr != nil || !isParseFound {
		parseT.Fatalf("Get(stored anchor): found=%t err=%v", isParseFound, parseErr)
	}
	parseSnapshot.SetSync(parseStoredRecord)
	parseView, parseErr := ReadConversationListScrollAnchor(parseCtx, parseAPI, parseScopeKey, parseRouteKey, nil)
	if parseErr != nil {
		parseT.Fatalf("ReadConversationListScrollAnchor: %v", parseErr)
	}
	if !parseView.IsFound || !parseView.IsCached {
		parseT.Fatalf("expected cached anchor view, got %+v", parseView)
	}
}
