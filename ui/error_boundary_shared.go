package ui

import (
	"reflect"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// createErrorBoundaryElement is a core package helper.
func createErrorBoundaryElement(parseBoundary runtimeErrorBoundaryComponent, parseRawProps interface{}) Node {
	parseRuntimeBoundary := parseBoundary.runtimeErrorBoundary()
	if parseRuntimeBoundary == nil {
		return nil
	}

	parsePropsMap := map[string]interface{}{}
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
func extractErrorBoundaryFallback(parseRawProps interface{}) (Node, bool) {
	if parseRawProps == nil {
		return nil, false
	}
	if parseProps, parseOk := parseRawProps.(map[string]interface{}); parseOk {
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
func extractErrorBoundaryErrorFallback(parseRawProps interface{}) (func(error, func()) Node, bool) {
	if parseRawProps == nil {
		return nil, false
	}
	if parseProps, parseOk := parseRawProps.(map[string]interface{}); parseOk {
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
func extractErrorBoundaryOnError(parseRawProps interface{}) (func(error), bool) {
	if parseRawProps == nil {
		return nil, false
	}
	if parseProps, parseOk := parseRawProps.(map[string]interface{}); parseOk {
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
func extractErrorBoundaryResetKeys(parseRawProps interface{}) []interface{} {
	if parseRawProps == nil {
		return nil
	}
	if parseProps, parseOk := parseRawProps.(map[string]interface{}); parseOk {
		for _, parseKey := range []string{"resetKeys", "ResetKeys"} {
			if parseKeys, parseOk2 := parseProps[parseKey].([]interface{}); parseOk2 {
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
	parseKeys2, _ := parseField.Interface().([]interface{})
	return parseKeys2
}

// extractErrorBoundaryChildren is a core package helper.
func extractErrorBoundaryChildren(parseRawProps interface{}) []interface{} {
	if parseRawProps == nil {
		return nil
	}
	if parseProps, parseOk := parseRawProps.(map[string]interface{}); parseOk {
		parseChildren := make([]interface{}, 0, 2)
		for _, parseKey := range []string{"child", "Child"} {
			if parseChild, parseOk2 := parseProps[parseKey].(*runtime.Element); parseOk2 && parseChild != nil {
				parseChildren = append(parseChildren, parseChild)
				break
			}
		}
		for _, parseKey2 := range []string{"children", "Children"} {
			if parseChild2, parseOk3 := parseProps[parseKey2].([]interface{}); parseOk3 && len(parseChild2) > 0 {
				parseChildren = append(parseChildren, parseChild2...)
				break
			}
			if parseTypedChildren, parseOk4 := parseProps[parseKey2].([]Node); parseOk4 && len(parseTypedChildren) > 0 {
				for _, parseChild3 := range parseTypedChildren {
					if parseChild3 != nil {
						parseChildren = append(parseChildren, parseChild3)
					}
				}
				break
			}
		}
		return parseChildren
	}

	parseValue := dereferenceStructValue(parseRawProps)
	if !parseValue.IsValid() {
		return nil
	}

	parseChildren2 := make([]interface{}, 0, 2)
	if parseChildField := parseValue.FieldByName("Child"); parseChildField.IsValid() && parseChildField.CanInterface() {
		if parseChild4, parseOk5 := parseChildField.Interface().(*runtime.Element); parseOk5 && parseChild4 != nil {
			parseChildren2 = append(parseChildren2, parseChild4)
		}
	}
	if parseChildrenField := parseValue.FieldByName("Children"); parseChildrenField.IsValid() && parseChildrenField.CanInterface() {
		if parseTypedChildren2, parseOk6 := parseChildrenField.Interface().([]Node); parseOk6 {
			for _, parseChild5 := range parseTypedChildren2 {
				if parseChild5 != nil {
					parseChildren2 = append(parseChildren2, parseChild5)
				}
			}
		}
	}

	return parseChildren2
}

// dereferenceStructValue is a core package helper.
func dereferenceStructValue(parseRawProps interface{}) reflect.Value {
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
func mapBoundaryNode(parseProps map[string]interface{}, parseKeys ...string) (Node, bool) {
	for _, parseKey := range parseKeys {
		parseFallback, parseOk := parseProps[parseKey].(*runtime.Element)
		if parseOk && parseFallback != nil {
			return parseFallback, true
		}
	}
	return nil, false
}

// mapBoundaryFallback is a core package helper.
func mapBoundaryFallback(parseProps map[string]interface{}, parseKeys ...string) (func(error, func()) Node, bool) {
	for _, parseKey := range parseKeys {
		parseFallback, parseOk := parseProps[parseKey].(func(error, func()) *runtime.Element)
		if parseOk && parseFallback != nil {
			return parseFallback, true
		}
	}
	return nil, false
}

// mapBoundaryOnError is a core package helper.
func mapBoundaryOnError(parseProps map[string]interface{}, parseKeys ...string) (func(error), bool) {
	for _, parseKey := range parseKeys {
		parseOnError, parseOk := parseProps[parseKey].(func(error))
		if parseOk && parseOnError != nil {
			return parseOnError, true
		}
	}
	return nil, false
}
