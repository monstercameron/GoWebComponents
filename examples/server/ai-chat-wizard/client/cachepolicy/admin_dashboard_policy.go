package cachepolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/client/cachecore"
)

const adminDashboardResourcePrefix = "admin.dashboard.snapshot"

// AdminDashboardSnapshotInput stores one admin-dashboard cache envelope keyed by role/surface/scope controls.
type AdminDashboardSnapshotInput struct {
	ScopeKey     string
	RoleScope    string
	Surface      string
	LookbackDays int
}

// AdminDashboardSnapshotPayload stores one cached admin dashboard payload with explicit last-updated metadata.
type AdminDashboardSnapshotPayload struct {
	LastUpdated string `json:"last_updated"`
	Payload     []byte `json:"payload"`
}

// BuildAdminDashboardSnapshotPolicy returns one short-lived admin-dashboard read cache policy.
func BuildAdminDashboardSnapshotPolicy() cachecore.CachePolicy {
	return cachecore.BuildCachePolicy(cachecore.PolicyClassVolatile, 15*time.Second, 2*time.Minute, true, true, false)
}

// BuildAdminDashboardResourceKey builds one admin-dashboard cache key keyed by role scope, surface, and lookback range.
func BuildAdminDashboardResourceKey(parseRoleScope string, parseSurface string, parseLookbackDays int) string {
	parseRoleScope = normalizeAdminDashboardRoleScope(parseRoleScope)
	parseSurface = normalizeAdminDashboardSurface(parseSurface)
	if parseLookbackDays <= 0 {
		parseLookbackDays = 7
	}
	return fmt.Sprintf("%s|role=%s|surface=%s|lookback_days=%d", adminDashboardResourcePrefix, parseRoleScope, parseSurface, parseLookbackDays)
}

// BuildAdminDashboardSnapshotPayloadJSON builds one payload JSON contract with explicit last-updated metadata.
func BuildAdminDashboardSnapshotPayloadJSON(parsePayload AdminDashboardSnapshotPayload) ([]byte, error) {
	parsePayload.LastUpdated = strings.TrimSpace(parsePayload.LastUpdated)
	if parsePayload.LastUpdated == "" {
		parsePayload.LastUpdated = time.Now().UTC().Format(time.RFC3339)
	}
	parsePayload.Payload = append([]byte(nil), parsePayload.Payload...)
	return json.Marshal(parsePayload)
}

// ParseAdminDashboardSnapshotPayloadJSON parses one stored payload JSON contract.
func ParseAdminDashboardSnapshotPayloadJSON(parseRaw []byte) (AdminDashboardSnapshotPayload, error) {
	parsePayload := AdminDashboardSnapshotPayload{}
	if parseErr := json.Unmarshal(parseRaw, &parsePayload); parseErr != nil {
		return AdminDashboardSnapshotPayload{}, parseErr
	}
	parsePayload.LastUpdated = strings.TrimSpace(parsePayload.LastUpdated)
	parsePayload.Payload = append([]byte(nil), parsePayload.Payload...)
	return parsePayload, nil
}

