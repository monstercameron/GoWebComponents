package cachepolicy

import (
	"context"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/client/cachecore"
)

const unsentMessageQueuePrefix = "outbox.unsent_messages.thread="
const unsentMessageResourceKey = "chat.unsent_message"
const unsentMessageNewThreadKey = "new_chat"

// UnsentMessageOutboxPolicy stores retry/drop settings for unsent chat messages.
type UnsentMessageOutboxPolicy struct {
	MaxAttempts int
	MaxAge      time.Duration
	RetryPolicy cachecore.OutboxRetryPolicy
}

// BuildUnsentMessageOutboxPolicy returns one default unsent-message outbox policy.
func BuildUnsentMessageOutboxPolicy() UnsentMessageOutboxPolicy {
	return UnsentMessageOutboxPolicy{
		MaxAttempts: 20,
		MaxAge:      7 * 24 * time.Hour,
		RetryPolicy: cachecore.BuildOutboxRetryPolicyDefaults(),
	}
}

// BuildUnsentMessageQueueKey builds one scoped unsent-message queue key for one canonical thread route context.
func BuildUnsentMessageQueueKey(parseScopeKey string, parseThreadRoutePublicID string) string {
	parseScopeKey = strings.TrimSpace(parseScopeKey)
	parseThreadKey := normalizeUnsentMessageThreadKey(parseThreadRoutePublicID)
	return cachecore.BuildScopedResourceKey(parseScopeKey, unsentMessageQueuePrefix+parseThreadKey)
}

// StoreUnsentMessage appends one unsent message payload into one thread-scoped outbox queue and dedupes by operation key.
func StoreUnsentMessage(parseCtx context.Context, parseStorage cachecore.Storage, parseScopeKey string, parseThreadRoutePublicID string, parseOperationKey string, parsePayload []byte) error {
	if parseStorage == nil {
		return nil
	}
	parseQueueKey := BuildUnsentMessageQueueKey(parseScopeKey, parseThreadRoutePublicID)
	parseOperationKey = strings.TrimSpace(parseOperationKey)
	_, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []cachecore.OutboxRecordEnvelope) ([]cachecore.OutboxRecordEnvelope, error) {
		for _, parseRecord := range parseQueue {
			if parseOperationKey != "" && strings.TrimSpace(parseRecord.OperationKey) == parseOperationKey {
				return parseQueue, nil
			}
		}
		parseRecord := cachecore.BuildOutboxRecordEnvelope(
			parseQueueKey,
			unsentMessageResourceKey,
			parseOperationKey,
			time.Now().UTC(),
			time.Now().UTC(),
			append([]byte(nil), parsePayload...),
			cachecore.OutboxStatusQueued,
			"",
			"",
		)
		return append(parseQueue, parseRecord), nil
	})
	return parseErr
}

// GetUnsentMessageQueue lists unsent messages queued for one canonical thread route context.
func GetUnsentMessageQueue(parseCtx context.Context, parseStorage cachecore.Storage, parseScopeKey string, parseThreadRoutePublicID string) ([]cachecore.OutboxRecordEnvelope, error) {
	if parseStorage == nil {
		return []cachecore.OutboxRecordEnvelope{}, nil
	}
	parseQueueKey := BuildUnsentMessageQueueKey(parseScopeKey, parseThreadRoutePublicID)
	return parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []cachecore.OutboxRecordEnvelope) ([]cachecore.OutboxRecordEnvelope, error) {
		return parseQueue, nil
	})
}

// GetUnsentMessageRetryBatchOnReconnect returns one reconnect send batch and updates retry metadata in-place.
func GetUnsentMessageRetryBatchOnReconnect(parseCtx context.Context, parseStorage cachecore.Storage, parseScopeKey string, parseThreadRoutePublicID string, parseNow time.Time, parsePolicy UnsentMessageOutboxPolicy) ([]cachecore.OutboxRecordEnvelope, error) {
	if parseStorage == nil {
		return []cachecore.OutboxRecordEnvelope{}, nil
	}
	parseNow = parseNow.UTC()
	parsePolicy = normalizeUnsentMessageOutboxPolicy(parsePolicy)
	parseSendBatch := []cachecore.OutboxRecordEnvelope{}
	parseQueueKey := BuildUnsentMessageQueueKey(parseScopeKey, parseThreadRoutePublicID)
	_, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []cachecore.OutboxRecordEnvelope) ([]cachecore.OutboxRecordEnvelope, error) {
		parseNextQueue := make([]cachecore.OutboxRecordEnvelope, 0, len(parseQueue))
		for _, parseRecord := range parseQueue {
			parseRecord = cachecore.NormalizeOutboxRecordEnvelope(parseRecord)
			if isUndeliveredLogRecordExpired(parseRecord, parseNow, parsePolicy.MaxAge) || parseRecord.AttemptCount >= parsePolicy.MaxAttempts {
				continue
			}
			if !isOutboxRetryDue(parseRecord, parseNow) {
				parseNextQueue = append(parseNextQueue, parseRecord)
				continue
			}
			parseRetryDecision := cachecore.ResolveOutboxRetryDecision(parseRecord, parseNow, parsePolicy.RetryPolicy, nil)
			if parseRetryDecision.IsTerminal {
				continue
			}
			if !parseRetryDecision.ShouldRetry {
				parseNextQueue = append(parseNextQueue, parseRecord)
				continue
			}
			parseRecord = cachecore.NormalizeOutboxRecordEnvelope(parseRetryDecision.NextRecord)
			parseRecord.Status = cachecore.OutboxStatusSending
			parseSendBatch = append(parseSendBatch, parseRecord)
			parseNextQueue = append(parseNextQueue, parseRecord)
		}
		return parseNextQueue, nil
	})
	return parseSendBatch, parseErr
}

