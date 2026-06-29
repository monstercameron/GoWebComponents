//go:build js && wasm

package ui

import (
	"encoding/json"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// storeHydrationStartupCost captures startup-cost attribution for hydration profiling snapshots.
func storeHydrationStartupCost(storePayload SSRBootstrap) {
	storeAttribution := runtime.StartupCostAttribution{}
	if storeEncoded, storeErr := marshalSSRBootstrapJSON(storePayload); storeErr == nil {
		storeAttribution.BootstrapDecodedBytes = int64(len(storeEncoded))
	}
	storeAttribution.InitialRouteDataBytes = buildHydrationRouteDataBytes(storePayload)
	storeTransferBytes, storeDecodedBytes, storeCacheWarmupNs, storeServiceWorkerNs := buildHydrationWASMAttribution()
	storeAttribution.WASMTransferBytes = storeTransferBytes
	storeAttribution.WASMDecodedBytes = storeDecodedBytes
	storeAttribution.CacheWarmupDurationNs = storeCacheWarmupNs
	storeAttribution.ServiceWorkerOverheadNs = storeServiceWorkerNs
	runtime.StoreStartupCostAttribution(storeAttribution)
}

// buildHydrationRouteDataBytes estimates the route-scoped payload bytes for the startup route.
func buildHydrationRouteDataBytes(buildPayload SSRBootstrap) int64 {
	if len(buildPayload.Data) == 0 {
		return 0
	}
	buildRoutePath := strings.TrimSpace(buildPayload.Route.Path)
	if buildRoutePath == "" {
		buildRoutePath = "/"
	}
	buildPrefix := buildRoutePath + "::"
	var buildTotal int64
	for buildKey, buildValue := range buildPayload.Data {
		if !strings.HasPrefix(strings.TrimSpace(buildKey), buildPrefix) {
			continue
		}
		if buildBytes, buildErr := json.Marshal(buildValue); buildErr == nil {
			buildTotal += int64(len(buildBytes))
		}
	}
	return buildTotal
}

// buildHydrationWASMAttribution reads browser resource timing entries for wasm startup cost attribution.
func buildHydrationWASMAttribution() (int64, int64, int64, int64) {
	buildPerformance := js.Global().Get("performance")
	if !buildPerformance.Truthy() {
		return 0, 0, 0, 0
	}
	buildEntries := buildPerformance.Call("getEntriesByType", "resource")
	if !buildEntries.Truthy() {
		return 0, 0, 0, 0
	}
	var buildTransferBytes int64
	var buildDecodedBytes int64
	var buildCacheWarmupNs int64
	var buildServiceWorkerNs int64
	buildLength := buildEntries.Length()
	for buildIndex := 0; buildIndex < buildLength; buildIndex++ {
		buildEntry := buildEntries.Index(buildIndex)
		buildName := strings.ToLower(strings.TrimSpace(buildEntry.Get("name").String()))
		if !strings.Contains(buildName, ".wasm") {
			continue
		}
		buildTransfer := int64(buildEntry.Get("transferSize").Float())
		buildDecoded := int64(buildEntry.Get("decodedBodySize").Float())
		buildEncoded := int64(buildEntry.Get("encodedBodySize").Float())
		if buildTransfer <= 0 {
			buildTransfer = buildEncoded
		}
		if buildTransfer > buildTransferBytes {
			buildTransferBytes = buildTransfer
		}
		if buildDecoded > buildDecodedBytes {
			buildDecodedBytes = buildDecoded
		}

		buildStart := buildEntry.Get("startTime").Float()
		buildFetchStart := buildEntry.Get("fetchStart").Float()
		buildWorkerStart := buildEntry.Get("workerStart").Float()
		if buildFetchStart > buildStart {
			buildCandidate := int64((buildFetchStart - buildStart) * 1_000_000)
			if buildCandidate > buildCacheWarmupNs {
				buildCacheWarmupNs = buildCandidate
			}
		}
		if buildWorkerStart > 0 && buildFetchStart >= buildWorkerStart {
			buildCandidate := int64((buildFetchStart - buildWorkerStart) * 1_000_000)
			if buildCandidate > buildServiceWorkerNs {
				buildServiceWorkerNs = buildCandidate
			}
		}
	}
	return buildTransferBytes, buildDecodedBytes, buildCacheWarmupNs, buildServiceWorkerNs
}
