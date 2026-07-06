package runtime2

import "fmt"

// PatchKeyedMoveOpRaw stores the wire-format payload for one keyed-move patch op.
type PatchKeyedMoveOpRaw struct {
	ParentNodeID     uint64
	SourceNodeID     uint64
	DestinationIndex uint32
}

// PatchKeyedMoveOp stores one validated keyed-move patch op.
type PatchKeyedMoveOp struct {
	ParentNodeID     uint64
	SourceNodeID     uint64
	DestinationIndex uint32
}

// ParsePatchKeyedMoveOp decodes and validates one keyed-move patch op payload.
func ParsePatchKeyedMoveOp(parseRaw PatchKeyedMoveOpRaw, parseKnownNodeIDs map[uint64]struct{}, parseSiblingCountByParent map[uint64]uint32) (PatchKeyedMoveOp, error) {
	if parseRaw.ParentNodeID == 0 {
		return PatchKeyedMoveOp{}, fmt.Errorf("runtime2: keyed-move op parent node id is required")
	}
	if _, hasParentNodeID := parseKnownNodeIDs[parseRaw.ParentNodeID]; !hasParentNodeID {
		return PatchKeyedMoveOp{}, fmt.Errorf("runtime2: keyed-move op parent node id %d is unknown", parseRaw.ParentNodeID)
	}
	if parseRaw.SourceNodeID == 0 {
		return PatchKeyedMoveOp{}, fmt.Errorf("runtime2: keyed-move op source node id is required")
	}
	if _, hasSourceNodeID := parseKnownNodeIDs[parseRaw.SourceNodeID]; !hasSourceNodeID {
		return PatchKeyedMoveOp{}, fmt.Errorf("runtime2: keyed-move op source node id %d is unknown", parseRaw.SourceNodeID)
	}
	getSiblingCount, hasSiblingCount := parseSiblingCountByParent[parseRaw.ParentNodeID]
	if !hasSiblingCount {
		return PatchKeyedMoveOp{}, fmt.Errorf("runtime2: keyed-move op parent node id %d has no sibling bounds", parseRaw.ParentNodeID)
	}
	// The applier removes the moved node before reinserting, so the valid
	// destination range is 0..siblingCount-1; DestinationIndex == siblingCount
	// passed validation here but always failed at commit (wasted rollback +
	// fallback cycle). Reject it at parse time to match the applier's ceiling.
	if parseRaw.DestinationIndex >= getSiblingCount {
		return PatchKeyedMoveOp{}, fmt.Errorf("runtime2: keyed-move op destination index %d is out of range for sibling count %d", parseRaw.DestinationIndex, getSiblingCount)
	}
	return PatchKeyedMoveOp(parseRaw), nil
}
