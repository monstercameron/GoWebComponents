package ui

import (
	"reflect"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// coalesceChildren flattens a boundary/provider's Child + Children props into one ordered []any.
// It is the single place the Child/Children duality is resolved, so each boundary component
// (ErrorBoundary, context Provider, …) does not re-implement the same ~50-line extraction. It
// accepts both shapes a component can receive props in — the map form from CreateElement and the
// typed-struct form via reflection — and either key casing, returning Child first then Children.
func coalesceChildren(parseRawProps any) []any {
	if parseRawProps == nil {
		return nil
	}
	parseOut := make([]any, 0, 2)

	// Map form (CreateElement passes props as map[string]any).
	if parseMap, parseOk := parseRawProps.(map[string]any); parseOk {
		for _, parseKey := range []string{"child", "Child"} {
			if parseChild, parseOk2 := parseMap[parseKey].(*runtime.Element); parseOk2 && parseChild != nil {
				parseOut = append(parseOut, parseChild)
				break
			}
		}
		for _, parseKey := range []string{"children", "Children"} {
			if parseNodes, parseOk3 := parseMap[parseKey].([]any); parseOk3 && len(parseNodes) > 0 {
				parseOut = append(parseOut, parseNodes...)
				break
			}
			if parseTyped, parseOk4 := parseMap[parseKey].([]Node); parseOk4 && len(parseTyped) > 0 {
				parseOut = appendNonNilNodes(parseOut, parseTyped)
				break
			}
		}
		return parseOut
	}

	// Typed-struct form (a props value passed directly), read by reflection.
	parseValue := dereferenceStructValue(parseRawProps)
	if !parseValue.IsValid() || parseValue.Kind() != reflect.Struct {
		return nil
	}
	if parseField := parseValue.FieldByName("Child"); parseField.IsValid() && parseField.CanInterface() {
		if parseChild, parseOk := parseField.Interface().(*runtime.Element); parseOk && parseChild != nil {
			parseOut = append(parseOut, parseChild)
		}
	}
	if parseField := parseValue.FieldByName("Children"); parseField.IsValid() && parseField.CanInterface() {
		if parseTyped, parseOk := parseField.Interface().([]Node); parseOk {
			parseOut = appendNonNilNodes(parseOut, parseTyped)
		}
	}
	return parseOut
}

// appendNonNilNodes appends every non-nil node to dst.
func appendNonNilNodes(parseDst []any, parseNodes []Node) []any {
	for _, parseNode := range parseNodes {
		if parseNode != nil {
			parseDst = append(parseDst, parseNode)
		}
	}
	return parseDst
}
