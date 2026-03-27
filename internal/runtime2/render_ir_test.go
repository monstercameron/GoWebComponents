package runtime2

import "testing"

// TestBuildCanonicalRenderIRRootConventions verifies root conventions for empty, single-text, and host-element outputs.
func TestBuildCanonicalRenderIRRootConventions(parseTesting *testing.T) {
	parseCases := []struct {
		parseName         string
		parseRenderOutput any
		parseWantKind     RenderNodeKind
	}{
		{
			parseName:         "empty",
			parseRenderOutput: nil,
			parseWantKind:     RenderNodeKindFragment,
		},
		{
			parseName:         "single-text",
			parseRenderOutput: "hello",
			parseWantKind:     RenderNodeKindText,
		},
		{
			parseName: "host-element",
			parseRenderOutput: map[string]any{
				"kind": "host-element",
				"tag":  "div",
			},
			parseWantKind: RenderNodeKindHostElement,
		},
	}
	for _, parseCase := range parseCases {
		parseCase := parseCase
		parseTesting.Run(parseCase.parseName, func(parseTesting *testing.T) {
			parseCanonicalIR, parseCanonicalErr := BuildCanonicalRenderIR(parseCase.parseRenderOutput)
			if parseCanonicalErr != nil {
				parseTesting.Fatalf("BuildCanonicalRenderIR(%s) error = %v", parseCase.parseName, parseCanonicalErr)
			}
			if len(parseCanonicalIR.GetNodeRecords) == 0 {
				parseTesting.Fatalf("BuildCanonicalRenderIR(%s) returned no node records", parseCase.parseName)
			}
			parseRootRecord := parseCanonicalIR.GetNodeRecords[0]
			if RenderNodeKind(parseRootRecord.Kind) != parseCase.parseWantKind {
				parseTesting.Fatalf(
					"BuildCanonicalRenderIR(%s) root kind = %v, want %v",
					parseCase.parseName,
					RenderNodeKind(parseRootRecord.Kind),
					parseCase.parseWantKind,
				)
			}
		})
	}
}

// TestBuildRenderStringTableFromRenderOutputCanonicalOrdering verifies extracted string tables are canonical.
func TestBuildRenderStringTableFromRenderOutputCanonicalOrdering(parseTesting *testing.T) {
	parseStringTable, parseStringTableErr := BuildRenderStringTableFromRenderOutput(map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"props": map[string]any{
			"data-z": "z",
			"class":  "a",
		},
		"children": []any{
			map[string]any{
				"kind": "text",
				"text": "hello",
			},
		},
	})
	if parseStringTableErr != nil {
		parseTesting.Fatalf("BuildRenderStringTableFromRenderOutput returned error: %v", parseStringTableErr)
	}
	if len(parseStringTable.Entries) == 0 {
		parseTesting.Fatal("BuildRenderStringTableFromRenderOutput expected at least one string entry")
	}
	for parseIndex := 1; parseIndex < len(parseStringTable.Entries); parseIndex++ {
		if parseStringTable.Entries[parseIndex-1] > parseStringTable.Entries[parseIndex] {
			parseTesting.Fatalf("string table is not sorted at index %d", parseIndex)
		}
	}
}

// TestBuildRenderPropRecordsFromRenderOutputCanonicalProps verifies canonical prop extraction for one host element.
func TestBuildRenderPropRecordsFromRenderOutputCanonicalProps(parseTesting *testing.T) {
	parsePropRecords, parseStringTable, parsePropErr := BuildRenderPropRecordsFromRenderOutput(map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"props": map[string]any{
			"data-id": "42",
			"class":   "active",
		},
	})
	if parsePropErr != nil {
		parseTesting.Fatalf("BuildRenderPropRecordsFromRenderOutput returned error: %v", parsePropErr)
	}
	if len(parsePropRecords) != 2 {
		parseTesting.Fatalf("BuildRenderPropRecordsFromRenderOutput prop count = %d, want 2", len(parsePropRecords))
	}
	parseDecodedProps, parseDecodeErr := ParseRenderPropRecords(parsePropRecords, parseStringTable)
	if parseDecodeErr != nil {
		parseTesting.Fatalf("ParseRenderPropRecords returned error: %v", parseDecodeErr)
	}
	if parseDecodedProps[0].Key != "class" || parseDecodedProps[1].Key != "data-id" {
		parseTesting.Fatalf("decoded prop ordering = %q, %q; want class then data-id", parseDecodedProps[0].Key, parseDecodedProps[1].Key)
	}
}

