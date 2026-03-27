package runtime2

import "fmt"

// PatchSetStyleOpRaw stores the wire-format payload for one set-style patch op.
type PatchSetStyleOpRaw struct {
	TargetNodeID uint64
	StyleRef     uint32
}

// PatchSetStyleOp stores one validated set-style patch op.
type PatchSetStyleOp struct {
	TargetNodeID uint64
	StyleValue   string
}

// ParsePatchSetStyleOp decodes and validates one set-style patch op payload.
func ParsePatchSetStyleOp(parseRaw PatchSetStyleOpRaw, parseKnownNodeIDs map[uint64]struct{}, parseStringTable RenderStringTable) (PatchSetStyleOp, error) {
	if parseRaw.TargetNodeID == 0 {
		return PatchSetStyleOp{}, fmt.Errorf("runtime2: set-style op target node id is required")
	}
	if _, hasKnownNodeID := parseKnownNodeIDs[parseRaw.TargetNodeID]; !hasKnownNodeID {
		return PatchSetStyleOp{}, fmt.Errorf("runtime2: set-style op target node id %d is unknown", parseRaw.TargetNodeID)
	}
	getStyleValue, getStyleErr := parseStringTable.GetRenderStringByRef(parseRaw.StyleRef)
	if getStyleErr != nil {
		return PatchSetStyleOp{}, fmt.Errorf("runtime2: set-style op style reference %d is invalid: %w", parseRaw.StyleRef, getStyleErr)
	}
	return PatchSetStyleOp{
		TargetNodeID: parseRaw.TargetNodeID,
		StyleValue:   getStyleValue,
	}, nil
}

