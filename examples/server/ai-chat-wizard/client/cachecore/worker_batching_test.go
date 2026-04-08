package cachecore

import "testing"

// TestGetWorkerBatchesGroupsByDeltaClass verifies grouped snapshot/delta batching behavior.
func TestGetWorkerBatchesGroupsByDeltaClass(parseT *testing.T) {
	parseBatcher := BuildWorkerBatcher(WorkerBatchRule{MaxDeltasPerBatch: 10})
	parseScopeKey := BuildScopeKey(ScopeKey{AppVersion: "v1", SessionID: "s1", UserID: "u1", Workspace: "w1"})

	parseBatcher.ApplyDelta(WorkerDelta{DeltaType: WorkerDeltaInvalidationScan, ScopeKey: parseScopeKey, ResourceKey: "locale.catalog"})
	parseBatcher.ApplyDelta(WorkerDelta{DeltaType: WorkerDeltaInvalidationScan, ScopeKey: parseScopeKey, ResourceKey: "sidebar.page"})
	parseBatcher.ApplyDelta(WorkerDelta{DeltaType: WorkerDeltaQueueFlush, ScopeKey: parseScopeKey, QueueKey: "queue.logs", OperationKey: "op-1"})
	parseBatcher.ApplyDelta(WorkerDelta{DeltaType: WorkerDeltaBackgroundWrite, ScopeKey: parseScopeKey, ResourceKey: "thread.cache"})

	parseBatches := parseBatcher.GetBatches()
	if len(parseBatches) != 3 {
		parseT.Fatalf("expected 3 grouped batches, got %d", len(parseBatches))
	}
	if parseBatches[0].BatchType != WorkerDeltaInvalidationScan || len(parseBatches[0].Deltas) != 2 {
		parseT.Fatalf("expected invalidation batch with 2 deltas, got %+v", parseBatches[0])
	}
	if parseBatches[1].BatchType != WorkerDeltaQueueFlush || len(parseBatches[1].Deltas) != 1 {
		parseT.Fatalf("expected queue-flush batch, got %+v", parseBatches[1])
	}
	if parseBatches[2].BatchType != WorkerDeltaBackgroundWrite || len(parseBatches[2].Deltas) != 1 {
		parseT.Fatalf("expected background-write batch, got %+v", parseBatches[2])
	}
	if len(parseBatcher.GetBatches()) != 0 {
		parseT.Fatalf("expected pending state to clear after GetBatches")
	}
}

// TestGetWorkerBatchesRespectsBatchSize verifies per-group chunking behavior under one max batch size.
func TestGetWorkerBatchesRespectsBatchSize(parseT *testing.T) {
	parseBatcher := BuildWorkerBatcher(WorkerBatchRule{MaxDeltasPerBatch: 2})
	parseScopeKey := BuildScopeKey(ScopeKey{AppVersion: "v1", SessionID: "s1", UserID: "u1", Workspace: "w1"})

	parseBatcher.ApplyDelta(WorkerDelta{DeltaType: WorkerDeltaInvalidationScan, ScopeKey: parseScopeKey, ResourceKey: "r1"})
	parseBatcher.ApplyDelta(WorkerDelta{DeltaType: WorkerDeltaInvalidationScan, ScopeKey: parseScopeKey, ResourceKey: "r2"})
	parseBatcher.ApplyDelta(WorkerDelta{DeltaType: WorkerDeltaInvalidationScan, ScopeKey: parseScopeKey, ResourceKey: "r3"})
	parseBatcher.ApplyDelta(WorkerDelta{DeltaType: WorkerDeltaInvalidationScan, ScopeKey: parseScopeKey, ResourceKey: "r4"})
	parseBatcher.ApplyDelta(WorkerDelta{DeltaType: WorkerDeltaInvalidationScan, ScopeKey: parseScopeKey, ResourceKey: "r5"})

	parseBatches := parseBatcher.GetBatches()
	if len(parseBatches) != 3 {
		parseT.Fatalf("expected 3 batches, got %d", len(parseBatches))
	}
	if len(parseBatches[0].Deltas) != 2 || len(parseBatches[1].Deltas) != 2 || len(parseBatches[2].Deltas) != 1 {
		parseT.Fatalf("expected chunk sizes 2/2/1, got %+v", parseBatches)
	}
}

// TestNormalizeWorkerDeltaDefaults verifies type and timestamp fallback normalization.
func TestNormalizeWorkerDeltaDefaults(parseT *testing.T) {
	parseDelta := normalizeWorkerDelta(WorkerDelta{
		DeltaType:   WorkerDeltaType("unknown"),
		ScopeKey:    " scope-a ",
		ResourceKey: " resource-a ",
	})
	if parseDelta.DeltaType != WorkerDeltaBackgroundWrite {
		parseT.Fatalf("expected fallback delta type %q, got %q", WorkerDeltaBackgroundWrite, parseDelta.DeltaType)
	}
	if parseDelta.ScopeKey != "scope-a" || parseDelta.ResourceKey != "resource-a" {
		parseT.Fatalf("expected trimmed fields, got %+v", parseDelta)
	}
	if parseDelta.UpdatedAt == "" {
		parseT.Fatalf("expected UpdatedAt timestamp fallback")
	}
}
