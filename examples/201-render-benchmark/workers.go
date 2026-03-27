//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	benchmarkshared "github.com/monstercameron/GoWebComponents/examples/201-render-benchmark/shared"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
)

const (
	getBenchmarkWorkerRuntimeURL     = "/examples/201-render-benchmark/vendor/wasm_exec.js"
	getBenchmarkWorkerWASMURL        = "/static/bin/render-benchmark-worker.wasm"
	getBenchmarkWorkerReadyTimeout   = 5 * time.Second
	getBenchmarkWorkerRequestTimeout = 5 * time.Second
	getBenchmarkWorkerDefaultScale   = 1
	getBenchmarkWorkerDispatchBatch  = "batch"
	getBenchmarkWorkerDispatchChunk  = "chunk"
	getBenchmarkWorkerCoreFastPath   = "local-fastpath"
)

const (
	getBenchmarkWorkerDependencySeed uint64 = 14695981039346656037
	getBenchmarkWorkerDependencyStep uint64 = 1099511628211
	getBenchmarkWorkerSampleWidth    int    = 8
)

const (
	getBenchmarkWorkerCoreFastPathMaxItems = 64
	getBenchmarkWorkerCoreFastPathCacheMax = 4096
)

type buildBenchmarkWorkerState struct {
	IsBooting             bool
	IsReady               bool
	IsPreparing           bool
	GetWorkerCount        int
	GetPreparedBatchCount int
	GetPreparedChunks     int
	GetAdaptiveChunkCount int
	GetCacheHitCount      int
	GetPreparedItems      int
	GetLastBatchMS        int64
	GetErrorText          string
}

type buildBenchmarkWorkerChunkPlan struct {
	GetChunkIndex int
	GetStart      int
	GetEnd        int
	GetWeight     int
}

type buildBenchmarkWorkerLanePlan struct {
	GetWorkerIndex int
	GetChunkPlans  []buildBenchmarkWorkerChunkPlan
	GetTotalWeight int
}

type buildBenchmarkWorkerCoreBatchReport struct {
	GetChunks             []benchmarkshared.BenchmarkWorkerCoreChunkResult
	GetAdaptiveChunkCount int
	GetCacheHitCount      int
}

type buildBenchmarkWorkerContentBatchReport struct {
	GetChunks             []benchmarkshared.BenchmarkWorkerContentChunkResult
	GetAdaptiveChunkCount int
	GetCacheHitCount      int
}

var warnBenchmarkWorkerChunkDispatchOnce sync.Once
var warnBenchmarkWorkerCoreFastPathOnce sync.Once
var getBenchmarkWorkerCoreFastPathCacheMu sync.RWMutex
var getBenchmarkWorkerCoreFastPathCacheByKey = map[string]benchmarkshared.BenchmarkPreparedCoreItem{}

// hasBenchmarkWorkerMode reports whether one benchmark mode should offload chunk preparation to a Go WASM worker fleet.
func hasBenchmarkWorkerMode(parseMode string) bool {
	return parseMode == benchmarkModeRuntime3 || strings.HasPrefix(parseMode, benchmarkModeRuntime3Workers)
}

// buildBenchmarkWorkerCount resolves the Go WASM worker-fleet size for one benchmark mode.
func buildBenchmarkWorkerCount(parseMode string) int {
	if parseMode == benchmarkModeRuntime3 {
		return 1
	}
	if !strings.HasPrefix(parseMode, benchmarkModeRuntime3Workers) {
		return 0
	}
	getWorkerCountText := strings.TrimSpace(buildBenchmarkQueryValues().Get("workers"))
	if getWorkerCountText != "" {
		if getWorkerCount, parseWorkerCountErr := strconv.Atoi(getWorkerCountText); parseWorkerCountErr == nil && getWorkerCount > 0 {
			return getWorkerCount
		}
	}
	getWorkerSuffix := strings.TrimPrefix(parseMode, benchmarkModeRuntime3Workers)
	getWorkerSuffix = strings.TrimLeft(getWorkerSuffix, "-")
	if getWorkerSuffix != "" {
		if getWorkerCount, parseWorkerCountErr := strconv.Atoi(getWorkerSuffix); parseWorkerCountErr == nil && getWorkerCount > 0 {
			return getWorkerCount
		}
	}
	return benchmarkRuntime3ShardCount
}

// buildBenchmarkWorkerWorkScale resolves the worker-preparation scale factor for one benchmark run.
func buildBenchmarkWorkerWorkScale() int {
	getWorkScaleText := strings.TrimSpace(buildBenchmarkQueryValues().Get("runtime2WorkScale"))
	if getWorkScaleText == "" {
		return getBenchmarkWorkerDefaultScale
	}
	getWorkScale, parseWorkScaleErr := strconv.Atoi(getWorkScaleText)
	if parseWorkScaleErr != nil || getWorkScale < 1 {
		return getBenchmarkWorkerDefaultScale
	}
	return getWorkScale
}

// buildBenchmarkWorkerDispatchMode resolves the worker dispatch strategy for one benchmark run.
func buildBenchmarkWorkerDispatchMode() string {
	getDispatchText := strings.ToLower(strings.TrimSpace(buildBenchmarkQueryValues().Get("runtime2Dispatch")))
	switch getDispatchText {
	case "", getBenchmarkWorkerDispatchBatch, "lanes", "lane", "adaptive":
		return getBenchmarkWorkerDispatchBatch
	case getBenchmarkWorkerDispatchChunk, "legacy", "legacy-chunk", "per-chunk":
		return getBenchmarkWorkerDispatchChunk
	default:
		fmt.Printf("[render-benchmark/runtime2][warn] unknown runtime2Dispatch=%q, defaulting to %q\n", getDispatchText, getBenchmarkWorkerDispatchBatch)
		return getBenchmarkWorkerDispatchBatch
	}
}

// hasBenchmarkWorkerCoreFastPath reports whether one runtime2 core batch should bypass worker RPC and use local cached preparation.
func hasBenchmarkWorkerCoreFastPath(parseMode string, parseItemCount int) bool {
	if !hasBenchmarkWorkerMode(parseMode) || parseItemCount < 1 || parseItemCount > getBenchmarkWorkerCoreFastPathMaxItems {
		return false
	}
	if buildBenchmarkWorkerWorkScale() > 1 {
		return false
	}
	getFastPathText := strings.ToLower(strings.TrimSpace(buildBenchmarkQueryValues().Get("runtime2CoreFastPath")))
	switch getFastPathText {
	case "", "1", "true", "on", "auto", "local":
		return true
	case "0", "false", "off", "worker":
		return false
	default:
		fmt.Printf("[render-benchmark/runtime2][warn] unknown runtime2CoreFastPath=%q, defaulting to enabled\n", getFastPathText)
		return true
	}
}

// warnBenchmarkWorkerCoreFastPath logs one warning when runtime2 enables local core fast-path preparation.
func warnBenchmarkWorkerCoreFastPath() {
	warnBenchmarkWorkerCoreFastPathOnce.Do(func() {
		fmt.Printf("[render-benchmark/runtime2][warn] runtime2CoreFastPath enabled; core batches <= %d items bypass worker RPC using one local cache-backed prepare path\n", getBenchmarkWorkerCoreFastPathMaxItems)
	})
}

// buildBenchmarkWorkerCoreItemsDependency hashes core-item payload content into one comparable dependency token.
// Every item is mixed so collisions become negligible and the chunk-cache guard never needs a full item scan.
func buildBenchmarkWorkerCoreItemsDependency(parseItems []benchmarkshared.BenchmarkCoreRowData) uint64 {
	if len(parseItems) == 0 {
		return getBenchmarkWorkerDependencySeed
	}
	getHash := getBenchmarkWorkerDependencySeed ^ uint64(len(parseItems))
	for parseItemIndex, parseItem := range parseItems {
		getHash = buildBenchmarkWorkerCoreItemDependency(getHash, parseItem, parseItemIndex)
	}
	return getHash
}

// buildBenchmarkWorkerContentItemsDependency hashes content-card payload content into one comparable dependency token.
// Every item is mixed so collisions become negligible and the chunk-cache guard never needs a full item scan.
func buildBenchmarkWorkerContentItemsDependency(parseItems []benchmarkshared.BenchmarkContentCardData) uint64 {
	if len(parseItems) == 0 {
		return getBenchmarkWorkerDependencySeed
	}
	getHash := getBenchmarkWorkerDependencySeed ^ uint64(len(parseItems))
	for parseItemIndex, parseItem := range parseItems {
		getHash = buildBenchmarkWorkerContentItemDependency(getHash, parseItem, parseItemIndex)
	}
	return getHash
}

// buildBenchmarkWorkerSampleIndexes returns one deduplicated first/middle/last sample index set for one item count.
func buildBenchmarkWorkerSampleIndexes(parseItemCount int) []int {
	if parseItemCount < 1 {
		return nil
	}
	getIndexes := []int{
		0,
		parseItemCount / 3,
		(parseItemCount * 2) / 3,
		parseItemCount - 1,
	}
	getSampleIndexes := make([]int, 0, len(getIndexes))
	getSeenIndexByValue := map[int]struct{}{}
	for _, getIndex := range getIndexes {
		if getIndex < 0 || getIndex >= parseItemCount {
			continue
		}
		if _, hasIndex := getSeenIndexByValue[getIndex]; hasIndex {
			continue
		}
		getSeenIndexByValue[getIndex] = struct{}{}
		getSampleIndexes = append(getSampleIndexes, getIndex)
	}
	return getSampleIndexes
}

// buildBenchmarkWorkerCoreItemDependency mixes one sampled core item into one dependency hash.
func buildBenchmarkWorkerCoreItemDependency(parseHash uint64, parseItem benchmarkshared.BenchmarkCoreRowData, parseItemIndex int) uint64 {
	getHash := parseHash
	getHash ^= (uint64(parseItem.GetID) << 1) + uint64(parseItemIndex+1)
	getHash *= getBenchmarkWorkerDependencyStep
	getHash ^= uint64(len(parseItem.GetText))
	getHash *= getBenchmarkWorkerDependencyStep
	getHash ^= buildBenchmarkWorkerStringSampleDependency(parseItem.GetText)
	getHash *= getBenchmarkWorkerDependencyStep
	return getHash
}

