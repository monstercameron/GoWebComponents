package cachepolicy

import (
	"context"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/client/cachecore"
)

// TestStoreUnsentMessageScopesByThread verifies unsent-message queues stay isolated by canonical thread context.
func TestStoreUnsentMessageScopesByThread(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})

	if parseErr := StoreUnsentMessage(parseCtx, parseStorage, parseScopeKey, "thread-a", "op-1", []byte(`{"text":"a"}`)); parseErr != nil {
		parseT.Fatalf("StoreUnsentMessage(thread-a): %v", parseErr)
	}
	if parseErr := StoreUnsentMessage(parseCtx, parseStorage, parseScopeKey, "thread-b", "op-1", []byte(`{"text":"b"}`)); parseErr != nil {
		parseT.Fatalf("StoreUnsentMessage(thread-b): %v", parseErr)
	}
	parseThreadAQueue, parseErr := GetUnsentMessageQueue(parseCtx, parseStorage, parseScopeKey, "thread-a")
	if parseErr != nil {
		parseT.Fatalf("GetUnsentMessageQueue(thread-a): %v", parseErr)
	}
	parseThreadBQueue, parseErr := GetUnsentMessageQueue(parseCtx, parseStorage, parseScopeKey, "thread-b")
	if parseErr != nil {
		parseT.Fatalf("GetUnsentMessageQueue(thread-b): %v", parseErr)
	}
	if len(parseThreadAQueue) != 1 || len(parseThreadBQueue) != 1 {
		parseT.Fatalf("expected isolated thread queues, got thread-a=%d thread-b=%d", len(parseThreadAQueue), len(parseThreadBQueue))
	}
}

// TestStoreUnsentMessageDedupesByOperationKey verifies duplicate local enqueue attempts do not duplicate queue rows.
func TestStoreUnsentMessageDedupesByOperationKey(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})

	if parseErr := StoreUnsentMessage(parseCtx, parseStorage, parseScopeKey, "", "op-1", []byte(`{"text":"draft"}`)); parseErr != nil {
		parseT.Fatalf("StoreUnsentMessage(first): %v", parseErr)
	}
	if parseErr := StoreUnsentMessage(parseCtx, parseStorage, parseScopeKey, "", "op-1", []byte(`{"text":"draft duplicate"}`)); parseErr != nil {
		parseT.Fatalf("StoreUnsentMessage(duplicate): %v", parseErr)
	}
	parseQueue, parseErr := GetUnsentMessageQueue(parseCtx, parseStorage, parseScopeKey, "")
	if parseErr != nil {
		parseT.Fatalf("GetUnsentMessageQueue: %v", parseErr)
	}
	if len(parseQueue) != 1 {
		parseT.Fatalf("expected deduped queue with one row, got %+v", parseQueue)
	}
}

// TestApplyUnsentMessageRouteNormalizationDedupes verifies route canonicalization moves queued sends without duplicating operation keys.
func TestApplyUnsentMessageRouteNormalizationDedupes(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})

	if parseErr := StoreUnsentMessage(parseCtx, parseStorage, parseScopeKey, "", "op-1", []byte(`{"text":"from new chat"}`)); parseErr != nil {
		parseT.Fatalf("StoreUnsentMessage(new-chat op-1): %v", parseErr)
	}
	if parseErr := StoreUnsentMessage(parseCtx, parseStorage, parseScopeKey, "", "op-2", []byte(`{"text":"from new chat op-2"}`)); parseErr != nil {
		parseT.Fatalf("StoreUnsentMessage(new-chat op-2): %v", parseErr)
	}
	if parseErr := StoreUnsentMessage(parseCtx, parseStorage, parseScopeKey, "thread-123", "op-2", []byte(`{"text":"already in thread"}`)); parseErr != nil {
		parseT.Fatalf("StoreUnsentMessage(thread op-2): %v", parseErr)
	}

	parseMovedCount, parseErr := ApplyUnsentMessageRouteNormalization(parseCtx, parseStorage, parseScopeKey, "", "thread-123")
	if parseErr != nil {
		parseT.Fatalf("ApplyUnsentMessageRouteNormalization: %v", parseErr)
	}
	if parseMovedCount != 1 {
		parseT.Fatalf("expected one moved row after dedupe, got %d", parseMovedCount)
	}
	parseSourceQueue, parseErr := GetUnsentMessageQueue(parseCtx, parseStorage, parseScopeKey, "")
	if parseErr != nil {
		parseT.Fatalf("GetUnsentMessageQueue(new-chat): %v", parseErr)
	}
	if len(parseSourceQueue) != 0 {
		parseT.Fatalf("expected new-chat queue to be drained, got %+v", parseSourceQueue)
	}
	parseTargetQueue, parseErr := GetUnsentMessageQueue(parseCtx, parseStorage, parseScopeKey, "thread-123")
	if parseErr != nil {
		parseT.Fatalf("GetUnsentMessageQueue(thread-123): %v", parseErr)
	}
	if len(parseTargetQueue) != 2 {
		parseT.Fatalf("expected thread queue to contain two deduped ops, got %+v", parseTargetQueue)
	}
}

// TestUnsentMessageRetryAndAck verifies reconnect retry metadata updates and ack removal behavior.
func TestUnsentMessageRetryAndAck(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})
	parseThreadKey := "thread-456"

	if parseErr := StoreUnsentMessage(parseCtx, parseStorage, parseScopeKey, parseThreadKey, "op-1", []byte(`{"text":"hello"}`)); parseErr != nil {
		parseT.Fatalf("StoreUnsentMessage: %v", parseErr)
	}
	parseSendBatch, parseErr := GetUnsentMessageRetryBatchOnReconnect(parseCtx, parseStorage, parseScopeKey, parseThreadKey, time.Now().UTC(), BuildUnsentMessageOutboxPolicy())
	if parseErr != nil {
		parseT.Fatalf("GetUnsentMessageRetryBatchOnReconnect: %v", parseErr)
	}
	if len(parseSendBatch) != 1 || parseSendBatch[0].Status != cachecore.OutboxStatusSending || parseSendBatch[0].AttemptCount != 1 {
		parseT.Fatalf("unexpected send batch: %+v", parseSendBatch)
	}
	parseRemovedCount, parseErr := ApplyUnsentMessageAck(parseCtx, parseStorage, parseScopeKey, parseThreadKey, "op-1")
	if parseErr != nil {
		parseT.Fatalf("ApplyUnsentMessageAck: %v", parseErr)
	}
	if parseRemovedCount != 1 {
		parseT.Fatalf("expected one ack removal, got %d", parseRemovedCount)
	}
	parseQueueAfterAck, parseErr := GetUnsentMessageQueue(parseCtx, parseStorage, parseScopeKey, parseThreadKey)
	if parseErr != nil {
		parseT.Fatalf("GetUnsentMessageQueue(after): %v", parseErr)
	}
	if len(parseQueueAfterAck) != 0 {
		parseT.Fatalf("expected empty thread queue after ack, got %+v", parseQueueAfterAck)
	}
}
