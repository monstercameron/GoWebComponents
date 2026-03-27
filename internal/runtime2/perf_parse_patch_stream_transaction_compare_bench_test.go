package runtime2

import "testing"

var storePatchStreamParseResultSink PatchStreamParseResult

// buildPatchStreamParseKeyedMoveBenchFixture stores one parse benchmark fixture that exercises keyed-move sibling-bounds validation with a large sibling-count map.
type buildPatchStreamParseKeyedMoveBenchFixture struct {
	getPatchStream          PatchStreamRaw
	getKnownNodeIDs         map[uint64]struct{}
	getSiblingCountByParent map[uint64]uint32
}

// buildPatchStreamParseKeyedMoveBenchmarkFixture builds one repeatable parse workload with one keyed-move op and a large sibling-count lookup map.
func buildPatchStreamParseKeyedMoveBenchmarkFixture() buildPatchStreamParseKeyedMoveBenchFixture {
	const (
		getBenchParentCount      = 1024
		getBenchParentBaseNodeID = uint64(10_000)
		getBenchSourceNodeID     = uint64(999_001)
		getBenchParentNodeID     = getBenchParentBaseNodeID + 513
	)
	buildSiblingCountByParent := make(map[uint64]uint32, getBenchParentCount)
	for parseParentOffset := 0; parseParentOffset < getBenchParentCount; parseParentOffset++ {
		getParentNodeID := getBenchParentBaseNodeID + uint64(parseParentOffset)
		buildSiblingCountByParent[getParentNodeID] = 32
	}
	buildKnownNodeIDs := map[uint64]struct{}{
		getBenchParentNodeID: {},
		getBenchSourceNodeID: {},
	}
	buildPatchStream := PatchStreamRaw{
		GetHeader: PatchStreamHeaderRaw{
			ProtocolVersion: PatchStreamProtocolVersion,
			RegionID:        "bench-region",
			Epoch:           1,
			InputVersion:    2,
			PatchVersion:    2,
		},
		GetOps: []PatchStreamOpRaw{
			{
				GetOpCode: uint8(PatchOpCodeMoveKeyedChild),
				GetKeyedMoveOp: &PatchKeyedMoveOpRaw{
					ParentNodeID:     getBenchParentNodeID,
					SourceNodeID:     getBenchSourceNodeID,
					DestinationIndex: 7,
				},
			},
		},
		GetPatchIdentity: "bench-parse-keyed-move",
	}
	return buildPatchStreamParseKeyedMoveBenchFixture{
		getPatchStream:          buildPatchStream,
		getKnownNodeIDs:         buildKnownNodeIDs,
		getSiblingCountByParent: buildSiblingCountByParent,
	}
}

// buildParsePatchStreamTransactionLegacySiblingCopy preserves the previous keyed-move parse behavior that copied sibling-count maps before validation.
func buildParsePatchStreamTransactionLegacySiblingCopy(
	parseRaw PatchStreamRaw,
	parseExpectedRegionID string,
	parseExpectedEpoch uint64,
	parseKnownNodeIDs map[uint64]struct{},
	parseSiblingCountByParent map[uint64]uint32,
) (PatchStreamParseResult, bool, error) {
	buildMutableSiblingCountByParent := buildPatchStreamSiblingCountScratchMap()
	for getParentNodeID, getSiblingCount := range parseSiblingCountByParent {
		buildMutableSiblingCountByParent[getParentNodeID] = getSiblingCount
	}
	defer func() {
		clearPatchStreamSiblingCountScratchMap(buildMutableSiblingCountByParent)
		storePatchStreamSiblingCountScratchPool.Put(buildMutableSiblingCountByParent)
	}()
	return ParsePatchStreamTransactionWithKeyedMoveHint(
		parseRaw,
		parseExpectedRegionID,
		parseExpectedEpoch,
		parseKnownNodeIDs,
		buildMutableSiblingCountByParent,
		nil,
		true,
	)
}

// BenchmarkParsePatchStreamTransactionCurrentVsLegacySiblingCopy compares keyed-move parse validation against the previous sibling-count copy path.
func BenchmarkParsePatchStreamTransactionCurrentVsLegacySiblingCopy(parseB *testing.B) {
	getFixture := buildPatchStreamParseKeyedMoveBenchmarkFixture()
	parseB.Run("legacy_sibling_copy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getParseResult, hasApply, parseParseErr := buildParsePatchStreamTransactionLegacySiblingCopy(
				getFixture.getPatchStream,
				"bench-region",
				1,
				getFixture.getKnownNodeIDs,
				getFixture.getSiblingCountByParent,
			)
			if parseParseErr != nil {
				parseB.Fatalf("buildParsePatchStreamTransactionLegacySiblingCopy returned error: %v", parseParseErr)
			}
			if !hasApply {
				parseB.Fatal("buildParsePatchStreamTransactionLegacySiblingCopy unexpectedly ignored patch")
			}
			storePatchStreamParseResultSink = getParseResult
		}
	})
	parseB.Run("current_read_only_sibling_lookup", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getParseResult, hasApply, parseParseErr := ParsePatchStreamTransactionWithKeyedMoveHint(
				getFixture.getPatchStream,
				"bench-region",
				1,
				getFixture.getKnownNodeIDs,
				getFixture.getSiblingCountByParent,
				nil,
				true,
			)
			if parseParseErr != nil {
				parseB.Fatalf("ParsePatchStreamTransactionWithKeyedMoveHint returned error: %v", parseParseErr)
			}
			if !hasApply {
				parseB.Fatal("ParsePatchStreamTransactionWithKeyedMoveHint unexpectedly ignored patch")
			}
			storePatchStreamParseResultSink = getParseResult
		}
	})
}