// buildBenchmarkWorkerContentItemDependency mixes one sampled content card into one dependency hash.
func buildBenchmarkWorkerContentItemDependency(parseHash uint64, parseItem benchmarkshared.BenchmarkContentCardData, parseItemIndex int) uint64 {
	getHash := parseHash
	getHash ^= (uint64(parseItem.GetID) << 1) + uint64(parseItemIndex+1)
	getHash *= getBenchmarkWorkerDependencyStep
	getHash ^= uint64(len(parseItem.GetTitle))
	getHash *= getBenchmarkWorkerDependencyStep
	getHash ^= buildBenchmarkWorkerStringSampleDependency(parseItem.GetTitle)
	getHash *= getBenchmarkWorkerDependencyStep
	getHash ^= uint64(len(parseItem.GetSummary))
	getHash *= getBenchmarkWorkerDependencyStep
	getHash ^= buildBenchmarkWorkerStringSampleDependency(parseItem.GetSummary)
	getHash *= getBenchmarkWorkerDependencyStep
	getHash ^= uint64(len(parseItem.GetStatus))
	getHash *= getBenchmarkWorkerDependencyStep
	getHash ^= buildBenchmarkWorkerStringSampleDependency(parseItem.GetStatus)
	getHash *= getBenchmarkWorkerDependencyStep
	getHash ^= uint64(len(parseItem.GetMeta))
	getHash *= getBenchmarkWorkerDependencyStep
	getHash ^= buildBenchmarkWorkerStringSampleDependency(parseItem.GetMeta)
	getHash *= getBenchmarkWorkerDependencyStep
	for _, getTag := range parseItem.GetTags {
		getHash ^= uint64(len(getTag))
		getHash *= getBenchmarkWorkerDependencyStep
		getHash ^= buildBenchmarkWorkerStringSampleDependency(getTag)
		getHash *= getBenchmarkWorkerDependencyStep
	}
	return getHash
}

// buildBenchmarkWorkerStringSampleDependency hashes one short start/end sample window from one string.
func buildBenchmarkWorkerStringSampleDependency(parseText string) uint64 {
	if len(parseText) == 0 {
		return 0
	}
	getHash := uint64(len(parseText))
	getWidth := getBenchmarkWorkerSampleWidth
	if getWidth < 1 {
		getWidth = 1
	}
	if getWidth > len(parseText) {
		getWidth = len(parseText)
	}
	for parseOffset := 0; parseOffset < getWidth; parseOffset++ {
		getHash ^= uint64(parseText[parseOffset])
		getHash *= getBenchmarkWorkerDependencyStep
	}
	getTailStart := len(parseText) - getWidth
	for parseOffset := getTailStart; parseOffset < len(parseText); parseOffset++ {
		getHash ^= uint64(parseText[parseOffset])
		getHash *= getBenchmarkWorkerDependencyStep
	}
	getMiddleIndex := len(parseText) / 2
	getHash ^= uint64(parseText[getMiddleIndex])
	getHash *= getBenchmarkWorkerDependencyStep
	return getHash
}

// buildBenchmarkCoreItemsClone copies one core-item slice so prepared snapshots can be compared safely across updates.
func buildBenchmarkCoreItemsClone(parseItems []benchmarkshared.BenchmarkCoreRowData) []benchmarkshared.BenchmarkCoreRowData {
	if len(parseItems) < 1 {
		return nil
	}
	getItems := make([]benchmarkshared.BenchmarkCoreRowData, len(parseItems))
	copy(getItems, parseItems)
	return getItems
}

// buildBenchmarkCoreChunkItemCount counts prepared core items across one chunk result slice.
func buildBenchmarkCoreChunkItemCount(parseChunks []benchmarkshared.BenchmarkWorkerCoreChunkResult) int {
	getItemCount := 0
	for _, getChunk := range parseChunks {
		getItemCount += len(getChunk.GetItems)
	}
	return getItemCount
}

// buildBenchmarkCoreAppendTail detects append-only updates and returns only the appended core items.
func buildBenchmarkCoreAppendTail(parsePreviousItems []benchmarkshared.BenchmarkCoreRowData, parseCurrentItems []benchmarkshared.BenchmarkCoreRowData) ([]benchmarkshared.BenchmarkCoreRowData, bool) {
	if len(parsePreviousItems) < 1 || len(parseCurrentItems) <= len(parsePreviousItems) {
		return nil, false
	}
	for parseItemIndex := 0; parseItemIndex < len(parsePreviousItems); parseItemIndex++ {
		getPreviousItem := parsePreviousItems[parseItemIndex]
		getCurrentItem := parseCurrentItems[parseItemIndex]
		if getPreviousItem.GetID != getCurrentItem.GetID || getPreviousItem.GetText != getCurrentItem.GetText {
			return nil, false
		}
	}
	return parseCurrentItems[len(parsePreviousItems):], true
}

// warnBenchmarkWorkerChunkDispatch logs one warning when the benchmark is configured to use legacy per-chunk dispatch.
func warnBenchmarkWorkerChunkDispatch() {
	warnBenchmarkWorkerChunkDispatchOnce.Do(func() {
		fmt.Printf("[render-benchmark/runtime2][warn] runtime2Dispatch=chunk enables legacy per-chunk requests; use runtime2Dispatch=batch for lower worker-dispatch overhead\n")
	})
}

// buildBenchmarkWorkerAutoChunkCount resolves one data-driven chunk fanout target from worker count, item count, and last batch latency.
func buildBenchmarkWorkerAutoChunkCount(parseMode string, parseWorkerState buildBenchmarkWorkerState, parseItemCount int) int {
	if parseItemCount < 1 {
		return 0
	}
	getWorkerCount := buildBenchmarkRuntime3ChunkCount(parseMode)
	if getWorkerCount < 1 {
		getWorkerCount = 1
	}
	// scalar defaults to base worker count; scale up with item count and last-batch latency
	getAdaptiveChunkCount := getWorkerCount
	if parseItemCount >= getWorkerCount*4 {
		// scale chunk multiplier based on last observed batch latency:
		//   fast batches (< 3 ms)   → keep count low to reduce IPC round-trip overhead
		//   normal batches (3–8 ms) → 2× workers (original heuristic)
		//   slow batches (> 8 ms)   → 4× workers to maximise progressive delivery fanout
		getLastBatchMS := parseWorkerState.GetLastBatchMS
		switch {
		case getLastBatchMS > 0 && getLastBatchMS < 3:
			getAdaptiveChunkCount = getWorkerCount
		case getLastBatchMS > 8:
			getAdaptiveChunkCount = getWorkerCount * 4
		default:
			getAdaptiveChunkCount = getWorkerCount * 2
		}
	}
	if getAdaptiveChunkCount > parseItemCount {
		getAdaptiveChunkCount = parseItemCount
	}
	if getAdaptiveChunkCount < 1 {
		getAdaptiveChunkCount = 1
	}
	return getAdaptiveChunkCount
}

// buildBenchmarkWorkerCoreItemWeights estimates per-item preparation cost for core-list payloads.
func buildBenchmarkWorkerCoreItemWeights(parseItems []benchmarkshared.BenchmarkCoreRowData) []int {
	getWeights := make([]int, len(parseItems))
	for parseItemIndex, parseItem := range parseItems {
		getItemWeight := 32 + (len(strings.TrimSpace(parseItem.GetText)) * 6)
		if getItemWeight < 1 {
			getItemWeight = 1
		}
		getWeights[parseItemIndex] = getItemWeight
	}
	return getWeights
}

// buildBenchmarkWorkerContentItemWeights estimates per-item preparation cost for content-card payloads.
func buildBenchmarkWorkerContentItemWeights(parseItems []benchmarkshared.BenchmarkContentCardData) []int {
	getWeights := make([]int, len(parseItems))
	for parseItemIndex, parseItem := range parseItems {
		getTagTextLength := 0
		for _, getTag := range parseItem.GetTags {
			getTagTextLength += len(strings.TrimSpace(getTag))
		}
		getItemWeight := 64 +
			(len(strings.TrimSpace(parseItem.GetTitle)) * 6) +
			(len(strings.TrimSpace(parseItem.GetSummary)) * 8) +
			(len(strings.TrimSpace(parseItem.GetMeta)) * 4) +
			(len(strings.TrimSpace(parseItem.GetStatus)) * 2) +
			(len(parseItem.GetTags) * 20) +
			(getTagTextLength * 4)
		if getItemWeight < 1 {
			getItemWeight = 1
		}
		getWeights[parseItemIndex] = getItemWeight
	}
	return getWeights
}

// buildBenchmarkWorkerAdaptiveChunkBoundsFromWeights partitions list indexes into contiguous chunks with roughly balanced estimated weight.
func buildBenchmarkWorkerAdaptiveChunkBoundsFromWeights(parseWeights []int, parseChunkCount int) [][2]int {
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
	for parseItemIndex := 0; parseItemIndex < getItemCount; parseItemIndex++ {
		getItemWeight := parseWeights[parseItemIndex]
		if getItemWeight < 1 {
			getItemWeight = 1
		}
		getTotalWeight += getItemWeight
	}
	getBounds := make([][2]int, 0, getChunkCount)
	getRemainingWeight := getTotalWeight
	getStart := 0
	for parseChunkIndex := 0; parseChunkIndex < getChunkCount; parseChunkIndex++ {
		getRemainingChunkCount := getChunkCount - parseChunkIndex
		if getRemainingChunkCount == 1 {
			getBounds = append(getBounds, [2]int{getStart, getItemCount})
			break
		}
		getMaximumEnd := getItemCount - (getRemainingChunkCount - 1)
		getTargetWeight := getRemainingWeight / getRemainingChunkCount
		if getTargetWeight < 1 {
			getTargetWeight = 1
		}
		getEnd := getStart
		getCurrentWeight := 0
		for getEnd < getMaximumEnd {
			getItemWeight := parseWeights[getEnd]
			if getItemWeight < 1 {
				getItemWeight = 1
			}
			getCurrentWeight += getItemWeight
			getEnd++
			if getCurrentWeight >= getTargetWeight {
				break
			}
		}
		if getEnd <= getStart {
			getEnd = getStart + 1
		}
		getBounds = append(getBounds, [2]int{getStart, getEnd})
		getRemainingWeight -= getCurrentWeight
		getStart = getEnd
	}
	return getBounds
}

