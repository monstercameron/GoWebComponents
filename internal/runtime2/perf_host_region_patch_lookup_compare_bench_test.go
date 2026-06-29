package runtime2

import "testing"

var storeHostRegionPatchLookupBenchmarkSink HostRegionWorkerOutputResult

// buildHostRegionPatchLookupBenchFixture stores one mounted host-region fixture plus alternating large-region patch streams.
type buildHostRegionPatchLookupBenchFixture struct {
	getHostRegionAdapter *HostRegionAdapter
	getPatchStreamAToB   PatchStreamRaw
	getPatchStreamBToA   PatchStreamRaw
}

// buildHostRegionPatchLookupBenchmarkFixture builds one repeatable large-region patch-commit workload for lookup-cache benchmarks.
func buildHostRegionPatchLookupBenchmarkFixture(parseB *testing.B) buildHostRegionPatchLookupBenchFixture {
	parseB.Helper()
	buildHostRegionAdapter := buildPerfHotspotMountedHostRegionAdapter(parseB, RegionInstanceID("bench-region"))
	buildRenderOutputA := buildPerfHotspotKeyedListRenderOutput(2048, 0, "state-a")
	buildRenderOutputB := buildPerfHotspotKeyedListRenderOutput(2048, 0, "state-b")
	buildInitialIR, parseInitialIRErr := BuildCanonicalRenderIR(buildRenderOutputA)
	if parseInitialIRErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(initial) returned error: %v", parseInitialIRErr)
	}
	parseSeedRegionDOMIndexFromCanonical(parseB, buildHostRegionAdapter.GetHostRegionDOMIndex(), "bench-region", buildInitialIR)
	return buildHostRegionPatchLookupBenchFixture{
		getHostRegionAdapter: buildHostRegionAdapter,
		getPatchStreamAToB:   buildPerfHotspotPatchStream(parseB, buildRenderOutputA, buildRenderOutputB),
		getPatchStreamBToA:   buildPerfHotspotPatchStream(parseB, buildRenderOutputB, buildRenderOutputA),
	}
}

// buildHostRegionPatchLookupFilterBenchmarkFixture builds one repeatable filter-style patch workload that alternates remove-heavy and insert-heavy commits.
func buildHostRegionPatchLookupFilterBenchmarkFixture(parseB *testing.B) buildHostRegionPatchLookupBenchFixture {
	parseB.Helper()
	buildHostRegionAdapter := buildPerfHotspotMountedHostRegionAdapter(parseB, RegionInstanceID("bench-region"))
	buildRenderOutputA := buildPerfHotspotKeyedListRenderOutput(2048, 0, "state-a")
	buildRenderOutputB := buildPerfHotspotKeyedListRenderOutput(1024, 0, "state-a")
	buildInitialIR, parseInitialIRErr := BuildCanonicalRenderIR(buildRenderOutputA)
	if parseInitialIRErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(initial) returned error: %v", parseInitialIRErr)
	}
	parseSeedRegionDOMIndexFromCanonical(parseB, buildHostRegionAdapter.GetHostRegionDOMIndex(), "bench-region", buildInitialIR)
	return buildHostRegionPatchLookupBenchFixture{
		getHostRegionAdapter: buildHostRegionAdapter,
		getPatchStreamAToB:   buildPerfHotspotPatchStream(parseB, buildRenderOutputA, buildRenderOutputB),
		getPatchStreamBToA:   buildPerfHotspotPatchStream(parseB, buildRenderOutputB, buildRenderOutputA),
	}
}

