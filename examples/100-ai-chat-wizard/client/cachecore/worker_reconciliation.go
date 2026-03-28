package cachecore

import (
	"context"
	"strings"
	"sync"
	"time"
)

// WorkerSnapshotReconcileInput describes one worker->snapshot reconciliation update.
type WorkerSnapshotReconcileInput struct {
	Event                    WorkerEvent
	Record                   CacheRecordEnvelope
	AckQueueKey              string
	AckKey                   string
	AckOperationKey          string
	InvalidatedScopedRecords []string
}

// WorkerSnapshotReconcileResult describes one reconciliation application outcome.
type WorkerSnapshotReconcileResult struct {
	IsApplied         bool
	IsDuplicate       bool
	IsOutOfOrder      bool
	IsRecordUpdated   bool
	DeletedCount      int
	QueueRemovedCount int
}

// WorkerSnapshotReconciler merges worker maintenance results into storage plus snapshot state deterministically.
type WorkerSnapshotReconciler struct {
	parseMu                 sync.Mutex
	parseStorage            Storage
	parseSnapshot           *SnapshotStore
	parseAppliedEventSet    map[string]struct{}
	parseLastResourceUpdate map[string]time.Time
}

// BuildWorkerSnapshotReconciler creates one deterministic worker->snapshot reconciliation path.
func BuildWorkerSnapshotReconciler(parseStorage Storage, parseSnapshot *SnapshotStore) *WorkerSnapshotReconciler {
	return &WorkerSnapshotReconciler{
		parseStorage:            parseStorage,
		parseSnapshot:           parseSnapshot,
		parseAppliedEventSet:    map[string]struct{}{},
		parseLastResourceUpdate: map[string]time.Time{},
	}
}

// ApplyEvent applies one worker maintenance result into storage plus snapshot state.
func (parseReconciler *WorkerSnapshotReconciler) ApplyEvent(parseCtx context.Context, parseInput WorkerSnapshotReconcileInput) (WorkerSnapshotReconcileResult, error) {
	if parseReconciler == nil {
		return WorkerSnapshotReconcileResult{}, nil
	}
	parseReconciler.parseMu.Lock()
	defer parseReconciler.parseMu.Unlock()

	parseResult := WorkerSnapshotReconcileResult{}
	parseEvent := normalizeWorkerEvent(parseInput.Event)
	parseEventKey := buildWorkerEventFingerprint(parseEvent, parseInput.AckQueueKey, parseInput.AckKey, parseInput.AckOperationKey, parseInput.InvalidatedScopedRecords)
	if _, isParseApplied := parseReconciler.parseAppliedEventSet[parseEventKey]; isParseApplied {
		parseResult.IsDuplicate = true
		return parseResult, nil
	}
	parseResourceScopedKey := BuildScopedResourceKey(parseEvent.ScopeKey, parseEvent.ResourceKey)
	parseEventUpdatedAt, isParseEventUpdatedAt := parseTimeRFC3339(parseEvent.UpdatedAt)
	if parseResourceScopedKey != "" && isParseEventUpdatedAt {
		if parseLastUpdatedAt, isParseFound := parseReconciler.parseLastResourceUpdate[parseResourceScopedKey]; isParseFound && parseEventUpdatedAt.Before(parseLastUpdatedAt) {
			parseResult.IsOutOfOrder = true
			parseReconciler.parseAppliedEventSet[parseEventKey] = struct{}{}
			return parseResult, nil
		}
	}

	for _, parseScopedRecord := range parseInput.InvalidatedScopedRecords {
		if parseReconciler.parseStorage != nil {
			if parseErr := parseReconciler.parseStorage.Delete(parseCtx, strings.TrimSpace(parseScopedRecord)); parseErr != nil {
				return parseResult, parseErr
			}
		}
		parseScopeKey, parseResourceKey := parseSplitScopedResourceKey(parseScopedRecord)
		if parseReconciler.parseSnapshot != nil && parseScopeKey != "" && parseResourceKey != "" {
			parseReconciler.parseSnapshot.DeleteSync(parseScopeKey, parseResourceKey)
		}
		parseResult.DeletedCount++
	}

	if parseEvent.EventType == WorkerEventEvicted {
		parseScopeKey := strings.TrimSpace(parseEvent.ScopeKey)
		parseResourceKey := strings.TrimSpace(parseEvent.ResourceKey)
		parseScopedRecord := BuildScopedResourceKey(parseScopeKey, parseResourceKey)
		if parseScopedRecord != "" {
			if parseReconciler.parseStorage != nil {
				if parseErr := parseReconciler.parseStorage.Delete(parseCtx, parseScopedRecord); parseErr != nil {
					return parseResult, parseErr
				}
			}
			if parseReconciler.parseSnapshot != nil {
				parseReconciler.parseSnapshot.DeleteSync(parseScopeKey, parseResourceKey)
			}
			parseResult.DeletedCount++
		}
	}

	parseQueueKey := strings.TrimSpace(parseInput.AckQueueKey)
	parseAckKey := strings.TrimSpace(parseInput.AckKey)
	parseOperationKey := strings.TrimSpace(parseInput.AckOperationKey)
	if parseQueueKey != "" && parseReconciler.parseStorage != nil {
		if _, parseErr := parseReconciler.parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []OutboxRecordEnvelope) ([]OutboxRecordEnvelope, error) {
			parseNextQueue := make([]OutboxRecordEnvelope, 0, len(parseQueue))
			for _, parseRecord := range parseQueue {
				isParseAckMatch := parseAckKey != "" && strings.TrimSpace(parseRecord.AckKey) == parseAckKey
				isParseOperationMatch := parseOperationKey != "" && strings.TrimSpace(parseRecord.OperationKey) == parseOperationKey
				if isParseAckMatch || isParseOperationMatch {
					parseResult.QueueRemovedCount++
					continue
				}
				parseNextQueue = append(parseNextQueue, parseRecord)
			}
			return parseNextQueue, nil
		}); parseErr != nil {
			return parseResult, parseErr
		}
	}

	parseRecord := NormalizeCacheRecordEnvelope(parseInput.Record)
	if parseRecord.ScopeKey == "" {
		parseRecord.ScopeKey = strings.TrimSpace(parseEvent.ScopeKey)
	}
	if parseRecord.ResourceKey == "" {
		parseRecord.ResourceKey = strings.TrimSpace(parseEvent.ResourceKey)
	}
	switch parseEvent.EventType {
	case WorkerEventStale:
		parseRecord.Status = CacheStatusStale
		parseReconciler.setRecord(parseCtx, parseRecord, &parseResult)
	case WorkerEventUpdated, WorkerEventRefreshing, WorkerEventAcked:
		parseReconciler.setRecord(parseCtx, parseRecord, &parseResult)
	}

	if parseResourceScopedKey != "" && isParseEventUpdatedAt {
		parseReconciler.parseLastResourceUpdate[parseResourceScopedKey] = parseEventUpdatedAt
	}
	parseReconciler.parseAppliedEventSet[parseEventKey] = struct{}{}
	parseResult.IsApplied = true
	return parseResult, nil
}

