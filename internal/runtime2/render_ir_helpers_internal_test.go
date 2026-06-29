package runtime2

import (
	"reflect"
	"strings"
	"testing"
)

// TestRenderIRCanonicalNodeHelpersAssignIDsAndCollectStrings verifies canonical node helpers assign stable non-zero IDs and collect string payloads in traversal order.
func TestRenderIRCanonicalNodeHelpersAssignIDsAndCollectStrings(parseT *testing.T) {
	var parseNilNode *canonicalRenderNode
	parseAssignCanonicalNodeIDs(parseNilNode)
	var parseNilStrings []string
	parseAppendCanonicalStringValues(parseNilNode, &parseNilStrings)
	parseAppendCanonicalStringValues(&canonicalRenderNode{}, nil)

	parseRootNode := &canonicalRenderNode{
		getKind: RenderNodeKindHostElement,
		getTag:  "div",
		getProps: []RenderPropRecord{{
			Kind:  RenderPropKindClass,
			Key:   "class",
			Value: "root",
		}},
		getChildren: []*canonicalRenderNode{
			{
				getKind: RenderNodeKindText,
				getText: "hello",
			},
			{
				getKind: RenderNodeKindHostElement,
				getTag:  "span",
				getKey:  "item-1",
				hasKey:  true,
				getProps: []RenderPropRecord{{
					Kind:  RenderPropKindData,
					Key:   "data-id",
					Value: "7",
				}},
			},
		},
	}
	parseAssignCanonicalNodeIDs(parseRootNode)
	if parseRootNode.getNodeID == 0 || parseRootNode.getChildren[0].getNodeID == 0 || parseRootNode.getChildren[1].getNodeID == 0 {
		parseT.Fatalf("expected canonical node IDs to be assigned, got root=%d children=%d/%d", parseRootNode.getNodeID, parseRootNode.getChildren[0].getNodeID, parseRootNode.getChildren[1].getNodeID)
	}
	if parseRootNode.getNodeID == parseRootNode.getChildren[0].getNodeID || parseRootNode.getNodeID == parseRootNode.getChildren[1].getNodeID || parseRootNode.getChildren[0].getNodeID == parseRootNode.getChildren[1].getNodeID {
		parseT.Fatalf("expected canonical node IDs to stay unique, got root=%d children=%d/%d", parseRootNode.getNodeID, parseRootNode.getChildren[0].getNodeID, parseRootNode.getChildren[1].getNodeID)
	}

	parseUsedIDs := map[uint64]struct{}{}
	parseFirstHash := parseHashCanonicalString("root", parseUsedIDs)
	if parseFirstHash == 0 {
		parseT.Fatal("expected canonical string hash to be non-zero")
	}
	if _, hasFirstHash := parseUsedIDs[parseFirstHash]; !hasFirstHash {
		parseT.Fatalf("expected canonical string hash %d to be stored in used-ID set", parseFirstHash)
	}
	parseCollisionIDs := map[uint64]struct{}{parseFirstHash: {}}
	parseCollisionHash := parseHashCanonicalString("root", parseCollisionIDs)
	if parseCollisionHash == 0 || parseCollisionHash == parseFirstHash {
		parseT.Fatalf("expected canonical string hash collision to advance to a new ID, got first=%d collision=%d", parseFirstHash, parseCollisionHash)
	}
	if parseHashCanonicalString("root", nil) == 0 {
		parseT.Fatal("expected canonical string hash without collision set to be non-zero")
	}

	var parseStringValues []string
	parseAppendCanonicalStringValues(parseRootNode, &parseStringValues)
	if len(parseStringValues) != 8 {
		parseT.Fatalf("expected eight canonical string values, got %+v", parseStringValues)
	}
	parseWantStrings := []string{"div", "class", "root", "hello", "span", "item-1", "data-id", "7"}
	for parseIndex, parseWantString := range parseWantStrings {
		if parseStringValues[parseIndex] != parseWantString {
			parseT.Fatalf("canonical string values[%d] = %q, want %q", parseIndex, parseStringValues[parseIndex], parseWantString)
		}
	}
}

