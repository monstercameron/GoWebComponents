package runtime2

import "fmt"

// PatchSetTextOpRaw stores the wire-format payload for one set-text patch op.
type PatchSetTextOpRaw struct {
	TargetNodeID uint64
	TextRef      uint32
}

// PatchSetTextOp stores one validated set-text patch op.
type PatchSetTextOp struct {
	TargetNodeID uint64
	Text         string
}

// ParsePatchSetTextOp decodes and validates one set-text patch op payload.
func ParsePatchSetTextOp(parseRaw PatchSetTextOpRaw, parseKnownNodeIDs map[uint64]struct{}, parseStringTable RenderStringTable) (PatchSetTextOp, error) {
	if parseRaw.TargetNodeID == 0 {
		return PatchSetTextOp{}, fmt.Errorf("runtime2: set-text op target node id is required")
	}
	if _, hasKnownNodeID := parseKnownNodeIDs[parseRaw.TargetNodeID]; !hasKnownNodeID {
		return PatchSetTextOp{}, fmt.Errorf("runtime2: set-text op target node id %d is unknown", parseRaw.TargetNodeID)
	}
	getText, getTextErr := parseStringTable.GetRenderStringByRef(parseRaw.TextRef)
	if getTextErr != nil {
		return PatchSetTextOp{}, fmt.Errorf("runtime2: set-text op text reference %d is invalid: %w", parseRaw.TextRef, getTextErr)
	}
	return PatchSetTextOp{
		TargetNodeID: parseRaw.TargetNodeID,
		Text:         getText,
	}, nil
}
