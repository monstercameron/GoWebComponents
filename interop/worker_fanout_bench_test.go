package interop

import (
	"context"
	"sync"
	"testing"
)

const getWorkerFanoutBenchmarkWorkerCount = 8

type workerFanoutBenchmarkRequest struct {
	GetProbe int `json:"probe"`
}

type workerFanoutBenchmarkResult struct {
	GetProbe int `json:"probe"`
}

// buildWorkerFanoutBenchmarkWorker builds one no-op request-capable worker for dispatch microbenchmarks.
func buildWorkerFanoutBenchmarkWorker(parseWorkerOffset int) Worker {
	return Worker{
		request: func(parseCtx context.Context, parseName string, parsePayload any, parseOnProgress func(WorkerMessage, error)) (WorkerMessage, error) {
			select {
			case <-parseCtx.Done():
				return WorkerMessage{}, parseCtx.Err()
			default:
			}
			parseRequest, _ := parsePayload.(workerFanoutBenchmarkRequest)
			return WorkerMessage{
				ID:      "bench",
				Phase:   "result",
				Name:    parseName,
				Payload: workerFanoutBenchmarkResult{GetProbe: parseRequest.GetProbe + parseWorkerOffset},
			}, nil
		},
		terminate: func() error {
			return nil
		},
	}
}

// openWorkerFanoutBenchmarkPool opens one fixed worker pool used by fanout dispatch benchmarks.
func openWorkerFanoutBenchmarkPool(parseB *testing.B) WorkerPool {
	parseB.Helper()
	parseWorkerIndex := 0
	parsePool, parseErr := OpenWorkerPool(context.Background(), WorkerPoolOptions{
		Size:       getWorkerFanoutBenchmarkWorkerCount,
		QueueLimit: 0,
		OpenWorker: func(parseCtx context.Context) (Worker, error) {
			parseWorker := buildWorkerFanoutBenchmarkWorker(parseWorkerIndex)
			parseWorkerIndex++
			return parseWorker, nil
		},
	})
	if parseErr != nil {
		parseB.Fatalf("open worker fanout benchmark pool: %v", parseErr)
	}
	return parsePool
}

// runWorkerFanoutBenchmarkBatch executes one concurrent 8-request fanout and fails the benchmark on any request error.
func runWorkerFanoutBenchmarkBatch(parseB *testing.B, parseCtx context.Context, parseRequest func(context.Context, int) error) {
	parseB.Helper()
	parseErrorValues := make([]error, getWorkerFanoutBenchmarkWorkerCount)
	var parseWait sync.WaitGroup
	parseWait.Add(getWorkerFanoutBenchmarkWorkerCount)
	for parseProbeIndex := range getWorkerFanoutBenchmarkWorkerCount {
		go func(parseProbeOffset int) {
			defer parseWait.Done()
			parseErrorValues[parseProbeOffset] = parseRequest(parseCtx, parseProbeOffset+1)
		}(parseProbeIndex)
	}
	parseWait.Wait()
	for _, parseErr := range parseErrorValues {
		if parseErr != nil {
			parseB.Fatalf("run worker fanout benchmark batch: %v", parseErr)
		}
	}
}

// BenchmarkRequestWorkerDecodedFanoutDispatch compares direct worker-lane fanout against pooled worker fanout dispatch overhead.
func BenchmarkRequestWorkerDecodedFanoutDispatch(parseB *testing.B) {
	parseB.ReportAllocs()
	parseCtx := context.Background()

	parseB.Run("direct-lanes", func(parseB *testing.B) {
		parseWorkers := make([]Worker, getWorkerFanoutBenchmarkWorkerCount)
		for parseWorkerIndex := range getWorkerFanoutBenchmarkWorkerCount {
			parseWorkers[parseWorkerIndex] = buildWorkerFanoutBenchmarkWorker(parseWorkerIndex)
		}
		parseB.ResetTimer()
		for parseIteration := 0; parseIteration < parseB.N; parseIteration++ {
			runWorkerFanoutBenchmarkBatch(parseB, parseCtx, func(parseRequestCtx context.Context, parseProbe int) error {
				_, parseErr := RequestWorkerDecoded[workerFanoutBenchmarkRequest, struct{}, workerFanoutBenchmarkResult](
					parseRequestCtx,
					parseWorkers[parseProbe-1],
					"runtime2-status-probe",
					workerFanoutBenchmarkRequest{GetProbe: parseProbe},
					nil,
				)
				return parseErr
			})
		}
	})

	parseB.Run("worker-pool", func(parseB *testing.B) {
		parsePool := openWorkerFanoutBenchmarkPool(parseB)
		parseB.Cleanup(func() {
			_ = parsePool.Close()
		})
		parseB.ResetTimer()
		for parseIteration := 0; parseIteration < parseB.N; parseIteration++ {
			runWorkerFanoutBenchmarkBatch(parseB, parseCtx, func(parseRequestCtx context.Context, parseProbe int) error {
				_, parseErr := RequestWorkerDecoded[workerFanoutBenchmarkRequest, struct{}, workerFanoutBenchmarkResult](
					parseRequestCtx,
					parsePool,
					"runtime2-status-probe",
					workerFanoutBenchmarkRequest{GetProbe: parseProbe},
					nil,
				)
				return parseErr
			})
		}
	})
}
