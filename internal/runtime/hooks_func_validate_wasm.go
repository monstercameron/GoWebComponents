//go:build js && wasm

package runtime

import (
	"reflect"
	"syscall/js"
)

// isValidHookFunction is a core package helper.
func isValidHookFunction(parseHookFn interface{}) bool {
	switch parseHookFn.(type) {
	case func(),
		func(string),
		func(js.Value),
		func() error,
		func(js.Value) error,
		func(GoEvent),
		func(GoEvent) error:
		return true
	default:
		parseHookFnType := reflect.TypeOf(parseHookFn)
		return parseHookFnType != nil && parseHookFnType.Kind() == reflect.Func
	}
}
