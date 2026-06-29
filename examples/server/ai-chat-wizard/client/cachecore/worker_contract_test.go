package cachecore

import (
	"context"
	"testing"
)

// TestWorkerFlushQueueEnqueueDequeueAck verifies queue lifecycle behavior for worker flush tasks.
func TestWorkerFlushQueueEnqueueDequeueAck(parseT *testing.T) {
	parseCtx := context.Background()
	parseQueue := BuildWorkerFlushQueue()
	parseTaskOne := BuildWorkerFlushTask("task-1", FlushOperationPersistLowPriority, "scope-1", "queue-1", []byte(`{"persist":1}`))
	parseTaskTwo := BuildWorkerFlushTask("task-2", FlushOperationDrainQueuedLogs, "scope-1", "queue-logs", []byte(`{"logs":1}`))
	if parseErr := parseQueue.Enqueue(parseCtx, parseTaskOne); parseErr != nil {
		parseT.Fatalf("Enqueue(task-1): %v", parseErr)
	}
	if parseErr := parseQueue.Enqueue(parseCtx, parseTaskTwo); parseErr != nil {
		parseT.Fatalf("Enqueue(task-2): %v", parseErr)
	}
	parseBatch, parseErr := parseQueue.DequeueBatch(parseCtx, 2)
	if parseErr != nil {
		parseT.Fatalf("DequeueBatch: %v", parseErr)
	}
	if len(parseBatch) != 2 {
		parseT.Fatalf("expected 2 tasks in batch, got %d", len(parseBatch))
	}
	if parseBatch[0].TaskKey != "task-1" || parseBatch[1].TaskKey != "task-2" {
		parseT.Fatalf("unexpected task order/content: %+v", parseBatch)
	}
	if parseErr = parseQueue.Ack(parseCtx, []string{"task-1"}); parseErr != nil {
		parseT.Fatalf("Ack(task-1): %v", parseErr)
	}
	parseRemainingBatch, parseErr := parseQueue.DequeueBatch(parseCtx, 10)
	if parseErr != nil {
		parseT.Fatalf("DequeueBatch(remaining): %v", parseErr)
	}
	if len(parseRemainingBatch) != 1 || parseRemainingBatch[0].TaskKey != "task-2" {
		parseT.Fatalf("expected only task-2 remaining, got %+v", parseRemainingBatch)
	}
}

// TestBuildWorkerFlushTaskDefaults verifies default operation and generated task-key behavior.
func TestBuildWorkerFlushTaskDefaults(parseT *testing.T) {
	parseTask := BuildWorkerFlushTask("", FlushOperation("unknown-op"), "scope-a", "queue-a", []byte(`{"x":1}`))
	if parseTask.Operation != FlushOperationPersistLowPriority {
		parseT.Fatalf("expected fallback operation, got %q", parseTask.Operation)
	}
	if parseTask.TaskKey == "" || parseTask.CreatedAt == "" {
		parseT.Fatalf("expected generated task metadata, got %+v", parseTask)
	}
}
