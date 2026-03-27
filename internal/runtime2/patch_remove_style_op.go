package runtime2

import "fmt"

// PatchRemoveStyleOpRaw stores the wire-format payload for one remove-style patch op.
type PatchRemoveStyleOpRaw struct {
	TargetNodeID uint64
}

// PatchRemoveStyleOp stores one validated remove-style patch op.
type PatchRemoveStyleOp struct {
	TargetNodeID uint64
}

// ParsePatchRemoveStyleOp decodes and validates one remove-style patch op payload.
func ParsePatchRemoveStyleOp(parseRaw PatchRemoveStyleOpRaw, parseKnownNodeIDs map[uint64]struct{}) (PatchRemoveStyleOp, error) {
	if parseRaw.TargetNodeID == 0 {
		return PatchRemoveStyleOp{}, fmt.Errorf("runtime2: remove-style op target node id is required")
	}
	if _, hasKnownNodeID := parseKnownNodeIDs[parseRaw.TargetNodeID]; !hasKnownNodeID {
		return PatchRemoveStyleOp{}, fmt.Errorf("runtime2: remove-style op target node id %d is unknown", parseRaw.TargetNodeID)
	}
	return PatchRemoveStyleOp{
		TargetNodeID: parseRaw.TargetNodeID,
	}, nil
}

