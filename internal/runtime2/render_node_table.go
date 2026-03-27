package runtime2

import (
	"fmt"
)

const getRenderNodeSiblingKeyDenseLimit = 8
const getRenderNodeSiblingKeyDenseBucketCount = 64
const getRenderNodeSiblingKeyPairwiseLimit = 64
const getRenderNodeSiblingKeyProbeTableSize = 128
const getRenderNodeDenseIDRangeFactor = 2
const getRenderNodeDenseIDRangeMax = 1 << 20

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
	parseRecords, parseMinNodeID, parseMaxNodeID, isNodeIDSequenceDense, parseDecodeErr := parseDecodeRenderNodeTableRecords(parseRawRecords)
	if parseDecodeErr != nil {
		return RenderNodeTable{}, parseDecodeErr
	}
	if parseErr := parseValidateRenderNodeUniqueIDs(parseRecords, parseMinNodeID, parseMaxNodeID, isNodeIDSequenceDense); parseErr != nil {
		return RenderNodeTable{}, parseErr
	}
	if parseErr := parseRenderNodeChildSpans(parseRecords); parseErr != nil {
		return RenderNodeTable{}, parseErr
	}
	if parseErr := parseRenderNodeSiblingKeys(parseRecords); parseErr != nil {
		return RenderNodeTable{}, parseErr
	}
	return RenderNodeTable{Records: parseRecords}, nil
}

// parseDecodeRenderNodeTableRecords decodes raw render-node records and collects unique-ID summary state for later validation.
func parseDecodeRenderNodeTableRecords(parseRawRecords []RenderNodeRecordRaw) ([]RenderNodeRecord, uint64, uint64, bool, error) {
	parseRecords := make([]RenderNodeRecord, len(parseRawRecords))
	parseMinNodeID := uint64(0)
	parseMaxNodeID := uint64(0)
	parseExpectedNodeID := uint64(1)
	isNodeIDSequenceDense := true
	for parseIndex, parseRawRecord := range parseRawRecords {
		parseRecord, parseErr := ParseRenderNodeRecord(parseRawRecord)
		if parseErr != nil {
			return nil, 0, 0, false, fmt.Errorf("runtime2: render node record %d is invalid: %w", parseIndex, parseErr)
		}
		if parseRecord.NodeID == 0 {
			return nil, 0, 0, false, fmt.Errorf("runtime2: render node record %d has invalid node id 0", parseIndex)
		}
		parseRecords[parseIndex] = parseRecord
		if parseRecord.NodeID != parseExpectedNodeID {
			isNodeIDSequenceDense = false
		}
		parseExpectedNodeID++
		if parseIndex == 0 || parseRecord.NodeID < parseMinNodeID {
			parseMinNodeID = parseRecord.NodeID
		}
		if parseIndex == 0 || parseRecord.NodeID > parseMaxNodeID {
			parseMaxNodeID = parseRecord.NodeID
		}
	}
	return parseRecords, parseMinNodeID, parseMaxNodeID, isNodeIDSequenceDense, nil
}

// parseValidateRenderNodeUniqueIDs validates node ID uniqueness using fast sequential or dense-range tracking before sparse-map fallback.
func parseValidateRenderNodeUniqueIDs(parseRecords []RenderNodeRecord, parseMinNodeID uint64, parseMaxNodeID uint64, isNodeIDSequenceDense bool) error {
	if len(parseRecords) <= 1 {
		return nil
	}
	if isNodeIDSequenceDense {
		return nil
	}
	parseNodeIDRange := parseMaxNodeID - parseMinNodeID + 1
	if parseNodeIDRange <= uint64(len(parseRecords))*getRenderNodeDenseIDRangeFactor &&
		parseNodeIDRange <= uint64(getRenderNodeDenseIDRangeMax) {
		parseSeenNodeIDs := make([]uint8, int(parseNodeIDRange))
		for parseIndex, parseRecord := range parseRecords {
			parseNodeIDOffset := int(parseRecord.NodeID - parseMinNodeID)
			if parseSeenNodeIDs[parseNodeIDOffset] != 0 {
				return fmt.Errorf("runtime2: render node record %d duplicates node id %d", parseIndex, parseRecord.NodeID)
			}
			parseSeenNodeIDs[parseNodeIDOffset] = 1
		}
		return nil
	}
	parseSeenNodeIDs := make(map[uint64]struct{}, len(parseRecords))
	for parseIndex, parseRecord := range parseRecords {
		if _, hasNodeID := parseSeenNodeIDs[parseRecord.NodeID]; hasNodeID {
			return fmt.Errorf("runtime2: render node record %d duplicates node id %d", parseIndex, parseRecord.NodeID)
		}
		parseSeenNodeIDs[parseRecord.NodeID] = struct{}{}
	}
	return nil
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
		if parseKeyedSiblingCount <= getRenderNodeSiblingKeyDenseLimit {
			if parseErr := parseRenderNodeSiblingKeysWithDenseBuckets(
				parseRecords,
				parseRecord,
				parseChildStart,
				parseChildEnd,
			); parseErr != nil {
				return parseErr
			}
			continue
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

// parseRenderNodeSiblingKeysWithDenseBuckets validates one small keyed-sibling set with a compact bucket bitmap and inline collision scan.
func parseRenderNodeSiblingKeysWithDenseBuckets(
	parseRecords []RenderNodeRecord,
	parseRecord RenderNodeRecord,
	parseChildStart int,
	parseChildEnd int,
) error {
	parseBucketMask := uint64(0)
	parseDenseCount := 0
	var parseDenseBuckets [getRenderNodeSiblingKeyDenseLimit]uint8
	var parseDenseIndexes [getRenderNodeSiblingKeyDenseLimit]int
	for parseChildIndex := parseChildStart; parseChildIndex < parseChildEnd; parseChildIndex++ {
		parseChildRecord := parseRecords[parseChildIndex]
		if parseChildRecord.KeyHash == 0 {
			continue
		}
		parseBucketIndex := uint8(parseChildRecord.KeyHash & (getRenderNodeSiblingKeyDenseBucketCount - 1))
		parseBucketBit := uint64(1) << parseBucketIndex
		if parseBucketMask&parseBucketBit != 0 {
			for parseDenseIndex := 0; parseDenseIndex < parseDenseCount; parseDenseIndex++ {
				if parseDenseBuckets[parseDenseIndex] != parseBucketIndex {
					continue
				}
				getDenseRecord := parseRecords[parseDenseIndexes[parseDenseIndex]]
				if getDenseRecord.KeyHash == parseChildRecord.KeyHash && getDenseRecord.KeyText == parseChildRecord.KeyText {
					return fmt.Errorf(
						"runtime2: duplicate keyed child %d:%q under parent node id %d for node ids %d and %d",
						parseChildRecord.KeyHash,
						parseChildRecord.KeyText,
						parseRecord.NodeID,
						getDenseRecord.NodeID,
						parseChildRecord.NodeID,
					)
				}
			}
		}
		parseBucketMask |= parseBucketBit
		parseDenseBuckets[parseDenseCount] = parseBucketIndex
		parseDenseIndexes[parseDenseCount] = parseChildIndex
		parseDenseCount++
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
