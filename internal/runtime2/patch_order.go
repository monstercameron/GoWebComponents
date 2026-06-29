package runtime2

import "fmt"

// PatchOrderEntry stores the ordering-relevant fields for one patch operation.
type PatchOrderEntry struct {
	OpCode       PatchOpCode
	NodeID       uint64
	ParentNodeID uint64
	SourceNodeID uint64
}

// ValidatePatchOrder validates structural patch ordering rules for one patch stream.
func ValidatePatchOrder(parseEntries []PatchOrderEntry, parseKnownNodeIDs map[uint64]struct{}) error {
	parseInsertedNodeIDs := make(map[uint64]struct{}, len(parseEntries))
	parseRemovedNodeIDs := make(map[uint64]struct{}, len(parseEntries))
	for parseIndex, parseEntry := range parseEntries {
		switch parseEntry.OpCode {
		case PatchOpCodeInsertNode:
			if parseEntry.ParentNodeID == 0 {
				return fmt.Errorf("runtime2: patch order entry %d insert parent node id is required", parseIndex)
			}
			if _, hasKnownParentNodeID := parseKnownNodeIDs[parseEntry.ParentNodeID]; !hasKnownParentNodeID {
				if _, hasInsertedParentNodeID := parseInsertedNodeIDs[parseEntry.ParentNodeID]; !hasInsertedParentNodeID {
					return fmt.Errorf("runtime2: patch order entry %d inserted child node id %d before parent node id %d", parseIndex, parseEntry.NodeID, parseEntry.ParentNodeID)
				}
			}
			if parseEntry.NodeID == 0 {
				return fmt.Errorf("runtime2: patch order entry %d insert node id is required", parseIndex)
			}
			parseInsertedNodeIDs[parseEntry.NodeID] = struct{}{}
		case PatchOpCodeRemoveNode:
			if parseEntry.NodeID == 0 {
				return fmt.Errorf("runtime2: patch order entry %d remove node id is required", parseIndex)
			}
			parseRemovedNodeIDs[parseEntry.NodeID] = struct{}{}
		case PatchOpCodeMoveKeyedChild:
			if parseEntry.SourceNodeID == 0 {
				return fmt.Errorf("runtime2: patch order entry %d move source node id is required", parseIndex)
			}
			if _, hasRemovedSourceNodeID := parseRemovedNodeIDs[parseEntry.SourceNodeID]; hasRemovedSourceNodeID {
				return fmt.Errorf("runtime2: patch order entry %d moves removed node id %d", parseIndex, parseEntry.SourceNodeID)
			}
		}
	}
	return nil
}