// buildBenchmarkWorkerChunkPlans materializes weighted chunk plans from contiguous chunk bounds.
func buildBenchmarkWorkerChunkPlans(parseChunkBounds [][2]int, parseWeights []int) []buildBenchmarkWorkerChunkPlan {
	getChunkPlans := make([]buildBenchmarkWorkerChunkPlan, 0, len(parseChunkBounds))
	for parseChunkIndex, getChunkBound := range parseChunkBounds {
		getStart := getChunkBound[0]
		getEnd := getChunkBound[1]
		getChunkWeight := 0
		for parseItemIndex := getStart; parseItemIndex < getEnd && parseItemIndex < len(parseWeights); parseItemIndex++ {
			getItemWeight := parseWeights[parseItemIndex]
			if getItemWeight < 1 {
				getItemWeight = 1
			}
			getChunkWeight += getItemWeight
		}
		if getChunkWeight < 1 {
			getChunkWeight = 1
		}
		getChunkPlans = append(getChunkPlans, buildBenchmarkWorkerChunkPlan{
			GetChunkIndex: parseChunkIndex,
			GetStart:      getStart,
			GetEnd:        getEnd,
			GetWeight:     getChunkWeight,
		})
	}
	return getChunkPlans
}

// buildBenchmarkWorkerLanePlans assigns chunk plans to worker lanes using weighted greedy balancing.
func buildBenchmarkWorkerLanePlans(parseWorkerCount int, parseChunkPlans []buildBenchmarkWorkerChunkPlan) []buildBenchmarkWorkerLanePlan {
	getWorkerCount := parseWorkerCount
	if getWorkerCount < 1 {
		getWorkerCount = 1
	}
	getLanePlans := make([]buildBenchmarkWorkerLanePlan, getWorkerCount)
	for parseWorkerIndex := 0; parseWorkerIndex < getWorkerCount; parseWorkerIndex++ {
		getLanePlans[parseWorkerIndex].GetWorkerIndex = parseWorkerIndex
	}
	getSortedChunkPlans := append([]buildBenchmarkWorkerChunkPlan(nil), parseChunkPlans...)
	sort.SliceStable(getSortedChunkPlans, func(parseLeft int, parseRight int) bool {
		if getSortedChunkPlans[parseLeft].GetWeight == getSortedChunkPlans[parseRight].GetWeight {
			return getSortedChunkPlans[parseLeft].GetChunkIndex < getSortedChunkPlans[parseRight].GetChunkIndex
		}
		return getSortedChunkPlans[parseLeft].GetWeight > getSortedChunkPlans[parseRight].GetWeight
	})
	for _, getChunkPlan := range getSortedChunkPlans {
		getTargetLaneIndex := 0
		for parseLaneIndex := 1; parseLaneIndex < len(getLanePlans); parseLaneIndex++ {
			if getLanePlans[parseLaneIndex].GetTotalWeight < getLanePlans[getTargetLaneIndex].GetTotalWeight {
				getTargetLaneIndex = parseLaneIndex
			}
		}
		getLanePlans[getTargetLaneIndex].GetChunkPlans = append(getLanePlans[getTargetLaneIndex].GetChunkPlans, getChunkPlan)
		getLanePlans[getTargetLaneIndex].GetTotalWeight += getChunkPlan.GetWeight
	}
	for parseLaneIndex := 0; parseLaneIndex < len(getLanePlans); parseLaneIndex++ {
		sort.Slice(getLanePlans[parseLaneIndex].GetChunkPlans, func(parseLeft int, parseRight int) bool {
			return getLanePlans[parseLaneIndex].GetChunkPlans[parseLeft].GetChunkIndex < getLanePlans[parseLaneIndex].GetChunkPlans[parseRight].GetChunkIndex
		})
	}
	return getLanePlans
}

// openBenchmarkWorkerFleet opens the Go WASM worker fleet used by the runtime2 benchmark modes.
func openBenchmarkWorkerFleet(parseCtx context.Context, parseMode string) ([]interop.Worker, error) {
	getWorkerCount := buildBenchmarkWorkerCount(parseMode)
	getWorkers := make([]interop.Worker, 0, getWorkerCount)
	for parseWorkerIndex := 0; parseWorkerIndex < getWorkerCount; parseWorkerIndex++ {
		getWorkerName := fmt.Sprintf("render-benchmark-%s-%d", parseMode, parseWorkerIndex+1)
		getWorker, parseErr := interop.OpenGoWASMWorker(parseCtx, interop.GoWASMWorkerOptions{
			RuntimeURL:   getBenchmarkWorkerRuntimeURL,
			WASMURL:      getBenchmarkWorkerWASMURL,
			Name:         getWorkerName,
			Ready:        true,
			ReadyTimeout: getBenchmarkWorkerReadyTimeout,
		})
		if parseErr != nil {
			_ = closeBenchmarkWorkerFleet(getWorkers)
			return nil, parseErr
		}
		getWorkers = append(getWorkers, getWorker)
	}
	return getWorkers, nil
}

// closeBenchmarkWorkerFleet terminates each worker in one benchmark worker fleet and joins non-disposed errors.
func closeBenchmarkWorkerFleet(parseWorkers []interop.Worker) error {
	var getCloseErrors []error
	for _, getWorker := range parseWorkers {
		if parseErr := getWorker.Terminate(); parseErr != nil && !interop.IsCode(parseErr, interop.CodeDisposed) {
			getCloseErrors = append(getCloseErrors, parseErr)
		}
	}
	return errors.Join(getCloseErrors...)
}

// handleBenchmarkWorkerFleetEffect boots and tears down the benchmark worker fleet for runtime2 benchmark modes.
func handleBenchmarkWorkerFleetEffect(parseMode string, parseWorkersRef ui.Ref[[]interop.Worker], parseWorkersRevisionState ui.State[int], parseWorkerState ui.State[buildBenchmarkWorkerState]) {
	ui.UseEffect(func() func() {
		if !hasBenchmarkWorkerMode(parseMode) {
			return nil
		}
		if len(parseWorkersRef.Get()) > 0 {
			return nil
		}

		getWorkerCount := buildBenchmarkWorkerCount(parseMode)
		parseWorkerState.Set(buildBenchmarkWorkerState{
			IsBooting:      true,
			GetWorkerCount: getWorkerCount,
		})
		parseCtx, parseCancel := context.WithCancel(context.Background())
		go func() {
			getWorkers, parseErr := openBenchmarkWorkerFleet(parseCtx, parseMode)
			if parseErr != nil {
				if parseCtx.Err() != nil {
					return
				}
				parseWorkerState.Set(buildBenchmarkWorkerState{
					GetWorkerCount: getWorkerCount,
					GetErrorText:   parseErr.Error(),
				})
				return
			}
			if parseCtx.Err() != nil {
				_ = closeBenchmarkWorkerFleet(getWorkers)
				return
			}
			parseWorkersRef.Set(getWorkers)
			parseWorkerState.Set(buildBenchmarkWorkerState{
				IsReady:        true,
				GetWorkerCount: getWorkerCount,
			})
			parseWorkersRevisionState.Update(func(parsePrevious int) int {
				return parsePrevious + 1
			})
		}()

		return func() {
			parseCancel()
			if getWorkers := parseWorkersRef.Get(); len(getWorkers) > 0 {
				parseWorkersRef.Set(nil)
				_ = closeBenchmarkWorkerFleet(getWorkers)
			}
		}
	}, parseMode)
}

