//go:build js && wasm
// +build js,wasm

package example

import (
	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/render"
)

// Type aliases for convenience
type Attrs = dom.Attrs
type Element = render.Element
