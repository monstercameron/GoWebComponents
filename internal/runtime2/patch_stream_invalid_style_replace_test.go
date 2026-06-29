package runtime2

import (
	"strings"
	"testing"
)

func parseBuildInvalidPatchStream(parseOp PatchStreamOpRaw) PatchStreamRaw {
	return PatchStreamRaw{
		GetHeader: PatchStreamHeaderRaw{
			ProtocolVersion: PatchStreamProtocolVersion,
			RegionID:        "region-a",
			Epoch:           1,
			InputVersion:    2,
			PatchVersion:    2,
		},
		GetPatchIdentity: "invalid-op-test",
		GetOps:           []PatchStreamOpRaw{parseOp},
	}
}

// TestParsePatchStreamTransactionRejectsMissingSetStylePayload verifies missing set-style payloads fail parse validation.
func TestParsePatchStreamTransactionRejectsMissingSetStylePayload(parseTesting *testing.T) {
	parsePatchStream := parseBuildInvalidPatchStream(PatchStreamOpRaw{
		GetOpCode: uint8(PatchOpCodeSetStyle),
	})
	_, _, parseErr := ParsePatchStreamTransaction(
		parsePatchStream,
		"region-a",
		1,
		map[uint64]struct{}{1: {}},
		map[uint64]uint32{},
		nil,
	)
	if parseErr == nil {
		parseTesting.Fatal("expected missing set-style payload to fail")
	}
	if !strings.Contains(parseErr.Error(), "set-style payload is required") {
		parseTesting.Fatalf("expected set-style payload error, got %v", parseErr)
	}
}

// TestParsePatchStreamTransactionRejectsMissingRemoveStylePayload verifies missing remove-style payloads fail parse validation.
func TestParsePatchStreamTransactionRejectsMissingRemoveStylePayload(parseTesting *testing.T) {
	parsePatchStream := parseBuildInvalidPatchStream(PatchStreamOpRaw{
		GetOpCode: uint8(PatchOpCodeRemoveStyle),
	})
	_, _, parseErr := ParsePatchStreamTransaction(
		parsePatchStream,
		"region-a",
		1,
		map[uint64]struct{}{1: {}},
		map[uint64]uint32{},
		nil,
	)
	if parseErr == nil {
		parseTesting.Fatal("expected missing remove-style payload to fail")
	}
	if !strings.Contains(parseErr.Error(), "remove-style payload is required") {
		parseTesting.Fatalf("expected remove-style payload error, got %v", parseErr)
	}
}

// TestParsePatchStreamTransactionRejectsInvalidReplaceSubtreePayload verifies malformed replace-subtree payloads fail parse validation.
func TestParsePatchStreamTransactionRejectsInvalidReplaceSubtreePayload(parseTesting *testing.T) {
	parsePatchStream := parseBuildInvalidPatchStream(PatchStreamOpRaw{
		GetOpCode: uint8(PatchOpCodeReplaceSubtree),
		GetReplaceSubtreeOp: &PatchReplaceSubtreeOpRaw{
			TargetNodeID: 1,
			Subtree: PatchReplaceSubtreePayloadRaw{
				RootNodeID:  1,
				StringTable: []string{"x"},
				NodeRecords: []RenderNodeRecordRaw{},
			},
		},
	})
	_, _, parseErr := ParsePatchStreamTransaction(
		parsePatchStream,
		"region-a",
		1,
		map[uint64]struct{}{1: {}},
		map[uint64]uint32{},
		nil,
	)
	if parseErr == nil {
		parseTesting.Fatal("expected invalid replace-subtree payload to fail")
	}
	if !strings.Contains(parseErr.Error(), "replace-subtree is invalid") {
		parseTesting.Fatalf("expected replace-subtree payload error, got %v", parseErr)
	}
}
