//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	benchmarkshared "github.com/monstercameron/GoWebComponents/v4/examples/testing/render-benchmark/shared"
	"github.com/monstercameron/GoWebComponents/v4/interop"
)

const (
	getBenchmarkWorkerFallbackName = "render-benchmark-worker"
	getBenchmarkWorkerCacheLimit   = 8192
)

var getBenchmarkWorkerCacheMu sync.RWMutex
var getBenchmarkWorkerCoreItemCacheByKey = map[string]benchmarkshared.BenchmarkPreparedCoreItem{}
var getBenchmarkWorkerContentItemCacheByKey = map[string]benchmarkshared.BenchmarkPreparedContentCard{}
var getBenchmarkWorkerGenerationMu sync.Mutex
var getBenchmarkWorkerLatestGenerationByRequest = map[string]uint64{}

// getBenchmarkWorkerCoreLatestGeneration tracks the newest core-type request generation for mid-loop abort checks.
var getBenchmarkWorkerCoreLatestGeneration atomic.Uint64

// getBenchmarkWorkerContentLatestGeneration tracks the newest content-type request generation for mid-loop abort checks.
var getBenchmarkWorkerContentLatestGeneration atomic.Uint64

// getBenchmarkWorkerCoreChunkResultByIndex caches the most recent complete core chunk result for each chunk index.
// Used by the dirty-set fast path to apply delta updates without rebuilding clean items.
var getBenchmarkWorkerCoreChunkResultByIndex = map[int]benchmarkshared.BenchmarkWorkerCoreChunkResult{}

// getBenchmarkWorkerContentChunkResultByIndex caches the most recent complete content chunk result for each chunk index.
var getBenchmarkWorkerContentChunkResultByIndex = map[int]benchmarkshared.BenchmarkWorkerContentChunkResult{}

// getBenchmarkWorkerChunkResultMu guards the per-chunk result caches across concurrent goroutines.
var getBenchmarkWorkerChunkResultMu sync.Mutex

// main registers the benchmark worker scope and routes chunk-preparation requests.
func main() {
	getScope, parseErr := interop.GetWorkerScope()
	if parseErr != nil {
		panic(parseErr)
	}
	if _, parseSubscribeErr := getScope.Subscribe(func(parseMessage interop.WorkerMessage, parseMessageErr error) {
		if parseMessageErr != nil {
			return
		}
		go handleBenchmarkWorkerMessage(getScope, parseMessage)
	}); parseSubscribeErr != nil {
		panic(parseSubscribeErr)
	}
	if parseReadyErr := getScope.Ready("bootstrap"); parseReadyErr != nil {
		panic(parseReadyErr)
	}
	fmt.Printf("[render-benchmark/runtime2] worker ready name=%s\n", getBenchmarkWorkerName())
	select {}
}

// handleBenchmarkWorkerMessage routes one request-phase worker message to the typed chunk handlers.
func handleBenchmarkWorkerMessage(parseScope interop.WorkerScope, parseMessage interop.WorkerMessage) {
	if strings.TrimSpace(parseMessage.Phase) != "request" {
		return
	}
	switch strings.TrimSpace(parseMessage.Name) {
	case benchmarkshared.BenchmarkWorkerRequestCoreBatch:
		handleBenchmarkWorkerCoreBatch(parseScope, parseMessage)
	case benchmarkshared.BenchmarkWorkerRequestContentBatch:
		handleBenchmarkWorkerContentBatch(parseScope, parseMessage)
	case benchmarkshared.BenchmarkWorkerRequestCoreChunk:
		handleBenchmarkWorkerCoreChunk(parseScope, parseMessage)
	case benchmarkshared.BenchmarkWorkerRequestContentChunk:
		handleBenchmarkWorkerContentChunk(parseScope, parseMessage)
	default:
		_ = parseScope.Error(parseMessage.ID, parseMessage.Name, "unknown benchmark worker request", map[string]any{
			"worker": getBenchmarkWorkerName(),
		})
	}
}

