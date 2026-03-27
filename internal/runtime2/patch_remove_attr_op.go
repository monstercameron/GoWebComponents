package runtime2

import "fmt"

// PatchRemoveAttrOpRaw stores the wire-format payload for one remove-attr patch op.
type PatchRemoveAttrOpRaw struct {
	TargetNodeID uint64
	KeyRef       uint32
}

// PatchRemoveAttrOp stores one validated remove-attr patch op.
type PatchRemoveAttrOp struct {
	TargetNodeID uint64
	Key          string
}

// ParsePatchRemoveAttrOp decodes and validates one remove-attr patch op payload.
func ParsePatchRemoveAttrOp(parseRaw PatchRemoveAttrOpRaw, parseKnownNodeIDs map[uint64]struct{}, parseStringTable RenderStringTable, parseRemovedAttrKeys map[string]struct{}) (PatchRemoveAttrOp, error) {
	if parseRaw.TargetNodeID == 0 {
		return PatchRemoveAttrOp{}, fmt.Errorf("runtime2: remove-attr op target node id is required")
	}
	if _, hasKnownNodeID := parseKnownNodeIDs[parseRaw.TargetNodeID]; !hasKnownNodeID {
		return PatchRemoveAttrOp{}, fmt.Errorf("runtime2: remove-attr op target node id %d is unknown", parseRaw.TargetNodeID)
	}
	getKey, getKeyErr := parseStringTable.GetRenderStringByRef(parseRaw.KeyRef)
	if getKeyErr != nil {
		return PatchRemoveAttrOp{}, fmt.Errorf("runtime2: remove-attr op key reference %d is invalid: %w", parseRaw.KeyRef, getKeyErr)
	}
	parseAttrIdentifier := fmt.Sprintf("%d:%s", parseRaw.TargetNodeID, getKey)
	if _, hasRemovedAttrKey := parseRemovedAttrKeys[parseAttrIdentifier]; hasRemovedAttrKey {
		return PatchRemoveAttrOp{}, fmt.Errorf("runtime2: remove-attr op key %q for node id %d was already removed", getKey, parseRaw.TargetNodeID)
	}
	parseRemovedAttrKeys[parseAttrIdentifier] = struct{}{}
	return PatchRemoveAttrOp{
		TargetNodeID: parseRaw.TargetNodeID,
		Key:          getKey,
	}, nil
}
