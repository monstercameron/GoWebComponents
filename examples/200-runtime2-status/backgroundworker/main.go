//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/interop"
)

const (
	getRuntime2StatusWorkerRequestName  = "runtime2-status-probe"
	getRuntime2StatusWorkerFallbackName = "runtime2-status-worker"
)

type runtime2StatusWorkerRequest struct {
	RegionID   string `json:"regionId"`
	Count      int    `json:"count"`
	Probe      int    `json:"probe"`
	Tone       string `json:"tone"`
	WorkScale  int    `json:"workScale"`
	GetTraceID string `json:"traceId"`
}

type runtime2StatusWorkerResult struct {
	Worker            string `json:"worker"`
	RegionID          string `json:"regionId"`
	Count             int    `json:"count"`
	Probe             int    `json:"probe"`
	Tone              string `json:"tone"`
	Summary           string `json:"summary"`
	GetWorkIterations int    `json:"workIterations"`
	GetWorkDigest     uint64 `json:"workDigest"`
	GetWorkDurationMS int64  `json:"workDurationMs"`
	GetTraceID        string `json:"traceId"`
}

// main registers the runtime2 status worker scope and handles request messages.
func main() {
	parseScope, parseErr := interop.GetWorkerScope()
	if parseErr != nil {
		panic(parseErr)
	}
	if _, parseErr2 := parseScope.Subscribe(func(parseMessage interop.WorkerMessage, parseMessageErr error) {
		if parseMessageErr != nil {
			return
		}
		go handleRuntime2StatusWorkerMessage(parseScope, parseMessage)
	}); parseErr2 != nil {
		panic(parseErr2)
	}
	if parseErr3 := parseScope.Ready("bootstrap"); parseErr3 != nil {
		panic(parseErr3)
	}
	fmt.Printf("[runtime2-status-worker/runtime2] worker ready name=%s request=%s\n", getRuntime2StatusWorkerName(), getRuntime2StatusWorkerRequestName)
	select {}
}

// handleRuntime2StatusWorkerMessage routes worker messages to the probe handler.
func handleRuntime2StatusWorkerMessage(parseScope interop.WorkerScope, parseMessage interop.WorkerMessage) {
	if strings.TrimSpace(parseMessage.Phase) != "request" {
		return
	}

	parseRequestName := strings.TrimSpace(parseMessage.Name)
	if parseRequestName == "" {
		parseRequestName = getRuntime2StatusWorkerRequestName
	}

	switch parseRequestName {
	case getRuntime2StatusWorkerRequestName:
		handleRuntime2StatusProbeRequest(parseScope, parseMessage)
	default:
		fmt.Printf("[runtime2-status-worker/runtime2][warn] unknown request id=%s name=%s\n", parseMessage.ID, parseRequestName)
		_ = parseScope.Error(parseMessage.ID, parseRequestName, "unknown runtime2 status request", runtime2StatusWorkerResult{
			Worker:  getRuntime2StatusWorkerName(),
			Summary: "unknown runtime2 status request",
		})
	}
}

// handleRuntime2StatusProbeRequest decodes and fulfills one runtime2 status probe request.
func handleRuntime2StatusProbeRequest(parseScope interop.WorkerScope, parseMessage interop.WorkerMessage) {
	var parseRequest runtime2StatusWorkerRequest
	if parseErr := interop.Decode(parseMessage.Payload, &parseRequest); parseErr != nil {
		fmt.Printf("[runtime2-status-worker/runtime2][error] decode failed id=%s error=%v\n", parseMessage.ID, parseErr)
		_ = parseScope.Error(parseMessage.ID, getRuntime2StatusWorkerRequestName, parseErr.Error(), runtime2StatusWorkerResult{
			Worker:  getRuntime2StatusWorkerName(),
			Summary: parseErr.Error(),
		})
		return
	}

	parseWorkerName := getRuntime2StatusWorkerName()
	parseWorkIterations, parseWorkDigest, parseWorkDuration := buildRuntime2StatusWorkerLoad(parseRequest.Count, parseRequest.Probe, parseRequest.WorkScale)
	parseSummary := buildRuntime2StatusWorkerSummary(parseWorkerName, parseRequest, parseWorkIterations, parseWorkDuration, parseWorkDigest)
	_ = parseScope.Result(parseMessage.ID, getRuntime2StatusWorkerRequestName, runtime2StatusWorkerResult{
		Worker:            parseWorkerName,
		RegionID:          strings.TrimSpace(parseRequest.RegionID),
		Count:             parseRequest.Count,
		Probe:             parseRequest.Probe,
		Tone:              strings.TrimSpace(parseRequest.Tone),
		Summary:           parseSummary,
		GetWorkIterations: parseWorkIterations,
		GetWorkDigest:     parseWorkDigest,
		GetWorkDurationMS: parseWorkDuration.Milliseconds(),
		GetTraceID:        strings.TrimSpace(parseRequest.GetTraceID),
	})
	fmt.Printf("[runtime2-status-worker/runtime2] probe complete id=%s trace=%s worker=%s summary=%s\n", parseMessage.ID, strings.TrimSpace(parseRequest.GetTraceID), parseWorkerName, parseSummary)
}