// handleBenchmarkWorkerCoreBatch decodes and fulfills one core-list multi-chunk batch request.
func handleBenchmarkWorkerCoreBatch(parseScope interop.WorkerScope, parseMessage interop.WorkerMessage) {
	var getRequest benchmarkshared.BenchmarkWorkerCoreBatchRequest
	if parseDecodeErr := interop.Decode(parseMessage.Payload, &getRequest); parseDecodeErr != nil {
		_ = parseScope.Error(parseMessage.ID, benchmarkshared.BenchmarkWorkerRequestCoreBatch, parseDecodeErr.Error(), map[string]any{
			"worker": getBenchmarkWorkerName(),
		})
		return
	}
	getWorkerName := getBenchmarkWorkerName()
	if shouldBenchmarkWorkerDropStaleRequest(benchmarkshared.BenchmarkWorkerRequestCoreBatch, getRequest.GetGeneration) {
		_ = parseScope.Result(parseMessage.ID, benchmarkshared.BenchmarkWorkerRequestCoreBatch, benchmarkshared.BenchmarkWorkerCoreBatchResult{
			GetWorker:     getWorkerName,
			GetGeneration: getRequest.GetGeneration,
			HasStale:      true,
		})
		return
	}
	getResult := buildBenchmarkWorkerCoreBatchWithCache(getWorkerName, getRequest)
	_ = parseScope.Result(parseMessage.ID, benchmarkshared.BenchmarkWorkerRequestCoreBatch, getResult)
}

// handleBenchmarkWorkerContentBatch decodes and fulfills one content-card multi-chunk batch request.
func handleBenchmarkWorkerContentBatch(parseScope interop.WorkerScope, parseMessage interop.WorkerMessage) {
	var getRequest benchmarkshared.BenchmarkWorkerContentBatchRequest
	if parseDecodeErr := interop.Decode(parseMessage.Payload, &getRequest); parseDecodeErr != nil {
		_ = parseScope.Error(parseMessage.ID, benchmarkshared.BenchmarkWorkerRequestContentBatch, parseDecodeErr.Error(), map[string]any{
			"worker": getBenchmarkWorkerName(),
		})
		return
	}
	getWorkerName := getBenchmarkWorkerName()
	if shouldBenchmarkWorkerDropStaleRequest(benchmarkshared.BenchmarkWorkerRequestContentBatch, getRequest.GetGeneration) {
		_ = parseScope.Result(parseMessage.ID, benchmarkshared.BenchmarkWorkerRequestContentBatch, benchmarkshared.BenchmarkWorkerContentBatchResult{
			GetWorker:     getWorkerName,
			GetGeneration: getRequest.GetGeneration,
			HasStale:      true,
		})
		return
	}
	getResult := buildBenchmarkWorkerContentBatchWithCache(getWorkerName, getRequest)
	_ = parseScope.Result(parseMessage.ID, benchmarkshared.BenchmarkWorkerRequestContentBatch, getResult)
}

// handleBenchmarkWorkerCoreChunk decodes and fulfills one core-list chunk request.
func handleBenchmarkWorkerCoreChunk(parseScope interop.WorkerScope, parseMessage interop.WorkerMessage) {
	var getRequest benchmarkshared.BenchmarkWorkerCoreChunkRequest
	if parseDecodeErr := interop.Decode(parseMessage.Payload, &getRequest); parseDecodeErr != nil {
		_ = parseScope.Error(parseMessage.ID, benchmarkshared.BenchmarkWorkerRequestCoreChunk, parseDecodeErr.Error(), map[string]any{
			"worker": getBenchmarkWorkerName(),
		})
		return
	}
	getWorkerName := getBenchmarkWorkerName()
	if shouldBenchmarkWorkerDropStaleRequest(benchmarkshared.BenchmarkWorkerRequestCoreChunk, getRequest.GetGeneration) {
		_ = parseScope.Result(parseMessage.ID, benchmarkshared.BenchmarkWorkerRequestCoreChunk, benchmarkshared.BenchmarkWorkerCoreChunkResult{
			GetChunkIndex: getRequest.GetChunkIndex,
			GetWorker:     getWorkerName,
			GetGeneration: getRequest.GetGeneration,
			HasStale:      true,
		})
		return
	}
	getResult, _ := buildBenchmarkWorkerCoreChunkWithCache(getWorkerName, getRequest.GetChunkIndex, getRequest.GetItems, getRequest.GetWorkScale, getRequest.GetGeneration, nil)
	_ = parseScope.Result(parseMessage.ID, benchmarkshared.BenchmarkWorkerRequestCoreChunk, getResult)
}

