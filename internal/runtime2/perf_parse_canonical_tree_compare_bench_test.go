package runtime2

import (
	"fmt"
	"testing"
)

// buildRuntime2LegacyCanonicalRenderTree decodes canonical render IR using the pre-optimization tree-state assembly path.
func buildRuntime2LegacyCanonicalRenderTree(parseIR CanonicalRenderIR) (canonicalRenderTree, error) {
	if len(parseIR.GetNodeRecords) == 0 {
		return canonicalRenderTree{}, fmt.Errorf("runtime2: canonical render IR node records are required")
	}
	parseNodeTable, parseNodeTableErr := ParseRenderNodeTable(parseIR.GetNodeRecords)
	if parseNodeTableErr != nil {
		return canonicalRenderTree{}, parseNodeTableErr
	}
	parseNodeRecords := parseNodeTable.Records
	buildTotalChildren := 0
	for parseRecordIndex := range parseNodeRecords {
		buildTotalChildren += int(parseNodeRecords[parseRecordIndex].ChildCount)
	}
	buildChildIDPool := make([]uint64, buildTotalChildren)
	buildChildPoolOffset := 0
	parseParentNodeIDByRecordIndex := make([]uint64, len(parseNodeRecords))
	parseDepthByRecordIndex := make([]int, len(parseNodeRecords))
	parseChildOrderByRecordIndex := make([][]uint64, len(parseNodeRecords))
	parseHasKeyedChildByRecordIndex := make([]bool, len(parseNodeRecords))
	for parseRecordIndex := range parseNodeRecords {
		getNodeRecord := parseNodeRecords[parseRecordIndex]
		if getNodeRecord.ChildCount == 0 {
			continue
		}
		buildChildStart := int(getNodeRecord.ChildStart)
		buildChildEnd := buildChildStart + int(getNodeRecord.ChildCount)
		if buildChildStart < 0 || buildChildEnd > len(parseNodeRecords) {
			return canonicalRenderTree{}, fmt.Errorf("runtime2: child span for node id %d is out of range", getNodeRecord.NodeID)
		}
		buildSlotStart := buildChildPoolOffset
		parseHasKeyedChild := false
		for parseChildIndex := buildChildStart; parseChildIndex < buildChildEnd; parseChildIndex++ {
			buildChildIDPool[buildChildPoolOffset] = parseNodeRecords[parseChildIndex].NodeID
			parseParentNodeIDByRecordIndex[parseChildIndex] = getNodeRecord.NodeID
			parseDepthByRecordIndex[parseChildIndex] = parseDepthByRecordIndex[parseRecordIndex] + 1
			if !parseHasKeyedChild && hasCanonicalNodeKeyText(parseNodeRecords[parseChildIndex].KeyText) {
				parseHasKeyedChild = true
			}
			buildChildPoolOffset++
		}
		parseChildOrderByRecordIndex[parseRecordIndex] = buildChildIDPool[buildSlotStart:buildChildPoolOffset]
		parseHasKeyedChildByRecordIndex[parseRecordIndex] = parseHasKeyedChild
	}
	parseNodeByID := make(map[uint64]canonicalRenderNodeState, len(parseNodeRecords))
	for parseRecordIndex := range parseNodeRecords {
		getNodeRecord := parseNodeRecords[parseRecordIndex]
		buildNodeState := canonicalRenderNodeState{
			getNodeID:       getNodeRecord.NodeID,
			getKind:         getNodeRecord.Kind,
			getDepth:        parseDepthByRecordIndex[parseRecordIndex],
			getKey:          getNodeRecord.KeyText,
			hasKey:          hasCanonicalNodeKeyText(getNodeRecord.KeyText),
			hasKeyedChild:   parseHasKeyedChildByRecordIndex[parseRecordIndex],
			getParentNodeID: parseParentNodeIDByRecordIndex[parseRecordIndex],
			getChildNodeIDs: parseChildOrderByRecordIndex[parseRecordIndex],
		}
		switch getNodeRecord.Kind {
		case RenderNodeKindText:
			getTextValue, getTextErr := parseIR.GetStringTable.GetRenderStringByRef(getNodeRecord.TextRef)
			if getTextErr != nil {
				return canonicalRenderTree{}, fmt.Errorf("runtime2: text node id %d has invalid text reference: %w", getNodeRecord.NodeID, getTextErr)
			}
			buildNodeState.getText = getTextValue
		case RenderNodeKindHostElement:
			getTagValue, getTagErr := parseIR.GetStringTable.GetRenderStringByRef(getNodeRecord.TextRef)
			if getTagErr != nil {
				return canonicalRenderTree{}, fmt.Errorf("runtime2: host node id %d has invalid tag reference: %w", getNodeRecord.NodeID, getTagErr)
			}
			buildNodeState.getTag = getTagValue
		}
		if getNodeRecord.PropCount > 0 {
			buildPropStart := int(getNodeRecord.PropStart)
			buildPropEnd := buildPropStart + int(getNodeRecord.PropCount)
			if buildPropStart < 0 || buildPropEnd > len(parseIR.GetPropRecords) {
				return canonicalRenderTree{}, fmt.Errorf("runtime2: node id %d prop span [%d:%d) is out of range", getNodeRecord.NodeID, buildPropStart, buildPropEnd)
			}
			parsePropByKey, parsePropErr := parseBuildCanonicalPropByKeyFromRaw(parseIR.GetPropRecords[buildPropStart:buildPropEnd], parseIR.GetStringTable)
			if parsePropErr != nil {
				return canonicalRenderTree{}, fmt.Errorf("runtime2: node id %d props are invalid: %w", getNodeRecord.NodeID, parsePropErr)
			}
			buildNodeState.getPropByKey = parsePropByKey
		}
		parseNodeByID[getNodeRecord.NodeID] = buildNodeState
	}
	parseRootNodeID := parseIR.GetRootNodeID
	if parseRootNodeID == 0 {
		parseRootNodeID = parseIR.GetNodeRecords[0].NodeID
	}
	if _, hasRootNode := parseNodeByID[parseRootNodeID]; !hasRootNode {
		return canonicalRenderTree{}, fmt.Errorf("runtime2: canonical root node id %d is missing", parseRootNodeID)
	}
	return canonicalRenderTree{
		getRootNodeID: parseRootNodeID,
		getNodeByID:   parseNodeByID,
	}, nil
}

// BenchmarkParseCanonicalRenderTreeCurrentVsLegacy compares the optimized tree decode path against the previous assembly path.
func BenchmarkParseCanonicalRenderTreeCurrentVsLegacy(parseB *testing.B) {
	parseRenderOutput := buildPerfHotspotKeyedListRenderOutput(64, 0, "active")
	parseCanonicalIR, parseCanonicalErr := BuildCanonicalRenderIR(parseRenderOutput)
	if parseCanonicalErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR returned error: %v", parseCanonicalErr)
	}
	parseB.Run("current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseTreeErr := ParseCanonicalRenderTree(parseCanonicalIR); parseTreeErr != nil {
				parseB.Fatalf("ParseCanonicalRenderTree returned error: %v", parseTreeErr)
			}
		}
	})
	parseB.Run("legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseTreeErr := buildRuntime2LegacyCanonicalRenderTree(parseCanonicalIR); parseTreeErr != nil {
				parseB.Fatalf("buildRuntime2LegacyCanonicalRenderTree returned error: %v", parseTreeErr)
			}
		}
	})
}
