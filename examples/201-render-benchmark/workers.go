//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
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
	getBenchmarkWorkerWorkScale      = 3
)

type buildBenchmarkWorkerState struct {
	IsBooting         bool
	IsReady           bool
	IsPreparing       bool
	GetWorkerCount    int
	GetPreparedChunks int
	GetPreparedItems  int
	GetLastBatchMS    int64
	GetErrorText      string
}

// hasBenchmarkWorkerMode reports whether one benchmark mode should offload chunk preparation to a Go WASM worker pool.
func hasBenchmarkWorkerMode(parseMode string) bool {
	return parseMode == benchmarkModeRuntime3 || parseMode == benchmarkModeRuntime3Workers
}

// buildBenchmarkWorkerCount resolves the Go WASM worker-pool size for one benchmark mode.
func buildBenchmarkWorkerCount(parseMode string) int {
	switch parseMode {
	case benchmarkModeRuntime3Workers:
		return 4
	case benchmarkModeRuntime3:
		return 1
	default:
		return 0
	}
}

// openBenchmarkWorkerPool opens the Go WASM worker pool used by the runtime2 benchmark modes.
func openBenchmarkWorkerPool(parseCtx context.Context, parseMode string) (interop.WorkerPool, error) {
	getWorkerCount := buildBenchmarkWorkerCount(parseMode)
	getWorkerIndex := 0
	return interop.OpenWorkerPool(parseCtx, interop.WorkerPoolOptions{
		Size:       getWorkerCount,
		QueueLimit: benchmarkRuntime3ShardCount,
		OpenWorker: func(parseOpenCtx context.Context) (interop.Worker, error) {
			getWorkerIndex++
			getWorkerName := fmt.Sprintf("render-benchmark-%s-%d", parseMode, getWorkerIndex)
			return interop.OpenGoWASMWorker(parseOpenCtx, interop.GoWASMWorkerOptions{
				RuntimeURL:   getBenchmarkWorkerRuntimeURL,
				WASMURL:      getBenchmarkWorkerWASMURL,
				Name:         getWorkerName,
				Ready:        true,
				ReadyTimeout: getBenchmarkWorkerReadyTimeout,
			})
		},
	})
}

// handleBenchmarkWorkerPoolEffect boots and tears down the benchmark worker pool for runtime2 benchmark modes.
func handleBenchmarkWorkerPoolEffect(parseMode string, parsePoolRef ui.Ref[*interop.WorkerPool], parsePoolRevisionState ui.State[int], parseWorkerState ui.State[buildBenchmarkWorkerState]) {
	ui.UseEffect(func() func() {
		if !hasBenchmarkWorkerMode(parseMode) {
			return nil
		}
		if parsePoolRef.Get() != nil {
			return nil
		}

		getWorkerCount := buildBenchmarkWorkerCount(parseMode)
		parseWorkerState.Set(buildBenchmarkWorkerState{
			IsBooting:      true,
			GetWorkerCount: getWorkerCount,
		})
		parseCtx, parseCancel := context.WithCancel(context.Background())
		go func() {
			getPool, parseErr := openBenchmarkWorkerPool(parseCtx, parseMode)
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
				_ = getPool.Close()
				return
			}
			parsePoolRef.Set(&getPool)
			parseWorkerState.Set(buildBenchmarkWorkerState{
				IsReady:        true,
				GetWorkerCount: getWorkerCount,
			})
			parsePoolRevisionState.Update(func(parsePrevious int) int {
				return parsePrevious + 1
			})
		}()

		return func() {
			parseCancel()
			if getPool := parsePoolRef.Get(); getPool != nil {
				parsePoolRef.Set(nil)
				_ = getPool.Close()
			}
		}
	}, parseMode)
}

