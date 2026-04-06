package runtime2

import (
	"reflect"
	"testing"
)

// TestRenderIRCoverageBranches covers the low-level canonical render IR parsing branches that the behavior tests do not reach.
func TestRenderIRCoverageBranches(parseT *testing.T) {
	parseT.Run("root-and-node-parsing", func(parseT *testing.T) {
		var parseNilSlice *[]any
		if getNode, parseErr := parseBuildCanonicalRootNode(nil); parseErr != nil || getNode.getKind != RenderNodeKindFragment {
			parseT.Fatalf("parseBuildCanonicalRootNode(nil) = %+v, %v", getNode, parseErr)
		}
		if getNode, parseErr := parseBuildCanonicalRootNode(any(parseNilSlice)); parseErr != nil || getNode.getKind != RenderNodeKindFragment {
			parseT.Fatalf("parseBuildCanonicalRootNode(nil slice pointer) = %+v, %v", getNode, parseErr)
		}
		if getNode, parseErr := parseBuildCanonicalRootNode([]any{}); parseErr != nil || getNode.getKind != RenderNodeKindFragment {
			parseT.Fatalf("parseBuildCanonicalRootNode(empty slice) = %+v, %v", getNode, parseErr)
		}
		if getNode, parseErr := parseBuildCanonicalRootNode([]any{"hello"}); parseErr != nil || getNode.getKind != RenderNodeKindText || getNode.getText != "hello" {
			parseT.Fatalf("parseBuildCanonicalRootNode(single text) = %+v, %v", getNode, parseErr)
		}
		if getNode, parseErr := parseBuildCanonicalRootNode([]any{map[string]any{"kind": "host-element", "tag": "div"}}); parseErr != nil || getNode.getKind != RenderNodeKindHostElement || getNode.getTag != "div" {
			parseT.Fatalf("parseBuildCanonicalRootNode(single host) = %+v, %v", getNode, parseErr)
		}
		if getNode, parseErr := parseBuildCanonicalRootNode([]any{map[string]any{"kind": "fragment", "children": []any{"child"}}}); parseErr != nil || getNode.getKind != RenderNodeKindFragment || len(getNode.getChildren) != 1 {
			parseT.Fatalf("parseBuildCanonicalRootNode(fragment wrapper) = %+v, %v", getNode, parseErr)
		}
		if getNode, parseErr := parseBuildCanonicalRootNode([2]any{"a", "b"}); parseErr != nil || getNode.getKind != RenderNodeKindFragment || len(getNode.getChildren) != 2 {
			parseT.Fatalf("parseBuildCanonicalRootNode(array) = %+v, %v", getNode, parseErr)
		}

		if getNode, parseErr := parseBuildCanonicalRenderNode(nil); parseErr != nil || getNode.getKind != RenderNodeKindFragment {
			parseT.Fatalf("parseBuildCanonicalRenderNode(nil) = %+v, %v", getNode, parseErr)
		}
		if getNode, parseErr := parseBuildCanonicalRenderNode("hello"); parseErr != nil || getNode.getKind != RenderNodeKindText || getNode.getText != "hello" {
			parseT.Fatalf("parseBuildCanonicalRenderNode(string) = %+v, %v", getNode, parseErr)
		}
		if getNode, parseErr := parseBuildCanonicalRenderNode(true); parseErr != nil || getNode.getText != "true" {
			parseT.Fatalf("parseBuildCanonicalRenderNode(bool) = %+v, %v", getNode, parseErr)
		}
		if getNode, parseErr := parseBuildCanonicalRenderNode(float32(1.5)); parseErr != nil || getNode.getText != "1.5" {
			parseT.Fatalf("parseBuildCanonicalRenderNode(float32) = %+v, %v", getNode, parseErr)
		}
		if getNode, parseErr := parseBuildCanonicalRenderNode([]any{"a", map[string]any{"kind": "text", "text": "b"}}); parseErr != nil || getNode.getKind != RenderNodeKindFragment || len(getNode.getChildren) != 2 {
			parseT.Fatalf("parseBuildCanonicalRenderNode(list) = %+v, %v", getNode, parseErr)
		}
		if getNode, parseErr := parseBuildCanonicalRenderNode(map[string]any{"foo": "bar"}); parseErr != nil || getNode.getKind != RenderNodeKindText {
			parseT.Fatalf("parseBuildCanonicalRenderNode(json fallback) = %+v, %v", getNode, parseErr)
		}
		if _, parseErr := parseBuildCanonicalRenderNode(make(chan int)); parseErr == nil {
			parseT.Fatal("expected parseBuildCanonicalRenderNode(channel) to fail")
		}
		if _, parseErr := parseBuildCanonicalRenderNode(map[string]any{"kind": "unsupported"}); parseErr == nil {
			parseT.Fatal("expected parseBuildCanonicalRenderNode(unsupported kind) to fail")
		}
		if getNode, parseErr := parseBuildCanonicalRenderNode(any(map[int]any{1: "one"})); parseErr != nil || getNode.getKind != RenderNodeKindText {
			parseT.Fatalf("parseBuildCanonicalRenderNode(reflect map) = %+v, %v", getNode, parseErr)
		}

		if getNode, parseErr := parseBuildCanonicalRenderNodeFromMap(map[string]any{"kind": "text", "text": "trim"}); parseErr != nil || getNode.getKind != RenderNodeKindText {
			parseT.Fatalf("parseBuildCanonicalRenderNodeFromMap(text) = %+v, %v", getNode, parseErr)
		}
		if getNode, parseErr := parseBuildCanonicalRenderNodeFromMap(map[string]any{"kind": "fragment", "key": " item ", "children": []any{"child"}}); parseErr != nil || getNode.getKind != RenderNodeKindFragment || !getNode.hasKey || getNode.getKey != " item " {
			parseT.Fatalf("parseBuildCanonicalRenderNodeFromMap(fragment) = %+v, %v", getNode, parseErr)
		}
		if getNode, parseErr := parseBuildCanonicalRenderNodeFromMap(map[string]any{"kind": "element", "tag": "div", "props": map[string]any{"class": "cta"}, "children": []any{"ok"}, "key": " btn "}); parseErr != nil || getNode.getKind != RenderNodeKindHostElement || getNode.getTag != "div" || !getNode.hasKey || getNode.getKey != " btn " {
			parseT.Fatalf("parseBuildCanonicalRenderNodeFromMap(element) = %+v, %v", getNode, parseErr)
		}
		if _, parseErr := parseBuildCanonicalRenderNodeFromMap(map[string]any{"kind": "bogus"}); parseErr == nil {
			parseT.Fatal("expected parseBuildCanonicalRenderNodeFromMap(unsupported) to fail")
		}

		if getKind := parseGetCanonicalRenderNodeKind("text"); getKind != "text" {
			parseT.Fatalf("parseGetCanonicalRenderNodeKind(text) = %q", getKind)
		}
		if getKind := parseGetCanonicalRenderNodeKind("  HOST-ELEMENT  "); getKind != "host-element" {
			parseT.Fatalf("parseGetCanonicalRenderNodeKind(trimmed) = %q", getKind)
		}
		if getKind := parseGetCanonicalRenderNodeKind(3); getKind != "3" {
			parseT.Fatalf("parseGetCanonicalRenderNodeKind(non-string) = %q", getKind)
		}
		if getText := parseGetCanonicalTrimmedStringValue("  hi  "); getText != "hi" {
			parseT.Fatalf("parseGetCanonicalTrimmedStringValue = %q", getText)
		}
		if !hasCanonicalPropKeyText("class") || hasCanonicalPropKeyText("   ") {
			parseT.Fatal("unexpected canonical prop key text classification")
		}
		if !hasCanonicalNodeKeyText("item") || hasCanonicalNodeKeyText("   ") {
			parseT.Fatal("unexpected canonical node key text classification")
		}
		if !parseHasCanonicalAriaPropKey("aria-label") || parseHasCanonicalAriaPropKey("data-id") {
			parseT.Fatal("unexpected aria prop key classification")
		}
		if !parseHasCanonicalDataPropKey("data-id") || parseHasCanonicalDataPropKey("aria-label") {
			parseT.Fatal("unexpected data prop key classification")
		}
		if parseErr := parseValidateCanonicalPropFamilyConstant("bogus"); parseErr == nil {
			parseT.Fatal("expected unknown prop family to fail")
		}
	})

	parseT.Run("children-and-props", func(parseT *testing.T) {
		if getChildren, parseErr := parseBuildCanonicalChildren(nil); parseErr != nil || getChildren != nil {
			parseT.Fatalf("parseBuildCanonicalChildren(nil) = %+v, %v", getChildren, parseErr)
		}
		if getChildren, parseErr := parseBuildCanonicalChildren([]any{"a"}); parseErr != nil || len(getChildren) != 1 {
			parseT.Fatalf("parseBuildCanonicalChildren(list) = %+v, %v", getChildren, parseErr)
		}
		if getChildren, parseErr := parseBuildCanonicalChildren([]int{1, 2}); parseErr != nil || len(getChildren) != 2 {
			parseT.Fatalf("parseBuildCanonicalChildren(reflect slice) = %+v, %v", getChildren, parseErr)
		}
		if getChildren, parseErr := parseBuildCanonicalChildren(3); parseErr != nil || len(getChildren) != 1 || getChildren[0].getText != "3" {
			parseT.Fatalf("parseBuildCanonicalChildren(scalar) = %+v, %v", getChildren, parseErr)
		}
		if getChildren, parseErr := parseBuildCanonicalChildren(reflect.ValueOf([]any{"x"}).Interface()); parseErr != nil || len(getChildren) != 1 {
			parseT.Fatalf("parseBuildCanonicalChildren(interface slice) = %+v, %v", getChildren, parseErr)
		}
		var parseNilMap map[string]any
		if getProps, parseErr := parseBuildCanonicalPropsFromPropsValue(parseNilMap, false); parseErr != nil || getProps != nil {
			parseT.Fatalf("parseBuildCanonicalPropsFromPropsValue(missing) = %+v, %v", getProps, parseErr)
		}
		if getProps, parseErr := parseBuildCanonicalPropsFromPropsValue(map[string]any{"class": "card"}, true); parseErr != nil || len(getProps) != 1 || getProps[0].Key != "class" {
			parseT.Fatalf("parseBuildCanonicalPropsFromPropsValue(map[string]any) = %+v, %v", getProps, parseErr)
		}
		if getProps, parseErr := parseBuildCanonicalPropsFromPropsValue(map[string]string{"data-id": "7"}, true); parseErr != nil || len(getProps) != 1 || getProps[0].Key != "data-id" {
			parseT.Fatalf("parseBuildCanonicalPropsFromPropsValue(map[string]string) = %+v, %v", getProps, parseErr)
		}
		if getProps, parseErr := parseBuildCanonicalPropsFromPropsValue(3, true); parseErr != nil || getProps != nil {
			parseT.Fatalf("parseBuildCanonicalPropsFromPropsValue(unsupported) = %+v, %v", getProps, parseErr)
		}

		if getProps, parseErr := parseBuildCanonicalProps(map[string]any{
			"kind": "host-element",
			"tag":  "div",
			"props": map[string]any{
				"class": "card",
			},
			"data-id": "7",
		}); parseErr != nil || len(getProps) != 2 {
			parseT.Fatalf("parseBuildCanonicalProps(inline merge) = %+v, %v", getProps, parseErr)
		}
		if getProps, parseErr := parseBuildCanonicalProps(map[string]any{
			"kind": "host-element",
			"tag":  "div",
			"props": map[string]string{
				"class": "card",
			},
		}); parseErr != nil || len(getProps) != 1 || getProps[0].Key != "class" {
			parseT.Fatalf("parseBuildCanonicalProps(string props) = %+v, %v", getProps, parseErr)
		}
		if getProps, parseErr := parseBuildCanonicalProps(map[string]any{
			"kind":  "host-element",
			"tag":   "div",
			"props": struct{}{},
		}); parseErr != nil || getProps != nil {
			parseT.Fatalf("parseBuildCanonicalProps(unsupported props) = %+v, %v", getProps, parseErr)
		}

		if getRecord, hasRecord, parseErr := parseBuildCanonicalPropRecord("class", "card"); parseErr != nil || !hasRecord || getRecord.Kind != RenderPropKindClass {
			parseT.Fatalf("parseBuildCanonicalPropRecord(class) = %+v, %v", getRecord, parseErr)
		}
		if getRecord, hasRecord, parseErr := parseBuildCanonicalPropRecord("style", map[string]any{"color": "red"}); parseErr != nil || !hasRecord || getRecord.Kind != RenderPropKindStyle {
			parseT.Fatalf("parseBuildCanonicalPropRecord(style) = %+v, %v", getRecord, parseErr)
		}
		if getRecord, hasRecord, parseErr := parseBuildCanonicalPropRecord("text", 42); parseErr != nil || !hasRecord || getRecord.Kind != RenderPropKindTextAdjacent {
			parseT.Fatalf("parseBuildCanonicalPropRecord(text) = %+v, %v", getRecord, parseErr)
		}
		if getRecord, hasRecord, parseErr := parseBuildCanonicalPropRecord("aria-label", "name"); parseErr != nil || !hasRecord || getRecord.Kind != RenderPropKindAria {
			parseT.Fatalf("parseBuildCanonicalPropRecord(aria) = %+v, %v", getRecord, parseErr)
		}
		if getRecord, hasRecord, parseErr := parseBuildCanonicalPropRecord("data-id", "7"); parseErr != nil || !hasRecord || getRecord.Kind != RenderPropKindData {
			parseT.Fatalf("parseBuildCanonicalPropRecord(data) = %+v, %v", getRecord, parseErr)
		}
		if getRecord, hasRecord, parseErr := parseBuildCanonicalPropRecord("", "ignored"); parseErr != nil || hasRecord {
			parseT.Fatalf("parseBuildCanonicalPropRecord(empty) = %+v, %v", getRecord, parseErr)
		}
		if getRecord, hasRecord, parseErr := parseBuildCanonicalPropRecord("unknown", "ignored"); parseErr != nil || hasRecord {
			parseT.Fatalf("parseBuildCanonicalPropRecord(unknown) = %+v, %v", getRecord, parseErr)
		}

		parseProps := []RenderPropRecord{
			{Kind: RenderPropKindData, Key: "data-z", Value: "z"},
			{Kind: RenderPropKindClass, Key: "class", Value: "a"},
		}
		if getProps, parseErr := parseFinalizeCanonicalPropRecords(append([]RenderPropRecord(nil), parseProps...)); parseErr != nil || len(getProps) != 2 || getProps[0].Key != "class" {
			parseT.Fatalf("parseFinalizeCanonicalPropRecords(sorted) = %+v, %v", getProps, parseErr)
		}
		if getProps, parseErr := parseFinalizeCanonicalPropRecords([]RenderPropRecord{{Kind: RenderPropKindClass, Key: "class", Value: "a"}}); parseErr != nil || len(getProps) != 1 {
			parseT.Fatalf("parseFinalizeCanonicalPropRecords(single) = %+v, %v", getProps, parseErr)
		}
		if _, parseErr := parseFinalizeCanonicalPropRecords([]RenderPropRecord{
			{Kind: RenderPropKindClass, Key: "class", Value: "a"},
			{Kind: RenderPropKindData, Key: "class", Value: "b"},
		}); parseErr == nil {
			parseT.Fatal("expected duplicate canonical prop records to fail")
		}
	})

	parseT.Run("raw-record-and-tree", func(parseT *testing.T) {
		parseStringTable := RenderStringTable{Entries: []string{" ", "class", "div", "hello", "root", "data-id", "7"}}
		if getPropByKey, parseErr := parseBuildCanonicalPropByKeyFromRaw([]RenderPropRecordRaw{
			{Kind: uint8(RenderPropKindClass), KeyRef: 1, ValueRef: 3},
			{Kind: uint8(RenderPropKindData), KeyRef: 5, ValueRef: 6},
		}, parseStringTable); parseErr != nil || len(getPropByKey) != 2 {
			parseT.Fatalf("parseBuildCanonicalPropByKeyFromRaw(valid) = %+v, %v", getPropByKey, parseErr)
		}
		if _, parseErr := parseBuildCanonicalPropByKeyFromRaw([]RenderPropRecordRaw{{Kind: 0, KeyRef: 1, ValueRef: 3}}, parseStringTable); parseErr == nil {
			parseT.Fatal("expected invalid prop kind to fail")
		}
		if _, parseErr := parseBuildCanonicalPropByKeyFromRaw([]RenderPropRecordRaw{{Kind: uint8(RenderPropKindClass), KeyRef: 99, ValueRef: 3}}, parseStringTable); parseErr == nil {
			parseT.Fatal("expected out-of-range key reference to fail")
		}
		if _, parseErr := parseBuildCanonicalPropByKeyFromRaw([]RenderPropRecordRaw{{Kind: uint8(RenderPropKindClass), KeyRef: 0, ValueRef: 3}}, parseStringTable); parseErr == nil {
			parseT.Fatal("expected empty canonical prop key to fail")
		}
		if _, parseErr := parseBuildCanonicalPropByKeyFromRaw([]RenderPropRecordRaw{{Kind: uint8(RenderPropKindClass), KeyRef: 1, ValueRef: 99}}, parseStringTable); parseErr == nil {
			parseT.Fatal("expected out-of-range value reference to fail")
		}
		if _, parseErr := parseBuildCanonicalPropByKeyFromRaw([]RenderPropRecordRaw{
			{Kind: uint8(RenderPropKindClass), KeyRef: 1, ValueRef: 3},
			{Kind: uint8(RenderPropKindData), KeyRef: 1, ValueRef: 6},
		}, parseStringTable); parseErr == nil {
			parseT.Fatal("expected duplicate canonical prop key to fail")
		}
		if getDuplicate := getDuplicateCanonicalPropKey([]RenderPropRecordRaw{
			{Kind: uint8(RenderPropKindClass), KeyRef: 1, ValueRef: 3},
			{Kind: uint8(RenderPropKindData), KeyRef: 1, ValueRef: 6},
		}, parseStringTable.Entries); getDuplicate != "class" {
			parseT.Fatalf("getDuplicateCanonicalPropKey = %q, want %q", getDuplicate, "class")
		}
		if getDuplicate := getDuplicateCanonicalPropKey([]RenderPropRecordRaw{{Kind: uint8(RenderPropKindClass), KeyRef: 1, ValueRef: 3}}, parseStringTable.Entries); getDuplicate != "" {
			parseT.Fatalf("getDuplicateCanonicalPropKey(single) = %q, want empty", getDuplicate)
		}

		parseIR := CanonicalRenderIR{
			GetRootNodeID:  1,
			GetStringTable: BuildRenderStringTable([]string{"class", "div", "hello", "root"}),
			GetNodeRecords: []RenderNodeRecordRaw{
				{NodeID: 1, Kind: uint8(RenderNodeKindHostElement), ChildStart: 1, ChildCount: 1, PropStart: 0, PropCount: 1, TextRef: 1},
				{NodeID: 2, Kind: uint8(RenderNodeKindText), TextRef: 2, KeyHash: 1, KeyText: "kid"},
			},
			GetPropRecords: []RenderPropRecordRaw{
				{Kind: uint8(RenderPropKindClass), KeyRef: 0, ValueRef: 3},
			},
		}
		parseTree, parseErr := ParseCanonicalRenderTree(parseIR)
		if parseErr != nil {
			parseT.Fatalf("ParseCanonicalRenderTree(valid) error = %v", parseErr)
		}
		if getState, hasState := parseTree.GetCanonicalNodeState(1); !hasState || getState.getTag != "div" || !getState.hasKeyedChild {
			parseT.Fatalf("GetCanonicalNodeState(root) = %+v, %v", getState, hasState)
		}
		if getState, hasState := parseTree.GetCanonicalNodeState(2); !hasState || getState.getText != "hello" || parseTree.GetCanonicalNodeDepth(2) != 1 {
			parseT.Fatalf("GetCanonicalNodeState(child) = %+v, %v depth=%d", getState, hasState, parseTree.GetCanonicalNodeDepth(2))
		}
		if _, hasState := parseTree.GetCanonicalNodeState(99); hasState {
			parseT.Fatal("expected unknown node ID lookup to fail")
		}

		if _, parseErr := ParseCanonicalRenderTree(CanonicalRenderIR{}); parseErr == nil {
			parseT.Fatal("expected empty canonical IR to fail")
		}
		if _, parseErr := ParseCanonicalRenderTree(CanonicalRenderIR{
			GetRootNodeID:  1,
			GetStringTable: BuildRenderStringTable([]string{"div"}),
			GetNodeRecords: []RenderNodeRecordRaw{{NodeID: 1, Kind: uint8(RenderNodeKindText), TextRef: 99}},
		}); parseErr == nil {
			parseT.Fatal("expected invalid text reference to fail")
		}
		if _, parseErr := ParseCanonicalRenderTree(CanonicalRenderIR{
			GetRootNodeID:  1,
			GetStringTable: BuildRenderStringTable([]string{"div"}),
			GetNodeRecords: []RenderNodeRecordRaw{{NodeID: 1, Kind: uint8(RenderNodeKindHostElement), TextRef: 99}},
		}); parseErr == nil {
			parseT.Fatal("expected invalid tag reference to fail")
		}
		if _, parseErr := ParseCanonicalRenderTree(CanonicalRenderIR{
			GetRootNodeID:  1,
			GetStringTable: BuildRenderStringTable([]string{"class", "div", "hello"}),
			GetNodeRecords: []RenderNodeRecordRaw{{NodeID: 1, Kind: uint8(RenderNodeKindHostElement), ChildStart: 0, ChildCount: 0, PropStart: 99, PropCount: 1, TextRef: 1}},
			GetPropRecords: []RenderPropRecordRaw{{Kind: uint8(RenderPropKindClass), KeyRef: 0, ValueRef: 2}},
		}); parseErr == nil {
			parseT.Fatal("expected invalid prop span to fail")
		}
		if _, parseErr := ParseCanonicalRenderTree(CanonicalRenderIR{
			GetRootNodeID:  99,
			GetStringTable: BuildRenderStringTable([]string{"div"}),
			GetNodeRecords: []RenderNodeRecordRaw{{NodeID: 1, Kind: uint8(RenderNodeKindHostElement), TextRef: 0}},
		}); parseErr == nil {
			parseT.Fatal("expected missing canonical root node to fail")
		}

		parseSiblingRoot := &canonicalRenderNode{
			getKind: RenderNodeKindFragment,
			getChildren: []*canonicalRenderNode{
				{getKind: RenderNodeKindHostElement, hasKey: true, getKey: "dup"},
				{getKind: RenderNodeKindHostElement, hasKey: true, getKey: "dup"},
			},
		}
		if parseErr := parseValidateCanonicalSiblingKeys(parseSiblingRoot); parseErr == nil {
			parseT.Fatal("expected duplicate sibling keys to fail")
		}
		if parseErr := parseValidateCanonicalChildSiblingKeys(parseSiblingRoot); parseErr == nil {
			parseT.Fatal("expected duplicate child sibling keys to fail")
		}
		if parseErr := parseValidateCanonicalSiblingKeys(nil); parseErr != nil {
			parseT.Fatalf("parseValidateCanonicalSiblingKeys(nil) error = %v", parseErr)
		}
		if parseErr := parseValidateCanonicalChildSiblingKeys(nil); parseErr != nil {
			parseT.Fatalf("parseValidateCanonicalChildSiblingKeys(nil) error = %v", parseErr)
		}
	})
}