// ApplyUnsentMessageAck removes one acked unsent message from one thread-scoped queue.
func ApplyUnsentMessageAck(parseCtx context.Context, parseStorage cachecore.Storage, parseScopeKey string, parseThreadRoutePublicID string, parseOperationKey string) (int, error) {
	if parseStorage == nil {
		return 0, nil
	}
	parseQueueKey := BuildUnsentMessageQueueKey(parseScopeKey, parseThreadRoutePublicID)
	parseOperationKey = strings.TrimSpace(parseOperationKey)
	parseRemovedCount := 0
	_, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []cachecore.OutboxRecordEnvelope) ([]cachecore.OutboxRecordEnvelope, error) {
		parseNextQueue := make([]cachecore.OutboxRecordEnvelope, 0, len(parseQueue))
		for _, parseRecord := range parseQueue {
			if parseOperationKey != "" && strings.TrimSpace(parseRecord.OperationKey) == parseOperationKey {
				parseRemovedCount++
				continue
			}
			parseNextQueue = append(parseNextQueue, parseRecord)
		}
		return parseNextQueue, nil
	})
	return parseRemovedCount, parseErr
}

// ApplyUnsentMessageRouteNormalization moves queue entries from one thread context into one canonical thread context with op-key dedupe.
func ApplyUnsentMessageRouteNormalization(parseCtx context.Context, parseStorage cachecore.Storage, parseScopeKey string, parseFromThreadRoutePublicID string, parseToThreadRoutePublicID string) (int, error) {
	if parseStorage == nil {
		return 0, nil
	}
	parseFromQueueKey := BuildUnsentMessageQueueKey(parseScopeKey, parseFromThreadRoutePublicID)
	parseToQueueKey := BuildUnsentMessageQueueKey(parseScopeKey, parseToThreadRoutePublicID)
	if parseFromQueueKey == parseToQueueKey {
		return 0, nil
	}
	parseSourceQueue := []cachecore.OutboxRecordEnvelope{}
	if _, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseFromQueueKey, func(parseQueue []cachecore.OutboxRecordEnvelope) ([]cachecore.OutboxRecordEnvelope, error) {
		parseSourceQueue = append(parseSourceQueue, parseQueue...)
		return []cachecore.OutboxRecordEnvelope{}, nil
	}); parseErr != nil {
		return 0, parseErr
	}
	if len(parseSourceQueue) == 0 {
		return 0, nil
	}
	parseMovedCount := 0
	_, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseToQueueKey, func(parseQueue []cachecore.OutboxRecordEnvelope) ([]cachecore.OutboxRecordEnvelope, error) {
		parseExistingByOperationKey := map[string]struct{}{}
		for _, parseRecord := range parseQueue {
			parseOperationKey := strings.TrimSpace(parseRecord.OperationKey)
			if parseOperationKey == "" {
				continue
			}
			parseExistingByOperationKey[parseOperationKey] = struct{}{}
		}
		parseNextQueue := append([]cachecore.OutboxRecordEnvelope(nil), parseQueue...)
		for _, parseSourceRecord := range parseSourceQueue {
			parseSourceRecord = cachecore.NormalizeOutboxRecordEnvelope(parseSourceRecord)
			parseSourceRecord.QueueKey = parseToQueueKey
			parseOperationKey := strings.TrimSpace(parseSourceRecord.OperationKey)
			if parseOperationKey != "" {
				if _, isParseExists := parseExistingByOperationKey[parseOperationKey]; isParseExists {
					continue
				}
				parseExistingByOperationKey[parseOperationKey] = struct{}{}
			}
			parseNextQueue = append(parseNextQueue, parseSourceRecord)
			parseMovedCount++
		}
		return parseNextQueue, nil
	})
	return parseMovedCount, parseErr
}

// normalizeUnsentMessageOutboxPolicy normalizes one unsent-message outbox policy to safe defaults.
func normalizeUnsentMessageOutboxPolicy(parsePolicy UnsentMessageOutboxPolicy) UnsentMessageOutboxPolicy {
	parseDefaults := BuildUnsentMessageOutboxPolicy()
	if parsePolicy.MaxAttempts <= 0 {
		parsePolicy.MaxAttempts = parseDefaults.MaxAttempts
	}
	if parsePolicy.MaxAge <= 0 {
		parsePolicy.MaxAge = parseDefaults.MaxAge
	}
	parsePolicy.RetryPolicy = cachecore.NormalizeOutboxRetryPolicy(parsePolicy.RetryPolicy)
	return parsePolicy
}

// normalizeUnsentMessageThreadKey normalizes one thread-route key with one explicit new-chat fallback bucket.
func normalizeUnsentMessageThreadKey(parseThreadRoutePublicID string) string {
	parseThreadRoutePublicID = strings.TrimSpace(parseThreadRoutePublicID)
	if parseThreadRoutePublicID == "" {
		return unsentMessageNewThreadKey
	}
	return parseThreadRoutePublicID
}