// handleBenchmarkWorkerContentChunk decodes and fulfills one content-card chunk request.
func handleBenchmarkWorkerContentChunk(parseScope interop.WorkerScope, parseMessage interop.WorkerMessage) {
	var getRequest benchmarkshared.BenchmarkWorkerContentChunkRequest
	if parseDecodeErr := interop.Decode(parseMessage.Payload, &getRequest); parseDecodeErr != nil {
		_ = parseScope.Error(parseMessage.ID, benchmarkshared.BenchmarkWorkerRequestContentChunk, parseDecodeErr.Error(), map[string]any{
			"worker": getBenchmarkWorkerName(),
		})
		return
	}
	getWorkerName := getBenchmarkWorkerName()
	if shouldBenchmarkWorkerDropStaleRequest(benchmarkshared.BenchmarkWorkerRequestContentChunk, getRequest.GetGeneration) {
		_ = parseScope.Result(parseMessage.ID, benchmarkshared.BenchmarkWorkerRequestContentChunk, benchmarkshared.BenchmarkWorkerContentChunkResult{
			GetChunkIndex: getRequest.GetChunkIndex,
			GetWorker:     getWorkerName,
			GetGeneration: getRequest.GetGeneration,
			HasStale:      true,
		})
		return
	}
	getResult, _ := buildBenchmarkWorkerContentChunkWithCache(getWorkerName, getRequest.GetChunkIndex, getRequest.GetItems, getRequest.GetWorkScale, getRequest.GetGeneration, nil)
	_ = parseScope.Result(parseMessage.ID, benchmarkshared.BenchmarkWorkerRequestContentChunk, getResult)
}

// shouldBenchmarkWorkerDropStaleRequest tracks per-request latest generation and reports whether one incoming request is stale.
func shouldBenchmarkWorkerDropStaleRequest(parseRequestName string, parseGeneration uint64) bool {
	if parseGeneration == 0 {
		return false
	}
	getRequestName := strings.TrimSpace(parseRequestName)
	if getRequestName == "" {
		return false
	}
	getBenchmarkWorkerGenerationMu.Lock()
	defer getBenchmarkWorkerGenerationMu.Unlock()
	getLatestGeneration := getBenchmarkWorkerLatestGenerationByRequest[getRequestName]
	if parseGeneration < getLatestGeneration {
		fmt.Printf("[render-benchmark/runtime2][warn] stale request dropped name=%s generation=%d latest=%d\n", getRequestName, parseGeneration, getLatestGeneration)
		return true
	}
	if parseGeneration > getLatestGeneration {
		getBenchmarkWorkerLatestGenerationByRequest[getRequestName] = parseGeneration
		// update per-type atomic so mid-loop abort checks in chunk builders can short-circuit
		switch getRequestName {
		case benchmarkshared.BenchmarkWorkerRequestCoreBatch, benchmarkshared.BenchmarkWorkerRequestCoreChunk:
			getBenchmarkWorkerCoreLatestGeneration.Store(parseGeneration)
		case benchmarkshared.BenchmarkWorkerRequestContentBatch, benchmarkshared.BenchmarkWorkerRequestContentChunk:
			getBenchmarkWorkerContentLatestGeneration.Store(parseGeneration)
		}
	}
	return false
}

