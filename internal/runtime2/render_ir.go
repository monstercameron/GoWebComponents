package runtime2

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
)

// CanonicalRenderIR stores one fully canonicalized render tree for worker diffing.
type CanonicalRenderIR struct {
	GetRootNodeID  uint64
	GetStringTable RenderStringTable
	GetNodeRecords []RenderNodeRecordRaw
	GetPropRecords []RenderPropRecordRaw
}

type canonicalRenderNode struct {
	getNodeID   uint64
	getKind     RenderNodeKind
	getTag      string
	getText     string
	getKey      string
	hasKey      bool
	getProps    []RenderPropRecord
	getChildren []*canonicalRenderNode
}

type canonicalRenderTree struct {
	getRootNodeID uint64
	getNodeByID   map[uint64]canonicalRenderNodeState
}

type canonicalRenderNodeState struct {
	getNodeID       uint64
	getKind         RenderNodeKind
	getDepth        int
	getTag          string
	getText         string
	getKey          string
	hasKey          bool
	hasKeyedChild   bool
	getParentNodeID uint64
	getChildNodeIDs []uint64
	getPropByKey    map[string]RenderPropRecord
}

var storeCanonicalReservedFieldSet = map[string]struct{}{
	"children": {},
	"key":      {},
	"kind":     {},
	"props":    {},
	"tag":      {},
	"text":     {},
}

var storeCanonicalRenderIRInvariantValidationEnabled atomic.Bool

// SetCanonicalRenderIRInvariantValidationEnabled enables or disables strict canonical IR self-parse invariant checks.
func SetCanonicalRenderIRInvariantValidationEnabled(parseIsEnabled bool) {
	storeCanonicalRenderIRInvariantValidationEnabled.Store(parseIsEnabled)
}

// HasCanonicalRenderIRInvariantValidationEnabled reports whether strict canonical IR self-parse invariant checks are enabled.
func HasCanonicalRenderIRInvariantValidationEnabled() bool {
	return storeCanonicalRenderIRInvariantValidationEnabled.Load()
}

// BuildCanonicalRenderIR converts one validated display-only render output into canonical render IR tables.
func BuildCanonicalRenderIR(parseRenderOutput any) (CanonicalRenderIR, error) {
	buildRootNode, parseRootErr := parseBuildCanonicalRootNode(parseRenderOutput)
	if parseRootErr != nil {
		return CanonicalRenderIR{}, parseRootErr
	}
	if parseSiblingKeyErr := parseValidateCanonicalSiblingKeys(buildRootNode); parseSiblingKeyErr != nil {
		return CanonicalRenderIR{}, parseSiblingKeyErr
	}
	parseAssignCanonicalNodeIDs(buildRootNode)
	buildNodeOrder := parseBuildCanonicalNodeOrder(buildRootNode)
	buildStringValues, buildPropRecordCapacity := buildCanonicalStringValuesAndPropCapacityFromNodeOrder(buildNodeOrder)
	buildStringTable := BuildRenderStringTable(buildStringValues)
	buildPropRecords := make([]RenderPropRecordRaw, 0, buildPropRecordCapacity)
	buildNodeRecords := make([]RenderNodeRecordRaw, 0, len(buildNodeOrder))
	// In breadth-first order, each parent's direct children are appended contiguously.
	// Track the next child-start index incrementally so node-record emission avoids map lookups.
	buildNextChildStart := 1
	for _, getNode := range buildNodeOrder {
		buildChildStart := 0
		getChildCount := len(getNode.getChildren)
		if getChildCount > 0 {
			buildChildStart = buildNextChildStart
			buildNextChildStart += getChildCount
		}
		buildPropStart := len(buildPropRecords)
		for _, getPropRecord := range getNode.getProps {
			getKeyRef, hasKeyRef := buildStringTable.GetRenderStringRef(getPropRecord.Key)
			if !hasKeyRef {
				return CanonicalRenderIR{}, fmt.Errorf("runtime2: canonical prop key %q missing from string table", getPropRecord.Key)
			}
			getValueRef, hasValueRef := buildStringTable.GetRenderStringRef(getPropRecord.Value)
			if !hasValueRef {
				return CanonicalRenderIR{}, fmt.Errorf("runtime2: canonical prop value %q missing from string table", getPropRecord.Value)
			}
			buildPropRecords = append(buildPropRecords, RenderPropRecordRaw{
				Kind:     uint8(getPropRecord.Kind),
				KeyRef:   getKeyRef,
				ValueRef: getValueRef,
			})
		}
		buildTextRef := uint32(0)
		switch getNode.getKind {
		case RenderNodeKindText:
			getTextRef, hasTextRef := buildStringTable.GetRenderStringRef(getNode.getText)
			if !hasTextRef {
				return CanonicalRenderIR{}, fmt.Errorf("runtime2: canonical text payload %q missing from string table", getNode.getText)
			}
			buildTextRef = getTextRef
		case RenderNodeKindHostElement:
			getTagRef, hasTagRef := buildStringTable.GetRenderStringRef(getNode.getTag)
			if !hasTagRef {
				return CanonicalRenderIR{}, fmt.Errorf("runtime2: canonical host tag %q missing from string table", getNode.getTag)
			}
			buildTextRef = getTagRef
		}
		buildNodeRecord := RenderNodeRecordRaw{
			NodeID:     getNode.getNodeID,
			Kind:       uint8(getNode.getKind),
			ChildStart: uint32(buildChildStart),
			ChildCount: uint32(getChildCount),
			PropStart:  uint32(buildPropStart),
			PropCount:  uint32(len(getNode.getProps)),
			TextRef:    buildTextRef,
		}
		if getNode.hasKey {
			buildNodeRecord.KeyText = getNode.getKey
			buildNodeRecord.KeyHash = parseHashCanonicalString("key:"+getNode.getKey, nil)
		}
		buildNodeRecords = append(buildNodeRecords, buildNodeRecord)
	}
	if HasCanonicalRenderIRInvariantValidationEnabled() {
		if _, parseNodeTableErr := ParseRenderNodeTable(buildNodeRecords); parseNodeTableErr != nil {
			return CanonicalRenderIR{}, fmt.Errorf("runtime2: canonical node table is invalid: %w", parseNodeTableErr)
		}
		for _, getNodeRecord := range buildNodeRecords {
			if getNodeRecord.PropCount == 0 {
				continue
			}
			buildPropStart := int(getNodeRecord.PropStart)
			buildPropEnd := buildPropStart + int(getNodeRecord.PropCount)
			if buildPropStart < 0 || buildPropEnd > len(buildPropRecords) {
				return CanonicalRenderIR{}, fmt.Errorf("runtime2: canonical prop span [%d:%d) is out of range", buildPropStart, buildPropEnd)
			}
			if _, parsePropsErr := ParseRenderPropRecords(buildPropRecords[buildPropStart:buildPropEnd], buildStringTable); parsePropsErr != nil {
				return CanonicalRenderIR{}, fmt.Errorf("runtime2: canonical props are invalid: %w", parsePropsErr)
			}
		}
	}
	return CanonicalRenderIR{
		GetRootNodeID:  buildRootNode.getNodeID,
		GetStringTable: buildStringTable,
		GetNodeRecords: buildNodeRecords,
		GetPropRecords: buildPropRecords,
	}, nil
}

