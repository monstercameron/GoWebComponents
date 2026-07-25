package cachepolicy

import (
	"context"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/client/cachecore"
)

// TestStoreUndeliveredLogRetryAndAck verifies undelivered diagnostics logs persist, retry on reconnect, and remove on ack.
func TestStoreUndeliveredLogRetryAndAck(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})
	parseOperationKey := "log-op-1"

	if parseErr := StoreUndeliveredLog(parseCtx, parseStorage, parseScopeKey, parseOperationKey, []byte(`{"level":"warn","message":"transport down"}`)); parseErr != nil {
		parseT.Fatalf("StoreUndeliveredLog: %v", parseErr)
	}
	parseQueueBeforeRetry, parseErr := GetUndeliveredLogQueue(parseCtx, parseStorage, parseScopeKey)
	if parseErr != nil {
		parseT.Fatalf("GetUndeliveredLogQueue(before): %v", parseErr)
	}
	if len(parseQueueBeforeRetry) != 1 {
		parseT.Fatalf("expected one queued log record before retry, got %+v", parseQueueBeforeRetry)
	}

	parseSendBatch, parseErr := GetUndeliveredLogRetryBatchOnReconnect(parseCtx, parseStorage, parseScopeKey, time.Now().UTC(), BuildUndeliveredLogOutboxPolicy())
	if parseErr != nil {
		parseT.Fatalf("GetUndeliveredLogRetryBatchOnReconnect: %v", parseErr)
	}
	if len(parseSendBatch) != 1 {
		parseT.Fatalf("expected one reconnect send batch row, got %+v", parseSendBatch)
	}
	if parseSendBatch[0].OperationKey != parseOperationKey || parseSendBatch[0].Status != cachecore.OutboxStatusSending || parseSendBatch[0].AttemptCount != 1 {
		parseT.Fatalf("unexpected send batch row: %+v", parseSendBatch[0])
	}

	parseRemovedCount, parseErr := ApplyUndeliveredLogAck(parseCtx, parseStorage, parseScopeKey, parseOperationKey)
	if parseErr != nil {
		parseT.Fatalf("ApplyUndeliveredLogAck: %v", parseErr)
	}
	if parseRemovedCount != 1 {
		parseT.Fatalf("expected one ack removal, got %d", parseRemovedCount)
	}
	parseQueueAfterAck, parseErr := GetUndeliveredLogQueue(parseCtx, parseStorage, parseScopeKey)
	if parseErr != nil {
		parseT.Fatalf("GetUndeliveredLogQueue(after): %v", parseErr)
	}
	if len(parseQueueAfterAck) != 0 {
		parseT.Fatalf("expected empty queue after ack, got %+v", parseQueueAfterAck)
	}
}

// TestGetUndeliveredLogRetryBatchOnReconnectDropsByCaps verifies max-age and max-attempt drop behavior.
func TestGetUndeliveredLogRetryBatchOnReconnectDropsByCaps(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})
	parseQueueKey := BuildUndeliveredLogQueueKey(parseScopeKey)
	parseNow := time.Now().UTC()

	parseOldRecord := cachecore.BuildOutboxRecordEnvelope(parseQueueKey, undeliveredLogResourceKey, "old-op", parseNow.Add(-48*time.Hour), parseNow, []byte(`{"level":"info"}`), cachecore.OutboxStatusQueued, "", "")
	parseCappedRecord := cachecore.BuildOutboxRecordEnvelope(parseQueueKey, undeliveredLogResourceKey, "capped-op", parseNow, parseNow, []byte(`{"level":"warn"}`), cachecore.OutboxStatusQueued, "", "")
	parseCappedRecord.AttemptCount = 3
	if _, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []cachecore.OutboxRecordEnvelope) ([]cachecore.OutboxRecordEnvelope, error) {
		return []cachecore.OutboxRecordEnvelope{parseOldRecord, parseCappedRecord}, nil
	}); parseErr != nil {
		parseT.Fatalf("UpdateQueueAtomic(seed): %v", parseErr)
	}

	parsePolicy := BuildUndeliveredLogOutboxPolicy()
	parsePolicy.MaxAge = 24 * time.Hour
	parsePolicy.MaxAttempts = 3
	parseSendBatch, parseErr := GetUndeliveredLogRetryBatchOnReconnect(parseCtx, parseStorage, parseScopeKey, parseNow, parsePolicy)
	if parseErr != nil {
		parseT.Fatalf("GetUndeliveredLogRetryBatchOnReconnect: %v", parseErr)
	}
	if len(parseSendBatch) != 0 {
		parseT.Fatalf("expected no send batch rows when all entries are capped, got %+v", parseSendBatch)
	}
	parseQueueAfterDrop, parseErr := GetUndeliveredLogQueue(parseCtx, parseStorage, parseScopeKey)
	if parseErr != nil {
		parseT.Fatalf("GetUndeliveredLogQueue(after): %v", parseErr)
	}
	if len(parseQueueAfterDrop) != 0 {
		parseT.Fatalf("expected capped records to be dropped, got %+v", parseQueueAfterDrop)
	}
}

// TestBuildUndeliveredLogQueueKey verifies scope isolation for undelivered-log queue keys.
func TestBuildUndeliveredLogQueueKey(parseT *testing.T) {
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})
	parseQueueKey := BuildUndeliveredLogQueueKey(parseScopeKey)
	parseExpected := parseScopeKey + "::" + undeliveredLogQueueResourceKey
	if parseQueueKey != parseExpected {
		parseT.Fatalf("unexpected queue key: %q != %q", parseQueueKey, parseExpected)
	}
}
