package cachecore

import (
	"context"
	"time"
)

// SWRReadResult stores one stale-while-revalidate read result and metadata.
type SWRReadResult struct {
	Record           CacheRecordEnvelope
	IsFound          bool
	IsCached         bool
	IsStale          bool
	Freshness        FreshnessState
	RefreshTriggered bool
}

// ReadWithSWR reads one scoped resource from cache and triggers one background refresh when policy allows.
func ReadWithSWR(
	parseCtx context.Context,
	parseStorage Storage,
	parseCoordinator *SWRCoordinator,
	parsePolicy CachePolicy,
	parseScopeKey string,
	parseResourceKey string,
	parseRefresh func(context.Context, string, string) (CacheRecordEnvelope, error),
) (SWRReadResult, error) {
	parseScopedResourceKey := BuildScopedResourceKey(parseScopeKey, parseResourceKey)
	parseRecord, isParseFound, parseErr := parseStorage.Get(parseCtx, parseScopedResourceKey)
	if parseErr != nil {
		return SWRReadResult{}, parseErr
	}
	parseResult := SWRReadResult{
		Record:   parseRecord,
		IsFound:  isParseFound,
		IsCached: isParseFound,
	}
	if !isParseFound {
		parseResult.Freshness = FreshnessMissing
		return parseResult, parseStartSWRBackgroundRefresh(parseCtx, parseStorage, parseCoordinator, parsePolicy, parseScopeKey, parseResourceKey, parseRefresh, &parseResult)
	}
	parseFreshness := ResolveFreshnessState(parseRecord, time.Now().UTC())
	parseResult.Freshness = parseFreshness
	parseDecision := ResolvePolicyReadDecision(parsePolicy, parseFreshness)
	parseResult.IsStale = parseDecision.IsStale
	if parseDecision.ShouldRefreshInBackground {
		parseErr = parseStartSWRBackgroundRefresh(parseCtx, parseStorage, parseCoordinator, parsePolicy, parseScopeKey, parseResourceKey, parseRefresh, &parseResult)
		if parseErr != nil {
			return parseResult, parseErr
		}
	}
	return parseResult, nil
}

// parseStartSWRBackgroundRefresh starts one deduplicated background refresh when callback and coordinator are available.
func parseStartSWRBackgroundRefresh(
	parseCtx context.Context,
	parseStorage Storage,
	parseCoordinator *SWRCoordinator,
	parsePolicy CachePolicy,
	parseScopeKey string,
	parseResourceKey string,
	parseRefresh func(context.Context, string, string) (CacheRecordEnvelope, error),
	parseResult *SWRReadResult,
) error {
	_ = parsePolicy
	if parseRefresh == nil || parseStorage == nil || parseCoordinator == nil {
		return nil
	}
	parseRelease, isParseStarted := parseCoordinator.StartRefresh(parseScopeKey, parseResourceKey)
	if !isParseStarted {
		return nil
	}
	if parseResult != nil {
		parseResult.RefreshTriggered = true
	}
	go func() {
		defer parseRelease()
		parseRefreshedRecord, parseErr := parseRefresh(parseCtx, parseScopeKey, parseResourceKey)
		if parseErr != nil {
			return
		}
		parseRefreshedRecord = NormalizeCacheRecordEnvelope(parseRefreshedRecord)
		if parseRefreshedRecord.ScopeKey == "" {
			parseRefreshedRecord.ScopeKey = parseScopeKey
		}
		if parseRefreshedRecord.ResourceKey == "" {
			parseRefreshedRecord.ResourceKey = parseResourceKey
		}
		_ = parseStorage.Set(context.Background(), parseRefreshedRecord)
	}()
	return nil
}
