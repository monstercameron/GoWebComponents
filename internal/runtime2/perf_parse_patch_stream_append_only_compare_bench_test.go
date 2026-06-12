package runtime2

import (
	"strconv"
	"testing"
)

var storePatchStreamAppendOnlyParseResultSink PatchStreamParseResult

// buildPatchStreamParseAppendOnlyBenchFixture stores one append-only parse benchmark fixture with a large known-node set and a large insert batch.
type buildPatchStreamParseAppendOnlyBenchFixture struct {
	getPatchStream  PatchStreamRaw
	getKnownNodeIDs map[uint64]struct{}
}

// buildPatchStreamParseAppendOnlyBenchmarkFixture builds one repeatable append-only insert patch stream that previously forced a full known-node copy before validation.
func buildPatchStreamParseAppendOnlyBenchmarkFixture(parseAppendCount int) buildPatchStreamParseAppendOnlyBenchFixture {
	if parseAppendCount < 2 {
		parseAppendCount = 2
	}
	const getKnownNodeCount = 4096
	buildKnownNodeIDs := make(map[uint64]struct{}, getKnownNodeCount)
	for parseNodeOffset := range getKnownNodeCount {
		buildKnownNodeIDs[uint64(parseNodeOffset+1)] = struct{}{}
	}
	buildStringTable := BuildRenderStringTable([]string{"li"})
	buildOps := make([]PatchStreamOpRaw, 0, parseAppendCount)
	for parseAppendOffset := 0; parseAppendOffset < parseAppendCount; parseAppendOffset++ {
		buildOps = append(buildOps, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeInsertNode),
			GetInsertOp: &PatchInsertOpRaw{
				ParentNodeID: 1,
				Node: RenderNodeRecordRaw{
					NodeID:  uint64(getKnownNodeCount + parseAppendOffset + 1),
					Kind:    uint8(RenderNodeKindHostElement),
					TextRef: 0,
				},
			},
		})
	}
	return buildPatchStreamParseAppendOnlyBenchFixture{
		getPatchStream: PatchStreamRaw{
			GetHeader: PatchStreamHeaderRaw{
				ProtocolVersion: PatchStreamProtocolVersion,
				RegionID:        "bench-region",
				Epoch:           1,
				InputVersion:    2,
				PatchVersion:    2,
			},
			GetStringTable:   buildStringTable.Entries,
			GetOps:           buildOps,
			GetPatchIdentity: "bench-append-only-parse",
		},
		getKnownNodeIDs: buildKnownNodeIDs,
	}
}

