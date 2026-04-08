package cachecore

import (
	"context"
	"strings"
)

const (
	InvalidationReasonLogout                = "logout"
	InvalidationReasonSessionRevoked        = "session_revoked"
	InvalidationReasonWorkspaceSwitch       = "workspace_switch"
	InvalidationReasonLocaleChange          = "locale_change"
	InvalidationReasonAppVersionChange      = "app_version_change"
	InvalidationReasonMutationAck           = "mutation_ack"
	InvalidationReasonRouteThreadNormalize  = "route_thread_normalization"
	InvalidationReasonServerVersionMismatch = "server_version_hash_mismatch"
)

// InvalidationEngine applies reason-based cache eviction/downgrade behavior.
type InvalidationEngine struct {
	parseStorage     Storage
	parseInvalidator *Invalidator
}

// BuildInvalidationEngine creates one invalidation engine bound to storage and hook dispatcher.
func BuildInvalidationEngine(parseStorage Storage, parseInvalidator *Invalidator) *InvalidationEngine {
	return &InvalidationEngine{
		parseStorage:     parseStorage,
		parseInvalidator: parseInvalidator,
	}
}

// ApplyInvalidation applies one reason-based invalidation event and returns affected-record count.
func (parseEngine *InvalidationEngine) ApplyInvalidation(parseCtx context.Context, parseEvent InvalidationEvent) (int, error) {
	if parseEngine == nil || parseEngine.parseStorage == nil {
		return 0, nil
	}
	parseEvent.Reason = normalizeInvalidationReason(parseEvent.Reason)
	if parseEngine.parseInvalidator != nil {
		parseEngine.parseInvalidator.ApplyInvalidation(parseEvent)
	}
	switch parseEvent.Reason {
	case InvalidationReasonServerVersionMismatch:
		return parseEngine.applyDowngradeToStale(parseCtx, parseEvent)
	case InvalidationReasonMutationAck:
		return parseEngine.applyResourcePrefixDelete(parseCtx, parseEvent)
	default:
		return parseEngine.parseStorage.DeleteByPrefix(parseCtx, strings.TrimSpace(parseEvent.ScopePrefix))
	}
}

// applyDowngradeToStale downgrades matching records to stale state instead of full eviction.
func (parseEngine *InvalidationEngine) applyDowngradeToStale(parseCtx context.Context, parseEvent InvalidationEvent) (int, error) {
	parseRecords, parseErr := parseEngine.parseStorage.List(parseCtx, strings.TrimSpace(parseEvent.ScopePrefix))
	if parseErr != nil {
		return 0, parseErr
	}
	parseAffectedCount := 0
	for _, parseRecord := range parseRecords {
		if strings.TrimSpace(parseEvent.ResourcePrefix) != "" && !strings.HasPrefix(parseRecord.ResourceKey, strings.TrimSpace(parseEvent.ResourcePrefix)) {
			continue
		}
		parseRecord.Status = CacheStatusStale
		parseRecord.ETagOrHash = ""
		if parseErr = parseEngine.parseStorage.Set(parseCtx, parseRecord); parseErr != nil {
			return parseAffectedCount, parseErr
		}
		parseAffectedCount++
	}
	return parseAffectedCount, nil
}

// applyResourcePrefixDelete evicts records for one scope prefix and one optional resource prefix filter.
func (parseEngine *InvalidationEngine) applyResourcePrefixDelete(parseCtx context.Context, parseEvent InvalidationEvent) (int, error) {
	parseRecords, parseErr := parseEngine.parseStorage.List(parseCtx, strings.TrimSpace(parseEvent.ScopePrefix))
	if parseErr != nil {
		return 0, parseErr
	}
	parseAffectedCount := 0
	for _, parseRecord := range parseRecords {
		if strings.TrimSpace(parseEvent.ResourcePrefix) != "" && !strings.HasPrefix(parseRecord.ResourceKey, strings.TrimSpace(parseEvent.ResourcePrefix)) {
			continue
		}
		parseScopedResourceKey := BuildScopedResourceKey(parseRecord.ScopeKey, parseRecord.ResourceKey)
		if parseErr = parseEngine.parseStorage.Delete(parseCtx, parseScopedResourceKey); parseErr != nil {
			return parseAffectedCount, parseErr
		}
		parseAffectedCount++
	}
	return parseAffectedCount, nil
}

// normalizeInvalidationReason resolves one canonical invalidation reason fallback.
func normalizeInvalidationReason(parseReason string) string {
	switch strings.TrimSpace(strings.ToLower(parseReason)) {
	case InvalidationReasonSessionRevoked:
		return InvalidationReasonSessionRevoked
	case InvalidationReasonWorkspaceSwitch:
		return InvalidationReasonWorkspaceSwitch
	case InvalidationReasonLocaleChange:
		return InvalidationReasonLocaleChange
	case InvalidationReasonAppVersionChange:
		return InvalidationReasonAppVersionChange
	case InvalidationReasonMutationAck:
		return InvalidationReasonMutationAck
	case InvalidationReasonRouteThreadNormalize:
		return InvalidationReasonRouteThreadNormalize
	case InvalidationReasonServerVersionMismatch:
		return InvalidationReasonServerVersionMismatch
	default:
		return InvalidationReasonLogout
	}
}
