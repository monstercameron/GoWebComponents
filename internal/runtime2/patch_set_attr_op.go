package runtime2

import "fmt"

// PatchSetAttrOpRaw stores the wire-format payload for one set-attr patch op.
type PatchSetAttrOpRaw struct {
	TargetNodeID uint64
	Attr         RenderPropRecordRaw
}

// PatchSetAttrOp stores one validated set-attr patch op.
type PatchSetAttrOp struct {
	TargetNodeID uint64
	Attr         RenderPropRecord
}

// ParsePatchSetAttrOp decodes and validates one set-attr patch op payload.
func ParsePatchSetAttrOp(parseRaw PatchSetAttrOpRaw, parseKnownNodeIDs map[uint64]struct{}, parseStringTable RenderStringTable) (PatchSetAttrOp, error) {
	if parseRaw.TargetNodeID == 0 {
		return PatchSetAttrOp{}, fmt.Errorf("runtime2: set-attr op target node id is required")
	}
	if _, hasKnownNodeID := parseKnownNodeIDs[parseRaw.TargetNodeID]; !hasKnownNodeID {
		return PatchSetAttrOp{}, fmt.Errorf("runtime2: set-attr op target node id %d is unknown", parseRaw.TargetNodeID)
	}
	parseAttrRecord, parseAttrErr := ParseRenderPropRecord(parseRaw.Attr, parseStringTable)
	if parseAttrErr != nil {
		return PatchSetAttrOp{}, fmt.Errorf("runtime2: set-attr payload is invalid: %w", parseAttrErr)
	}
	if parseAttrRecord.Kind == RenderPropKindTextAdjacent {
		return PatchSetAttrOp{}, fmt.Errorf("runtime2: set-attr op does not support prop kind %v", parseAttrRecord.Kind)
	}
	return PatchSetAttrOp{
		TargetNodeID: parseRaw.TargetNodeID,
		Attr:         parseAttrRecord,
	}, nil
}