// parseValidateCanonicalSiblingKeys rejects duplicate keyed children before node-table encoding so canonical IR build preserves keyed-sibling invariants even when self-parse validation is disabled.
func parseValidateCanonicalSiblingKeys(parseParentNode *canonicalRenderNode) error {
	if parseParentNode == nil {
		return nil
	}
	if parseSiblingKeyErr := parseValidateCanonicalChildSiblingKeys(parseParentNode); parseSiblingKeyErr != nil {
		return parseSiblingKeyErr
	}
	for _, getChildNode := range parseParentNode.getChildren {
		if parseChildErr := parseValidateCanonicalSiblingKeys(getChildNode); parseChildErr != nil {
			return parseChildErr
		}
	}
	return nil
}

// parseValidateCanonicalChildSiblingKeys checks one parent node's direct children for duplicate keyed siblings.
func parseValidateCanonicalChildSiblingKeys(parseParentNode *canonicalRenderNode) error {
	if parseParentNode == nil || len(parseParentNode.getChildren) <= 1 {
		return nil
	}
	parseSeenChildIndexByKey := make(map[string]int, len(parseParentNode.getChildren))
	for parseChildIndex, getChildNode := range parseParentNode.getChildren {
		if getChildNode == nil || !getChildNode.hasKey {
			continue
		}
		if getFirstChildIndex, hasChildIndex := parseSeenChildIndexByKey[getChildNode.getKey]; hasChildIndex {
			return fmt.Errorf(
				"runtime2: duplicate keyed child %q under sibling indexes %d and %d",
				getChildNode.getKey,
				getFirstChildIndex,
				parseChildIndex,
			)
		}
		parseSeenChildIndexByKey[getChildNode.getKey] = parseChildIndex
	}
	return nil
}

// BuildRenderStringTableFromRenderOutput extracts one canonical string table from one render output tree.
func BuildRenderStringTableFromRenderOutput(parseRenderOutput any) (RenderStringTable, error) {
	buildRootNode, parseRootErr := parseBuildCanonicalRootNode(parseRenderOutput)
	if parseRootErr != nil {
		return RenderStringTable{}, parseRootErr
	}
	buildNodeOrder := parseBuildCanonicalNodeOrder(buildRootNode)
	buildStringValues, _ := buildCanonicalStringValuesAndPropCapacityFromNodeOrder(buildNodeOrder)
	return BuildRenderStringTable(buildStringValues), nil
}

// BuildRenderPropRecordsFromRenderOutput extracts canonical prop records from one render output tree.
func BuildRenderPropRecordsFromRenderOutput(parseRenderOutput any) ([]RenderPropRecordRaw, RenderStringTable, error) {
	buildCanonicalIR, parseCanonicalErr := BuildCanonicalRenderIR(parseRenderOutput)
	if parseCanonicalErr != nil {
		return nil, RenderStringTable{}, parseCanonicalErr
	}
	return append([]RenderPropRecordRaw(nil), buildCanonicalIR.GetPropRecords...), buildCanonicalIR.GetStringTable, nil
}

