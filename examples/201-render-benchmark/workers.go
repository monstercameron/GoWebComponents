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
	getBenchmarkWorkerDefaultScale   = 3
	getBenchmarkWorkerDispatchBatch  = "batch"
	getBenchmarkWorkerDispatchChunk  = "chunk"
)

const (
	getBenchmarkWorkerDependencySeed uint64 = 14695981039346656037
	getBenchmarkWorkerDependencyStep uint64 = 1099511628211
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

// buildBenchmarkWorkerCoreItemsDependency hashes core-item payload content into one comparable dependency token.
func buildBenchmarkWorkerCoreItemsDependency(parseItems []benchmarkshared.BenchmarkCoreRowData) uint64 {
	getHash := getBenchmarkWorkerDependencySeed ^ uint64(len(parseItems))
	for parseItemIndex, parseItem := range parseItems {
		getHash ^= uint64(parseItem.GetID) + uint64(parseItemIndex+1)
		getHash *= getBenchmarkWorkerDependencyStep
		for parseRuneOffset := 0; parseRuneOffset < len(parseItem.GetText); parseRuneOffset++ {
			getHash ^= uint64(parseItem.GetText[parseRuneOffset])
			getHash *= getBenchmarkWorkerDependencyStep
		}
	}
	return getHash
}

// buildBenchmarkWorkerContentItemsDependency hashes content-card payload content into one comparable dependency token.
func buildBenchmarkWorkerContentItemsDependency(parseItems []benchmarkshared.BenchmarkContentCardData) uint64 {
	getHash := getBenchmarkWorkerDependencySeed ^ uint64(len(parseItems))
	for parseItemIndex, parseItem := range parseItems {
		getHash ^= uint64(parseItem.GetID) + uint64(parseItemIndex+1)
		getHash *= getBenchmarkWorkerDependencyStep
		for parseRuneOffset := 0; parseRuneOffset < len(parseItem.GetTitle); parseRuneOffset++ {
			getHash ^= uint64(parseItem.GetTitle[parseRuneOffset])
			getHash *= getBenchmarkWorkerDependencyStep
		}
		for parseRuneOffset := 0; parseRuneOffset < len(parseItem.GetSummary); parseRuneOffset++ {
			getHash ^= uint64(parseItem.GetSummary[parseRuneOffset])
			getHash *= getBenchmarkWorkerDependencyStep
		}
		for parseRuneOffset := 0; parseRuneOffset < len(parseItem.GetStatus); parseRuneOffset++ {
			getHash ^= uint64(parseItem.GetStatus[parseRuneOffset])
			getHash *= getBenchmarkWorkerDependencyStep
		}
		for parseRuneOffset := 0; parseRuneOffset < len(parseItem.GetMeta); parseRuneOffset++ {
			getHash ^= uint64(parseItem.GetMeta[parseRuneOffset])
			getHash *= getBenchmarkWorkerDependencyStep
		}
		for _, getTag := range parseItem.GetTags {
			for parseRuneOffset := 0; parseRuneOffset < len(getTag); parseRuneOffset++ {
				getHash ^= uint64(getTag[parseRuneOffset])
				getHash *= getBenchmarkWorkerDependencyStep
			}
		}
	}
	return getHash
}

// warnBenchmarkWorkerChunkDispatch logs one warning when the benchmark is configured to use legacy per-chunk dispatch.
func warnBenchmarkWorkerChunkDispatch() {
	warnBenchmarkWorkerChunkDispatchOnce.Do(func() {
		fmt.Printf("[render-benchmark/runtime2][warn] runtime2Dispatch=chunk enables legacy per-chunk requests; use runtime2Dispatch=batch for lower worker-dispatch overhead\n")
	})
}

// buildBenchmarkWorkerAutoChunkCount resolves one deterministic chunk fanout target from worker count and item count.
func buildBenchmarkWorkerAutoChunkCount(parseMode string, parseWorkerState buildBenchmarkWorkerState, parseItemCount int) int {
	_ = parseWorkerState
	if parseItemCount < 1 {
		return 0
	}
	getWorkerCount := buildBenchmarkRuntime3ChunkCount(parseMode)
	if getWorkerCount < 1 {
		getWorkerCount = 1
	}
	getAdaptiveChunkCount := getWorkerCount
	if parseItemCount >= getWorkerCount*4 {
		getAdaptiveChunkCount = getWorkerCount * 2
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
	parseCoreChunkState ui.State[[]benchmarkshared.BenchmarkWorkerCoreChunkResult],
	parseContentChunkState ui.State[[]benchmarkshared.BenchmarkWorkerContentChunkResult],
	parseWorkerState ui.State[buildBenchmarkWorkerState],
) {
	getCoreItemsDependency := buildBenchmarkWorkerCoreItemsDependency(parseCoreItems)
	getContentItemsDependency := buildBenchmarkWorkerContentItemsDependency(parseContentItems)
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
			parseWorkerState.Update(func(parsePrevious buildBenchmarkWorkerState) buildBenchmarkWorkerState {
				parsePrevious.IsPreparing = false
				parsePrevious.GetErrorText = ""
				return parsePrevious
			})
			return nil
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
				getBatchReport, parseErr := requestBenchmarkWorkerCoreChunks(parseCtx, getWorkers, parseMode, parseCoreItems, parseWorkerSnapshot, parseGeneration)
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
				parseCoreChunkState.Set(getBatchReport.GetChunks)
				parseWorkerState.Update(func(parsePrevious buildBenchmarkWorkerState) buildBenchmarkWorkerState {
					parsePrevious.IsPreparing = false
					parsePrevious.IsReady = true
					parsePrevious.GetPreparedBatchCount++
					parsePrevious.GetPreparedChunks = len(getBatchReport.GetChunks)
					parsePrevious.GetAdaptiveChunkCount = getBatchReport.GetAdaptiveChunkCount
					parsePrevious.GetCacheHitCount = getBatchReport.GetCacheHitCount
					parsePrevious.GetPreparedItems = len(parseCoreItems)
					parsePrevious.GetLastBatchMS = time.Since(parseStartedAt).Milliseconds()
					parsePrevious.GetErrorText = ""
					return parsePrevious
				})
				return
			}

			getBatchReport, parseErr := requestBenchmarkWorkerContentChunks(parseCtx, getWorkers, parseMode, parseContentItems, parseWorkerSnapshot, parseGeneration)
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
			parseContentChunkState.Set(getBatchReport.GetChunks)
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
	}, parseMode, parseView, parsePrepareRevision, parseWorkersRevision, getCoreItemsDependency, getContentItemsDependency)
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

