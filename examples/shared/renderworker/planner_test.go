package renderworker

import "testing"

// TestBuildRenderWorkerAdaptiveChunkBoundsCoversAllIndexes validates weighted chunk bounds cover the full index range without gaps.
func TestBuildRenderWorkerAdaptiveChunkBoundsCoversAllIndexes(parseT *testing.T) {
	parseWeights := []int{1, 2, 3, 4, 5, 6}
	parseBounds := BuildRenderWorkerAdaptiveChunkBounds(parseWeights, 3)
	if len(parseBounds) != 3 {
		parseT.Fatalf("expected 3 chunk bounds, got %d", len(parseBounds))
	}
	parseExpectedStart := 0
	for _, parseBound := range parseBounds {
		if parseBound[0] != parseExpectedStart {
			parseT.Fatalf("expected contiguous start %d, got %d", parseExpectedStart, parseBound[0])
		}
		if parseBound[1] <= parseBound[0] {
			parseT.Fatalf("expected non-empty chunk bound, got [%d,%d)", parseBound[0], parseBound[1])
		}
		parseExpectedStart = parseBound[1]
	}
	if parseExpectedStart != len(parseWeights) {
		parseT.Fatalf("expected full range coverage to index %d, got %d", len(parseWeights), parseExpectedStart)
	}
}

// TestBuildRenderWorkerAdaptiveChunkBoundsClampsChunkCount validates chunk-count clamping for low and high inputs.
func TestBuildRenderWorkerAdaptiveChunkBoundsClampsChunkCount(parseT *testing.T) {
	parseWeights := []int{1, 1, 1}
	parseBoundsLow := BuildRenderWorkerAdaptiveChunkBounds(parseWeights, 0)
	if len(parseBoundsLow) != 1 {
		parseT.Fatalf("expected low chunk count to clamp to 1, got %d", len(parseBoundsLow))
	}
	parseBoundsHigh := BuildRenderWorkerAdaptiveChunkBounds(parseWeights, 99)
	if len(parseBoundsHigh) != len(parseWeights) {
		parseT.Fatalf("expected high chunk count to clamp to %d, got %d", len(parseWeights), len(parseBoundsHigh))
	}
}

// TestBuildRenderWorkerChunkPlansBuildsStableWeights validates chunk plan generation preserves chunk indexes and positive weights.
func TestBuildRenderWorkerChunkPlansBuildsStableWeights(parseT *testing.T) {
	parseBounds := [][2]int{{0, 2}, {2, 4}}
	parseWeights := []int{1, 5, 2, 0}
	parsePlans := BuildRenderWorkerChunkPlans(parseBounds, parseWeights)
	if len(parsePlans) != 2 {
		parseT.Fatalf("expected 2 chunk plans, got %d", len(parsePlans))
	}
	if parsePlans[0].GetChunkIndex != 0 || parsePlans[1].GetChunkIndex != 1 {
		parseT.Fatalf("expected stable chunk indexes [0,1], got [%d,%d]", parsePlans[0].GetChunkIndex, parsePlans[1].GetChunkIndex)
	}
	if parsePlans[0].GetWeight <= 0 || parsePlans[1].GetWeight <= 0 {
		parseT.Fatalf("expected positive weights, got [%d,%d]", parsePlans[0].GetWeight, parsePlans[1].GetWeight)
	}
}

// TestBuildRenderWorkerLanePlansGreedyBalance validates weighted greedy lane assignment balance and deterministic ordering.
func TestBuildRenderWorkerLanePlansGreedyBalance(parseT *testing.T) {
	parseChunkPlans := []RenderWorkerChunkPlan{
		{GetChunkIndex: 0, GetWeight: 9},
		{GetChunkIndex: 1, GetWeight: 8},
		{GetChunkIndex: 2, GetWeight: 3},
		{GetChunkIndex: 3, GetWeight: 2},
	}
	parseLanePlans := BuildRenderWorkerLanePlans(2, parseChunkPlans)
	if len(parseLanePlans) != 2 {
		parseT.Fatalf("expected 2 lane plans, got %d", len(parseLanePlans))
	}
	parseLaneWeightDelta := parseLanePlans[0].GetTotalWeight - parseLanePlans[1].GetTotalWeight
	if parseLaneWeightDelta < 0 {
		parseLaneWeightDelta = -parseLaneWeightDelta
	}
	if parseLaneWeightDelta > 2 {
		parseT.Fatalf("expected weighted balance delta <= 2, got %d (lane0=%d lane1=%d)", parseLaneWeightDelta, parseLanePlans[0].GetTotalWeight, parseLanePlans[1].GetTotalWeight)
	}
	for _, parseLanePlan := range parseLanePlans {
		for parseIndex := 1; parseIndex < len(parseLanePlan.GetChunkPlans); parseIndex++ {
			if parseLanePlan.GetChunkPlans[parseIndex-1].GetChunkIndex > parseLanePlan.GetChunkPlans[parseIndex].GetChunkIndex {
				parseT.Fatalf("expected lane chunk plans sorted by chunk index")
			}
		}
	}
}

// TestBuildRenderWorkerChunkLocalIndexesMapsGlobalToLocal validates global dirty indexes map to chunk-local offsets.
func TestBuildRenderWorkerChunkLocalIndexesMapsGlobalToLocal(parseT *testing.T) {
	parseLocal := BuildRenderWorkerChunkLocalIndexes([]int{0, 3, 7, 9}, 3, 8)
	if len(parseLocal) != 2 {
		parseT.Fatalf("expected 2 local indexes, got %d", len(parseLocal))
	}
	if parseLocal[0] != 0 || parseLocal[1] != 4 {
		parseT.Fatalf("expected local indexes [0,4], got [%d,%d]", parseLocal[0], parseLocal[1])
	}
}
