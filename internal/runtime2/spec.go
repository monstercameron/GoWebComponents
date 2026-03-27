package runtime2

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
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
	parseSeenSourceIDs := make(map[string]bool, len(parseSourceIDs))
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
		if parseSeenSourceIDs[parseTrimmedSourceID] {
			continue
		}
		parseSeenSourceIDs[parseTrimmedSourceID] = true
		parseNormalizedSourceIDs = append(parseNormalizedSourceIDs, parseTrimmedSourceID)
	}
	sort.Strings(parseNormalizedSourceIDs)
	return parseNormalizedSourceIDs, nil
}

// ValidateSerializableProps verifies props are serializable through the supported Track A contract.
func ValidateSerializableProps(parseProps any) error {
	if parseProps == nil {
		return nil
	}
	parseValue := reflect.ValueOf(parseProps)
	if isSerializableValueFast(parseValue) {
		return nil
	}
	return validateSerializableValue(parseValue, "props")
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
			if isRefLikeName(parseKeyName) {
				return false
			}
			if isDOMInteropLikeName(parseKeyName) {
				return false
			}
			parseMapValue := parseMapIter.Value()
			if isEventClosureLikeName(parseKeyName) && isEventClosureValue(parseMapValue) {
				return false
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
			if isRefLikeName(parseField.Name) {
				return false
			}
			if isDOMInteropLikeName(parseField.Name) {
				return false
			}
			parseFieldValue := parseValue.Field(parseIndex)
			if isEventClosureLikeName(parseField.Name) && isEventClosureValue(parseFieldValue) {
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
			if isRefLikeName(parseKeyName) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported ref marker", parsePath, parseKeyName)
			}
			if isDOMInteropLikeName(parseKeyName) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported direct DOM interop marker", parsePath, parseKeyName)
			}
			parseMapValue := parseMapIter.Value()
			if isEventClosureLikeName(parseKeyName) && isEventClosureValue(parseMapValue) {
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
			if isRefLikeName(parseField.Name) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported ref marker", parsePath, parseField.Name)
			}
			if isDOMInteropLikeName(parseField.Name) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported direct DOM interop marker", parsePath, parseField.Name)
			}
			parseFieldValue := parseValue.Field(parseIndex)
			if isEventClosureLikeName(parseField.Name) && isEventClosureValue(parseFieldValue) {
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

// isRefLikeName reports whether one field or key name represents a disallowed ref marker in first-slice worker-renderable inputs.
func isRefLikeName(parseName string) bool {
	parseNormalizedName := strings.ToLower(strings.TrimSpace(parseName))
	return parseNormalizedName == "ref" || parseNormalizedName == "refs"
}

// isDOMInteropLikeName reports whether one field or key name represents a disallowed direct DOM interop marker in first-slice worker-renderable inputs.
func isDOMInteropLikeName(parseName string) bool {
	parseNormalizedName := strings.ToLower(strings.TrimSpace(parseName))
	switch parseNormalizedName {
	case "dom", "domnode", "dom_node", "dom-node", "domref", "dom_ref", "dom-ref", "domhandle", "dom_handle", "dom-handle", "elementref", "element_ref", "element-ref", "elementhandle", "element_handle", "element-handle", "noderef", "node_ref", "node-ref", "nodehandle", "node_handle", "node-handle", "document", "window", "interop", "queryselector", "query_selector", "query-selector":
		return true
	default:
		return false
	}
}

// isEventClosureLikeName reports whether one field or key name maps to an event-style prop key.
func isEventClosureLikeName(parseName string) bool {
	parseNormalizedName := strings.ToLower(strings.TrimSpace(parseName))
	switch parseNormalizedName {
	case "onclick", "onchange", "oninput", "onsubmit", "onfocus", "onblur", "onkeydown", "onkeyup", "onkeypress", "onmousedown", "onmouseup", "onmouseenter", "onmouseleave", "onmouseover", "onmouseout", "onpointerdown", "onpointerup", "onpointermove", "onpointerenter", "onpointerleave", "ontouchstart", "ontouchend", "onscroll", "onwheel", "onload", "onerror", "onselect", "ondblclick":
		return true
	default:
		return strings.HasPrefix(parseNormalizedName, "on_") || strings.HasPrefix(parseNormalizedName, "on-")
	}
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
