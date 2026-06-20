//go:build js && wasm

package ui

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/internal/platform/jsdom"
)

// Value returns the underlying js.Value for the referenced element, or js.Null()
// when the ref is not currently mounted. wasm build only.
func (parseR DOMRef) Value() js.Value {
	if parseWASM, parseOk := parseR.Node().(*jsdom.WASMDOMNode); parseOk && parseWASM != nil {
		return parseWASM.Value()
	}
	return js.Null()
}

// Focus focuses the referenced element if it is mounted. No-op otherwise.
func (parseR DOMRef) Focus() {
	if parseValue := parseR.Value(); parseValue.Truthy() {
		parseValue.Call("focus")
	}
}