// IsCanonicalRenderIREqual reports whether two canonical render trees are identical.
func IsCanonicalRenderIREqual(parseLeft CanonicalRenderIR, parseRight CanonicalRenderIR) bool {
	if parseLeft.GetRootNodeID != parseRight.GetRootNodeID {
		return false
	}
	if len(parseLeft.GetStringTable.Entries) != len(parseRight.GetStringTable.Entries) {
		return false
	}
	for parseIndex := range parseLeft.GetStringTable.Entries {
		if parseLeft.GetStringTable.Entries[parseIndex] != parseRight.GetStringTable.Entries[parseIndex] {
			return false
		}
	}
	if len(parseLeft.GetNodeRecords) != len(parseRight.GetNodeRecords) {
		return false
	}
	for parseIndex := range parseLeft.GetNodeRecords {
		if parseLeft.GetNodeRecords[parseIndex] != parseRight.GetNodeRecords[parseIndex] {
			return false
		}
	}
	if len(parseLeft.GetPropRecords) != len(parseRight.GetPropRecords) {
		return false
	}
	for parseIndex := range parseLeft.GetPropRecords {
		if parseLeft.GetPropRecords[parseIndex] != parseRight.GetPropRecords[parseIndex] {
			return false
		}
	}
	return true
}

// ParseCanonicalRenderTree decodes one canonical render IR payload into node-level lookup state.
func ParseCanonicalRenderTree(parseIR CanonicalRenderIR) (canonicalRenderTree, error) {
	if len(parseIR.GetNodeRecords) == 0 {
		return canonicalRenderTree{}, fmt.Errorf("runtime2: canonical render IR node records are required")
	}
	parseNodeTable, parseNodeTableErr := ParseRenderNodeTable(parseIR.GetNodeRecords)
	if parseNodeTableErr != nil {
		return canonicalRenderTree{}, parseNodeTableErr
	}
	parseNodeRecords := parseNodeTable.Records
	buildTotalChildren := 0
	for parseRecordIndex := range parseNodeRecords {
		buildTotalChildren += int(parseNodeRecords[parseRecordIndex].ChildCount)
	}
	buildChildIDPool := make([]uint64, buildTotalChildren)
	buildChildPoolOffset := 0
	parseParentNodeIDByRecordIndex := make([]uint64, len(parseNodeRecords))
	parseDepthByRecordIndex := make([]uint32, len(parseNodeRecords))
	parseChildPoolStartByRecordIndex := make([]uint32, len(parseNodeRecords))
	parseHasKeyedChildByRecordIndex := make([]bool, len(parseNodeRecords))
	for parseRecordIndex := range parseNodeRecords {
		getNodeRecord := parseNodeRecords[parseRecordIndex]
		if getNodeRecord.ChildCount == 0 {
			continue
		}
		buildChildStart := int(getNodeRecord.ChildStart)
		buildChildEnd := buildChildStart + int(getNodeRecord.ChildCount)
		buildSlotStart := buildChildPoolOffset
		parseHasKeyedChild := false
		// Compute depth inline: each child is one level deeper than its parent.
		for parseChildIndex := buildChildStart; parseChildIndex < buildChildEnd; parseChildIndex++ {
			buildChildIDPool[buildChildPoolOffset] = parseNodeRecords[parseChildIndex].NodeID
			parseParentNodeIDByRecordIndex[parseChildIndex] = getNodeRecord.NodeID
			parseDepthByRecordIndex[parseChildIndex] = parseDepthByRecordIndex[parseRecordIndex] + 1
			if !parseHasKeyedChild && hasCanonicalNodeKeyText(parseNodeRecords[parseChildIndex].KeyText) {
				parseHasKeyedChild = true
			}
			buildChildPoolOffset++
		}
		parseChildPoolStartByRecordIndex[parseRecordIndex] = uint32(buildSlotStart)
		parseHasKeyedChildByRecordIndex[parseRecordIndex] = parseHasKeyedChild
	}
	parseNodeByID := make(map[uint64]canonicalRenderNodeState, len(parseNodeRecords))
	parseStringEntries := parseIR.GetStringTable.Entries
	parsePropRecords := parseIR.GetPropRecords
	for parseRecordIndex := range parseNodeRecords {
		getNodeRecord := parseNodeRecords[parseRecordIndex]
		buildNodeState := canonicalRenderNodeState{
			getNodeID:       getNodeRecord.NodeID,
			getKind:         getNodeRecord.Kind,
			getDepth:        int(parseDepthByRecordIndex[parseRecordIndex]),
			getKey:          getNodeRecord.KeyText,
			hasKey:          hasCanonicalNodeKeyText(getNodeRecord.KeyText),
			hasKeyedChild:   parseHasKeyedChildByRecordIndex[parseRecordIndex],
			getParentNodeID: parseParentNodeIDByRecordIndex[parseRecordIndex],
		}
		if getNodeRecord.ChildCount > 0 {
			buildChildPoolStart := int(parseChildPoolStartByRecordIndex[parseRecordIndex])
			buildChildPoolEnd := buildChildPoolStart + int(getNodeRecord.ChildCount)
			buildNodeState.getChildNodeIDs = buildChildIDPool[buildChildPoolStart:buildChildPoolEnd]
		}
		switch getNodeRecord.Kind {
		case RenderNodeKindText:
			if getNodeRecord.TextRef >= uint32(len(parseStringEntries)) {
				return canonicalRenderTree{}, fmt.Errorf("runtime2: text node id %d has invalid text reference %d", getNodeRecord.NodeID, getNodeRecord.TextRef)
			}
			buildNodeState.getText = parseStringEntries[getNodeRecord.TextRef]
		case RenderNodeKindHostElement:
			if getNodeRecord.TextRef >= uint32(len(parseStringEntries)) {
				return canonicalRenderTree{}, fmt.Errorf("runtime2: host node id %d has invalid tag reference %d", getNodeRecord.NodeID, getNodeRecord.TextRef)
			}
			buildNodeState.getTag = parseStringEntries[getNodeRecord.TextRef]
		}
		if getNodeRecord.PropCount > 0 {
			buildPropStart := int(getNodeRecord.PropStart)
			buildPropEnd := buildPropStart + int(getNodeRecord.PropCount)
			if buildPropStart < 0 || buildPropEnd > len(parsePropRecords) {
				return canonicalRenderTree{}, fmt.Errorf("runtime2: node id %d prop span [%d:%d) is out of range", getNodeRecord.NodeID, buildPropStart, buildPropEnd)
			}
			parsePropByKey, parsePropErr := parseBuildCanonicalPropByKeyFromRaw(parsePropRecords[buildPropStart:buildPropEnd], parseIR.GetStringTable)
			if parsePropErr != nil {
				return canonicalRenderTree{}, fmt.Errorf("runtime2: node id %d props are invalid: %w", getNodeRecord.NodeID, parsePropErr)
			}
			buildNodeState.getPropByKey = parsePropByKey
		}
		parseNodeByID[getNodeRecord.NodeID] = buildNodeState
	}
	parseRootNodeID := parseIR.GetRootNodeID
	if parseRootNodeID == 0 {
		parseRootNodeID = parseIR.GetNodeRecords[0].NodeID
	}
	if _, hasRootNode := parseNodeByID[parseRootNodeID]; !hasRootNode {
		return canonicalRenderTree{}, fmt.Errorf("runtime2: canonical root node id %d is missing", parseRootNodeID)
	}
	return canonicalRenderTree{
		getRootNodeID: parseRootNodeID,
		getNodeByID:   parseNodeByID,
	}, nil
}

