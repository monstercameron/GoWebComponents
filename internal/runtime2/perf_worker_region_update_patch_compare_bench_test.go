package runtime2

import "testing"

var storeWorkerRegionUpdatePatchCompareBenchmarkSink PatchStreamRaw

// buildWorkerRegionUpdatePatchCompareIRPair builds one single-text IR pair used by worker update patch compare benchmarks.
func buildWorkerRegionUpdatePatchCompareIRPair() (CanonicalRenderIR, CanonicalRenderIR, error) {
	parsePreviousIR, parsePreviousIRErr := BuildCanonicalRenderIR(map[string]any{
		"kind": "text",
		"text": "before",
	})
	if parsePreviousIRErr != nil {
		return CanonicalRenderIR{}, CanonicalRenderIR{}, parsePreviousIRErr
	}
	parseNextIR, parseNextIRErr := BuildCanonicalRenderIR(map[string]any{
		"kind": "text",
		"text": "after",
	})
	if parseNextIRErr != nil {
		return CanonicalRenderIR{}, CanonicalRenderIR{}, parseNextIRErr
	}
	return parsePreviousIR, parseNextIR, nil
}

// BenchmarkBuildWorkerRegionUpdatePatchCurrentVsLegacy compares the worker update patch fast path against legacy canonical patch generation.
func BenchmarkBuildWorkerRegionUpdatePatchCurrentVsLegacy(parseB *testing.B) {
	parsePreviousIR, parseNextIR, parseIRErr := buildWorkerRegionUpdatePatchCompareIRPair()
	if parseIRErr != nil {
		parseB.Fatalf("buildWorkerRegionUpdatePatchCompareIRPair returned error: %v", parseIRErr)
	}
	parseB.Run("legacy_build_canonical_patch_stream", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parsePatch, parseNoOp, parseErr := BuildCanonicalPatchStream(
				"region-bench",
				1,
				2,
				uint64(parseIndex+2),
				parsePreviousIR,
				parseNextIR,
			)
			if parseErr != nil {
				parseB.Fatalf("BuildCanonicalPatchStream returned error: %v", parseErr)
			}
			if parseNoOp {
				parseB.Fatal("BuildCanonicalPatchStream unexpectedly returned no-op for changed text")
			}
			storeWorkerRegionUpdatePatchCompareBenchmarkSink = parsePatch
		}
	})
	parseB.Run("current_worker_update_patch_builder", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parsePatch, parseNoOp, parseErr := buildWorkerRegionUpdatePatch(
				"region-bench",
				1,
				2,
				uint64(parseIndex+2),
				parsePreviousIR,
				parseNextIR,
			)
			if parseErr != nil {
				parseB.Fatalf("buildWorkerRegionUpdatePatch returned error: %v", parseErr)
			}
			if parseNoOp {
				parseB.Fatal("buildWorkerRegionUpdatePatch unexpectedly returned no-op for changed text")
			}
			storeWorkerRegionUpdatePatchCompareBenchmarkSink = parsePatch
		}
	})
}
