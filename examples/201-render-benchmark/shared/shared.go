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
)

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
	GetChunkIndex int      `json:"chunkIndex"`
	GetItems      []string `json:"items"`
	GetWorkScale  int      `json:"workScale"`
}

// BenchmarkWorkerCoreChunkResult stores one worker-prepared core-list chunk.
type BenchmarkWorkerCoreChunkResult struct {
	GetChunkIndex     int                         `json:"chunkIndex"`
	GetItems          []BenchmarkPreparedCoreItem `json:"items"`
	GetWorker         string                      `json:"worker"`
	GetWorkDigest     uint64                      `json:"workDigest"`
	GetWorkDurationMS int64                       `json:"workDurationMs"`
}

// BenchmarkWorkerContentChunkRequest stores one worker request for a content-card chunk.
type BenchmarkWorkerContentChunkRequest struct {
	GetChunkIndex int                        `json:"chunkIndex"`
	GetItems      []BenchmarkContentCardData `json:"items"`
	GetWorkScale  int                        `json:"workScale"`
}

// BenchmarkWorkerContentChunkResult stores one worker-prepared content-card chunk.
type BenchmarkWorkerContentChunkResult struct {
	GetChunkIndex     int                            `json:"chunkIndex"`
	GetItems          []BenchmarkPreparedContentCard `json:"items"`
	GetWorker         string                         `json:"worker"`
	GetWorkDigest     uint64                         `json:"workDigest"`
	GetWorkDurationMS int64                          `json:"workDurationMs"`
}

// BuildBenchmarkWorkerCoreChunkResult builds one prepared core-list chunk result.
func BuildBenchmarkWorkerCoreChunkResult(parseWorkerName string, parseRequest BenchmarkWorkerCoreChunkRequest) BenchmarkWorkerCoreChunkResult {
	parseStartedAt := time.Now()
	getPreparedItems := make([]BenchmarkPreparedCoreItem, len(parseRequest.GetItems))
	var getBatchDigest uint64
	for parseIndex, parseItem := range parseRequest.GetItems {
		getDigest := buildBenchmarkWorkerStringDigest(parseItem, parseIndex, parseRequest.GetWorkScale)
		getPreparedItems[parseIndex] = BenchmarkPreparedCoreItem{
			GetText:   parseItem,
			GetDigest: getDigest,
		}
		getBatchDigest ^= getDigest
	}
	return BenchmarkWorkerCoreChunkResult{
		GetChunkIndex:     parseRequest.GetChunkIndex,
		GetItems:          getPreparedItems,
		GetWorker:         strings.TrimSpace(parseWorkerName),
		GetWorkDigest:     getBatchDigest,
		GetWorkDurationMS: time.Since(parseStartedAt).Milliseconds(),
	}
}

// BuildBenchmarkWorkerContentChunkResult builds one prepared content-card chunk result.
func BuildBenchmarkWorkerContentChunkResult(parseWorkerName string, parseRequest BenchmarkWorkerContentChunkRequest) BenchmarkWorkerContentChunkResult {
	parseStartedAt := time.Now()
	getPreparedItems := make([]BenchmarkPreparedContentCard, len(parseRequest.GetItems))
	var getBatchDigest uint64
	for parseIndex, parseItem := range parseRequest.GetItems {
		getDigest := buildBenchmarkWorkerContentDigest(parseItem, parseIndex, parseRequest.GetWorkScale)
		getPreparedItems[parseIndex] = BenchmarkPreparedContentCard{
			GetID:      parseItem.GetID,
			GetTitle:   parseItem.GetTitle,
			GetSummary: parseItem.GetSummary,
			GetStatus:  parseItem.GetStatus,
			GetMeta:    parseItem.GetMeta,
			GetTags:    append([]string(nil), parseItem.GetTags...),
			GetDigest:  getDigest,
		}
		getBatchDigest ^= getDigest
	}
	return BenchmarkWorkerContentChunkResult{
		GetChunkIndex:     parseRequest.GetChunkIndex,
		GetItems:          getPreparedItems,
		GetWorker:         strings.TrimSpace(parseWorkerName),
		GetWorkDigest:     getBatchDigest,
		GetWorkDurationMS: time.Since(parseStartedAt).Milliseconds(),
	}
}

func buildBenchmarkWorkerContentDigest(parseItem BenchmarkContentCardData, parseIndex int, parseWorkScale int) uint64 {
	getDigest := buildBenchmarkWorkerStringDigest(parseItem.GetTitle, parseIndex, parseWorkScale)
	getDigest ^= buildBenchmarkWorkerStringDigest(parseItem.GetSummary, parseIndex+3, parseWorkScale)
	getDigest ^= buildBenchmarkWorkerStringDigest(parseItem.GetMeta, parseIndex+7, parseWorkScale)
	getDigest ^= buildBenchmarkWorkerStringDigest(parseItem.GetStatus, parseIndex+11, parseWorkScale)
	for parseTagIndex, parseTag := range parseItem.GetTags {
		getDigest ^= buildBenchmarkWorkerStringDigest(parseTag, parseIndex+parseTagIndex+17, parseWorkScale)
	}
	return getDigest
}

func buildBenchmarkWorkerStringDigest(parseText string, parseIndex int, parseWorkScale int) uint64 {
	getScale := parseWorkScale
	if getScale < 1 {
		getScale = 1
	}
	getDigest := uint64(1469598103934665603)
	getTextBytes := []byte(parseText)
	if len(getTextBytes) == 0 {
		getTextBytes = []byte{0}
	}
	for _, parseByte := range getTextBytes {
		getDigest ^= uint64(parseByte) + uint64(parseIndex+1)
		getDigest *= 1099511628211
	}
	getIterations := (2500 + (len(getTextBytes) * 450) + (parseIndex * 75)) * getScale
	for parseStep := 0; parseStep < getIterations; parseStep++ {
		getDigest ^= getDigest << 13
		getDigest ^= getDigest >> 7
		getDigest ^= getDigest << 17
		getDigest += uint64(parseStep*97 + len(getTextBytes)*31 + parseIndex*11)
	}
	return getDigest
}
