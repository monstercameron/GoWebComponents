package cachecore

import (
	"strings"
	"sync"
	"time"
)

// WorkerDeltaType identifies one worker-maintenance delta class.
type WorkerDeltaType string

const (
	WorkerDeltaInvalidationScan WorkerDeltaType = "invalidation_scan"
	WorkerDeltaQueueFlush       WorkerDeltaType = "queue_flush"
	WorkerDeltaBackgroundWrite  WorkerDeltaType = "background_write"
)

// WorkerDelta stores one payload-agnostic maintenance delta.
type WorkerDelta struct {
	DeltaType    WorkerDeltaType
	ScopeKey     string
	ResourceKey  string
	QueueKey     string
	OperationKey string
	Reason       string
	UpdatedAt    string
}

// WorkerMessageBatch stores one grouped worker-maintenance delta batch.
type WorkerMessageBatch struct {
	BatchType WorkerDeltaType
	ScopeKey  string
	QueueKey  string
	BatchedAt string
	Deltas    []WorkerDelta
}

// WorkerBatchRule defines one deterministic worker communication batching policy.
type WorkerBatchRule struct {
	MaxDeltasPerBatch int
}

// WorkerBatcher batches worker-maintenance deltas before cross-thread transfer.
type WorkerBatcher struct {
	parseMu      sync.Mutex
	parseRule    WorkerBatchRule
	parsePending []WorkerDelta
}

// BuildWorkerBatcher creates one worker batching controller.
func BuildWorkerBatcher(parseRule WorkerBatchRule) *WorkerBatcher {
	parseRule = normalizeWorkerBatchRule(parseRule)
	return &WorkerBatcher{
		parseRule:    parseRule,
		parsePending: []WorkerDelta{},
	}
}

// ApplyDelta appends one normalized maintenance delta into pending batching state.
func (parseBatcher *WorkerBatcher) ApplyDelta(parseDelta WorkerDelta) {
	if parseBatcher == nil {
		return
	}
	parseDelta = normalizeWorkerDelta(parseDelta)
	parseBatcher.parseMu.Lock()
	defer parseBatcher.parseMu.Unlock()
	parseBatcher.parsePending = append(parseBatcher.parsePending, parseDelta)
}

// GetBatches returns grouped/chunked worker deltas and clears pending state.
func (parseBatcher *WorkerBatcher) GetBatches() []WorkerMessageBatch {
	if parseBatcher == nil {
		return []WorkerMessageBatch{}
	}
	parseBatcher.parseMu.Lock()
	defer parseBatcher.parseMu.Unlock()
	if len(parseBatcher.parsePending) == 0 {
		return []WorkerMessageBatch{}
	}
	parseRule := normalizeWorkerBatchRule(parseBatcher.parseRule)
	parseGrouped := map[string][]WorkerDelta{}
	parseGroupOrder := make([]string, 0, len(parseBatcher.parsePending))
	for _, parseDelta := range parseBatcher.parsePending {
		parseGroupKey := buildWorkerDeltaGroupKey(parseDelta)
		if _, isParseFound := parseGrouped[parseGroupKey]; !isParseFound {
			parseGroupOrder = append(parseGroupOrder, parseGroupKey)
		}
		parseGrouped[parseGroupKey] = append(parseGrouped[parseGroupKey], parseDelta)
	}
	parseBatches := make([]WorkerMessageBatch, 0, len(parseGroupOrder))
	for _, parseGroupKey := range parseGroupOrder {
		parseGroupDeltas := parseGrouped[parseGroupKey]
		if len(parseGroupDeltas) == 0 {
			continue
		}
		parseBatchType := parseGroupDeltas[0].DeltaType
		parseScopeKey := parseGroupDeltas[0].ScopeKey
		parseQueueKey := parseGroupDeltas[0].QueueKey
		for parseStart := 0; parseStart < len(parseGroupDeltas); parseStart += parseRule.MaxDeltasPerBatch {
			parseEnd := parseStart + parseRule.MaxDeltasPerBatch
			if parseEnd > len(parseGroupDeltas) {
				parseEnd = len(parseGroupDeltas)
			}
			parseChunk := append([]WorkerDelta(nil), parseGroupDeltas[parseStart:parseEnd]...)
			parseBatches = append(parseBatches, WorkerMessageBatch{
				BatchType: parseBatchType,
				ScopeKey:  parseScopeKey,
				QueueKey:  parseQueueKey,
				BatchedAt: time.Now().UTC().Format(time.RFC3339),
				Deltas:    parseChunk,
			})
		}
	}
	parseBatcher.parsePending = []WorkerDelta{}
	return parseBatches
}

// normalizeWorkerBatchRule resolves one stable batching rule fallback.
func normalizeWorkerBatchRule(parseRule WorkerBatchRule) WorkerBatchRule {
	if parseRule.MaxDeltasPerBatch <= 0 {
		parseRule.MaxDeltasPerBatch = 32
	}
	return parseRule
}

// normalizeWorkerDelta normalizes one worker delta contract shape.
func normalizeWorkerDelta(parseDelta WorkerDelta) WorkerDelta {
	parseDelta.DeltaType = normalizeWorkerDeltaType(parseDelta.DeltaType)
	parseDelta.ScopeKey = strings.TrimSpace(parseDelta.ScopeKey)
	parseDelta.ResourceKey = strings.TrimSpace(parseDelta.ResourceKey)
	parseDelta.QueueKey = strings.TrimSpace(parseDelta.QueueKey)
	parseDelta.OperationKey = strings.TrimSpace(parseDelta.OperationKey)
	parseDelta.Reason = strings.TrimSpace(parseDelta.Reason)
	parseDelta.UpdatedAt = strings.TrimSpace(parseDelta.UpdatedAt)
	if parseDelta.UpdatedAt == "" {
		parseDelta.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return parseDelta
}

// normalizeWorkerDeltaType resolves one canonical worker delta type fallback.
func normalizeWorkerDeltaType(parseDeltaType WorkerDeltaType) WorkerDeltaType {
	switch parseDeltaType {
	case WorkerDeltaInvalidationScan, WorkerDeltaQueueFlush:
		return parseDeltaType
	default:
		return WorkerDeltaBackgroundWrite
	}
}

// buildWorkerDeltaGroupKey builds one deterministic grouping key for batched worker deltas.
func buildWorkerDeltaGroupKey(parseDelta WorkerDelta) string {
	return strings.Join([]string{
		string(parseDelta.DeltaType),
		parseDelta.ScopeKey,
		parseDelta.QueueKey,
	}, "|")
}
