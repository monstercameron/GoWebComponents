package ui

import (
	"reflect"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

func createErrorBoundaryElement(boundary runtimeErrorBoundaryComponent, rawProps interface{}) Node {
	runtimeBoundary := boundary.runtimeErrorBoundary()
	if runtimeBoundary == nil {
		return nil
	}

	propsMap := map[string]interface{}{}
	if fallback, ok := extractErrorBoundaryFallback(rawProps); ok {
		propsMap["fallback"] = fallback
	}
	if fallback, ok := extractErrorBoundaryErrorFallback(rawProps); ok {
		propsMap["errorFallback"] = fallback
	}
	if onError, ok := extractErrorBoundaryOnError(rawProps); ok {
		propsMap["onError"] = onError
	}
	if keys := extractErrorBoundaryResetKeys(rawProps); len(keys) > 0 {
		propsMap["resetKeys"] = keys
	}

	children := extractErrorBoundaryChildren(rawProps)
	return runtime.CreateElement(runtimeBoundary, propsMap, children...)
}

func extractErrorBoundaryFallback(rawProps interface{}) (Node, bool) {
	if rawProps == nil {
		return nil, false
	}
	if props, ok := rawProps.(map[string]interface{}); ok {
		return mapBoundaryNode(props, "fallback", "Fallback")
	}
	value := dereferenceStructValue(rawProps)
	if !value.IsValid() {
		return nil, false
	}
	field := value.FieldByName("Fallback")
	if !field.IsValid() || !field.CanInterface() {
		return nil, false
	}
	fallback, ok := field.Interface().(*runtime.Element)
	return fallback, ok && fallback != nil
}

func extractErrorBoundaryErrorFallback(rawProps interface{}) (func(error, func()) Node, bool) {
	if rawProps == nil {
		return nil, false
	}
	if props, ok := rawProps.(map[string]interface{}); ok {
		return mapBoundaryFallback(props, "errorFallback", "ErrorFallback")
	}
	value := dereferenceStructValue(rawProps)
	if !value.IsValid() {
		return nil, false
	}
	field := value.FieldByName("ErrorFallback")
	if !field.IsValid() || !field.CanInterface() {
		return nil, false
	}
	fallback, ok := field.Interface().(func(error, func()) *runtime.Element)
	return fallback, ok && fallback != nil
}

func extractErrorBoundaryOnError(rawProps interface{}) (func(error), bool) {
	if rawProps == nil {
		return nil, false
	}
	if props, ok := rawProps.(map[string]interface{}); ok {
		return mapBoundaryOnError(props, "onError", "OnError")
	}
	value := dereferenceStructValue(rawProps)
	if !value.IsValid() {
		return nil, false
	}
	field := value.FieldByName("OnError")
	if !field.IsValid() || !field.CanInterface() {
		return nil, false
	}
	onError, ok := field.Interface().(func(error))
	return onError, ok && onError != nil
}

func extractErrorBoundaryResetKeys(rawProps interface{}) []interface{} {
	if rawProps == nil {
		return nil
	}
	if props, ok := rawProps.(map[string]interface{}); ok {
		for _, key := range []string{"resetKeys", "ResetKeys"} {
			if keys, ok := props[key].([]interface{}); ok {
				return keys
			}
		}
		return nil
	}
	value := dereferenceStructValue(rawProps)
	if !value.IsValid() {
		return nil
	}
	field := value.FieldByName("ResetKeys")
	if !field.IsValid() || !field.CanInterface() {
		return nil
	}
	keys, _ := field.Interface().([]interface{})
	return keys
}

func extractErrorBoundaryChildren(rawProps interface{}) []interface{} {
	if rawProps == nil {
		return nil
	}
	if props, ok := rawProps.(map[string]interface{}); ok {
		children := make([]interface{}, 0, 2)
		for _, key := range []string{"child", "Child"} {
			if child, ok := props[key].(*runtime.Element); ok && child != nil {
				children = append(children, child)
				break
			}
		}
		for _, key := range []string{"children", "Children"} {
			if child, ok := props[key].([]interface{}); ok && len(child) > 0 {
				children = append(children, child...)
				break
			}
			if typedChildren, ok := props[key].([]Node); ok && len(typedChildren) > 0 {
				for _, child := range typedChildren {
					if child != nil {
						children = append(children, child)
					}
				}
				break
			}
		}
		return children
	}

	value := dereferenceStructValue(rawProps)
	if !value.IsValid() {
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

func dereferenceStructValue(rawProps interface{}) reflect.Value {
	value := reflect.ValueOf(rawProps)
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return reflect.Value{}
	}
	return value
}

func mapBoundaryNode(props map[string]interface{}, keys ...string) (Node, bool) {
	for _, key := range keys {
		fallback, ok := props[key].(*runtime.Element)
		if ok && fallback != nil {
			return fallback, true
		}
	}
	return nil, false
}

func mapBoundaryFallback(props map[string]interface{}, keys ...string) (func(error, func()) Node, bool) {
	for _, key := range keys {
		fallback, ok := props[key].(func(error, func()) *runtime.Element)
		if ok && fallback != nil {
			return fallback, true
		}
	}
	return nil, false
}

func mapBoundaryOnError(props map[string]interface{}, keys ...string) (func(error), bool) {
	for _, key := range keys {
		onError, ok := props[key].(func(error))
		if ok && onError != nil {
			return onError, true
		}
	}
	return nil, false
}
