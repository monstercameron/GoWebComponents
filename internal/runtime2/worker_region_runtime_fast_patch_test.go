package runtime2

import (
	"reflect"
	"testing"
)

// TestBuildWorkerRegionUpdatePatchSingleTextMatchesCanonicalBuilder verifies the worker update text fast path preserves canonical patch output.
func TestBuildWorkerRegionUpdatePatchSingleTextMatchesCanonicalBuilder(parseTesting *testing.T) {
	parsePreviousIR, parsePreviousIRErr := BuildCanonicalRenderIR(map[string]any{
		"kind": "text",
		"text": "before",
	})
	if parsePreviousIRErr != nil {
		parseTesting.Fatalf("BuildCanonicalRenderIR(previous text) returned error: %v", parsePreviousIRErr)
	}
	parseNextIR, parseNextIRErr := BuildCanonicalRenderIR(map[string]any{
		"kind": "text",
		"text": "after",
	})
	if parseNextIRErr != nil {
		parseTesting.Fatalf("BuildCanonicalRenderIR(next text) returned error: %v", parseNextIRErr)
	}
	parseLegacyPatch, parseLegacyNoOp, parseLegacyErr := BuildCanonicalPatchStream(
		"region-fast-path",
		1,
		2,
		2,
		parsePreviousIR,
		parseNextIR,
	)
	if parseLegacyErr != nil {
		parseTesting.Fatalf("BuildCanonicalPatchStream returned error: %v", parseLegacyErr)
	}
	parseCurrentPatch, parseCurrentNoOp, parseCurrentErr := buildWorkerRegionUpdatePatch(
		"region-fast-path",
		1,
		2,
		2,
		parsePreviousIR,
		parseNextIR,
	)
	if parseCurrentErr != nil {
		parseTesting.Fatalf("buildWorkerRegionUpdatePatch returned error: %v", parseCurrentErr)
	}
	if parseLegacyNoOp != parseCurrentNoOp {
		parseTesting.Fatalf("buildWorkerRegionUpdatePatch no-op = %t, want %t", parseCurrentNoOp, parseLegacyNoOp)
	}
	if !reflect.DeepEqual(parseCurrentPatch, parseLegacyPatch) {
		parseTesting.Fatalf("buildWorkerRegionUpdatePatch patch mismatch\ncurrent=%+v\nlegacy=%+v", parseCurrentPatch, parseLegacyPatch)
	}
}

// TestBuildWorkerRegionUpdatePatchFallbackMatchesCanonicalBuilder verifies non-text updates continue to use canonical patch behavior.
func TestBuildWorkerRegionUpdatePatchFallbackMatchesCanonicalBuilder(parseTesting *testing.T) {
	parsePreviousIR, parsePreviousIRErr := BuildCanonicalRenderIR(map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"props": map[string]any{
			"class": "before",
		},
	})
	if parsePreviousIRErr != nil {
		parseTesting.Fatalf("BuildCanonicalRenderIR(previous host) returned error: %v", parsePreviousIRErr)
	}
	parseNextIR, parseNextIRErr := BuildCanonicalRenderIR(map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"props": map[string]any{
			"class": "after",
		},
	})
	if parseNextIRErr != nil {
		parseTesting.Fatalf("BuildCanonicalRenderIR(next host) returned error: %v", parseNextIRErr)
	}
	parseLegacyPatch, parseLegacyNoOp, parseLegacyErr := BuildCanonicalPatchStream(
		"region-fallback-path",
		1,
		2,
		2,
		parsePreviousIR,
		parseNextIR,
	)
	if parseLegacyErr != nil {
		parseTesting.Fatalf("BuildCanonicalPatchStream returned error: %v", parseLegacyErr)
	}
	parseCurrentPatch, parseCurrentNoOp, parseCurrentErr := buildWorkerRegionUpdatePatch(
		"region-fallback-path",
		1,
		2,
		2,
		parsePreviousIR,
		parseNextIR,
	)
	if parseCurrentErr != nil {
		parseTesting.Fatalf("buildWorkerRegionUpdatePatch returned error: %v", parseCurrentErr)
	}
	if parseLegacyNoOp != parseCurrentNoOp {
		parseTesting.Fatalf("buildWorkerRegionUpdatePatch no-op = %t, want %t", parseCurrentNoOp, parseLegacyNoOp)
	}
	if !reflect.DeepEqual(parseCurrentPatch, parseLegacyPatch) {
		parseTesting.Fatalf("buildWorkerRegionUpdatePatch patch mismatch\ncurrent=%+v\nlegacy=%+v", parseCurrentPatch, parseLegacyPatch)
	}
}
