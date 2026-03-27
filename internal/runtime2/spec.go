package runtime2

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"
)

type serializableScalar interface {
	~bool |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~string
}

const (
	getSerializablePropsShapeHashSeed  uint64 = 14695981039346656037
	getSerializablePropsShapeHashPrime uint64 = 1099511628211
)

// ParallelRegionSpec stores the serializable public input contract for one parallel region instance.
type ParallelRegionSpec struct {
	RendererID       RendererID
	RegionInstanceID RegionInstanceID
	Props            any
	SourceIDs        []string
}

// NormalizeParallelRegionSpec validates and canonicalizes a parallel region spec before dispatch.
func NormalizeParallelRegionSpec(parseSpec ParallelRegionSpec) (ParallelRegionSpec, error) {
	if _, parseErr := ParseRendererID(string(parseSpec.RendererID)); parseErr != nil {
		return ParallelRegionSpec{}, parseErr
	}
	if _, parseErr := ParseRegionInstanceID(string(parseSpec.RegionInstanceID)); parseErr != nil {
		return ParallelRegionSpec{}, parseErr
	}
	if parseErr := ValidateSerializableProps(parseSpec.Props); parseErr != nil {
		return ParallelRegionSpec{}, parseErr
	}
	parseSourceIDs, parseErr := NormalizeSourceIDs(parseSpec.SourceIDs)
	if parseErr != nil {
		return ParallelRegionSpec{}, parseErr
	}
	parseSpec.SourceIDs = parseSourceIDs
	return parseSpec, nil
}

// ValidateParallelRegionSpec verifies the required IDs, props, and declared sources for a parallel region.
func ValidateParallelRegionSpec(parseSpec ParallelRegionSpec) error {
	_, parseErr := NormalizeParallelRegionSpec(parseSpec)
	return parseErr
}

// NormalizeSourceIDs validates and canonicalizes declared source IDs.
func NormalizeSourceIDs(parseSourceIDs []string) ([]string, error) {
	if len(parseSourceIDs) == 0 {
		return nil, nil
	}
	// Validate all IDs first; copy into a single output slice so we can sort+dedup without a map.
	parseNormalizedSourceIDs := make([]string, 0, len(parseSourceIDs))
	for _, parseSourceID := range parseSourceIDs {
		parseTrimmedSourceID := strings.TrimSpace(parseSourceID)
		if parseTrimmedSourceID == "" {
			return nil, fmt.Errorf("runtime2: source ID is required")
		}
		if parseTrimmedSourceID != parseSourceID {
			return nil, fmt.Errorf("runtime2: source ID %q must not contain surrounding whitespace", parseSourceID)
		}
		for _, parseRune := range parseTrimmedSourceID {
			parseAllowed := parseRune == '.' || parseRune == '-' || parseRune == '_' || parseRune == ':'
			if !parseAllowed && !(parseRune >= 'a' && parseRune <= 'z') && !(parseRune >= 'A' && parseRune <= 'Z') && !(parseRune >= '0' && parseRune <= '9') {
				return nil, fmt.Errorf("runtime2: source ID %q contains unsupported character %q", parseSourceID, string(parseRune))
			}
		}
		parseNormalizedSourceIDs = append(parseNormalizedSourceIDs, parseTrimmedSourceID)
	}
	// Sort then dedup in-place: consecutive equal entries after sort are duplicates.
	sort.Strings(parseNormalizedSourceIDs)
	parseDedupLen := 1
	for parseIndex := 1; parseIndex < len(parseNormalizedSourceIDs); parseIndex++ {
		if parseNormalizedSourceIDs[parseIndex] != parseNormalizedSourceIDs[parseIndex-1] {
			parseNormalizedSourceIDs[parseDedupLen] = parseNormalizedSourceIDs[parseIndex]
			parseDedupLen++
		}
	}
	return parseNormalizedSourceIDs[:parseDedupLen], nil
}

// ValidateSerializableProps verifies props are serializable through the supported Track A contract.
func ValidateSerializableProps(parseProps any) error {
	if parseProps == nil {
		return nil
	}
	if isSerializableAnyFast(parseProps) {
		return nil
	}
	parseValue := reflect.ValueOf(parseProps)
	if isSerializableValueFast(parseValue) {
		return nil
	}
	return validateSerializableValue(parseValue, "props")
}

