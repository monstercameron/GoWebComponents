package cachecore

import (
	"context"
	"time"
)

// CachedResourceView stores UI-facing cached-read metadata.
type CachedResourceView struct {
	Record           CacheRecordEnvelope
	IsFound          bool
	IsCached         bool
	IsStale          bool
	Freshness        FreshnessState
	RefreshTriggered bool
}

// QueuedMutationView stores UI-facing queue-state metadata.
type QueuedMutationView struct {
	QueueKey     string
	PendingCount int
	NextRetryAt  string
	IsPending    bool
}

// UIAPI provides reusable UI-facing cache/outbox helpers over core runtime primitives.
type UIAPI struct {
	parseStorage      Storage
	parseSnapshot     *SnapshotStore
	parseCoordinator  *SWRCoordinator
	parseInvalidation *InvalidationEngine
}

// BuildUIAPI creates one UI helper API wrapper for cache-core primitives.
func BuildUIAPI(parseStorage Storage, parseCoordinator *SWRCoordinator, parseInvalidation *InvalidationEngine) *UIAPI {
	return &UIAPI{
		parseStorage:      parseStorage,
		parseCoordinator:  parseCoordinator,
		parseInvalidation: parseInvalidation,
	}
}

// BuildUIAPIWithSnapshot creates one UI helper API wrapper with one main-thread snapshot mirror for render-safe reads.
func BuildUIAPIWithSnapshot(parseStorage Storage, parseSnapshot *SnapshotStore, parseCoordinator *SWRCoordinator, parseInvalidation *InvalidationEngine) *UIAPI {
	parseAPI := BuildUIAPI(parseStorage, parseCoordinator, parseInvalidation)
	parseAPI.parseSnapshot = parseSnapshot
	return parseAPI
}

// SetSnapshotStore sets one main-thread snapshot mirror for render-safe cached reads.
func (parseAPI *UIAPI) SetSnapshotStore(parseSnapshot *SnapshotStore) {
	if parseAPI == nil {
		return
	}
	parseAPI.parseSnapshot = parseSnapshot
}

// ReadCachedResource reads one cached resource with stale-while-revalidate semantics.
func (parseAPI *UIAPI) ReadCachedResource(
	parseCtx context.Context,
	parseScopeKey string,
	parseResourceKey string,
	parsePolicy CachePolicy,
	parseRefresh func(context.Context, string, string) (CacheRecordEnvelope, error),
) (CachedResourceView, error) {
	if parseAPI == nil {
		return CachedResourceView{}, nil
	}
	if parseAPI.parseSnapshot != nil {
		return parseAPI.getCachedResourceFromSnapshot(parseCtx, parseScopeKey, parseResourceKey, parsePolicy, parseRefresh)
	}
	parseResult, parseErr := ReadWithSWR(parseCtx, parseAPI.parseStorage, parseAPI.parseCoordinator, parsePolicy, parseScopeKey, parseResourceKey, parseRefresh)
	if parseErr != nil {
		return CachedResourceView{}, parseErr
	}
	return CachedResourceView{
		Record:           parseResult.Record,
		IsFound:          parseResult.IsFound,
		IsCached:         parseResult.IsCached,
		IsStale:          parseResult.IsStale,
		Freshness:        parseResult.Freshness,
		RefreshTriggered: parseResult.RefreshTriggered,
	}, nil
}

// getCachedResourceFromSnapshot reads one cached resource from snapshot-first state and only triggers async maintenance work.
func (parseAPI *UIAPI) getCachedResourceFromSnapshot(
	parseCtx context.Context,
	parseScopeKey string,
	parseResourceKey string,
	parsePolicy CachePolicy,
	parseRefresh func(context.Context, string, string) (CacheRecordEnvelope, error),
) (CachedResourceView, error) {
	parseRecord, isParseFound := parseAPI.parseSnapshot.GetSync(parseScopeKey, parseResourceKey)
	parseResult := SWRReadResult{
		Record:   parseRecord,
		IsFound:  isParseFound,
		IsCached: isParseFound,
	}
	if !isParseFound {
		parseResult.Freshness = FreshnessMissing
		if parseErr := parseStartSWRBackgroundRefresh(parseCtx, parseAPI.parseStorage, parseAPI.parseCoordinator, parsePolicy, parseScopeKey, parseResourceKey, parseRefresh, &parseResult); parseErr != nil {
			return CachedResourceView{}, parseErr
		}
		return CachedResourceView{
			Record:           parseResult.Record,
			IsFound:          parseResult.IsFound,
			IsCached:         parseResult.IsCached,
			IsStale:          parseResult.IsStale,
			Freshness:        parseResult.Freshness,
			RefreshTriggered: parseResult.RefreshTriggered,
		}, nil
	}
	parseFreshness := ResolveFreshnessState(parseRecord, parseNowUTC())
	parseDecision := ResolvePolicyReadDecision(parsePolicy, parseFreshness)
	parseResult.Freshness = parseFreshness
	parseResult.IsStale = parseDecision.IsStale
	if parseDecision.ShouldRefreshInBackground {
		if parseErr := parseStartSWRBackgroundRefresh(parseCtx, parseAPI.parseStorage, parseAPI.parseCoordinator, parsePolicy, parseScopeKey, parseResourceKey, parseRefresh, &parseResult); parseErr != nil {
			return CachedResourceView{}, parseErr
		}
	}
	return CachedResourceView{
		Record:           parseResult.Record,
		IsFound:          parseResult.IsFound,
		IsCached:         parseResult.IsCached,
		IsStale:          parseResult.IsStale,
		Freshness:        parseResult.Freshness,
		RefreshTriggered: parseResult.RefreshTriggered,
	}, nil
}

