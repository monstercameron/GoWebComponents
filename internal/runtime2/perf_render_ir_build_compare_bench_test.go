package runtime2

import (
	"fmt"
	"strconv"
	"testing"
)

var storeBuildCanonicalRenderIRSink CanonicalRenderIR

// buildRenderIRLargeHostTreeBenchOutput builds one large host-element-heavy render payload used by canonical IR compare benchmarks.
func buildRenderIRLargeHostTreeBenchOutput() map[string]any {
	buildSectionChildren := make([]any, 0, 96)
	for parseSectionIndex := 0; parseSectionIndex < 96; parseSectionIndex++ {
		buildListChildren := make([]any, 0, 6)
		for parseItemIndex := 0; parseItemIndex < 6; parseItemIndex++ {
			buildListChildren = append(buildListChildren, map[string]any{
				"kind": "host-element",
				"tag":  "li",
				"key":  "item-" + strconv.Itoa(parseSectionIndex) + "-" + strconv.Itoa(parseItemIndex),
				"children": []any{
					map[string]any{
						"kind": "text",
						"text": "label-" + strconv.Itoa(parseSectionIndex) + "-" + strconv.Itoa(parseItemIndex),
					},
				},
			})
		}
		buildSectionChildren = append(buildSectionChildren, map[string]any{
			"kind": "host-element",
			"tag":  "section",
			"key":  "section-" + strconv.Itoa(parseSectionIndex),
			"props": map[string]any{
				"class": "pane",
				"data-id": strconv.Itoa(parseSectionIndex),
			},
			"children": []any{
				map[string]any{
					"kind": "host-element",
					"tag":  "h3",
					"children": []any{
						map[string]any{
							"kind": "text",
							"text": "header-" + strconv.Itoa(parseSectionIndex),
						},
					},
				},
				map[string]any{
					"kind":     "host-element",
					"tag":      "ul",
					"children": buildListChildren,
				},
			},
		})
	}
	return map[string]any{
		"kind": "host-element",
		"tag":  "main",
		"props": map[string]any{
			"class": "dashboard",
		},
		"children": buildSectionChildren,
	}
}

// buildLegacyCanonicalRenderIRChildIndexMap preserves the previous BuildCanonicalRenderIR child-start implementation that relied on node-id index map lookups.
func buildLegacyCanonicalRenderIRChildIndexMap(parseRenderOutput any) (CanonicalRenderIR, error) {
	buildRootNode, parseRootErr := parseBuildCanonicalRootNode(parseRenderOutput)
	if parseRootErr != nil {
		return CanonicalRenderIR{}, parseRootErr
	}
	parseAssignCanonicalNodeIDs(buildRootNode)
	buildStringValues := []string{}
	parseAppendCanonicalStringValues(buildRootNode, &buildStringValues)
	buildStringTable := BuildRenderStringTable(buildStringValues)
	buildNodeOrder := parseBuildCanonicalNodeOrder(buildRootNode)
	buildNodeIndexByID := make(map[uint64]int, len(buildNodeOrder))
	for parseIndex, getNode := range buildNodeOrder {
		buildNodeIndexByID[getNode.getNodeID] = parseIndex
	}
	buildPropRecords := make([]RenderPropRecordRaw, 0)
	buildNodeRecords := make([]RenderNodeRecordRaw, 0, len(buildNodeOrder))
	for _, getNode := range buildNodeOrder {
		buildChildStart := 0
		if len(getNode.getChildren) > 0 {
			buildChildStart = buildNodeIndexByID[getNode.getChildren[0].getNodeID]
		}
		buildPropStart := len(buildPropRecords)
		for _, getPropRecord := range getNode.getProps {
			getKeyRef, hasKeyRef := buildStringTable.GetRenderStringRef(getPropRecord.Key)
			if !hasKeyRef {
				return CanonicalRenderIR{}, fmt.Errorf("runtime2: canonical prop key %q missing from string table", getPropRecord.Key)
			}
			getValueRef, hasValueRef := buildStringTable.GetRenderStringRef(getPropRecord.Value)
			if !hasValueRef {
				return CanonicalRenderIR{}, fmt.Errorf("runtime2: canonical prop value %q missing from string table", getPropRecord.Value)
			}
			buildPropRecords = append(buildPropRecords, RenderPropRecordRaw{
				Kind:     uint8(getPropRecord.Kind),
				KeyRef:   getKeyRef,
				ValueRef: getValueRef,
			})
		}
		buildTextRef := uint32(0)
		switch getNode.getKind {
		case RenderNodeKindText:
			getTextRef, hasTextRef := buildStringTable.GetRenderStringRef(getNode.getText)
			if !hasTextRef {
				return CanonicalRenderIR{}, fmt.Errorf("runtime2: canonical text payload %q missing from string table", getNode.getText)
			}
			buildTextRef = getTextRef
		case RenderNodeKindHostElement:
			getTagRef, hasTagRef := buildStringTable.GetRenderStringRef(getNode.getTag)
			if !hasTagRef {
				return CanonicalRenderIR{}, fmt.Errorf("runtime2: canonical host tag %q missing from string table", getNode.getTag)
			}
			buildTextRef = getTagRef
		}
		buildNodeRecord := RenderNodeRecordRaw{
			NodeID:     getNode.getNodeID,
			Kind:       uint8(getNode.getKind),
			ChildStart: uint32(buildChildStart),
			ChildCount: uint32(len(getNode.getChildren)),
			PropStart:  uint32(buildPropStart),
			PropCount:  uint32(len(getNode.getProps)),
			TextRef:    buildTextRef,
		}
		if getNode.hasKey {
			buildNodeRecord.KeyText = getNode.getKey
			buildNodeRecord.KeyHash = parseHashCanonicalString("key:"+getNode.getKey, nil)
		}
		buildNodeRecords = append(buildNodeRecords, buildNodeRecord)
	}
	return CanonicalRenderIR{
		GetRootNodeID:  buildRootNode.getNodeID,
		GetStringTable: buildStringTable,
		GetNodeRecords: buildNodeRecords,
		GetPropRecords: buildPropRecords,
	}, nil
}

// BenchmarkBuildCanonicalRenderIRCurrentVsLegacy compares the current child-start index-slice path against the previous node-id map lookup path.
func BenchmarkBuildCanonicalRenderIRCurrentVsLegacy(parseB *testing.B) {
	parseRenderOutput := buildRenderIRLargeHostTreeBenchOutput()
	parseB.Run("legacy_node_id_index_map", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			buildCanonicalIR, parseErr := buildLegacyCanonicalRenderIRChildIndexMap(parseRenderOutput)
			if parseErr != nil {
				parseB.Fatalf("buildLegacyCanonicalRenderIRChildIndexMap returned error: %v", parseErr)
			}
			storeBuildCanonicalRenderIRSink = buildCanonicalIR
		}
	})
	parseB.Run("current_child_start_slice", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			buildCanonicalIR, parseErr := BuildCanonicalRenderIR(parseRenderOutput)
			if parseErr != nil {
				parseB.Fatalf("BuildCanonicalRenderIR returned error: %v", parseErr)
			}
			storeBuildCanonicalRenderIRSink = buildCanonicalIR
		}
	})
}
