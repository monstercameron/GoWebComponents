package runtime2

import "fmt"

// PatchInsertOpRaw stores the wire-format payload for one insert patch op.
type PatchInsertOpRaw struct {
	ParentNodeID uint64
	AnchorNodeID uint64
	Node         RenderNodeRecordRaw
}

// PatchInsertOp stores one validated insert patch op.
type PatchInsertOp struct {
	ParentNodeID uint64
	AnchorNodeID uint64
	Node         RenderNodeRecord
}

// ParsePatchInsertOp decodes and validates one insert patch op payload.
func ParsePatchInsertOp(parseRaw PatchInsertOpRaw, parseKnownNodeIDs map[uint64]struct{}) (PatchInsertOp, error) {
	if parseRaw.ParentNodeID == 0 {
		return PatchInsertOp{}, fmt.Errorf("runtime2: insert op parent reference is required")
	}
	if _, hasParentNodeID := parseKnownNodeIDs[parseRaw.ParentNodeID]; !hasParentNodeID {
		return PatchInsertOp{}, fmt.Errorf("runtime2: insert op parent node id %d is unknown", parseRaw.ParentNodeID)
	}
	if parseRaw.AnchorNodeID != 0 {
		if _, hasAnchorNodeID := parseKnownNodeIDs[parseRaw.AnchorNodeID]; !hasAnchorNodeID {
			return PatchInsertOp{}, fmt.Errorf("runtime2: insert op anchor node id %d is unknown", parseRaw.AnchorNodeID)
		}
	}
	parseNodeRecord, parseErr := ParseRenderNodeRecord(parseRaw.Node)
	if parseErr != nil {
		return PatchInsertOp{}, fmt.Errorf("runtime2: insert op node payload is invalid: %w", parseErr)
	}
	return PatchInsertOp{
		ParentNodeID: parseRaw.ParentNodeID,
		AnchorNodeID: parseRaw.AnchorNodeID,
		Node:         parseNodeRecord,
	}, nil
}