// handleBenchmarkWorkerPrepareEffect refreshes prepared chunk state for the worker-backed runtime2 benchmark modes.
func handleBenchmarkWorkerPrepareEffect(
	parseMode string,
	parseView string,
	parseCoreItems []benchmarkshared.BenchmarkCoreRowData,
	parseContentItems []benchmarkshared.BenchmarkContentCardData,
	parsePrepareRevision int,
	parseWorkersRevision int,
	parseWorkersRef ui.Ref[[]interop.Worker],
	parseGenerationRef ui.Ref[uint64],
	parseCoreChunkCacheByDependencyRef ui.Ref[map[uint64][]benchmarkshared.BenchmarkWorkerCoreChunkResult],
	parseContentChunkCacheByDependencyRef ui.Ref[map[uint64][]benchmarkshared.BenchmarkWorkerContentChunkResult],
	parseLastCorePreparedItemsRef ui.Ref[[]benchmarkshared.BenchmarkCoreRowData],
	parseCoreChunkState ui.State[[]benchmarkshared.BenchmarkWorkerCoreChunkResult],
	parseContentChunkState ui.State[[]benchmarkshared.BenchmarkWorkerContentChunkResult],
	parseWorkerState ui.State[buildBenchmarkWorkerState],
) {
	ui.UseEffect(func() func() {
		if !hasBenchmarkWorkerMode(parseMode) {
			return nil
		}
		getWorkers := parseWorkersRef.Get()
		if len(getWorkers) == 0 {
			return nil
		}
		if parseView == "core" && len(parseCoreItems) == 0 {
			parseCoreChunkState.Set(nil)
			parseLastCorePreparedItemsRef.Set(nil)
			parseWorkerState.Update(func(parsePrevious buildBenchmarkWorkerState) buildBenchmarkWorkerState {
				parsePrevious.IsPreparing = false
				parsePrevious.GetPreparedChunks = 0
				parsePrevious.GetAdaptiveChunkCount = 0
				parsePrevious.GetCacheHitCount = 0
				parsePrevious.GetPreparedItems = 0
				parsePrevious.GetLastBatchMS = 0
				parsePrevious.GetErrorText = ""
				return parsePrevious
			})
			return nil
		}
		if parseView == "content" && len(parseContentItems) == 0 {
			parseContentChunkState.Set(nil)
			parseLastCorePreparedItemsRef.Set(nil)
			parseWorkerState.Update(func(parsePrevious buildBenchmarkWorkerState) buildBenchmarkWorkerState {
				parsePrevious.IsPreparing = false
				parsePrevious.GetPreparedChunks = 0
				parsePrevious.GetAdaptiveChunkCount = 0
				parsePrevious.GetCacheHitCount = 0
				parsePrevious.GetPreparedItems = 0
				parsePrevious.GetLastBatchMS = 0
				parsePrevious.GetErrorText = ""
				return parsePrevious
			})
			return nil
		}
		if parseView != "core" && parseView != "content" {
			parseLastCorePreparedItemsRef.Set(nil)
			parseWorkerState.Update(func(parsePrevious buildBenchmarkWorkerState) buildBenchmarkWorkerState {
				parsePrevious.IsPreparing = false
				parsePrevious.GetErrorText = ""
				return parsePrevious
			})
			return nil
		}
		getCoreItemsDependency := uint64(0)
		getContentItemsDependency := uint64(0)
		if parseView == "core" {
			getCoreItemsDependency = buildBenchmarkWorkerCoreItemsDependency(parseCoreItems)
			getCachedCoreChunks, hasCachedCoreChunks := getBenchmarkWorkerCoreChunkCache(parseCoreChunkCacheByDependencyRef, getCoreItemsDependency, parseCoreItems)
			if hasCachedCoreChunks {
				parseCoreChunkState.Set(getCachedCoreChunks)
				parseLastCorePreparedItemsRef.Set(buildBenchmarkCoreItemsClone(parseCoreItems))
				parseWorkerState.Update(func(parsePrevious buildBenchmarkWorkerState) buildBenchmarkWorkerState {
					parsePrevious.IsPreparing = false
					parsePrevious.IsReady = true
					parsePrevious.GetPreparedChunks = len(getCachedCoreChunks)
					parsePrevious.GetAdaptiveChunkCount = len(getCachedCoreChunks)
					parsePrevious.GetCacheHitCount = len(parseCoreItems)
					parsePrevious.GetPreparedItems = len(parseCoreItems)
					parsePrevious.GetLastBatchMS = 0
					parsePrevious.GetErrorText = ""
					return parsePrevious
				})
				return nil
			}
		}
		if parseView == "content" {
			getContentItemsDependency = buildBenchmarkWorkerContentItemsDependency(parseContentItems)
			getCachedContentChunks, hasCachedContentChunks := getBenchmarkWorkerContentChunkCache(parseContentChunkCacheByDependencyRef, getContentItemsDependency, parseContentItems)
			if hasCachedContentChunks {
				parseContentChunkState.Set(getCachedContentChunks)
				parseLastCorePreparedItemsRef.Set(nil)
				parseWorkerState.Update(func(parsePrevious buildBenchmarkWorkerState) buildBenchmarkWorkerState {
					parsePrevious.IsPreparing = false
					parsePrevious.IsReady = true
					parsePrevious.GetPreparedChunks = len(getCachedContentChunks)
					parsePrevious.GetAdaptiveChunkCount = len(getCachedContentChunks)
					parsePrevious.GetCacheHitCount = len(parseContentItems)
					parsePrevious.GetPreparedItems = len(parseContentItems)
					parsePrevious.GetLastBatchMS = 0
					parsePrevious.GetErrorText = ""
					return parsePrevious
				})
				return nil
			}
		}

		getCoreItemsForRequest := parseCoreItems
		var getCoreReuseChunks []benchmarkshared.BenchmarkWorkerCoreChunkResult
		if parseView == "core" {
			getAppendItems, hasAppendItems := buildBenchmarkCoreAppendTail(parseLastCorePreparedItemsRef.Get(), parseCoreItems)
			if hasAppendItems {
				getPreviousCoreChunks := parseCoreChunkState.Get()
				if len(getPreviousCoreChunks) > 0 && buildBenchmarkCoreChunkItemCount(getPreviousCoreChunks) == (len(parseCoreItems)-len(getAppendItems)) {
					getCoreItemsForRequest = getAppendItems
					getCoreReuseChunks = cloneBenchmarkWorkerCoreChunks(getPreviousCoreChunks)
				}
			}
		}

		parseCtx, parseCancel := context.WithTimeout(context.Background(), getBenchmarkWorkerRequestTimeout)
		parseStartedAt := time.Now()
		parseWorkerSnapshot := parseWorkerState.Get()
		parseGeneration := parseGenerationRef.Get() + 1
		parseGenerationRef.Set(parseGeneration)
		parseWorkerState.Update(func(parsePrevious buildBenchmarkWorkerState) buildBenchmarkWorkerState {
			parsePrevious.IsPreparing = true
			parsePrevious.GetErrorText = ""
			return parsePrevious
		})
		go func() {
			if parseView == "core" {
				// progressive partial delivery: emit partial chunk state as each worker lane responds
				onPartialCoreLaneDone := func(getPartialChunks []benchmarkshared.BenchmarkWorkerCoreChunkResult) {
					if parseGeneration != parseGenerationRef.Get() {
						return
					}
					getPreparedPartial := getPartialChunks
					if len(getCoreReuseChunks) > 0 {
						getPreparedPartial = append(append(make([]benchmarkshared.BenchmarkWorkerCoreChunkResult, 0, len(getCoreReuseChunks)+len(getPartialChunks)), getCoreReuseChunks...), getPartialChunks...)
					}
					parseCoreChunkState.Set(getPreparedPartial)
				}
				// compute dirty-set indexes for delta sends when previous items are available and no append occurred
				var getCoreItemsDirtyIndexes []int
				if len(getCoreItemsForRequest) == len(parseCoreItems) {
					getLastCoreItems := parseLastCorePreparedItemsRef.Get()
					if len(getLastCoreItems) == len(parseCoreItems) {
						getCoreItemsDirtyIndexes = buildBenchmarkWorkerCoreItemDirtyIndexes(getLastCoreItems, parseCoreItems)
					}
				}
				getBatchReport, parseErr := requestBenchmarkWorkerCoreChunks(parseCtx, getWorkers, parseMode, getCoreItemsForRequest, parseWorkerSnapshot, parseGeneration, getCoreItemsDirtyIndexes, onPartialCoreLaneDone)
				if parseCtx.Err() != nil {
					return
				}
				if parseGeneration != parseGenerationRef.Get() {
					fmt.Printf("[render-benchmark/runtime2][warn] dropping stale core worker batch generation=%d latest=%d\n", parseGeneration, parseGenerationRef.Get())
					return
				}
				if parseErr != nil {
					parseWorkerState.Update(func(parsePrevious buildBenchmarkWorkerState) buildBenchmarkWorkerState {
						parsePrevious.IsPreparing = false
						parsePrevious.GetErrorText = parseErr.Error()
						return parsePrevious
					})
					return
				}
				getPreparedCoreChunks := cloneBenchmarkWorkerCoreChunks(getBatchReport.GetChunks)
				if len(getCoreReuseChunks) > 0 {
					getPreparedCoreChunks = append(append(make([]benchmarkshared.BenchmarkWorkerCoreChunkResult, 0, len(getCoreReuseChunks)+len(getPreparedCoreChunks)), getCoreReuseChunks...), getPreparedCoreChunks...)
				}
				storeBenchmarkWorkerCoreChunkCache(parseCoreChunkCacheByDependencyRef, getCoreItemsDependency, getPreparedCoreChunks)
				parseCoreChunkState.Set(getPreparedCoreChunks)
				parseLastCorePreparedItemsRef.Set(buildBenchmarkCoreItemsClone(parseCoreItems))
				parseWorkerState.Update(func(parsePrevious buildBenchmarkWorkerState) buildBenchmarkWorkerState {
					parsePrevious.IsPreparing = false
					parsePrevious.IsReady = true
					parsePrevious.GetPreparedBatchCount++
					parsePrevious.GetPreparedChunks = len(getPreparedCoreChunks)
					parsePrevious.GetAdaptiveChunkCount = len(getPreparedCoreChunks)
					parsePrevious.GetCacheHitCount = getBatchReport.GetCacheHitCount
					parsePrevious.GetPreparedItems = len(parseCoreItems)
					parsePrevious.GetLastBatchMS = time.Since(parseStartedAt).Milliseconds()
					parsePrevious.GetErrorText = ""
					return parsePrevious
				})
				return
			}

			parseLastCorePreparedItemsRef.Set(nil)
			// progressive partial delivery: emit partial chunk state as each worker lane responds
			onPartialContentLaneDone := func(getPartialChunks []benchmarkshared.BenchmarkWorkerContentChunkResult) {
				if parseGeneration != parseGenerationRef.Get() {
					return
				}
				parseContentChunkState.Set(cloneBenchmarkWorkerContentChunks(getPartialChunks))
			}
			getBatchReport, parseErr := requestBenchmarkWorkerContentChunks(parseCtx, getWorkers, parseMode, parseContentItems, parseWorkerSnapshot, parseGeneration, nil, onPartialContentLaneDone)
			if parseCtx.Err() != nil {
				return
			}
			if parseGeneration != parseGenerationRef.Get() {
				fmt.Printf("[render-benchmark/runtime2][warn] dropping stale content worker batch generation=%d latest=%d\n", parseGeneration, parseGenerationRef.Get())
				return
			}
			if parseErr != nil {
				parseWorkerState.Update(func(parsePrevious buildBenchmarkWorkerState) buildBenchmarkWorkerState {
					parsePrevious.IsPreparing = false
					parsePrevious.GetErrorText = parseErr.Error()
					return parsePrevious
				})
				return
			}
			storeBenchmarkWorkerContentChunkCache(parseContentChunkCacheByDependencyRef, getContentItemsDependency, getBatchReport.GetChunks)
			parseContentChunkState.Set(cloneBenchmarkWorkerContentChunks(getBatchReport.GetChunks))
			parseWorkerState.Update(func(parsePrevious buildBenchmarkWorkerState) buildBenchmarkWorkerState {
				parsePrevious.IsPreparing = false
				parsePrevious.IsReady = true
				parsePrevious.GetPreparedBatchCount++
				parsePrevious.GetPreparedChunks = len(getBatchReport.GetChunks)
				parsePrevious.GetAdaptiveChunkCount = getBatchReport.GetAdaptiveChunkCount
				parsePrevious.GetCacheHitCount = getBatchReport.GetCacheHitCount
				parsePrevious.GetPreparedItems = len(parseContentItems)
				parsePrevious.GetLastBatchMS = time.Since(parseStartedAt).Milliseconds()
				parsePrevious.GetErrorText = ""
				return parsePrevious
			})
		}()
		return parseCancel
	}, parseMode, parseView, parsePrepareRevision, parseWorkersRevision)
}