// buildBenchmarkWorkerCoreBatchWithCache builds one cached core-list batch result.
func buildBenchmarkWorkerCoreBatchWithCache(parseWorkerName string, parseRequest benchmarkshared.BenchmarkWorkerCoreBatchRequest) benchmarkshared.BenchmarkWorkerCoreBatchResult {
	parseStartedAt := time.Now()
	getChunkResults := make([]benchmarkshared.BenchmarkWorkerCoreChunkResult, len(parseRequest.GetChunks))
	getCacheHitCount := 0
	var getBatchDigest uint64
	for parseChunkOffset, getChunkRequest := range parseRequest.GetChunks {
		// between-chunk abort: newer core request arrived while processing this batch
		if getBenchmarkWorkerCoreLatestGeneration.Load() > parseRequest.GetGeneration {
			fmt.Printf("[render-benchmark/runtime2][warn] worker core batch aborted between chunks generation=%d\n", parseRequest.GetGeneration)
			return benchmarkshared.BenchmarkWorkerCoreBatchResult{
				GetWorker:     strings.TrimSpace(parseWorkerName),
				GetGeneration: parseRequest.GetGeneration,
				HasStale:      true,
			}
		}
		getChunkResult, getChunkCacheHitCount := buildBenchmarkWorkerCoreChunkWithCache(
			parseWorkerName,
			getChunkRequest.GetChunkIndex,
			getChunkRequest.GetItems,
			parseRequest.GetWorkScale,
			parseRequest.GetGeneration,
			getChunkRequest.GetDirtyItemIndexes,
		)
		if getChunkResult.HasStale {
			// propagate early abort from mid-item stale detection
			return benchmarkshared.BenchmarkWorkerCoreBatchResult{
				GetWorker:     strings.TrimSpace(parseWorkerName),
				GetGeneration: parseRequest.GetGeneration,
				HasStale:      true,
			}
		}
		getChunkResults[parseChunkOffset] = getChunkResult
		getCacheHitCount += getChunkCacheHitCount
		getBatchDigest ^= getChunkResult.GetWorkDigest
	}
	return benchmarkshared.BenchmarkWorkerCoreBatchResult{
		GetChunks:         getChunkResults,
		GetWorker:         strings.TrimSpace(parseWorkerName),
		GetWorkDigest:     getBatchDigest,
		GetWorkDurationMS: time.Since(parseStartedAt).Seconds() * 1000,
		GetGeneration:     parseRequest.GetGeneration,
		GetCacheHitCount:  getCacheHitCount,
	}
}

// buildBenchmarkWorkerContentBatchWithCache builds one cached content-card batch result.
func buildBenchmarkWorkerContentBatchWithCache(parseWorkerName string, parseRequest benchmarkshared.BenchmarkWorkerContentBatchRequest) benchmarkshared.BenchmarkWorkerContentBatchResult {
	parseStartedAt := time.Now()
	getChunkResults := make([]benchmarkshared.BenchmarkWorkerContentChunkResult, len(parseRequest.GetChunks))
	getCacheHitCount := 0
	var getBatchDigest uint64
	for parseChunkOffset, getChunkRequest := range parseRequest.GetChunks {
		// between-chunk abort: newer content request arrived while processing this batch
		if getBenchmarkWorkerContentLatestGeneration.Load() > parseRequest.GetGeneration {
			fmt.Printf("[render-benchmark/runtime2][warn] worker content batch aborted between chunks generation=%d\n", parseRequest.GetGeneration)
			return benchmarkshared.BenchmarkWorkerContentBatchResult{
				GetWorker:     strings.TrimSpace(parseWorkerName),
				GetGeneration: parseRequest.GetGeneration,
				HasStale:      true,
			}
		}
		getChunkResult, getChunkCacheHitCount := buildBenchmarkWorkerContentChunkWithCache(
			parseWorkerName,
			getChunkRequest.GetChunkIndex,
			getChunkRequest.GetItems,
			parseRequest.GetWorkScale,
			parseRequest.GetGeneration,
			getChunkRequest.GetDirtyItemIndexes,
		)
		if getChunkResult.HasStale {
			// propagate early abort from mid-item stale detection
			return benchmarkshared.BenchmarkWorkerContentBatchResult{
				GetWorker:     strings.TrimSpace(parseWorkerName),
				GetGeneration: parseRequest.GetGeneration,
				HasStale:      true,
			}
		}
		getChunkResults[parseChunkOffset] = getChunkResult
		getCacheHitCount += getChunkCacheHitCount
		getBatchDigest ^= getChunkResult.GetWorkDigest
	}
	return benchmarkshared.BenchmarkWorkerContentBatchResult{
		GetChunks:         getChunkResults,
		GetWorker:         strings.TrimSpace(parseWorkerName),
		GetWorkDigest:     getBatchDigest,
		GetWorkDurationMS: time.Since(parseStartedAt).Seconds() * 1000,
		GetGeneration:     parseRequest.GetGeneration,
		GetCacheHitCount:  getCacheHitCount,
	}
}

