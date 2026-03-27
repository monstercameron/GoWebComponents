package runtime2

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"reflect"
	"sort"
	"strings"
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
	getProps    []RenderPropRecord
	getChildren []*canonicalRenderNode
}

type canonicalRenderTree struct {
	getRootNodeID  uint64
	getNodeByID    map[uint64]canonicalRenderNodeState
	getDepthByNode map[uint64]int
}

type canonicalRenderNodeState struct {
	getNodeID       uint64
	getKind         RenderNodeKind
	getTag          string
	getText         string
	getKey          string
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

// BuildCanonicalRenderIR converts one validated display-only render output into canonical render IR tables.
func BuildCanonicalRenderIR(parseRenderOutput any) (CanonicalRenderIR, error) {
	buildRootNode, parseRootErr := parseBuildCanonicalRootNode(parseRenderOutput)
	if parseRootErr != nil {
		return CanonicalRenderIR{}, parseRootErr
	}
	parseAssignCanonicalNodeIDs(buildRootNode)
	buildStringValues := []string{}
	parseAppendCanonicalStringValues(buildRootNode, &buildStringValues)
	buildStringTable := BuildRenderStringTable(buildStringValues)
	buildNodeOrder := parseBuildCanonicalNodeOrder(buildRootNode)
	buildNodeIndexByID := make(map[uint64]int, len(buildNodeOrder))
	for parseIndex, getNode := range buildNodeOrder {
		buildNodeIndexByID[getNode.getNodeID] = parseIndex
	}
	buildPropRecords := make([]RenderPropRecordRaw, 0)
	buildNodeRecords := make([]RenderNodeRecordRaw, 0, len(buildNodeOrder))
	for _, getNode := range buildNodeOrder {
		buildChildStart := 0
		if len(getNode.getChildren) > 0 {
			buildChildStart = buildNodeIndexByID[getNode.getChildren[0].getNodeID]
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
			ChildCount: uint32(len(getNode.getChildren)),
			PropStart:  uint32(buildPropStart),
			PropCount:  uint32(len(getNode.getProps)),
			TextRef:    buildTextRef,
		}
		if strings.TrimSpace(getNode.getKey) != "" {
			buildNodeRecord.KeyText = getNode.getKey
			buildNodeRecord.KeyHash = parseHashCanonicalString("key:"+getNode.getKey, nil)
		}
		buildNodeRecords = append(buildNodeRecords, buildNodeRecord)
	}
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
	return CanonicalRenderIR{
		GetRootNodeID:  buildRootNode.getNodeID,
		GetStringTable: buildStringTable,
		GetNodeRecords: buildNodeRecords,
		GetPropRecords: buildPropRecords,
	}, nil
}

// BuildRenderStringTableFromRenderOutput extracts one canonical string table from one render output tree.
func BuildRenderStringTableFromRenderOutput(parseRenderOutput any) (RenderStringTable, error) {
	buildRootNode, parseRootErr := parseBuildCanonicalRootNode(parseRenderOutput)
	if parseRootErr != nil {
		return RenderStringTable{}, parseRootErr
	}
	buildStringValues := []string{}
	parseAppendCanonicalStringValues(buildRootNode, &buildStringValues)
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
	parseNodeByID := make(map[uint64]canonicalRenderNodeState, len(parseIR.GetNodeRecords))
	for _, getNodeRecord := range parseNodeTable.Records {
		buildNodeState := canonicalRenderNodeState{
			getNodeID:    getNodeRecord.NodeID,
			getKind:      getNodeRecord.Kind,
			getKey:       getNodeRecord.KeyText,
			getPropByKey: make(map[string]RenderPropRecord),
		}
		switch getNodeRecord.Kind {
		case RenderNodeKindText:
			getTextValue, getTextErr := parseIR.GetStringTable.GetRenderStringByRef(getNodeRecord.TextRef)
			if getTextErr != nil {
				return canonicalRenderTree{}, fmt.Errorf("runtime2: text node id %d has invalid text reference: %w", getNodeRecord.NodeID, getTextErr)
			}
			buildNodeState.getText = getTextValue
		case RenderNodeKindHostElement:
			getTagValue, getTagErr := parseIR.GetStringTable.GetRenderStringByRef(getNodeRecord.TextRef)
			if getTagErr != nil {
				return canonicalRenderTree{}, fmt.Errorf("runtime2: host node id %d has invalid tag reference: %w", getNodeRecord.NodeID, getTagErr)
			}
			buildNodeState.getTag = getTagValue
		}
		if getNodeRecord.PropCount > 0 {
			buildPropStart := int(getNodeRecord.PropStart)
			buildPropEnd := buildPropStart + int(getNodeRecord.PropCount)
			if buildPropStart < 0 || buildPropEnd > len(parseIR.GetPropRecords) {
				return canonicalRenderTree{}, fmt.Errorf("runtime2: node id %d prop span [%d:%d) is out of range", getNodeRecord.NodeID, buildPropStart, buildPropEnd)
			}
			parsePropRecords, parsePropErr := ParseRenderPropRecords(parseIR.GetPropRecords[buildPropStart:buildPropEnd], parseIR.GetStringTable)
			if parsePropErr != nil {
				return canonicalRenderTree{}, fmt.Errorf("runtime2: node id %d props are invalid: %w", getNodeRecord.NodeID, parsePropErr)
			}
			for _, getPropRecord := range parsePropRecords {
				buildNodeState.getPropByKey[getPropRecord.Key] = getPropRecord
			}
		}
		parseNodeByID[getNodeRecord.NodeID] = buildNodeState
	}
	for _, getNodeRecord := range parseNodeTable.Records {
		if getNodeRecord.ChildCount == 0 {
			continue
		}
		parseChildOrder, parseChildErr := parseNodeTable.GetRenderNodeChildOrder(getNodeRecord.NodeID)
		if parseChildErr != nil {
			return canonicalRenderTree{}, parseChildErr
		}
		getParentState := parseNodeByID[getNodeRecord.NodeID]
		getParentState.getChildNodeIDs = append([]uint64(nil), parseChildOrder...)
		parseNodeByID[getNodeRecord.NodeID] = getParentState
		for _, getChildNodeID := range parseChildOrder {
			getChildState := parseNodeByID[getChildNodeID]
			getChildState.getParentNodeID = getNodeRecord.NodeID
			parseNodeByID[getChildNodeID] = getChildState
		}
	}
	parseRootNodeID := parseIR.GetRootNodeID
	if parseRootNodeID == 0 {
		parseRootNodeID = parseIR.GetNodeRecords[0].NodeID
	}
	if _, hasRootNode := parseNodeByID[parseRootNodeID]; !hasRootNode {
		return canonicalRenderTree{}, fmt.Errorf("runtime2: canonical root node id %d is missing", parseRootNodeID)
	}
	buildDepthByNode := make(map[uint64]int, len(parseNodeByID))
	parseQueue := []uint64{parseRootNodeID}
	buildDepthByNode[parseRootNodeID] = 0
	for len(parseQueue) > 0 {
		getNodeID := parseQueue[0]
		parseQueue = parseQueue[1:]
		getDepth := buildDepthByNode[getNodeID]
		for _, getChildNodeID := range parseNodeByID[getNodeID].getChildNodeIDs {
			buildDepthByNode[getChildNodeID] = getDepth + 1
			parseQueue = append(parseQueue, getChildNodeID)
		}
	}
	return canonicalRenderTree{
		getRootNodeID:  parseRootNodeID,
		getNodeByID:    parseNodeByID,
		getDepthByNode: buildDepthByNode,
	}, nil
}

// GetCanonicalNodeState resolves one decoded canonical node by ID.
func (parseTree canonicalRenderTree) GetCanonicalNodeState(parseNodeID uint64) (canonicalRenderNodeState, bool) {
	getNodeState, hasNodeState := parseTree.getNodeByID[parseNodeID]
	return getNodeState, hasNodeState
}

// GetCanonicalNodeDepth resolves one decoded canonical node depth by ID.
func (parseTree canonicalRenderTree) GetCanonicalNodeDepth(parseNodeID uint64) int {
	return parseTree.getDepthByNode[parseNodeID]
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
			getText: fmt.Sprintf("%v", getValue),
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
	buildKind := strings.ToLower(strings.TrimSpace(parseGetCanonicalStringValue(parseMapValue["kind"])))
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
		return &canonicalRenderNode{
			getKind:     RenderNodeKindFragment,
			getKey:      parseGetCanonicalStringValue(parseMapValue["key"]),
			getChildren: buildChildren,
		}, nil
	case "host-element", "element":
		buildTag := strings.TrimSpace(parseGetCanonicalStringValue(parseMapValue["tag"]))
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
		return &canonicalRenderNode{
			getKind:     RenderNodeKindHostElement,
			getTag:      buildTag,
			getKey:      parseGetCanonicalStringValue(parseMapValue["key"]),
			getProps:    buildProps,
			getChildren: buildChildren,
		}, nil
	default:
		return nil, fmt.Errorf("runtime2: unsupported render node kind %q", buildKind)
	}
}

