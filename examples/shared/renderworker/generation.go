package renderworker

import (
	"strings"
	"sync"
)

// RenderWorkerGenerationTracker tracks latest request generations for stale-drop checks.
type RenderWorkerGenerationTracker struct {
	storeMu                     sync.Mutex
	storeLatestGenerationByName map[string]uint64
}

// BuildRenderWorkerGenerationTracker builds one empty request-generation tracker.
func BuildRenderWorkerGenerationTracker() *RenderWorkerGenerationTracker {
	return &RenderWorkerGenerationTracker{
		storeLatestGenerationByName: map[string]uint64{},
	}
}

// ShouldRenderWorkerDropStaleRequest reports whether one generation is stale for the named request and updates tracker state.
func (parseTracker *RenderWorkerGenerationTracker) ShouldRenderWorkerDropStaleRequest(parseRequestName string, parseGeneration uint64) bool {
	if parseTracker == nil || parseGeneration == 0 {
		return false
	}
	parseRequestName = strings.TrimSpace(parseRequestName)
	if parseRequestName == "" {
		return false
	}
	parseTracker.storeMu.Lock()
	defer parseTracker.storeMu.Unlock()
	getLatestGeneration := parseTracker.storeLatestGenerationByName[parseRequestName]
	if parseGeneration < getLatestGeneration {
		return true
	}
	if parseGeneration > getLatestGeneration {
		parseTracker.storeLatestGenerationByName[parseRequestName] = parseGeneration
	}
	return false
}

// GetRenderWorkerLatestGeneration returns the latest tracked generation for one request name.
func (parseTracker *RenderWorkerGenerationTracker) GetRenderWorkerLatestGeneration(parseRequestName string) uint64 {
	if parseTracker == nil {
		return 0
	}
	parseRequestName = strings.TrimSpace(parseRequestName)
	if parseRequestName == "" {
		return 0
	}
	parseTracker.storeMu.Lock()
	defer parseTracker.storeMu.Unlock()
	return parseTracker.storeLatestGenerationByName[parseRequestName]
}