// buildBenchmarkWorkerCoreChunkWithCache builds one cached core-list chunk result and returns cache-hit count.
// When parseDirtyItemIndexes is non-empty and a previous chunk result is cached, only dirty items are recomputed
// and merged into the previous result, skipping cache-key allocation for clean items.
func buildBenchmarkWorkerCoreChunkWithCache(parseWorkerName string, parseChunkIndex int, parseItems []benchmarkshared.BenchmarkCoreRowData, parseWorkScale int, parseGeneration uint64, parseDirtyItemIndexes []int) (benchmarkshared.BenchmarkWorkerCoreChunkResult, int) {
	parseStartedAt := time.Now()
	// dirty-set fast path: apply delta updates to the previous chunk result without rebuilding clean items
	if len(parseDirtyItemIndexes) > 0 {
		getBenchmarkWorkerChunkResultMu.Lock()
		getPreviousResult, hasPreviousResult := getBenchmarkWorkerCoreChunkResultByIndex[parseChunkIndex]
		getBenchmarkWorkerChunkResultMu.Unlock()
		if hasPreviousResult && len(getPreviousResult.GetItems) == len(parseItems) {
			getPreparedItems := make([]benchmarkshared.BenchmarkPreparedCoreItem, len(parseItems))
			copy(getPreparedItems, getPreviousResult.GetItems)
			// derive full-chunk XOR digest from previous items and apply per-dirty-item XOR deltas
			getBatchDigest := uint64(0)
			for _, getPrevItem := range getPreviousResult.GetItems {
				getBatchDigest ^= getPrevItem.GetDigest
			}
			hasCacheOverLimit := false
			for _, getDirtyIndex := range parseDirtyItemIndexes {
				if getDirtyIndex < 0 || getDirtyIndex >= len(parseItems) {
					continue
				}
				getCacheKey := buildBenchmarkWorkerCoreCacheKey(parseItems[getDirtyIndex], parseWorkScale)
				getBenchmarkWorkerCacheMu.RLock()
				getCachedItem, hasCachedItem := getBenchmarkWorkerCoreItemCacheByKey[getCacheKey]
				getBenchmarkWorkerCacheMu.RUnlock()
				var getNewItem benchmarkshared.BenchmarkPreparedCoreItem
				if hasCachedItem {
					getNewItem = getCachedItem
				} else {
					getNewItem = benchmarkshared.BuildBenchmarkPreparedCoreItem(parseItems[getDirtyIndex], getDirtyIndex, parseWorkScale)
					getBenchmarkWorkerCacheMu.Lock()
					getBenchmarkWorkerCoreItemCacheByKey[getCacheKey] = getNewItem
					if len(getBenchmarkWorkerCoreItemCacheByKey)+len(getBenchmarkWorkerContentItemCacheByKey) > getBenchmarkWorkerCacheLimit {
						hasCacheOverLimit = true
					}
					getBenchmarkWorkerCacheMu.Unlock()
				}
				// XOR out old digest, XOR in new digest
				getBatchDigest ^= getPreviousResult.GetItems[getDirtyIndex].GetDigest
				getBatchDigest ^= getNewItem.GetDigest
				getPreparedItems[getDirtyIndex] = getNewItem
			}
			if hasCacheOverLimit {
				clearBenchmarkWorkerCaches()
			}
			getDeltaResult := benchmarkshared.BenchmarkWorkerCoreChunkResult{
				GetChunkIndex:     parseChunkIndex,
				GetItems:          getPreparedItems,
				GetWorker:         strings.TrimSpace(parseWorkerName),
				GetWorkDigest:     getBatchDigest,
				GetWorkDurationMS: time.Since(parseStartedAt).Seconds() * 1000,
				GetGeneration:     parseGeneration,
			}
			getBenchmarkWorkerChunkResultMu.Lock()
			getBenchmarkWorkerCoreChunkResultByIndex[parseChunkIndex] = getDeltaResult
			getBenchmarkWorkerChunkResultMu.Unlock()
			// cache-hit count = items not in dirty set (pulled from previous result)
			return getDeltaResult, len(parseItems) - len(parseDirtyItemIndexes)
		}
	}
	// full-rebuild path
	getPreparedItems := make([]benchmarkshared.BenchmarkPreparedCoreItem, len(parseItems))
	getCacheHitCount := 0
	var getBatchDigest uint64
	hasCacheOverLimit := false
	for parseItemIndex, parseItem := range parseItems {
		// mid-loop abort: check for newer core generation every 8 items to amortize atomic read cost
		if parseItemIndex&7 == 0 && parseItemIndex > 0 && getBenchmarkWorkerCoreLatestGeneration.Load() > parseGeneration {
			fmt.Printf("[render-benchmark/runtime2][warn] worker core chunk aborted mid-loop item=%d generation=%d\n", parseItemIndex, parseGeneration)
			return benchmarkshared.BenchmarkWorkerCoreChunkResult{
				GetChunkIndex: parseChunkIndex,
				GetWorker:     strings.TrimSpace(parseWorkerName),
				GetGeneration: parseGeneration,
				HasStale:      true,
			}, getCacheHitCount
		}
		getCacheKey := buildBenchmarkWorkerCoreCacheKey(parseItem, parseWorkScale)
		getBenchmarkWorkerCacheMu.RLock()
		getCachedPreparedItem, hasCachedPreparedItem := getBenchmarkWorkerCoreItemCacheByKey[getCacheKey]
		getBenchmarkWorkerCacheMu.RUnlock()
		if hasCachedPreparedItem {
			getPreparedItems[parseItemIndex] = getCachedPreparedItem
			getBatchDigest ^= getCachedPreparedItem.GetDigest
			getCacheHitCount++
			continue
		}
		getPreparedItem := benchmarkshared.BuildBenchmarkPreparedCoreItem(parseItem, parseItemIndex, parseWorkScale)
		getPreparedItems[parseItemIndex] = getPreparedItem
		getBatchDigest ^= getPreparedItem.GetDigest
		getBenchmarkWorkerCacheMu.Lock()
		getBenchmarkWorkerCoreItemCacheByKey[getCacheKey] = getPreparedItem
		if len(getBenchmarkWorkerCoreItemCacheByKey)+len(getBenchmarkWorkerContentItemCacheByKey) > getBenchmarkWorkerCacheLimit {
			hasCacheOverLimit = true
		}
		getBenchmarkWorkerCacheMu.Unlock()
	}
	if hasCacheOverLimit {
		clearBenchmarkWorkerCaches()
	}
	getChunkResult := benchmarkshared.BenchmarkWorkerCoreChunkResult{
		GetChunkIndex:     parseChunkIndex,
		GetItems:          getPreparedItems,
		GetWorker:         strings.TrimSpace(parseWorkerName),
		GetWorkDigest:     getBatchDigest,
		GetWorkDurationMS: time.Since(parseStartedAt).Seconds() * 1000,
		GetGeneration:     parseGeneration,
	}
	getBenchmarkWorkerChunkResultMu.Lock()
	getBenchmarkWorkerCoreChunkResultByIndex[parseChunkIndex] = getChunkResult
	getBenchmarkWorkerChunkResultMu.Unlock()
	return getChunkResult, getCacheHitCount
}

