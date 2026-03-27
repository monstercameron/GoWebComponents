package runtime2

import "testing"

func parseBuildAgent3BenchRenderOutput(parseText string, parseClass string) map[string]any {
	return map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"props": map[string]any{
			"class": parseClass,
		},
		"children": []any{
			map[string]any{
				"kind": "host-element",
				"tag":  "ul",
				"children": []any{
					map[string]any{"kind": "host-element", "tag": "li", "key": "a"},
					map[string]any{"kind": "host-element", "tag": "li", "key": "b"},
					map[string]any{"kind": "text", "text": parseText},
				},
			},
		},
	}
}

// BenchmarkBuildCanonicalRenderIR benchmarks worker-side canonical IR build.
func BenchmarkBuildCanonicalRenderIR(parseB *testing.B) {
	parseRenderOutput := parseBuildAgent3BenchRenderOutput("hello", "active")
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, parseErr := BuildCanonicalRenderIR(parseRenderOutput); parseErr != nil {
			parseB.Fatalf("BuildCanonicalRenderIR returned error: %v", parseErr)
		}
	}
}

// BenchmarkBuildCanonicalPatchStream benchmarks worker-side canonical diff and patch generation.
func BenchmarkBuildCanonicalPatchStream(parseB *testing.B) {
	parsePreviousIR, parsePreviousErr := BuildCanonicalRenderIR(parseBuildAgent3BenchRenderOutput("before", "active"))
	if parsePreviousErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(previous) returned error: %v", parsePreviousErr)
	}
	parseNextIR, parseNextErr := BuildCanonicalRenderIR(parseBuildAgent3BenchRenderOutput("after", "idle"))
	if parseNextErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(next) returned error: %v", parseNextErr)
	}
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, _, parsePatchErr := BuildCanonicalPatchStream("bench-region", 1, uint64(parseIndex+2), uint64(parseIndex+2), parsePreviousIR, parseNextIR); parsePatchErr != nil {
			parseB.Fatalf("BuildCanonicalPatchStream returned error: %v", parsePatchErr)
		}
	}
}

// BenchmarkParsePatchStreamTransaction benchmarks patch-stream decode and validation.
func BenchmarkParsePatchStreamTransaction(parseB *testing.B) {
	parsePreviousIR, parsePreviousErr := BuildCanonicalRenderIR(parseBuildAgent3BenchRenderOutput("before", "active"))
	if parsePreviousErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(previous) returned error: %v", parsePreviousErr)
	}
	parseNextIR, parseNextErr := BuildCanonicalRenderIR(parseBuildAgent3BenchRenderOutput("after", "idle"))
	if parseNextErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(next) returned error: %v", parseNextErr)
	}
	parsePatchStream, hasNoOp, parsePatchErr := BuildCanonicalPatchStream("bench-region", 1, 2, 2, parsePreviousIR, parseNextIR)
	if parsePatchErr != nil {
		parseB.Fatalf("BuildCanonicalPatchStream returned error: %v", parsePatchErr)
	}
	if hasNoOp {
		parseB.Fatal("BuildCanonicalPatchStream returned no-op for differing payloads")
	}
	parsePreviousTree, parseTreeErr := ParseCanonicalRenderTree(parsePreviousIR)
	if parseTreeErr != nil {
		parseB.Fatalf("ParseCanonicalRenderTree returned error: %v", parseTreeErr)
	}
	parseKnownNodeIDs := make(map[uint64]struct{}, len(parsePreviousTree.getNodeByID))
	for getNodeID := range parsePreviousTree.getNodeByID {
		parseKnownNodeIDs[getNodeID] = struct{}{}
	}
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, _, parseParseErr := ParsePatchStreamTransaction(parsePatchStream, "bench-region", 1, parseKnownNodeIDs, map[uint64]uint32{}, nil); parseParseErr != nil {
			parseB.Fatalf("ParsePatchStreamTransaction returned error: %v", parseParseErr)
		}
	}
}

// BenchmarkCommitRegionPatchTransaction benchmarks patch transaction commit.
func BenchmarkCommitRegionPatchTransaction(parseB *testing.B) {
	parsePreviousIR, parsePreviousErr := BuildCanonicalRenderIR(parseBuildAgent3BenchRenderOutput("before", "active"))
	if parsePreviousErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(previous) returned error: %v", parsePreviousErr)
	}
	parseNextIR, parseNextErr := BuildCanonicalRenderIR(parseBuildAgent3BenchRenderOutput("after", "idle"))
	if parseNextErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(next) returned error: %v", parseNextErr)
	}
	parsePatchStream, hasNoOp, parsePatchErr := BuildCanonicalPatchStream("bench-region", 1, 2, 2, parsePreviousIR, parseNextIR)
	if parsePatchErr != nil {
		parseB.Fatalf("BuildCanonicalPatchStream returned error: %v", parsePatchErr)
	}
	if hasNoOp {
		parseB.Fatal("BuildCanonicalPatchStream returned no-op for differing payloads")
	}
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		parseDOMIndex := BuildRegionDOMIndex()
		parseSeedRegionDOMIndexFromCanonical(parseB, parseDOMIndex, "bench-region", parsePreviousIR)
		parseKnownNodeIDs, parseSiblingCountByParent := BuildRegionDOMPatchLookupMaps(parseDOMIndex, "bench-region")
		parseParseResult, hasApply, parseParseErr := ParsePatchStreamTransaction(parsePatchStream, "bench-region", 1, parseKnownNodeIDs, parseSiblingCountByParent, BuildPatchIdempotencyTracker())
		if parseParseErr != nil {
			parseB.Fatalf("ParsePatchStreamTransaction returned error: %v", parseParseErr)
		}
		if !hasApply {
			parseB.Fatal("ParsePatchStreamTransaction unexpectedly ignored first patch")
		}
		parseDOMCommitter := BuildDOMCommitter(parseDOMIndex)
		if _, parseCommitErr := parseDOMCommitter.CommitRegionPatchTransaction(parseParseResult.GetTransaction); parseCommitErr != nil {
			parseB.Fatalf("CommitRegionPatchTransaction returned error: %v", parseCommitErr)
		}
	}
}

// BenchmarkCompareLocalVsWorkerBackedRendering benchmarks local render-only conversion versus worker-backed diff production.
func BenchmarkCompareLocalVsWorkerBackedRendering(parseB *testing.B) {
	parseLocalRenderOutput := parseBuildAgent3BenchRenderOutput("local", "active")
	parsePreviousIR, parsePreviousErr := BuildCanonicalRenderIR(parseBuildAgent3BenchRenderOutput("before", "active"))
	if parsePreviousErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(previous) returned error: %v", parsePreviousErr)
	}
	parseNextIR, parseNextErr := BuildCanonicalRenderIR(parseBuildAgent3BenchRenderOutput("after", "idle"))
	if parseNextErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(next) returned error: %v", parseNextErr)
	}
	parseB.Run("local", func(parseB *testing.B) {
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := BuildCanonicalRenderIR(parseLocalRenderOutput); parseErr != nil {
				parseB.Fatalf("BuildCanonicalRenderIR(local) returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("worker-backed", func(parseB *testing.B) {
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, _, parseErr := BuildCanonicalPatchStream("bench-region", 1, uint64(parseIndex+2), uint64(parseIndex+2), parsePreviousIR, parseNextIR); parseErr != nil {
				parseB.Fatalf("BuildCanonicalPatchStream(worker-backed) returned error: %v", parseErr)
			}
		}
	})
}
