package cachepolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/client/cachecore"
)

const conversationListPageResourcePrefix = "conversation_list.page"
const conversationListAnchorResourcePrefix = "conversation_list.anchor"

// ConversationListPageInput stores one conversation-list page cache read/write envelope.
type ConversationListPageInput struct {
	ScopeKey   string
	PageCursor string
	PageSize   int
}

// ConversationListPagePayload stores one cached sidebar page payload plus has-more and anchor metadata.
type ConversationListPagePayload struct {
	HasMore                    bool   `json:"has_more"`
	AnchorConversationPublicID string `json:"anchor_conversation_public_id"`
	PagePayload                []byte `json:"page_payload"`
}

// BuildConversationListPagePolicy returns one page-cache policy tuned for sidebar restore with SWR updates.
func BuildConversationListPagePolicy() cachecore.CachePolicy {
	return cachecore.BuildCachePolicy(cachecore.PolicyClassSession, 45*time.Second, 30*time.Minute, true, true, false)
}

// BuildConversationListPageResourceKey builds one conversation-list page resource key from cursor+page-size dimensions.
func BuildConversationListPageResourceKey(parsePageCursor string, parsePageSize int) string {
	parsePageCursor = strings.TrimSpace(parsePageCursor)
	if parsePageSize <= 0 {
		parsePageSize = 20
	}
	return fmt.Sprintf("%s|cursor=%s|size=%d", conversationListPageResourcePrefix, parsePageCursor, parsePageSize)
}

// BuildConversationListAnchorResourceKey builds one scroll-anchor resource key for one sidebar route context.
func BuildConversationListAnchorResourceKey(parseRouteKey string) string {
	return conversationListAnchorResourcePrefix + "|route=" + strings.TrimSpace(parseRouteKey)
}

// BuildConversationListPagePayloadJSON builds one JSON payload for has-more plus anchor metadata and page payload bytes.
func BuildConversationListPagePayloadJSON(parseHasMore bool, parseAnchorConversationPublicID string, parsePagePayload []byte) ([]byte, error) {
	return json.Marshal(ConversationListPagePayload{
		HasMore:                    parseHasMore,
		AnchorConversationPublicID: strings.TrimSpace(parseAnchorConversationPublicID),
		PagePayload:                append([]byte(nil), parsePagePayload...),
	})
}

// ParseConversationListPagePayloadJSON parses one stored conversation-list page payload contract.
func ParseConversationListPagePayloadJSON(parseRaw []byte) (ConversationListPagePayload, error) {
	parsePayload := ConversationListPagePayload{}
	if parseErr := json.Unmarshal(parseRaw, &parsePayload); parseErr != nil {
		return ConversationListPagePayload{}, parseErr
	}
	parsePayload.AnchorConversationPublicID = strings.TrimSpace(parsePayload.AnchorConversationPublicID)
	parsePayload.PagePayload = append([]byte(nil), parsePayload.PagePayload...)
	return parsePayload, nil
}

// StoreConversationListPage stores one conversation-list page snapshot for quick sidebar restore.
func StoreConversationListPage(parseCtx context.Context, parseStorage cachecore.Storage, parseInput ConversationListPageInput, parseBundleVersion string, parsePagePayload []byte, parseHasMore bool, parseAnchorConversationPublicID string) error {
	if parseStorage == nil {
		return nil
	}
	parseInput = normalizeConversationListPageInput(parseInput)
	parsePolicy := BuildConversationListPagePolicy()
	parsePayloadJSON, parseErr := BuildConversationListPagePayloadJSON(parseHasMore, parseAnchorConversationPublicID, parsePagePayload)
	if parseErr != nil {
		return parseErr
	}
	parseRecord := cachecore.BuildCacheRecordEnvelope(
		parseInput.ScopeKey,
		BuildConversationListPageResourceKey(parseInput.PageCursor, parseInput.PageSize),
		strings.TrimSpace(parseBundleVersion),
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

// ReadConversationListPage reads one sidebar page with snapshot-first SWR behavior.
func ReadConversationListPage(
	parseCtx context.Context,
	parseAPI *cachecore.UIAPI,
	parseInput ConversationListPageInput,
	parseRefresh func(context.Context, ConversationListPageInput) (cachecore.CacheRecordEnvelope, error),
) (cachecore.CachedResourceView, error) {
	if parseAPI == nil {
		return cachecore.CachedResourceView{}, nil
	}
	parseInput = normalizeConversationListPageInput(parseInput)
	parseResourceKey := BuildConversationListPageResourceKey(parseInput.PageCursor, parseInput.PageSize)
	return parseAPI.ReadCachedResource(parseCtx, parseInput.ScopeKey, parseResourceKey, BuildConversationListPagePolicy(), func(parseCtx context.Context, parseScopeKey string, parseResourceKey string) (cachecore.CacheRecordEnvelope, error) {
		_ = parseScopeKey
		_ = parseResourceKey
		if parseRefresh == nil {
			return cachecore.CacheRecordEnvelope{}, nil
		}
		return parseRefresh(parseCtx, parseInput)
	})
}

// StoreConversationListScrollAnchor stores one lightweight scroll-memory anchor for sidebar restore.
func StoreConversationListScrollAnchor(parseCtx context.Context, parseStorage cachecore.Storage, parseScopeKey string, parseRouteKey string, parseAnchorPayload []byte) error {
	if parseStorage == nil {
		return nil
	}
	parsePolicy := BuildConversationListPagePolicy()
	parseRecord := cachecore.BuildCacheRecordEnvelope(
		strings.TrimSpace(parseScopeKey),
		BuildConversationListAnchorResourceKey(parseRouteKey),
		"",
		"",
		time.Now().UTC(),
		parsePolicy.StaleAfter,
		parsePolicy.ExpiresAfter,
		append([]byte(nil), parseAnchorPayload...),
		cachecore.CacheStatusReady,
		"",
	)
	return parseStorage.Set(parseCtx, parseRecord)
}

// ReadConversationListScrollAnchor reads one sidebar scroll anchor with snapshot-first SWR behavior.
func ReadConversationListScrollAnchor(
	parseCtx context.Context,
	parseAPI *cachecore.UIAPI,
	parseScopeKey string,
	parseRouteKey string,
	parseRefresh func(context.Context, string, string) (cachecore.CacheRecordEnvelope, error),
) (cachecore.CachedResourceView, error) {
	if parseAPI == nil {
		return cachecore.CachedResourceView{}, nil
	}
	return parseAPI.ReadCachedResource(
		parseCtx,
		strings.TrimSpace(parseScopeKey),
		BuildConversationListAnchorResourceKey(parseRouteKey),
		BuildConversationListPagePolicy(),
		parseRefresh,
	)
}

// normalizeConversationListPageInput normalizes one conversation-list page envelope.
func normalizeConversationListPageInput(parseInput ConversationListPageInput) ConversationListPageInput {
	parseInput.ScopeKey = strings.TrimSpace(parseInput.ScopeKey)
	parseInput.PageCursor = strings.TrimSpace(parseInput.PageCursor)
	if parseInput.PageSize <= 0 {
		parseInput.PageSize = 20
	}
	return parseInput
}