// ReadQueuedMutationState reads one queued mutation status snapshot for one queue key.
func (parseAPI *UIAPI) ReadQueuedMutationState(parseCtx context.Context, parseQueueKey string) (QueuedMutationView, error) {
	if parseAPI == nil || parseAPI.parseStorage == nil {
		return QueuedMutationView{QueueKey: parseQueueKey}, nil
	}
	parseQueue, parseErr := parseAPI.parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []OutboxRecordEnvelope) ([]OutboxRecordEnvelope, error) {
		return parseQueue, nil
	})
	if parseErr != nil {
		return QueuedMutationView{}, parseErr
	}
	parseView := QueuedMutationView{
		QueueKey:     parseQueueKey,
		PendingCount: len(parseQueue),
		IsPending:    len(parseQueue) > 0,
	}
	if len(parseQueue) > 0 {
		parseView.NextRetryAt = parseQueue[0].NextRetryAt
	}
	return parseView, nil
}

// RefreshResource forces one authoritative refresh and stores the returned cache record.
func (parseAPI *UIAPI) RefreshResource(
	parseCtx context.Context,
	parseScopeKey string,
	parseResourceKey string,
	parseRefresh func(context.Context, string, string) (CacheRecordEnvelope, error),
) error {
	if parseAPI == nil || parseAPI.parseStorage == nil || parseRefresh == nil {
		return nil
	}
	parseRecord, parseErr := parseRefresh(parseCtx, parseScopeKey, parseResourceKey)
	if parseErr != nil {
		return parseErr
	}
	parseRecord = NormalizeCacheRecordEnvelope(parseRecord)
	if parseRecord.ScopeKey == "" {
		parseRecord.ScopeKey = parseScopeKey
	}
	if parseRecord.ResourceKey == "" {
		parseRecord.ResourceKey = parseResourceKey
	}
	return parseAPI.parseStorage.Set(parseCtx, parseRecord)
}

// InvalidateResources triggers reason-based invalidation through the configured engine.
func (parseAPI *UIAPI) InvalidateResources(parseCtx context.Context, parseReason string, parseScopePrefix string, parseResourcePrefix string) (int, error) {
	if parseAPI == nil || parseAPI.parseInvalidation == nil {
		return 0, nil
	}
	return parseAPI.parseInvalidation.ApplyInvalidation(parseCtx, InvalidationEvent{
		Reason:         parseReason,
		ScopePrefix:    parseScopePrefix,
		ResourcePrefix: parseResourcePrefix,
	})
}

// ApplyOptimisticPatch reads one resource and writes one patched optimistic snapshot.
func (parseAPI *UIAPI) ApplyOptimisticPatch(
	parseCtx context.Context,
	parseScopeKey string,
	parseResourceKey string,
	parsePatch func(CacheRecordEnvelope) CacheRecordEnvelope,
) error {
	if parseAPI == nil || parseAPI.parseStorage == nil || parsePatch == nil {
		return nil
	}
	parseScopedResourceKey := BuildScopedResourceKey(parseScopeKey, parseResourceKey)
	parseRecord, isParseFound, parseErr := parseAPI.parseStorage.Get(parseCtx, parseScopedResourceKey)
	if parseErr != nil {
		return parseErr
	}
	if !isParseFound {
		parseRecord = BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "", "", parseNowUTC(), 0, 0, []byte{}, CacheStatusReady, "")
	}
	parsePatchedRecord := NormalizeCacheRecordEnvelope(parsePatch(parseRecord))
	if parsePatchedRecord.ScopeKey == "" {
		parsePatchedRecord.ScopeKey = parseScopeKey
	}
	if parsePatchedRecord.ResourceKey == "" {
		parsePatchedRecord.ResourceKey = parseResourceKey
	}
	return parseAPI.parseStorage.Set(parseCtx, parsePatchedRecord)
}

// parseNowUTC returns one UTC wall clock timestamp.
func parseNowUTC() (parseNow time.Time) {
	return time.Now().UTC()
}