// cloneBenchmarkWorkerCoreChunks clones one core chunk result slice so cache-backed reads keep stable ownership.
func cloneBenchmarkWorkerCoreChunks(parseChunks []benchmarkshared.BenchmarkWorkerCoreChunkResult) []benchmarkshared.BenchmarkWorkerCoreChunkResult {
	if len(parseChunks) == 0 {
		return nil
	}
	getChunks := make([]benchmarkshared.BenchmarkWorkerCoreChunkResult, len(parseChunks))
	copy(getChunks, parseChunks)
	return getChunks
}

// cloneBenchmarkWorkerContentChunks clones one content chunk result slice so cache-backed reads keep stable ownership.
func cloneBenchmarkWorkerContentChunks(parseChunks []benchmarkshared.BenchmarkWorkerContentChunkResult) []benchmarkshared.BenchmarkWorkerContentChunkResult {
	if len(parseChunks) == 0 {
		return nil
	}
	getChunks := make([]benchmarkshared.BenchmarkWorkerContentChunkResult, len(parseChunks))
	copy(getChunks, parseChunks)
	return getChunks
}

// getBenchmarkWorkerCoreChunkCache loads one cached core chunk result set for one dependency hash.
func getBenchmarkWorkerCoreChunkCache(parseCacheByDependencyRef ui.Ref[map[uint64][]benchmarkshared.BenchmarkWorkerCoreChunkResult], parseDependency uint64, parseItems []benchmarkshared.BenchmarkCoreRowData) ([]benchmarkshared.BenchmarkWorkerCoreChunkResult, bool) {
	getCacheByDependency := parseCacheByDependencyRef.Get()
	if len(getCacheByDependency) == 0 {
		return nil, false
	}
	getCachedChunks, hasCachedChunks := getCacheByDependency[parseDependency]
	if !hasCachedChunks || len(getCachedChunks) == 0 {
		return nil, false
	}
	if !hasBenchmarkWorkerCoreChunkCacheMatch(getCachedChunks, parseItems) {
		fmt.Printf("[render-benchmark/runtime2][warn] core cache guard rejected dependency=%d item-count=%d\n", parseDependency, len(parseItems))
		clearBenchmarkWorkerCoreChunkCacheByDependency(parseCacheByDependencyRef, parseDependency)
		return nil, false
	}
	return cloneBenchmarkWorkerCoreChunks(getCachedChunks), true
}

// getBenchmarkWorkerContentChunkCache loads one cached content chunk result set for one dependency hash.
func getBenchmarkWorkerContentChunkCache(parseCacheByDependencyRef ui.Ref[map[uint64][]benchmarkshared.BenchmarkWorkerContentChunkResult], parseDependency uint64, parseItems []benchmarkshared.BenchmarkContentCardData) ([]benchmarkshared.BenchmarkWorkerContentChunkResult, bool) {
	getCacheByDependency := parseCacheByDependencyRef.Get()
	if len(getCacheByDependency) == 0 {
		return nil, false
	}
	getCachedChunks, hasCachedChunks := getCacheByDependency[parseDependency]
	if !hasCachedChunks || len(getCachedChunks) == 0 {
		return nil, false
	}
	if !hasBenchmarkWorkerContentChunkCacheMatch(getCachedChunks, parseItems) {
		fmt.Printf("[render-benchmark/runtime2][warn] content cache guard rejected dependency=%d item-count=%d\n", parseDependency, len(parseItems))
		clearBenchmarkWorkerContentChunkCacheByDependency(parseCacheByDependencyRef, parseDependency)
		return nil, false
	}
	return cloneBenchmarkWorkerContentChunks(getCachedChunks), true
}

// hasBenchmarkWorkerCoreChunkCacheMatch verifies one cached core chunk set item count matches the requested source core items.
// The full per-item scan is no longer required because buildBenchmarkWorkerCoreItemsDependency now hashes every item;
// a count mismatch is the only case the stronger hash cannot rule out on its own.
func hasBenchmarkWorkerCoreChunkCacheMatch(parseChunks []benchmarkshared.BenchmarkWorkerCoreChunkResult, parseItems []benchmarkshared.BenchmarkCoreRowData) bool {
	return buildBenchmarkCoreChunkItemCount(parseChunks) == len(parseItems)
}

// hasBenchmarkWorkerContentChunkCacheMatch verifies one cached content chunk set item count matches the requested source content items.
// The full per-item scan is no longer required because buildBenchmarkWorkerContentItemsDependency now hashes every item;
// a count mismatch is the only case the stronger hash cannot rule out on its own.
func hasBenchmarkWorkerContentChunkCacheMatch(parseChunks []benchmarkshared.BenchmarkWorkerContentChunkResult, parseItems []benchmarkshared.BenchmarkContentCardData) bool {
	getItemCount := 0
	for _, getChunk := range parseChunks {
		getItemCount += len(getChunk.GetItems)
	}
	return getItemCount == len(parseItems)
}

// hasBenchmarkWorkerTagSetMatch reports whether two tag slices are equal in length and order.
func hasBenchmarkWorkerTagSetMatch(parseLeftTags []string, parseRightTags []string) bool {
	if len(parseLeftTags) != len(parseRightTags) {
		return false
	}
	for parseTagIndex := 0; parseTagIndex < len(parseLeftTags); parseTagIndex++ {
		if parseLeftTags[parseTagIndex] != parseRightTags[parseTagIndex] {
			return false
		}
	}
	return true
}

// clearBenchmarkWorkerCoreChunkCacheByDependency removes one cached core chunk entry for one dependency hash.
func clearBenchmarkWorkerCoreChunkCacheByDependency(parseCacheByDependencyRef ui.Ref[map[uint64][]benchmarkshared.BenchmarkWorkerCoreChunkResult], parseDependency uint64) {
	getCacheByDependency := parseCacheByDependencyRef.Get()
	if len(getCacheByDependency) == 0 {
		return
	}
	delete(getCacheByDependency, parseDependency)
	parseCacheByDependencyRef.Set(getCacheByDependency)
}

// clearBenchmarkWorkerContentChunkCacheByDependency removes one cached content chunk entry for one dependency hash.
func clearBenchmarkWorkerContentChunkCacheByDependency(parseCacheByDependencyRef ui.Ref[map[uint64][]benchmarkshared.BenchmarkWorkerContentChunkResult], parseDependency uint64) {
	getCacheByDependency := parseCacheByDependencyRef.Get()
	if len(getCacheByDependency) == 0 {
		return
	}
	delete(getCacheByDependency, parseDependency)
	parseCacheByDependencyRef.Set(getCacheByDependency)
}

// storeBenchmarkWorkerCoreChunkCache stores one core chunk result set under one dependency hash.
func storeBenchmarkWorkerCoreChunkCache(parseCacheByDependencyRef ui.Ref[map[uint64][]benchmarkshared.BenchmarkWorkerCoreChunkResult], parseDependency uint64, parseChunks []benchmarkshared.BenchmarkWorkerCoreChunkResult) {
	if len(parseChunks) == 0 {
		return
	}
	getCacheByDependency := parseCacheByDependencyRef.Get()
	if getCacheByDependency == nil {
		getCacheByDependency = map[uint64][]benchmarkshared.BenchmarkWorkerCoreChunkResult{}
	}
	getCacheByDependency[parseDependency] = cloneBenchmarkWorkerCoreChunks(parseChunks)
	parseCacheByDependencyRef.Set(getCacheByDependency)
}

// storeBenchmarkWorkerContentChunkCache stores one content chunk result set under one dependency hash.
func storeBenchmarkWorkerContentChunkCache(parseCacheByDependencyRef ui.Ref[map[uint64][]benchmarkshared.BenchmarkWorkerContentChunkResult], parseDependency uint64, parseChunks []benchmarkshared.BenchmarkWorkerContentChunkResult) {
	if len(parseChunks) == 0 {
		return
	}
	getCacheByDependency := parseCacheByDependencyRef.Get()
	if getCacheByDependency == nil {
		getCacheByDependency = map[uint64][]benchmarkshared.BenchmarkWorkerContentChunkResult{}
	}
	getCacheByDependency[parseDependency] = cloneBenchmarkWorkerContentChunks(parseChunks)
	parseCacheByDependencyRef.Set(getCacheByDependency)
}

// setBenchmarkWorkerRequestError stores the first request error observed across concurrent lane requests.
func setBenchmarkWorkerRequestError(parseErrorMu *sync.Mutex, parseErrorRef *error, parseError error) {
	if parseError == nil {
		return
	}
	parseErrorMu.Lock()
	if *parseErrorRef == nil {
		*parseErrorRef = parseError
	}
	parseErrorMu.Unlock()
}

// buildBenchmarkWorkerCoreItemDirtyIndexes compares two equal-length core item slices and returns
// the list of indexes where ID or text differ. Returns nil when lengths differ (signals full recompute).
func buildBenchmarkWorkerCoreItemDirtyIndexes(parsePreviousItems, parseCurrentItems []benchmarkshared.BenchmarkCoreRowData) []int {
	if len(parsePreviousItems) != len(parseCurrentItems) {
		return nil
	}
	getDirtyIndexes := make([]int, 0)
	for getIndex := range parseCurrentItems {
		if parsePreviousItems[getIndex].GetID != parseCurrentItems[getIndex].GetID ||
			parsePreviousItems[getIndex].GetText != parseCurrentItems[getIndex].GetText {
			getDirtyIndexes = append(getDirtyIndexes, getIndex)
		}
	}
	return getDirtyIndexes
}

