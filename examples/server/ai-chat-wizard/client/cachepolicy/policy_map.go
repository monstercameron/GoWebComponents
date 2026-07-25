package cachepolicy

import (
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/client/cachecore"
)

// ExampleCacheConsistencyMode identifies one cache consistency mode used by the example-100 cache policy map.
type ExampleCacheConsistencyMode string

const (
	ExampleCacheConsistencyDisplayOnly           ExampleCacheConsistencyMode = "display_only"
	ExampleCacheConsistencyQueuedWrite           ExampleCacheConsistencyMode = "queued_write"
	ExampleCacheConsistencyAuthoritativeAfterAck ExampleCacheConsistencyMode = "authoritative_after_ack"
)

// ExampleCachePolicyEntry stores one teachable cache policy row for one example-100 cached resource.
type ExampleCachePolicyEntry struct {
	ResourceID           string
	ResourceKeyPattern   string
	ScopeKeyShape        string
	PolicyClass          cachecore.PolicyClass
	StaleAfter           time.Duration
	ExpiresAfter         time.Duration
	InvalidationTriggers []string
	OfflineBehavior      string
	ConsistencyMode      ExampleCacheConsistencyMode
}

// BuildExampleCachePolicyMap returns one complete map of example-100 cache/outbox resources and policy behavior.
func BuildExampleCachePolicyMap() []ExampleCachePolicyEntry {
	parseLocalePolicy := BuildLocaleCatalogPolicy()
	parseThreadDraftPolicy := BuildThreadDraftPolicy()
	parseConvListPolicy := BuildConversationListPagePolicy()
	parseThreadHistoryPolicy := BuildThreadHistoryPolicy()
	parseSettingsPolicy := BuildSettingsSnapshotPolicy()
	parseModelCatalogPolicy := BuildModelCatalogSnapshotPolicy()
	parseBillingPolicy := BuildBillingSummarySnapshotPolicy()
	parseAdminPolicy := BuildAdminDashboardSnapshotPolicy()
	parseCanvasPolicy := BuildCanvasSessionSnapshotPolicy()
	return []ExampleCachePolicyEntry{
		{
			ResourceID:           "locale_catalog",
			ResourceKeyPattern:   "catalog.locale|locale=<locale>|namespace=<namespace>|bundle=<bundle_version>",
			ScopeKeyShape:        "app+session+user+workspace+locale+route+thread",
			PolicyClass:          parseLocalePolicy.Class,
			StaleAfter:           parseLocalePolicy.StaleAfter,
			ExpiresAfter:         parseLocalePolicy.ExpiresAfter,
			InvalidationTriggers: []string{"locale_change", "server_version_hash_mismatch"},
			OfflineBehavior:      "render last payload and schedule background refresh",
			ConsistencyMode:      ExampleCacheConsistencyDisplayOnly,
		},
		{
			ResourceID:           "undelivered_log_outbox",
			ResourceKeyPattern:   "outbox.undelivered_logs",
			ScopeKeyShape:        "app+session+user+workspace+locale+route+thread",
			PolicyClass:          cachecore.PolicyClassQueued,
			StaleAfter:           0,
			ExpiresAfter:         BuildUndeliveredLogOutboxPolicy().MaxAge,
			InvalidationTriggers: []string{"ack_received", "max_attempts_reached", "max_age_reached"},
			OfflineBehavior:      "enqueue locally and retry on reconnect",
			ConsistencyMode:      ExampleCacheConsistencyQueuedWrite,
		},
		{
			ResourceID:           "unsent_message_outbox",
			ResourceKeyPattern:   "outbox.unsent_messages.thread=<thread_or_new_chat>",
			ScopeKeyShape:        "app+session+user+workspace+locale+route+thread",
			PolicyClass:          cachecore.PolicyClassQueued,
			StaleAfter:           0,
			ExpiresAfter:         BuildUnsentMessageOutboxPolicy().MaxAge,
			InvalidationTriggers: []string{"ack_received", "route_thread_normalization", "max_attempts_reached", "max_age_reached"},
			OfflineBehavior:      "enqueue per thread and retry on reconnect",
			ConsistencyMode:      ExampleCacheConsistencyQueuedWrite,
		},
		{
			ResourceID:           "thread_draft",
			ResourceKeyPattern:   "draft.thread=<thread_or_new_chat>",
			ScopeKeyShape:        "app+session+user+workspace+locale+route+thread",
			PolicyClass:          parseThreadDraftPolicy.Class,
			StaleAfter:           parseThreadDraftPolicy.StaleAfter,
			ExpiresAfter:         parseThreadDraftPolicy.ExpiresAfter,
			InvalidationTriggers: []string{"message_send_success", "user_clear_draft"},
			OfflineBehavior:      "restore local draft without queue-send side effects",
			ConsistencyMode:      ExampleCacheConsistencyDisplayOnly,
		},
		{
			ResourceID:           "conversation_list_page",
			ResourceKeyPattern:   "conversation_list.page|cursor=<cursor>|size=<size>",
			ScopeKeyShape:        "app+session+user+workspace+locale+route+thread",
			PolicyClass:          parseConvListPolicy.Class,
			StaleAfter:           parseConvListPolicy.StaleAfter,
			ExpiresAfter:         parseConvListPolicy.ExpiresAfter,
			InvalidationTriggers: []string{"mutation_ack", "route_change", "reconnect_refresh"},
			OfflineBehavior:      "restore cached page+anchor then refresh in background",
			ConsistencyMode:      ExampleCacheConsistencyDisplayOnly,
		},
		{
			ResourceID:           "thread_history",
			ResourceKeyPattern:   "thread.history=<thread_or_new_chat>",
			ScopeKeyShape:        "app+session+user+workspace+locale+route+thread",
			PolicyClass:          parseThreadHistoryPolicy.Class,
			StaleAfter:           parseThreadHistoryPolicy.StaleAfter,
			ExpiresAfter:         parseThreadHistoryPolicy.ExpiresAfter,
			InvalidationTriggers: []string{"reconnect_authoritative_snapshot", "eviction_event"},
			OfflineBehavior:      "render cached thread while authoritative reconcile runs",
			ConsistencyMode:      ExampleCacheConsistencyAuthoritativeAfterAck,
		},
		{
			ResourceID:           "settings_snapshot",
			ResourceKeyPattern:   "settings.snapshot|section=<profile|tone|prompt|reasoning|tts|remembered_preferences>",
			ScopeKeyShape:        "app+session+user+workspace+locale+route+thread",
			PolicyClass:          parseSettingsPolicy.Class,
			StaleAfter:           parseSettingsPolicy.StaleAfter,
			ExpiresAfter:         parseSettingsPolicy.ExpiresAfter,
			InvalidationTriggers: []string{"settings_write_success", "settings_write_failure"},
			OfflineBehavior:      "show last snapshot with stale marker on failure",
			ConsistencyMode:      ExampleCacheConsistencyAuthoritativeAfterAck,
		},
		{
			ResourceID:           "model_catalog_metadata",
			ResourceKeyPattern:   "model_catalog.metadata|class=<providers|models|pricing|capabilities>|version=<server_version>|hash=<catalog_hash>",
			ScopeKeyShape:        "app+session+user+workspace+locale+route+thread",
			PolicyClass:          parseModelCatalogPolicy.Class,
			StaleAfter:           parseModelCatalogPolicy.StaleAfter,
			ExpiresAfter:         parseModelCatalogPolicy.ExpiresAfter,
			InvalidationTriggers: []string{"server_version_hash_mismatch", "catalog_changed"},
			OfflineBehavior:      "reuse briefly then refresh",
			ConsistencyMode:      ExampleCacheConsistencyDisplayOnly,
		},
		{
			ResourceID:           "billing_summary",
			ResourceKeyPattern:   "billing.summary.snapshot",
			ScopeKeyShape:        "app+session+user+workspace+locale+route+thread",
			PolicyClass:          parseBillingPolicy.Class,
			StaleAfter:           parseBillingPolicy.StaleAfter,
			ExpiresAfter:         parseBillingPolicy.ExpiresAfter,
			InvalidationTriggers: []string{"billing_refresh", "billing_unavailable"},
			OfflineBehavior:      "show stale snapshot metadata while server remains authoritative",
			ConsistencyMode:      ExampleCacheConsistencyDisplayOnly,
		},
		{
			ResourceID:           "admin_dashboard",
			ResourceKeyPattern:   "admin.dashboard.snapshot|role=<role_scope>|surface=<surface>|lookback_days=<range>",
			ScopeKeyShape:        "app+session+user+workspace+locale+route+thread",
			PolicyClass:          parseAdminPolicy.Class,
			StaleAfter:           parseAdminPolicy.StaleAfter,
			ExpiresAfter:         parseAdminPolicy.ExpiresAfter,
			InvalidationTriggers: []string{"role_change", "session_change", "workspace_change"},
			OfflineBehavior:      "show last short-lived snapshot only",
			ConsistencyMode:      ExampleCacheConsistencyDisplayOnly,
		},
		{
			ResourceID:           "canvas_session",
			ResourceKeyPattern:   "canvas.session|route=<route>",
			ScopeKeyShape:        "app+session+user+workspace+locale+route+thread",
			PolicyClass:          parseCanvasPolicy.Class,
			StaleAfter:           parseCanvasPolicy.StaleAfter,
			ExpiresAfter:         parseCanvasPolicy.ExpiresAfter,
			InvalidationTriggers: []string{"session_change", "workspace_change", "explicit_clear"},
			OfflineBehavior:      "restore lightweight state, exclude debug payloads",
			ConsistencyMode:      ExampleCacheConsistencyDisplayOnly,
		},
	}
}

// GetExampleCachePolicyEntry returns one policy map row by resource id.
func GetExampleCachePolicyEntry(parseResourceID string) (ExampleCachePolicyEntry, bool) {
	parseResourceID = strings.TrimSpace(strings.ToLower(parseResourceID))
	for _, parseEntry := range BuildExampleCachePolicyMap() {
		if strings.EqualFold(parseEntry.ResourceID, parseResourceID) {
			return parseEntry, true
		}
	}
	return ExampleCachePolicyEntry{}, false
}
