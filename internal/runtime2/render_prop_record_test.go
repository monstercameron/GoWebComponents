package runtime2

import "testing"

// TestParseRenderPropRecordSupportedPropKindsRoundTrip verifies supported prop kinds decode successfully.
func TestParseRenderPropRecordSupportedPropKindsRoundTrip(parseTesting *testing.T) {
	buildStringTable := BuildRenderStringTable([]string{"class", "style", "aria-label", "data-id", "text", "value"})
	getValueRef, hasValueRef := buildStringTable.GetRenderStringRef("value")
	if !hasValueRef {
		parseTesting.Fatal("GetRenderStringRef(value) did not find value entry")
	}
	parseCases := []struct {
		parseName string
		parseKind RenderPropKind
		parseKey  string
	}{
		{parseName: "class", parseKind: RenderPropKindClass, parseKey: "class"},
		{parseName: "style", parseKind: RenderPropKindStyle, parseKey: "style"},
		{parseName: "aria", parseKind: RenderPropKindAria, parseKey: "aria-label"},
		{parseName: "data", parseKind: RenderPropKindData, parseKey: "data-id"},
		{parseName: "text", parseKind: RenderPropKindTextAdjacent, parseKey: "text"},
	}
	for _, parseCase := range parseCases {
		parseCase := parseCase
		parseTesting.Run(parseCase.parseName, func(parseTesting *testing.T) {
			getKeyRef, hasKeyRef := buildStringTable.GetRenderStringRef(parseCase.parseKey)
			if !hasKeyRef {
				parseTesting.Fatalf("GetRenderStringRef(%s) did not find key entry", parseCase.parseKey)
			}
			parseRawRecord := RenderPropRecordRaw{
				Kind:     uint8(parseCase.parseKind),
				KeyRef:   getKeyRef,
				ValueRef: getValueRef,
			}
			parseRecord, parseErr := ParseRenderPropRecord(parseRawRecord, buildStringTable)
			if parseErr != nil {
				parseTesting.Fatalf("ParseRenderPropRecord(%s) error = %v", parseCase.parseName, parseErr)
			}
			if parseRecord.Kind != parseCase.parseKind {
				parseTesting.Fatalf("ParseRenderPropRecord(%s) Kind = %v, want %v", parseCase.parseName, parseRecord.Kind, parseCase.parseKind)
			}
			if parseRecord.Key != parseCase.parseKey {
				parseTesting.Fatalf("ParseRenderPropRecord(%s) Key = %q, want %q", parseCase.parseName, parseRecord.Key, parseCase.parseKey)
			}
			if parseRecord.Value != "value" {
				parseTesting.Fatalf("ParseRenderPropRecord(%s) Value = %q, want %q", parseCase.parseName, parseRecord.Value, "value")
			}
		})
	}
}

// TestParseRenderPropRecordUnsupportedPropKindFails verifies unsupported prop kinds are rejected.
func TestParseRenderPropRecordUnsupportedPropKindFails(parseTesting *testing.T) {
	buildStringTable := BuildRenderStringTable([]string{"class", "value"})
	getClassRef, hasClassRef := buildStringTable.GetRenderStringRef("class")
	if !hasClassRef {
		parseTesting.Fatal("GetRenderStringRef(class) did not find class entry")
	}
	getValueRef, hasValueRef := buildStringTable.GetRenderStringRef("value")
	if !hasValueRef {
		parseTesting.Fatal("GetRenderStringRef(value) did not find value entry")
	}
	parseRawRecord := RenderPropRecordRaw{
		Kind:     255,
		KeyRef:   getClassRef,
		ValueRef: getValueRef,
	}
	_, parseErr := ParseRenderPropRecord(parseRawRecord, buildStringTable)
	if parseErr == nil {
		parseTesting.Fatal("ParseRenderPropRecord(unsupported kind) error = nil, want error")
	}
}

// TestParseRenderPropRecordMissingKeyReferenceFails verifies missing key references are rejected.
func TestParseRenderPropRecordMissingKeyReferenceFails(parseTesting *testing.T) {
	buildStringTable := BuildRenderStringTable([]string{"value"})
	getValueRef, hasValueRef := buildStringTable.GetRenderStringRef("value")
	if !hasValueRef {
		parseTesting.Fatal("GetRenderStringRef(value) did not find value entry")
	}
	parseRawRecord := RenderPropRecordRaw{
		Kind:     uint8(RenderPropKindClass),
		KeyRef:   99,
		ValueRef: getValueRef,
	}
	_, parseErr := ParseRenderPropRecord(parseRawRecord, buildStringTable)
	if parseErr == nil {
		parseTesting.Fatal("ParseRenderPropRecord(missing key ref) error = nil, want error")
	}
}