// TestBuildCanonicalRenderIRStableKeyedNodeIDsAcrossReorder verifies keyed sibling node IDs stay stable across reorder updates.
func TestBuildCanonicalRenderIRStableKeyedNodeIDsAcrossReorder(parseTesting *testing.T) {
	parseBuildNodeIDsByKey := func(parseRenderOutput any) map[string]uint64 {
		parseCanonicalIR, parseCanonicalErr := BuildCanonicalRenderIR(parseRenderOutput)
		if parseCanonicalErr != nil {
			parseTesting.Fatalf("BuildCanonicalRenderIR returned error: %v", parseCanonicalErr)
		}
		parseTree, parseTreeErr := ParseCanonicalRenderTree(parseCanonicalIR)
		if parseTreeErr != nil {
			parseTesting.Fatalf("ParseCanonicalRenderTree returned error: %v", parseTreeErr)
		}
		buildNodeIDsByKey := map[string]uint64{}
		for getNodeID, getNode := range parseTree.getNodeByID {
			if getNode.getKey == "" {
				continue
			}
			buildNodeIDsByKey[getNode.getKey] = getNodeID
		}
		return buildNodeIDsByKey
	}
	parseFirstNodeIDsByKey := parseBuildNodeIDsByKey(map[string]any{
		"kind": "host-element",
		"tag":  "ul",
		"children": []any{
			map[string]any{"kind": "host-element", "tag": "li", "key": "a"},
			map[string]any{"kind": "host-element", "tag": "li", "key": "b"},
		},
	})
	parseSecondNodeIDsByKey := parseBuildNodeIDsByKey(map[string]any{
		"kind": "host-element",
		"tag":  "ul",
		"children": []any{
			map[string]any{"kind": "host-element", "tag": "li", "key": "b"},
			map[string]any{"kind": "host-element", "tag": "li", "key": "a"},
		},
	})
	if parseFirstNodeIDsByKey["a"] != parseSecondNodeIDsByKey["a"] {
		parseTesting.Fatalf("keyed node ID for key=a changed across reorder (%d vs %d)", parseFirstNodeIDsByKey["a"], parseSecondNodeIDsByKey["a"])
	}
	if parseFirstNodeIDsByKey["b"] != parseSecondNodeIDsByKey["b"] {
		parseTesting.Fatalf("keyed node ID for key=b changed across reorder (%d vs %d)", parseFirstNodeIDsByKey["b"], parseSecondNodeIDsByKey["b"])
	}
}

// TestSetCanonicalRenderIRInvariantValidationEnabledToggles verifies canonical IR invariant validation can be toggled for strict debug checks.
func TestSetCanonicalRenderIRInvariantValidationEnabledToggles(parseTesting *testing.T) {
	parseOriginalState := HasCanonicalRenderIRInvariantValidationEnabled()
	SetCanonicalRenderIRInvariantValidationEnabled(false)
	if HasCanonicalRenderIRInvariantValidationEnabled() {
		parseTesting.Fatal("expected canonical IR invariant validation gate disabled")
	}
	SetCanonicalRenderIRInvariantValidationEnabled(true)
	if !HasCanonicalRenderIRInvariantValidationEnabled() {
		parseTesting.Fatal("expected canonical IR invariant validation gate enabled")
	}
	SetCanonicalRenderIRInvariantValidationEnabled(parseOriginalState)
}
