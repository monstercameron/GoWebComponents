//go:build js && wasm
// +build js,wasm

package shared

import "github.com/monstercameron/GoWebComponents/internal/runtime"

// Common type aliases used across examples
type Attrs = map[string]interface{}
type Element = runtime.Element
