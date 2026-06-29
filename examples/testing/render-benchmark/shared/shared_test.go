package shared

import "testing"

// TestBuildBenchmarkWorkerCoreChunkResultBuildsStablePreparedItems verifies core chunk preparation preserves item order and emits worker metadata.
func TestBuildBenchmarkWorkerCoreChunkResultBuildsStablePreparedItems(parseT *testing.T) {
	getResult := BuildBenchmarkWorkerCoreChunkResult("worker-a", BenchmarkWorkerCoreChunkRequest{
		GetChunkIndex: 2,
		GetItems: []BenchmarkCoreRowData{
			{GetID: 1, GetText: "Item 1"},
			{GetID: 2, GetText: "Item 2"},
		},
		GetWorkScale: 1,
	})
	if getResult.GetChunkIndex != 2 {
		parseT.Fatalf("expected chunk index 2, got %d", getResult.GetChunkIndex)
	}
	if getResult.GetWorker != "worker-a" {
		parseT.Fatalf("expected worker-a, got %q", getResult.GetWorker)
	}
	if len(getResult.GetItems) != 2 {
		parseT.Fatalf("expected 2 prepared items, got %d", len(getResult.GetItems))
	}
	if getResult.GetItems[0].GetID != 1 || getResult.GetItems[1].GetID != 2 {
		parseT.Fatalf("unexpected prepared item IDs: %+v", getResult.GetItems)
	}
	if getResult.GetItems[0].GetText != "Item 1" || getResult.GetItems[1].GetText != "Item 2" {
		parseT.Fatalf("unexpected prepared items: %+v", getResult.GetItems)
	}
	if getResult.GetItems[0].GetDigest == 0 || getResult.GetItems[1].GetDigest == 0 || getResult.GetWorkDigest == 0 {
		parseT.Fatalf("expected non-zero digests, got %+v", getResult)
	}
}

// TestBuildBenchmarkWorkerContentChunkResultCopiesContentFields verifies content chunk preparation preserves visible card fields.
func TestBuildBenchmarkWorkerContentChunkResultCopiesContentFields(parseT *testing.T) {
	getResult := BuildBenchmarkWorkerContentChunkResult("worker-b", BenchmarkWorkerContentChunkRequest{
		GetChunkIndex: 1,
		GetItems: []BenchmarkContentCardData{
			{
				GetID:      7,
				GetTitle:   "Article 7",
				GetSummary: "Summary",
				GetStatus:  "draft",
				GetMeta:    "Section 1",
				GetTags:    []string{"perf", "card-3"},
			},
		},
		GetWorkScale: 1,
	})
	if getResult.GetWorker != "worker-b" {
		parseT.Fatalf("expected worker-b, got %q", getResult.GetWorker)
	}
	if len(getResult.GetItems) != 1 {
		parseT.Fatalf("expected 1 prepared card, got %d", len(getResult.GetItems))
	}
	getItem := getResult.GetItems[0]
	if getItem.GetID != 7 || getItem.GetTitle != "Article 7" || getItem.GetStatus != "draft" {
		parseT.Fatalf("unexpected prepared content card: %+v", getItem)
	}
	if len(getItem.GetTags) != 2 || getItem.GetTags[0] != "perf" {
		parseT.Fatalf("unexpected tags: %+v", getItem.GetTags)
	}
	if getItem.GetDigest == 0 || getResult.GetWorkDigest == 0 {
		parseT.Fatalf("expected non-zero digests, got %+v", getResult)
	}
}
