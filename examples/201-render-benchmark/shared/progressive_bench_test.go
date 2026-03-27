package shared

import (
	"testing"
)

// buildBenchmarkProgressiveCoreChunkFixture builds one fixed-size core chunk result slice for progressive delivery benchmarks.
func buildBenchmarkProgressiveCoreChunkFixture(parseChunkCount int, parseItemsPerChunk int) []BenchmarkWorkerCoreChunkResult {
	getChunks := make([]BenchmarkWorkerCoreChunkResult, parseChunkCount)
	for parseChunkIndex := range getChunks {
		getItems := make([]BenchmarkPreparedCoreItem, parseItemsPerChunk)
		for parseItemIndex := range getItems {
			getItems[parseItemIndex] = BuildBenchmarkPreparedCoreItem(
				BenchmarkCoreRowData{GetID: parseChunkIndex*parseItemsPerChunk + parseItemIndex, GetText: "bench item text string"},
				parseItemIndex,
				1,
			)
		}
		getChunks[parseChunkIndex] = BenchmarkWorkerCoreChunkResult{
			GetChunkIndex: parseChunkIndex,
			GetItems:      getItems,
			GetWorker:     "bench-worker",
		}
	}
	return getChunks
}

// buildBenchmarkProgressiveContentChunkFixture builds one fixed-size content chunk result slice for progressive delivery benchmarks.
func buildBenchmarkProgressiveContentChunkFixture(parseChunkCount int, parseItemsPerChunk int) []BenchmarkWorkerContentChunkResult {
	getBaseTags := []string{"tag-a", "tag-b", "tag-c"}
	getChunks := make([]BenchmarkWorkerContentChunkResult, parseChunkCount)
	for parseChunkIndex := range getChunks {
		getItems := make([]BenchmarkPreparedContentCard, parseItemsPerChunk)
		for parseItemIndex := range getItems {
			getItems[parseItemIndex] = BuildBenchmarkPreparedContentCard(
				BenchmarkContentCardData{
					GetID:      parseChunkIndex*parseItemsPerChunk + parseItemIndex,
					GetTitle:   "Bench Article Title",
					GetSummary: "Summary text for the benchmark content card item.",
					GetStatus:  "published",
					GetMeta:    "Section A",
					GetTags:    getBaseTags,
				},
				parseItemIndex,
				1,
			)
		}
		getChunks[parseChunkIndex] = BenchmarkWorkerContentChunkResult{
			GetChunkIndex: parseChunkIndex,
			GetItems:      getItems,
			GetWorker:     "bench-worker",
		}
	}
	return getChunks
}

// BenchmarkProgressiveCoreChunkSnapshotCopyNilCallback measures the zero-callback (no progressive delivery) code path.
// This is the baseline: onPartialChunks == nil, so no copy is made. Any allocation here is a regression.
func BenchmarkProgressiveCoreChunkSnapshotCopyNilCallback(parseBench *testing.B) {
	getChunks := buildBenchmarkProgressiveCoreChunkFixture(8, 5)
	parseBench.ReportAllocs()
	parseBench.ResetTimer()
	for parseBenchIndex := 0; parseBenchIndex < parseBench.N; parseBenchIndex++ {
		// simulate the nil-callback fast path: no copy, no call
		var onPartialChunks func([]BenchmarkWorkerCoreChunkResult)
		if onPartialChunks != nil {
			getSnapshot := make([]BenchmarkWorkerCoreChunkResult, len(getChunks))
			copy(getSnapshot, getChunks)
			onPartialChunks(getSnapshot)
		}
	}
}

// BenchmarkProgressiveCoreChunkSnapshotCopy4Chunks measures per-lane snapshot copy cost for the 4-chunk (1-worker) case.
// Represents the single-worker runtime2 mode where each lane holds all chunks.
func BenchmarkProgressiveCoreChunkSnapshotCopy4Chunks(parseBench *testing.B) {
	getChunks := buildBenchmarkProgressiveCoreChunkFixture(4, 10)
	parseBench.ReportAllocs()
	parseBench.ResetTimer()
	for parseBenchIndex := 0; parseBenchIndex < parseBench.N; parseBenchIndex++ {
		// simulate per-lane snapshot: one copy of the results slice after a lane completes
		getSnapshot := make([]BenchmarkWorkerCoreChunkResult, len(getChunks))
		copy(getSnapshot, getChunks)
		_ = getSnapshot
	}
}

// BenchmarkProgressiveCoreChunkSnapshotCopy8Chunks measures per-lane snapshot copy cost for the 8-chunk (4-worker) case.
// Represents the runtime2-workers4 mode with adaptive doubling.
func BenchmarkProgressiveCoreChunkSnapshotCopy8Chunks(parseBench *testing.B) {
	getChunks := buildBenchmarkProgressiveCoreChunkFixture(8, 5)
	parseBench.ReportAllocs()
	parseBench.ResetTimer()
	for parseBenchIndex := 0; parseBenchIndex < parseBench.N; parseBenchIndex++ {
		getSnapshot := make([]BenchmarkWorkerCoreChunkResult, len(getChunks))
		copy(getSnapshot, getChunks)
		_ = getSnapshot
	}
}

