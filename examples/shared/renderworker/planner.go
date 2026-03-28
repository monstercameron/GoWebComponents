package renderworker

import "sort"

// RenderWorkerChunkPlan stores one contiguous chunk and its estimated work weight.
type RenderWorkerChunkPlan struct {
	GetChunkIndex int
	GetStart      int
	GetEnd        int
	GetWeight     int
}

// RenderWorkerLanePlan stores one worker-lane assignment and its total estimated work weight.
type RenderWorkerLanePlan struct {
	GetWorkerIndex int
	GetTotalWeight int
	GetChunkPlans  []RenderWorkerChunkPlan
}

// BuildRenderWorkerAdaptiveChunkBounds partitions index ranges into contiguous chunks with roughly balanced weight.
func BuildRenderWorkerAdaptiveChunkBounds(parseWeights []int, parseChunkCount int) [][2]int {
	getItemCount := len(parseWeights)
	if getItemCount < 1 {
		return nil
	}
	getChunkCount := parseChunkCount
	if getChunkCount < 1 {
		getChunkCount = 1
	}
	if getChunkCount > getItemCount {
		getChunkCount = getItemCount
	}
	getTotalWeight := 0
	for _, parseWeight := range parseWeights {
		getItemWeight := parseWeight
		if getItemWeight < 1 {
			getItemWeight = 1
		}
		getTotalWeight += getItemWeight
	}
	if getTotalWeight < 1 {
		getTotalWeight = getItemCount
	}
	getBounds := make([][2]int, 0, getChunkCount)
	getCurrentStart := 0
	getRemainingWeight := getTotalWeight
	for parseChunkIndex := 0; parseChunkIndex < getChunkCount; parseChunkIndex++ {
		getRemainingChunkCount := getChunkCount - parseChunkIndex
		if getRemainingChunkCount == 1 {
			getBounds = append(getBounds, [2]int{getCurrentStart, getItemCount})
			break
		}
		getMaximumEnd := getItemCount - (getRemainingChunkCount - 1)
		getTargetWeight := getRemainingWeight / getRemainingChunkCount
		if getTargetWeight < 1 {
			getTargetWeight = 1
		}
		getCurrentWeight := 0
		getCurrentEnd := getCurrentStart
		for getCurrentEnd < getMaximumEnd {
			getItemWeight := parseWeights[getCurrentEnd]
			if getItemWeight < 1 {
				getItemWeight = 1
			}
			if getCurrentEnd > getCurrentStart && getCurrentWeight >= getTargetWeight {
				break
			}
			getCurrentWeight += getItemWeight
			getCurrentEnd++
		}
		if getCurrentEnd <= getCurrentStart {
			getCurrentEnd = getCurrentStart + 1
			getCurrentWeight = parseWeights[getCurrentStart]
			if getCurrentWeight < 1 {
				getCurrentWeight = 1
			}
		}
		getBounds = append(getBounds, [2]int{getCurrentStart, getCurrentEnd})
		getCurrentStart = getCurrentEnd
		getRemainingWeight -= getCurrentWeight
		if getRemainingWeight < 0 {
			getRemainingWeight = 0
		}
	}
	return getBounds
}

// BuildRenderWorkerChunkPlans materializes weighted chunk plans from contiguous chunk bounds.
func BuildRenderWorkerChunkPlans(parseChunkBounds [][2]int, parseWeights []int) []RenderWorkerChunkPlan {
	getChunkPlans := make([]RenderWorkerChunkPlan, 0, len(parseChunkBounds))
	for parseChunkIndex, parseChunkBound := range parseChunkBounds {
		getStart := parseChunkBound[0]
		getEnd := parseChunkBound[1]
		if getStart < 0 {
			getStart = 0
		}
		if getEnd < getStart {
			getEnd = getStart
		}
		if getEnd > len(parseWeights) {
			getEnd = len(parseWeights)
		}
		getChunkWeight := 0
		for parseWeightIndex := getStart; parseWeightIndex < getEnd; parseWeightIndex++ {
			getItemWeight := parseWeights[parseWeightIndex]
			if getItemWeight < 1 {
				getItemWeight = 1
			}
			getChunkWeight += getItemWeight
		}
		if getChunkWeight < 1 {
			getChunkWeight = 1
		}
		getChunkPlans = append(getChunkPlans, RenderWorkerChunkPlan{
			GetChunkIndex: parseChunkIndex,
			GetStart:      getStart,
			GetEnd:        getEnd,
			GetWeight:     getChunkWeight,
		})
	}
	return getChunkPlans
}

