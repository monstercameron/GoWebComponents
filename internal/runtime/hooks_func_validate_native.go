//go:build !js || !wasm

package runtime

import "reflect"

// isValidHookFunction is a core package helper.
func isValidHookFunction(parseHookFn any) bool {
	switch parseHookFn.(type) {
	case func(), func(string), func() error:
		return true
	default:
		parseHookFnType := reflect.TypeOf(parseHookFn)
		return parseHookFnType != nil && parseHookFnType.Kind() == reflect.Func
	}
}
