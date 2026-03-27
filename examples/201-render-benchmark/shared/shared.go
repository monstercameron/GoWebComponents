package shared

import (
	"strings"
	"time"
)

const (
	// BenchmarkWorkerRequestCoreChunk identifies one core-list chunk preparation request.
	BenchmarkWorkerRequestCoreChunk = "prepare-core-chunk"
	// BenchmarkWorkerRequestContentChunk identifies one content-card chunk preparation request.
	BenchmarkWorkerRequestContentChunk = "prepare-content-chunk"
	// BenchmarkWorkerRequestCoreBatch identifies one core-list multi-chunk batch preparation request.
	BenchmarkWorkerRequestCoreBatch = "prepare-core-batch"
	// BenchmarkWorkerRequestContentBatch identifies one content-card multi-chunk batch preparation request.
	BenchmarkWorkerRequestContentBatch = "prepare-content-batch"
)

const (
	getBenchmarkWorkerDigestSeedMask = 63
)

// BenchmarkCoreRowData stores one core-list source row used by the browser benchmark.
type BenchmarkCoreRowData struct {
	GetID   int    `json:"id"`
	GetText string `json:"text"`
}

// BenchmarkContentCardData stores one content-card source item used by the browser benchmark.
type BenchmarkContentCardData struct {
	GetID      int      `json:"id"`
	GetTitle   string   `json:"title"`
	GetSummary string   `json:"summary"`
	GetStatus  string   `json:"status"`
	GetMeta    string   `json:"meta"`
	GetTags    []string `json:"tags"`
}

// BenchmarkPreparedCoreItem stores one worker-prepared core-list item.
type BenchmarkPreparedCoreItem struct {
	GetID     int    `json:"id"`
	GetText   string `json:"text"`
	GetDigest uint64 `json:"digest"`
}

// BenchmarkPreparedContentCard stores one worker-prepared content-card view model.
type BenchmarkPreparedContentCard struct {
	GetID      int      `json:"id"`
	GetTitle   string   `json:"title"`
	GetSummary string   `json:"summary"`
	GetStatus  string   `json:"status"`
	GetMeta    string   `json:"meta"`
	GetTags    []string `json:"tags"`
	GetDigest  uint64   `json:"digest"`
}

// BenchmarkWorkerCoreChunkRequest stores one worker request for a core-list chunk.
type BenchmarkWorkerCoreChunkRequest struct {
	GetChunkIndex int                    `json:"chunkIndex"`
	GetItems      []BenchmarkCoreRowData `json:"items"`
	GetWorkScale  int                    `json:"workScale"`
	GetGeneration uint64                 `json:"generation"`
}

// BenchmarkWorkerCoreChunkResult stores one worker-prepared core-list chunk.
type BenchmarkWorkerCoreChunkResult struct {
	GetChunkIndex     int                         `json:"chunkIndex"`
	GetItems          []BenchmarkPreparedCoreItem `json:"items"`
	GetWorker         string                      `json:"worker"`
	GetWorkDigest     uint64                      `json:"workDigest"`
	GetWorkDurationMS int64                       `json:"workDurationMs"`
	GetGeneration     uint64                      `json:"generation"`
	HasStale          bool                        `json:"stale"`
}

// BenchmarkWorkerContentChunkRequest stores one worker request for a content-card chunk.
type BenchmarkWorkerContentChunkRequest struct {
	GetChunkIndex int                        `json:"chunkIndex"`
	GetItems      []BenchmarkContentCardData `json:"items"`
	GetWorkScale  int                        `json:"workScale"`
	GetGeneration uint64                     `json:"generation"`
}

// BenchmarkWorkerContentChunkResult stores one worker-prepared content-card chunk.
type BenchmarkWorkerContentChunkResult struct {
	GetChunkIndex     int                            `json:"chunkIndex"`
	GetItems          []BenchmarkPreparedContentCard `json:"items"`
	GetWorker         string                         `json:"worker"`
	GetWorkDigest     uint64                         `json:"workDigest"`
	GetWorkDurationMS int64                          `json:"workDurationMs"`
	GetGeneration     uint64                         `json:"generation"`
	HasStale          bool                           `json:"stale"`
}

// BenchmarkWorkerCoreBatchChunkRequest stores one chunk payload in a multi-chunk core batch request.
type BenchmarkWorkerCoreBatchChunkRequest struct {
	GetChunkIndex       int                    `json:"chunkIndex"`
	GetItems            []BenchmarkCoreRowData `json:"items"`
	GetDirtyItemIndexes []int                  `json:"dirtyItemIndexes,omitempty"`
}

