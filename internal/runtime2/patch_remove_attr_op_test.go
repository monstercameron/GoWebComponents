package runtime2

import (
	"fmt"
	"testing"
)

var storePatchRemoveAttrOpBenchmarkSink PatchRemoveAttrOp

// TestParsePatchRemoveAttrOpValidRemoveAttrOpDecodes verifies valid remove-attr payloads decode successfully.
func TestParsePatchRemoveAttrOpValidRemoveAttrOpDecodes(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		8: {},
	}
	parseRemovedAttrKeys := map[patchRemovedAttrKey]struct{}{}
	buildStringTable := BuildRenderStringTable([]string{"class"})
	getClassRef, _ := buildStringTable.GetRenderStringRef("class")
	parseRawOp := PatchRemoveAttrOpRaw{
		TargetNodeID: 8,
		KeyRef:       getClassRef,
	}
	parseOp, parseErr := ParsePatchRemoveAttrOp(parseRawOp, parseKnownNodeIDs, buildStringTable, parseRemovedAttrKeys)
	if parseErr != nil {
		parseTesting.Fatalf("ParsePatchRemoveAttrOp(valid) error = %v", parseErr)
	}
	if parseOp.Key != "class" {
		parseTesting.Fatalf("ParsePatchRemoveAttrOp(valid) key = %q, want %q", parseOp.Key, "class")
	}
}

// TestParsePatchRemoveAttrOpUnknownAttrReferenceFails verifies invalid attr references are rejected.
func TestParsePatchRemoveAttrOpUnknownAttrReferenceFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		8: {},
	}
	parseRemovedAttrKeys := map[patchRemovedAttrKey]struct{}{}
	buildStringTable := BuildRenderStringTable([]string{"class"})
	parseRawOp := PatchRemoveAttrOpRaw{
		TargetNodeID: 8,
		KeyRef:       77,
	}
	_, parseErr := ParsePatchRemoveAttrOp(parseRawOp, parseKnownNodeIDs, buildStringTable, parseRemovedAttrKeys)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchRemoveAttrOp(invalid key ref) error = nil, want error")
	}
}

// TestParsePatchRemoveAttrOpDuplicateRemoveBehavesConsistently verifies duplicate attr removes are rejected consistently.
func TestParsePatchRemoveAttrOpDuplicateRemoveBehavesConsistently(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		8: {},
	}
	parseRemovedAttrKeys := map[patchRemovedAttrKey]struct{}{}
	buildStringTable := BuildRenderStringTable([]string{"class"})
	getClassRef, _ := buildStringTable.GetRenderStringRef("class")
	parseRawOp := PatchRemoveAttrOpRaw{
		TargetNodeID: 8,
		KeyRef:       getClassRef,
	}
	_, parseErr := ParsePatchRemoveAttrOp(parseRawOp, parseKnownNodeIDs, buildStringTable, parseRemovedAttrKeys)
	if parseErr != nil {
		parseTesting.Fatalf("ParsePatchRemoveAttrOp(first remove) error = %v", parseErr)
	}
	_, parseErr = ParsePatchRemoveAttrOp(parseRawOp, parseKnownNodeIDs, buildStringTable, parseRemovedAttrKeys)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchRemoveAttrOp(duplicate remove) error = nil, want error")
	}
}

// BenchmarkParsePatchRemoveAttrOpCurrentVsLegacy compares the typed remove-attr dedupe key against the previous formatted-string key.
func BenchmarkParsePatchRemoveAttrOpCurrentVsLegacy(parseBenchmark *testing.B) {
	parseKnownNodeIDs := map[uint64]struct{}{
		8: {},
	}
	parseStringTable := BuildRenderStringTable([]string{"class"})
	parseClassRef, _ := parseStringTable.GetRenderStringRef("class")
	parseRawOp := PatchRemoveAttrOpRaw{
		TargetNodeID: 8,
		KeyRef:       parseClassRef,
	}
	parseBenchmark.Run("legacy", func(parseLegacyBenchmark *testing.B) {
		parseRemovedAttrKeys := make(map[string]struct{}, 1)
		parseLegacyBenchmark.ReportAllocs()
		parseLegacyBenchmark.ResetTimer()
		for parseIndex := 0; parseLegacyBenchmark.Loop(); parseIndex++ {
			clear(parseRemovedAttrKeys)
			parseOp, parseErr := parsePatchRemoveAttrOpLegacyBenchmark(parseRawOp, parseKnownNodeIDs, parseStringTable, parseRemovedAttrKeys)
			if parseErr != nil {
				parseLegacyBenchmark.Fatalf("parsePatchRemoveAttrOpLegacyBenchmark() error = %v", parseErr)
			}
			storePatchRemoveAttrOpBenchmarkSink = parseOp
		}
	})
	parseBenchmark.Run("current", func(parseCurrentBenchmark *testing.B) {
		parseRemovedAttrKeys := make(map[patchRemovedAttrKey]struct{}, 1)
		parseCurrentBenchmark.ReportAllocs()
		parseCurrentBenchmark.ResetTimer()
		for parseIndex := 0; parseCurrentBenchmark.Loop(); parseIndex++ {
			clear(parseRemovedAttrKeys)
			parseOp, parseErr := ParsePatchRemoveAttrOp(parseRawOp, parseKnownNodeIDs, parseStringTable, parseRemovedAttrKeys)
			if parseErr != nil {
				parseCurrentBenchmark.Fatalf("ParsePatchRemoveAttrOp() error = %v", parseErr)
			}
			storePatchRemoveAttrOpBenchmarkSink = parseOp
		}
	})
}

// parsePatchRemoveAttrOpLegacyBenchmark preserves the previous string-key remove-attr dedupe path for benchmark comparison.
func parsePatchRemoveAttrOpLegacyBenchmark(parseRaw PatchRemoveAttrOpRaw, parseKnownNodeIDs map[uint64]struct{}, parseStringTable RenderStringTable, parseRemovedAttrKeys map[string]struct{}) (PatchRemoveAttrOp, error) {
	if parseRaw.TargetNodeID == 0 {
		return PatchRemoveAttrOp{}, fmt.Errorf("runtime2: remove-attr op target node id is required")
	}
	if _, hasKnownNodeID := parseKnownNodeIDs[parseRaw.TargetNodeID]; !hasKnownNodeID {
		return PatchRemoveAttrOp{}, fmt.Errorf("runtime2: remove-attr op target node id %d is unknown", parseRaw.TargetNodeID)
	}
	parseKey, parseKeyErr := parseStringTable.GetRenderStringByRef(parseRaw.KeyRef)
	if parseKeyErr != nil {
		return PatchRemoveAttrOp{}, fmt.Errorf("runtime2: remove-attr op key reference %d is invalid: %w", parseRaw.KeyRef, parseKeyErr)
	}
	parseAttrIdentifier := fmt.Sprintf("%d:%s", parseRaw.TargetNodeID, parseKey)
	if _, hasRemovedAttrKey := parseRemovedAttrKeys[parseAttrIdentifier]; hasRemovedAttrKey {
		return PatchRemoveAttrOp{}, fmt.Errorf("runtime2: remove-attr op key %q for node id %d was already removed", parseKey, parseRaw.TargetNodeID)
	}
	parseRemovedAttrKeys[parseAttrIdentifier] = struct{}{}
	return PatchRemoveAttrOp{
		TargetNodeID: parseRaw.TargetNodeID,
		Key:          parseKey,
	}, nil
}