// TestRenderIRExportedRoundTripBranches covers exported canonical IR entry points and their equality checks.
func TestRenderIRExportedRoundTripBranches(parseT *testing.T) {
	parseOutput := map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"props": map[string]any{
			"class":   "card",
			"data-id": "7",
		},
		"children": []any{
			"hello",
		},
	}
	parseCanonicalIR, parseErr := BuildCanonicalRenderIR(parseOutput)
	if parseErr != nil {
		parseT.Fatalf("BuildCanonicalRenderIR(valid) error = %v", parseErr)
	}
	if parseCanonicalIR.GetRootNodeID == 0 || len(parseCanonicalIR.GetNodeRecords) != 2 || len(parseCanonicalIR.GetPropRecords) != 2 {
		parseT.Fatalf("unexpected canonical IR: %+v", parseCanonicalIR)
	}
	parseStringTable, parseStringErr := BuildRenderStringTableFromRenderOutput(parseOutput)
	if parseStringErr != nil {
		parseT.Fatalf("BuildRenderStringTableFromRenderOutput error = %v", parseStringErr)
	}
	if len(parseStringTable.Entries) == 0 {
		parseT.Fatal("expected string table entries")
	}
	parsePropRecords, parsePropTable, parsePropErr := BuildRenderPropRecordsFromRenderOutput(parseOutput)
	if parsePropErr != nil {
		parseT.Fatalf("BuildRenderPropRecordsFromRenderOutput error = %v", parsePropErr)
	}
	if len(parsePropRecords) != 2 || len(parsePropTable.Entries) == 0 {
		parseT.Fatalf("unexpected prop records/table: %+v %+v", parsePropRecords, parsePropTable)
	}
	if !IsCanonicalRenderIREqual(parseCanonicalIR, parseCanonicalIR) {
		parseT.Fatal("expected identical canonical IR values to compare equal")
	}
	parseDifferentIR := parseCanonicalIR
	parseDifferentIR.GetRootNodeID++
	if IsCanonicalRenderIREqual(parseCanonicalIR, parseDifferentIR) {
		parseT.Fatal("expected different canonical IR values to compare unequal")
	}
	if _, parseErr := BuildCanonicalRenderIR([]any{
		map[string]any{"kind": "host-element", "tag": "div", "key": "dup"},
		map[string]any{"kind": "host-element", "tag": "span", "key": "dup"},
	}); parseErr == nil {
		parseT.Fatal("expected duplicate keyed siblings to fail")
	}
}