// handleBenchmarkWorkerPrepareEffect refreshes prepared chunk state for the worker-backed runtime2 benchmark modes.
func handleBenchmarkWorkerPrepareEffect(
	parseMode string,
	parseView string,
	parseCoreItems []string,
	parseContentItems []benchmarkshared.BenchmarkContentCardData,
	parsePrepareRevision int,
	parsePoolRevision int,
	parsePoolRef ui.Ref[*interop.WorkerPool],
	parseCoreChunkState ui.State[[]benchmarkshared.BenchmarkWorkerCoreChunkResult],
	parseContentChunkState ui.State[[]benchmarkshared.BenchmarkWorkerContentChunkResult],
	parseWorkerState ui.State[buildBenchmarkWorkerState],
) {
	ui.UseEffect(func() func() {
		if !hasBenchmarkWorkerMode(parseMode) {
			return nil
		}
		getPool := parsePoolRef.Get()
		if getPool == nil {
			return nil
		}
		if parseView == "core" && len(parseCoreItems) == 0 {
			parseCoreChunkState.Set(nil)
			parseWorkerState.Update(func(parsePrevious buildBenchmarkWorkerState) buildBenchmarkWorkerState {
				parsePrevious.IsPreparing = false
				parsePrevious.GetPreparedChunks = 0
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
		parseWorkerState.Update(func(parsePrevious buildBenchmarkWorkerState) buildBenchmarkWorkerState {
			parsePrevious.IsPreparing = true
			parsePrevious.GetErrorText = ""
			return parsePrevious
		})
		go func() {
			if parseView == "core" {
				getChunks, parseErr := requestBenchmarkWorkerCoreChunks(parseCtx, *getPool, parseCoreItems)
				if parseCtx.Err() != nil {
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
				parseCoreChunkState.Set(getChunks)
				parseWorkerState.Update(func(parsePrevious buildBenchmarkWorkerState) buildBenchmarkWorkerState {
					parsePrevious.IsPreparing = false
					parsePrevious.IsReady = true
					parsePrevious.GetPreparedChunks = len(getChunks)
					parsePrevious.GetPreparedItems = len(parseCoreItems)
					parsePrevious.GetLastBatchMS = time.Since(parseStartedAt).Milliseconds()
					parsePrevious.GetErrorText = ""
					return parsePrevious
				})
				return
			}

			getChunks, parseErr := requestBenchmarkWorkerContentChunks(parseCtx, *getPool, parseContentItems)
			if parseCtx.Err() != nil {
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
			parseContentChunkState.Set(getChunks)
			parseWorkerState.Update(func(parsePrevious buildBenchmarkWorkerState) buildBenchmarkWorkerState {
				parsePrevious.IsPreparing = false
				parsePrevious.IsReady = true
				parsePrevious.GetPreparedChunks = len(getChunks)
				parsePrevious.GetPreparedItems = len(parseContentItems)
				parsePrevious.GetLastBatchMS = time.Since(parseStartedAt).Milliseconds()
				parsePrevious.GetErrorText = ""
				return parsePrevious
			})
		}()
		return parseCancel
	}, parseMode, parseView, parsePrepareRevision, parsePoolRevision)
}

// requestBenchmarkWorkerCoreChunks fans out one core-list preparation batch across the benchmark worker pool.
func requestBenchmarkWorkerCoreChunks(parseCtx context.Context, parsePool interop.WorkerPool, parseItems []string) ([]benchmarkshared.BenchmarkWorkerCoreChunkResult, error) {
	getBounds := buildBenchmarkRuntime3ChunkBounds(len(parseItems))
	getResults := make([]benchmarkshared.BenchmarkWorkerCoreChunkResult, len(getBounds))
	var getResultErr error
	var getResultMu sync.Mutex
	var getWait sync.WaitGroup
	for parseChunkIndex, getBound := range getBounds {
		getWait.Add(1)
		go func(parseChunkIndex int, parseStart int, parseEnd int) {
			defer getWait.Done()
			getChunkItems := append([]string(nil), parseItems[parseStart:parseEnd]...)
			getResult, parseErr := interop.RequestWorkerDecoded[benchmarkshared.BenchmarkWorkerCoreChunkRequest, struct{}, benchmarkshared.BenchmarkWorkerCoreChunkResult](
				parseCtx,
				parsePool,
				benchmarkshared.BenchmarkWorkerRequestCoreChunk,
				benchmarkshared.BenchmarkWorkerCoreChunkRequest{
					GetChunkIndex: parseChunkIndex,
					GetItems:      getChunkItems,
					GetWorkScale:  getBenchmarkWorkerWorkScale,
				},
				nil,
			)
			getResultMu.Lock()
			defer getResultMu.Unlock()
			if parseErr != nil {
				if getResultErr == nil {
					getResultErr = parseErr
				}
				return
			}
			getResults[parseChunkIndex] = getResult
		}(parseChunkIndex, getBound[0], getBound[1])
	}
	getWait.Wait()
	if getResultErr != nil {
		return nil, getResultErr
	}
	return getResults, nil
}

// requestBenchmarkWorkerContentChunks fans out one content-card preparation batch across the benchmark worker pool.
func requestBenchmarkWorkerContentChunks(parseCtx context.Context, parsePool interop.WorkerPool, parseItems []benchmarkshared.BenchmarkContentCardData) ([]benchmarkshared.BenchmarkWorkerContentChunkResult, error) {
	getBounds := buildBenchmarkRuntime3ChunkBounds(len(parseItems))
	getResults := make([]benchmarkshared.BenchmarkWorkerContentChunkResult, len(getBounds))
	var getResultErr error
	var getResultMu sync.Mutex
	var getWait sync.WaitGroup
	for parseChunkIndex, getBound := range getBounds {
		getWait.Add(1)
		go func(parseChunkIndex int, parseStart int, parseEnd int) {
			defer getWait.Done()
			getChunkItems := append([]benchmarkshared.BenchmarkContentCardData(nil), parseItems[parseStart:parseEnd]...)
			getResult, parseErr := interop.RequestWorkerDecoded[benchmarkshared.BenchmarkWorkerContentChunkRequest, struct{}, benchmarkshared.BenchmarkWorkerContentChunkResult](
				parseCtx,
				parsePool,
				benchmarkshared.BenchmarkWorkerRequestContentChunk,
				benchmarkshared.BenchmarkWorkerContentChunkRequest{
					GetChunkIndex: parseChunkIndex,
					GetItems:      getChunkItems,
					GetWorkScale:  getBenchmarkWorkerWorkScale,
				},
				nil,
			)
			getResultMu.Lock()
			defer getResultMu.Unlock()
			if parseErr != nil {
				if getResultErr == nil {
					getResultErr = parseErr
				}
				return
			}
			getResults[parseChunkIndex] = getResult
		}(parseChunkIndex, getBound[0], getBound[1])
	}
	getWait.Wait()
	if getResultErr != nil {
		return nil, getResultErr
	}
	return getResults, nil
}