// parseBuildCanonicalPropByKeyFromRaw decodes one raw prop span into key lookup state with duplicate-key validation.
func parseBuildCanonicalPropByKeyFromRaw(parseRawRecords []RenderPropRecordRaw, parseStringTable RenderStringTable) (map[string]RenderPropRecord, error) {
	if len(parseRawRecords) == 0 {
		return nil, nil
	}
	parseStringEntries := parseStringTable.Entries
	parseStringEntryLimit := uint32(len(parseStringEntries))
	buildPropByKey := make(map[string]RenderPropRecord, len(parseRawRecords))
	for parseIndex, parseRawRecord := range parseRawRecords {
		parseKind := RenderPropKind(parseRawRecord.Kind)
		switch parseKind {
		case RenderPropKindClass, RenderPropKindStyle, RenderPropKindAria, RenderPropKindData, RenderPropKindTextAdjacent:
		case renderPropKindInvalid:
			return nil, fmt.Errorf("runtime2: prop record %d is invalid: runtime2: render prop kind %d is invalid", parseIndex, parseRawRecord.Kind)
		default:
			return nil, fmt.Errorf("runtime2: prop record %d is invalid: runtime2: render prop kind %d is unsupported", parseIndex, parseRawRecord.Kind)
		}
		if parseRawRecord.KeyRef >= parseStringEntryLimit {
			return nil, fmt.Errorf("runtime2: key reference %d is out of range", parseRawRecord.KeyRef)
		}
		parseKey := parseStringEntries[parseRawRecord.KeyRef]
		if !hasCanonicalPropKeyText(parseKey) {
			return nil, fmt.Errorf("runtime2: key reference %d resolved to an empty key", parseRawRecord.KeyRef)
		}
		if parseRawRecord.ValueRef >= parseStringEntryLimit {
			return nil, fmt.Errorf("runtime2: value reference %d is out of range", parseRawRecord.ValueRef)
		}
		buildPropByKey[parseKey] = RenderPropRecord{
			Kind:  parseKind,
			Key:   parseKey,
			Value: parseStringEntries[parseRawRecord.ValueRef],
		}
	}
	if len(buildPropByKey) != len(parseRawRecords) {
		parseDuplicateKey := getDuplicateCanonicalPropKey(parseRawRecords, parseStringEntries)
		if parseDuplicateKey == "" {
			return nil, fmt.Errorf("runtime2: duplicate prop key is unsupported")
		}
		return nil, fmt.Errorf("runtime2: duplicate prop key %q", parseDuplicateKey)
	}
	return buildPropByKey, nil
}

// hasCanonicalPropKeyText reports whether one prop key is non-empty after trim with an ASCII-boundary fast path.
func hasCanonicalPropKeyText(parseKey string) bool {
	if parseKey == "" {
		return false
	}
	parseLastIndex := len(parseKey) - 1
	if parseKey[0] > ' ' && parseKey[parseLastIndex] > ' ' {
		return true
	}
	return parseRuntimeHasTrimmedNonWhitespaceText(parseKey)
}

