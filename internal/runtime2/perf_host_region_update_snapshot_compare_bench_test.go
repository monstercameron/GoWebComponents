package runtime2

import (
	"fmt"
	"testing"
)

// buildRuntime2LegacyHostRegionUpdateSnapshot preserves the previous multi-pass source snapshot capture path for benchmark comparison.
func buildRuntime2LegacyHostRegionUpdateSnapshot(
	parseHostRegionAdapter *HostRegionAdapter,
	parseSpec ParallelRegionSpec,
	parseInputVersion uint64,
	parseShouldStoreSnapshotVersion bool,
) (SnapshotEnvelope, bool, []string, error) {
	if parseHostRegionAdapter == nil {
		return SnapshotEnvelope{}, false, nil, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseInputVersion == 0 {
		return SnapshotEnvelope{}, false, nil, fmt.Errorf("runtime2: host update snapshot input version is required")
	}
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return SnapshotEnvelope{}, false, nil, fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	getSpec := parseSpec
	hasExactSourceIDs := parseHasHostRegionExactSourceIDList(parseSpec.SourceIDs, getCoordinatorEntry.SourceIDs)
	if parseSpec.RegionInstanceID == parseHostRegionAdapter.storeRegionInstanceID &&
		parseSpec.RendererID == getCoordinatorEntry.RendererID &&
		hasExactSourceIDs {
		if parsePropsErr := ValidateSerializableProps(parseSpec.Props); parsePropsErr != nil {
			return SnapshotEnvelope{}, false, nil, parsePropsErr
		}
	} else {
		getNormalizedSpec, parseSpecErr := NormalizeParallelRegionSpec(parseSpec)
		if parseSpecErr != nil {
			return SnapshotEnvelope{}, false, nil, parseSpecErr
		}
		getSpec = getNormalizedSpec
		if getSpec.RegionInstanceID != parseHostRegionAdapter.storeRegionInstanceID {
			return SnapshotEnvelope{}, false, nil, fmt.Errorf(
				"runtime2: host region adapter mounted for %q cannot capture snapshot for region %q",
				parseHostRegionAdapter.storeRegionInstanceID,
				getSpec.RegionInstanceID,
			)
		}
		if getCoordinatorEntry.RendererID != getSpec.RendererID {
			return SnapshotEnvelope{}, false, nil, fmt.Errorf(
				"runtime2: mounted renderer ID %q does not match snapshot renderer ID %q",
				getCoordinatorEntry.RendererID,
				getSpec.RendererID,
			)
		}
		hasExactSourceIDs = parseHasHostRegionExactSourceIDList(getSpec.SourceIDs, getCoordinatorEntry.SourceIDs)
	}
	if len(getSpec.SourceIDs) == 0 {
		getSnapshotEnvelope := SnapshotEnvelope{
			RegionInstanceID: getSpec.RegionInstanceID,
			Epoch:            getCoordinatorEntry.Epoch,
			InputVersion:     parseInputVersion,
			Props:            getSpec.Props,
		}
		if parseShouldStoreSnapshotVersion {
			if _, parseSnapshotStateErr := parseHostRegionAdapter.storeCoordinator.SetRegionSnapshotState(
				getSpec.RegionInstanceID,
				parseInputVersion,
				getSpec.SourceIDs,
				!hasExactSourceIDs,
			); parseSnapshotStateErr != nil {
				return SnapshotEnvelope{}, false, nil, parseSnapshotStateErr
			}
		}
		return getSnapshotEnvelope, hasExactSourceIDs, getSpec.SourceIDs, nil
	}
	if parseHostRegionAdapter.storeHostRegionSourceLookup == nil {
		return SnapshotEnvelope{}, false, nil, fmt.Errorf("runtime2: host source lookup is not configured")
	}
	getSourceValues, getSourceVersions, parseSourceLookupErr := parseHostRegionAdapter.storeHostRegionSourceLookup(getSpec.SourceIDs)
	if parseSourceLookupErr != nil {
		return SnapshotEnvelope{}, false, nil, parseSourceLookupErr
	}
	buildSourceValues, parseSourceValuesErr := buildSnapshotSourceValues(getSpec.SourceIDs, getSourceValues)
	if parseSourceValuesErr != nil {
		return SnapshotEnvelope{}, false, nil, parseSourceValuesErr
	}
	buildSourceVersions := make(map[string]uint64, len(getSpec.SourceIDs))
	for _, getSourceID := range getSpec.SourceIDs {
		getSourceVersion, hasSourceVersion := getSourceVersions[getSourceID]
		if !hasSourceVersion {
			continue
		}
		buildSourceVersions[getSourceID] = getSourceVersion
	}
	getSnapshotEnvelope, parseSnapshotEnvelopeErr := buildSnapshotEnvelopeFromNormalizedSourceSnapshotWithoutValidation(
		getSpec.RegionInstanceID,
		getCoordinatorEntry.Epoch,
		parseInputVersion,
		getSpec.Props,
		getSpec.SourceIDs,
		buildSourceValues,
		buildSourceVersions,
	)
	if parseSnapshotEnvelopeErr != nil {
		return SnapshotEnvelope{}, false, nil, parseSnapshotEnvelopeErr
	}
	if parseShouldStoreSnapshotVersion {
		if _, parseSnapshotStateErr := parseHostRegionAdapter.storeCoordinator.SetRegionSnapshotState(
			getSpec.RegionInstanceID,
			parseInputVersion,
			getSpec.SourceIDs,
			!hasExactSourceIDs,
		); parseSnapshotStateErr != nil {
			return SnapshotEnvelope{}, false, nil, parseSnapshotStateErr
		}
	}
	return getSnapshotEnvelope, hasExactSourceIDs, getSpec.SourceIDs, nil
}

// BenchmarkHandleHostRegionUpdateSnapshotCurrentVsLegacy compares the current snapshot capture path against the previous multi-pass source snapshot flow.
func BenchmarkHandleHostRegionUpdateSnapshotCurrentVsLegacy(parseB *testing.B) {
	buildSpec := ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-bench"),
		Props: map[string]any{
			"title": "Orders",
			"tick":  0,
		},
		SourceIDs: []string{"count", "status"},
	}
	buildAdapter := func() *HostRegionAdapter {
		parseHostRegionAdapter, parseBuildErr := BuildHostRegionAdapter(
			RegionInstanceID("region-bench"),
			[]SchedulerShardID{"shard-a"},
		)
		if parseBuildErr != nil {
			parseB.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
		}
		_, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(ParallelRegionSpec{
			RendererID:       RendererID("dashboard.hot-panel"),
			RegionInstanceID: RegionInstanceID("region-bench"),
		}, 3)
		if parseMountErr != nil {
			parseB.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
		}
		parseLookupErr := parseHostRegionAdapter.SetHostRegionSourceLookup(
			func(parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
				return map[string]any{
						"count":  5,
						"status": "healthy",
					},
					map[string]uint64{
						"count":  9,
						"status": 9,
					},
					nil
			},
		)
		if parseLookupErr != nil {
			parseB.Fatalf("SetHostRegionSourceLookup returned error: %v", parseLookupErr)
		}
		return parseHostRegionAdapter
	}
	parseB.Run("current-changed-props", func(parseB *testing.B) {
		parseHostRegionAdapter := buildAdapter()
		parseProps := buildSpec.Props.(map[string]any)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseProps["tick"] = parseIndex
			if _, _, _, parseErr := parseHostRegionAdapter.handleHostRegionUpdateSnapshot(buildSpec, uint64(parseIndex+1), true); parseErr != nil {
				parseB.Fatalf("handleHostRegionUpdateSnapshot returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("legacy-changed-props", func(parseB *testing.B) {
		parseHostRegionAdapter := buildAdapter()
		parseProps := buildSpec.Props.(map[string]any)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseProps["tick"] = parseIndex
			if _, _, _, parseErr := buildRuntime2LegacyHostRegionUpdateSnapshot(parseHostRegionAdapter, buildSpec, uint64(parseIndex+1), true); parseErr != nil {
				parseB.Fatalf("buildRuntime2LegacyHostRegionUpdateSnapshot returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("current-stable-props", func(parseB *testing.B) {
		parseHostRegionAdapter := buildAdapter()
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, _, _, parseErr := parseHostRegionAdapter.handleHostRegionUpdateSnapshot(buildSpec, uint64(parseIndex+1), true); parseErr != nil {
				parseB.Fatalf("handleHostRegionUpdateSnapshot returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("legacy-stable-props", func(parseB *testing.B) {
		parseHostRegionAdapter := buildAdapter()
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, _, _, parseErr := buildRuntime2LegacyHostRegionUpdateSnapshot(parseHostRegionAdapter, buildSpec, uint64(parseIndex+1), true); parseErr != nil {
				parseB.Fatalf("buildRuntime2LegacyHostRegionUpdateSnapshot returned error: %v", parseErr)
			}
		}
	})
}
