//go:build js && wasm
// +build js,wasm

package runtime

import (
	"fmt"
	"reflect"
	"syscall/js"
)

// Global wrapper functions that provide simplified API for public packages
// These match the old fiber package API

// GoUseStateGlobal wraps GoUseState with global fiber context
func GoUseStateGlobal[T any](initialValue T) (func() T, func(interface{})) {
	rt := GetGlobalRuntime()
	return GoUseState(rt, initialValue)
}

// GoUseEffectGlobal wraps GoUseEffect
func GoUseEffectGlobal(effect func() func(), deps ...interface{}) {
	// GoUseEffect doesn't need Runtime, it works with current fiber
	GoUseEffect(effect, deps...)
}

// GoUseMemoGlobal wraps GoUseMemo
func GoUseMemoGlobal(compute func() interface{}, deps ...interface{}) interface{} {
	// GoUseMemo doesn't need Runtime, it works with current fiber
	return GoUseMemo(compute, deps...)
}

// GoUseCallbackGlobal wraps GoUseCallback
func GoUseCallbackGlobal(fn interface{}, deps ...interface{}) interface{} {
	// GoUseCallback doesn't need Runtime, it works with current fiber
	return GoUseCallback(fn, deps...)
}

// GoUseRefGlobal wraps GoUseRef
func GoUseRefGlobal(initialValue interface{}) *RefValue {
	// GoUseRef doesn't need Runtime, it works with current fiber
	return GoUseRef(initialValue)
}

// GoUseIdGlobal wraps GoUseId
func GoUseIdGlobal() string {
	// GoUseId doesn't need Runtime, it works with current fiber
	return GoUseId()
}

// GoUseFetchGlobal wraps GoUseFetch with global fiber context
func GoUseFetchGlobal(url string, options ...interface{}) (func() FetchState, func()) {
	// GoUseFetch doesn't need Runtime, it works with current fiber
	return GoUseFetch(url, options...)
}

// GoUseFuncGlobal wraps GoUseFunc with WASM event handler wrapping
func GoUseFuncGlobal(fn interface{}) interface{} {
	// First, call the core GoUseFunc to validate and store the function
	storedFn := GoUseFunc(fn)

	// Then wrap it based on the function signature
	fnType := reflect.TypeOf(storedFn)
	if fnType == nil || fnType.Kind() != reflect.Func {
		panic("GoUseFuncGlobal: invalid function")
	}

	numIn := fnType.NumIn()
	numOut := fnType.NumOut()

	// Get the input type name if there's an input parameter
	var inTypeName string
	if numIn > 0 {
		inTypeName = fnType.In(0).String()
	}

	// Create the appropriate wrapper based on function signature
	switch {
	// func()
	case numIn == 0 && numOut == 0:
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			storedFn.(func())()
			return nil
		})

	// func() error
	case numIn == 0 && numOut == 1:
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			result := reflect.ValueOf(storedFn).Call([]reflect.Value{})
			if len(result) > 0 && !result[0].IsNil() {
				fmt.Printf("Handler error: %v\n", result[0].Interface())
			}
			return nil
		})

	// func(string) - input handler
	case numIn == 1 && fnType.In(0).Kind() == reflect.String:
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				value := args[0].Get("target").Get("value").String()
				storedFn.(func(string))(value)
			}
			return nil
		})

	// func(GoEvent) - GoEvent handler (check by type name, including aliases)
	case numIn == 1 && (inTypeName == "github.com/monstercameron/GoWebComponents/internal/runtime.GoEvent" || inTypeName == "github.com/monstercameron/GoWebComponents/dom.GoEvent" || inTypeName == "GoEvent" || inTypeName == "runtime.GoEvent"):
		// Type assert to func(GoEvent)
		handler := storedFn.(func(GoEvent))
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				handler(NewGoEvent(args[0]))
			}
			return nil
		})

	// func(GoEvent) error
	case numIn == 1 && numOut == 1 && (inTypeName == "github.com/monstercameron/GoWebComponents/internal/runtime.GoEvent" || inTypeName == "github.com/monstercameron/GoWebComponents/dom.GoEvent" || inTypeName == "GoEvent" || inTypeName == "runtime.GoEvent"):
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				result := reflect.ValueOf(storedFn).Call([]reflect.Value{reflect.ValueOf(NewGoEvent(args[0]))})
				if len(result) > 0 && !result[0].IsNil() {
					fmt.Printf("Handler error: %v\n", result[0].Interface())
				}
			}
			return nil
		})

	// func(js.Value) - event handler
	case numIn == 1 && inTypeName == "syscall/js.Value":
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				storedFn.(func(js.Value))(args[0])
			}
			return nil
		})

	// func(js.Value) error - event handler with error
	case numIn == 1 && inTypeName == "syscall/js.Value" && numOut == 1:
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				result := reflect.ValueOf(storedFn).Call([]reflect.Value{reflect.ValueOf(args[0])})
				if len(result) > 0 && !result[0].IsNil() {
					fmt.Printf("Handler error: %v\n", result[0].Interface())
				}
			}
			return nil
		})

	default:
		panic(fmt.Sprintf("GoUseFuncGlobal: unsupported function signature: %v", fnType))
	}
}

// GoUseAtomGlobal wraps GoUseAtom with global runtime
func GoUseAtomGlobal[T any](id string, initialValue T) (func() T, func(T)) {
	rt := GetGlobalRuntime()
	get, set := GoUseAtom(rt, id, initialValue)
	// Convert internal setter func(interface{}) to typed setter func(T)
	typedSet := func(v T) {
		set(v)
	}
	return get, typedSet
}

// Text creates a text node
func Text(content string) *Element {
	return &Element{
		Type:     "TEXT_ELEMENT",
		Props:    map[string]interface{}{"nodeValue": content},
		Children: []interface{}{},
	}
}