// buildBenchmarkWorkerContentItemDirtyIndexes compares two equal-length content item slices and returns
// the list of indexes where any field differs. Returns nil when lengths differ (signals full recompute).
func buildBenchmarkWorkerContentItemDirtyIndexes(parsePreviousItems, parseCurrentItems []benchmarkshared.BenchmarkContentCardData) []int {
	if len(parsePreviousItems) != len(parseCurrentItems) {
		return nil
	}
	getDirtyIndexes := make([]int, 0)
	for getIndex := range parseCurrentItems {
		getPrev := parsePreviousItems[getIndex]
		getCurr := parseCurrentItems[getIndex]
		if getPrev.GetID != getCurr.GetID ||
			getPrev.GetTitle != getCurr.GetTitle ||
			getPrev.GetSummary != getCurr.GetSummary ||
			getPrev.GetStatus != getCurr.GetStatus ||
			getPrev.GetMeta != getCurr.GetMeta {
			getDirtyIndexes = append(getDirtyIndexes, getIndex)
			continue
		}
		if len(getPrev.GetTags) != len(getCurr.GetTags) {
			getDirtyIndexes = append(getDirtyIndexes, getIndex)
			continue
		}
		for getTagIndex := range getCurr.GetTags {
			if getPrev.GetTags[getTagIndex] != getCurr.GetTags[getTagIndex] {
				getDirtyIndexes = append(getDirtyIndexes, getIndex)
				break
			}
		}
	}
	return getDirtyIndexes
}

// buildBenchmarkWorkerChunkLocalDirtyIndexes filters one global dirty index list to chunk-local indexes within [parseStart, parseEnd).
func buildBenchmarkWorkerChunkLocalDirtyIndexes(parseDirtyItemIndexes []int, parseStart, parseEnd int) []int {
	if len(parseDirtyItemIndexes) == 0 {
		return nil
	}
	getLocalIndexes := make([]int, 0)
	for _, getDirtyIndex := range parseDirtyItemIndexes {
		if getDirtyIndex >= parseStart && getDirtyIndex < parseEnd {
			getLocalIndexes = append(getLocalIndexes, getDirtyIndex-parseStart)
		}
	}
	return getLocalIndexes
}

// requestBenchmarkWorkerCoreChunks fans out one adaptive core-list preparation batch across one benchmark worker fleet.
func requestBenchmarkWorkerCoreChunks(parseCtx context.Context, parseWorkers []interop.Worker, parseMode string, parseItems []benchmarkshared.BenchmarkCoreRowData, parseWorkerState buildBenchmarkWorkerState, parseGeneration uint64, parseDirtyItemIndexes []int, onPartialChunks func([]benchmarkshared.BenchmarkWorkerCoreChunkResult)) (buildBenchmarkWorkerCoreBatchReport, error) {
	getReport := buildBenchmarkWorkerCoreBatchReport{}
	if len(parseWorkers) < 1 {
		return getReport, fmt.Errorf("runtime2 benchmark worker fleet is empty for mode %s", parseMode)
	}
	getWorkScale := buildBenchmarkWorkerWorkScale()
	if hasBenchmarkWorkerCoreFastPath(parseMode, len(parseItems)) {
		warnBenchmarkWorkerCoreFastPath()
		getChunks, getCacheHitCount := requestBenchmarkWorkerCoreChunksByLocalCache(parseItems, getWorkScale, parseGeneration)
		getReport.GetChunks = getChunks
		getReport.GetAdaptiveChunkCount = len(getChunks)
		getReport.GetCacheHitCount = getCacheHitCount
		return getReport, nil
	}
	getAdaptiveChunkCount := buildBenchmarkWorkerAutoChunkCount(parseMode, parseWorkerState, len(parseItems))
	getItemWeights := buildBenchmarkWorkerCoreItemWeights(parseItems)
	getChunkBounds := buildBenchmarkWorkerAdaptiveChunkBoundsFromWeights(getItemWeights, getAdaptiveChunkCount)
	getChunkPlans := buildBenchmarkWorkerChunkPlans(getChunkBounds, getItemWeights)
	if len(getChunkPlans) < 1 {
		return getReport, nil
	}
	getDispatchMode := buildBenchmarkWorkerDispatchMode()
	getCacheHitCount := 0
	var getChunks []benchmarkshared.BenchmarkWorkerCoreChunkResult
	var parseErr error
	if getDispatchMode == getBenchmarkWorkerDispatchChunk {
		warnBenchmarkWorkerChunkDispatch()
		getChunks, parseErr = requestBenchmarkWorkerCoreChunksByChunk(parseCtx, parseWorkers, parseItems, getChunkPlans, getWorkScale, parseGeneration, onPartialChunks)
	} else {
		getChunks, getCacheHitCount, parseErr = requestBenchmarkWorkerCoreChunksByBatch(parseCtx, parseWorkers, parseItems, getChunkPlans, getWorkScale, parseGeneration, parseDirtyItemIndexes, onPartialChunks)
	}
	if parseErr != nil {
		return getReport, parseErr
	}
	getReport.GetChunks = getChunks
	getReport.GetAdaptiveChunkCount = len(getChunkPlans)
	getReport.GetCacheHitCount = getCacheHitCount
	return getReport, nil
}

// buildBenchmarkWorkerCoreFastPathCacheKey builds one stable local fast-path cache key from row ID, scale, and text.
func buildBenchmarkWorkerCoreFastPathCacheKey(parseItem benchmarkshared.BenchmarkCoreRowData, parseWorkScale int) string {
	return strconv.Itoa(parseWorkScale) + "|" + strconv.Itoa(parseItem.GetID) + "|" + parseItem.GetText
}

// clearBenchmarkWorkerCoreFastPathCache clears local cache-backed prepared core items when the cache reaches one hard cap.
func clearBenchmarkWorkerCoreFastPathCache() {
	getBenchmarkWorkerCoreFastPathCacheMu.Lock()
	getCachedEntryCount := len(getBenchmarkWorkerCoreFastPathCacheByKey)
	getBenchmarkWorkerCoreFastPathCacheByKey = map[string]benchmarkshared.BenchmarkPreparedCoreItem{}
	getBenchmarkWorkerCoreFastPathCacheMu.Unlock()
	fmt.Printf("[render-benchmark/runtime2][warn] cleared local core fast-path cache entries=%d limit=%d\n", getCachedEntryCount, getBenchmarkWorkerCoreFastPathCacheMax)
}

// requestBenchmarkWorkerCoreChunksByLocalCache prepares one core batch on the main runtime with one reusable local prepared-item cache.
func requestBenchmarkWorkerCoreChunksByLocalCache(parseItems []benchmarkshared.BenchmarkCoreRowData, parseWorkScale int, parseGeneration uint64) ([]benchmarkshared.BenchmarkWorkerCoreChunkResult, int) {
	parseStartedAt := time.Now()
	getPreparedItems := make([]benchmarkshared.BenchmarkPreparedCoreItem, len(parseItems))
	getCacheHitCount := 0
	var getWorkDigest uint64
	hasCacheOverLimit := false
	for parseItemIndex, parseItem := range parseItems {
		getCacheKey := buildBenchmarkWorkerCoreFastPathCacheKey(parseItem, parseWorkScale)
		getBenchmarkWorkerCoreFastPathCacheMu.RLock()
		getCachedItem, hasCachedItem := getBenchmarkWorkerCoreFastPathCacheByKey[getCacheKey]
		getBenchmarkWorkerCoreFastPathCacheMu.RUnlock()
		if hasCachedItem {
			getPreparedItems[parseItemIndex] = getCachedItem
			getWorkDigest ^= getCachedItem.GetDigest
			getCacheHitCount++
			continue
		}
		getPreparedItem := benchmarkshared.BuildBenchmarkPreparedCoreItem(parseItem, parseItemIndex, parseWorkScale)
		getPreparedItems[parseItemIndex] = getPreparedItem
		getWorkDigest ^= getPreparedItem.GetDigest
		getBenchmarkWorkerCoreFastPathCacheMu.Lock()
		getBenchmarkWorkerCoreFastPathCacheByKey[getCacheKey] = getPreparedItem
		if len(getBenchmarkWorkerCoreFastPathCacheByKey) > getBenchmarkWorkerCoreFastPathCacheMax {
			hasCacheOverLimit = true
		}
		getBenchmarkWorkerCoreFastPathCacheMu.Unlock()
	}
	if hasCacheOverLimit {
		clearBenchmarkWorkerCoreFastPathCache()
	}
	return []benchmarkshared.BenchmarkWorkerCoreChunkResult{
		{
			GetChunkIndex:     0,
			GetItems:          getPreparedItems,
			GetWorker:         getBenchmarkWorkerCoreFastPath,
			GetWorkDigest:     getWorkDigest,
			GetWorkDurationMS: time.Since(parseStartedAt).Milliseconds(),
			GetGeneration:     parseGeneration,
		},
	}, getCacheHitCount
}

