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

func (provider *ContextProvider[T]) runtimeContextProvider() *runtime.ContextProviderType {
	if provider == nil {
		return nil
	}
	return provider.providerType
}

// CreateContext creates a typed context with its default value and provider.
func CreateContext[T any](defaultValue T) *Context[T] {
	descriptor := runtime.NewContextDescriptor(defaultValue)
	return &Context[T]{
		descriptor: descriptor,
		Provider:   &ContextProvider[T]{providerType: runtime.NewContextProviderType(descriptor)},
	}
}

func createContextProviderElement(provider contextProviderComponent, rawProps interface{}) Node {
	runtimeProvider := provider.runtimeContextProvider()
	if runtimeProvider == nil {
		return nil
	}

	props := map[string]interface{}{}
	children := extractContextProviderChildren(rawProps)
	if contextValue, ok := extractContextProviderValue(rawProps); ok {
		props["value"] = contextValue
	}

	return runtime.CreateElement(runtimeProvider, props, children...)
}

func extractContextProviderValue(rawProps interface{}) (interface{}, bool) {
	if rawProps == nil {
		return nil, false
	}

	if propsMap, ok := rawProps.(map[string]interface{}); ok {
		contextValue, hasValue := propsMap["value"]
		return contextValue, hasValue
	}

	reflectedValue := reflect.ValueOf(rawProps)
	for reflectedValue.IsValid() && reflectedValue.Kind() == reflect.Pointer {
		if reflectedValue.IsNil() {
			return nil, false
		}
		reflectedValue = reflectedValue.Elem()
	}

	if !reflectedValue.IsValid() {
		return nil, false
	}

	if reflectedValue.Kind() != reflect.Struct {
		return rawProps, true
	}

	valueField := reflectedValue.FieldByName("Value")
	if !valueField.IsValid() || !valueField.CanInterface() {
		return nil, false
	}

	return valueField.Interface(), true
}

func extractContextProviderChildren(rawProps interface{}) []interface{} {
	if rawProps == nil {
		return nil
	}

	if propsMap, ok := rawProps.(map[string]interface{}); ok {
		children := make([]interface{}, 0, 2)
		if child, ok := propsMap["child"].(*runtime.Element); ok && child != nil {
			children = append(children, child)
		}
		if childNodes, ok := propsMap["children"].([]interface{}); ok && len(childNodes) > 0 {
			children = append(children, childNodes...)
		}
		return children
	}

	reflectedValue := reflect.ValueOf(rawProps)
	for reflectedValue.IsValid() && reflectedValue.Kind() == reflect.Pointer {
		if reflectedValue.IsNil() {
			return nil
		}
		reflectedValue = reflectedValue.Elem()
	}

	if !reflectedValue.IsValid() || reflectedValue.Kind() != reflect.Struct {
		return nil
	}

	children := make([]interface{}, 0, 2)
	if childField := reflectedValue.FieldByName("Child"); childField.IsValid() && childField.CanInterface() {
		if child, ok := childField.Interface().(*runtime.Element); ok && child != nil {
			children = append(children, child)
		}
	}
	if childrenField := reflectedValue.FieldByName("Children"); childrenField.IsValid() && childrenField.CanInterface() {
		if typedChildren, ok := childrenField.Interface().([]Node); ok {
			for _, child := range typedChildren {
				if child != nil {
					children = append(children, child)
				}
			}
		}
	}

	return children
}

func castContextValue[T any](value interface{}) T {
	if typedValue, ok := value.(T); ok {
		return typedValue
	}
	var zero T
	return zero
}