// requestBenchmarkWorkerCoreChunks fans out one adaptive core-list preparation batch across one benchmark worker fleet.
func requestBenchmarkWorkerCoreChunks(parseCtx context.Context, parseWorkers []interop.Worker, parseMode string, parseItems []benchmarkshared.BenchmarkCoreRowData, parseWorkerState buildBenchmarkWorkerState, parseGeneration uint64) (buildBenchmarkWorkerCoreBatchReport, error) {
	getReport := buildBenchmarkWorkerCoreBatchReport{}
	if len(parseWorkers) < 1 {
		return getReport, fmt.Errorf("runtime2 benchmark worker fleet is empty for mode %s", parseMode)
	}
	getAdaptiveChunkCount := buildBenchmarkWorkerAutoChunkCount(parseMode, parseWorkerState, len(parseItems))
	getItemWeights := buildBenchmarkWorkerCoreItemWeights(parseItems)
	getChunkBounds := buildBenchmarkWorkerAdaptiveChunkBoundsFromWeights(getItemWeights, getAdaptiveChunkCount)
	getChunkPlans := buildBenchmarkWorkerChunkPlans(getChunkBounds, getItemWeights)
	if len(getChunkPlans) < 1 {
		return getReport, nil
	}
	getWorkScale := buildBenchmarkWorkerWorkScale()
	getDispatchMode := buildBenchmarkWorkerDispatchMode()
	getCacheHitCount := 0
	var getChunks []benchmarkshared.BenchmarkWorkerCoreChunkResult
	var parseErr error
	if getDispatchMode == getBenchmarkWorkerDispatchChunk {
		warnBenchmarkWorkerChunkDispatch()
		getChunks, parseErr = requestBenchmarkWorkerCoreChunksByChunk(parseCtx, parseWorkers, parseItems, getChunkPlans, getWorkScale, parseGeneration)
	} else {
		getChunks, getCacheHitCount, parseErr = requestBenchmarkWorkerCoreChunksByBatch(parseCtx, parseWorkers, parseItems, getChunkPlans, getWorkScale, parseGeneration)
	}
	if parseErr != nil {
		return getReport, parseErr
	}
	getReport.GetChunks = getChunks
	getReport.GetAdaptiveChunkCount = len(getChunkPlans)
	getReport.GetCacheHitCount = getCacheHitCount
	return getReport, nil
}