// requestBenchmarkWorkerCoreChunksByBatch sends one multi-chunk core preparation request per worker lane.
func requestBenchmarkWorkerCoreChunksByBatch(parseCtx context.Context, parseWorkers []interop.Worker, parseItems []benchmarkshared.BenchmarkCoreRowData, parseChunkPlans []buildBenchmarkWorkerChunkPlan, parseWorkScale int, parseGeneration uint64, parseDirtyItemIndexes []int, onPartialChunks func([]benchmarkshared.BenchmarkWorkerCoreChunkResult)) ([]benchmarkshared.BenchmarkWorkerCoreChunkResult, int, error) {
	getLanePlans := buildBenchmarkWorkerLanePlans(len(parseWorkers), parseChunkPlans)
	getResults := make([]benchmarkshared.BenchmarkWorkerCoreChunkResult, len(parseChunkPlans))
	hasResultByChunkIndex := make([]bool, len(parseChunkPlans))
	getCacheHitCount := 0
	var getResultErr error
	var getResultErrMu sync.Mutex
	var getResultMu sync.Mutex
	var getWait sync.WaitGroup
	for _, getLanePlan := range getLanePlans {
		if len(getLanePlan.GetChunkPlans) < 1 {
			continue
		}
		getWait.Add(1)
		go func(parseLanePlan buildBenchmarkWorkerLanePlan) {
			defer getWait.Done()
			getBatchChunks := make([]benchmarkshared.BenchmarkWorkerCoreBatchChunkRequest, len(parseLanePlan.GetChunkPlans))
			for parseBatchChunkIndex, getChunkPlan := range parseLanePlan.GetChunkPlans {
				getBatchChunks[parseBatchChunkIndex] = benchmarkshared.BenchmarkWorkerCoreBatchChunkRequest{
					GetChunkIndex:       getChunkPlan.GetChunkIndex,
					GetItems:            parseItems[getChunkPlan.GetStart:getChunkPlan.GetEnd],
					GetDirtyItemIndexes: buildBenchmarkWorkerChunkLocalDirtyIndexes(parseDirtyItemIndexes, getChunkPlan.GetStart, getChunkPlan.GetEnd),
				}
			}
			getWorker := parseWorkers[parseLanePlan.GetWorkerIndex%len(parseWorkers)]
			getBatchResult, parseErr := interop.RequestWorkerDecoded[benchmarkshared.BenchmarkWorkerCoreBatchRequest, struct{}, benchmarkshared.BenchmarkWorkerCoreBatchResult](
				parseCtx,
				getWorker,
				benchmarkshared.BenchmarkWorkerRequestCoreBatch,
				benchmarkshared.BenchmarkWorkerCoreBatchRequest{
					GetChunks:     getBatchChunks,
					GetWorkScale:  parseWorkScale,
					GetGeneration: parseGeneration,
				},
				nil,
			)
			if parseErr != nil {
				setBenchmarkWorkerRequestError(&getResultErrMu, &getResultErr, parseErr)
				return
			}
			if getBatchResult.HasStale || getBatchResult.GetGeneration != parseGeneration {
				setBenchmarkWorkerRequestError(
					&getResultErrMu,
					&getResultErr,
					fmt.Errorf("runtime2 core worker batch stale or generation mismatch worker=%s result-generation=%d expected=%d", strings.TrimSpace(getBatchResult.GetWorker), getBatchResult.GetGeneration, parseGeneration),
				)
				return
			}
			getResultMu.Lock()
			getCacheHitCount += getBatchResult.GetCacheHitCount
			getResultMu.Unlock()
			for _, getChunkResult := range getBatchResult.GetChunks {
				if getChunkResult.HasStale || getChunkResult.GetGeneration != parseGeneration {
					setBenchmarkWorkerRequestError(
						&getResultErrMu,
						&getResultErr,
						fmt.Errorf("runtime2 core worker chunk stale or generation mismatch chunk=%d worker=%s result-generation=%d expected=%d", getChunkResult.GetChunkIndex, strings.TrimSpace(getChunkResult.GetWorker), getChunkResult.GetGeneration, parseGeneration),
					)
					return
				}
				getChunkIndex := getChunkResult.GetChunkIndex
				if getChunkIndex < 0 || getChunkIndex >= len(getResults) {
					setBenchmarkWorkerRequestError(&getResultErrMu, &getResultErr, fmt.Errorf("runtime2 core worker chunk index out of range index=%d chunks=%d", getChunkIndex, len(getResults)))
					return
				}
				getResultMu.Lock()
				if hasResultByChunkIndex[getChunkIndex] {
					getResultMu.Unlock()
					setBenchmarkWorkerRequestError(&getResultErrMu, &getResultErr, fmt.Errorf("runtime2 core worker duplicate chunk index=%d", getChunkIndex))
					return
				}
				getResults[getChunkIndex] = getChunkResult
				hasResultByChunkIndex[getChunkIndex] = true
				getResultMu.Unlock()
			}
			// progressive partial delivery: snapshot results after all lane chunks are filled
			if onPartialChunks != nil {
				getResultMu.Lock()
				getPartialSnapshot := make([]benchmarkshared.BenchmarkWorkerCoreChunkResult, len(getResults))
				copy(getPartialSnapshot, getResults)
				getResultMu.Unlock()
				onPartialChunks(getPartialSnapshot)
			}
		}(getLanePlan)
	}
	getWait.Wait()
	if getResultErr != nil {
		return nil, 0, getResultErr
	}
	for parseChunkIndex := 0; parseChunkIndex < len(hasResultByChunkIndex); parseChunkIndex++ {
		if !hasResultByChunkIndex[parseChunkIndex] {
			return nil, 0, fmt.Errorf("runtime2 core worker missing chunk index=%d", parseChunkIndex)
		}
	}
	return getResults, getCacheHitCount, nil
}

// requestBenchmarkWorkerCoreChunksByChunk sends one legacy per-chunk core preparation request.
func requestBenchmarkWorkerCoreChunksByChunk(parseCtx context.Context, parseWorkers []interop.Worker, parseItems []benchmarkshared.BenchmarkCoreRowData, parseChunkPlans []buildBenchmarkWorkerChunkPlan, parseWorkScale int, parseGeneration uint64, onPartialChunks func([]benchmarkshared.BenchmarkWorkerCoreChunkResult)) ([]benchmarkshared.BenchmarkWorkerCoreChunkResult, error) {
	getResults := make([]benchmarkshared.BenchmarkWorkerCoreChunkResult, len(parseChunkPlans))
	hasResultByChunkIndex := make([]bool, len(parseChunkPlans))
	var getResultErr error
	var getResultErrMu sync.Mutex
	var getResultMu sync.Mutex
	var getWait sync.WaitGroup
	for _, getChunkPlan := range parseChunkPlans {
		getWait.Add(1)
		go func(parseChunkPlan buildBenchmarkWorkerChunkPlan) {
			defer getWait.Done()
			getWorker := parseWorkers[parseChunkPlan.GetChunkIndex%len(parseWorkers)]
			getChunkResult, parseErr := interop.RequestWorkerDecoded[benchmarkshared.BenchmarkWorkerCoreChunkRequest, struct{}, benchmarkshared.BenchmarkWorkerCoreChunkResult](
				parseCtx,
				getWorker,
				benchmarkshared.BenchmarkWorkerRequestCoreChunk,
				benchmarkshared.BenchmarkWorkerCoreChunkRequest{
					GetChunkIndex: parseChunkPlan.GetChunkIndex,
					GetItems:      parseItems[parseChunkPlan.GetStart:parseChunkPlan.GetEnd],
					GetWorkScale:  parseWorkScale,
					GetGeneration: parseGeneration,
				},
				nil,
			)
			if parseErr != nil {
				setBenchmarkWorkerRequestError(&getResultErrMu, &getResultErr, parseErr)
				return
			}
			if getChunkResult.HasStale || getChunkResult.GetGeneration != parseGeneration {
				setBenchmarkWorkerRequestError(
					&getResultErrMu,
					&getResultErr,
					fmt.Errorf("runtime2 core worker chunk stale or generation mismatch chunk=%d worker=%s result-generation=%d expected=%d", getChunkResult.GetChunkIndex, strings.TrimSpace(getChunkResult.GetWorker), getChunkResult.GetGeneration, parseGeneration),
				)
				return
			}
			getChunkIndex := getChunkResult.GetChunkIndex
			if getChunkIndex < 0 || getChunkIndex >= len(getResults) {
				setBenchmarkWorkerRequestError(&getResultErrMu, &getResultErr, fmt.Errorf("runtime2 core worker chunk index out of range index=%d chunks=%d", getChunkIndex, len(getResults)))
				return
			}
			getResultMu.Lock()
			if hasResultByChunkIndex[getChunkIndex] {
				getResultMu.Unlock()
				setBenchmarkWorkerRequestError(&getResultErrMu, &getResultErr, fmt.Errorf("runtime2 core worker duplicate chunk index=%d", getChunkIndex))
				return
			}
			getResults[getChunkIndex] = getChunkResult
			hasResultByChunkIndex[getChunkIndex] = true
			getResultMu.Unlock()
			// progressive partial delivery: snapshot result after each chunk lands
			if onPartialChunks != nil {
				getResultMu.Lock()
				getPartialSnapshot := make([]benchmarkshared.BenchmarkWorkerCoreChunkResult, len(getResults))
				copy(getPartialSnapshot, getResults)
				getResultMu.Unlock()
				onPartialChunks(getPartialSnapshot)
			}
		}(getChunkPlan)
	}
	getWait.Wait()
	if getResultErr != nil {
		return nil, getResultErr
	}
	for parseChunkIndex := 0; parseChunkIndex < len(hasResultByChunkIndex); parseChunkIndex++ {
		if !hasResultByChunkIndex[parseChunkIndex] {
			return nil, fmt.Errorf("runtime2 core worker missing chunk index=%d", parseChunkIndex)
		}
	}
	return getResults, nil
}

// requestBenchmarkWorkerContentChunks fans out one adaptive content-card preparation batch across one benchmark worker fleet.
func requestBenchmarkWorkerContentChunks(parseCtx context.Context, parseWorkers []interop.Worker, parseMode string, parseItems []benchmarkshared.BenchmarkContentCardData, parseWorkerState buildBenchmarkWorkerState, parseGeneration uint64, parseDirtyItemIndexes []int, onPartialChunks func([]benchmarkshared.BenchmarkWorkerContentChunkResult)) (buildBenchmarkWorkerContentBatchReport, error) {
	getReport := buildBenchmarkWorkerContentBatchReport{}
	if len(parseWorkers) < 1 {
		return getReport, fmt.Errorf("runtime2 benchmark worker fleet is empty for mode %s", parseMode)
	}
	getAdaptiveChunkCount := buildBenchmarkWorkerAutoChunkCount(parseMode, parseWorkerState, len(parseItems))
	getItemWeights := buildBenchmarkWorkerContentItemWeights(parseItems)
	getChunkBounds := buildBenchmarkWorkerAdaptiveChunkBoundsFromWeights(getItemWeights, getAdaptiveChunkCount)
	getChunkPlans := buildBenchmarkWorkerChunkPlans(getChunkBounds, getItemWeights)
	if len(getChunkPlans) < 1 {
		return getReport, nil
	}
	getWorkScale := buildBenchmarkWorkerWorkScale()
	getDispatchMode := buildBenchmarkWorkerDispatchMode()
	getCacheHitCount := 0
	var getChunks []benchmarkshared.BenchmarkWorkerContentChunkResult
	var parseErr error
	if getDispatchMode == getBenchmarkWorkerDispatchChunk {
		warnBenchmarkWorkerChunkDispatch()
		getChunks, parseErr = requestBenchmarkWorkerContentChunksByChunk(parseCtx, parseWorkers, parseItems, getChunkPlans, getWorkScale, parseGeneration, onPartialChunks)
	} else {
		getChunks, getCacheHitCount, parseErr = requestBenchmarkWorkerContentChunksByBatch(parseCtx, parseWorkers, parseItems, getChunkPlans, getWorkScale, parseGeneration, parseDirtyItemIndexes, onPartialChunks)
	}
	if parseErr != nil {
		return getReport, parseErr
	}
	getReport.GetChunks = getChunks
	getReport.GetAdaptiveChunkCount = len(getChunkPlans)
	getReport.GetCacheHitCount = getCacheHitCount
	return getReport, nil
}

