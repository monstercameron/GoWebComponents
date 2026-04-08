package cachepolicy

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/client/cachecore"
)

// TestBuildParseCanvasSessionSnapshotPayloadJSON verifies persisted canvas snapshots keep lightweight fields and exclude debug state.
func TestBuildParseCanvasSessionSnapshotPayloadJSON(parseT *testing.T) {
	parseRaw, parseErr := BuildCanvasSessionSnapshotPayloadJSON(CanvasSessionSnapshotPayload{
		PaneWidthRatio: 0.1,
		SelectedItemID: " item-1 ",
		SessionState:   []byte(`{"selection":"item-1"}`),
		DebugState:     []byte(`{"trace":"debug-only"}`),
	})
	if parseErr != nil {
		parseT.Fatalf("BuildCanvasSessionSnapshotPayloadJSON: %v", parseErr)
	}
	if bytes.Contains(parseRaw, []byte("debug-only")) {
		parseT.Fatalf("expected debug state to be excluded from persisted payload: %s", string(parseRaw))
	}
	parsePayload, parseErr := ParseCanvasSessionSnapshotPayloadJSON(parseRaw)
	if parseErr != nil {
		parseT.Fatalf("ParseCanvasSessionSnapshotPayloadJSON: %v", parseErr)
	}
	if parsePayload.PaneWidthRatio != 0.2 || parsePayload.SelectedItemID != "item-1" {
		parseT.Fatalf("unexpected payload normalization: %+v", parsePayload)
	}
}

// TestStoreReadCanvasSessionSnapshot verifies canvas/session snapshots restore quickly and trigger refresh when stale.
func TestStoreReadCanvasSessionSnapshot(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseAPI := cachecore.BuildUIAPIWithSnapshot(parseStorage, parseSnapshot, cachecore.BuildSWRCoordinator(), cachecore.BuildInvalidationEngine(parseStorage, cachecore.BuildInvalidator()))
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})
	parseRouteKey := "/app/thread/c1"
	parsePolicy := BuildCanvasSessionSnapshotPolicy()

	if parseErr := StoreCanvasSessionSnapshot(parseCtx, parseStorage, parseScopeKey, parseRouteKey, CanvasSessionSnapshotPayload{
		PaneWidthRatio: 0.6,
		SelectedItemID: "canvas-item-1",
		SessionState:   []byte(`{"pane":"canvas"}`),
	}); parseErr != nil {
		parseT.Fatalf("StoreCanvasSessionSnapshot: %v", parseErr)
	}
	parseStoredRecord, isParseFound, parseErr := parseStorage.Get(parseCtx, cachecore.BuildScopedResourceKey(parseScopeKey, BuildCanvasSessionResourceKey(parseRouteKey)))
	if parseErr != nil || !isParseFound {
		parseT.Fatalf("Get(stored): found=%t err=%v", isParseFound, parseErr)
	}
	parseStaleRecord := cachecore.BuildCacheRecordEnvelope(parseScopeKey, BuildCanvasSessionResourceKey(parseRouteKey), "", "", time.Now().UTC().Add(-5*time.Minute), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, parseStoredRecord.Payload, cachecore.CacheStatusReady, "")
	parseSnapshot.SetSync(parseStaleRecord)
	parseRefreshDone := make(chan struct{}, 1)
	parseView, parseErr := ReadCanvasSessionSnapshot(parseCtx, parseAPI, parseScopeKey, parseRouteKey, func(parseCtx context.Context, parseScopeKey string, parseResourceKey string) (cachecore.CacheRecordEnvelope, error) {
		_ = parseCtx
		parseRefreshDone <- struct{}{}
		return cachecore.BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "", "", time.Now().UTC(), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, []byte(`{"pane_width_ratio":0.55}`), cachecore.CacheStatusReady, ""), nil
	})
	if parseErr != nil {
		parseT.Fatalf("ReadCanvasSessionSnapshot: %v", parseErr)
	}
	if !parseView.IsFound || !parseView.IsCached || !parseView.IsStale || !parseView.RefreshTriggered {
		parseT.Fatalf("unexpected canvas snapshot view: %+v", parseView)
	}
	select {
	case <-parseRefreshDone:
	case <-time.After(2 * time.Second):
		parseT.Fatal("timed out waiting for canvas snapshot background refresh")
	}
}