// requestBenchmarkWorkerCoreChunksByBatch sends one multi-chunk core preparation request per worker lane.
func requestBenchmarkWorkerCoreChunksByBatch(parseCtx context.Context, parseWorkers []interop.Worker, parseItems []benchmarkshared.BenchmarkCoreRowData, parseChunkPlans []buildBenchmarkWorkerChunkPlan, parseWorkScale int, parseGeneration uint64) ([]benchmarkshared.BenchmarkWorkerCoreChunkResult, int, error) {
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
					GetChunkIndex: getChunkPlan.GetChunkIndex,
					GetItems:      parseItems[getChunkPlan.GetStart:getChunkPlan.GetEnd],
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
func requestBenchmarkWorkerCoreChunksByChunk(parseCtx context.Context, parseWorkers []interop.Worker, parseItems []benchmarkshared.BenchmarkCoreRowData, parseChunkPlans []buildBenchmarkWorkerChunkPlan, parseWorkScale int, parseGeneration uint64) ([]benchmarkshared.BenchmarkWorkerCoreChunkResult, error) {
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
func requestBenchmarkWorkerContentChunks(parseCtx context.Context, parseWorkers []interop.Worker, parseMode string, parseItems []benchmarkshared.BenchmarkContentCardData, parseWorkerState buildBenchmarkWorkerState, parseGeneration uint64) (buildBenchmarkWorkerContentBatchReport, error) {
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
		getChunks, parseErr = requestBenchmarkWorkerContentChunksByChunk(parseCtx, parseWorkers, parseItems, getChunkPlans, getWorkScale, parseGeneration)
	} else {
		getChunks, getCacheHitCount, parseErr = requestBenchmarkWorkerContentChunksByBatch(parseCtx, parseWorkers, parseItems, getChunkPlans, getWorkScale, parseGeneration)
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
func requestBenchmarkWorkerContentChunksByBatch(parseCtx context.Context, parseWorkers []interop.Worker, parseItems []benchmarkshared.BenchmarkContentCardData, parseChunkPlans []buildBenchmarkWorkerChunkPlan, parseWorkScale int, parseGeneration uint64) ([]benchmarkshared.BenchmarkWorkerContentChunkResult, int, error) {
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
					GetChunkIndex: getChunkPlan.GetChunkIndex,
					GetItems:      parseItems[getChunkPlan.GetStart:getChunkPlan.GetEnd],
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
func requestBenchmarkWorkerContentChunksByChunk(parseCtx context.Context, parseWorkers []interop.Worker, parseItems []benchmarkshared.BenchmarkContentCardData, parseChunkPlans []buildBenchmarkWorkerChunkPlan, parseWorkScale int, parseGeneration uint64) ([]benchmarkshared.BenchmarkWorkerContentChunkResult, error) {
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