// buildSerializablePropsFlatShapeFingerprintWithTypeScratch builds a flat-shape fingerprint and returns ordered key/type scratch slices for cache matching.
func buildSerializablePropsFlatShapeFingerprintWithTypeScratch(
	parseProps any,
	parseScratchKeys []string,
	parseScratchTypeMarkers []uint64,
) (uint64, uint64, uint64, []string, []uint64, bool) {
	parsePropsMap, hasPropsMap := parseProps.(map[string]any)
	if !hasPropsMap || len(parsePropsMap) == 0 {
		return 0, 0, 0, parseScratchKeys[:0], parseScratchTypeMarkers[:0], false
	}
	if cap(parseScratchKeys) < len(parsePropsMap) {
		parseScratchKeys = make([]string, 0, len(parsePropsMap))
	}
	if cap(parseScratchTypeMarkers) < len(parsePropsMap) {
		parseScratchTypeMarkers = make([]uint64, 0, len(parsePropsMap))
	}
	parseScratchKeys = parseScratchKeys[:0]
	parseScratchTypeMarkers = parseScratchTypeMarkers[:0]
	for getKey := range parsePropsMap {
		parseScratchKeys = append(parseScratchKeys, getKey)
	}
	sort.Strings(parseScratchKeys)
	buildKeyHash := getSerializablePropsShapeHashSeed ^ 0x11
	buildTypeHash := getSerializablePropsShapeHashSeed ^ 0x31
	for _, getKey := range parseScratchKeys {
		getTypeMarker, hasTypeMarker := buildSerializablePropsFlatTypeMarker(parsePropsMap[getKey])
		if !hasTypeMarker {
			return 0, 0, 0, parseScratchKeys[:0], parseScratchTypeMarkers[:0], false
		}
		parseScratchTypeMarkers = append(parseScratchTypeMarkers, getTypeMarker)
		buildKeyHash = buildSerializablePropsShapeHashString(buildKeyHash, getKey)
		buildTypeHash = buildSerializablePropsShapeHashString(buildTypeHash, getKey)
		buildTypeHash = buildSerializablePropsShapeHashUint64(buildTypeHash, getTypeMarker)
	}
	buildKeyHash = buildSerializablePropsShapeHashUint64(buildKeyHash, uint64(len(parseScratchKeys)))
	buildTypeHash = buildSerializablePropsShapeHashUint64(buildTypeHash, uint64(len(parseScratchKeys)))
	if buildKeyHash == 0 {
		buildKeyHash = 1
	}
	if buildTypeHash == 0 {
		buildTypeHash = 1
	}
	return uint64(len(parseScratchKeys)), buildKeyHash, buildTypeHash, parseScratchKeys, parseScratchTypeMarkers, true
}

// hasSerializablePropsFlatShapeFingerprintMatch reports whether one flat map[string]any matches one cached sorted-key and type-marker shape.
func hasSerializablePropsFlatShapeFingerprintMatch(
	parseProps any,
	parseKeyCount uint64,
	parseSortedKeys []string,
	parseTypeMarkers []uint64,
) bool {
	parsePropsMap, hasPropsMap := parseProps.(map[string]any)
	if !hasPropsMap || len(parsePropsMap) == 0 {
		return false
	}
	if uint64(len(parsePropsMap)) != parseKeyCount {
		return false
	}
	if len(parseSortedKeys) != len(parseTypeMarkers) || len(parseSortedKeys) != len(parsePropsMap) {
		return false
	}
	for parseIndex, getKey := range parseSortedKeys {
		getValue, hasValue := parsePropsMap[getKey]
		if !hasValue {
			return false
		}
		getTypeMarker, hasTypeMarker := buildSerializablePropsFlatTypeMarker(getValue)
		if !hasTypeMarker {
			return false
		}
		if getTypeMarker != parseTypeMarkers[parseIndex] {
			return false
		}
	}
	return true
}

// buildSerializablePropsFlatTypeMarker reports one scalar type marker used by buildSerializablePropsFlatShapeFingerprint.
func buildSerializablePropsFlatTypeMarker(parseValue any) (uint64, bool) {
	switch parseValue.(type) {
	case nil:
		return 1, true
	case bool:
		return 2, true
	case int:
		return 3, true
	case int8:
		return 4, true
	case int16:
		return 5, true
	case int32:
		return 6, true
	case int64:
		return 7, true
	case uint:
		return 8, true
	case uint8:
		return 9, true
	case uint16:
		return 10, true
	case uint32:
		return 11, true
	case uint64:
		return 12, true
	case uintptr:
		return 13, true
	case float32:
		return 14, true
	case float64:
		return 15, true
	case string:
		return 16, true
	default:
		return 0, false
	}
}

