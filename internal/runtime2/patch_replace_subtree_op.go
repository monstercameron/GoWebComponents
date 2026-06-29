package runtime2

import "fmt"

// PatchReplaceSubtreePayloadRaw stores one replacement-subtree payload in wire format.
type PatchReplaceSubtreePayloadRaw struct {
	RootNodeID  uint64                `json:"root_node_id"`
	StringTable []string              `json:"string_table,omitempty"`
	NodeRecords []RenderNodeRecordRaw `json:"node_records"`
	PropRecords []RenderPropRecordRaw `json:"prop_records,omitempty"`
}

// PatchReplaceSubtreeOpRaw stores the wire-format payload for one replace-subtree patch op.
type PatchReplaceSubtreeOpRaw struct {
	TargetNodeID uint64                        `json:"target_node_id"`
	Subtree      PatchReplaceSubtreePayloadRaw `json:"subtree"`
}

// PatchReplaceSubtreeOp stores one validated replace-subtree patch op.
type PatchReplaceSubtreeOp struct {
	TargetNodeID uint64
	SubtreeIR    CanonicalRenderIR
}

// ParsePatchReplaceSubtreeOp decodes and validates one replace-subtree patch op payload.
func ParsePatchReplaceSubtreeOp(parseRaw PatchReplaceSubtreeOpRaw, parseKnownNodeIDs map[uint64]struct{}) (PatchReplaceSubtreeOp, error) {
	if parseRaw.TargetNodeID == 0 {
		return PatchReplaceSubtreeOp{}, fmt.Errorf("runtime2: replace-subtree op target node id is required")
	}
	if _, hasKnownNodeID := parseKnownNodeIDs[parseRaw.TargetNodeID]; !hasKnownNodeID {
		return PatchReplaceSubtreeOp{}, fmt.Errorf("runtime2: replace-subtree op target node id %d is unknown", parseRaw.TargetNodeID)
	}
	parseStringTable, parseStringTableErr := ParseRenderStringTable(parseRaw.Subtree.StringTable)
	if parseStringTableErr != nil {
		return PatchReplaceSubtreeOp{}, fmt.Errorf("runtime2: replace-subtree string table is invalid: %w", parseStringTableErr)
	}
	parseSubtreeIR := CanonicalRenderIR{
		GetRootNodeID:  parseRaw.Subtree.RootNodeID,
		GetStringTable: parseStringTable,
		GetNodeRecords: append([]RenderNodeRecordRaw(nil), parseRaw.Subtree.NodeRecords...),
		GetPropRecords: append([]RenderPropRecordRaw(nil), parseRaw.Subtree.PropRecords...),
	}
	if _, parseSubtreeErr := ParseCanonicalRenderTree(parseSubtreeIR); parseSubtreeErr != nil {
		return PatchReplaceSubtreeOp{}, fmt.Errorf("runtime2: replace-subtree payload is invalid: %w", parseSubtreeErr)
	}
	return PatchReplaceSubtreeOp{
		TargetNodeID: parseRaw.TargetNodeID,
		SubtreeIR:    parseSubtreeIR,
	}, nil
}
