package runtime2

import (
	"fmt"
	"sync"
	"testing"
)

// buildCoordinatorGetUpdateLegacyState stores the previous value-backed coordinator map used by compare benchmarks.
type buildCoordinatorGetUpdateLegacyState struct {
	storeMu      sync.RWMutex
	storeEntries map[RegionInstanceID]CoordinatorEntry
}

// buildCoordinatorGetUpdateBench builds one mounted coordinator used by GetEntry and UpdateRegion compare benchmarks.
func buildCoordinatorGetUpdateBench(parseB *testing.B) *Coordinator {
	parseB.Helper()
	getCoordinator := BuildCoordinator()
	if parseMountErr := getCoordinator.MountRegion(CoordinatorEntry{
		RegionInstanceID: RegionInstanceID("region-1"),
		RendererID:       RendererID("dashboard.hot-panel"),
		Epoch:            1,
		SourceIDs:        []string{"count", "status"},
	}); parseMountErr != nil {
		parseB.Fatalf("MountRegion returned error: %v", parseMountErr)
	}
	return getCoordinator
}

// buildCoordinatorGetUpdateLegacyBench builds one value-backed coordinator state used by legacy compare benchmarks.
func buildCoordinatorGetUpdateLegacyBench(parseB *testing.B) *buildCoordinatorGetUpdateLegacyState {
	parseB.Helper()
	return &buildCoordinatorGetUpdateLegacyState{
		storeEntries: map[RegionInstanceID]CoordinatorEntry{
			RegionInstanceID("region-1"): {
				RegionInstanceID: RegionInstanceID("region-1"),
				RendererID:       RendererID("dashboard.hot-panel"),
				Epoch:            1,
				SourceIDs:        []string{"count", "status"},
				CurrentState:     CoordinatorStateMounted,
			},
		},
	}
}

// getCoordinatorEntryLegacy reads one coordinator entry using the previous defer-based lock path.
func getCoordinatorEntryLegacy(parseCoordinator *buildCoordinatorGetUpdateLegacyState, parseRegionInstanceID RegionInstanceID) (CoordinatorEntry, bool) {
	if parseCoordinator == nil {
		return CoordinatorEntry{}, false
	}
	parseCoordinator.storeMu.RLock()
	defer parseCoordinator.storeMu.RUnlock()
	parseEntry, parseHasEntry := parseCoordinator.storeEntries[parseRegionInstanceID]
	if !parseHasEntry {
		return CoordinatorEntry{}, false
	}
	return parseEntry, true
}

// updateCoordinatorRegionLegacy records one dispatched version using the previous UpdateRegion forwarding path.
func updateCoordinatorRegionLegacy(parseCoordinator *buildCoordinatorGetUpdateLegacyState, parseRegionInstanceID RegionInstanceID, parseInputVersion uint64) error {
	_, parseErr := updateCoordinatorRegionAndGetEntryLegacy(parseCoordinator, parseRegionInstanceID, parseInputVersion)
	return parseErr
}

// updateCoordinatorRegionAndGetEntryLegacy records one dispatched version using the previous defer-based UpdateRegionAndGetEntry path.
func updateCoordinatorRegionAndGetEntryLegacy(
	parseCoordinator *buildCoordinatorGetUpdateLegacyState,
	parseRegionInstanceID RegionInstanceID,
	parseInputVersion uint64,
) (CoordinatorEntry, error) {
	if parseCoordinator == nil {
		return CoordinatorEntry{}, nil
	}
	if _, parseErr := ParseRegionInstanceID(string(parseRegionInstanceID)); parseErr != nil {
		return CoordinatorEntry{}, parseErr
	}
	parseCoordinator.storeMu.Lock()
	defer parseCoordinator.storeMu.Unlock()
	parseEntry, parseHasEntry := parseCoordinator.storeEntries[parseRegionInstanceID]
	if !parseHasEntry {
		return CoordinatorEntry{}, fmt.Errorf("runtime2: coordinator entry %q is not mounted", parseRegionInstanceID)
	}
	if parseEntry.IsFallback {
		return CoordinatorEntry{}, fmt.Errorf("runtime2: coordinator entry %q is in fallback mode", parseRegionInstanceID)
	}
	if parseErr := ValidateMonotonicInputVersion(parseEntry.LastDispatchedVersion, parseInputVersion); parseErr != nil {
		return CoordinatorEntry{}, parseErr
	}
	parseEntry.LastDispatchedVersion = parseInputVersion
	parseEntry.CurrentState = CoordinatorStateActive
	parseCoordinator.storeEntries[parseRegionInstanceID] = parseEntry
	return parseEntry, nil
}

// BenchmarkCoordinatorGetEntryCurrentVsLegacy compares the current GetEntry hot path against the previous defer-based read lock path.
func BenchmarkCoordinatorGetEntryCurrentVsLegacy(parseB *testing.B) {
	parseB.Run("legacy_defer_read_lock", func(parseB *testing.B) {
		getCoordinator := buildCoordinatorGetUpdateLegacyBench(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseHasEntry := getCoordinatorEntryLegacy(getCoordinator, RegionInstanceID("region-1")); !parseHasEntry {
				parseB.Fatal("expected mounted coordinator entry")
			}
		}
	})

	parseB.Run("current_inline_read_unlock", func(parseB *testing.B) {
		getCoordinator := buildCoordinatorGetUpdateBench(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseHasEntry := getCoordinator.GetEntry(RegionInstanceID("region-1")); !parseHasEntry {
				parseB.Fatal("expected mounted coordinator entry")
			}
		}
	})
}

// BenchmarkCoordinatorUpdateRegionCurrentVsLegacy compares the direct UpdateRegion hot path against the previous forwarding UpdateRegion path.
func BenchmarkCoordinatorUpdateRegionCurrentVsLegacy(parseB *testing.B) {
	parseB.Run("legacy_forward_to_update_and_get_entry", func(parseB *testing.B) {
		getCoordinator := buildCoordinatorGetUpdateLegacyBench(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if parseErr := updateCoordinatorRegionLegacy(getCoordinator, RegionInstanceID("region-1"), uint64(parseIndex+1)); parseErr != nil {
				parseB.Fatalf("updateCoordinatorRegionLegacy returned error: %v", parseErr)
			}
		}
	})

	parseB.Run("current_direct_update", func(parseB *testing.B) {
		getCoordinator := buildCoordinatorGetUpdateBench(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if parseErr := getCoordinator.UpdateRegion(RegionInstanceID("region-1"), uint64(parseIndex+1)); parseErr != nil {
				parseB.Fatalf("UpdateRegion returned error: %v", parseErr)
			}
		}
	})
}

// BenchmarkCoordinatorGetMutableEntryValidatedVsTrusted compares region-ID validated reads against trusted-ID reads.
func BenchmarkCoordinatorGetMutableEntryValidatedVsTrusted(parseB *testing.B) {
	parseB.Run("validated_get_mutable_entry", func(parseB *testing.B) {
		getCoordinator := buildCoordinatorGetUpdateBench(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := getCoordinator.getMutableEntry(RegionInstanceID("region-1")); parseErr != nil {
				parseB.Fatalf("getMutableEntry returned error: %v", parseErr)
			}
		}
	})

	parseB.Run("trusted_get_mutable_entry", func(parseB *testing.B) {
		getCoordinator := buildCoordinatorGetUpdateBench(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := getCoordinator.getMutableEntryTrusted(RegionInstanceID("region-1")); parseErr != nil {
				parseB.Fatalf("getMutableEntryTrusted returned error: %v", parseErr)
			}
		}
	})
}