// requestBenchmarkWorkerContentChunksByBatch sends one multi-chunk content preparation request per worker lane.
func requestBenchmarkWorkerContentChunksByBatch(parseCtx context.Context, parseWorkers []interop.Worker, parseItems []benchmarkshared.BenchmarkContentCardData, parseChunkPlans []buildBenchmarkWorkerChunkPlan, parseWorkScale int, parseGeneration uint64, parseDirtyItemIndexes []int, onPartialChunks func([]benchmarkshared.BenchmarkWorkerContentChunkResult)) ([]benchmarkshared.BenchmarkWorkerContentChunkResult, int, error) {
	getLanePlans := buildBenchmarkWorkerLanePlans(len(parseWorkers), parseChunkPlans)
	getResults := make([]benchmarkshared.BenchmarkWorkerContentChunkResult, len(parseChunkPlans))
	hasResultByChunkIndex := make([]bool, len(parseChunkPlans))
	getCacheHitCount := 0
	var getResultErr error
	var getResultErrMu sync.Mutex
	var getResultMu sync.Mutex
	var getWait sync.WaitGroup
	for _, getLanePlan := range getLanePlans {
		if len(getLanePlan.GetChunkPlans) < 1 {
			continue
		}
		getWait.Add(1)
		go func(parseLanePlan buildBenchmarkWorkerLanePlan) {
			defer getWait.Done()
			getBatchChunks := make([]benchmarkshared.BenchmarkWorkerContentBatchChunkRequest, len(parseLanePlan.GetChunkPlans))
			for parseBatchChunkIndex, getChunkPlan := range parseLanePlan.GetChunkPlans {
				getBatchChunks[parseBatchChunkIndex] = benchmarkshared.BenchmarkWorkerContentBatchChunkRequest{
					GetChunkIndex:       getChunkPlan.GetChunkIndex,
					GetItems:            parseItems[getChunkPlan.GetStart:getChunkPlan.GetEnd],
					GetDirtyItemIndexes: buildBenchmarkWorkerChunkLocalDirtyIndexes(parseDirtyItemIndexes, getChunkPlan.GetStart, getChunkPlan.GetEnd),
				}
			}
			getWorker := parseWorkers[parseLanePlan.GetWorkerIndex%len(parseWorkers)]
			getBatchResult, parseErr := interop.RequestWorkerDecoded[benchmarkshared.BenchmarkWorkerContentBatchRequest, struct{}, benchmarkshared.BenchmarkWorkerContentBatchResult](
				parseCtx,
				getWorker,
				benchmarkshared.BenchmarkWorkerRequestContentBatch,
				benchmarkshared.BenchmarkWorkerContentBatchRequest{
					GetChunks:     getBatchChunks,
					GetWorkScale:  parseWorkScale,
					GetGeneration: parseGeneration,
				},
				nil,
			)
			if parseErr != nil {
				setBenchmarkWorkerRequestError(&getResultErrMu, &getResultErr, parseErr)
				return
			}
			if getBatchResult.HasStale || getBatchResult.GetGeneration != parseGeneration {
				setBenchmarkWorkerRequestError(
					&getResultErrMu,
					&getResultErr,
					fmt.Errorf("runtime2 content worker batch stale or generation mismatch worker=%s result-generation=%d expected=%d", strings.TrimSpace(getBatchResult.GetWorker), getBatchResult.GetGeneration, parseGeneration),
				)
				return
			}
			getResultMu.Lock()
			getCacheHitCount += getBatchResult.GetCacheHitCount
			getResultMu.Unlock()
			for _, getChunkResult := range getBatchResult.GetChunks {
				if getChunkResult.HasStale || getChunkResult.GetGeneration != parseGeneration {
					setBenchmarkWorkerRequestError(
						&getResultErrMu,
						&getResultErr,
						fmt.Errorf("runtime2 content worker chunk stale or generation mismatch chunk=%d worker=%s result-generation=%d expected=%d", getChunkResult.GetChunkIndex, strings.TrimSpace(getChunkResult.GetWorker), getChunkResult.GetGeneration, parseGeneration),
					)
					return
				}
				getChunkIndex := getChunkResult.GetChunkIndex
				if getChunkIndex < 0 || getChunkIndex >= len(getResults) {
					setBenchmarkWorkerRequestError(&getResultErrMu, &getResultErr, fmt.Errorf("runtime2 content worker chunk index out of range index=%d chunks=%d", getChunkIndex, len(getResults)))
					return
				}
				getResultMu.Lock()
				if hasResultByChunkIndex[getChunkIndex] {
					getResultMu.Unlock()
					setBenchmarkWorkerRequestError(&getResultErrMu, &getResultErr, fmt.Errorf("runtime2 content worker duplicate chunk index=%d", getChunkIndex))
					return
				}
				getResults[getChunkIndex] = getChunkResult
				hasResultByChunkIndex[getChunkIndex] = true
				getResultMu.Unlock()
			}
			// progressive partial delivery: snapshot results after all lane chunks are filled
			if onPartialChunks != nil {
				getResultMu.Lock()
				getPartialSnapshot := make([]benchmarkshared.BenchmarkWorkerContentChunkResult, len(getResults))
				copy(getPartialSnapshot, getResults)
				getResultMu.Unlock()
				onPartialChunks(getPartialSnapshot)
			}
		}(getLanePlan)
	}
	getWait.Wait()
	if getResultErr != nil {
		return nil, 0, getResultErr
	}
	for parseChunkIndex := 0; parseChunkIndex < len(hasResultByChunkIndex); parseChunkIndex++ {
		if !hasResultByChunkIndex[parseChunkIndex] {
			return nil, 0, fmt.Errorf("runtime2 content worker missing chunk index=%d", parseChunkIndex)
		}
	}
	return getResults, getCacheHitCount, nil
}

// requestBenchmarkWorkerContentChunksByChunk sends one legacy per-chunk content preparation request.
func requestBenchmarkWorkerContentChunksByChunk(parseCtx context.Context, parseWorkers []interop.Worker, parseItems []benchmarkshared.BenchmarkContentCardData, parseChunkPlans []buildBenchmarkWorkerChunkPlan, parseWorkScale int, parseGeneration uint64, onPartialChunks func([]benchmarkshared.BenchmarkWorkerContentChunkResult)) ([]benchmarkshared.BenchmarkWorkerContentChunkResult, error) {
	getResults := make([]benchmarkshared.BenchmarkWorkerContentChunkResult, len(parseChunkPlans))
	hasResultByChunkIndex := make([]bool, len(parseChunkPlans))
	var getResultErr error
	var getResultErrMu sync.Mutex
	var getResultMu sync.Mutex
	var getWait sync.WaitGroup
	for _, getChunkPlan := range parseChunkPlans {
		getWait.Add(1)
		go func(parseChunkPlan buildBenchmarkWorkerChunkPlan) {
			defer getWait.Done()
			getWorker := parseWorkers[parseChunkPlan.GetChunkIndex%len(parseWorkers)]
			getChunkResult, parseErr := interop.RequestWorkerDecoded[benchmarkshared.BenchmarkWorkerContentChunkRequest, struct{}, benchmarkshared.BenchmarkWorkerContentChunkResult](
				parseCtx,
				getWorker,
				benchmarkshared.BenchmarkWorkerRequestContentChunk,
				benchmarkshared.BenchmarkWorkerContentChunkRequest{
					GetChunkIndex: parseChunkPlan.GetChunkIndex,
					GetItems:      parseItems[parseChunkPlan.GetStart:parseChunkPlan.GetEnd],
					GetWorkScale:  parseWorkScale,
					GetGeneration: parseGeneration,
				},
				nil,
			)
			if parseErr != nil {
				setBenchmarkWorkerRequestError(&getResultErrMu, &getResultErr, parseErr)
				return
			}
			if getChunkResult.HasStale || getChunkResult.GetGeneration != parseGeneration {
				setBenchmarkWorkerRequestError(
					&getResultErrMu,
					&getResultErr,
					fmt.Errorf("runtime2 content worker chunk stale or generation mismatch chunk=%d worker=%s result-generation=%d expected=%d", getChunkResult.GetChunkIndex, strings.TrimSpace(getChunkResult.GetWorker), getChunkResult.GetGeneration, parseGeneration),
				)
				return
			}
			getChunkIndex := getChunkResult.GetChunkIndex
			if getChunkIndex < 0 || getChunkIndex >= len(getResults) {
				setBenchmarkWorkerRequestError(&getResultErrMu, &getResultErr, fmt.Errorf("runtime2 content worker chunk index out of range index=%d chunks=%d", getChunkIndex, len(getResults)))
				return
			}
			getResultMu.Lock()
			if hasResultByChunkIndex[getChunkIndex] {
				getResultMu.Unlock()
				setBenchmarkWorkerRequestError(&getResultErrMu, &getResultErr, fmt.Errorf("runtime2 content worker duplicate chunk index=%d", getChunkIndex))
				return
			}
			getResults[getChunkIndex] = getChunkResult
			hasResultByChunkIndex[getChunkIndex] = true
			getResultMu.Unlock()
			// progressive partial delivery: snapshot result after each chunk lands
			if onPartialChunks != nil {
				getResultMu.Lock()
				getPartialSnapshot := make([]benchmarkshared.BenchmarkWorkerContentChunkResult, len(getResults))
				copy(getPartialSnapshot, getResults)
				getResultMu.Unlock()
				onPartialChunks(getPartialSnapshot)
			}
		}(getChunkPlan)
	}
	getWait.Wait()
	if getResultErr != nil {
		return nil, getResultErr
	}
	for parseChunkIndex := 0; parseChunkIndex < len(hasResultByChunkIndex); parseChunkIndex++ {
		if !hasResultByChunkIndex[parseChunkIndex] {
			return nil, fmt.Errorf("runtime2 content worker missing chunk index=%d", parseChunkIndex)
		}
	}
	return getResults, nil
}
