package renderworker

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/interop"
)

type parseBatchDispatchChunkRequest struct {
	GetChunkIndex int `json:"chunkIndex"`
}

type parseBatchDispatchBatchRequest struct {
	GetChunks     []parseBatchDispatchChunkRequest `json:"chunks"`
	GetGeneration uint64                           `json:"generation"`
}

type parseBatchDispatchChunkResult struct {
	GetChunkIndex int    `json:"chunkIndex"`
	GetPayload    string `json:"payload"`
	GetGeneration uint64 `json:"generation"`
	HasStale      bool   `json:"stale"`
}

type parseBatchDispatchBatchResult struct {
	GetChunks     []parseBatchDispatchChunkResult `json:"chunks"`
	GetGeneration uint64                          `json:"generation"`
	HasStale      bool                            `json:"stale"`
}

type parseBatchDispatchFakeRequester struct {
	storeMu         sync.Mutex
	storeRequestHit int
	handleRequest   func(context.Context, string, any) (interop.WorkerMessage, error)
}

// Request implements interop.WorkerRequester with a test-provided response function.
func (parseRequester *parseBatchDispatchFakeRequester) Request(parseCtx context.Context, parseName string, parsePayload any, parseOnProgress func(interop.WorkerMessage, error)) (interop.WorkerMessage, error) {
	_ = parseOnProgress
	if parseRequester == nil || parseRequester.handleRequest == nil {
		return interop.WorkerMessage{}, fmt.Errorf("missing fake requester handler")
	}
	parseRequester.storeMu.Lock()
	parseRequester.storeRequestHit++
	parseRequester.storeMu.Unlock()
	return parseRequester.handleRequest(parseCtx, parseName, parsePayload)
}

// getBatchDispatchRequestHitCount returns the number of requests received by one fake requester.
func (parseRequester *parseBatchDispatchFakeRequester) getBatchDispatchRequestHitCount() int {
	parseRequester.storeMu.Lock()
	defer parseRequester.storeMu.Unlock()
	return parseRequester.storeRequestHit
}