// buildHostRegionPatchLookupAppendBenchmarkFixture builds one repeatable append-only patch workload that grows one keyed list without keyed moves or remove traffic.
func buildHostRegionPatchLookupAppendBenchmarkFixture(parseB *testing.B) buildHostRegionPatchLookupBenchFixture {
	parseB.Helper()
	buildHostRegionAdapter := buildPerfHotspotMountedHostRegionAdapter(parseB, RegionInstanceID("bench-region"))
	buildRenderOutputA := buildPerfHotspotKeyedListRenderOutput(256, 0, "state-a")
	buildRenderOutputB := buildPerfHotspotKeyedListRenderOutput(512, 0, "state-a")
	buildInitialIR, parseInitialIRErr := BuildCanonicalRenderIR(buildRenderOutputA)
	if parseInitialIRErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(initial append) returned error: %v", parseInitialIRErr)
	}
	parseSeedRegionDOMIndexFromCanonical(parseB, buildHostRegionAdapter.GetHostRegionDOMIndex(), "bench-region", buildInitialIR)
	return buildHostRegionPatchLookupBenchFixture{
		getHostRegionAdapter: buildHostRegionAdapter,
		getPatchStreamAToB:   buildPerfHotspotPatchStream(parseB, buildRenderOutputA, buildRenderOutputB),
	}
}

// buildHostRegionPatchLookupBenchmarkPatchStream clones one benchmark patch stream with fresh input and patch versions.
func buildHostRegionPatchLookupBenchmarkPatchStream(parsePatchStream PatchStreamRaw, parseVersion uint64) PatchStreamRaw {
	parsePatchStream.GetHeader.InputVersion = parseVersion
	parsePatchStream.GetHeader.PatchVersion = parseVersion
	return parsePatchStream
}

// handleHostRegionPatchCommitLegacyLookupRebuild preserves the previous host patch-commit path that rebuilt lookup maps from the DOM index every time.
func handleHostRegionPatchCommitLegacyLookupRebuild(
	parseHostRegionAdapter *HostRegionAdapter,
	parsePatchStream PatchStreamRaw,
) (HostRegionWorkerOutputResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionWorkerOutputResult{}, nil
	}
	parseHostRegionAdapter.clearHostRegionPatchLookupCache()
	return parseHostRegionAdapter.HandleHostRegionPatchCommit(parsePatchStream, nil)
}

// handleHostRegionPatchCommitLegacyLookupDeltaBuild preserves the previous insert-heavy host patch path that always precomputed one structural lookup delta before commit, even when the transaction could update the cache incrementally after commit.
func handleHostRegionPatchCommitLegacyLookupDeltaBuild(
	parseHostRegionAdapter *HostRegionAdapter,
	parsePatchStream PatchStreamRaw,
) (HostRegionWorkerOutputResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionWorkerOutputResult{}, nil
	}
	parseDOMCommitter := BuildDOMCommitter(parseHostRegionAdapter.storeRegionDOMIndexHandle)
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return HostRegionWorkerOutputResult{}, nil
	}
	buildRegionID := string(parseHostRegionAdapter.storeRegionInstanceID)
	hasPatchKeyedMoveOp := parseHasPatchKeyedMoveOp(parsePatchStream.GetOps)
	buildKnownNodeIDs, buildSiblingCountByParent := parseHostRegionAdapter.getHostRegionPatchLookupState(hasPatchKeyedMoveOp)
	parsePatchResult, hasPatchApply, parsePatchErr := ParsePatchStreamTransactionWithKeyedMoveHint(
		parsePatchStream,
		buildRegionID,
		getCoordinatorEntry.Epoch,
		buildKnownNodeIDs,
		buildSiblingCountByParent,
		parseHostRegionAdapter.storeHostRegionPatchIdempotency,
		hasPatchKeyedMoveOp,
	)
	if parsePatchErr != nil {
		return HostRegionWorkerOutputResult{}, parsePatchErr
	}
	if !hasPatchApply {
		return HostRegionWorkerOutputResult{HasIgnored: true}, nil
	}
	getPatchReadyResult, parsePatchReadyErr := parseHostRegionAdapter.HandleHostRegionPatchReadyWithVersion(
		parsePatchResult.GetHeader.PatchVersion,
		parsePatchResult.GetHeader.InputVersion,
	)
	if parsePatchReadyErr != nil {
		return HostRegionWorkerOutputResult{}, parsePatchReadyErr
	}
	if getPatchReadyResult.HasIgnored {
		return HostRegionWorkerOutputResult{HasIgnored: true}, nil
	}
	getLookupDelta, hasLookupDelta := parseHostRegionAdapter.buildHostRegionPatchLookupDelta(parsePatchResult.GetTransaction)
	_, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(parsePatchResult.GetTransaction)
	if parseTransactionErr != nil {
		return HostRegionWorkerOutputResult{}, parseTransactionErr
	}
	if hasLookupDelta {
		if !parseHostRegionAdapter.applyHostRegionPatchLookupDelta(getLookupDelta) {
			parseHostRegionAdapter.clearHostRegionPatchLookupCache()
		}
	} else if !parseHostRegionAdapter.applyHostRegionPatchLookupTransaction(parsePatchResult.GetTransaction) {
		parseHostRegionAdapter.clearHostRegionPatchLookupCache()
	}
	parseWorkerOutputResult, parseWorkerOutputErr := parseHostRegionAdapter.parseHandleHostRegionWorkerOutputCommit(parsePatchResult.GetHeader.InputVersion)
	if parseWorkerOutputErr != nil {
		return HostRegionWorkerOutputResult{}, parseWorkerOutputErr
	}
	if parseWorkerOutputResult.HasCommitted {
		parseHostRegionAdapter.storeHostRegionLastPatchVersion = parsePatchResult.GetHeader.PatchVersion
	}
	return parseWorkerOutputResult, nil
}

