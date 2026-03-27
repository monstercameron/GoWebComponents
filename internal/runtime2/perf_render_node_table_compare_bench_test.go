package runtime2

import (
	"fmt"
	"testing"
)

// buildRuntime2LegacyRenderNodeTable preserves the previous render-node table parse flow for benchmark comparison.
func buildRuntime2LegacyRenderNodeTable(parseRawRecords []RenderNodeRecordRaw) (RenderNodeTable, error) {
	parseRecords := make([]RenderNodeRecord, len(parseRawRecords))
	for parseIndex, parseRawRecord := range parseRawRecords {
		parseRecord, parseErr := ParseRenderNodeRecord(parseRawRecord)
		if parseErr != nil {
			return RenderNodeTable{}, fmt.Errorf("runtime2: render node record %d is invalid: %w", parseIndex, parseErr)
		}
		if parseRecord.NodeID == 0 {
			return RenderNodeTable{}, fmt.Errorf("runtime2: render node record %d has invalid node id 0", parseIndex)
		}
		parseRecords[parseIndex] = parseRecord
	}
	if parseErr := buildRuntime2LegacyValidateRenderNodeUniqueIDs(parseRecords); parseErr != nil {
		return RenderNodeTable{}, parseErr
	}
	if parseErr := parseRenderNodeChildSpans(parseRecords); parseErr != nil {
		return RenderNodeTable{}, parseErr
	}
	if parseErr := parseRenderNodeSiblingKeys(parseRecords); parseErr != nil {
		return RenderNodeTable{}, parseErr
	}
	return RenderNodeTable{Records: parseRecords}, nil
}

// buildRuntime2LegacyValidateRenderNodeUniqueIDs preserves the previous unique-ID validation pass for benchmark comparison.
func buildRuntime2LegacyValidateRenderNodeUniqueIDs(parseRecords []RenderNodeRecord) error {
	if len(parseRecords) <= 1 {
		return nil
	}
	parseMinNodeID := parseRecords[0].NodeID
	parseMaxNodeID := parseRecords[0].NodeID
	parseExpectedNodeID := uint64(1)
	isNodeIDSequenceDense := true
	for _, parseRecord := range parseRecords {
		if parseRecord.NodeID != parseExpectedNodeID {
			isNodeIDSequenceDense = false
		}
		parseExpectedNodeID++
		if parseRecord.NodeID < parseMinNodeID {
			parseMinNodeID = parseRecord.NodeID
		}
		if parseRecord.NodeID > parseMaxNodeID {
			parseMaxNodeID = parseRecord.NodeID
		}
	}
	if isNodeIDSequenceDense {
		return nil
	}
	parseNodeIDRange := parseMaxNodeID - parseMinNodeID + 1
	if parseNodeIDRange <= uint64(len(parseRecords))*getRenderNodeDenseIDRangeFactor &&
		parseNodeIDRange <= uint64(getRenderNodeDenseIDRangeMax) {
		parseSeenNodeIDs := make([]uint8, int(parseNodeIDRange))
		for parseIndex, parseRecord := range parseRecords {
			parseNodeIDOffset := int(parseRecord.NodeID - parseMinNodeID)
			if parseSeenNodeIDs[parseNodeIDOffset] != 0 {
				return fmt.Errorf("runtime2: render node record %d duplicates node id %d", parseIndex, parseRecord.NodeID)
			}
			parseSeenNodeIDs[parseNodeIDOffset] = 1
		}
		return nil
	}
	parseSeenNodeIDs := make(map[uint64]struct{}, len(parseRecords))
	for parseIndex, parseRecord := range parseRecords {
		if _, hasNodeID := parseSeenNodeIDs[parseRecord.NodeID]; hasNodeID {
			return fmt.Errorf("runtime2: render node record %d duplicates node id %d", parseIndex, parseRecord.NodeID)
		}
		parseSeenNodeIDs[parseRecord.NodeID] = struct{}{}
	}
	return nil
}

// BenchmarkParseRenderNodeTableCurrentVsLegacy compares the optimized node-table decode path against the previous full-rescan flow.
func BenchmarkParseRenderNodeTableCurrentVsLegacy(parseB *testing.B) {
	parseRenderOutput := buildPerfHotspotKeyedListRenderOutput(64, 0, "active")
	parseCanonicalIR, parseCanonicalErr := BuildCanonicalRenderIR(parseRenderOutput)
	if parseCanonicalErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR returned error: %v", parseCanonicalErr)
	}
	parseRawRecords := append([]RenderNodeRecordRaw(nil), parseCanonicalIR.GetNodeRecords...)
	parseB.Run("current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseTableErr := ParseRenderNodeTable(parseRawRecords); parseTableErr != nil {
				parseB.Fatalf("ParseRenderNodeTable returned error: %v", parseTableErr)
			}
		}
	})
	parseB.Run("legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseTableErr := buildRuntime2LegacyRenderNodeTable(parseRawRecords); parseTableErr != nil {
				parseB.Fatalf("buildRuntime2LegacyRenderNodeTable returned error: %v", parseTableErr)
			}
		}
	})
}