// TestRequestRenderWorkerChunkBatchesAggregatesLaneBatches validates one request-per-lane dispatch and chunk-index aggregation.
func TestRequestRenderWorkerChunkBatchesAggregatesLaneBatches(parseT *testing.T) {
	parseChunkPlans := []RenderWorkerChunkPlan{
		{GetChunkIndex: 0, GetWeight: 8},
		{GetChunkIndex: 1, GetWeight: 6},
		{GetChunkIndex: 2, GetWeight: 4},
		{GetChunkIndex: 3, GetWeight: 2},
	}
	parseLanePlans := BuildRenderWorkerLanePlans(2, parseChunkPlans)
	parseGeneration := uint64(7)
	parseRequesterA := &parseBatchDispatchFakeRequester{
		handleRequest: func(parseCtx context.Context, parseName string, parsePayload any) (interop.WorkerMessage, error) {
			_ = parseCtx
			_ = parseName
			parseRequest, hasRequest := parsePayload.(parseBatchDispatchBatchRequest)
			if !hasRequest {
				return interop.WorkerMessage{}, fmt.Errorf("unexpected payload type %T", parsePayload)
			}
			parseChunkResults := make([]parseBatchDispatchChunkResult, 0, len(parseRequest.GetChunks))
			for _, parseChunk := range parseRequest.GetChunks {
				parseChunkResults = append(parseChunkResults, parseBatchDispatchChunkResult{
					GetChunkIndex: parseChunk.GetChunkIndex,
					GetPayload:    "worker-a",
					GetGeneration: parseRequest.GetGeneration,
				})
			}
			return interop.WorkerMessage{
				Phase:   "result",
				Name:    "prepare",
				Payload: parseBatchDispatchBatchResult{GetChunks: parseChunkResults, GetGeneration: parseRequest.GetGeneration},
			}, nil
		},
	}
	parseRequesterB := &parseBatchDispatchFakeRequester{
		handleRequest: func(parseCtx context.Context, parseName string, parsePayload any) (interop.WorkerMessage, error) {
			_ = parseCtx
			_ = parseName
			parseRequest, hasRequest := parsePayload.(parseBatchDispatchBatchRequest)
			if !hasRequest {
				return interop.WorkerMessage{}, fmt.Errorf("unexpected payload type %T", parsePayload)
			}
			parseChunkResults := make([]parseBatchDispatchChunkResult, 0, len(parseRequest.GetChunks))
			for _, parseChunk := range parseRequest.GetChunks {
				parseChunkResults = append(parseChunkResults, parseBatchDispatchChunkResult{
					GetChunkIndex: parseChunk.GetChunkIndex,
					GetPayload:    "worker-b",
					GetGeneration: parseRequest.GetGeneration,
				})
			}
			return interop.WorkerMessage{
				Phase:   "result",
				Name:    "prepare",
				Payload: parseBatchDispatchBatchResult{GetChunks: parseChunkResults, GetGeneration: parseRequest.GetGeneration},
			}, nil
		},
	}

	parsePartialCallCount := 0
	parseResults, parseErr := RequestRenderWorkerChunkBatches(BatchDispatchOptions[parseBatchDispatchBatchRequest, parseBatchDispatchBatchResult, parseBatchDispatchChunkResult]{
		GetCtx:                context.Background(),
		GetRequestName:        "prepare",
		GetExpectedGeneration: parseGeneration,
		GetRequesters:         []interop.WorkerRequester{parseRequesterA, parseRequesterB},
		GetLanePlans:          parseLanePlans,
		GetChunkCount:         len(parseChunkPlans),
		BuildBatchRequest: func(parseLanePlan RenderWorkerLanePlan) parseBatchDispatchBatchRequest {
			parseChunks := make([]parseBatchDispatchChunkRequest, len(parseLanePlan.GetChunkPlans))
			for parseChunkOffset, parseChunkPlan := range parseLanePlan.GetChunkPlans {
				parseChunks[parseChunkOffset] = parseBatchDispatchChunkRequest{GetChunkIndex: parseChunkPlan.GetChunkIndex}
			}
			return parseBatchDispatchBatchRequest{GetChunks: parseChunks, GetGeneration: parseGeneration}
		},
		GetBatchChunks: func(parseBatchResult parseBatchDispatchBatchResult) []parseBatchDispatchChunkResult {
			return parseBatchResult.GetChunks
		},
		GetChunkIndex: func(parseChunkResult parseBatchDispatchChunkResult) int {
			return parseChunkResult.GetChunkIndex
		},
		GetBatchGeneration: func(parseBatchResult parseBatchDispatchBatchResult) uint64 {
			return parseBatchResult.GetGeneration
		},
		GetChunkGeneration: func(parseChunkResult parseBatchDispatchChunkResult) uint64 {
			return parseChunkResult.GetGeneration
		},
		IsBatchStale: func(parseBatchResult parseBatchDispatchBatchResult) bool {
			return parseBatchResult.HasStale
		},
		IsChunkStale: func(parseChunkResult parseBatchDispatchChunkResult) bool {
			return parseChunkResult.HasStale
		},
		HandlePartialChunkResults: func(parsePartial []parseBatchDispatchChunkResult) {
			_ = parsePartial
			parsePartialCallCount++
		},
	})
	if parseErr != nil {
		parseT.Fatalf("expected lane-batched dispatch to succeed, got %v", parseErr)
	}
	if len(parseResults) != len(parseChunkPlans) {
		parseT.Fatalf("expected %d results, got %d", len(parseChunkPlans), len(parseResults))
	}
	if parseRequesterA.getBatchDispatchRequestHitCount()+parseRequesterB.getBatchDispatchRequestHitCount() != 2 {
		parseT.Fatalf("expected one request per lane (2 total), got requester-a=%d requester-b=%d", parseRequesterA.getBatchDispatchRequestHitCount(), parseRequesterB.getBatchDispatchRequestHitCount())
	}
	for parseChunkIndex := range parseResults {
		if parseResults[parseChunkIndex].GetChunkIndex != parseChunkIndex {
			parseT.Fatalf("expected chunk result index %d, got %d", parseChunkIndex, parseResults[parseChunkIndex].GetChunkIndex)
		}
	}
	if parsePartialCallCount < 1 {
		parseT.Fatalf("expected partial callback to be invoked at least once")
	}
}

// TestRequestRenderWorkerChunkBatchesRejectsStaleBatch validates stale-batch rejection.
func TestRequestRenderWorkerChunkBatchesRejectsStaleBatch(parseT *testing.T) {
	parseRequester := &parseBatchDispatchFakeRequester{
		handleRequest: func(parseCtx context.Context, parseName string, parsePayload any) (interop.WorkerMessage, error) {
			_ = parseCtx
			_ = parseName
			_ = parsePayload
			return interop.WorkerMessage{
				Phase: "result",
				Name:  "prepare",
				Payload: parseBatchDispatchBatchResult{
					HasStale: true,
				},
			}, nil
		},
	}
	_, parseErr := RequestRenderWorkerChunkBatches(BatchDispatchOptions[parseBatchDispatchBatchRequest, parseBatchDispatchBatchResult, parseBatchDispatchChunkResult]{
		GetCtx:                context.Background(),
		GetRequestName:        "prepare",
		GetExpectedGeneration: 1,
		GetRequesters:         []interop.WorkerRequester{parseRequester},
		GetLanePlans:          []RenderWorkerLanePlan{{GetWorkerIndex: 0, GetChunkPlans: []RenderWorkerChunkPlan{{GetChunkIndex: 0}}}},
		GetChunkCount:         1,
		BuildBatchRequest: func(parseLanePlan RenderWorkerLanePlan) parseBatchDispatchBatchRequest {
			_ = parseLanePlan
			return parseBatchDispatchBatchRequest{}
		},
		GetBatchChunks: func(parseBatchResult parseBatchDispatchBatchResult) []parseBatchDispatchChunkResult {
			return parseBatchResult.GetChunks
		},
		GetChunkIndex: func(parseChunkResult parseBatchDispatchChunkResult) int {
			return parseChunkResult.GetChunkIndex
		},
		IsBatchStale: func(parseBatchResult parseBatchDispatchBatchResult) bool {
			return parseBatchResult.HasStale
		},
	})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "stale batch") {
		parseT.Fatalf("expected stale-batch error, got %v", parseErr)
	}
}

