package runtime2

import (
	"fmt"
)

const getRenderNodeSiblingKeyPairwiseLimit = 64
const getRenderNodeSiblingKeyProbeTableSize = 128

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

type parseRenderNodeSiblingHashEntry struct {
	getNodeID  uint64
	getKeyText string
}

// ParseRenderNodeTable decodes and validates a region-local render-node table.
func ParseRenderNodeTable(parseRawRecords []RenderNodeRecordRaw) (RenderNodeTable, error) {
	parseRecords := make([]RenderNodeRecord, len(parseRawRecords))
	var parseSeenNodeIDs map[uint64]struct{}
	parseExpectedNodeID := uint64(1)
	for parseIndex, parseRawRecord := range parseRawRecords {
		parseRecord, parseErr := ParseRenderNodeRecord(parseRawRecord)
		if parseErr != nil {
			return RenderNodeTable{}, fmt.Errorf("runtime2: render node record %d is invalid: %w", parseIndex, parseErr)
		}
		if parseRecord.NodeID == 0 {
			return RenderNodeTable{}, fmt.Errorf("runtime2: render node record %d has invalid node id 0", parseIndex)
		}
		if parseSeenNodeIDs == nil {
			if parseRecord.NodeID == parseExpectedNodeID {
				parseExpectedNodeID++
			} else {
				parseSeenNodeIDs = make(map[uint64]struct{}, len(parseRawRecords))
				for parseSeedIndex := 0; parseSeedIndex < parseIndex; parseSeedIndex++ {
					parseSeenNodeIDs[parseRecords[parseSeedIndex].NodeID] = struct{}{}
				}
				if _, hasNodeID := parseSeenNodeIDs[parseRecord.NodeID]; hasNodeID {
					return RenderNodeTable{}, fmt.Errorf("runtime2: render node record %d duplicates node id %d", parseIndex, parseRecord.NodeID)
				}
				parseSeenNodeIDs[parseRecord.NodeID] = struct{}{}
			}
		} else {
			if _, hasNodeID := parseSeenNodeIDs[parseRecord.NodeID]; hasNodeID {
				return RenderNodeTable{}, fmt.Errorf("runtime2: render node record %d duplicates node id %d", parseIndex, parseRecord.NodeID)
			}
			parseSeenNodeIDs[parseRecord.NodeID] = struct{}{}
		}
		parseRecords[parseIndex] = parseRecord
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
		parseChildStart := int(parseRecord.ChildStart)
		parseChildEnd := parseChildStart + int(parseRecord.ChildCount)
		parseKeyedSiblingCount := 0
		parseScanIndex := parseChildStart
		for ; parseScanIndex < parseChildEnd; parseScanIndex++ {
			if parseRecords[parseScanIndex].KeyHash != 0 {
				parseKeyedSiblingCount++
				if parseKeyedSiblingCount > 1 {
					parseScanIndex++
					break
				}
			}
		}
		if parseKeyedSiblingCount <= 1 {
			continue
		}
		for ; parseScanIndex < parseChildEnd; parseScanIndex++ {
			if parseRecords[parseScanIndex].KeyHash == 0 {
				continue
			}
			parseKeyedSiblingCount++
			if parseKeyedSiblingCount > getRenderNodeSiblingKeyPairwiseLimit {
				break
			}
		}
		if parseKeyedSiblingCount <= getRenderNodeSiblingKeyPairwiseLimit {
			var parseProbeHashes [getRenderNodeSiblingKeyProbeTableSize]uint64
			var parseProbeRecordIndexes [getRenderNodeSiblingKeyProbeTableSize]int
			for parseChildIndex := parseChildStart; parseChildIndex < parseChildEnd; parseChildIndex++ {
				parseChildRecord := parseRecords[parseChildIndex]
				if parseChildRecord.KeyHash == 0 {
					continue
				}
				parseProbeIndex := int(parseChildRecord.KeyHash & (getRenderNodeSiblingKeyProbeTableSize - 1))
				for {
					getProbeRecordIndex := parseProbeRecordIndexes[parseProbeIndex]
					if getProbeRecordIndex == 0 {
						parseProbeHashes[parseProbeIndex] = parseChildRecord.KeyHash
						parseProbeRecordIndexes[parseProbeIndex] = parseChildIndex + 1
						break
					}
					if parseProbeHashes[parseProbeIndex] == parseChildRecord.KeyHash {
						getProbeRecord := parseRecords[getProbeRecordIndex-1]
						if getProbeRecord.KeyText == parseChildRecord.KeyText {
							return fmt.Errorf(
								"runtime2: duplicate keyed child %d:%q under parent node id %d for node ids %d and %d",
								parseChildRecord.KeyHash,
								parseChildRecord.KeyText,
								parseRecord.NodeID,
								getProbeRecord.NodeID,
								parseChildRecord.NodeID,
							)
						}
					}
					parseProbeIndex++
					if parseProbeIndex == getRenderNodeSiblingKeyProbeTableSize {
						parseProbeIndex = 0
					}
				}
			}
			continue
		}
		if parseErr := parseRenderNodeSiblingKeysWithMap(
			parseRecords,
			parseRecord,
			parseChildStart,
			parseChildEnd,
		); parseErr != nil {
			return parseErr
		}
	}
	return nil
}

// parseRenderNodeSiblingKeysWithMap validates one large keyed-sibling set using hash-first tracking with collision promotion.
func parseRenderNodeSiblingKeysWithMap(
	parseRecords []RenderNodeRecord,
	parseRecord RenderNodeRecord,
	parseChildStart int,
	parseChildEnd int,
) error {
	parseSiblingHashEntries := make(map[uint64]parseRenderNodeSiblingHashEntry, int(parseRecord.ChildCount))
	var parseSiblingKeys map[parseRenderNodeSiblingKey]uint64
	for parseChildIndex := parseChildStart; parseChildIndex < parseChildEnd; parseChildIndex++ {
		parseChildRecord := parseRecords[parseChildIndex]
		if parseChildRecord.KeyHash == 0 {
			continue
		}
		if parseSiblingKeys != nil {
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
			continue
		}
		getHashEntry, hasHashEntry := parseSiblingHashEntries[parseChildRecord.KeyHash]
		if !hasHashEntry {
			parseSiblingHashEntries[parseChildRecord.KeyHash] = parseRenderNodeSiblingHashEntry{
				getNodeID:  parseChildRecord.NodeID,
				getKeyText: parseChildRecord.KeyText,
			}
			continue
		}
		if getHashEntry.getKeyText == parseChildRecord.KeyText {
			return fmt.Errorf(
				"runtime2: duplicate keyed child %d:%q under parent node id %d for node ids %d and %d",
				parseChildRecord.KeyHash,
				parseChildRecord.KeyText,
				parseRecord.NodeID,
				getHashEntry.getNodeID,
				parseChildRecord.NodeID,
			)
		}
		parseSiblingKeys = make(map[parseRenderNodeSiblingKey]uint64, len(parseSiblingHashEntries)+1)
		for parseHashKey, parseHashEntry := range parseSiblingHashEntries {
			parseSiblingKeys[parseRenderNodeSiblingKey{
				getKeyHash: parseHashKey,
				getKeyText: parseHashEntry.getKeyText,
			}] = parseHashEntry.getNodeID
		}
		parseSiblingKey := parseRenderNodeSiblingKey{
			getKeyHash: parseChildRecord.KeyHash,
			getKeyText: parseChildRecord.KeyText,
		}
		parseSiblingKeys[parseSiblingKey] = parseChildRecord.NodeID
	}
	return nil
}
