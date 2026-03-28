package cachepolicy

import (
	"context"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/client/cachecore"
)

const threadDraftResourcePrefix = "draft.thread="
const threadDraftDiscardReasonSend = "sent"
const threadDraftDiscardReasonUserClear = "user_clear"

// BuildThreadDraftPolicy returns one per-thread draft cache policy tuned for quick route-switch restore.
func BuildThreadDraftPolicy() cachecore.CachePolicy {
	return cachecore.BuildCachePolicy(cachecore.PolicyClassSession, 30*time.Second, 12*time.Hour, true, true, false)
}

// BuildThreadDraftResourceKey builds one thread-scoped draft cache resource key.
func BuildThreadDraftResourceKey(parseThreadRoutePublicID string) string {
	return threadDraftResourcePrefix + normalizeUnsentMessageThreadKey(parseThreadRoutePublicID)
}

// StoreThreadDraft writes one thread draft payload independently from unsent-message outbox state.
func StoreThreadDraft(parseCtx context.Context, parseStorage cachecore.Storage, parseScopeKey string, parseThreadRoutePublicID string, parseDraftPayload []byte) error {
	if parseStorage == nil {
		return nil
	}
	parsePolicy := BuildThreadDraftPolicy()
	parseScopeKey = strings.TrimSpace(parseScopeKey)
	parseResourceKey := BuildThreadDraftResourceKey(parseThreadRoutePublicID)
	parseRecord := cachecore.BuildCacheRecordEnvelope(
		parseScopeKey,
		parseResourceKey,
		"draft-v1",
		"",
		time.Now().UTC(),
		parsePolicy.StaleAfter,
		parsePolicy.ExpiresAfter,
		append([]byte(nil), parseDraftPayload...),
		cachecore.CacheStatusReady,
		"",
	)
	return parseStorage.Set(parseCtx, parseRecord)
}

// ReadThreadDraft reads one thread draft from cache with snapshot-first policy behavior.
func ReadThreadDraft(
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
		BuildThreadDraftResourceKey(parseThreadRoutePublicID),
		BuildThreadDraftPolicy(),
		parseRefresh,
	)
}

// ApplyThreadDraftDiscardAfterSend removes one thread draft after successful message send.
func ApplyThreadDraftDiscardAfterSend(parseCtx context.Context, parseStorage cachecore.Storage, parseSnapshot *cachecore.SnapshotStore, parseScopeKey string, parseThreadRoutePublicID string) (int, error) {
	return applyThreadDraftDiscard(parseCtx, parseStorage, parseSnapshot, parseScopeKey, parseThreadRoutePublicID, threadDraftDiscardReasonSend)
}

// ApplyThreadDraftDiscardByUserClear removes one thread draft after explicit user clear action.
func ApplyThreadDraftDiscardByUserClear(parseCtx context.Context, parseStorage cachecore.Storage, parseSnapshot *cachecore.SnapshotStore, parseScopeKey string, parseThreadRoutePublicID string) (int, error) {
	return applyThreadDraftDiscard(parseCtx, parseStorage, parseSnapshot, parseScopeKey, parseThreadRoutePublicID, threadDraftDiscardReasonUserClear)
}

// applyThreadDraftDiscard removes one thread draft resource from storage and snapshot state.
func applyThreadDraftDiscard(parseCtx context.Context, parseStorage cachecore.Storage, parseSnapshot *cachecore.SnapshotStore, parseScopeKey string, parseThreadRoutePublicID string, parseReason string) (int, error) {
	_ = strings.TrimSpace(parseReason)
	if parseStorage == nil {
		return 0, nil
	}
	parseScopeKey = strings.TrimSpace(parseScopeKey)
	parseResourceKey := BuildThreadDraftResourceKey(parseThreadRoutePublicID)
	parseScopedResourceKey := cachecore.BuildScopedResourceKey(parseScopeKey, parseResourceKey)
	if parseErr := parseStorage.Delete(parseCtx, parseScopedResourceKey); parseErr != nil {
		return 0, parseErr
	}
	if parseSnapshot != nil {
		parseSnapshot.DeleteSync(parseScopeKey, parseResourceKey)
	}
	return 1, nil
}