// TestRequestRenderWorkerChunkBatchesRejectsDuplicateChunk validates duplicate chunk-index rejection.
func TestRequestRenderWorkerChunkBatchesRejectsDuplicateChunk(parseT *testing.T) {
	parseRequester := &parseBatchDispatchFakeRequester{
		handleRequest: func(parseCtx context.Context, parseName string, parsePayload any) (interop.WorkerMessage, error) {
			_ = parseCtx
			_ = parseName
			_ = parsePayload
			return interop.WorkerMessage{
				Phase: "result",
				Name:  "prepare",
				Payload: parseBatchDispatchBatchResult{
					GetChunks: []parseBatchDispatchChunkResult{
						{GetChunkIndex: 0},
						{GetChunkIndex: 0},
					},
				},
			}, nil
		},
	}
	_, parseErr := RequestRenderWorkerChunkBatches(BatchDispatchOptions[parseBatchDispatchBatchRequest, parseBatchDispatchBatchResult, parseBatchDispatchChunkResult]{
		GetCtx:         context.Background(),
		GetRequestName: "prepare",
		GetRequesters:  []interop.WorkerRequester{parseRequester},
		GetLanePlans:   []RenderWorkerLanePlan{{GetWorkerIndex: 0, GetChunkPlans: []RenderWorkerChunkPlan{{GetChunkIndex: 0}}}},
		GetChunkCount:  1,
		BuildBatchRequest: func(parseLanePlan RenderWorkerLanePlan) parseBatchDispatchBatchRequest {
			_ = parseLanePlan
			return parseBatchDispatchBatchRequest{}
		},
		GetBatchChunks: func(parseBatchResult parseBatchDispatchBatchResult) []parseBatchDispatchChunkResult {
			return parseBatchResult.GetChunks
		},
		GetChunkIndex: func(parseChunkResult parseBatchDispatchChunkResult) int {
			return parseChunkResult.GetChunkIndex
		},
	})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "duplicate chunk index") {
		parseT.Fatalf("expected duplicate chunk error, got %v", parseErr)
	}
}

// TestRequestRenderWorkerChunkBatchesRejectsMissingChunk validates missing chunk-index rejection.
func TestRequestRenderWorkerChunkBatchesRejectsMissingChunk(parseT *testing.T) {
	parseRequester := &parseBatchDispatchFakeRequester{
		handleRequest: func(parseCtx context.Context, parseName string, parsePayload any) (interop.WorkerMessage, error) {
			_ = parseCtx
			_ = parseName
			_ = parsePayload
			return interop.WorkerMessage{
				Phase: "result",
				Name:  "prepare",
				Payload: parseBatchDispatchBatchResult{
					GetChunks: []parseBatchDispatchChunkResult{{GetChunkIndex: 0}},
				},
			}, nil
		},
	}
	_, parseErr := RequestRenderWorkerChunkBatches(BatchDispatchOptions[parseBatchDispatchBatchRequest, parseBatchDispatchBatchResult, parseBatchDispatchChunkResult]{
		GetCtx:         context.Background(),
		GetRequestName: "prepare",
		GetRequesters:  []interop.WorkerRequester{parseRequester},
		GetLanePlans:   []RenderWorkerLanePlan{{GetWorkerIndex: 0, GetChunkPlans: []RenderWorkerChunkPlan{{GetChunkIndex: 0}, {GetChunkIndex: 1}}}},
		GetChunkCount:  2,
		BuildBatchRequest: func(parseLanePlan RenderWorkerLanePlan) parseBatchDispatchBatchRequest {
			_ = parseLanePlan
			return parseBatchDispatchBatchRequest{}
		},
		GetBatchChunks: func(parseBatchResult parseBatchDispatchBatchResult) []parseBatchDispatchChunkResult {
			return parseBatchResult.GetChunks
		},
		GetChunkIndex: func(parseChunkResult parseBatchDispatchChunkResult) int {
			return parseChunkResult.GetChunkIndex
		},
	})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "missing chunk index") {
		parseT.Fatalf("expected missing chunk error, got %v", parseErr)
	}
}