// BenchmarkProgressiveCoreChunkSnapshotCopy16Chunks measures per-lane snapshot copy cost for the 16-chunk upper bound.
// Represents the maximum chunk count at workerCount*4 with 4 workers under load.
func BenchmarkProgressiveCoreChunkSnapshotCopy16Chunks(parseBench *testing.B) {
	getChunks := buildBenchmarkProgressiveCoreChunkFixture(16, 3)
	parseBench.ReportAllocs()
	parseBench.ResetTimer()
	for parseBenchIndex := 0; parseBenchIndex < parseBench.N; parseBenchIndex++ {
		getSnapshot := make([]BenchmarkWorkerCoreChunkResult, len(getChunks))
		copy(getSnapshot, getChunks)
		_ = getSnapshot
	}
}

// BenchmarkProgressiveContentChunkSnapshotCopy8Chunks measures per-lane snapshot copy cost for the 8-chunk content case.
func BenchmarkProgressiveContentChunkSnapshotCopy8Chunks(parseBench *testing.B) {
	getChunks := buildBenchmarkProgressiveContentChunkFixture(8, 2)
	parseBench.ReportAllocs()
	parseBench.ResetTimer()
	for parseBenchIndex := 0; parseBenchIndex < parseBench.N; parseBenchIndex++ {
		getSnapshot := make([]BenchmarkWorkerContentChunkResult, len(getChunks))
		copy(getSnapshot, getChunks)
		_ = getSnapshot
	}
}

// BenchmarkBuildBenchmarkPreparedCoreItemScale1 measures single-item preparation cost at scale 1.
// This is the inner-loop cost that every worker chunk bears; regressions here affect all modes.
func BenchmarkBuildBenchmarkPreparedCoreItemScale1(parseBench *testing.B) {
	getItem := BenchmarkCoreRowData{GetID: 42, GetText: "A reasonably sized benchmark row item text"}
	parseBench.ReportAllocs()
	parseBench.ResetTimer()
	for parseBenchIndex := 0; parseBenchIndex < parseBench.N; parseBenchIndex++ {
		_ = BuildBenchmarkPreparedCoreItem(getItem, 0, 1)
	}
}

// BenchmarkBuildBenchmarkPreparedCoreItemScale4 measures single-item preparation cost at scale 4 (work amplifier).
func BenchmarkBuildBenchmarkPreparedCoreItemScale4(parseBench *testing.B) {
	getItem := BenchmarkCoreRowData{GetID: 42, GetText: "A reasonably sized benchmark row item text"}
	parseBench.ReportAllocs()
	parseBench.ResetTimer()
	for parseBenchIndex := 0; parseBenchIndex < parseBench.N; parseBenchIndex++ {
		_ = BuildBenchmarkPreparedCoreItem(getItem, 0, 4)
	}
}

// BenchmarkBuildBenchmarkPreparedContentCardScale1 measures single content-card preparation cost at scale 1.
func BenchmarkBuildBenchmarkPreparedContentCardScale1(parseBench *testing.B) {
	getItem := BenchmarkContentCardData{
		GetID:      99,
		GetTitle:   "Bench Article Title For Content Card",
		GetSummary: "A detailed summary for the content card benchmark item with sufficient text.",
		GetStatus:  "published",
		GetMeta:    "Section B",
		GetTags:    []string{"performance", "benchmark", "runtime2"},
	}
	parseBench.ReportAllocs()
	parseBench.ResetTimer()
	for parseBenchIndex := 0; parseBenchIndex < parseBench.N; parseBenchIndex++ {
		_ = BuildBenchmarkPreparedContentCard(getItem, 0, 1)
	}
}

// BenchmarkBuildBenchmarkWorkerCoreChunkResult40Items measures full-chunk preparation for the default 40-item core list.
// This represents one lane's work unit in the single-worker runtime2 mode.
func BenchmarkBuildBenchmarkWorkerCoreChunkResult40Items(parseBench *testing.B) {
	getItems := make([]BenchmarkCoreRowData, 40)
	for parseItemIndex := range getItems {
		getItems[parseItemIndex] = BenchmarkCoreRowData{GetID: parseItemIndex + 1, GetText: "Row item label text for benchmark"}
	}
	getRequest := BenchmarkWorkerCoreChunkRequest{
		GetChunkIndex: 0,
		GetItems:      getItems,
		GetWorkScale:  1,
	}
	parseBench.ReportAllocs()
	parseBench.ResetTimer()
	for parseBenchIndex := 0; parseBenchIndex < parseBench.N; parseBenchIndex++ {
		_ = BuildBenchmarkWorkerCoreChunkResult("bench-worker", getRequest)
	}
}

// BenchmarkBuildBenchmarkWorkerCoreChunkResult240Items measures full-chunk preparation for the 240-item stress list.
// This is the maximum core-list size and the highest-cost lane work unit.
func BenchmarkBuildBenchmarkWorkerCoreChunkResult240Items(parseBench *testing.B) {
	getItems := make([]BenchmarkCoreRowData, 240)
	for parseItemIndex := range getItems {
		getItems[parseItemIndex] = BenchmarkCoreRowData{GetID: parseItemIndex + 1, GetText: "Row item label text for benchmark"}
	}
	getRequest := BenchmarkWorkerCoreChunkRequest{
		GetChunkIndex: 0,
		GetItems:      getItems,
		GetWorkScale:  1,
	}
	parseBench.ReportAllocs()
	parseBench.ResetTimer()
	for parseBenchIndex := 0; parseBenchIndex < parseBench.N; parseBenchIndex++ {
		_ = BuildBenchmarkWorkerCoreChunkResult("bench-worker", getRequest)
	}
}
