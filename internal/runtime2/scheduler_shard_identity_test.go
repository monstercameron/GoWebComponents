package runtime2

import (
	"fmt"
	"testing"
)

var storeSchedulerShardIDBenchmarkSink SchedulerShardID

// TestGetSchedulerShardIDKeepsStableWorkerShardPerLiveWorker verifies one live worker resolves to one stable shard ID.
func TestGetSchedulerShardIDKeepsStableWorkerShardPerLiveWorker(getTesting *testing.T) {
	getSchedulerShardModel := BuildSchedulerShardModel()
	getWorkerShardFirst, getWorkerShardFirstErr := getSchedulerShardModel.GetSchedulerShardID("worker-a")
	if getWorkerShardFirstErr != nil {
		getTesting.Fatalf("GetSchedulerShardID(worker-a) returned error: %v", getWorkerShardFirstErr)
	}
	getWorkerShardSecond, getWorkerShardSecondErr := getSchedulerShardModel.GetSchedulerShardID("worker-a")
	if getWorkerShardSecondErr != nil {
		getTesting.Fatalf("GetSchedulerShardID(worker-a) second call returned error: %v", getWorkerShardSecondErr)
	}
	if getWorkerShardFirst != getWorkerShardSecond {
		getTesting.Fatalf("expected stable shard identity for live worker, first=%q second=%q", getWorkerShardFirst, getWorkerShardSecond)
	}
}

// TestGetSchedulerShardIDSeparatesDifferentLiveWorkers verifies distinct live workers receive distinct shard IDs.
func TestGetSchedulerShardIDSeparatesDifferentLiveWorkers(getTesting *testing.T) {
	getSchedulerShardModel := BuildSchedulerShardModel()
	getWorkerShardA, getWorkerShardAErr := getSchedulerShardModel.GetSchedulerShardID("worker-a")
	if getWorkerShardAErr != nil {
		getTesting.Fatalf("GetSchedulerShardID(worker-a) returned error: %v", getWorkerShardAErr)
	}
	getWorkerShardB, getWorkerShardBErr := getSchedulerShardModel.GetSchedulerShardID("worker-b")
	if getWorkerShardBErr != nil {
		getTesting.Fatalf("GetSchedulerShardID(worker-b) returned error: %v", getWorkerShardBErr)
	}
	if getWorkerShardA == getWorkerShardB {
		getTesting.Fatalf("expected different workers to map to different shard IDs, got %q", getWorkerShardA)
	}
}

// TestClearSchedulerShardIDAvoidsUnsafeReuseAfterDispose verifies disposed shard IDs are not immediately reused.
func TestClearSchedulerShardIDAvoidsUnsafeReuseAfterDispose(getTesting *testing.T) {
	getSchedulerShardModel := BuildSchedulerShardModel()
	getWorkerShardBeforeDispose, getWorkerShardBeforeDisposeErr := getSchedulerShardModel.GetSchedulerShardID("worker-a")
	if getWorkerShardBeforeDisposeErr != nil {
		getTesting.Fatalf("GetSchedulerShardID(worker-a) before dispose returned error: %v", getWorkerShardBeforeDisposeErr)
	}
	getDisposedShardID, hasDisposedShardID := getSchedulerShardModel.ClearSchedulerShardID("worker-a")
	if !hasDisposedShardID {
		getTesting.Fatalf("ClearSchedulerShardID(worker-a) did not report a disposed shard")
	}
	if getDisposedShardID != getWorkerShardBeforeDispose {
		getTesting.Fatalf("expected cleared shard ID %q to match prior assignment %q", getDisposedShardID, getWorkerShardBeforeDispose)
	}
	getWorkerShardAfterDispose, getWorkerShardAfterDisposeErr := getSchedulerShardModel.GetSchedulerShardID("worker-a")
	if getWorkerShardAfterDisposeErr != nil {
		getTesting.Fatalf("GetSchedulerShardID(worker-a) after dispose returned error: %v", getWorkerShardAfterDisposeErr)
	}
	if getWorkerShardBeforeDispose == getWorkerShardAfterDispose {
		getTesting.Fatalf("expected disposed shard ID to not be reused unsafely, before=%q after=%q", getWorkerShardBeforeDispose, getWorkerShardAfterDispose)
	}
}

// BenchmarkFormatSchedulerShardIDCurrentVsLegacy compares the current shard-ID formatter against the previous fmt.Sprintf path.
func BenchmarkFormatSchedulerShardIDCurrentVsLegacy(parseBenchmark *testing.B) {
	parseSequences := []uint64{1, 42, 1234567}
	parseBenchmark.Run("legacy", func(parseLegacyBenchmark *testing.B) {
		parseLegacyBenchmark.ReportAllocs()
		parseLegacyBenchmark.ResetTimer()
		for parseIndex := 0; parseLegacyBenchmark.Loop(); parseIndex++ {
			storeSchedulerShardIDBenchmarkSink = formatSchedulerShardIDLegacy(parseSequences[parseIndex%len(parseSequences)])
		}
	})
	parseBenchmark.Run("current", func(parseCurrentBenchmark *testing.B) {
		parseCurrentBenchmark.ReportAllocs()
		parseCurrentBenchmark.ResetTimer()
		for parseIndex := 0; parseCurrentBenchmark.Loop(); parseIndex++ {
			storeSchedulerShardIDBenchmarkSink = formatSchedulerShardID(parseSequences[parseIndex%len(parseSequences)])
		}
	})
}

// formatSchedulerShardIDLegacy preserves the previous fmt.Sprintf shard-ID formatting path for benchmark comparison.
func formatSchedulerShardIDLegacy(parseSequence uint64) SchedulerShardID {
	return SchedulerShardID(fmt.Sprintf("shard-%d", parseSequence))
}