// buildBenchmarkWorkerContentChunkWithCache builds one cached content-card chunk result and returns cache-hit count.
// When parseDirtyItemIndexes is non-empty and a previous chunk result is cached, only dirty items are recomputed
// and merged into the previous result, skipping cache-key allocation for clean items.
func buildBenchmarkWorkerContentChunkWithCache(parseWorkerName string, parseChunkIndex int, parseItems []benchmarkshared.BenchmarkContentCardData, parseWorkScale int, parseGeneration uint64, parseDirtyItemIndexes []int) (benchmarkshared.BenchmarkWorkerContentChunkResult, int) {
	parseStartedAt := time.Now()
	// dirty-set fast path: apply delta updates to the previous chunk result without rebuilding clean items
	if len(parseDirtyItemIndexes) > 0 {
		getBenchmarkWorkerChunkResultMu.Lock()
		getPreviousResult, hasPreviousResult := getBenchmarkWorkerContentChunkResultByIndex[parseChunkIndex]
		getBenchmarkWorkerChunkResultMu.Unlock()
		if hasPreviousResult && len(getPreviousResult.GetItems) == len(parseItems) {
			getPreparedItems := make([]benchmarkshared.BenchmarkPreparedContentCard, len(parseItems))
			copy(getPreparedItems, getPreviousResult.GetItems)
			// derive full-chunk XOR digest from previous items and apply per-dirty-item XOR deltas
			getBatchDigest := uint64(0)
			for _, getPrevItem := range getPreviousResult.GetItems {
				getBatchDigest ^= getPrevItem.GetDigest
			}
			hasCacheOverLimit := false
			for _, getDirtyIndex := range parseDirtyItemIndexes {
				if getDirtyIndex < 0 || getDirtyIndex >= len(parseItems) {
					continue
				}
				getCacheKey := buildBenchmarkWorkerContentCacheKey(parseItems[getDirtyIndex], getDirtyIndex, parseWorkScale)
				getBenchmarkWorkerCacheMu.RLock()
				getCachedItem, hasCachedItem := getBenchmarkWorkerContentItemCacheByKey[getCacheKey]
				getBenchmarkWorkerCacheMu.RUnlock()
				var getNewItem benchmarkshared.BenchmarkPreparedContentCard
				if hasCachedItem {
					getNewItem = getCachedItem
				} else {
					getNewItem = benchmarkshared.BuildBenchmarkPreparedContentCard(parseItems[getDirtyIndex], getDirtyIndex, parseWorkScale)
					getBenchmarkWorkerCacheMu.Lock()
					getBenchmarkWorkerContentItemCacheByKey[getCacheKey] = getNewItem
					if len(getBenchmarkWorkerCoreItemCacheByKey)+len(getBenchmarkWorkerContentItemCacheByKey) > getBenchmarkWorkerCacheLimit {
						hasCacheOverLimit = true
					}
					getBenchmarkWorkerCacheMu.Unlock()
				}
				// XOR out old digest, XOR in new digest
				getBatchDigest ^= getPreviousResult.GetItems[getDirtyIndex].GetDigest
				getBatchDigest ^= getNewItem.GetDigest
				getPreparedItems[getDirtyIndex] = getNewItem
			}
			if hasCacheOverLimit {
				clearBenchmarkWorkerCaches()
			}
			getDeltaResult := benchmarkshared.BenchmarkWorkerContentChunkResult{
				GetChunkIndex:     parseChunkIndex,
				GetItems:          getPreparedItems,
				GetWorker:         strings.TrimSpace(parseWorkerName),
				GetWorkDigest:     getBatchDigest,
				GetWorkDurationMS: time.Since(parseStartedAt).Seconds() * 1000,
				GetGeneration:     parseGeneration,
			}
			getBenchmarkWorkerChunkResultMu.Lock()
			getBenchmarkWorkerContentChunkResultByIndex[parseChunkIndex] = getDeltaResult
			getBenchmarkWorkerChunkResultMu.Unlock()
			// cache-hit count = items not in dirty set (pulled from previous result)
			return getDeltaResult, len(parseItems) - len(parseDirtyItemIndexes)
		}
	}
	// full-rebuild path
	getPreparedItems := make([]benchmarkshared.BenchmarkPreparedContentCard, len(parseItems))
	getCacheHitCount := 0
	var getBatchDigest uint64
	hasCacheOverLimit := false
	for parseItemIndex, parseItem := range parseItems {
		// mid-loop abort: check for newer content generation every 8 items to amortize atomic read cost
		if parseItemIndex&7 == 0 && parseItemIndex > 0 && getBenchmarkWorkerContentLatestGeneration.Load() > parseGeneration {
			fmt.Printf("[render-benchmark/runtime2][warn] worker content chunk aborted mid-loop item=%d generation=%d\n", parseItemIndex, parseGeneration)
			return benchmarkshared.BenchmarkWorkerContentChunkResult{
				GetChunkIndex: parseChunkIndex,
				GetWorker:     strings.TrimSpace(parseWorkerName),
				GetGeneration: parseGeneration,
				HasStale:      true,
			}, getCacheHitCount
		}
		getCacheKey := buildBenchmarkWorkerContentCacheKey(parseItem, parseItemIndex, parseWorkScale)
		getBenchmarkWorkerCacheMu.RLock()
		getCachedPreparedItem, hasCachedPreparedItem := getBenchmarkWorkerContentItemCacheByKey[getCacheKey]
		getBenchmarkWorkerCacheMu.RUnlock()
		if hasCachedPreparedItem {
			getPreparedItems[parseItemIndex] = getCachedPreparedItem
			getBatchDigest ^= getCachedPreparedItem.GetDigest
			getCacheHitCount++
			continue
		}
		getPreparedItem := benchmarkshared.BuildBenchmarkPreparedContentCard(parseItem, parseItemIndex, parseWorkScale)
		getPreparedItems[parseItemIndex] = getPreparedItem
		getBatchDigest ^= getPreparedItem.GetDigest
		getBenchmarkWorkerCacheMu.Lock()
		getBenchmarkWorkerContentItemCacheByKey[getCacheKey] = getPreparedItem
		if len(getBenchmarkWorkerCoreItemCacheByKey)+len(getBenchmarkWorkerContentItemCacheByKey) > getBenchmarkWorkerCacheLimit {
			hasCacheOverLimit = true
		}
		getBenchmarkWorkerCacheMu.Unlock()
	}
	if hasCacheOverLimit {
		clearBenchmarkWorkerCaches()
	}
	getChunkResult := benchmarkshared.BenchmarkWorkerContentChunkResult{
		GetChunkIndex:     parseChunkIndex,
		GetItems:          getPreparedItems,
		GetWorker:         strings.TrimSpace(parseWorkerName),
		GetWorkDigest:     getBatchDigest,
		GetWorkDurationMS: time.Since(parseStartedAt).Seconds() * 1000,
		GetGeneration:     parseGeneration,
	}
	getBenchmarkWorkerChunkResultMu.Lock()
	getBenchmarkWorkerContentChunkResultByIndex[parseChunkIndex] = getChunkResult
	getBenchmarkWorkerChunkResultMu.Unlock()
	return getChunkResult, getCacheHitCount
}

