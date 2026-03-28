package cachepolicy

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/client/cachecore"
)

const billingSummaryResourceKey = "billing.summary.snapshot"

// BillingSummarySnapshotPayload stores one canonical customer billing snapshot payload.
type BillingSummarySnapshotPayload struct {
	PlatformFeeCents    int64  `json:"platform_fee_cents"`
	RawUsageCents       int64  `json:"raw_usage_cents"`
	ServicePremiumCents int64  `json:"service_premium_cents"`
	TotalCents          int64  `json:"total_cents"`
	Currency            string `json:"currency"`
	CapturedAt          string `json:"captured_at"`
	IsStale             bool   `json:"is_stale"`
	StaleReason         string `json:"stale_reason"`
}

// BuildBillingSummarySnapshotPolicy returns one billing-summary snapshot cache policy.
func BuildBillingSummarySnapshotPolicy() cachecore.CachePolicy {
	return cachecore.BuildCachePolicy(cachecore.PolicyClassStatic, 90*time.Second, 4*time.Hour, true, true, false)
}

// BuildBillingSummarySnapshotPayloadJSON builds one canonical billing-summary payload JSON contract.
func BuildBillingSummarySnapshotPayloadJSON(parsePayload BillingSummarySnapshotPayload) ([]byte, error) {
	parsePayload.Currency = strings.TrimSpace(strings.ToUpper(parsePayload.Currency))
	if parsePayload.Currency == "" {
		parsePayload.Currency = "USD"
	}
	parsePayload.CapturedAt = strings.TrimSpace(parsePayload.CapturedAt)
	if parsePayload.CapturedAt == "" {
		parsePayload.CapturedAt = time.Now().UTC().Format(time.RFC3339)
	}
	parsePayload.StaleReason = strings.TrimSpace(parsePayload.StaleReason)
	if parsePayload.TotalCents == 0 {
		parsePayload.TotalCents = parsePayload.PlatformFeeCents + parsePayload.RawUsageCents + parsePayload.ServicePremiumCents
	}
	return json.Marshal(parsePayload)
}

// ParseBillingSummarySnapshotPayloadJSON parses one canonical billing-summary payload JSON contract.
func ParseBillingSummarySnapshotPayloadJSON(parseRaw []byte) (BillingSummarySnapshotPayload, error) {
	parsePayload := BillingSummarySnapshotPayload{}
	if parseErr := json.Unmarshal(parseRaw, &parsePayload); parseErr != nil {
		return BillingSummarySnapshotPayload{}, parseErr
	}
	parsePayload.Currency = strings.TrimSpace(strings.ToUpper(parsePayload.Currency))
	if parsePayload.Currency == "" {
		parsePayload.Currency = "USD"
	}
	parsePayload.StaleReason = strings.TrimSpace(parsePayload.StaleReason)
	return parsePayload, nil
}

// StoreBillingSummarySnapshot stores one canonical billing snapshot payload.
func StoreBillingSummarySnapshot(parseCtx context.Context, parseStorage cachecore.Storage, parseScopeKey string, parseVersion string, parsePayload BillingSummarySnapshotPayload) error {
	if parseStorage == nil {
		return nil
	}
	parsePolicy := BuildBillingSummarySnapshotPolicy()
	parsePayloadJSON, parseErr := BuildBillingSummarySnapshotPayloadJSON(parsePayload)
	if parseErr != nil {
		return parseErr
	}
	parseRecord := cachecore.BuildCacheRecordEnvelope(
		strings.TrimSpace(parseScopeKey),
		billingSummaryResourceKey,
		strings.TrimSpace(parseVersion),
		"",
		time.Now().UTC(),
		parsePolicy.StaleAfter,
		parsePolicy.ExpiresAfter,
		parsePayloadJSON,
		cachecore.CacheStatusReady,
		"",
	)
	return parseStorage.Set(parseCtx, parseRecord)
}

// ReadBillingSummarySnapshot reads one billing-summary snapshot with snapshot-first SWR behavior.
func ReadBillingSummarySnapshot(
	parseCtx context.Context,
	parseAPI *cachecore.UIAPI,
	parseScopeKey string,
	parseRefresh func(context.Context, string, string) (cachecore.CacheRecordEnvelope, error),
) (cachecore.CachedResourceView, error) {
	if parseAPI == nil {
		return cachecore.CachedResourceView{}, nil
	}
	return parseAPI.ReadCachedResource(
		parseCtx,
		strings.TrimSpace(parseScopeKey),
		billingSummaryResourceKey,
		BuildBillingSummarySnapshotPolicy(),
		parseRefresh,
	)
}

// ApplyBillingSummaryUnavailable marks one cached billing snapshot as stale/unavailable while preserving the last payload.
func ApplyBillingSummaryUnavailable(parseCtx context.Context, parseStorage cachecore.Storage, parseSnapshot *cachecore.SnapshotStore, parseScopeKey string, parseReason string) error {
	if parseStorage == nil {
		return nil
	}
	parseScopeKey = strings.TrimSpace(parseScopeKey)
	parseScopedResourceKey := cachecore.BuildScopedResourceKey(parseScopeKey, billingSummaryResourceKey)
	parseRecord, isParseFound, parseErr := parseStorage.Get(parseCtx, parseScopedResourceKey)
	if parseErr != nil {
		return parseErr
	}
	parseReason = strings.TrimSpace(parseReason)
	if !isParseFound {
		parsePayloadJSON, parsePayloadErr := BuildBillingSummarySnapshotPayloadJSON(BillingSummarySnapshotPayload{
			Currency:    "USD",
			IsStale:     true,
			StaleReason: parseReason,
		})
		if parsePayloadErr != nil {
			return parsePayloadErr
		}
		parseRecord = cachecore.BuildCacheRecordEnvelope(parseScopeKey, billingSummaryResourceKey, "unavailable", "", time.Now().UTC(), 0, BuildBillingSummarySnapshotPolicy().ExpiresAfter, parsePayloadJSON, cachecore.CacheStatusStale, parseReason)
	} else {
		parseRecord = cachecore.NormalizeCacheRecordEnvelope(parseRecord)
		parsePayload, parsePayloadErr := ParseBillingSummarySnapshotPayloadJSON(parseRecord.Payload)
		if parsePayloadErr == nil {
			parsePayload.IsStale = true
			parsePayload.StaleReason = parseReason
			parsePayloadJSON, parsePayloadBuildErr := BuildBillingSummarySnapshotPayloadJSON(parsePayload)
			if parsePayloadBuildErr == nil {
				parseRecord.Payload = parsePayloadJSON
			}
		}
		parseRecord.Status = cachecore.CacheStatusStale
		parseRecord.Error = parseReason
	}
	if parseErr = parseStorage.Set(parseCtx, parseRecord); parseErr != nil {
		return parseErr
	}
	if parseSnapshot != nil {
		parseSnapshot.SetSync(parseRecord)
	}
	return nil
}
