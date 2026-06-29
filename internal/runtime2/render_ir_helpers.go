package runtime2

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

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
		// Build the child path in one concatenation to avoid extra intermediate allocs.
		var buildChildPath string
		if getChildNode.hasKey {
			buildChildPath = parsePath + "/key:" + getChildNode.getKey
		} else {
			var buildIndexBuf [32]byte
			buildChildPath = parsePath + "/idx:" + string(strconv.AppendInt(buildIndexBuf[:0], int64(parseChildIndex), 10))
		}
		parseAssignCanonicalNodeID(getChildNode, buildChildPath, parseUsedNodeIDs)
	}
}

// parseHashCanonicalString hashes one canonical string and returns a stable, non-zero, collision-safe ID.
func parseHashCanonicalString(parseValue string, parseUsedIDs map[uint64]struct{}) uint64 {
	// Inline FNV-64a to avoid allocating the hash.Hash64 interface and the []byte string copy.
	const fnv64aOffset uint64 = 14695981039346656037
	const fnv64aPrime uint64 = 1099511628211
	buildID := fnv64aOffset
	for parseIndex := 0; parseIndex < len(parseValue); parseIndex++ {
		buildID ^= uint64(parseValue[parseIndex])
		buildID *= fnv64aPrime
	}
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
	if parseNode.hasKey {
		*parseStringValues = append(*parseStringValues, parseNode.getKey)
	}
	for _, getProp := range parseNode.getProps {
		*parseStringValues = append(*parseStringValues, getProp.Key, getProp.Value)
	}
	for _, getChildNode := range parseNode.getChildren {
		parseAppendCanonicalStringValues(getChildNode, parseStringValues)
	}
}

// buildCanonicalStringValuesAndPropCapacityFromNodeOrder collects canonical string payloads and total prop-record capacity from one node order.
func buildCanonicalStringValuesAndPropCapacityFromNodeOrder(parseNodeOrder []*canonicalRenderNode) ([]string, int) {
	if len(parseNodeOrder) == 0 {
		return nil, 0
	}
	buildStringValueCount := 0
	buildPropRecordCapacity := 0
	for _, getNode := range parseNodeOrder {
		switch getNode.getKind {
		case RenderNodeKindText, RenderNodeKindHostElement:
			buildStringValueCount++
		}
		if getNode.hasKey {
			buildStringValueCount++
		}
		getPropCount := len(getNode.getProps)
		buildPropRecordCapacity += getPropCount
		buildStringValueCount += getPropCount * 2
	}
	buildStringValues := make([]string, 0, buildStringValueCount)
	for _, getNode := range parseNodeOrder {
		switch getNode.getKind {
		case RenderNodeKindText:
			buildStringValues = append(buildStringValues, getNode.getText)
		case RenderNodeKindHostElement:
			buildStringValues = append(buildStringValues, getNode.getTag)
		}
		if getNode.hasKey {
			buildStringValues = append(buildStringValues, getNode.getKey)
		}
		for _, getProp := range getNode.getProps {
			buildStringValues = append(buildStringValues, getProp.Key, getProp.Value)
		}
	}
	return buildStringValues, buildPropRecordCapacity
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
	parseKeyKind := parseMapValue.Type().Key().Kind()
	parseMapIterator := parseMapValue.MapRange()
	if parseKeyKind == reflect.String {
		for parseMapIterator.Next() {
			buildMapValue[parseMapIterator.Key().String()] = parseMapIterator.Value().Interface()
		}
		return buildMapValue, nil
	}
	for parseMapIterator.Next() {
		getMapKey := parseMapIterator.Key()
		buildMapValue[parseFormatCanonicalMapKeyValue(getMapKey, parseKeyKind)] = parseMapIterator.Value().Interface()
	}
	return buildMapValue, nil
}

// parseFormatCanonicalMapKeyValue stringifies one reflected map key using kind-specific fast paths.
func parseFormatCanonicalMapKeyValue(parseKeyValue reflect.Value, parseKeyKind reflect.Kind) string {
	switch parseKeyKind {
	case reflect.Bool:
		return strconv.FormatBool(parseKeyValue.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(parseKeyValue.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(parseKeyValue.Uint(), 10)
	case reflect.Float32:
		return strconv.FormatFloat(parseKeyValue.Float(), 'g', -1, 32)
	case reflect.Float64:
		return strconv.FormatFloat(parseKeyValue.Float(), 'g', -1, 64)
	default:
		return parseFormatCanonicalScalarValue(parseKeyValue.Interface())
	}
}

// parseFormatCanonicalScalarValue stringifies scalar render values with a cheaper numeric fast path.
func parseFormatCanonicalScalarValue(parseValue any) string {
	switch getValue := parseValue.(type) {
	case nil:
		return ""
	case string:
		return getValue
	case bool:
		return strconv.FormatBool(getValue)
	case int:
		return strconv.FormatInt(int64(getValue), 10)
	case int8:
		return strconv.FormatInt(int64(getValue), 10)
	case int16:
		return strconv.FormatInt(int64(getValue), 10)
	case int32:
		return strconv.FormatInt(int64(getValue), 10)
	case int64:
		return strconv.FormatInt(getValue, 10)
	case uint:
		return strconv.FormatUint(uint64(getValue), 10)
	case uint8:
		return strconv.FormatUint(uint64(getValue), 10)
	case uint16:
		return strconv.FormatUint(uint64(getValue), 10)
	case uint32:
		return strconv.FormatUint(uint64(getValue), 10)
	case uint64:
		return strconv.FormatUint(getValue, 10)
	case float32:
		return strconv.FormatFloat(float64(getValue), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(getValue, 'g', -1, 64)
	default:
		return fmt.Sprint(getValue)
	}
}

// parseGetCanonicalStringValue stringifies one arbitrary value with stable formatting.
func parseGetCanonicalStringValue(parseValue any) string {
	return parseFormatCanonicalScalarValue(parseValue)
}

// parseBuildCanonicalJSON builds one deterministic JSON text representation for scalar fallback rendering.
func parseBuildCanonicalJSON(parseValue any) (string, error) {
	buildValueJSON, parseValueErr := json.Marshal(parseValue)
	if parseValueErr != nil {
		return "", fmt.Errorf("runtime2: canonicalize render value %T: %w", parseValue, parseValueErr)
	}
	return string(buildValueJSON), nil
}
