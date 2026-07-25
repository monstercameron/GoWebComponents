package cachepolicy

import (
	"context"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/client/cachecore"
)

// TestBuildParseBillingSummarySnapshotPayloadJSON verifies billing payload contract fields and total math.
func TestBuildParseBillingSummarySnapshotPayloadJSON(parseT *testing.T) {
	parseRaw, parseErr := BuildBillingSummarySnapshotPayloadJSON(BillingSummarySnapshotPayload{
		PlatformFeeCents:    2900,
		RawUsageCents:       1200,
		ServicePremiumCents: 210,
		Currency:            "usd",
	})
	if parseErr != nil {
		parseT.Fatalf("BuildBillingSummarySnapshotPayloadJSON: %v", parseErr)
	}
	parsePayload, parseErr := ParseBillingSummarySnapshotPayloadJSON(parseRaw)
	if parseErr != nil {
		parseT.Fatalf("ParseBillingSummarySnapshotPayloadJSON: %v", parseErr)
	}
	if parsePayload.TotalCents != 4310 || parsePayload.Currency != "USD" {
		parseT.Fatalf("unexpected billing payload normalization: %+v", parsePayload)
	}
}

// TestStoreReadBillingSummarySnapshot verifies billing snapshots restore quickly and trigger SWR refresh on stale reads.
func TestStoreReadBillingSummarySnapshot(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseAPI := cachecore.BuildUIAPIWithSnapshot(parseStorage, parseSnapshot, cachecore.BuildSWRCoordinator(), cachecore.BuildInvalidationEngine(parseStorage, cachecore.BuildInvalidator()))
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})
	parsePolicy := BuildBillingSummarySnapshotPolicy()

	if parseErr := StoreBillingSummarySnapshot(parseCtx, parseStorage, parseScopeKey, "v1", BillingSummarySnapshotPayload{
		PlatformFeeCents:    2900,
		RawUsageCents:       900,
		ServicePremiumCents: 45,
		Currency:            "USD",
	}); parseErr != nil {
		parseT.Fatalf("StoreBillingSummarySnapshot: %v", parseErr)
	}
	parseStoredRecord, isParseFound, parseErr := parseStorage.Get(parseCtx, cachecore.BuildScopedResourceKey(parseScopeKey, billingSummaryResourceKey))
	if parseErr != nil || !isParseFound {
		parseT.Fatalf("Get(stored): found=%t err=%v", isParseFound, parseErr)
	}
	parseStaleRecord := cachecore.BuildCacheRecordEnvelope(parseScopeKey, billingSummaryResourceKey, "v1", "", time.Now().UTC().Add(-10*time.Minute), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, parseStoredRecord.Payload, cachecore.CacheStatusReady, "")
	parseSnapshot.SetSync(parseStaleRecord)
	parseRefreshDone := make(chan struct{}, 1)
	parseView, parseErr := ReadBillingSummarySnapshot(parseCtx, parseAPI, parseScopeKey, func(parseCtx context.Context, parseScopeKey string, parseResourceKey string) (cachecore.CacheRecordEnvelope, error) {
		_ = parseCtx
		parseRefreshDone <- struct{}{}
		return cachecore.BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "v2", "", time.Now().UTC(), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, []byte(`{"total_cents":0}`), cachecore.CacheStatusReady, ""), nil
	})
	if parseErr != nil {
		parseT.Fatalf("ReadBillingSummarySnapshot: %v", parseErr)
	}
	if !parseView.IsFound || !parseView.IsCached || !parseView.IsStale || !parseView.RefreshTriggered {
		parseT.Fatalf("unexpected billing snapshot read view: %+v", parseView)
	}
	select {
	case <-parseRefreshDone:
	case <-time.After(2 * time.Second):
		parseT.Fatal("timed out waiting for billing summary background refresh")
	}
}

// TestApplyBillingSummaryUnavailable verifies stale/unavailable billing fallback retains customer-visible stale metadata.
func TestApplyBillingSummaryUnavailable(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})

	if parseErr := StoreBillingSummarySnapshot(parseCtx, parseStorage, parseScopeKey, "v1", BillingSummarySnapshotPayload{
		PlatformFeeCents:    2900,
		RawUsageCents:       900,
		ServicePremiumCents: 45,
		Currency:            "USD",
	}); parseErr != nil {
		parseT.Fatalf("StoreBillingSummarySnapshot: %v", parseErr)
	}
	parseSeedRecord, _, _ := parseStorage.Get(parseCtx, cachecore.BuildScopedResourceKey(parseScopeKey, billingSummaryResourceKey))
	parseSnapshot.SetSync(parseSeedRecord)

	if parseErr := ApplyBillingSummaryUnavailable(parseCtx, parseStorage, parseSnapshot, parseScopeKey, "billing service unavailable"); parseErr != nil {
		parseT.Fatalf("ApplyBillingSummaryUnavailable: %v", parseErr)
	}
	parseRecord, isParseFound, parseErr := parseStorage.Get(parseCtx, cachecore.BuildScopedResourceKey(parseScopeKey, billingSummaryResourceKey))
	if parseErr != nil || !isParseFound {
		parseT.Fatalf("Get(after unavailable): found=%t err=%v", isParseFound, parseErr)
	}
	if parseRecord.Status != cachecore.CacheStatusStale || parseRecord.Error != "billing service unavailable" {
		parseT.Fatalf("unexpected stale/unavailable record: %+v", parseRecord)
	}
	parsePayload, parseErr := ParseBillingSummarySnapshotPayloadJSON(parseRecord.Payload)
	if parseErr != nil {
		parseT.Fatalf("ParseBillingSummarySnapshotPayloadJSON(after): %v", parseErr)
	}
	if !parsePayload.IsStale || parsePayload.StaleReason != "billing service unavailable" {
		parseT.Fatalf("expected stale payload metadata, got %+v", parsePayload)
	}
}
