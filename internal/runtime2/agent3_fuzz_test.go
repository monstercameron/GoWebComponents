package runtime2

import (
	"strings"
	"testing"
)

// FuzzParseCanonicalRenderTree fuzzes canonical render IR build plus decode to keep malformed shapes panic-safe.
func FuzzParseCanonicalRenderTree(parseF *testing.F) {
	parseF.Add("hello")
	parseF.Add("")
	parseF.Fuzz(func(parseT *testing.T, parseText string) {
		parseCanonicalIR, parseCanonicalErr := BuildCanonicalRenderIR(map[string]any{
			"kind": "text",
			"text": parseText,
		})
		if parseCanonicalErr != nil {
			return
		}
		_, _ = ParseCanonicalRenderTree(parseCanonicalIR)
	})
}

// FuzzParseRenderStringTable fuzzes canonical string-table decoding.
func FuzzParseRenderStringTable(parseF *testing.F) {
	parseF.Add("a|b|c")
	parseF.Add("b|a")
	parseF.Fuzz(func(parseT *testing.T, parseInput string) {
		parseEntries := strings.Split(parseInput, "|")
		_, _ = ParseRenderStringTable(parseEntries)
	})
}

// FuzzParseRenderPropRecords fuzzes prop-record decoding against randomized key/value and kind payloads.
func FuzzParseRenderPropRecords(parseF *testing.F) {
	parseF.Add("class", "active", uint8(RenderPropKindClass))
	parseF.Add("data-id", "42", uint8(RenderPropKindData))
	parseF.Fuzz(func(parseT *testing.T, parseKey string, parseValue string, parseKind uint8) {
		parseStringTable := BuildRenderStringTable([]string{parseKey, parseValue})
		parseKeyRef, hasKeyRef := parseStringTable.GetRenderStringRef(parseKey)
		parseValueRef, hasValueRef := parseStringTable.GetRenderStringRef(parseValue)
		if !hasKeyRef || !hasValueRef {
			return
		}
		parseRawRecords := []RenderPropRecordRaw{
			{
				Kind:     parseKind,
				KeyRef:   parseKeyRef,
				ValueRef: parseValueRef,
			},
		}
		_, _ = ParseRenderPropRecords(parseRawRecords, parseStringTable)
	})
}

// FuzzParsePatchStreamTransaction fuzzes patch-stream parsing and validation.
func FuzzParsePatchStreamTransaction(parseF *testing.F) {
	parseF.Add("region-a", "before", "after", byte(PatchOpCodeSetText))
	parseF.Fuzz(func(parseT *testing.T, parseRegionID string, parseBefore string, parseAfter string, parseOpCode byte) {
		if strings.TrimSpace(parseRegionID) == "" {
			parseRegionID = "region-a"
		}
		parsePreviousIR, parsePreviousErr := BuildCanonicalRenderIR(map[string]any{"kind": "text", "text": parseBefore})
		if parsePreviousErr != nil {
			return
		}
		parseNextIR, parseNextErr := BuildCanonicalRenderIR(map[string]any{"kind": "text", "text": parseAfter})
		if parseNextErr != nil {
			return
		}
		parsePatchStream, hasNoOp, parsePatchErr := BuildCanonicalPatchStream(parseRegionID, 1, 2, 2, parsePreviousIR, parseNextIR)
		if parsePatchErr != nil || hasNoOp {
			return
		}
		if len(parsePatchStream.GetOps) > 0 {
			parsePatchStream.GetOps[0].GetOpCode = parseOpCode
		}
		parsePreviousTree, parseTreeErr := ParseCanonicalRenderTree(parsePreviousIR)
		if parseTreeErr != nil {
			return
		}
		parseKnownNodeIDs := make(map[uint64]struct{}, len(parsePreviousTree.getNodeByID))
		for getNodeID := range parsePreviousTree.getNodeByID {
			parseKnownNodeIDs[getNodeID] = struct{}{}
		}
		_, _, _ = ParsePatchStreamTransaction(
			parsePatchStream,
			parseRegionID,
			1,
			parseKnownNodeIDs,
			map[uint64]uint32{},
			BuildPatchIdempotencyTracker(),
		)
	})
}

// FuzzPatchDOMPrevalidation fuzzes patch parse plus transaction commit prevalidation against a seeded DOM index.
func FuzzPatchDOMPrevalidation(parseF *testing.F) {
	parseF.Add("before", "after")
	parseF.Fuzz(func(parseT *testing.T, parseBefore string, parseAfter string) {
		parseRegionID := "region-fuzz"
		parsePreviousIR, parsePreviousErr := BuildCanonicalRenderIR(map[string]any{"kind": "text", "text": parseBefore})
		if parsePreviousErr != nil {
			return
		}
		parseNextIR, parseNextErr := BuildCanonicalRenderIR(map[string]any{"kind": "text", "text": parseAfter})
		if parseNextErr != nil {
			return
		}
		parsePatchStream, hasNoOp, parsePatchErr := BuildCanonicalPatchStream(parseRegionID, 1, 2, 2, parsePreviousIR, parseNextIR)
		if parsePatchErr != nil || hasNoOp {
			return
		}
		parseDOMIndex := BuildRegionDOMIndex()
		parseSeedRegionDOMIndexFromCanonical(parseT, parseDOMIndex, parseRegionID, parsePreviousIR)
		parseDOMCommitter := BuildDOMCommitter(parseDOMIndex)
		parseKnownNodeIDs := BuildKnownNodeIDsForRegionDOMIndex(parseDOMIndex, parseRegionID)
		parseSiblingCountByParent := BuildSiblingCountByParentForRegionDOMIndex(parseDOMIndex, parseRegionID)
		parsePatchResult, hasApply, parseParseErr := ParsePatchStreamTransaction(
			parsePatchStream,
			parseRegionID,
			1,
			parseKnownNodeIDs,
			parseSiblingCountByParent,
			BuildPatchIdempotencyTracker(),
		)
		if parseParseErr != nil || !hasApply {
			return
		}
		_, _ = parseDOMCommitter.CommitRegionPatchTransaction(parsePatchResult.GetTransaction)
	})
}
