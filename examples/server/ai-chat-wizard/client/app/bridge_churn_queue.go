//go:build js && wasm

package app

import (
	"sync"
	"time"
)

type bridgeRPCMetrics struct {
	parseMutex sync.Mutex
	parseCount map[string]int
}

func (parseMetrics *bridgeRPCMetrics) record(parseFamily bridgeRPCFamily, parseEvent string) {
	parseMetrics.parseMutex.Lock()
	defer parseMetrics.parseMutex.Unlock()
	if parseMetrics.parseCount == nil {
		parseMetrics.parseCount = map[string]int{}
	}
	parseMetrics.parseCount[string(parseFamily)+":"+parseEvent]++
}

func (parseMetrics *bridgeRPCMetrics) snapshot() map[string]int {
	parseMetrics.parseMutex.Lock()
	defer parseMetrics.parseMutex.Unlock()
	parseSnapshot := make(map[string]int, len(parseMetrics.parseCount))
	for parseKey, parseValue := range parseMetrics.parseCount {
		parseSnapshot[parseKey] = parseValue
	}
	return parseSnapshot
}

type bridgeDeferredLogWork struct {
	Level      string
	Scope      string
	Message    string
	Fields     map[string]any
	EnqueuedAt time.Time
	ExpiresAt  time.Time
}

type bridgeDeferredLogQueue struct {
	parseMutex sync.Mutex
	parseItems []bridgeDeferredLogWork
	parseMax   int
	parseTTL   time.Duration
}

func parseNewBridgeDeferredLogQueue(parseMax int, parseTTL time.Duration) *bridgeDeferredLogQueue {
	if parseMax < 1 {
		parseMax = 1
	}
	if parseTTL <= 0 {
		parseTTL = 15 * time.Second
	}
	return &bridgeDeferredLogQueue{parseMax: parseMax, parseTTL: parseTTL}
}

func (parseQueue *bridgeDeferredLogQueue) enqueue(parseWork bridgeDeferredLogWork, parseNow time.Time) bool {
	parseQueue.parseMutex.Lock()
	defer parseQueue.parseMutex.Unlock()
	parseQueue.dropExpiredLocked(parseNow)
	if len(parseQueue.parseItems) >= parseQueue.parseMax {
		parseQueue.parseItems = parseQueue.parseItems[1:]
		parseBridgeChurnMetrics.record(bridgeRPCTelemetryRelay, "dropped")
	}
	parseWork.EnqueuedAt = parseNow
	parseWork.ExpiresAt = parseNow.Add(parseQueue.parseTTL)
	parseQueue.parseItems = append(parseQueue.parseItems, parseWork)
	parseBridgeChurnMetrics.record(bridgeRPCTelemetryRelay, "deferred")
	return true
}

func (parseQueue *bridgeDeferredLogQueue) drain(parseNow time.Time, parseLimit int) []bridgeDeferredLogWork {
	parseQueue.parseMutex.Lock()
	defer parseQueue.parseMutex.Unlock()
	parseQueue.dropExpiredLocked(parseNow)
	if parseLimit <= 0 || parseLimit > len(parseQueue.parseItems) {
		parseLimit = len(parseQueue.parseItems)
	}
	parseDrained := append([]bridgeDeferredLogWork(nil), parseQueue.parseItems[:parseLimit]...)
	parseQueue.parseItems = append([]bridgeDeferredLogWork(nil), parseQueue.parseItems[parseLimit:]...)
	return parseDrained
}

func (parseQueue *bridgeDeferredLogQueue) len(parseNow time.Time) int {
	parseQueue.parseMutex.Lock()
	defer parseQueue.parseMutex.Unlock()
	parseQueue.dropExpiredLocked(parseNow)
	return len(parseQueue.parseItems)
}

func (parseQueue *bridgeDeferredLogQueue) dropExpiredLocked(parseNow time.Time) {
	if len(parseQueue.parseItems) == 0 {
		return
	}
	parseKept := parseQueue.parseItems[:0]
	for _, parseItem := range parseQueue.parseItems {
		if !parseItem.ExpiresAt.IsZero() && parseNow.After(parseItem.ExpiresAt) {
			parseBridgeChurnMetrics.record(bridgeRPCTelemetryRelay, "dropped")
			continue
		}
		parseKept = append(parseKept, parseItem)
	}
	parseQueue.parseItems = parseKept
}

var parseBridgeChurnMetrics = &bridgeRPCMetrics{}
var parseClientLogDeferredQueue = parseNewBridgeDeferredLogQueue(64, 30*time.Second)

func parseSnapshotBridgeChurnMetrics() map[string]int {
	return parseBridgeChurnMetrics.snapshot()
}