// BenchmarkWorkerCoreBatchRequest stores one worker request for a multi-chunk core-list batch.
type BenchmarkWorkerCoreBatchRequest struct {
	GetChunks     []BenchmarkWorkerCoreBatchChunkRequest `json:"chunks"`
	GetWorkScale  int                                    `json:"workScale"`
	GetGeneration uint64                                 `json:"generation"`
}

// BenchmarkWorkerCoreBatchResult stores one worker-prepared core-list batch response.
type BenchmarkWorkerCoreBatchResult struct {
	GetChunks         []BenchmarkWorkerCoreChunkResult `json:"chunks"`
	GetWorker         string                           `json:"worker"`
	GetWorkDigest     uint64                           `json:"workDigest"`
	GetWorkDurationMS int64                            `json:"workDurationMs"`
	GetGeneration     uint64                           `json:"generation"`
	GetCacheHitCount  int                              `json:"cacheHitCount"`
	HasStale          bool                             `json:"stale"`
}

// BenchmarkWorkerContentBatchChunkRequest stores one chunk payload in a multi-chunk content batch request.
type BenchmarkWorkerContentBatchChunkRequest struct {
	GetChunkIndex       int                        `json:"chunkIndex"`
	GetItems            []BenchmarkContentCardData `json:"items"`
	GetDirtyItemIndexes []int                      `json:"dirtyItemIndexes,omitempty"`
}

// BenchmarkWorkerContentBatchRequest stores one worker request for a multi-chunk content-card batch.
type BenchmarkWorkerContentBatchRequest struct {
	GetChunks     []BenchmarkWorkerContentBatchChunkRequest `json:"chunks"`
	GetWorkScale  int                                       `json:"workScale"`
	GetGeneration uint64                                    `json:"generation"`
}

// BenchmarkWorkerContentBatchResult stores one worker-prepared content-card batch response.
type BenchmarkWorkerContentBatchResult struct {
	GetChunks         []BenchmarkWorkerContentChunkResult `json:"chunks"`
	GetWorker         string                              `json:"worker"`
	GetWorkDigest     uint64                              `json:"workDigest"`
	GetWorkDurationMS int64                               `json:"workDurationMs"`
	GetGeneration     uint64                              `json:"generation"`
	GetCacheHitCount  int                                 `json:"cacheHitCount"`
	HasStale          bool                                `json:"stale"`
}

// BuildBenchmarkPreparedCoreItem builds one prepared core-list item with its deterministic digest.
func BuildBenchmarkPreparedCoreItem(parseItem BenchmarkCoreRowData, parseIndex int, parseWorkScale int) BenchmarkPreparedCoreItem {
	return BenchmarkPreparedCoreItem{
		GetID:     parseItem.GetID,
		GetText:   parseItem.GetText,
		GetDigest: BuildBenchmarkWorkerCoreItemDigest(parseItem.GetText, parseItem.GetID, parseWorkScale),
	}
}

// BuildBenchmarkPreparedContentCard builds one prepared content-card item with its deterministic digest.
func BuildBenchmarkPreparedContentCard(parseItem BenchmarkContentCardData, parseIndex int, parseWorkScale int) BenchmarkPreparedContentCard {
	return BenchmarkPreparedContentCard{
		GetID:      parseItem.GetID,
		GetTitle:   parseItem.GetTitle,
		GetSummary: parseItem.GetSummary,
		GetStatus:  parseItem.GetStatus,
		GetMeta:    parseItem.GetMeta,
		GetTags:    append([]string(nil), parseItem.GetTags...),
		GetDigest:  BuildBenchmarkWorkerContentItemDigest(parseItem, parseIndex, parseWorkScale),
	}
}

// BuildBenchmarkWorkerCoreChunkResult builds one prepared core-list chunk result.
func BuildBenchmarkWorkerCoreChunkResult(parseWorkerName string, parseRequest BenchmarkWorkerCoreChunkRequest) BenchmarkWorkerCoreChunkResult {
	parseStartedAt := time.Now()
	getPreparedItems := make([]BenchmarkPreparedCoreItem, len(parseRequest.GetItems))
	var getBatchDigest uint64
	for parseIndex, parseItem := range parseRequest.GetItems {
		getPreparedItem := BuildBenchmarkPreparedCoreItem(parseItem, parseIndex, parseRequest.GetWorkScale)
		getPreparedItems[parseIndex] = getPreparedItem
		getBatchDigest ^= getPreparedItem.GetDigest
	}
	return BenchmarkWorkerCoreChunkResult{
		GetChunkIndex:     parseRequest.GetChunkIndex,
		GetItems:          getPreparedItems,
		GetWorker:         strings.TrimSpace(parseWorkerName),
		GetWorkDigest:     getBatchDigest,
		GetWorkDurationMS: time.Since(parseStartedAt).Milliseconds(),
		GetGeneration:     parseRequest.GetGeneration,
	}
}