// buildSerializablePropsShapeHashString appends one string into a FNV-1a shape hash state.
func buildSerializablePropsShapeHashString(parseSeed uint64, parseValue string) uint64 {
	buildHash := parseSeed
	buildHash = buildSerializablePropsShapeHashUint64(buildHash, uint64(len(parseValue)))
	for parseIndex := 0; parseIndex < len(parseValue); parseIndex++ {
		buildHash ^= uint64(parseValue[parseIndex])
		buildHash *= getSerializablePropsShapeHashPrime
	}
	return buildHash
}

// buildSerializablePropsShapeHashUint64 appends one uint64 value into a FNV-1a shape hash state.
func buildSerializablePropsShapeHashUint64(parseSeed uint64, parseValue uint64) uint64 {
	buildHash := parseSeed
	for parseShift := 0; parseShift < 64; parseShift += 8 {
		buildHash ^= uint64(byte(parseValue >> parseShift))
		buildHash *= getSerializablePropsShapeHashPrime
	}
	return buildHash
}

// isSerializableAnyFast reports whether one common any-shaped value is serializable without reflection-heavy traversal.
func isSerializableAnyFast(parseValue any) bool {
	switch getValue := parseValue.(type) {
	case nil:
		return true
	case bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, uintptr,
		float32, float64,
		string:
		return true
	case []bool, []int, []int8, []int16, []int32, []int64,
		[]uint, []uint8, []uint16, []uint32, []uint64, []uintptr,
		[]float32, []float64,
		[]string:
		return true
	case []any:
		for _, getItem := range getValue {
			if !isSerializableAnyFast(getItem) {
				return false
			}
		}
		return true
	case map[string]bool:
		return isSerializableScalarMapFast(getValue)
	case map[string]int:
		return isSerializableScalarMapFast(getValue)
	case map[string]int8:
		return isSerializableScalarMapFast(getValue)
	case map[string]int16:
		return isSerializableScalarMapFast(getValue)
	case map[string]int32:
		return isSerializableScalarMapFast(getValue)
	case map[string]int64:
		return isSerializableScalarMapFast(getValue)
	case map[string]uint:
		return isSerializableScalarMapFast(getValue)
	case map[string]uint8:
		return isSerializableScalarMapFast(getValue)
	case map[string]uint16:
		return isSerializableScalarMapFast(getValue)
	case map[string]uint32:
		return isSerializableScalarMapFast(getValue)
	case map[string]uint64:
		return isSerializableScalarMapFast(getValue)
	case map[string]uintptr:
		return isSerializableScalarMapFast(getValue)
	case map[string]float32:
		return isSerializableScalarMapFast(getValue)
	case map[string]float64:
		return isSerializableScalarMapFast(getValue)
	case map[string]string:
		return isSerializableScalarMapFast(getValue)
	case map[string]any:
		for getKey, getItem := range getValue {
			if !hasSerializableSafeMapKey(getKey) {
				getNormalizedKey := getSerializableNormalizedName(getKey)
				if hasSerializableRefName(getNormalizedKey) {
					return false
				}
				if hasSerializableDOMInteropName(getNormalizedKey) {
					return false
				}
				if hasSerializableEventClosureName(getNormalizedKey) && hasSerializableFunctionValueAnyFast(getItem) {
					return false
				}
			}
			if !isSerializableAnyFast(getItem) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// isSerializableScalarMapFast reports whether one typed map[string]scalar value stays within the supported serializable key contract.
func isSerializableScalarMapFast[T serializableScalar](parseValue map[string]T) bool {
	for getKey := range parseValue {
		if !hasSerializableSafeMapKey(getKey) {
			getNormalizedKey := getSerializableNormalizedName(getKey)
			if hasSerializableRefName(getNormalizedKey) {
				return false
			}
			if hasSerializableDOMInteropName(getNormalizedKey) {
				return false
			}
		}
	}
	return true
}

// hasSerializableFunctionValueAnyFast reports whether one any-shaped value resolves to a function closure.
func hasSerializableFunctionValueAnyFast(parseValue any) bool {
	if parseValue == nil {
		return false
	}
	parseType := reflect.TypeOf(parseValue)
	if parseType == nil {
		return false
	}
	return parseType.Kind() == reflect.Func
}

// hasSerializableSafeMapKey reports whether one map key is plain lowercase alphanumeric text that cannot match guarded special-name checks.
func hasSerializableSafeMapKey(parseKey string) bool {
	if len(parseKey) == 0 {
		return false
	}
	parseFirstKeyByte := parseKey[0]
	if parseFirstKeyByte >= 'a' && parseFirstKeyByte <= 'z' {
		switch parseFirstKeyByte {
		case 'r', 'd', 'e', 'n', 'w', 'i', 'q', 'o':
			return false
		}
	} else if parseFirstKeyByte < '0' || parseFirstKeyByte > '9' {
		return false
	}
	for parseIndex := 1; parseIndex < len(parseKey); parseIndex++ {
		parseByte := parseKey[parseIndex]
		if (parseByte < 'a' || parseByte > 'z') && (parseByte < '0' || parseByte > '9') {
			return false
		}
	}
	return true
}

// isSerializableValueFast reports whether one value is serializable without building detailed error paths.
func isSerializableValueFast(parseValue reflect.Value) bool {
	if !parseValue.IsValid() {
		return true
	}
	switch parseValue.Kind() {
	case reflect.Interface, reflect.Pointer:
		if parseValue.IsNil() {
			return true
		}
		return isSerializableValueFast(parseValue.Elem())
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return true
	case reflect.Slice, reflect.Array:
		for parseIndex := 0; parseIndex < parseValue.Len(); parseIndex++ {
			if !isSerializableValueFast(parseValue.Index(parseIndex)) {
				return false
			}
		}
		return true
	case reflect.Map:
		if parseValue.Type().Key().Kind() != reflect.String {
			return false
		}
		parseMapIter := parseValue.MapRange()
		for parseMapIter.Next() {
			parseKeyName := parseMapIter.Key().String()
			parseMapValue := parseMapIter.Value()
			if !hasSerializableSafeMapKey(parseKeyName) {
				parseNormalizedKeyName := getSerializableNormalizedName(parseKeyName)
				if hasSerializableRefName(parseNormalizedKeyName) {
					return false
				}
				if hasSerializableDOMInteropName(parseNormalizedKeyName) {
					return false
				}
				if hasSerializableEventClosureName(parseNormalizedKeyName) && isEventClosureValue(parseMapValue) {
					return false
				}
			}
			if !isSerializableValueFast(parseMapValue) {
				return false
			}
		}
		return true
	case reflect.Struct:
		for parseIndex := 0; parseIndex < parseValue.NumField(); parseIndex++ {
			parseField := parseValue.Type().Field(parseIndex)
			if parseField.PkgPath != "" {
				continue
			}
			parseNormalizedFieldName := getSerializableNormalizedName(parseField.Name)
			if hasSerializableRefName(parseNormalizedFieldName) {
				return false
			}
			if hasSerializableDOMInteropName(parseNormalizedFieldName) {
				return false
			}
			parseFieldValue := parseValue.Field(parseIndex)
			if hasSerializableEventClosureName(parseNormalizedFieldName) && isEventClosureValue(parseFieldValue) {
				return false
			}
			if !isSerializableValueFast(parseFieldValue) {
				return false
			}
		}
		return true
	case reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return false
	default:
		return false
	}
}

// validateSerializableValue walks a value recursively and rejects unsupported runtime-local data.
func validateSerializableValue(parseValue reflect.Value, parsePath string) error {
	if !parseValue.IsValid() {
		return nil
	}
	switch parseValue.Kind() {
	case reflect.Interface, reflect.Pointer:
		if parseValue.IsNil() {
			return nil
		}
		return validateSerializableValue(parseValue.Elem(), parsePath)
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return nil
	case reflect.Slice, reflect.Array:
		for parseIndex := 0; parseIndex < parseValue.Len(); parseIndex++ {
			parseItemPath := fmt.Sprintf("%s[%d]", parsePath, parseIndex)
			if parseErr := validateSerializableValue(parseValue.Index(parseIndex), parseItemPath); parseErr != nil {
				return parseErr
			}
		}
		return nil
	case reflect.Map:
		if parseValue.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("runtime2: %s uses unsupported map key kind %s", parsePath, parseValue.Type().Key())
		}
		parseMapIter := parseValue.MapRange()
		for parseMapIter.Next() {
			parseKeyName := parseMapIter.Key().String()
			parseNormalizedKeyName := getSerializableNormalizedName(parseKeyName)
			if hasSerializableRefName(parseNormalizedKeyName) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported ref marker", parsePath, parseKeyName)
			}
			if hasSerializableDOMInteropName(parseNormalizedKeyName) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported direct DOM interop marker", parsePath, parseKeyName)
			}
			parseMapValue := parseMapIter.Value()
			if hasSerializableEventClosureName(parseNormalizedKeyName) && isEventClosureValue(parseMapValue) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported event-closure prop", parsePath, parseKeyName)
			}
			parseItemPath := fmt.Sprintf("%s.%s", parsePath, parseKeyName)
			if parseErr := validateSerializableValue(parseMapValue, parseItemPath); parseErr != nil {
				return parseErr
			}
		}
		return nil
	case reflect.Struct:
		for parseIndex := 0; parseIndex < parseValue.NumField(); parseIndex++ {
			parseField := parseValue.Type().Field(parseIndex)
			if parseField.PkgPath != "" {
				continue
			}
			parseNormalizedFieldName := getSerializableNormalizedName(parseField.Name)
			if hasSerializableRefName(parseNormalizedFieldName) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported ref marker", parsePath, parseField.Name)
			}
			if hasSerializableDOMInteropName(parseNormalizedFieldName) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported direct DOM interop marker", parsePath, parseField.Name)
			}
			parseFieldValue := parseValue.Field(parseIndex)
			if hasSerializableEventClosureName(parseNormalizedFieldName) && isEventClosureValue(parseFieldValue) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported event-closure prop", parsePath, parseField.Name)
			}
			parseFieldPath := fmt.Sprintf("%s.%s", parsePath, parseField.Name)
			if parseErr := validateSerializableValue(parseFieldValue, parseFieldPath); parseErr != nil {
				return parseErr
			}
		}
		return nil
	case reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return fmt.Errorf("runtime2: %s uses unsupported kind %s", parsePath, parseValue.Kind())
	default:
		return fmt.Errorf("runtime2: %s uses unsupported kind %s", parsePath, parseValue.Kind())
	}
}

