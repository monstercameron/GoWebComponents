package runtime2

import "fmt"

// RenderNodeTable stores the validated render-node records for one region.
type RenderNodeTable struct {
	Records []RenderNodeRecord
}

// RenderNodeKeyMetadata reports keyed metadata for one render-node record.
type RenderNodeKeyMetadata struct {
	HasKey  bool
	KeyHash uint64
	KeyText string
}

type parseRenderNodeSiblingKey struct {
	getKeyHash uint64
	getKeyText string
}

// ParseRenderNodeTable decodes and validates a region-local render-node table.
func ParseRenderNodeTable(parseRawRecords []RenderNodeRecordRaw) (RenderNodeTable, error) {
	parseRecords := make([]RenderNodeRecord, 0, len(parseRawRecords))
	parseSeenNodeIDs := make(map[uint64]struct{}, len(parseRawRecords))
	for parseIndex, parseRawRecord := range parseRawRecords {
		parseRecord, parseErr := ParseRenderNodeRecord(parseRawRecord)
		if parseErr != nil {
			return RenderNodeTable{}, fmt.Errorf("runtime2: render node record %d is invalid: %w", parseIndex, parseErr)
		}
		if parseRecord.NodeID == 0 {
			return RenderNodeTable{}, fmt.Errorf("runtime2: render node record %d has invalid node id 0", parseIndex)
		}
		if _, hasNodeID := parseSeenNodeIDs[parseRecord.NodeID]; hasNodeID {
			return RenderNodeTable{}, fmt.Errorf("runtime2: render node record %d duplicates node id %d", parseIndex, parseRecord.NodeID)
		}
		parseSeenNodeIDs[parseRecord.NodeID] = struct{}{}
		parseRecords = append(parseRecords, parseRecord)
	}
	if parseErr := parseRenderNodeChildSpans(parseRecords); parseErr != nil {
		return RenderNodeTable{}, parseErr
	}
	if parseErr := parseRenderNodeSiblingKeys(parseRecords); parseErr != nil {
		return RenderNodeTable{}, parseErr
	}
	return RenderNodeTable{Records: parseRecords}, nil
}

// GetRenderNodeChildOrder resolves one node's child node IDs in table order.
func (parseTable RenderNodeTable) GetRenderNodeChildOrder(parseNodeID uint64) ([]uint64, error) {
	for _, parseRecord := range parseTable.Records {
		if parseRecord.NodeID != parseNodeID {
			continue
		}
		if parseRecord.ChildCount == 0 {
			return []uint64{}, nil
		}
		parseChildStart := int(parseRecord.ChildStart)
		parseChildEnd := parseChildStart + int(parseRecord.ChildCount)
		if parseChildStart < 0 || parseChildEnd > len(parseTable.Records) {
			return nil, fmt.Errorf("runtime2: child span for node id %d is out of range", parseNodeID)
		}
		parseChildOrder := make([]uint64, 0, parseRecord.ChildCount)
		for parseChildIndex := parseChildStart; parseChildIndex < parseChildEnd; parseChildIndex++ {
			parseChildOrder = append(parseChildOrder, parseTable.Records[parseChildIndex].NodeID)
		}
		return parseChildOrder, nil
	}
	return nil, fmt.Errorf("runtime2: node id %d not found", parseNodeID)
}

// GetRenderNodeKeyMetadata resolves one node's keyed metadata.
func (parseTable RenderNodeTable) GetRenderNodeKeyMetadata(parseNodeID uint64) (RenderNodeKeyMetadata, error) {
	for _, parseRecord := range parseTable.Records {
		if parseRecord.NodeID != parseNodeID {
			continue
		}
		if parseRecord.KeyHash == 0 {
			return RenderNodeKeyMetadata{}, nil
		}
		return RenderNodeKeyMetadata{
			HasKey:  true,
			KeyHash: parseRecord.KeyHash,
			KeyText: parseRecord.KeyText,
		}, nil
	}
	return RenderNodeKeyMetadata{}, fmt.Errorf("runtime2: node id %d not found", parseNodeID)
}

// parseRenderNodeChildSpans validates child-span references and overlap rules for one table.
func parseRenderNodeChildSpans(parseRecords []RenderNodeRecord) error {
	parseOwnerByChildIndex := make([]uint64, len(parseRecords))
	for parseRecordIndex, parseRecord := range parseRecords {
		if parseRecord.ChildCount == 0 {
			continue
		}
		parseChildStart := int(parseRecord.ChildStart)
		parseChildEnd := parseChildStart + int(parseRecord.ChildCount)
		if parseChildStart < 0 || parseChildEnd > len(parseRecords) {
			return fmt.Errorf("runtime2: node id %d child span [%d:%d) references missing child", parseRecord.NodeID, parseChildStart, parseChildEnd)
		}
		for parseChildIndex := parseChildStart; parseChildIndex < parseChildEnd; parseChildIndex++ {
			if parseChildIndex == parseRecordIndex {
				return fmt.Errorf("runtime2: node id %d cannot reference itself as a child", parseRecord.NodeID)
			}
			if getOwnerNodeID := parseOwnerByChildIndex[parseChildIndex]; getOwnerNodeID != 0 {
				return fmt.Errorf("runtime2: node id %d child span overlaps node id %d at child index %d", parseRecord.NodeID, getOwnerNodeID, parseChildIndex)
			}
			parseOwnerByChildIndex[parseChildIndex] = parseRecord.NodeID
		}
	}
	return nil
}

// parseRenderNodeSiblingKeys validates keyed-child uniqueness within each sibling set.
func parseRenderNodeSiblingKeys(parseRecords []RenderNodeRecord) error {
	for _, parseRecord := range parseRecords {
		if parseRecord.ChildCount == 0 {
			continue
		}
		parseSiblingKeys := make(map[parseRenderNodeSiblingKey]uint64, parseRecord.ChildCount)
		parseChildStart := int(parseRecord.ChildStart)
		parseChildEnd := parseChildStart + int(parseRecord.ChildCount)
		for parseChildIndex := parseChildStart; parseChildIndex < parseChildEnd; parseChildIndex++ {
			parseChildRecord := parseRecords[parseChildIndex]
			if parseChildRecord.KeyHash == 0 {
				continue
			}
			parseSiblingKey := parseRenderNodeSiblingKey{
				getKeyHash: parseChildRecord.KeyHash,
				getKeyText: parseChildRecord.KeyText,
			}
			if getNodeID, hasNodeID := parseSiblingKeys[parseSiblingKey]; hasNodeID {
				return fmt.Errorf(
					"runtime2: duplicate keyed child %d:%q under parent node id %d for node ids %d and %d",
					parseChildRecord.KeyHash,
					parseChildRecord.KeyText,
					parseRecord.NodeID,
					getNodeID,
					parseChildRecord.NodeID,
				)
			}
			parseSiblingKeys[parseSiblingKey] = parseChildRecord.NodeID
		}
	}
	return nil
}
