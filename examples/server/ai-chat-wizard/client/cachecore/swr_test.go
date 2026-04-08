package cachecore

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestReadWithSWRStaleSingleFlight verifies stale reads trigger one deduplicated background refresh.
func TestReadWithSWRStaleSingleFlight(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := BuildMemoryStorage()
	parseCoordinator := BuildSWRCoordinator()
	parseScopeKey := BuildScopeKey(ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1", Locale: "en-US", RouteKey: "/app", ThreadKey: "thread-1"})
	parseRecord := BuildCacheRecordEnvelope(
		parseScopeKey,
		"resource.settings",
		"v1",
		"etag-1",
		time.Now().UTC().Add(-2*time.Minute),
		30*time.Second,
		5*time.Minute,
		[]byte(`{"cached":true}`),
		CacheStatusReady,
		"",
	)
	if parseErr := parseStorage.Set(parseCtx, parseRecord); parseErr != nil {
		parseT.Fatalf("Set(stale record): %v", parseErr)
	}
	parsePolicy := BuildCachePolicy(PolicyClassSession, 30*time.Second, 5*time.Minute, true, true, false)
	var parseRefreshCount int32
	parseRefreshDone := make(chan struct{}, 1)
	parseRefresh := func(parseCtx context.Context, parseScopeKey string, parseResourceKey string) (CacheRecordEnvelope, error) {
		_ = parseCtx
		atomic.AddInt32(&parseRefreshCount, 1)
		parseRefreshDone <- struct{}{}
		return BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "v2", "etag-2", time.Now().UTC(), 30*time.Second, 5*time.Minute, []byte(`{"cached":false}`), CacheStatusReady, ""), nil
	}
	parseResultOne, parseErr := ReadWithSWR(parseCtx, parseStorage, parseCoordinator, parsePolicy, parseScopeKey, "resource.settings", parseRefresh)
	if parseErr != nil {
		parseT.Fatalf("ReadWithSWR(first): %v", parseErr)
	}
	parseResultTwo, parseErr := ReadWithSWR(parseCtx, parseStorage, parseCoordinator, parsePolicy, parseScopeKey, "resource.settings", parseRefresh)
	if parseErr != nil {
		parseT.Fatalf("ReadWithSWR(second): %v", parseErr)
	}
	if !parseResultOne.IsFound || !parseResultOne.IsCached || !parseResultOne.IsStale || !parseResultOne.RefreshTriggered {
		parseT.Fatalf("unexpected first SWR read result: %+v", parseResultOne)
	}
	if !parseResultTwo.IsFound || !parseResultTwo.IsCached || !parseResultTwo.IsStale {
		parseT.Fatalf("unexpected second SWR read result: %+v", parseResultTwo)
	}
	select {
	case <-parseRefreshDone:
	case <-time.After(2 * time.Second):
		parseT.Fatal("timed out waiting for background refresh")
	}
	if parseCount := atomic.LoadInt32(&parseRefreshCount); parseCount != 1 {
		parseT.Fatalf("expected one background refresh invocation, got %d", parseCount)
	}
}

// TestReadWithSWRMissingTriggersRefresh verifies missing reads can trigger one refresh when policy allows.
func TestReadWithSWRMissingTriggersRefresh(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := BuildMemoryStorage()
	parseCoordinator := BuildSWRCoordinator()
	parseScopeKey := BuildScopeKey(ScopeKey{AppVersion: "v1", SessionID: "sid-2", UserID: "u-1", Workspace: "w-1", Locale: "en-US", RouteKey: "/app", ThreadKey: "thread-2"})
	parsePolicy := BuildCachePolicy(PolicyClassSession, 30*time.Second, 5*time.Minute, true, true, false)
	parseRefreshDone := make(chan struct{}, 1)
	parseRefresh := func(parseCtx context.Context, parseScopeKey string, parseResourceKey string) (CacheRecordEnvelope, error) {
		_ = parseCtx
		parseRefreshDone <- struct{}{}
		return BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "v1", "etag-1", time.Now().UTC(), 30*time.Second, 5*time.Minute, []byte(`{"value":1}`), CacheStatusReady, ""), nil
	}
	parseResult, parseErr := ReadWithSWR(parseCtx, parseStorage, parseCoordinator, parsePolicy, parseScopeKey, "resource.profile", parseRefresh)
	if parseErr != nil {
		parseT.Fatalf("ReadWithSWR(missing): %v", parseErr)
	}
	if parseResult.IsFound || parseResult.Freshness != FreshnessMissing || !parseResult.RefreshTriggered {
		parseT.Fatalf("unexpected missing SWR read result: %+v", parseResult)
	}
	select {
	case <-parseRefreshDone:
	case <-time.After(2 * time.Second):
		parseT.Fatal("timed out waiting for missing-resource refresh")
	}
}