// hasSerializableRefName reports whether one normalized field or key name represents a disallowed ref marker.
func hasSerializableRefName(parseNormalizedName string) bool {
	return parseNormalizedName == "ref" || parseNormalizedName == "refs"
}

// hasSerializableDOMInteropName reports whether one normalized field or key name represents a disallowed direct DOM interop marker.
func hasSerializableDOMInteropName(parseNormalizedName string) bool {
	switch parseNormalizedName {
	case "dom", "domnode", "dom_node", "dom-node", "domref", "dom_ref", "dom-ref", "domhandle", "dom_handle", "dom-handle", "elementref", "element_ref", "element-ref", "elementhandle", "element_handle", "element-handle", "noderef", "node_ref", "node-ref", "nodehandle", "node_handle", "node-handle", "document", "window", "interop", "queryselector", "query_selector", "query-selector":
		return true
	default:
		return false
	}
}

// hasSerializableEventClosureName reports whether one normalized field or key name maps to an event-style prop key.
func hasSerializableEventClosureName(parseNormalizedName string) bool {
	switch parseNormalizedName {
	case "onclick", "onchange", "oninput", "onsubmit", "onfocus", "onblur", "onkeydown", "onkeyup", "onkeypress", "onmousedown", "onmouseup", "onmouseenter", "onmouseleave", "onmouseover", "onmouseout", "onpointerdown", "onpointerup", "onpointermove", "onpointerenter", "onpointerleave", "ontouchstart", "ontouchend", "onscroll", "onwheel", "onload", "onerror", "onselect", "ondblclick":
		return true
	default:
		return strings.HasPrefix(parseNormalizedName, "on_") || strings.HasPrefix(parseNormalizedName, "on-")
	}
}

// getSerializableNormalizedName normalizes one field or key name while avoiding lowercasing allocations for already-lowercase ASCII names.
func getSerializableNormalizedName(parseName string) string {
	parseTrimmedName := strings.TrimSpace(parseName)
	for parseIndex := 0; parseIndex < len(parseTrimmedName); parseIndex++ {
		parseByte := parseTrimmedName[parseIndex]
		if parseByte >= 'A' && parseByte <= 'Z' {
			return strings.ToLower(parseTrimmedName)
		}
		if parseByte >= utf8.RuneSelf {
			return strings.ToLower(parseTrimmedName)
		}
	}
	return parseTrimmedName
}

// isEventClosureValue reports whether one value resolves to a function closure.
func isEventClosureValue(parseValue reflect.Value) bool {
	for parseValue.IsValid() {
		switch parseValue.Kind() {
		case reflect.Interface, reflect.Pointer:
			if parseValue.IsNil() {
				return false
			}
			parseValue = parseValue.Elem()
		case reflect.Func:
			return true
		default:
			return false
		}
	}
	return false
}