// BenchmarkHandleHostRegionPatchCommitCurrentVsLegacyLookupRebuild compares cached patch lookup reuse against the previous rebuild-every-commit path.
func BenchmarkHandleHostRegionPatchCommitCurrentVsLegacyLookupRebuild(parseB *testing.B) {
	parseB.Run("legacy_rebuild_lookup_every_commit", func(parseB *testing.B) {
		getFixture := buildHostRegionPatchLookupBenchmarkFixture(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getPatchVersion := uint64(parseIndex + 2)
			getPatchStream := buildHostRegionPatchLookupBenchmarkPatchStream(getFixture.getPatchStreamAToB, getPatchVersion)
			if parseIndex%2 == 1 {
				getPatchStream = buildHostRegionPatchLookupBenchmarkPatchStream(getFixture.getPatchStreamBToA, getPatchVersion)
			}
			getCommitResult, parseCommitErr := handleHostRegionPatchCommitLegacyLookupRebuild(getFixture.getHostRegionAdapter, getPatchStream)
			if parseCommitErr != nil {
				parseB.Fatalf("handleHostRegionPatchCommitLegacyLookupRebuild returned error: %v", parseCommitErr)
			}
			storeHostRegionPatchLookupBenchmarkSink = getCommitResult
		}
	})
	parseB.Run("current_reuse_lookup_cache", func(parseB *testing.B) {
		getFixture := buildHostRegionPatchLookupBenchmarkFixture(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getPatchVersion := uint64(parseIndex + 2)
			getPatchStream := buildHostRegionPatchLookupBenchmarkPatchStream(getFixture.getPatchStreamAToB, getPatchVersion)
			if parseIndex%2 == 1 {
				getPatchStream = buildHostRegionPatchLookupBenchmarkPatchStream(getFixture.getPatchStreamBToA, getPatchVersion)
			}
			getCommitResult, parseCommitErr := getFixture.getHostRegionAdapter.HandleHostRegionPatchCommit(getPatchStream, nil)
			if parseCommitErr != nil {
				parseB.Fatalf("HandleHostRegionPatchCommit returned error: %v", parseCommitErr)
			}
			storeHostRegionPatchLookupBenchmarkSink = getCommitResult
		}
	})
}

