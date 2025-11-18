//go:build js && wasm
// +build js,wasm

package shared

import (
	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/render"
)

// Common type aliases used across examples
type Attrs = dom.Attrs
type Element = render.Element