// BuildBenchmarkWorkerContentChunkResult builds one prepared content-card chunk result.
func BuildBenchmarkWorkerContentChunkResult(parseWorkerName string, parseRequest BenchmarkWorkerContentChunkRequest) BenchmarkWorkerContentChunkResult {
	parseStartedAt := time.Now()
	getPreparedItems := make([]BenchmarkPreparedContentCard, len(parseRequest.GetItems))
	var getBatchDigest uint64
	for parseIndex, parseItem := range parseRequest.GetItems {
		getPreparedItem := BuildBenchmarkPreparedContentCard(parseItem, parseIndex, parseRequest.GetWorkScale)
		getPreparedItems[parseIndex] = getPreparedItem
		getBatchDigest ^= getPreparedItem.GetDigest
	}
	return BenchmarkWorkerContentChunkResult{
		GetChunkIndex:     parseRequest.GetChunkIndex,
		GetItems:          getPreparedItems,
		GetWorker:         strings.TrimSpace(parseWorkerName),
		GetWorkDigest:     getBatchDigest,
		GetWorkDurationMS: time.Since(parseStartedAt).Milliseconds(),
		GetGeneration:     parseRequest.GetGeneration,
	}
}

// BuildBenchmarkWorkerContentItemDigest computes one deterministic digest for one content-card source item.
func BuildBenchmarkWorkerContentItemDigest(parseItem BenchmarkContentCardData, parseIndex int, parseWorkScale int) uint64 {
	getDigest := BuildBenchmarkWorkerCoreItemDigest(parseItem.GetTitle, parseIndex, parseWorkScale)
	getDigest ^= BuildBenchmarkWorkerCoreItemDigest(parseItem.GetSummary, parseIndex+3, parseWorkScale)
	getDigest ^= BuildBenchmarkWorkerCoreItemDigest(parseItem.GetMeta, parseIndex+7, parseWorkScale)
	getDigest ^= BuildBenchmarkWorkerCoreItemDigest(parseItem.GetStatus, parseIndex+11, parseWorkScale)
	for parseTagIndex, parseTag := range parseItem.GetTags {
		getDigest ^= BuildBenchmarkWorkerCoreItemDigest(parseTag, parseIndex+parseTagIndex+17, parseWorkScale)
	}
	return getDigest
}

// buildBenchmarkWorkerDigestSeed normalizes one digest seed into a bounded range so large row IDs do not dominate benchmark work.
func buildBenchmarkWorkerDigestSeed(parseDigestSeed int) int {
	getSeed := parseDigestSeed
	if getSeed < 0 {
		getSeed = -getSeed
		if getSeed < 0 {
			getSeed = 0
		}
	}
	getSeed ^= getSeed >> 6
	getSeed ^= getSeed >> 12
	return getSeed & getBenchmarkWorkerDigestSeedMask
}

// BuildBenchmarkWorkerCoreItemDigest computes one deterministic digest for one core-list source item.
func BuildBenchmarkWorkerCoreItemDigest(parseText string, parseDigestSeed int, parseWorkScale int) uint64 {
	getScale := parseWorkScale
	if getScale < 1 {
		getScale = 1
	}
	getDigestSeed := buildBenchmarkWorkerDigestSeed(parseDigestSeed)
	getDigest := uint64(1469598103934665603)
	getTextBytes := []byte(parseText)
	if len(getTextBytes) == 0 {
		getTextBytes = []byte{0}
	}
	for _, parseByte := range getTextBytes {
		getDigest ^= uint64(parseByte) + uint64(getDigestSeed+1)
		getDigest *= 1099511628211
	}
	getIterations := (2500 + (len(getTextBytes) * 450) + (getDigestSeed * 75)) * getScale
	for parseStep := 0; parseStep < getIterations; parseStep++ {
		getDigest ^= getDigest << 13
		getDigest ^= getDigest >> 7
		getDigest ^= getDigest << 17
		getDigest += uint64(parseStep*97 + len(getTextBytes)*31 + getDigestSeed*11)
	}
	return getDigest
}
