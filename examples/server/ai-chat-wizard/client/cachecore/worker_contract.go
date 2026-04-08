package cachecore

import (
	"context"
	"strings"
	"sync"
	"time"
)

// FlushOperation identifies one worker-eligible flush operation category.
type FlushOperation string

const (
	FlushOperationPersistLowPriority = "persist_low_priority"
	FlushOperationDrainQueuedLogs    = "drain_queued_logs"
	FlushOperationResendQueuedSends  = "resend_queued_sends"
)

// WorkerFlushTask stores one worker-friendly flush request envelope.
type WorkerFlushTask struct {
	TaskKey   string
	Operation FlushOperation
	ScopeKey  string
	QueueKey  string
	Payload   []byte
	CreatedAt string
}

// WorkerFlushContract defines a worker-friendly flush queue API.
type WorkerFlushContract interface {
	Enqueue(context.Context, WorkerFlushTask) error
	DequeueBatch(context.Context, int) ([]WorkerFlushTask, error)
	Ack(context.Context, []string) error
}

// WorkerFlushQueue provides one in-process implementation of WorkerFlushContract.
type WorkerFlushQueue struct {
	parseMu    sync.Mutex
	parseTasks []WorkerFlushTask
}

// BuildWorkerFlushTask builds one normalized worker flush task.
func BuildWorkerFlushTask(parseTaskKey string, parseOperation FlushOperation, parseScopeKey string, parseQueueKey string, parsePayload []byte) WorkerFlushTask {
	parseTask := WorkerFlushTask{
		TaskKey:   strings.TrimSpace(parseTaskKey),
		Operation: normalizeFlushOperation(parseOperation),
		ScopeKey:  strings.TrimSpace(parseScopeKey),
		QueueKey:  strings.TrimSpace(parseQueueKey),
		Payload:   append([]byte(nil), parsePayload...),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if parseTask.TaskKey == "" {
		parseTask.TaskKey = BuildScopedResourceKey(parseTask.ScopeKey, string(parseTask.Operation))
	}
	return parseTask
}

// BuildWorkerFlushQueue creates one worker-friendly in-memory flush queue.
func BuildWorkerFlushQueue() *WorkerFlushQueue {
	return &WorkerFlushQueue{
		parseTasks: []WorkerFlushTask{},
	}
}

// Enqueue appends one normalized flush task.
func (parseQueue *WorkerFlushQueue) Enqueue(parseCtx context.Context, parseTask WorkerFlushTask) error {
	_ = parseCtx
	if parseQueue == nil {
		return nil
	}
	parseTask = BuildWorkerFlushTask(parseTask.TaskKey, parseTask.Operation, parseTask.ScopeKey, parseTask.QueueKey, parseTask.Payload)
	parseQueue.parseMu.Lock()
	defer parseQueue.parseMu.Unlock()
	parseQueue.parseTasks = append(parseQueue.parseTasks, parseTask)
	return nil
}

// DequeueBatch returns up to one batch of pending flush tasks without removing them.
func (parseQueue *WorkerFlushQueue) DequeueBatch(parseCtx context.Context, parseMaxTasks int) ([]WorkerFlushTask, error) {
	_ = parseCtx
	if parseQueue == nil || parseMaxTasks <= 0 {
		return []WorkerFlushTask{}, nil
	}
	parseQueue.parseMu.Lock()
	defer parseQueue.parseMu.Unlock()
	if len(parseQueue.parseTasks) == 0 {
		return []WorkerFlushTask{}, nil
	}
	if parseMaxTasks > len(parseQueue.parseTasks) {
		parseMaxTasks = len(parseQueue.parseTasks)
	}
	parseTasks := make([]WorkerFlushTask, 0, parseMaxTasks)
	for _, parseTask := range parseQueue.parseTasks[:parseMaxTasks] {
		parseTasks = append(parseTasks, BuildWorkerFlushTask(parseTask.TaskKey, parseTask.Operation, parseTask.ScopeKey, parseTask.QueueKey, parseTask.Payload))
	}
	return parseTasks, nil
}

// Ack removes tasks that match one or more acknowledged task keys.
func (parseQueue *WorkerFlushQueue) Ack(parseCtx context.Context, parseTaskKeys []string) error {
	_ = parseCtx
	if parseQueue == nil || len(parseTaskKeys) == 0 {
		return nil
	}
	parseAckedSet := map[string]struct{}{}
	for _, parseTaskKey := range parseTaskKeys {
		parseTaskKey = strings.TrimSpace(parseTaskKey)
		if parseTaskKey == "" {
			continue
		}
		parseAckedSet[parseTaskKey] = struct{}{}
	}
	if len(parseAckedSet) == 0 {
		return nil
	}
	parseQueue.parseMu.Lock()
	defer parseQueue.parseMu.Unlock()
	parseNextTasks := make([]WorkerFlushTask, 0, len(parseQueue.parseTasks))
	for _, parseTask := range parseQueue.parseTasks {
		if _, isParseAcked := parseAckedSet[strings.TrimSpace(parseTask.TaskKey)]; isParseAcked {
			continue
		}
		parseNextTasks = append(parseNextTasks, parseTask)
	}
	parseQueue.parseTasks = parseNextTasks
	return nil
}

// normalizeFlushOperation resolves one canonical flush operation fallback.
func normalizeFlushOperation(parseOperation FlushOperation) FlushOperation {
	switch parseOperation {
	case FlushOperationDrainQueuedLogs, FlushOperationResendQueuedSends:
		return parseOperation
	default:
		return FlushOperationPersistLowPriority
	}
}
