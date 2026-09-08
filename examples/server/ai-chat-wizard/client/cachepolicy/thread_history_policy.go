package cachepolicy

import (
	"context"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/client/cachecore"
)

const threadHistoryResourcePrefix = "thread.history="

// BuildThreadHistoryPolicy returns one thread-history cache policy tuned for fast reopen plus frequent refresh.
func BuildThreadHistoryPolicy() cachecore.CachePolicy {
	return cachecore.BuildCachePolicy(cachecore.PolicyClassSession, 20*time.Second, 20*time.Minute, true, true, false)
}

// BuildThreadHistoryResourceKey builds one thread-history resource key for one canonical thread route context.
func BuildThreadHistoryResourceKey(parseThreadRoutePublicID string) string {
	return threadHistoryResourcePrefix + normalizeUnsentMessageThreadKey(parseThreadRoutePublicID)
}

// StoreThreadHistorySnapshot stores one thread-history payload snapshot.
func StoreThreadHistorySnapshot(parseCtx context.Context, parseStorage cachecore.Storage, parseScopeKey string, parseThreadRoutePublicID string, parseVersion string, parsePayload []byte) error {
	if parseStorage == nil {
		return nil
	}
	parsePolicy := BuildThreadHistoryPolicy()
	parseRecord := cachecore.BuildCacheRecordEnvelope(
		strings.TrimSpace(parseScopeKey),
		BuildThreadHistoryResourceKey(parseThreadRoutePublicID),
		strings.TrimSpace(parseVersion),
		"",
		time.Now().UTC(),
		parsePolicy.StaleAfter,
		parsePolicy.ExpiresAfter,
		append([]byte(nil), parsePayload...),
		cachecore.CacheStatusReady,
		"",
	)
	return parseStorage.Set(parseCtx, parseRecord)
}

// ReadThreadHistory reads one thread-history record with snapshot-first SWR behavior.
func ReadThreadHistory(
	parseCtx context.Context,
	parseAPI *cachecore.UIAPI,
	parseScopeKey string,
	parseThreadRoutePublicID string,
	parseRefresh func(context.Context, string, string) (cachecore.CacheRecordEnvelope, error),
) (cachecore.CachedResourceView, error) {
	if parseAPI == nil {
		return cachecore.CachedResourceView{}, nil
	}
	return parseAPI.ReadCachedResource(
		parseCtx,
		strings.TrimSpace(parseScopeKey),
		BuildThreadHistoryResourceKey(parseThreadRoutePublicID),
		BuildThreadHistoryPolicy(),
		parseRefresh,
	)
}

// ApplyThreadHistoryStreamPatch applies one streamed assistant-output patch to one cached thread-history record.
func ApplyThreadHistoryStreamPatch(
	parseCtx context.Context,
	parseStorage cachecore.Storage,
	parseSnapshot *cachecore.SnapshotStore,
	parseScopeKey string,
	parseThreadRoutePublicID string,
	parsePatch func([]byte) []byte,
) error {
	if parseStorage == nil || parsePatch == nil {
		return nil
	}
	parseScopeKey = strings.TrimSpace(parseScopeKey)
	parseResourceKey := BuildThreadHistoryResourceKey(parseThreadRoutePublicID)
	parseScopedResourceKey := cachecore.BuildScopedResourceKey(parseScopeKey, parseResourceKey)
	parseRecord, isParseFound, parseErr := parseStorage.Get(parseCtx, parseScopedResourceKey)
	if parseErr != nil {
		return parseErr
	}
	if !isParseFound {
		parseRecord = cachecore.BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "stream-v1", "", time.Now().UTC(), BuildThreadHistoryPolicy().StaleAfter, BuildThreadHistoryPolicy().ExpiresAfter, []byte{}, cachecore.CacheStatusReady, "")
	}
	parseRecord = cachecore.NormalizeCacheRecordEnvelope(parseRecord)
	parseRecord.Payload = append([]byte(nil), parsePatch(parseRecord.Payload)...)
	parseRecord.Status = cachecore.CacheStatusReady
	parseRecord.FetchedAt = time.Now().UTC().Format(time.RFC3339)
	if parseErr = parseStorage.Set(parseCtx, parseRecord); parseErr != nil {
		return parseErr
	}
	if parseSnapshot != nil {
		parseSnapshot.SetSync(parseRecord)
	}
	return nil
}

// ApplyThreadHistoryAuthoritativeReconcile merges one authoritative reconnect snapshot through deterministic worker reconciliation.
func ApplyThreadHistoryAuthoritativeReconcile(
	parseCtx context.Context,
	parseReconciler *cachecore.WorkerSnapshotReconciler,
	parseScopeKey string,
	parseThreadRoutePublicID string,
	parseUpdatedAt time.Time,
	parseRecord cachecore.CacheRecordEnvelope,
) (cachecore.WorkerSnapshotReconcileResult, error) {
	if parseReconciler == nil {
		return cachecore.WorkerSnapshotReconcileResult{}, nil
	}
	parseRecord = cachecore.NormalizeCacheRecordEnvelope(parseRecord)
	if parseRecord.ScopeKey == "" {
		parseRecord.ScopeKey = strings.TrimSpace(parseScopeKey)
	}
	if parseRecord.ResourceKey == "" {
		parseRecord.ResourceKey = BuildThreadHistoryResourceKey(parseThreadRoutePublicID)
	}
	if parseUpdatedAt.IsZero() {
		parseUpdatedAt = time.Now().UTC()
	}
	return parseReconciler.ApplyEvent(parseCtx, cachecore.WorkerSnapshotReconcileInput{
		Event: cachecore.WorkerEvent{
			EventType:   cachecore.WorkerEventUpdated,
			ScopeKey:    parseRecord.ScopeKey,
			ResourceKey: parseRecord.ResourceKey,
			UpdatedAt:   parseUpdatedAt.UTC().Format(time.RFC3339),
		},
		Record: parseRecord,
	})
}