// setRecord writes one authoritative cache record into storage and snapshot mirrors.
func (parseReconciler *WorkerSnapshotReconciler) setRecord(parseCtx context.Context, parseRecord CacheRecordEnvelope, parseResult *WorkerSnapshotReconcileResult) {
	if parseResult == nil {
		return
	}
	if strings.TrimSpace(parseRecord.ScopeKey) == "" || strings.TrimSpace(parseRecord.ResourceKey) == "" {
		return
	}
	if parseReconciler.parseStorage != nil {
		if parseErr := parseReconciler.parseStorage.Set(parseCtx, parseRecord); parseErr != nil {
			return
		}
	}
	if parseReconciler.parseSnapshot != nil {
		parseReconciler.parseSnapshot.SetSync(parseRecord)
	}
	parseResult.IsRecordUpdated = true
}

// parseSplitScopedResourceKey splits one "<scope>::<resource>" key into parts.
func parseSplitScopedResourceKey(parseScopedRecord string) (string, string) {
	parseParts := strings.SplitN(strings.TrimSpace(parseScopedRecord), "::", 2)
	if len(parseParts) == 2 {
		return strings.TrimSpace(parseParts[0]), strings.TrimSpace(parseParts[1])
	}
	return "", ""
}

// buildWorkerEventFingerprint builds one deterministic fingerprint for dedupe.
func buildWorkerEventFingerprint(
	parseEvent WorkerEvent,
	parseAckQueueKey string,
	parseAckKey string,
	parseAckOperationKey string,
	parseInvalidatedScopedRecords []string,
) string {
	parseParts := []string{
		string(parseEvent.EventType),
		strings.TrimSpace(parseEvent.ScopeKey),
		strings.TrimSpace(parseEvent.ResourceKey),
		strings.TrimSpace(parseEvent.QueueKey),
		strings.TrimSpace(parseEvent.OperationKey),
		strings.TrimSpace(parseEvent.Reason),
		strings.TrimSpace(parseEvent.ErrorCode),
		strings.TrimSpace(parseEvent.UpdatedAt),
		strings.TrimSpace(parseAckQueueKey),
		strings.TrimSpace(parseAckKey),
		strings.TrimSpace(parseAckOperationKey),
	}
	for _, parseScopedRecord := range parseInvalidatedScopedRecords {
		parseParts = append(parseParts, strings.TrimSpace(parseScopedRecord))
	}
	return strings.Join(parseParts, "|")
}

// parseTimeRFC3339 parses one RFC3339 timestamp for deterministic event ordering.
func parseTimeRFC3339(parseTimestamp string) (time.Time, bool) {
	parseTimestamp = strings.TrimSpace(parseTimestamp)
	if parseTimestamp == "" {
		return time.Time{}, false
	}
	parseParsed, parseErr := time.Parse(time.RFC3339, parseTimestamp)
	if parseErr != nil {
		return time.Time{}, false
	}
	return parseParsed.UTC(), true
}
