package renderworker

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/monstercameron/GoWebComponents/v5/interop"
)

// BatchDispatchOptions configures one lane-batched multi-worker request pass that aggregates chunk results by index.
type BatchDispatchOptions[BatchReq any, BatchRes any, ChunkRes any] struct {
	GetCtx                    context.Context
	GetRequestName            string
	GetExpectedGeneration     uint64
	GetRequesters             []interop.WorkerRequester
	GetLanePlans              []RenderWorkerLanePlan
	GetChunkCount             int
	BuildBatchRequest         func(RenderWorkerLanePlan) BatchReq
	GetBatchChunks            func(BatchRes) []ChunkRes
	GetChunkIndex             func(ChunkRes) int
	GetBatchGeneration        func(BatchRes) uint64
	GetChunkGeneration        func(ChunkRes) uint64
	IsBatchStale              func(BatchRes) bool
	IsChunkStale              func(ChunkRes) bool
	HandlePartialChunkResults func([]ChunkRes)
}

// RequestRenderWorkerChunkBatches fans out one lane-batched request plan and aggregates chunk results into stable chunk-index order.
func RequestRenderWorkerChunkBatches[BatchReq any, BatchRes any, ChunkRes any](parseOptions BatchDispatchOptions[BatchReq, BatchRes, ChunkRes]) ([]ChunkRes, error) {
	parseCtx := parseOptions.GetCtx
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	parseRequestName := strings.TrimSpace(parseOptions.GetRequestName)
	if parseRequestName == "" {
		return nil, fmt.Errorf("renderworker: request name is required")
	}
	if len(parseOptions.GetRequesters) < 1 {
		return nil, fmt.Errorf("renderworker: at least one requester is required")
	}
	if len(parseOptions.GetLanePlans) < 1 {
		return nil, fmt.Errorf("renderworker: at least one lane plan is required")
	}
	if parseOptions.GetChunkCount < 1 {
		return nil, fmt.Errorf("renderworker: chunk count must be positive")
	}
	if parseOptions.BuildBatchRequest == nil {
		return nil, fmt.Errorf("renderworker: batch request builder is required")
	}
	if parseOptions.GetBatchChunks == nil {
		return nil, fmt.Errorf("renderworker: batch chunk accessor is required")
	}
	if parseOptions.GetChunkIndex == nil {
		return nil, fmt.Errorf("renderworker: chunk index accessor is required")
	}

	getResults := make([]ChunkRes, parseOptions.GetChunkCount)
	hasResultByChunkIndex := make([]bool, parseOptions.GetChunkCount)
	var getResultErr error
	var getResultStateMu sync.Mutex
	var getErrorMu sync.Mutex
	setRenderWorkerBatchDispatchError := func(parseErr error) {
		if parseErr == nil {
			return
		}
		getErrorMu.Lock()
		defer getErrorMu.Unlock()
		if getResultErr == nil {
			getResultErr = parseErr
		}
	}
	var parseWait sync.WaitGroup
	for _, parseLanePlan := range parseOptions.GetLanePlans {
		if len(parseLanePlan.GetChunkPlans) < 1 {
			continue
		}
		parseWait.Add(1)
		go func(parseLane RenderWorkerLanePlan) {
			defer parseWait.Done()
			parseRequester := parseOptions.GetRequesters[parseLane.GetWorkerIndex%len(parseOptions.GetRequesters)]
			if parseRequester == nil {
				setRenderWorkerBatchDispatchError(fmt.Errorf("renderworker: lane %d requester is nil", parseLane.GetWorkerIndex))
				return
			}
			parseBatchResult, parseRequestErr := interop.RequestWorkerDecoded[BatchReq, struct{}, BatchRes](
				parseCtx,
				parseRequester,
				parseRequestName,
				parseOptions.BuildBatchRequest(parseLane),
				nil,
			)
			if parseRequestErr != nil {
				setRenderWorkerBatchDispatchError(parseRequestErr)
				return
			}
			if parseOptions.IsBatchStale != nil && parseOptions.IsBatchStale(parseBatchResult) {
				setRenderWorkerBatchDispatchError(fmt.Errorf("renderworker: stale batch result for lane %d", parseLane.GetWorkerIndex))
				return
			}
			if parseOptions.GetBatchGeneration != nil && parseOptions.GetExpectedGeneration > 0 {
				parseBatchGeneration := parseOptions.GetBatchGeneration(parseBatchResult)
				if parseBatchGeneration != parseOptions.GetExpectedGeneration {
					setRenderWorkerBatchDispatchError(fmt.Errorf("renderworker: batch generation mismatch lane=%d got=%d want=%d", parseLane.GetWorkerIndex, parseBatchGeneration, parseOptions.GetExpectedGeneration))
					return
				}
			}
			parseChunkResults := parseOptions.GetBatchChunks(parseBatchResult)
			getResultStateMu.Lock()
			for _, parseChunkResult := range parseChunkResults {
				if parseOptions.IsChunkStale != nil && parseOptions.IsChunkStale(parseChunkResult) {
					setRenderWorkerBatchDispatchError(fmt.Errorf("renderworker: stale chunk result in lane %d", parseLane.GetWorkerIndex))
					continue
				}
				if parseOptions.GetChunkGeneration != nil && parseOptions.GetExpectedGeneration > 0 {
					parseChunkGeneration := parseOptions.GetChunkGeneration(parseChunkResult)
					if parseChunkGeneration != parseOptions.GetExpectedGeneration {
						setRenderWorkerBatchDispatchError(fmt.Errorf("renderworker: chunk generation mismatch lane=%d got=%d want=%d", parseLane.GetWorkerIndex, parseChunkGeneration, parseOptions.GetExpectedGeneration))
						continue
					}
				}
				parseChunkIndex := parseOptions.GetChunkIndex(parseChunkResult)
				if parseChunkIndex < 0 || parseChunkIndex >= len(getResults) {
					setRenderWorkerBatchDispatchError(fmt.Errorf("renderworker: chunk index out of range index=%d chunks=%d", parseChunkIndex, len(getResults)))
					continue
				}
				if hasResultByChunkIndex[parseChunkIndex] {
					setRenderWorkerBatchDispatchError(fmt.Errorf("renderworker: duplicate chunk index=%d", parseChunkIndex))
					continue
				}
				getResults[parseChunkIndex] = parseChunkResult
				hasResultByChunkIndex[parseChunkIndex] = true
			}
			var getPartialSnapshot []ChunkRes
			if parseOptions.HandlePartialChunkResults != nil {
				getPartialSnapshot = make([]ChunkRes, len(getResults))
				copy(getPartialSnapshot, getResults)
			}
			getResultStateMu.Unlock()
			if parseOptions.HandlePartialChunkResults != nil {
				parseOptions.HandlePartialChunkResults(getPartialSnapshot)
			}
		}(parseLanePlan)
	}
	parseWait.Wait()
	if getResultErr != nil {
		return nil, getResultErr
	}
	for parseChunkIndex := range hasResultByChunkIndex {
		if !hasResultByChunkIndex[parseChunkIndex] {
			return nil, fmt.Errorf("renderworker: missing chunk index=%d", parseChunkIndex)
		}
	}
	return getResults, nil
}
