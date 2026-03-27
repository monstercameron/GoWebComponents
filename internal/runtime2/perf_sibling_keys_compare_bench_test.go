package runtime2

import (
	"fmt"
	"testing"
)

// buildPerfSiblingKeyCompareRenderNodeRecords builds one parent-plus-children table for sibling key-validation benchmarks.
func buildPerfSiblingKeyCompareRenderNodeRecords(parseChildCount int) []RenderNodeRecord {
	if parseChildCount <= 0 {
		parseChildCount = 1
	}
	parseRecords := make([]RenderNodeRecord, 0, parseChildCount+1)
	parseRecords = append(parseRecords, RenderNodeRecord{
		NodeID:     1,
		Kind:       RenderNodeKindFragment,
		ChildStart: 1,
		ChildCount: uint32(parseChildCount),
	})
	for parseIndex := 0; parseIndex < parseChildCount; parseIndex++ {
		parseRecords = append(parseRecords, RenderNodeRecord{
			NodeID:  uint64(parseIndex + 2),
			Kind:    RenderNodeKindHostElement,
			KeyHash: uint64(parseIndex + 1000),
			KeyText: fmt.Sprintf("item-%d", parseIndex),
		})
	}
	return parseRecords
}

// parseRenderNodeSiblingKeysLegacyBenchmark preserves the legacy keyed-sibling branch-selection behavior for benchmark comparison.
func parseRenderNodeSiblingKeysLegacyBenchmark(parseRecords []RenderNodeRecord) error {
	for _, parseRecord := range parseRecords {
		if parseRecord.ChildCount == 0 {
			continue
		}
		parseChildStart := int(parseRecord.ChildStart)
		parseChildEnd := parseChildStart + int(parseRecord.ChildCount)
		parseKeyedSiblingCount := 0
		for parseChildIndex := parseChildStart; parseChildIndex < parseChildEnd; parseChildIndex++ {
			if parseRecords[parseChildIndex].KeyHash != 0 {
				parseKeyedSiblingCount++
				if parseKeyedSiblingCount > 1 {
					break
				}
			}
		}
		if parseKeyedSiblingCount <= 1 {
			continue
		}
		if parseKeyedSiblingCount <= getRenderNodeSiblingKeyPairwiseLimit {
			for parseLeftIndex := parseChildStart; parseLeftIndex < parseChildEnd; parseLeftIndex++ {
				parseLeftRecord := parseRecords[parseLeftIndex]
				if parseLeftRecord.KeyHash == 0 {
					continue
				}
				for parseRightIndex := parseLeftIndex + 1; parseRightIndex < parseChildEnd; parseRightIndex++ {
					parseRightRecord := parseRecords[parseRightIndex]
					if parseRightRecord.KeyHash == 0 {
						continue
					}
					if parseLeftRecord.KeyHash == parseRightRecord.KeyHash && parseLeftRecord.KeyText == parseRightRecord.KeyText {
						return fmt.Errorf(
							"runtime2: duplicate keyed child %d:%q under parent node id %d for node ids %d and %d",
							parseRightRecord.KeyHash,
							parseRightRecord.KeyText,
							parseRecord.NodeID,
							parseLeftRecord.NodeID,
							parseRightRecord.NodeID,
						)
					}
				}
			}
			continue
		}
		parseSiblingKeys := make(map[parseRenderNodeSiblingKey]uint64, int(parseRecord.ChildCount))
		for parseChildIndex := parseChildStart; parseChildIndex < parseChildEnd; parseChildIndex++ {
			parseChildRecord := parseRecords[parseChildIndex]
			if parseChildRecord.KeyHash == 0 {
				continue
			}
			parseSiblingKey := parseRenderNodeSiblingKey{
				getKeyHash: parseChildRecord.KeyHash,
				getKeyText: parseChildRecord.KeyText,
			}
			if getNodeID, hasNodeID := parseSiblingKeys[parseSiblingKey]; hasNodeID {
				return fmt.Errorf(
					"runtime2: duplicate keyed child %d:%q under parent node id %d for node ids %d and %d",
					parseChildRecord.KeyHash,
					parseChildRecord.KeyText,
					parseRecord.NodeID,
					getNodeID,
					parseChildRecord.NodeID,
				)
			}
			parseSiblingKeys[parseSiblingKey] = parseChildRecord.NodeID
		}
	}
	return nil
}

// BenchmarkParseRenderNodeSiblingKeysCurrentVsLegacy compares the current keyed-sibling validation against legacy branch-selection behavior.
func BenchmarkParseRenderNodeSiblingKeysCurrentVsLegacy(parseB *testing.B) {
	parseB.Run("medium-64/current", func(parseB *testing.B) {
		parseRecords := buildPerfSiblingKeyCompareRenderNodeRecords(64)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if parseErr := parseRenderNodeSiblingKeys(parseRecords); parseErr != nil {
				parseB.Fatalf("parseRenderNodeSiblingKeys returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("medium-64/legacy", func(parseB *testing.B) {
		parseRecords := buildPerfSiblingKeyCompareRenderNodeRecords(64)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if parseErr := parseRenderNodeSiblingKeysLegacyBenchmark(parseRecords); parseErr != nil {
				parseB.Fatalf("parseRenderNodeSiblingKeysLegacyBenchmark returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("large-512/current", func(parseB *testing.B) {
		parseRecords := buildPerfSiblingKeyCompareRenderNodeRecords(512)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if parseErr := parseRenderNodeSiblingKeys(parseRecords); parseErr != nil {
				parseB.Fatalf("parseRenderNodeSiblingKeys returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("large-512/legacy", func(parseB *testing.B) {
		parseRecords := buildPerfSiblingKeyCompareRenderNodeRecords(512)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if parseErr := parseRenderNodeSiblingKeysLegacyBenchmark(parseRecords); parseErr != nil {
				parseB.Fatalf("parseRenderNodeSiblingKeysLegacyBenchmark returned error: %v", parseErr)
			}
		}
	})
}