// buildRuntime2StatusWorkerSummary builds the probe summary returned to the main thread.
func buildRuntime2StatusWorkerSummary(parseWorkerName string, parseRequest runtime2StatusWorkerRequest, parseWorkIterations int, parseWorkDuration time.Duration, parseWorkDigest uint64) string {
	parseTone := strings.TrimSpace(parseRequest.Tone)
	if parseTone == "" {
		parseTone = "idle"
	}
	return fmt.Sprintf(
		"%s handled probe %d for %s at count %d (%s), offloaded iterations=%d in %s digest=%d",
		parseWorkerName,
		parseRequest.Probe,
		strings.TrimSpace(parseRequest.RegionID),
		parseRequest.Count,
		parseTone,
		parseWorkIterations,
		parseWorkDuration,
		parseWorkDigest,
	)
}

// buildRuntime2StatusWorkerLoad runs one deterministic CPU-bound workload on the worker and returns iterations, digest, and duration.
func buildRuntime2StatusWorkerLoad(parseCount int, parseProbe int, parseWorkScale int) (int, uint64, time.Duration) {
	parseIterations := buildRuntime2StatusWorkerIterationCount(parseCount, parseProbe, parseWorkScale)
	parseStartedAt := time.Now()
	parseDigest := buildRuntime2StatusWorkerDigest(parseCount, parseProbe, parseIterations)
	return parseIterations, parseDigest, time.Since(parseStartedAt)
}

// buildRuntime2StatusWorkerIterationCount derives a bounded deterministic iteration count for one probe workload.
func buildRuntime2StatusWorkerIterationCount(parseCount int, parseProbe int, parseWorkScale int) int {
	parseAbsoluteCount := parseCount
	if parseAbsoluteCount < 0 {
		parseAbsoluteCount = -parseAbsoluteCount
	}
	parseScale := parseWorkScale
	if parseScale < 1 {
		parseScale = 1
	}
	parseIterations := 250000 + (parseAbsoluteCount * 2500) + (parseProbe * 60000)
	parseIterations *= parseScale
	if parseIterations > 5000000 {
		parseIterations = 5000000
	}
	return parseIterations
}

// buildRuntime2StatusWorkerDigest computes one deterministic digest used to prove worker-side CPU work happened.
func buildRuntime2StatusWorkerDigest(parseCount int, parseProbe int, parseIterations int) uint64 {
	parseSeed := uint64(parseProbe+1)*2654435761 + uint64(parseCount+257)*11400714819323198485
	for parseStep := 0; parseStep < parseIterations; parseStep++ {
		parseSeed ^= parseSeed << 13
		parseSeed ^= parseSeed >> 7
		parseSeed ^= parseSeed << 17
		parseSeed += uint64(parseStep*97 + parseProbe*131 + parseCount*17)
	}
	return parseSeed
}

// getRuntime2StatusWorkerName resolves the worker label from the browser worker scope.
func getRuntime2StatusWorkerName() string {
	parseGlobal, parseErr := interop.GetGlobalThis()
	if parseErr != nil {
		return getRuntime2StatusWorkerFallbackName
	}

	parseName := parseGlobal.Get("name")
	if parseName.IsUndefined() || parseName.IsNull() {
		return getRuntime2StatusWorkerFallbackName
	}

	parseWorkerName := strings.TrimSpace(parseName.String())
	if parseWorkerName == "" {
		return getRuntime2StatusWorkerFallbackName
	}
	return parseWorkerName
}