// hasCanonicalNodeKeyText reports whether one node key is non-empty after trim with an ASCII-boundary fast path.
func hasCanonicalNodeKeyText(parseKey string) bool {
	if parseKey == "" {
		return false
	}
	parseLastIndex := len(parseKey) - 1
	if parseKey[0] > ' ' && parseKey[parseLastIndex] > ' ' {
		return true
	}
	return parseRuntimeHasTrimmedNonWhitespaceText(parseKey)
}

// getDuplicateCanonicalPropKey returns one duplicate prop key from one raw prop span, if any.
func getDuplicateCanonicalPropKey(parseRawRecords []RenderPropRecordRaw, parseStringEntries []string) string {
	if len(parseRawRecords) <= 1 {
		return ""
	}
	parseSeenByKey := make(map[string]struct{}, len(parseRawRecords))
	for _, parseRawRecord := range parseRawRecords {
		parseKey := parseStringEntries[parseRawRecord.KeyRef]
		if _, hasSeenKey := parseSeenByKey[parseKey]; hasSeenKey {
			return parseKey
		}
		parseSeenByKey[parseKey] = struct{}{}
	}
	return ""
}

// GetCanonicalNodeState resolves one decoded canonical node by ID.
func (parseTree canonicalRenderTree) GetCanonicalNodeState(parseNodeID uint64) (canonicalRenderNodeState, bool) {
	getNodeState, hasNodeState := parseTree.getNodeByID[parseNodeID]
	return getNodeState, hasNodeState
}

// GetCanonicalNodeDepth resolves one decoded canonical node depth by ID.
func (parseTree canonicalRenderTree) GetCanonicalNodeDepth(parseNodeID uint64) int {
	return parseTree.getNodeByID[parseNodeID].getDepth
}

// parseBuildCanonicalRootNode normalizes one render output into a canonical root convention.
func parseBuildCanonicalRootNode(parseRenderOutput any) (*canonicalRenderNode, error) {
	if parseRenderOutput == nil {
		return &canonicalRenderNode{
			getKind: RenderNodeKindFragment,
		}, nil
	}
	parseValue := reflect.ValueOf(parseRenderOutput)
	for parseValue.IsValid() && (parseValue.Kind() == reflect.Interface || parseValue.Kind() == reflect.Pointer) {
		if parseValue.IsNil() {
			return &canonicalRenderNode{
				getKind: RenderNodeKindFragment,
			}, nil
		}
		parseValue = parseValue.Elem()
	}
	if !parseValue.IsValid() {
		return &canonicalRenderNode{
			getKind: RenderNodeKindFragment,
		}, nil
	}
	if parseValue.Kind() == reflect.Slice || parseValue.Kind() == reflect.Array {
		buildLength := parseValue.Len()
		if buildLength == 0 {
			return &canonicalRenderNode{
				getKind: RenderNodeKindFragment,
			}, nil
		}
		if buildLength == 1 {
			buildNode, parseNodeErr := parseBuildCanonicalRenderNode(parseValue.Index(0).Interface())
			if parseNodeErr != nil {
				return nil, parseNodeErr
			}
			if buildNode.getKind == RenderNodeKindText || buildNode.getKind == RenderNodeKindHostElement {
				return buildNode, nil
			}
			return &canonicalRenderNode{
				getKind:     RenderNodeKindFragment,
				getChildren: []*canonicalRenderNode{buildNode},
			}, nil
		}
		buildChildren := make([]*canonicalRenderNode, 0, buildLength)
		for parseIndex := 0; parseIndex < buildLength; parseIndex++ {
			getChildNode, parseChildErr := parseBuildCanonicalRenderNode(parseValue.Index(parseIndex).Interface())
			if parseChildErr != nil {
				return nil, parseChildErr
			}
			buildChildren = append(buildChildren, getChildNode)
		}
		return &canonicalRenderNode{
			getKind:     RenderNodeKindFragment,
			getChildren: buildChildren,
		}, nil
	}
	return parseBuildCanonicalRenderNode(parseRenderOutput)
}

// parseBuildCanonicalRenderNode converts one arbitrary render payload into one canonical node.
func parseBuildCanonicalRenderNode(parseValue any) (*canonicalRenderNode, error) {
	switch getValue := parseValue.(type) {
	case nil:
		return &canonicalRenderNode{
			getKind: RenderNodeKindFragment,
		}, nil
	case string:
		return &canonicalRenderNode{
			getKind: RenderNodeKindText,
			getText: getValue,
		}, nil
	case bool, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return &canonicalRenderNode{
			getKind: RenderNodeKindText,
			getText: parseFormatCanonicalScalarValue(getValue),
		}, nil
	case []any:
		buildChildren := make([]*canonicalRenderNode, 0, len(getValue))
		for _, getChildValue := range getValue {
			getChildNode, parseChildErr := parseBuildCanonicalRenderNode(getChildValue)
			if parseChildErr != nil {
				return nil, parseChildErr
			}
			buildChildren = append(buildChildren, getChildNode)
		}
		return &canonicalRenderNode{
			getKind:     RenderNodeKindFragment,
			getChildren: buildChildren,
		}, nil
	case map[string]any:
		return parseBuildCanonicalRenderNodeFromMap(getValue)
	default:
		parseReflectValue := reflect.ValueOf(parseValue)
		if parseReflectValue.IsValid() && parseReflectValue.Kind() == reflect.Map {
			buildMapValue, parseMapValueErr := parseBuildCanonicalMapValue(parseReflectValue)
			if parseMapValueErr == nil {
				return parseBuildCanonicalRenderNodeFromMap(buildMapValue)
			}
		}
		buildTextPayload, parseTextErr := parseBuildCanonicalJSON(parseValue)
		if parseTextErr != nil {
			return nil, parseTextErr
		}
		return &canonicalRenderNode{
			getKind: RenderNodeKindText,
			getText: buildTextPayload,
		}, nil
	}
}