// StoreAdminDashboardSnapshot stores one short-lived admin dashboard snapshot.
func StoreAdminDashboardSnapshot(parseCtx context.Context, parseStorage cachecore.Storage, parseInput AdminDashboardSnapshotInput, parsePayload AdminDashboardSnapshotPayload) error {
	if parseStorage == nil {
		return nil
	}
	parseInput = normalizeAdminDashboardSnapshotInput(parseInput)
	parsePolicy := BuildAdminDashboardSnapshotPolicy()
	parsePayloadJSON, parseErr := BuildAdminDashboardSnapshotPayloadJSON(parsePayload)
	if parseErr != nil {
		return parseErr
	}
	parseRecord := cachecore.BuildCacheRecordEnvelope(
		parseInput.ScopeKey,
		BuildAdminDashboardResourceKey(parseInput.RoleScope, parseInput.Surface, parseInput.LookbackDays),
		"",
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

// ReadAdminDashboardSnapshot reads one admin dashboard snapshot with snapshot-first SWR behavior.
func ReadAdminDashboardSnapshot(
	parseCtx context.Context,
	parseAPI *cachecore.UIAPI,
	parseInput AdminDashboardSnapshotInput,
	parseRefresh func(context.Context, AdminDashboardSnapshotInput) (cachecore.CacheRecordEnvelope, error),
) (cachecore.CachedResourceView, error) {
	if parseAPI == nil {
		return cachecore.CachedResourceView{}, nil
	}
	parseInput = normalizeAdminDashboardSnapshotInput(parseInput)
	parseResourceKey := BuildAdminDashboardResourceKey(parseInput.RoleScope, parseInput.Surface, parseInput.LookbackDays)
	return parseAPI.ReadCachedResource(parseCtx, parseInput.ScopeKey, parseResourceKey, BuildAdminDashboardSnapshotPolicy(), func(parseCtx context.Context, parseScopeKey string, parseResourceKey string) (cachecore.CacheRecordEnvelope, error) {
		_ = parseScopeKey
		_ = parseResourceKey
		if parseRefresh == nil {
			return cachecore.CacheRecordEnvelope{}, nil
		}
		return parseRefresh(parseCtx, parseInput)
	})
}

// ApplyAdminDashboardHardInvalidation evicts all admin dashboard cache snapshots for role/session/workspace scope changes.
func ApplyAdminDashboardHardInvalidation(parseCtx context.Context, parseStorage cachecore.Storage, parseSnapshot *cachecore.SnapshotStore) (int, error) {
	if parseStorage == nil {
		return 0, nil
	}
	parseRecords, parseErr := parseStorage.List(parseCtx, "")
	if parseErr != nil {
		return 0, parseErr
	}
	parseEvictedCount := 0
	for _, parseRecord := range parseRecords {
		if !strings.HasPrefix(strings.TrimSpace(parseRecord.ResourceKey), adminDashboardResourcePrefix+"|") {
			continue
		}
		parseScopedResourceKey := cachecore.BuildScopedResourceKey(parseRecord.ScopeKey, parseRecord.ResourceKey)
		if parseErr = parseStorage.Delete(parseCtx, parseScopedResourceKey); parseErr != nil {
			return parseEvictedCount, parseErr
		}
		if parseSnapshot != nil {
			parseSnapshot.DeleteSync(parseRecord.ScopeKey, parseRecord.ResourceKey)
		}
		parseEvictedCount++
	}
	return parseEvictedCount, nil
}

// normalizeAdminDashboardSnapshotInput normalizes one admin dashboard snapshot input envelope.
func normalizeAdminDashboardSnapshotInput(parseInput AdminDashboardSnapshotInput) AdminDashboardSnapshotInput {
	parseInput.ScopeKey = strings.TrimSpace(parseInput.ScopeKey)
	parseInput.RoleScope = normalizeAdminDashboardRoleScope(parseInput.RoleScope)
	parseInput.Surface = normalizeAdminDashboardSurface(parseInput.Surface)
	if parseInput.LookbackDays <= 0 {
		parseInput.LookbackDays = 7
	}
	return parseInput
}

// normalizeAdminDashboardRoleScope normalizes one admin role scope token for cache keying.
func normalizeAdminDashboardRoleScope(parseRoleScope string) string {
	parseRoleScope = strings.TrimSpace(strings.ToLower(parseRoleScope))
	switch parseRoleScope {
	case "user", "workspace_admin", "superuser":
		return parseRoleScope
	default:
		return "user"
	}
}

// normalizeAdminDashboardSurface normalizes one dashboard surface token for cache keying.
func normalizeAdminDashboardSurface(parseSurface string) string {
	parseSurface = strings.TrimSpace(strings.ToLower(parseSurface))
	switch parseSurface {
	case "business", "customers", "chats", "providers", "ops":
		return parseSurface
	default:
		return "business"
	}
}
