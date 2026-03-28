package cachepolicy

import (
	"context"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/client/cachecore"
)

const undeliveredLogQueueResourceKey = "outbox.undelivered_logs"
const undeliveredLogResourceKey = "diagnostics.log"

// UndeliveredLogOutboxPolicy stores retry/drop settings for undelivered diagnostic logs.
type UndeliveredLogOutboxPolicy struct {
	MaxAttempts int
	MaxAge      time.Duration
	RetryPolicy cachecore.OutboxRetryPolicy
}

// BuildUndeliveredLogOutboxPolicy returns one default undelivered-log outbox policy.
func BuildUndeliveredLogOutboxPolicy() UndeliveredLogOutboxPolicy {
	return UndeliveredLogOutboxPolicy{
		MaxAttempts: 10,
		MaxAge:      72 * time.Hour,
		RetryPolicy: cachecore.NormalizeOutboxRetryPolicy(cachecore.OutboxRetryPolicy{
			InitialDelay: 500 * time.Millisecond,
			MaxDelay:     30 * time.Second,
			JitterRatio:  0,
			MaxAttempts:  10,
		}),
	}
}

// BuildUndeliveredLogQueueKey builds one scoped queue key for undelivered client diagnostics logs.
func BuildUndeliveredLogQueueKey(parseScopeKey string) string {
	return cachecore.BuildScopedResourceKey(strings.TrimSpace(parseScopeKey), undeliveredLogQueueResourceKey)
}

// StoreUndeliveredLog appends one undelivered diagnostics log payload into the retry queue.
func StoreUndeliveredLog(parseCtx context.Context, parseStorage cachecore.Storage, parseScopeKey string, parseOperationKey string, parsePayload []byte) error {
	if parseStorage == nil {
		return nil
	}
	parseScopeKey = strings.TrimSpace(parseScopeKey)
	parseOperationKey = strings.TrimSpace(parseOperationKey)
	parseQueueKey := BuildUndeliveredLogQueueKey(parseScopeKey)
	_, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []cachecore.OutboxRecordEnvelope) ([]cachecore.OutboxRecordEnvelope, error) {
		parseRecord := cachecore.BuildOutboxRecordEnvelope(
			parseQueueKey,
			undeliveredLogResourceKey,
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

// GetUndeliveredLogQueue lists queued undelivered log records for one scope.
func GetUndeliveredLogQueue(parseCtx context.Context, parseStorage cachecore.Storage, parseScopeKey string) ([]cachecore.OutboxRecordEnvelope, error) {
	if parseStorage == nil {
		return []cachecore.OutboxRecordEnvelope{}, nil
	}
	return parseStorage.UpdateQueueAtomic(parseCtx, BuildUndeliveredLogQueueKey(parseScopeKey), func(parseQueue []cachecore.OutboxRecordEnvelope) ([]cachecore.OutboxRecordEnvelope, error) {
		return parseQueue, nil
	})
}

// GetUndeliveredLogRetryBatchOnReconnect returns one send batch on reconnect, updates retry metadata, and drops capped records.
func GetUndeliveredLogRetryBatchOnReconnect(parseCtx context.Context, parseStorage cachecore.Storage, parseScopeKey string, parseNow time.Time, parsePolicy UndeliveredLogOutboxPolicy) ([]cachecore.OutboxRecordEnvelope, error) {
	if parseStorage == nil {
		return []cachecore.OutboxRecordEnvelope{}, nil
	}
	parseNow = parseNow.UTC()
	parsePolicy = normalizeUndeliveredLogOutboxPolicy(parsePolicy)
	parseSendBatch := []cachecore.OutboxRecordEnvelope{}
	_, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, BuildUndeliveredLogQueueKey(parseScopeKey), func(parseQueue []cachecore.OutboxRecordEnvelope) ([]cachecore.OutboxRecordEnvelope, error) {
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

// ApplyUndeliveredLogAck removes one acked undelivered log by operation key.
func ApplyUndeliveredLogAck(parseCtx context.Context, parseStorage cachecore.Storage, parseScopeKey string, parseOperationKey string) (int, error) {
	if parseStorage == nil {
		return 0, nil
	}
	parseOperationKey = strings.TrimSpace(parseOperationKey)
	parseRemovedCount := 0
	_, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, BuildUndeliveredLogQueueKey(parseScopeKey), func(parseQueue []cachecore.OutboxRecordEnvelope) ([]cachecore.OutboxRecordEnvelope, error) {
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

// normalizeUndeliveredLogOutboxPolicy normalizes one undelivered-log policy to safe defaults.
func normalizeUndeliveredLogOutboxPolicy(parsePolicy UndeliveredLogOutboxPolicy) UndeliveredLogOutboxPolicy {
	parseDefaults := BuildUndeliveredLogOutboxPolicy()
	if parsePolicy.MaxAttempts <= 0 {
		parsePolicy.MaxAttempts = parseDefaults.MaxAttempts
	}
	if parsePolicy.MaxAge <= 0 {
		parsePolicy.MaxAge = parseDefaults.MaxAge
	}
	parsePolicy.RetryPolicy = cachecore.NormalizeOutboxRetryPolicy(parsePolicy.RetryPolicy)
	return parsePolicy
}

// isUndeliveredLogRecordExpired reports whether one outbox record exceeds one age limit.
func isUndeliveredLogRecordExpired(parseRecord cachecore.OutboxRecordEnvelope, parseNow time.Time, parseMaxAge time.Duration) bool {
	if parseMaxAge <= 0 {
		return false
	}
	parseCreatedAt, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(parseRecord.CreatedAt))
	if parseErr != nil {
		return false
	}
	return parseNow.Sub(parseCreatedAt.UTC()) > parseMaxAge
}

// isOutboxRetryDue reports whether one outbox record is due for retry at one wall clock timestamp.
func isOutboxRetryDue(parseRecord cachecore.OutboxRecordEnvelope, parseNow time.Time) bool {
	parseNextRetryAt := strings.TrimSpace(parseRecord.NextRetryAt)
	if parseNextRetryAt == "" {
		return true
	}
	parseParsed, parseErr := time.Parse(time.RFC3339, parseNextRetryAt)
	if parseErr != nil {
		return true
	}
	return !parseParsed.UTC().After(parseNow.UTC())
}
