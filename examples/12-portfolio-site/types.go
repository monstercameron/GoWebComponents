//go:build js && wasm
// +build js,wasm

package main

import "github.com/monstercameron/GoWebComponents/internal/runtime"

// Type aliases for convenience - declared once for the entire package
type Attrs = map[string]interface{}
type Element = runtime.Element