// BuildRenderWorkerLanePlans assigns chunk plans to worker lanes using weighted greedy balancing.
func BuildRenderWorkerLanePlans(parseWorkerCount int, parseChunkPlans []RenderWorkerChunkPlan) []RenderWorkerLanePlan {
	getWorkerCount := parseWorkerCount
	if getWorkerCount < 1 {
		getWorkerCount = 1
	}
	getLanePlans := make([]RenderWorkerLanePlan, getWorkerCount)
	for parseWorkerIndex := 0; parseWorkerIndex < getWorkerCount; parseWorkerIndex++ {
		getLanePlans[parseWorkerIndex].GetWorkerIndex = parseWorkerIndex
	}
	getSortedChunkPlans := append([]RenderWorkerChunkPlan(nil), parseChunkPlans...)
	sort.SliceStable(getSortedChunkPlans, func(parseLeft int, parseRight int) bool {
		if getSortedChunkPlans[parseLeft].GetWeight == getSortedChunkPlans[parseRight].GetWeight {
			return getSortedChunkPlans[parseLeft].GetChunkIndex < getSortedChunkPlans[parseRight].GetChunkIndex
		}
		return getSortedChunkPlans[parseLeft].GetWeight > getSortedChunkPlans[parseRight].GetWeight
	})
	for _, parseChunkPlan := range getSortedChunkPlans {
		getTargetLaneIndex := 0
		for parseLaneIndex := 1; parseLaneIndex < len(getLanePlans); parseLaneIndex++ {
			if getLanePlans[parseLaneIndex].GetTotalWeight < getLanePlans[getTargetLaneIndex].GetTotalWeight {
				getTargetLaneIndex = parseLaneIndex
			}
		}
		getLanePlans[getTargetLaneIndex].GetChunkPlans = append(getLanePlans[getTargetLaneIndex].GetChunkPlans, parseChunkPlan)
		getLanePlans[getTargetLaneIndex].GetTotalWeight += parseChunkPlan.GetWeight
	}
	for parseLaneIndex := 0; parseLaneIndex < len(getLanePlans); parseLaneIndex++ {
		sort.Slice(getLanePlans[parseLaneIndex].GetChunkPlans, func(parseLeft int, parseRight int) bool {
			return getLanePlans[parseLaneIndex].GetChunkPlans[parseLeft].GetChunkIndex < getLanePlans[parseLaneIndex].GetChunkPlans[parseRight].GetChunkIndex
		})
	}
	return getLanePlans
}

// BuildRenderWorkerChunkLocalIndexes filters global dirty indexes into one chunk-local [start, end) index set.
func BuildRenderWorkerChunkLocalIndexes(parseGlobalIndexes []int, parseStart int, parseEnd int) []int {
	if len(parseGlobalIndexes) < 1 || parseEnd <= parseStart {
		return nil
	}
	getLocalIndexes := make([]int, 0, len(parseGlobalIndexes))
	for _, parseGlobalIndex := range parseGlobalIndexes {
		if parseGlobalIndex < parseStart || parseGlobalIndex >= parseEnd {
			continue
		}
		getLocalIndexes = append(getLocalIndexes, parseGlobalIndex-parseStart)
	}
	return getLocalIndexes
}