// parseBuildCanonicalRenderNodeFromMap decodes map-style render payloads into canonical node shapes.
func parseBuildCanonicalRenderNodeFromMap(parseMapValue map[string]any) (*canonicalRenderNode, error) {
	buildKind := parseGetCanonicalRenderNodeKind(parseMapValue["kind"])
	if buildKind == "" {
		switch {
		case parseMapValue["tag"] != nil:
			buildKind = "host-element"
		case parseMapValue["text"] != nil && parseMapValue["children"] == nil:
			buildKind = "text"
		case parseMapValue["children"] != nil:
			buildKind = "fragment"
		default:
			buildTextPayload, parseTextErr := parseBuildCanonicalJSON(parseMapValue)
			if parseTextErr != nil {
				return nil, parseTextErr
			}
			return &canonicalRenderNode{
				getKind: RenderNodeKindText,
				getText: buildTextPayload,
			}, nil
		}
	}
	switch buildKind {
	case "text":
		return &canonicalRenderNode{
			getKind: RenderNodeKindText,
			getText: parseGetCanonicalStringValue(parseMapValue["text"]),
		}, nil
	case "fragment":
		buildChildren, parseChildrenErr := parseBuildCanonicalChildren(parseMapValue["children"])
		if parseChildrenErr != nil {
			return nil, parseChildrenErr
		}
		buildKey := parseGetCanonicalStringValue(parseMapValue["key"])
		return &canonicalRenderNode{
			getKind:     RenderNodeKindFragment,
			getKey:      buildKey,
			hasKey:      hasCanonicalNodeKeyText(buildKey),
			getChildren: buildChildren,
		}, nil
	case "host-element", "element":
		buildTag := parseGetCanonicalTrimmedStringValue(parseMapValue["tag"])
		if parseTagErr := ValidateWorkerRenderableHostTag(buildTag); parseTagErr != nil {
			return nil, parseTagErr
		}
		buildProps, parsePropsErr := parseBuildCanonicalProps(parseMapValue)
		if parsePropsErr != nil {
			return nil, parsePropsErr
		}
		buildChildren, parseChildrenErr := parseBuildCanonicalChildren(parseMapValue["children"])
		if parseChildrenErr != nil {
			return nil, parseChildrenErr
		}
		buildKey := parseGetCanonicalStringValue(parseMapValue["key"])
		return &canonicalRenderNode{
			getKind:     RenderNodeKindHostElement,
			getTag:      buildTag,
			getKey:      buildKey,
			hasKey:      hasCanonicalNodeKeyText(buildKey),
			getProps:    buildProps,
			getChildren: buildChildren,
		}, nil
	default:
		return nil, fmt.Errorf("runtime2: unsupported render node kind %q", buildKind)
	}
}

// parseGetCanonicalRenderNodeKind normalizes one map payload node kind using a string fast path for already-canonical values.
func parseGetCanonicalRenderNodeKind(parseKindValue any) string {
	switch getKindValue := parseKindValue.(type) {
	case string:
		switch getKindValue {
		case "":
			return ""
		case "text", "fragment", "host-element", "element":
			return getKindValue
		}
		buildKind := parseGetCanonicalTrimmedStringValue(getKindValue)
		if buildKind == "" {
			return ""
		}
		for parseIndex := 0; parseIndex < len(buildKind); parseIndex++ {
			getChar := buildKind[parseIndex]
			if getChar >= 'A' && getChar <= 'Z' {
				return strings.ToLower(buildKind)
			}
		}
		return buildKind
	default:
		buildKind := parseGetCanonicalTrimmedStringValue(parseKindValue)
		if buildKind == "" {
			return ""
		}
		return strings.ToLower(buildKind)
	}
}

// parseGetCanonicalTrimmedStringValue normalizes one scalar value to string and trims surrounding ASCII/Unicode whitespace.
func parseGetCanonicalTrimmedStringValue(parseValue any) string {
	buildValue := parseGetCanonicalStringValue(parseValue)
	if buildValue == "" {
		return ""
	}
	parseLastIndex := len(buildValue) - 1
	if buildValue[0] > ' ' && buildValue[parseLastIndex] > ' ' {
		return buildValue
	}
	return strings.TrimSpace(buildValue)
}

