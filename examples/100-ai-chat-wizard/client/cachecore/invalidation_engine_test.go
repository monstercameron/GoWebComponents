package cachecore

import (
	"context"
	"testing"
	"time"
)

// TestApplyInvalidationLogout verifies logout invalidation evicts matching scope records.
func TestApplyInvalidationLogout(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := BuildMemoryStorage()
	parseScope := BuildScopeKey(ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1", Locale: "en-US", RouteKey: "/app", ThreadKey: "thread-1"})
	parseRecord := BuildCacheRecordEnvelope(parseScope, "resource.profile", "v1", "etag", time.Now().UTC(), 30*time.Second, 120*time.Second, []byte(`{"name":"sam"}`), CacheStatusReady, "")
	if parseErr := parseStorage.Set(parseCtx, parseRecord); parseErr != nil {
		parseT.Fatalf("Set(record): %v", parseErr)
	}
	parseEngine := BuildInvalidationEngine(parseStorage, BuildInvalidator())
	parseAffectedCount, parseErr := parseEngine.ApplyInvalidation(parseCtx, InvalidationEvent{
		Reason:      InvalidationReasonLogout,
		ScopePrefix: parseScope,
	})
	if parseErr != nil {
		parseT.Fatalf("ApplyInvalidation(logout): %v", parseErr)
	}
	if parseAffectedCount != 1 {
		parseT.Fatalf("expected one record evicted, got %d", parseAffectedCount)
	}
	parseRecords, parseErr := parseStorage.List(parseCtx, "")
	if parseErr != nil || len(parseRecords) != 0 {
		parseT.Fatalf("expected empty storage after logout invalidation, len=%d err=%v", len(parseRecords), parseErr)
	}
}

// TestApplyInvalidationServerVersionMismatch verifies version mismatch downgrades records to stale.
func TestApplyInvalidationServerVersionMismatch(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := BuildMemoryStorage()
	parseScope := BuildScopeKey(ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1", Locale: "en-US", RouteKey: "/app", ThreadKey: "thread-1"})
	parseRecord := BuildCacheRecordEnvelope(parseScope, "resource.catalog", "v1", "etag-1", time.Now().UTC(), 30*time.Second, 120*time.Second, []byte(`{"models":1}`), CacheStatusReady, "")
	if parseErr := parseStorage.Set(parseCtx, parseRecord); parseErr != nil {
		parseT.Fatalf("Set(record): %v", parseErr)
	}
	parseEngine := BuildInvalidationEngine(parseStorage, BuildInvalidator())
	parseAffectedCount, parseErr := parseEngine.ApplyInvalidation(parseCtx, InvalidationEvent{
		Reason:      InvalidationReasonServerVersionMismatch,
		ScopePrefix: parseScope,
	})
	if parseErr != nil {
		parseT.Fatalf("ApplyInvalidation(server mismatch): %v", parseErr)
	}
	if parseAffectedCount != 1 {
		parseT.Fatalf("expected one stale downgrade, got %d", parseAffectedCount)
	}
	parseScopedResourceKey := BuildScopedResourceKey(parseScope, "resource.catalog")
	parseLoadedRecord, isParseFound, parseErr := parseStorage.Get(parseCtx, parseScopedResourceKey)
	if parseErr != nil || !isParseFound {
		parseT.Fatalf("Get(record after downgrade): found=%v err=%v", isParseFound, parseErr)
	}
	if parseLoadedRecord.Status != CacheStatusStale || parseLoadedRecord.ETagOrHash != "" {
		parseT.Fatalf("expected stale downgraded record, got %+v", parseLoadedRecord)
	}
}

// TestApplyInvalidationMutationAckResourcePrefix verifies mutation-ack invalidation can target one resource prefix.
func TestApplyInvalidationMutationAckResourcePrefix(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := BuildMemoryStorage()
	parseScope := BuildScopeKey(ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1", Locale: "en-US", RouteKey: "/app", ThreadKey: "thread-1"})
	parseSettingsRecord := BuildCacheRecordEnvelope(parseScope, "resource.settings", "v1", "etag-s", time.Now().UTC(), 30*time.Second, 120*time.Second, []byte(`{"theme":"x"}`), CacheStatusReady, "")
	parseProfileRecord := BuildCacheRecordEnvelope(parseScope, "resource.profile", "v1", "etag-p", time.Now().UTC(), 30*time.Second, 120*time.Second, []byte(`{"name":"x"}`), CacheStatusReady, "")
	if parseErr := parseStorage.Set(parseCtx, parseSettingsRecord); parseErr != nil {
		parseT.Fatalf("Set(settings): %v", parseErr)
	}
	if parseErr := parseStorage.Set(parseCtx, parseProfileRecord); parseErr != nil {
		parseT.Fatalf("Set(profile): %v", parseErr)
	}
	parseEngine := BuildInvalidationEngine(parseStorage, BuildInvalidator())
	parseAffectedCount, parseErr := parseEngine.ApplyInvalidation(parseCtx, InvalidationEvent{
		Reason:         InvalidationReasonMutationAck,
		ScopePrefix:    parseScope,
		ResourcePrefix: "resource.settings",
	})
	if parseErr != nil {
		parseT.Fatalf("ApplyInvalidation(mutation ack): %v", parseErr)
	}
	if parseAffectedCount != 1 {
		parseT.Fatalf("expected one resource-prefix record deleted, got %d", parseAffectedCount)
	}
	if _, isParseFound, _ := parseStorage.Get(parseCtx, BuildScopedResourceKey(parseScope, "resource.settings")); isParseFound {
		parseT.Fatal("expected settings record to be deleted")
	}
	if _, isParseFound, _ := parseStorage.Get(parseCtx, BuildScopedResourceKey(parseScope, "resource.profile")); !isParseFound {
		parseT.Fatal("expected profile record to remain")
	}
}
