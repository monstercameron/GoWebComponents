//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"

	benchmarkshared "github.com/monstercameron/GoWebComponents/examples/201-render-benchmark/shared"
	"github.com/monstercameron/GoWebComponents/interop"
)

const getBenchmarkWorkerFallbackName = "render-benchmark-worker"

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

// handleBenchmarkWorkerCoreChunk decodes and fulfills one core-list chunk request.
func handleBenchmarkWorkerCoreChunk(parseScope interop.WorkerScope, parseMessage interop.WorkerMessage) {
	var getRequest benchmarkshared.BenchmarkWorkerCoreChunkRequest
	if parseDecodeErr := interop.Decode(parseMessage.Payload, &getRequest); parseDecodeErr != nil {
		_ = parseScope.Error(parseMessage.ID, benchmarkshared.BenchmarkWorkerRequestCoreChunk, parseDecodeErr.Error(), map[string]any{
			"worker": getBenchmarkWorkerName(),
		})
		return
	}
	getResult := benchmarkshared.BuildBenchmarkWorkerCoreChunkResult(getBenchmarkWorkerName(), getRequest)
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
	getResult := benchmarkshared.BuildBenchmarkWorkerContentChunkResult(getBenchmarkWorkerName(), getRequest)
	_ = parseScope.Result(parseMessage.ID, benchmarkshared.BenchmarkWorkerRequestContentChunk, getResult)
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
