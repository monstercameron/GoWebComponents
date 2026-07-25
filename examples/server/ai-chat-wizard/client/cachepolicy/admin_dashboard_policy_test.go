package cachepolicy

import (
	"context"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/client/cachecore"
)

// TestBuildAdminDashboardResourceKey verifies role/surface/lookback cache key dimensions.
func TestBuildAdminDashboardResourceKey(parseT *testing.T) {
	parseKey := BuildAdminDashboardResourceKey("superuser", "ops", 14)
	parseExpected := "admin.dashboard.snapshot|role=superuser|surface=ops|lookback_days=14"
	if parseKey != parseExpected {
		parseT.Fatalf("unexpected key: %q != %q", parseKey, parseExpected)
	}
}

// TestStoreReadAdminDashboardSnapshot verifies short-lived dashboard snapshots restore quickly and refresh in background.
func TestStoreReadAdminDashboardSnapshot(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseAPI := cachecore.BuildUIAPIWithSnapshot(parseStorage, parseSnapshot, cachecore.BuildSWRCoordinator(), cachecore.BuildInvalidationEngine(parseStorage, cachecore.BuildInvalidator()))
	parseInput := AdminDashboardSnapshotInput{
		ScopeKey:     cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"}),
		RoleScope:    "superuser",
		Surface:      "ops",
		LookbackDays: 7,
	}
	parsePolicy := BuildAdminDashboardSnapshotPolicy()
	parseResourceKey := BuildAdminDashboardResourceKey(parseInput.RoleScope, parseInput.Surface, parseInput.LookbackDays)
	parsePayloadJSON, parseErr := BuildAdminDashboardSnapshotPayloadJSON(AdminDashboardSnapshotPayload{
		LastUpdated: time.Now().UTC().Format(time.RFC3339),
		Payload:     []byte(`{"kpi":{"incidents_open":1}}`),
	})
	if parseErr != nil {
		parseT.Fatalf("BuildAdminDashboardSnapshotPayloadJSON: %v", parseErr)
	}
	parseStaleRecord := cachecore.BuildCacheRecordEnvelope(parseInput.ScopeKey, parseResourceKey, "", "", time.Now().UTC().Add(-2*time.Minute), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, parsePayloadJSON, cachecore.CacheStatusReady, "")
	if parseErr = parseStorage.Set(parseCtx, parseStaleRecord); parseErr != nil {
		parseT.Fatalf("Set(seed): %v", parseErr)
	}
	parseSnapshot.SetSync(parseStaleRecord)
	parseRefreshDone := make(chan struct{}, 1)
	parseView, parseErr := ReadAdminDashboardSnapshot(parseCtx, parseAPI, parseInput, func(parseCtx context.Context, parseInput AdminDashboardSnapshotInput) (cachecore.CacheRecordEnvelope, error) {
		_ = parseCtx
		parseRefreshDone <- struct{}{}
		return cachecore.BuildCacheRecordEnvelope(parseInput.ScopeKey, BuildAdminDashboardResourceKey(parseInput.RoleScope, parseInput.Surface, parseInput.LookbackDays), "", "", time.Now().UTC(), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, []byte(`{"last_updated":"now"}`), cachecore.CacheStatusReady, ""), nil
	})
	if parseErr != nil {
		parseT.Fatalf("ReadAdminDashboardSnapshot: %v", parseErr)
	}
	if !parseView.IsFound || !parseView.IsCached || !parseView.IsStale || !parseView.RefreshTriggered {
		parseT.Fatalf("unexpected dashboard snapshot view: %+v", parseView)
	}
	select {
	case <-parseRefreshDone:
	case <-time.After(2 * time.Second):
		parseT.Fatal("timed out waiting for admin dashboard background refresh")
	}
}

// TestApplyAdminDashboardHardInvalidation verifies role/session/workspace hard invalidation behavior.
func TestApplyAdminDashboardHardInvalidation(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})
	parsePolicy := BuildAdminDashboardSnapshotPolicy()
	parseSetRecord := func(parseResourceKey string) {
		parseRecord := cachecore.BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "", "", time.Now().UTC(), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, []byte(`{"ok":1}`), cachecore.CacheStatusReady, "")
		if parseErr := parseStorage.Set(parseCtx, parseRecord); parseErr != nil {
			parseT.Fatalf("Set(%s): %v", parseResourceKey, parseErr)
		}
		parseSnapshot.SetSync(parseRecord)
	}
	parseSetRecord(BuildAdminDashboardResourceKey("superuser", "business", 7))
	parseSetRecord(BuildAdminDashboardResourceKey("superuser", "ops", 7))
	parseSetRecord("conversation_list.page|cursor=|size=20")

	parseEvictedCount, parseErr := ApplyAdminDashboardHardInvalidation(parseCtx, parseStorage, parseSnapshot)
	if parseErr != nil {
		parseT.Fatalf("ApplyAdminDashboardHardInvalidation: %v", parseErr)
	}
	if parseEvictedCount != 2 {
		parseT.Fatalf("expected 2 evicted admin dashboard records, got %d", parseEvictedCount)
	}
	parseRemaining, parseErr := parseStorage.List(parseCtx, parseScopeKey)
	if parseErr != nil {
		parseT.Fatalf("List(remaining): %v", parseErr)
	}
	if len(parseRemaining) != 1 || parseRemaining[0].ResourceKey != "conversation_list.page|cursor=|size=20" {
		parseT.Fatalf("unexpected remaining records after hard invalidation: %+v", parseRemaining)
	}
}
