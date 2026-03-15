//go:build js && wasm
// +build js,wasm

package runtime

import (
	"reflect"
	"syscall/js"
)

func isValidHookFunction(fn interface{}) bool {
	switch fn.(type) {
	case func(),
		func(string),
		func(js.Value),
		func() error,
		func(js.Value) error,
		func(GoEvent),
		func(GoEvent) error:
		return true
	default:
		fnType := reflect.TypeOf(fn)
		return fnType != nil && fnType.Kind() == reflect.Func
	}
}
