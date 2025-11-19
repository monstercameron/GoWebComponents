//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/render"
)

// Type aliases for convenience - declared once for the entire package
type Attrs = dom.Attrs
type Element = render.Element
