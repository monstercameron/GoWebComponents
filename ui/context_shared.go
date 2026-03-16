package ui

import (
	"reflect"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

type Context[T any] struct {
	descriptor *runtime.ContextDescriptor
	Provider   *ContextProvider[T]
}

type ContextProvider[T any] struct {
	providerType *runtime.ContextProviderType
}

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
	if value, ok := extractContextProviderValue(rawProps); ok {
		props["value"] = value
	}

	return runtime.CreateElement(runtimeProvider, props, children...)
}

func extractContextProviderValue(rawProps interface{}) (interface{}, bool) {
	if rawProps == nil {
		return nil, false
	}

	if props, ok := rawProps.(map[string]interface{}); ok {
		value, hasValue := props["value"]
		return value, hasValue
	}

	value := reflect.ValueOf(rawProps)
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, false
		}
		value = value.Elem()
	}

	if !value.IsValid() {
		return nil, false
	}

	if value.Kind() != reflect.Struct {
		return rawProps, true
	}

	field := value.FieldByName("Value")
	if !field.IsValid() || !field.CanInterface() {
		return nil, false
	}

	return field.Interface(), true
}

func extractContextProviderChildren(rawProps interface{}) []interface{} {
	if rawProps == nil {
		return nil
	}

	if props, ok := rawProps.(map[string]interface{}); ok {
		children := make([]interface{}, 0, 2)
		if child, ok := props["child"].(*runtime.Element); ok && child != nil {
			children = append(children, child)
		}
		if child, ok := props["children"].([]interface{}); ok && len(child) > 0 {
			children = append(children, child...)
		}
		return children
	}

	value := reflect.ValueOf(rawProps)
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}

	if !value.IsValid() || value.Kind() != reflect.Struct {
		return nil
	}

	children := make([]interface{}, 0, 2)
	if childField := value.FieldByName("Child"); childField.IsValid() && childField.CanInterface() {
		if child, ok := childField.Interface().(*runtime.Element); ok && child != nil {
			children = append(children, child)
		}
	}
	if childrenField := value.FieldByName("Children"); childrenField.IsValid() && childrenField.CanInterface() {
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