// parseBuildCanonicalChildren normalizes one children payload into a canonical child-node list.
func parseBuildCanonicalChildren(parseChildrenValue any) ([]*canonicalRenderNode, error) {
	if parseChildrenValue == nil {
		return nil, nil
	}
	switch getChildrenValue := parseChildrenValue.(type) {
	case []any:
		buildChildren := make([]*canonicalRenderNode, 0, len(getChildrenValue))
		for parseIndex := 0; parseIndex < len(getChildrenValue); parseIndex++ {
			getChildValue := getChildrenValue[parseIndex]
			buildNode, parseNodeErr := parseBuildCanonicalRenderNode(getChildValue)
			if parseNodeErr != nil {
				return nil, parseNodeErr
			}
			buildChildren = append(buildChildren, buildNode)
		}
		return buildChildren, nil
	}
	parseReflectValue := reflect.ValueOf(parseChildrenValue)
	for parseReflectValue.IsValid() && (parseReflectValue.Kind() == reflect.Interface || parseReflectValue.Kind() == reflect.Pointer) {
		if parseReflectValue.IsNil() {
			return nil, nil
		}
		parseReflectValue = parseReflectValue.Elem()
	}
	if !parseReflectValue.IsValid() {
		return nil, nil
	}
	if parseReflectValue.Kind() != reflect.Slice && parseReflectValue.Kind() != reflect.Array {
		buildNode, parseNodeErr := parseBuildCanonicalRenderNode(parseReflectValue.Interface())
		if parseNodeErr != nil {
			return nil, parseNodeErr
		}
		return []*canonicalRenderNode{buildNode}, nil
	}
	buildChildrenCount := parseReflectValue.Len()
	buildChildren := make([]*canonicalRenderNode, 0, buildChildrenCount)
	for parseIndex := 0; parseIndex < buildChildrenCount; parseIndex++ {
		buildNode, parseNodeErr := parseBuildCanonicalRenderNode(parseReflectValue.Index(parseIndex).Interface())
		if parseNodeErr != nil {
			return nil, parseNodeErr
		}
		buildChildren = append(buildChildren, buildNode)
	}
	return buildChildren, nil
}

// parseBuildCanonicalProps extracts canonical host props from one host-element payload map.
func parseBuildCanonicalProps(parseMapValue map[string]any) ([]RenderPropRecord, error) {
	parsePropsValue, hasPropsValue := parseMapValue["props"]
	parseHasInlineProps := false
	for getKey := range parseMapValue {
		if _, hasReservedKey := storeCanonicalReservedFieldSet[getKey]; hasReservedKey {
			continue
		}
		parseHasInlineProps = true
		break
	}
	if !parseHasInlineProps {
		return parseBuildCanonicalPropsFromPropsValue(parsePropsValue, hasPropsValue)
	}
	buildRawValueCapacity := len(parseMapValue)
	switch getPropsValue := parsePropsValue.(type) {
	case map[string]any:
		buildRawValueCapacity += len(getPropsValue)
	case map[string]string:
		buildRawValueCapacity += len(getPropsValue)
	}
	buildRawValueByKey := make(map[string]any, buildRawValueCapacity)
	if hasPropsValue {
		switch getPropsValue := parsePropsValue.(type) {
		case map[string]any:
			for getPropKey, getPropValue := range getPropsValue {
				buildRawValueByKey[getPropKey] = getPropValue
			}
		case map[string]string:
			for getPropKey, getPropValue := range getPropsValue {
				buildRawValueByKey[getPropKey] = getPropValue
			}
		default:
			parseReflectProps := reflect.ValueOf(parsePropsValue)
			buildPropsValue, parsePropsErr := parseBuildCanonicalMapValue(parseReflectProps)
			if parsePropsErr == nil {
				for getPropKey, getPropValue := range buildPropsValue {
					buildRawValueByKey[getPropKey] = getPropValue
				}
			}
		}
	}
	for getKey, getValue := range parseMapValue {
		if _, hasReservedKey := storeCanonicalReservedFieldSet[getKey]; hasReservedKey {
			continue
		}
		buildRawValueByKey[getKey] = getValue
	}
	return parseBuildCanonicalPropsFromRawValueByKey(buildRawValueByKey)
}

// parseBuildCanonicalPropsFromPropsValue extracts canonical props from one optional props payload.
func parseBuildCanonicalPropsFromPropsValue(parsePropsValue any, hasPropsValue bool) ([]RenderPropRecord, error) {
	if !hasPropsValue {
		return nil, nil
	}
	switch getPropsValue := parsePropsValue.(type) {
	case map[string]any:
		return parseBuildCanonicalPropsFromRawValueByKey(getPropsValue)
	case map[string]string:
		buildRawValueByKey := make(map[string]any, len(getPropsValue))
		for getPropKey, getPropValue := range getPropsValue {
			buildRawValueByKey[getPropKey] = getPropValue
		}
		return parseBuildCanonicalPropsFromRawValueByKey(buildRawValueByKey)
	default:
		parseReflectProps := reflect.ValueOf(parsePropsValue)
		buildPropsValue, parsePropsErr := parseBuildCanonicalMapValue(parseReflectProps)
		if parsePropsErr != nil {
			return nil, nil
		}
		return parseBuildCanonicalPropsFromRawValueByKey(buildPropsValue)
	}
}

// parseBuildCanonicalPropsFromRawValueByKey converts one raw prop map into canonical prop records.
func parseBuildCanonicalPropsFromRawValueByKey(parseRawValueByKey map[string]any) ([]RenderPropRecord, error) {
	if len(parseRawValueByKey) == 0 {
		return nil, nil
	}
	buildPropRecords := make([]RenderPropRecord, 0, len(parseRawValueByKey))
	for getPropKey, getPropRawValue := range parseRawValueByKey {
		buildPropRecord, hasPropRecord, parsePropErr := parseBuildCanonicalPropRecord(getPropKey, getPropRawValue)
		if parsePropErr != nil {
			return nil, parsePropErr
		}
		if !hasPropRecord {
			continue
		}
		buildPropRecords = append(buildPropRecords, buildPropRecord)
	}
	return parseFinalizeCanonicalPropRecords(buildPropRecords)
}

