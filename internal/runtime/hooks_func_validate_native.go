//go:build !js || !wasm
// +build !js !wasm

package runtime

import "reflect"

func isValidHookFunction(fn interface{}) bool {
	switch fn.(type) {
	case func(), func(string), func() error:
		return true
	default:
		fnType := reflect.TypeOf(fn)
		return fnType != nil && fnType.Kind() == reflect.Func
	}
}
