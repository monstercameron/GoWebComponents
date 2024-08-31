package components

import (
	"syscall/js"
)

var (
	// Map to hold bound functions
	boundFunctions = make(map[string]js.Func)
)

func Function(c *Component, id string, fn func(js.Value)) string {
	js.Global().Set(id, js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			fn(args[0])
		}
		return nil
	}))

	return id + "(event)"
}