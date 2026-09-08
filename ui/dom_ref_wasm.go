//go:build js && wasm

package ui

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v6/internal/platform/jsdom"
)

// Value returns the underlying js.Value for the referenced element, or js.Null()
// when the ref is not currently mounted. wasm build only. (Focus is defined
// cross-build in dom_ref.go via the runtime.Focuser capability.)
func (parseR DOMRef) Value() js.Value {
	if parseWASM, parseOk := parseR.Node().(*jsdom.WASMDOMNode); parseOk && parseWASM != nil {
		return parseWASM.Value()
	}
	return js.Null()
}
