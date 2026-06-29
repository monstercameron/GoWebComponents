package cachecore

import (
	"encoding/json"
	"testing"
	"time"
)

// TestBuildScopeKey verifies stable scope-key and scoped-resource-key formatting.
func TestBuildScopeKey(parseT *testing.T) {
	parseScopeKey := BuildScopeKey(ScopeKey{
		AppVersion: "v2026.03.28",
		SessionID:  "sid-123",
		UserID:     "42",
		Workspace:  "workspace-7",
		Locale:     "en-US",
		RouteKey:   "/app/thread/abc",
		ThreadKey:  "thread-public-123",
	})
	if parseScopeKey == "" {
		parseT.Fatal("expected non-empty scope key")
	}
	parseScopedResourceKey := BuildScopedResourceKey(parseScopeKey, "resource.thread")
	if parseScopedResourceKey == "" {
		parseT.Fatal("expected non-empty scoped resource key")
	}
}

// TestBuildScopePrefixes verifies deterministic eviction prefixes for logout/account/workspace/locale switches.
func TestBuildScopePrefixes(parseT *testing.T) {
	parseScope := ScopeKey{
		AppVersion: "v2026.03.28",
		SessionID:  "sid-123",
		UserID:     "42",
		Workspace:  "workspace-7",
		Locale:     "en-US",
		RouteKey:   "/app/thread/abc",
		ThreadKey:  "thread-public-123",
	}
	parseBaseKey := BuildScopeKey(parseScope)
	parseLogoutPrefix := BuildScopePrefixForLogout(parseScope)
	parseAccountPrefix := BuildScopePrefixForAccountSwitch(parseScope)
	parseWorkspacePrefix := BuildScopePrefixForWorkspaceSwitch(parseScope)
	parseLocalePrefix := BuildScopePrefixForLocaleSwitch(parseScope)
	if parseBaseKey == parseLogoutPrefix || parseBaseKey == parseAccountPrefix || parseBaseKey == parseWorkspacePrefix || parseBaseKey == parseLocalePrefix {
		parseT.Fatalf("expected eviction prefixes to differ from base scope key, base=%q logout=%q account=%q workspace=%q locale=%q", parseBaseKey, parseLogoutPrefix, parseAccountPrefix, parseWorkspacePrefix, parseLocalePrefix)
	}
}

// TestBuildFreshnessWindow verifies freshness timestamp generation and ordering.
func TestBuildFreshnessWindow(parseT *testing.T) {
	parseNow := time.Now().UTC().Truncate(time.Second)
	parseFetchedAt, parseStaleAt, parseExpiresAt := BuildFreshnessWindow(parseNow, 30*time.Second, 120*time.Second)
	if parseFetchedAt == "" || parseStaleAt == "" || parseExpiresAt == "" {
		parseT.Fatal("expected fetched/stale/expires timestamps")
	}
	parseFetchedAtTime, parseFetchedAtOk := parseParseRFC3339(parseFetchedAt)
	parseStaleAtTime, parseStaleAtOk := parseParseRFC3339(parseStaleAt)
	parseExpiresAtTime, parseExpiresAtOk := parseParseRFC3339(parseExpiresAt)
	if !parseFetchedAtOk || !parseStaleAtOk || !parseExpiresAtOk {
		parseT.Fatal("expected valid RFC3339 freshness values")
	}
	if parseStaleAtTime.Before(parseFetchedAtTime) || parseExpiresAtTime.Before(parseStaleAtTime) {
		parseT.Fatalf("unexpected freshness ordering fetched=%s stale=%s expires=%s", parseFetchedAt, parseStaleAt, parseExpiresAt)
	}
}

// TestResolveFreshnessState verifies fresh/stale/expired state resolution.
func TestResolveFreshnessState(parseT *testing.T) {
	parseNow := time.Now().UTC()
	parseFetchedAt, parseStaleAt, parseExpiresAt := BuildFreshnessWindow(parseNow.Add(-2*time.Minute), 30*time.Second, 3*time.Minute)
	parseState := ResolveFreshnessState(CacheRecordEnvelope{
		FetchedAt: parseFetchedAt,
		StaleAt:   parseStaleAt,
		ExpiresAt: parseExpiresAt,
	}, parseNow)
	if parseState != FreshnessStale {
		parseT.Fatalf("expected stale state, got %q", parseState)
	}
	parseExpiredState := ResolveFreshnessState(CacheRecordEnvelope{
		StaleAt:   parseNow.Add(-3 * time.Minute).Format(time.RFC3339),
		ExpiresAt: parseNow.Add(-1 * time.Minute).Format(time.RFC3339),
	}, parseNow)
	if parseExpiredState != FreshnessExpired {
		parseT.Fatalf("expected expired state, got %q", parseExpiredState)
	}
}

