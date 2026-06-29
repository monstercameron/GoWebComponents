package cachepolicy

import (
	"context"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/client/cachecore"
)

const rememberedPreferenceQueueResourceKey = "outbox.settings.remembered_preferences"

// ReconnectConflictResolutionInput stores reconnect conflict-resolution inputs across chat, settings, and pending preference edits.
type ReconnectConflictResolutionInput struct {
	ScopeKey                        string
	CanonicalThreadRoutePublicID    string
	UnsentSourceThreadRoutePublicID string
	ThreadUpdatedAt                 time.Time
	AuthoritativeThreadRecord       cachecore.CacheRecordEnvelope
	AuthoritativeSettingsBySection  map[string]cachecore.CacheRecordEnvelope
	AckedPreferenceOperationKeys    []string
}

// ReconnectConflictResolutionResult stores one reconnect conflict-resolution summary.
type ReconnectConflictResolutionResult struct {
	UnsentMovedCount     int
	UnsentDedupedCount   int
	PreferenceAckedCount int
	SettingsUpdatedCount int
	IsThreadUpdated      bool
}

// ApplyReconnectConflictResolution applies deterministic reconnect reconciliation for unsent messages, settings snapshots, and preference edits.
func ApplyReconnectConflictResolution(
	parseCtx context.Context,
	parseStorage cachecore.Storage,
	parseSnapshot *cachecore.SnapshotStore,
	parseReconciler *cachecore.WorkerSnapshotReconciler,
	parseInput ReconnectConflictResolutionInput,
) (ReconnectConflictResolutionResult, error) {
	if parseStorage == nil {
		return ReconnectConflictResolutionResult{}, nil
	}
	parseInput = normalizeReconnectConflictResolutionInput(parseInput)
	parseResult := ReconnectConflictResolutionResult{}

	parseMovedCount, parseErr := ApplyUnsentMessageRouteNormalization(
		parseCtx,
		parseStorage,
		parseInput.ScopeKey,
		parseInput.UnsentSourceThreadRoutePublicID,
		parseInput.CanonicalThreadRoutePublicID,
	)
	if parseErr != nil {
		return parseResult, parseErr
	}
	parseResult.UnsentMovedCount = parseMovedCount

	parseDedupedCount, parseErr := applyUnsentMessageQueueDedupe(parseCtx, parseStorage, parseInput.ScopeKey, parseInput.CanonicalThreadRoutePublicID)
	if parseErr != nil {
		return parseResult, parseErr
	}
	parseResult.UnsentDedupedCount = parseDedupedCount

	if parseInput.AuthoritativeThreadRecord.ResourceKey != "" || parseInput.AuthoritativeThreadRecord.Payload != nil {
		parseThreadResult, parseThreadErr := ApplyThreadHistoryAuthoritativeReconcile(
			parseCtx,
			parseReconciler,
			parseInput.ScopeKey,
			parseInput.CanonicalThreadRoutePublicID,
			parseInput.ThreadUpdatedAt,
			parseInput.AuthoritativeThreadRecord,
		)
		if parseThreadErr != nil {
			return parseResult, parseThreadErr
		}
		parseResult.IsThreadUpdated = parseThreadResult.IsRecordUpdated
	}

	for parseSection, parseRecord := range parseInput.AuthoritativeSettingsBySection {
		parseSection = normalizeSettingsSnapshotSection(parseSection)
		if parseErr = ApplySettingsSnapshotWriteSuccess(parseCtx, parseStorage, parseSnapshot, parseInput.ScopeKey, parseSection, strings.TrimSpace(parseRecord.Version), parseRecord.Payload); parseErr != nil {
			return parseResult, parseErr
		}
		parseResult.SettingsUpdatedCount++
	}

	parseAckedCount, parseErr := applyRememberedPreferenceOperationAcks(parseCtx, parseStorage, parseInput.ScopeKey, parseInput.AckedPreferenceOperationKeys)
	if parseErr != nil {
		return parseResult, parseErr
	}
	parseResult.PreferenceAckedCount = parseAckedCount
	return parseResult, nil
}

