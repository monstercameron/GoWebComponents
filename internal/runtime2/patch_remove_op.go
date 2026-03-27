package runtime2

import "fmt"

// PatchRemoveOpRaw stores the wire-format payload for one remove patch op.
type PatchRemoveOpRaw struct {
	TargetNodeID uint64
}

// PatchRemoveOp stores one validated remove patch op.
type PatchRemoveOp struct {
	TargetNodeID uint64
}

// ParsePatchRemoveOp decodes and validates one remove patch op payload.
func ParsePatchRemoveOp(parseRaw PatchRemoveOpRaw, parseKnownNodeIDs map[uint64]struct{}, parseRemovedNodeIDs map[uint64]struct{}) (PatchRemoveOp, error) {
	if parseRaw.TargetNodeID == 0 {
		return PatchRemoveOp{}, fmt.Errorf("runtime2: remove op target node id is required")
	}
	if _, hasKnownNodeID := parseKnownNodeIDs[parseRaw.TargetNodeID]; !hasKnownNodeID {
		return PatchRemoveOp{}, fmt.Errorf("runtime2: remove op target node id %d is unknown", parseRaw.TargetNodeID)
	}
	if _, hasRemovedNodeID := parseRemovedNodeIDs[parseRaw.TargetNodeID]; hasRemovedNodeID {
		return PatchRemoveOp{}, fmt.Errorf("runtime2: remove op target node id %d was already removed", parseRaw.TargetNodeID)
	}
	parseRemovedNodeIDs[parseRaw.TargetNodeID] = struct{}{}
	return PatchRemoveOp{TargetNodeID: parseRaw.TargetNodeID}, nil
}
