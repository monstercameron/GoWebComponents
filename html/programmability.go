package html

import (
	"fmt"
	"syscall/js"
)

// WasmFunc wraps a Go function to be callable from JavaScript and automatically sets it as a global function.
func WasmFunc(name string, f func()) {
	js.Global().Set(name, js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		f()
		fmt.Println("Wasm function called:", name) // Added console log
		return nil
	}))
}