// normalizeReconnectConflictResolutionInput normalizes one reconnect conflict-resolution input envelope.
func normalizeReconnectConflictResolutionInput(parseInput ReconnectConflictResolutionInput) ReconnectConflictResolutionInput {
	parseInput.ScopeKey = strings.TrimSpace(parseInput.ScopeKey)
	parseInput.CanonicalThreadRoutePublicID = strings.TrimSpace(parseInput.CanonicalThreadRoutePublicID)
	parseInput.UnsentSourceThreadRoutePublicID = strings.TrimSpace(parseInput.UnsentSourceThreadRoutePublicID)
	if parseInput.ThreadUpdatedAt.IsZero() {
		parseInput.ThreadUpdatedAt = time.Now().UTC()
	}
	if parseInput.AuthoritativeSettingsBySection == nil {
		parseInput.AuthoritativeSettingsBySection = map[string]cachecore.CacheRecordEnvelope{}
	}
	return parseInput
}

// applyUnsentMessageQueueDedupe removes duplicate unsent-message rows by operation key for one canonical thread queue.
func applyUnsentMessageQueueDedupe(parseCtx context.Context, parseStorage cachecore.Storage, parseScopeKey string, parseThreadRoutePublicID string) (int, error) {
	parseQueueKey := BuildUnsentMessageQueueKey(parseScopeKey, parseThreadRoutePublicID)
	parseDedupedCount := 0
	_, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []cachecore.OutboxRecordEnvelope) ([]cachecore.OutboxRecordEnvelope, error) {
		parseSeenByOperationKey := map[string]struct{}{}
		parseNextQueue := make([]cachecore.OutboxRecordEnvelope, 0, len(parseQueue))
		for _, parseRecord := range parseQueue {
			parseRecord = cachecore.NormalizeOutboxRecordEnvelope(parseRecord)
			parseOperationKey := strings.TrimSpace(parseRecord.OperationKey)
			if parseOperationKey != "" {
				if _, isParseSeen := parseSeenByOperationKey[parseOperationKey]; isParseSeen {
					parseDedupedCount++
					continue
				}
				parseSeenByOperationKey[parseOperationKey] = struct{}{}
			}
			parseNextQueue = append(parseNextQueue, parseRecord)
		}
		return parseNextQueue, nil
	})
	return parseDedupedCount, parseErr
}

// applyRememberedPreferenceOperationAcks removes acked remembered-preference edit operations from queue state.
func applyRememberedPreferenceOperationAcks(parseCtx context.Context, parseStorage cachecore.Storage, parseScopeKey string, parseOperationKeys []string) (int, error) {
	parseQueueKey := cachecore.BuildScopedResourceKey(strings.TrimSpace(parseScopeKey), rememberedPreferenceQueueResourceKey)
	parseAckSet := map[string]struct{}{}
	for _, parseOperationKey := range parseOperationKeys {
		parseOperationKey = strings.TrimSpace(parseOperationKey)
		if parseOperationKey == "" {
			continue
		}
		parseAckSet[parseOperationKey] = struct{}{}
	}
	if len(parseAckSet) == 0 {
		return 0, nil
	}
	parseAckedCount := 0
	_, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []cachecore.OutboxRecordEnvelope) ([]cachecore.OutboxRecordEnvelope, error) {
		parseNextQueue := make([]cachecore.OutboxRecordEnvelope, 0, len(parseQueue))
		for _, parseRecord := range parseQueue {
			parseOperationKey := strings.TrimSpace(parseRecord.OperationKey)
			if _, isParseAcked := parseAckSet[parseOperationKey]; isParseAcked {
				parseAckedCount++
				continue
			}
			parseNextQueue = append(parseNextQueue, parseRecord)
		}
		return parseNextQueue, nil
	})
	return parseAckedCount, parseErr
}