// parseFinalizeCanonicalPropRecords sorts canonical prop records and rejects duplicate canonical keys.
func parseFinalizeCanonicalPropRecords(parsePropRecords []RenderPropRecord) ([]RenderPropRecord, error) {
	if len(parsePropRecords) <= 1 {
		return parsePropRecords, nil
	}
	sort.Slice(parsePropRecords, func(parseLeftIndex int, parseRightIndex int) bool {
		parseLeftRecord := parsePropRecords[parseLeftIndex]
		parseRightRecord := parsePropRecords[parseRightIndex]
		if parseLeftRecord.Key == parseRightRecord.Key {
			return parseLeftRecord.Kind < parseRightRecord.Kind
		}
		return parseLeftRecord.Key < parseRightRecord.Key
	})
	for parseIndex := 1; parseIndex < len(parsePropRecords); parseIndex++ {
		if parsePropRecords[parseIndex-1].Key == parsePropRecords[parseIndex].Key {
			return nil, fmt.Errorf("runtime2: duplicate prop key %q", parsePropRecords[parseIndex].Key)
		}
	}
	return parsePropRecords, nil
}

// parseBuildCanonicalPropRecord maps one raw key-value pair into one supported prop record.
func parseBuildCanonicalPropRecord(parseKey string, parseRawValue any) (RenderPropRecord, bool, error) {
	if parseKey == "" {
		return RenderPropRecord{}, false, nil
	}
	buildKey := parseKey
	parseLastIndex := len(buildKey) - 1
	if buildKey[0] <= ' ' || buildKey[parseLastIndex] <= ' ' {
		buildKey = strings.TrimSpace(buildKey)
		if buildKey == "" {
			return RenderPropRecord{}, false, nil
		}
	}
	switch buildKey {
	case "class":
		if parseFamilyErr := parseValidateCanonicalPropFamilyConstant("class"); parseFamilyErr != nil {
			return RenderPropRecord{}, false, parseFamilyErr
		}
		return RenderPropRecord{
			Kind:  RenderPropKindClass,
			Key:   buildKey,
			Value: parseGetCanonicalStringValue(parseRawValue),
		}, true, nil
	case "style":
		buildStyleValue, parseStyleErr := FormatRenderStyleValue(parseRawValue)
		if parseStyleErr != nil {
			return RenderPropRecord{}, false, parseStyleErr
		}
		if parseFamilyErr := parseValidateCanonicalPropFamilyConstant("style"); parseFamilyErr != nil {
			return RenderPropRecord{}, false, parseFamilyErr
		}
		return RenderPropRecord{
			Kind:  RenderPropKindStyle,
			Key:   buildKey,
			Value: buildStyleValue,
		}, true, nil
	case "text":
		return RenderPropRecord{
			Kind:  RenderPropKindTextAdjacent,
			Key:   buildKey,
			Value: parseGetCanonicalStringValue(parseRawValue),
		}, true, nil
	}
	if parseHasCanonicalAriaPropKey(buildKey) {
		if parseFamilyErr := parseValidateCanonicalPropFamilyConstant("aria"); parseFamilyErr != nil {
			return RenderPropRecord{}, false, parseFamilyErr
		}
		return RenderPropRecord{
			Kind:  RenderPropKindAria,
			Key:   buildKey,
			Value: parseGetCanonicalStringValue(parseRawValue),
		}, true, nil
	}
	if parseHasCanonicalDataPropKey(buildKey) {
		if parseFamilyErr := parseValidateCanonicalPropFamilyConstant("data"); parseFamilyErr != nil {
			return RenderPropRecord{}, false, parseFamilyErr
		}
		return RenderPropRecord{
			Kind:  RenderPropKindData,
			Key:   buildKey,
			Value: parseGetCanonicalStringValue(parseRawValue),
		}, true, nil
	}
	return RenderPropRecord{}, false, nil
}

// parseValidateCanonicalPropFamilyConstant validates one known prop-family constant against the allowed-family set.
func parseValidateCanonicalPropFamilyConstant(parsePropFamily string) error {
	if _, hasRenderAllowedPropFamily := storeRenderAllowedPropFamilySet[parsePropFamily]; !hasRenderAllowedPropFamily {
		return fmt.Errorf("runtime2: prop family %q is not allowed in first-slice worker-renderable regions", parsePropFamily)
	}
	return nil
}

// parseHasCanonicalAriaPropKey reports whether one canonical key has an aria-* prefix.
func parseHasCanonicalAriaPropKey(parseKey string) bool {
	return len(parseKey) > 5 &&
		parseKey[0] == 'a' &&
		parseKey[1] == 'r' &&
		parseKey[2] == 'i' &&
		parseKey[3] == 'a' &&
		parseKey[4] == '-'
}

// parseHasCanonicalDataPropKey reports whether one canonical key has a data-* prefix.
func parseHasCanonicalDataPropKey(parseKey string) bool {
	return len(parseKey) > 5 &&
		parseKey[0] == 'd' &&
		parseKey[1] == 'a' &&
		parseKey[2] == 't' &&
		parseKey[3] == 'a' &&
		parseKey[4] == '-'
}