// parseBuildCanonicalChildren normalizes one children payload into a canonical child-node list.
func parseBuildCanonicalChildren(parseChildrenValue any) ([]*canonicalRenderNode, error) {
	if parseChildrenValue == nil {
		return nil, nil
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
		buildNode, parseNodeErr := parseBuildCanonicalRenderNode(parseChildrenValue)
		if parseNodeErr != nil {
			return nil, parseNodeErr
		}
		return []*canonicalRenderNode{buildNode}, nil
	}
	buildChildren := make([]*canonicalRenderNode, 0, parseReflectValue.Len())
	for parseIndex := 0; parseIndex < parseReflectValue.Len(); parseIndex++ {
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
	buildRawValueByKey := make(map[string]any)
	if parsePropsValue, hasPropsValue := parseMapValue["props"]; hasPropsValue {
		parseReflectProps := reflect.ValueOf(parsePropsValue)
		buildPropsValue, parsePropsErr := parseBuildCanonicalMapValue(parseReflectProps)
		if parsePropsErr == nil {
			for getPropKey, getPropValue := range buildPropsValue {
				buildRawValueByKey[getPropKey] = getPropValue
			}
		}
	}
	for getKey, getValue := range parseMapValue {
		if _, hasReservedKey := storeCanonicalReservedFieldSet[getKey]; hasReservedKey {
			continue
		}
		buildRawValueByKey[getKey] = getValue
	}
	buildPropRecords := make([]RenderPropRecord, 0, len(buildRawValueByKey))
	for getPropKey, getPropRawValue := range buildRawValueByKey {
		buildPropRecord, hasPropRecord, parsePropErr := parseBuildCanonicalPropRecord(getPropKey, getPropRawValue)
		if parsePropErr != nil {
			return nil, parsePropErr
		}
		if !hasPropRecord {
			continue
		}
		buildPropRecords = append(buildPropRecords, buildPropRecord)
	}
	sort.Slice(buildPropRecords, func(parseLeftIndex int, parseRightIndex int) bool {
		parseLeftRecord := buildPropRecords[parseLeftIndex]
		parseRightRecord := buildPropRecords[parseRightIndex]
		if parseLeftRecord.Key == parseRightRecord.Key {
			return parseLeftRecord.Kind < parseRightRecord.Kind
		}
		return parseLeftRecord.Key < parseRightRecord.Key
	})
	for parseIndex := 1; parseIndex < len(buildPropRecords); parseIndex++ {
		if buildPropRecords[parseIndex-1].Key == buildPropRecords[parseIndex].Key {
			return nil, fmt.Errorf("runtime2: duplicate prop key %q", buildPropRecords[parseIndex].Key)
		}
	}
	return buildPropRecords, nil
}

// parseBuildCanonicalPropRecord maps one raw key-value pair into one supported prop record.
func parseBuildCanonicalPropRecord(parseKey string, parseRawValue any) (RenderPropRecord, bool, error) {
	buildKey := strings.TrimSpace(parseKey)
	if buildKey == "" {
		return RenderPropRecord{}, false, nil
	}
	buildPropRecord := RenderPropRecord{
		Key: buildKey,
	}
	switch {
	case buildKey == "class":
		buildPropRecord.Kind = RenderPropKindClass
		buildPropRecord.Value = parseGetCanonicalStringValue(parseRawValue)
	case buildKey == "style":
		buildStyleValue, parseStyleErr := FormatRenderStyleValue(parseRawValue)
		if parseStyleErr != nil {
			return RenderPropRecord{}, false, parseStyleErr
		}
		buildPropRecord.Kind = RenderPropKindStyle
		buildPropRecord.Value = buildStyleValue
	case strings.HasPrefix(buildKey, "aria-"):
		buildPropRecord.Kind = RenderPropKindAria
		buildPropRecord.Value = parseGetCanonicalStringValue(parseRawValue)
	case strings.HasPrefix(buildKey, "data-"):
		buildPropRecord.Kind = RenderPropKindData
		buildPropRecord.Value = parseGetCanonicalStringValue(parseRawValue)
	case buildKey == "text":
		buildPropRecord.Kind = RenderPropKindTextAdjacent
		buildPropRecord.Value = parseGetCanonicalStringValue(parseRawValue)
	default:
		return RenderPropRecord{}, false, nil
	}
	buildPropFamily := "class"
	switch buildPropRecord.Kind {
	case RenderPropKindClass:
		buildPropFamily = "class"
	case RenderPropKindStyle:
		buildPropFamily = "style"
	case RenderPropKindAria:
		buildPropFamily = "aria"
	case RenderPropKindData:
		buildPropFamily = "data"
	}
	if buildPropRecord.Kind != RenderPropKindTextAdjacent {
		if parseFamilyErr := ValidateWorkerRenderablePropFamily(buildPropFamily); parseFamilyErr != nil {
			return RenderPropRecord{}, false, parseFamilyErr
		}
	}
	return buildPropRecord, true, nil
}

// parseAssignCanonicalNodeIDs assigns deterministic node IDs to one canonical tree.
func parseAssignCanonicalNodeIDs(parseRootNode *canonicalRenderNode) {
	if parseRootNode == nil {
		return
	}
	parseUsedNodeIDs := map[uint64]struct{}{}
	parseAssignCanonicalNodeID(parseRootNode, "root", parseUsedNodeIDs)
}

// parseAssignCanonicalNodeID recursively assigns deterministic path-hash node IDs.
func parseAssignCanonicalNodeID(parseNode *canonicalRenderNode, parsePath string, parseUsedNodeIDs map[uint64]struct{}) {
	if parseNode == nil {
		return
	}
	parseNode.getNodeID = parseHashCanonicalString(parsePath, parseUsedNodeIDs)
	for parseChildIndex, getChildNode := range parseNode.getChildren {
		buildPathToken := fmt.Sprintf("idx:%d", parseChildIndex)
		if strings.TrimSpace(getChildNode.getKey) != "" {
			buildPathToken = "key:" + getChildNode.getKey
		}
		parseAssignCanonicalNodeID(getChildNode, parsePath+"/"+buildPathToken, parseUsedNodeIDs)
	}
}

// parseHashCanonicalString hashes one canonical string and returns a stable, non-zero, collision-safe ID.
func parseHashCanonicalString(parseValue string, parseUsedIDs map[uint64]struct{}) uint64 {
	buildHasher := fnv.New64a()
	_, _ = buildHasher.Write([]byte(parseValue))
	buildID := buildHasher.Sum64()
	if buildID == 0 {
		buildID = 1
	}
	if parseUsedIDs == nil {
		return buildID
	}
	for {
		if _, hasExistingID := parseUsedIDs[buildID]; !hasExistingID {
			parseUsedIDs[buildID] = struct{}{}
			return buildID
		}
		buildID++
		if buildID == 0 {
			buildID = 1
		}
	}
}

// parseAppendCanonicalStringValues appends one canonical tree's string payload values.
func parseAppendCanonicalStringValues(parseNode *canonicalRenderNode, parseStringValues *[]string) {
	if parseNode == nil || parseStringValues == nil {
		return
	}
	switch parseNode.getKind {
	case RenderNodeKindText:
		*parseStringValues = append(*parseStringValues, parseNode.getText)
	case RenderNodeKindHostElement:
		*parseStringValues = append(*parseStringValues, parseNode.getTag)
	}
	if strings.TrimSpace(parseNode.getKey) != "" {
		*parseStringValues = append(*parseStringValues, parseNode.getKey)
	}
	for _, getProp := range parseNode.getProps {
		*parseStringValues = append(*parseStringValues, getProp.Key, getProp.Value)
	}
	for _, getChildNode := range parseNode.getChildren {
		parseAppendCanonicalStringValues(getChildNode, parseStringValues)
	}
}

// parseBuildCanonicalNodeOrder builds one breadth-first node order for child-span encoding.
func parseBuildCanonicalNodeOrder(parseRootNode *canonicalRenderNode) []*canonicalRenderNode {
	if parseRootNode == nil {
		return nil
	}
	buildNodeOrder := []*canonicalRenderNode{parseRootNode}
	for parseIndex := 0; parseIndex < len(buildNodeOrder); parseIndex++ {
		getNode := buildNodeOrder[parseIndex]
		buildNodeOrder = append(buildNodeOrder, getNode.getChildren...)
	}
	return buildNodeOrder
}

// parseBuildCanonicalMapValue normalizes one map-like reflection payload into map[string]any.
func parseBuildCanonicalMapValue(parseMapValue reflect.Value) (map[string]any, error) {
	if !parseMapValue.IsValid() {
		return nil, fmt.Errorf("runtime2: map value is invalid")
	}
	for parseMapValue.IsValid() && (parseMapValue.Kind() == reflect.Interface || parseMapValue.Kind() == reflect.Pointer) {
		if parseMapValue.IsNil() {
			return nil, fmt.Errorf("runtime2: map value is nil")
		}
		parseMapValue = parseMapValue.Elem()
	}
	if !parseMapValue.IsValid() || parseMapValue.Kind() != reflect.Map {
		return nil, fmt.Errorf("runtime2: value %T is not a map", parseMapValue.Interface())
	}
	buildMapValue := make(map[string]any, parseMapValue.Len())
	for _, getMapKey := range parseMapValue.MapKeys() {
		buildKey := fmt.Sprintf("%v", getMapKey.Interface())
		buildMapValue[buildKey] = parseMapValue.MapIndex(getMapKey).Interface()
	}
	return buildMapValue, nil
}

// parseGetCanonicalStringValue stringifies one arbitrary value with stable formatting.
func parseGetCanonicalStringValue(parseValue any) string {
	switch getValue := parseValue.(type) {
	case nil:
		return ""
	case string:
		return getValue
	default:
		return fmt.Sprintf("%v", getValue)
	}
}

// parseBuildCanonicalJSON builds one deterministic JSON text representation for scalar fallback rendering.
func parseBuildCanonicalJSON(parseValue any) (string, error) {
	buildValueJSON, parseValueErr := json.Marshal(parseValue)
	if parseValueErr != nil {
		return "", fmt.Errorf("runtime2: canonicalize render value %T: %w", parseValue, parseValueErr)
	}
	return string(buildValueJSON), nil
}
