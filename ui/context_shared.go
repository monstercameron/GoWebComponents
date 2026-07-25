package ui

import (
	"reflect"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
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
func createContextProviderElement(parseProvider contextProviderComponent, parseRawProps any) Node {
	parseRuntimeProvider := parseProvider.runtimeContextProvider()
	if parseRuntimeProvider == nil {
		return nil
	}

	parseProps := map[string]any{}
	parseChildren := extractContextProviderChildren(parseRawProps)
	if parseContextValue, parseOk := extractContextProviderValue(parseRawProps); parseOk {
		parseProps["value"] = parseContextValue
	}

	return runtime.CreateElementOwned(parseRuntimeProvider, parseProps, parseChildren...)
}

// extractContextProviderValue is a core package helper.
func extractContextProviderValue(parseRawProps any) (any, bool) {
	if parseRawProps == nil {
		return nil, false
	}

	if parsePropsMap, parseOk := parseRawProps.(map[string]any); parseOk {
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

// extractContextProviderChildren resolves the context Provider's Child/Children props via the
// shared coalesceChildren helper (one extraction implementation across all boundaries).
func extractContextProviderChildren(parseRawProps any) []any {
	return coalesceChildren(parseRawProps)
}

// castContextValue is a core package helper.
func castContextValue[T any](parseValue any) T {
	if parseTypedValue, parseOk := parseValue.(T); parseOk {
		return parseTypedValue
	}
	var parseZero T
	return parseZero
}
