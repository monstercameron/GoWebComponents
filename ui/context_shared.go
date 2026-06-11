package ui

import (
	"reflect"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

type Context[T any] struct {
	descriptor *runtime.ContextDescriptor
	Provider   *ContextProvider[T]
}

// ContextProvider creates a provider element for a Context value.
type ContextProvider[T any] struct {
	providerType *runtime.ContextProviderType
}

// ContextProviderProps defines the value and child content for a context provider.
type ContextProviderProps[T any] struct {
	Value    T
	Child    Node
	Children []Node
}

type contextProviderComponent interface {
	runtimeContextProvider() *runtime.ContextProviderType
}

// runtimeContextProvider is a core package helper.
func (parseProvider *ContextProvider[T]) runtimeContextProvider() *runtime.ContextProviderType {
	if parseProvider == nil {
		return nil
	}
	return parseProvider.providerType
}

// CreateContext creates a typed context with its default value and provider.
func CreateContext[T any](parseDefaultValue T) *Context[T] {
	parseDescriptor := runtime.NewContextDescriptor(parseDefaultValue)
	return &Context[T]{
		descriptor: parseDescriptor,
		Provider:   &ContextProvider[T]{providerType: runtime.NewContextProviderType(parseDescriptor)},
	}
}

// createContextProviderElement is a core package helper.
func createContextProviderElement(parseProvider contextProviderComponent, parseRawProps interface{}) Node {
	parseRuntimeProvider := parseProvider.runtimeContextProvider()
	if parseRuntimeProvider == nil {
		return nil
	}

	parseProps := map[string]interface{}{}
	parseChildren := extractContextProviderChildren(parseRawProps)
	if parseContextValue, parseOk := extractContextProviderValue(parseRawProps); parseOk {
		parseProps["value"] = parseContextValue
	}

	return runtime.CreateElementOwned(parseRuntimeProvider, parseProps, parseChildren...)
}

// extractContextProviderValue is a core package helper.
func extractContextProviderValue(parseRawProps interface{}) (interface{}, bool) {
	if parseRawProps == nil {
		return nil, false
	}

	if parsePropsMap, parseOk := parseRawProps.(map[string]interface{}); parseOk {
		parseContextValue, hasValue := parsePropsMap["value"]
		return parseContextValue, hasValue
	}

	parseReflectedValue := reflect.ValueOf(parseRawProps)
	for parseReflectedValue.IsValid() && parseReflectedValue.Kind() == reflect.Pointer {
		if parseReflectedValue.IsNil() {
			return nil, false
		}
		parseReflectedValue = parseReflectedValue.Elem()
	}

	if !parseReflectedValue.IsValid() {
		return nil, false
	}

	if parseReflectedValue.Kind() != reflect.Struct {
		return parseRawProps, true
	}

	parseValueField := parseReflectedValue.FieldByName("Value")
	if !parseValueField.IsValid() || !parseValueField.CanInterface() {
		return nil, false
	}

	return parseValueField.Interface(), true
}

// extractContextProviderChildren is a core package helper.
func extractContextProviderChildren(parseRawProps interface{}) []interface{} {
	if parseRawProps == nil {
		return nil
	}

	if parsePropsMap, parseOk := parseRawProps.(map[string]interface{}); parseOk {
		parseChildren := make([]interface{}, 0, 2)
		if parseChild, parseOk2 := parsePropsMap["child"].(*runtime.Element); parseOk2 && parseChild != nil {
			parseChildren = append(parseChildren, parseChild)
		}
		if parseChildNodes, parseOk3 := parsePropsMap["children"].([]interface{}); parseOk3 && len(parseChildNodes) > 0 {
			parseChildren = append(parseChildren, parseChildNodes...)
		}
		return parseChildren
	}

	parseReflectedValue := reflect.ValueOf(parseRawProps)
	for parseReflectedValue.IsValid() && parseReflectedValue.Kind() == reflect.Pointer {
		if parseReflectedValue.IsNil() {
			return nil
		}
		parseReflectedValue = parseReflectedValue.Elem()
	}

	if !parseReflectedValue.IsValid() || parseReflectedValue.Kind() != reflect.Struct {
		return nil
	}

	parseChildren2 := make([]interface{}, 0, 2)
	if parseChildField := parseReflectedValue.FieldByName("Child"); parseChildField.IsValid() && parseChildField.CanInterface() {
		if parseChild2, parseOk4 := parseChildField.Interface().(*runtime.Element); parseOk4 && parseChild2 != nil {
			parseChildren2 = append(parseChildren2, parseChild2)
		}
	}
	if parseChildrenField := parseReflectedValue.FieldByName("Children"); parseChildrenField.IsValid() && parseChildrenField.CanInterface() {
		if parseTypedChildren, parseOk5 := parseChildrenField.Interface().([]Node); parseOk5 {
			for _, parseChild3 := range parseTypedChildren {
				if parseChild3 != nil {
					parseChildren2 = append(parseChildren2, parseChild3)
				}
			}
		}
	}

	return parseChildren2
}

// castContextValue is a core package helper.
func castContextValue[T any](parseValue interface{}) T {
	if parseTypedValue, parseOk := parseValue.(T); parseOk {
		return parseTypedValue
	}
	var parseZero T
	return parseZero
}
