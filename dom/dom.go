//go:build js && wasm
// +build js,wasm

package dom

import (
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// Re-export core element creation functions from fiber
// This provides a clean API while maintaining backward compatibility during migration

// Text creates a text node element.
// This is the primary way to create text content in the virtual DOM.
//
// Example:
//
//	dom.Text("Hello, World!")
func Text(content string) *runtime.Element {
	return runtime.Text(content)
}

// CreateElement creates a virtual DOM element with the given type, props, and children.
// This is the low-level element creation function used by all HTML element constructors.
//
// The type can be:
//   - A string representing an HTML tag name (e.g., "div", "span")
//   - A component function that returns an Element
//
// Props is a map of HTML attributes and event handlers.
// Children can be Elements, strings (auto-converted to Text nodes), or component functions.
//
// Example:
//
//	dom.CreateElement("div", map[string]interface{}{
//	    "class": "container",
//	    "id": "main",
//	}, dom.Text("Content"))
func CreateElement(typ interface{}, props map[string]interface{}, children ...interface{}) *runtime.Element {
	return runtime.CreateElement(typ, props, children...)
}
