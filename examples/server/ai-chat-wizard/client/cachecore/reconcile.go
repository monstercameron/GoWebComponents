package cachecore

import (
	"context"
	"strings"
)

// AckReconcileInput describes one queue-ack reconciliation operation.
type AckReconcileInput struct {
	QueueKey                  string
	AckKey                    string
	OperationKey              string
	ScopeKey                  string
	ResourceKey               string
	AuthoritativeRecord       CacheRecordEnvelope
	OptimisticScopedResources []string
}

// AckReconcileResult describes one reconciliation outcome.
type AckReconcileResult struct {
	QueueRemovedCount int
	IsCacheUpdated    bool
}

// ApplyAckReconcile clears acked queue writes, merges authoritative records, and removes optimistic placeholders.
func ApplyAckReconcile(parseCtx context.Context, parseStorage Storage, parseInput AckReconcileInput) (AckReconcileResult, error) {
	if parseStorage == nil {
		return AckReconcileResult{}, nil
	}
	parseResult := AckReconcileResult{}
	parseQueueKey := strings.TrimSpace(parseInput.QueueKey)
	if parseQueueKey != "" {
		parseAckKey := strings.TrimSpace(parseInput.AckKey)
		parseOperationKey := strings.TrimSpace(parseInput.OperationKey)
		_, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []OutboxRecordEnvelope) ([]OutboxRecordEnvelope, error) {
			if len(parseQueue) == 0 {
				return parseQueue, nil
			}
			parseNextQueue := make([]OutboxRecordEnvelope, 0, len(parseQueue))
			for _, parseQueueRecord := range parseQueue {
				isParseAckMatched := parseAckKey != "" && strings.TrimSpace(parseQueueRecord.AckKey) == parseAckKey
				isParseOperationMatched := parseOperationKey != "" && strings.TrimSpace(parseQueueRecord.OperationKey) == parseOperationKey
				if isParseAckMatched || isParseOperationMatched {
					parseResult.QueueRemovedCount++
					continue
				}
				parseNextQueue = append(parseNextQueue, parseQueueRecord)
			}
			return parseNextQueue, nil
		})
		if parseErr != nil {
			return parseResult, parseErr
		}
	}
	parseAuthoritativeRecord := NormalizeCacheRecordEnvelope(parseInput.AuthoritativeRecord)
	if parseAuthoritativeRecord.ScopeKey == "" {
		parseAuthoritativeRecord.ScopeKey = strings.TrimSpace(parseInput.ScopeKey)
	}
	if parseAuthoritativeRecord.ResourceKey == "" {
		parseAuthoritativeRecord.ResourceKey = strings.TrimSpace(parseInput.ResourceKey)
	}
	if parseAuthoritativeRecord.ScopeKey != "" && parseAuthoritativeRecord.ResourceKey != "" {
		if parseErr := parseStorage.Set(parseCtx, parseAuthoritativeRecord); parseErr != nil {
			return parseResult, parseErr
		}
		parseResult.IsCacheUpdated = true
	}
	for _, parseScopedResourceKey := range parseInput.OptimisticScopedResources {
		parseScopedResourceKey = strings.TrimSpace(parseScopedResourceKey)
		if parseScopedResourceKey == "" {
			continue
		}
		if parseErr := parseStorage.Delete(parseCtx, parseScopedResourceKey); parseErr != nil {
			return parseResult, parseErr
		}
	}
	return parseResult, nil
}
