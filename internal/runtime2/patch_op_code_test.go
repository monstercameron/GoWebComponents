package runtime2

import "testing"

// TestParsePatchOpCodeSupportedOpCodesDecode verifies first-slice patch op codes decode successfully.
func TestParsePatchOpCodeSupportedOpCodesDecode(parseTesting *testing.T) {
	parseCases := []struct {
		parseName string
		parseRaw  uint8
		parseWant PatchOpCode
	}{
		{parseName: "insert", parseRaw: 1, parseWant: PatchOpCodeInsertNode},
		{parseName: "remove", parseRaw: 2, parseWant: PatchOpCodeRemoveNode},
		{parseName: "set-text", parseRaw: 3, parseWant: PatchOpCodeSetText},
		{parseName: "set-attr", parseRaw: 4, parseWant: PatchOpCodeSetAttr},
		{parseName: "remove-attr", parseRaw: 5, parseWant: PatchOpCodeRemoveAttr},
		{parseName: "set-style", parseRaw: 6, parseWant: PatchOpCodeSetStyle},
		{parseName: "remove-style", parseRaw: 7, parseWant: PatchOpCodeRemoveStyle},
		{parseName: "move-keyed", parseRaw: 8, parseWant: PatchOpCodeMoveKeyedChild},
		{parseName: "replace-subtree", parseRaw: 9, parseWant: PatchOpCodeReplaceSubtree},
	}
	for _, parseCase := range parseCases {
		parseCase := parseCase
		parseTesting.Run(parseCase.parseName, func(parseTesting *testing.T) {
			parseCode, parseErr := ParsePatchOpCode(parseCase.parseRaw)
			if parseErr != nil {
				parseTesting.Fatalf("ParsePatchOpCode(%d) error = %v", parseCase.parseRaw, parseErr)
			}
			if parseCode != parseCase.parseWant {
				parseTesting.Fatalf("ParsePatchOpCode(%d) = %v, want %v", parseCase.parseRaw, parseCode, parseCase.parseWant)
			}
		})
	}
}

// TestParsePatchOpCodeUnknownOpCodeFails verifies unknown patch op codes are rejected.
func TestParsePatchOpCodeUnknownOpCodeFails(parseTesting *testing.T) {
	_, parseErr := ParsePatchOpCode(99)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchOpCode(99) error = nil, want error")
	}
}

// TestParsePatchOpCodeZeroOrReservedCodeFails verifies zero/reserved patch op codes are rejected.
func TestParsePatchOpCodeZeroOrReservedCodeFails(parseTesting *testing.T) {
	_, parseErr := ParsePatchOpCode(0)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchOpCode(0) error = nil, want error")
	}
}