// TestRenderIRCanonicalMapHelpersNormalizeAndFormatKeyTypes verifies reflected map normalization and key formatting cover string, pointer, and numeric-key paths.
func TestRenderIRCanonicalMapHelpersNormalizeAndFormatKeyTypes(parseT *testing.T) {
	if _, parseErr := parseBuildCanonicalMapValue(reflect.Value{}); parseErr == nil {
		parseT.Fatal("expected invalid reflected map value to fail")
	}
	var parseNilMapPointer *map[string]int
	if _, parseErr := parseBuildCanonicalMapValue(reflect.ValueOf(parseNilMapPointer)); parseErr == nil {
		parseT.Fatal("expected nil reflected map pointer to fail")
	}
	if _, parseErr := parseBuildCanonicalMapValue(reflect.ValueOf(3)); parseErr == nil {
		parseT.Fatal("expected non-map reflected value to fail")
	}

	parseStringMapValue, parseStringMapErr := parseBuildCanonicalMapValue(reflect.ValueOf(map[string]any{"title": "Orders"}))
	if parseStringMapErr != nil {
		parseT.Fatalf("parseBuildCanonicalMapValue(string map) returned error: %v", parseStringMapErr)
	}
	if len(parseStringMapValue) != 1 || parseStringMapValue["title"] != "Orders" {
		parseT.Fatalf("unexpected normalized string-key map: %+v", parseStringMapValue)
	}

	parseWrappedMap := map[string]int{"count": 3}
	parseWrappedMapValue, parseWrappedMapErr := parseBuildCanonicalMapValue(reflect.ValueOf(any(&parseWrappedMap)))
	if parseWrappedMapErr != nil {
		parseT.Fatalf("parseBuildCanonicalMapValue(pointer-wrapped map) returned error: %v", parseWrappedMapErr)
	}
	if len(parseWrappedMapValue) != 1 || parseWrappedMapValue["count"] != 3 {
		parseT.Fatalf("unexpected normalized pointer-wrapped map: %+v", parseWrappedMapValue)
	}

	parseNumericMapValue, parseNumericMapErr := parseBuildCanonicalMapValue(reflect.ValueOf(map[int]any{1: "one", 2: "two"}))
	if parseNumericMapErr != nil {
		parseT.Fatalf("parseBuildCanonicalMapValue(numeric map) returned error: %v", parseNumericMapErr)
	}
	if len(parseNumericMapValue) != 2 || parseNumericMapValue["1"] != "one" || parseNumericMapValue["2"] != "two" {
		parseT.Fatalf("unexpected normalized numeric-key map: %+v", parseNumericMapValue)
	}

	if getKeyText := parseFormatCanonicalMapKeyValue(reflect.ValueOf(true), reflect.Bool); getKeyText != "true" {
		parseT.Fatalf("formatted bool key = %q, want %q", getKeyText, "true")
	}
	if getKeyText := parseFormatCanonicalMapKeyValue(reflect.ValueOf(int64(-7)), reflect.Int64); getKeyText != "-7" {
		parseT.Fatalf("formatted int key = %q, want %q", getKeyText, "-7")
	}
	if getKeyText := parseFormatCanonicalMapKeyValue(reflect.ValueOf(uint64(9)), reflect.Uint64); getKeyText != "9" {
		parseT.Fatalf("formatted uint key = %q, want %q", getKeyText, "9")
	}
	if getKeyText := parseFormatCanonicalMapKeyValue(reflect.ValueOf(float32(1.5)), reflect.Float32); getKeyText != "1.5" {
		parseT.Fatalf("formatted float32 key = %q, want %q", getKeyText, "1.5")
	}
	if getKeyText := parseFormatCanonicalMapKeyValue(reflect.ValueOf(float64(2.5)), reflect.Float64); getKeyText != "2.5" {
		parseT.Fatalf("formatted float64 key = %q, want %q", getKeyText, "2.5")
	}
	if getKeyText := parseFormatCanonicalMapKeyValue(reflect.ValueOf(struct{ Label string }{Label: "ok"}), reflect.Struct); getKeyText != "{ok}" {
		parseT.Fatalf("formatted fallback key = %q, want %q", getKeyText, "{ok}")
	}
}

// TestRenderIRCanonicalScalarFormattingCoversPrimitiveFastPaths verifies scalar formatting preserves stable text for supported primitive render values.
func TestRenderIRCanonicalScalarFormattingCoversPrimitiveFastPaths(parseT *testing.T) {
	parseScalarTests := []struct {
		name           string
		parseValue     any
		wantStringText string
	}{
		{name: "nil", parseValue: nil, wantStringText: ""},
		{name: "string", parseValue: "orders", wantStringText: "orders"},
		{name: "bool", parseValue: true, wantStringText: "true"},
		{name: "int", parseValue: int(-1), wantStringText: "-1"},
		{name: "int8", parseValue: int8(-2), wantStringText: "-2"},
		{name: "int16", parseValue: int16(-3), wantStringText: "-3"},
		{name: "int32", parseValue: int32(-4), wantStringText: "-4"},
		{name: "int64", parseValue: int64(-5), wantStringText: "-5"},
		{name: "uint", parseValue: uint(6), wantStringText: "6"},
		{name: "uint8", parseValue: uint8(7), wantStringText: "7"},
		{name: "uint16", parseValue: uint16(8), wantStringText: "8"},
		{name: "uint32", parseValue: uint32(9), wantStringText: "9"},
		{name: "uint64", parseValue: uint64(10), wantStringText: "10"},
		{name: "float32", parseValue: float32(1.25), wantStringText: "1.25"},
		{name: "float64", parseValue: 2.5, wantStringText: "2.5"},
		{name: "fallback", parseValue: struct{ Label string }{Label: "ok"}, wantStringText: "{ok}"},
	}
	for _, parseScalarTest := range parseScalarTests {
		if getStringText := parseFormatCanonicalScalarValue(parseScalarTest.parseValue); getStringText != parseScalarTest.wantStringText {
			parseT.Fatalf("%s canonical scalar string = %q, want %q", parseScalarTest.name, getStringText, parseScalarTest.wantStringText)
		}
	}
}

// TestRenderIRCanonicalJSONReportsMarshalFailures verifies canonical JSON fallback rendering preserves JSON text and wraps marshal failures with context.
func TestRenderIRCanonicalJSONReportsMarshalFailures(parseT *testing.T) {
	getCanonicalJSON, parseCanonicalJSONErr := parseBuildCanonicalJSON(map[string]any{"title": "Orders"})
	if parseCanonicalJSONErr != nil {
		parseT.Fatalf("parseBuildCanonicalJSON(valid) returned error: %v", parseCanonicalJSONErr)
	}
	if getCanonicalJSON != `{"title":"Orders"}` {
		parseT.Fatalf("canonical JSON = %q, want %q", getCanonicalJSON, `{"title":"Orders"}`)
	}
	if _, parseCanonicalJSONErr := parseBuildCanonicalJSON(make(chan int)); parseCanonicalJSONErr == nil {
		parseT.Fatal("expected canonical JSON marshal failure for channel value")
	} else if !strings.Contains(parseCanonicalJSONErr.Error(), "canonicalize render value") {
		parseT.Fatalf("expected canonical JSON marshal context, got %v", parseCanonicalJSONErr)
	}
}
