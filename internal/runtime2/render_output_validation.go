package runtime2

import (
	"fmt"
	"reflect"
	"strings"
)

// ValidateWorkerRenderableRenderOutput verifies worker render output excludes first-slice portal-like markers.
func ValidateWorkerRenderableRenderOutput(parseRenderOutput any) error {
	if parseRenderOutput == nil {
		return nil
	}
	return validateWorkerRenderableRenderOutputValue(reflect.ValueOf(parseRenderOutput), "render_output")
}

// validateWorkerRenderableRenderOutputValue walks one render-output graph and rejects portal-like markers.
func validateWorkerRenderableRenderOutputValue(parseOutputValue reflect.Value, parseOutputPath string) error {
	if !parseOutputValue.IsValid() {
		return nil
	}
	switch parseOutputValue.Kind() {
	case reflect.Interface, reflect.Pointer:
		if parseOutputValue.IsNil() {
			return nil
		}
		return validateWorkerRenderableRenderOutputValue(parseOutputValue.Elem(), parseOutputPath)
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return nil
	case reflect.Slice, reflect.Array:
		for parseIndex := 0; parseIndex < parseOutputValue.Len(); parseIndex++ {
			parseItemPath := fmt.Sprintf("%s[%d]", parseOutputPath, parseIndex)
			if parseErr := validateWorkerRenderableRenderOutputValue(parseOutputValue.Index(parseIndex), parseItemPath); parseErr != nil {
				return parseErr
			}
		}
		return nil
	case reflect.Map:
		for _, parseKey := range parseOutputValue.MapKeys() {
			parseKeyName := fmt.Sprintf("%v", parseKey.Interface())
			if parseKey.Kind() == reflect.String {
				parseKeyName = parseKey.String()
			}
			if isPortalLikeOutputName(parseKeyName) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported portal marker", parseOutputPath, parseKeyName)
			}
			if isDOMInteropLikeOutputName(parseKeyName) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported direct DOM interop marker", parseOutputPath, parseKeyName)
			}
			parseItemValue := parseOutputValue.MapIndex(parseKey)
			if isPortalLikeOutputKindField(parseKeyName) && isPortalLikeOutputKindValue(parseItemValue) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported portal node kind", parseOutputPath, parseKeyName)
			}
			if isPortalLikeOutputKindField(parseKeyName) && isDOMInteropLikeOutputKindValue(parseItemValue) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported direct DOM interop kind", parseOutputPath, parseKeyName)
			}
			parseItemPath := fmt.Sprintf("%s.%s", parseOutputPath, parseKeyName)
			if parseErr := validateWorkerRenderableRenderOutputValue(parseItemValue, parseItemPath); parseErr != nil {
				return parseErr
			}
		}
		return nil
	case reflect.Struct:
		for parseIndex := 0; parseIndex < parseOutputValue.NumField(); parseIndex++ {
			parseField := parseOutputValue.Type().Field(parseIndex)
			if parseField.PkgPath != "" {
				continue
			}
			if isPortalLikeOutputName(parseField.Name) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported portal marker", parseOutputPath, parseField.Name)
			}
			if isDOMInteropLikeOutputName(parseField.Name) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported direct DOM interop marker", parseOutputPath, parseField.Name)
			}
			parseFieldValue := parseOutputValue.Field(parseIndex)
			if isPortalLikeOutputKindField(parseField.Name) && isPortalLikeOutputKindValue(parseFieldValue) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported portal node kind", parseOutputPath, parseField.Name)
			}
			if isPortalLikeOutputKindField(parseField.Name) && isDOMInteropLikeOutputKindValue(parseFieldValue) {
				return fmt.Errorf("runtime2: %s.%s uses unsupported direct DOM interop kind", parseOutputPath, parseField.Name)
			}
			parseFieldPath := fmt.Sprintf("%s.%s", parseOutputPath, parseField.Name)
			if parseErr := validateWorkerRenderableRenderOutputValue(parseFieldValue, parseFieldPath); parseErr != nil {
				return parseErr
			}
		}
		return nil
	default:
		return nil
	}
}

// isPortalLikeOutputName reports whether one output field name indicates a portal payload.
func isPortalLikeOutputName(parseName string) bool {
	parseNormalizedName := strings.ToLower(strings.TrimSpace(parseName))
	switch parseNormalizedName {
	case "portal", "portals", "createportal", "portal_target", "portal-target", "portaltarget", "portal_container", "portal-container", "portalcontainer":
		return true
	default:
		return false
	}
}

// isPortalLikeOutputKindField reports whether one field name is used to describe render-node kind.
func isPortalLikeOutputKindField(parseName string) bool {
	parseNormalizedName := strings.ToLower(strings.TrimSpace(parseName))
	switch parseNormalizedName {
	case "kind", "type", "node_kind", "node-kind", "nodekind":
		return true
	default:
		return false
	}
}

// isPortalLikeOutputKindValue reports whether one field value describes portal output.
func isPortalLikeOutputKindValue(parseValue reflect.Value) bool {
	for parseValue.IsValid() {
		switch parseValue.Kind() {
		case reflect.Interface, reflect.Pointer:
			if parseValue.IsNil() {
				return false
			}
			parseValue = parseValue.Elem()
		case reflect.String:
			parseNormalizedValue := strings.ToLower(strings.TrimSpace(parseValue.String()))
			return parseNormalizedValue == "portal" || parseNormalizedValue == "createportal"
		default:
			return false
		}
	}
	return false
}

// isDOMInteropLikeOutputName reports whether one output field name indicates direct DOM interop usage.
func isDOMInteropLikeOutputName(parseName string) bool {
	parseNormalizedName := strings.ToLower(strings.TrimSpace(parseName))
	switch parseNormalizedName {
	case "dom", "domnode", "dom_node", "dom-node", "domref", "dom_ref", "dom-ref", "domhandle", "dom_handle", "dom-handle", "elementref", "element_ref", "element-ref", "elementhandle", "element_handle", "element-handle", "noderef", "node_ref", "node-ref", "nodehandle", "node_handle", "node-handle", "document", "window", "interop", "queryselector", "query_selector", "query-selector", "dominterop", "dom_interop", "dom-interop":
		return true
	default:
		return false
	}
}

// isDOMInteropLikeOutputKindValue reports whether one output kind value implies direct DOM interop output.
func isDOMInteropLikeOutputKindValue(parseValue reflect.Value) bool {
	for parseValue.IsValid() {
		switch parseValue.Kind() {
		case reflect.Interface, reflect.Pointer:
			if parseValue.IsNil() {
				return false
			}
			parseValue = parseValue.Elem()
		case reflect.String:
			parseNormalizedValue := strings.ToLower(strings.TrimSpace(parseValue.String()))
			switch parseNormalizedValue {
			case "dom", "domnode", "dom_node", "dom-node", "domref", "dom_ref", "dom-ref", "document", "window", "queryselector", "query_selector", "query-selector", "dominterop", "dom_interop", "dom-interop":
				return true
			default:
				return false
			}
		default:
			return false
		}
	}
	return false
}