// buildBenchmarkWorkerCoreCacheKey builds one stable core-item cache key from row ID, scale, and source text.
func buildBenchmarkWorkerCoreCacheKey(parseItem benchmarkshared.BenchmarkCoreRowData, parseWorkScale int) string {
	return strconv.Itoa(parseWorkScale) + "|" + strconv.Itoa(parseItem.GetID) + "|" + parseItem.GetText
}

// buildBenchmarkWorkerContentCacheKey builds one stable content-item cache key from index, scale, and card source fields.
func buildBenchmarkWorkerContentCacheKey(parseItem benchmarkshared.BenchmarkContentCardData, parseItemIndex int, parseWorkScale int) string {
	getTagText := strings.Join(parseItem.GetTags, "\x1f")
	return strconv.Itoa(parseWorkScale) +
		"|" + strconv.Itoa(parseItemIndex) +
		"|" + strconv.Itoa(parseItem.GetID) +
		"|" + parseItem.GetTitle +
		"|" + parseItem.GetSummary +
		"|" + parseItem.GetStatus +
		"|" + parseItem.GetMeta +
		"|" + getTagText
}

// clearBenchmarkWorkerCaches clears worker-local prepared-item caches when cache growth passes the configured limit.
func clearBenchmarkWorkerCaches() {
	getBenchmarkWorkerCacheMu.Lock()
	getCachedEntryCount := len(getBenchmarkWorkerCoreItemCacheByKey) + len(getBenchmarkWorkerContentItemCacheByKey)
	getBenchmarkWorkerCoreItemCacheByKey = map[string]benchmarkshared.BenchmarkPreparedCoreItem{}
	getBenchmarkWorkerContentItemCacheByKey = map[string]benchmarkshared.BenchmarkPreparedContentCard{}
	getBenchmarkWorkerCacheMu.Unlock()
	fmt.Printf("[render-benchmark/runtime2][warn] cleared worker caches entries=%d limit=%d\n", getCachedEntryCount, getBenchmarkWorkerCacheLimit)
}

// getBenchmarkWorkerName resolves the current worker scope name or returns a stable fallback.
func getBenchmarkWorkerName() string {
	getGlobal, parseErr := interop.GetGlobalThis()
	if parseErr != nil {
		return getBenchmarkWorkerFallbackName
	}
	getNameValue := getGlobal.Get("name")
	if getNameValue.IsUndefined() || getNameValue.IsNull() {
		return getBenchmarkWorkerFallbackName
	}
	getWorkerName := strings.TrimSpace(getNameValue.String())
	if getWorkerName == "" {
		return getBenchmarkWorkerFallbackName
	}
	return getWorkerName
}