// TestStartRefresh ensures stale-while-revalidate refreshes are de-duplicated by scoped key.
func TestStartRefresh(parseT *testing.T) {
	parseCoordinator := BuildSWRCoordinator()
	parseRelease, isParseStarted := parseCoordinator.StartRefresh("scope-a", "resource-x")
	if !isParseStarted || parseRelease == nil {
		parseT.Fatal("expected first refresh to start")
	}
	if _, isParseStartedSecond := parseCoordinator.StartRefresh("scope-a", "resource-x"); isParseStartedSecond {
		parseT.Fatal("expected duplicate refresh to be suppressed")
	}
	parseRelease()
	if _, isParseStartedThird := parseCoordinator.StartRefresh("scope-a", "resource-x"); !isParseStartedThird {
		parseT.Fatal("expected refresh start after release")
	}
}

// TestApplyInvalidation ensures registered invalidation hooks receive broadcast events.
func TestApplyInvalidation(parseT *testing.T) {
	parseInvalidator := BuildInvalidator()
	parseEventCount := 0
	parseInvalidator.SetHook("test-hook", func(parseEvent InvalidationEvent) {
		if parseEvent.Reason == "" {
			parseT.Fatal("expected invalidation reason")
		}
		parseEventCount++
	})
	parseInvalidator.ApplyInvalidation(InvalidationEvent{
		Reason:         "logout",
		ScopePrefix:    "scope-a",
		ResourcePrefix: "resource-",
	})
	if parseEventCount != 1 {
		parseT.Fatalf("expected one invalidation event delivery, got %d", parseEventCount)
	}
}

// TestCacheRecordEnvelopeJSONContract verifies canonical cache-record persistence keys.
func TestCacheRecordEnvelopeJSONContract(parseT *testing.T) {
	parseRecord := BuildCacheRecordEnvelope(
		"scope-1",
		"resource-1",
		"v1",
		"etag-1",
		time.Now().UTC(),
		30*time.Second,
		120*time.Second,
		[]byte(`{"ok":true}`),
		CacheStatusReady,
		"",
	)
	parseJSON, parseErr := BuildCacheRecordEnvelopeJSON(parseRecord)
	if parseErr != nil {
		parseT.Fatalf("BuildCacheRecordEnvelopeJSON: %v", parseErr)
	}
	parseRaw := map[string]any{}
	if parseErr = json.Unmarshal(parseJSON, &parseRaw); parseErr != nil {
		parseT.Fatalf("json.Unmarshal: %v", parseErr)
	}
	parseRequiredKeys := []string{
		"scope_key",
		"resource_key",
		"version",
		"etag_or_hash",
		"fetched_at",
		"stale_at",
		"expires_at",
		"parsePayload",
		"parseStatus",
		"parseError",
	}
	for _, parseKey := range parseRequiredKeys {
		if _, isParsePresent := parseRaw[parseKey]; !isParsePresent {
			parseT.Fatalf("missing canonical cache-record key %q in %s", parseKey, string(parseJSON))
		}
	}
	parseParsedRecord, parseErr := ParseCacheRecordEnvelopeJSON(parseJSON)
	if parseErr != nil {
		parseT.Fatalf("ParseCacheRecordEnvelopeJSON: %v", parseErr)
	}
	if parseParsedRecord.ScopeKey != parseRecord.ScopeKey || parseParsedRecord.ResourceKey != parseRecord.ResourceKey {
		parseT.Fatalf("parsed record mismatch: got %+v want %+v", parseParsedRecord, parseRecord)
	}
}

// TestOutboxRecordEnvelopeJSONContract verifies canonical outbox-record persistence keys.
func TestOutboxRecordEnvelopeJSONContract(parseT *testing.T) {
	parseRecord := BuildOutboxRecordEnvelope(
		"queue-1",
		"resource-1",
		"op-1",
		time.Now().UTC(),
		time.Now().UTC().Add(30*time.Second),
		[]byte(`{"send":true}`),
		OutboxStatusQueued,
		"ack-1",
		"",
	)
	parseJSON, parseErr := BuildOutboxRecordEnvelopeJSON(parseRecord)
	if parseErr != nil {
		parseT.Fatalf("BuildOutboxRecordEnvelopeJSON: %v", parseErr)
	}
	parseRaw := map[string]any{}
	if parseErr = json.Unmarshal(parseJSON, &parseRaw); parseErr != nil {
		parseT.Fatalf("json.Unmarshal: %v", parseErr)
	}
	parseRequiredKeys := []string{
		"queue_key",
		"resource_key",
		"operation_key",
		"created_at",
		"last_attempt_at",
		"attempt_count",
		"next_retry_at",
		"parsePayload",
		"parseStatus",
		"parseAckKey",
		"parseLastError",
	}
	for _, parseKey := range parseRequiredKeys {
		if _, isParsePresent := parseRaw[parseKey]; !isParsePresent {
			parseT.Fatalf("missing canonical outbox-record key %q in %s", parseKey, string(parseJSON))
		}
	}
	parseParsedRecord, parseErr := ParseOutboxRecordEnvelopeJSON(parseJSON)
	if parseErr != nil {
		parseT.Fatalf("ParseOutboxRecordEnvelopeJSON: %v", parseErr)
	}
	if parseParsedRecord.QueueKey != parseRecord.QueueKey || parseParsedRecord.OperationKey != parseRecord.OperationKey {
		parseT.Fatalf("parsed outbox record mismatch: got %+v want %+v", parseParsedRecord, parseRecord)
	}
}