// buildParsePatchStreamTransactionLegacyAppendOnlyKnownNodeCopy preserves the previous append-only parse behavior that cloned the full known-node set before validating the insert batch.
func buildParsePatchStreamTransactionLegacyAppendOnlyKnownNodeCopy(
	parseRaw PatchStreamRaw,
	parseExpectedRegionID string,
	parseExpectedEpoch uint64,
	parseKnownNodeIDs map[uint64]struct{},
) (PatchStreamParseResult, bool, error) {
	parseHeader, parseHeaderErr := ParsePatchStreamHeader(parseRaw.GetHeader, parseExpectedRegionID)
	if parseHeaderErr != nil {
		return PatchStreamParseResult{}, false, parseHeaderErr
	}
	if parseExpectedEpoch > 0 && parseHeader.Epoch != parseExpectedEpoch {
		return PatchStreamParseResult{}, false, nil
	}
	var parseStringTable RenderStringTable
	if parseHasCanonicalStringTableSortedUnique(parseRaw.GetStringTable) {
		parseStringTable = RenderStringTable{
			Entries: parseRaw.GetStringTable,
		}
	} else {
		parseCanonicalStringTable, parseStringTableErr := ParseRenderStringTable(parseRaw.GetStringTable)
		if parseStringTableErr != nil {
			return PatchStreamParseResult{}, false, parseStringTableErr
		}
		parseStringTable = parseCanonicalStringTable
	}
	buildKnownNodeIDs := parseKnownNodeIDs
	hasCopiedKnownNodeIDs := false
	defer func() {
		if !hasCopiedKnownNodeIDs || buildKnownNodeIDs == nil {
			return
		}
		clearPatchStreamKnownNodeIDScratchMap(buildKnownNodeIDs)
		storePatchStreamKnownNodeIDScratchPool.Put(buildKnownNodeIDs)
	}()
	buildTransaction := RegionPatchTransaction{
		GetRegionID: parseHeader.RegionID,
		GetOps:      make([]RegionPatchOp, 0, len(parseRaw.GetOps)),
	}
	for parseOpIndex, getRawOp := range parseRaw.GetOps {
		if getRawOp.GetInsertOp == nil {
			return PatchStreamParseResult{}, false, nil
		}
		parseKnownNodeIDsForMutation := getPatchMutableKnownNodeIDs(
			parseKnownNodeIDs,
			&buildKnownNodeIDs,
			&hasCopiedKnownNodeIDs,
		)
		parseInsertOp, parseInsertErr := ParsePatchInsertOp(*getRawOp.GetInsertOp, parseKnownNodeIDsForMutation)
		if parseInsertErr != nil {
			return PatchStreamParseResult{}, false, parseInsertErr
		}
		buildInsertNode, parseInsertNodeErr := parseBuildRegionDOMNodeFromPatchRecord(parseInsertOp.Node, parseStringTable)
		if parseInsertNodeErr != nil {
			return PatchStreamParseResult{}, false, parseInsertNodeErr
		}
		buildTransaction.GetOps = append(buildTransaction.GetOps, RegionPatchOp{
			GetKind:         RegionPatchOpKindInsertNode,
			GetParentNodeID: parseInsertOp.ParentNodeID,
			GetBeforeNodeID: parseInsertOp.AnchorNodeID,
			GetInsertNode:   buildInsertNode,
		})
		parseKnownNodeIDsForMutation[parseInsertOp.Node.NodeID] = struct{}{}
		if parseOpIndex == len(parseRaw.GetOps)-1 {
			return PatchStreamParseResult{
				GetHeader:      parseHeader,
				GetTransaction: buildTransaction,
			}, true, nil
		}
	}
	return PatchStreamParseResult{
		GetHeader:      parseHeader,
		GetTransaction: buildTransaction,
	}, true, nil
}

// BenchmarkParsePatchStreamTransactionAppendOnlyCurrentVsLegacy compares the append-only parse fast path against the previous full known-node copy path.
func BenchmarkParsePatchStreamTransactionAppendOnlyCurrentVsLegacy(parseB *testing.B) {
	parseAppendCounts := []int{32, 128, 256}
	for _, parseAppendCount := range parseAppendCounts {
		parseBenchName := "append-" + strconv.Itoa(parseAppendCount)
		parseB.Run(parseBenchName+"/legacy_copy_known_nodes", func(parseB *testing.B) {
			getFixture := buildPatchStreamParseAppendOnlyBenchmarkFixture(parseAppendCount)
			parseB.ReportAllocs()
			parseB.ResetTimer()
			for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
				getParseResult, hasPatchApply, parseParseErr := buildParsePatchStreamTransactionLegacyAppendOnlyKnownNodeCopy(
					getFixture.getPatchStream,
					"bench-region",
					1,
					getFixture.getKnownNodeIDs,
				)
				if parseParseErr != nil {
					parseB.Fatalf("buildParsePatchStreamTransactionLegacyAppendOnlyKnownNodeCopy returned error: %v", parseParseErr)
				}
				if !hasPatchApply {
					parseB.Fatal("buildParsePatchStreamTransactionLegacyAppendOnlyKnownNodeCopy unexpectedly ignored patch")
				}
				storePatchStreamAppendOnlyParseResultSink = getParseResult
			}
		})
		parseB.Run(parseBenchName+"/current_overlay_validation", func(parseB *testing.B) {
			getFixture := buildPatchStreamParseAppendOnlyBenchmarkFixture(parseAppendCount)
			parseB.ReportAllocs()
			parseB.ResetTimer()
			for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
				getParseResult, hasPatchApply, parseParseErr := ParsePatchStreamTransaction(
					getFixture.getPatchStream,
					"bench-region",
					1,
					getFixture.getKnownNodeIDs,
					nil,
					nil,
				)
				if parseParseErr != nil {
					parseB.Fatalf("ParsePatchStreamTransaction returned error: %v", parseParseErr)
				}
				if !hasPatchApply {
					parseB.Fatal("ParsePatchStreamTransaction unexpectedly ignored patch")
				}
				storePatchStreamAppendOnlyParseResultSink = getParseResult
			}
		})
	}
}
