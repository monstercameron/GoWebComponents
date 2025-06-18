// ./examples/hot_reload_test.go

//go:build js && wasm
// +build js,wasm

package examples

import (
	. "github.com/monstercameron/GoWebComponents/fiber"
)

func StateTest() func(Attrs) *Element {
	return func(props Attrs) *Element {
		// message, _ := GoUseState("Hello, World!")
		return Div(nil, Text("test"))
	}
}
