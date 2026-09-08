package ui

import (
	"reflect"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// createErrorBoundaryElement is a core package helper.
func createErrorBoundaryElement(parseBoundary runtimeErrorBoundaryComponent, parseRawProps any) Node {
	parseRuntimeBoundary := parseBoundary.runtimeErrorBoundary()
	if parseRuntimeBoundary == nil {
		return nil
	}

	parsePropsMap := map[string]any{}
	if parseFallback, parseOk := extractErrorBoundaryFallback(parseRawProps); parseOk {
		parsePropsMap["fallback"] = parseFallback
	}
	if parseFallback2, parseOk2 := extractErrorBoundaryErrorFallback(parseRawProps); parseOk2 {
		parsePropsMap["errorFallback"] = parseFallback2
	}
	if parseOnError, parseOk3 := extractErrorBoundaryOnError(parseRawProps); parseOk3 {
		parsePropsMap["onError"] = parseOnError
	}
	if parseKeys := extractErrorBoundaryResetKeys(parseRawProps); len(parseKeys) > 0 {
		parsePropsMap["resetKeys"] = parseKeys
	}

	parseChildren := extractErrorBoundaryChildren(parseRawProps)
	return runtime.CreateElementOwned(parseRuntimeBoundary, parsePropsMap, parseChildren...)
}

// extractErrorBoundaryFallback is a core package helper.
func extractErrorBoundaryFallback(parseRawProps any) (Node, bool) {
	if parseRawProps == nil {
		return nil, false
	}
	if parseProps, parseOk := parseRawProps.(map[string]any); parseOk {
		return mapBoundaryNode(parseProps, "fallback", "Fallback")
	}
	parseValue := dereferenceStructValue(parseRawProps)
	if !parseValue.IsValid() {
		return nil, false
	}
	parseField := parseValue.FieldByName("Fallback")
	if !parseField.IsValid() || !parseField.CanInterface() {
		return nil, false
	}
	parseFallback, parseOk2 := parseField.Interface().(*runtime.Element)
	return parseFallback, parseOk2 && parseFallback != nil
}

// extractErrorBoundaryErrorFallback is a core package helper.
func extractErrorBoundaryErrorFallback(parseRawProps any) (func(error, func()) Node, bool) {
	if parseRawProps == nil {
		return nil, false
	}
	if parseProps, parseOk := parseRawProps.(map[string]any); parseOk {
		return mapBoundaryFallback(parseProps, "errorFallback", "ErrorFallback")
	}
	parseValue := dereferenceStructValue(parseRawProps)
	if !parseValue.IsValid() {
		return nil, false
	}
	parseField := parseValue.FieldByName("ErrorFallback")
	if !parseField.IsValid() || !parseField.CanInterface() {
		return nil, false
	}
	parseFallback, parseOk2 := parseField.Interface().(func(error, func()) *runtime.Element)
	return parseFallback, parseOk2 && parseFallback != nil
}

// extractErrorBoundaryOnError is a core package helper.
func extractErrorBoundaryOnError(parseRawProps any) (func(error), bool) {
	if parseRawProps == nil {
		return nil, false
	}
	if parseProps, parseOk := parseRawProps.(map[string]any); parseOk {
		return mapBoundaryOnError(parseProps, "onError", "OnError")
	}
	parseValue := dereferenceStructValue(parseRawProps)
	if !parseValue.IsValid() {
		return nil, false
	}
	parseField := parseValue.FieldByName("OnError")
	if !parseField.IsValid() || !parseField.CanInterface() {
		return nil, false
	}
	parseOnError, parseOk2 := parseField.Interface().(func(error))
	return parseOnError, parseOk2 && parseOnError != nil
}

// extractErrorBoundaryResetKeys is a core package helper.
func extractErrorBoundaryResetKeys(parseRawProps any) []any {
	if parseRawProps == nil {
		return nil
	}
	if parseProps, parseOk := parseRawProps.(map[string]any); parseOk {
		for _, parseKey := range []string{"resetKeys", "ResetKeys"} {
			if parseKeys, parseOk2 := parseProps[parseKey].([]any); parseOk2 {
				return parseKeys
			}
		}
		return nil
	}
	parseValue := dereferenceStructValue(parseRawProps)
	if !parseValue.IsValid() {
		return nil
	}
	parseField := parseValue.FieldByName("ResetKeys")
	if !parseField.IsValid() || !parseField.CanInterface() {
		return nil
	}
	parseKeys2, _ := parseField.Interface().([]any)
	return parseKeys2
}

// extractErrorBoundaryChildren resolves the ErrorBoundary's Child/Children props. The shared
// coalesceChildren helper does the work (one extraction implementation across all boundaries).
func extractErrorBoundaryChildren(parseRawProps any) []any {
	return coalesceChildren(parseRawProps)
}

// dereferenceStructValue is a core package helper.
func dereferenceStructValue(parseRawProps any) reflect.Value {
	parseValue := reflect.ValueOf(parseRawProps)
	for parseValue.IsValid() && parseValue.Kind() == reflect.Pointer {
		if parseValue.IsNil() {
			return reflect.Value{}
		}
		parseValue = parseValue.Elem()
	}
	if !parseValue.IsValid() || parseValue.Kind() != reflect.Struct {
		return reflect.Value{}
	}
	return parseValue
}

// mapBoundaryNode is a core package helper.
func mapBoundaryNode(parseProps map[string]any, parseKeys ...string) (Node, bool) {
	for _, parseKey := range parseKeys {
		parseFallback, parseOk := parseProps[parseKey].(*runtime.Element)
		if parseOk && parseFallback != nil {
			return parseFallback, true
		}
	}
	return nil, false
}

// mapBoundaryFallback is a core package helper.
func mapBoundaryFallback(parseProps map[string]any, parseKeys ...string) (func(error, func()) Node, bool) {
	for _, parseKey := range parseKeys {
		parseFallback, parseOk := parseProps[parseKey].(func(error, func()) *runtime.Element)
		if parseOk && parseFallback != nil {
			return parseFallback, true
		}
	}
	return nil, false
}

// mapBoundaryOnError is a core package helper.
func mapBoundaryOnError(parseProps map[string]any, parseKeys ...string) (func(error), bool) {
	for _, parseKey := range parseKeys {
		parseOnError, parseOk := parseProps[parseKey].(func(error))
		if parseOk && parseOnError != nil {
			return parseOnError, true
		}
	}
	return nil, false
}