// BenchmarkHandleHostRegionPatchCommitAppendOnlyCurrentVsLegacyLookupDelta compares the new append-only host lookup-cache fast path against the previous path that still built one structural lookup delta before commit.
func BenchmarkHandleHostRegionPatchCommitAppendOnlyCurrentVsLegacyLookupDelta(parseB *testing.B) {
	parseB.Run("legacy_build_delta_before_commit", func(parseB *testing.B) {
		parseB.ReportAllocs()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseB.StopTimer()
			getFixture := buildHostRegionPatchLookupAppendBenchmarkFixture(parseB)
			getPatchStream := buildHostRegionPatchLookupBenchmarkPatchStream(getFixture.getPatchStreamAToB, 2)
			parseB.StartTimer()
			getCommitResult, parseCommitErr := handleHostRegionPatchCommitLegacyLookupDeltaBuild(getFixture.getHostRegionAdapter, getPatchStream)
			if parseCommitErr != nil {
				parseB.Fatalf("handleHostRegionPatchCommitLegacyLookupDeltaBuild returned error: %v", parseCommitErr)
			}
			storeHostRegionPatchLookupBenchmarkSink = getCommitResult
		}
	})
	parseB.Run("current_skip_delta_for_insert_only", func(parseB *testing.B) {
		parseB.ReportAllocs()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseB.StopTimer()
			getFixture := buildHostRegionPatchLookupAppendBenchmarkFixture(parseB)
			getPatchStream := buildHostRegionPatchLookupBenchmarkPatchStream(getFixture.getPatchStreamAToB, 2)
			parseB.StartTimer()
			getCommitResult, parseCommitErr := getFixture.getHostRegionAdapter.HandleHostRegionPatchCommit(getPatchStream, nil)
			if parseCommitErr != nil {
				parseB.Fatalf("HandleHostRegionPatchCommit returned error: %v", parseCommitErr)
			}
			storeHostRegionPatchLookupBenchmarkSink = getCommitResult
		}
	})
}

// BenchmarkHandleHostRegionPatchCommitFilterCurrentVsLegacyLookupRebuild compares remove-heavy patch lookup reuse against the previous rebuild-every-commit path.
func BenchmarkHandleHostRegionPatchCommitFilterCurrentVsLegacyLookupRebuild(parseB *testing.B) {
	parseB.Run("legacy_rebuild_lookup_every_commit", func(parseB *testing.B) {
		getFixture := buildHostRegionPatchLookupFilterBenchmarkFixture(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getPatchVersion := uint64(parseIndex + 2)
			getPatchStream := buildHostRegionPatchLookupBenchmarkPatchStream(getFixture.getPatchStreamAToB, getPatchVersion)
			if parseIndex%2 == 1 {
				getPatchStream = buildHostRegionPatchLookupBenchmarkPatchStream(getFixture.getPatchStreamBToA, getPatchVersion)
			}
			getCommitResult, parseCommitErr := handleHostRegionPatchCommitLegacyLookupRebuild(getFixture.getHostRegionAdapter, getPatchStream)
			if parseCommitErr != nil {
				parseB.Fatalf("handleHostRegionPatchCommitLegacyLookupRebuild returned error: %v", parseCommitErr)
			}
			storeHostRegionPatchLookupBenchmarkSink = getCommitResult
		}
	})
	parseB.Run("current_reuse_lookup_cache", func(parseB *testing.B) {
		getFixture := buildHostRegionPatchLookupFilterBenchmarkFixture(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getPatchVersion := uint64(parseIndex + 2)
			getPatchStream := buildHostRegionPatchLookupBenchmarkPatchStream(getFixture.getPatchStreamAToB, getPatchVersion)
			if parseIndex%2 == 1 {
				getPatchStream = buildHostRegionPatchLookupBenchmarkPatchStream(getFixture.getPatchStreamBToA, getPatchVersion)
			}
			getCommitResult, parseCommitErr := getFixture.getHostRegionAdapter.HandleHostRegionPatchCommit(getPatchStream, nil)
			if parseCommitErr != nil {
				parseB.Fatalf("HandleHostRegionPatchCommit returned error: %v", parseCommitErr)
			}
			storeHostRegionPatchLookupBenchmarkSink = getCommitResult
		}
	})
}
