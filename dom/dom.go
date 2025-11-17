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

// Fragment groups multiple elements without creating a wrapper DOM node.
// This is equivalent to React's Fragment (<> syntax).
// Fragments allow you to return multiple elements from a component without
// introducing an extra div or other wrapper element.
//
// Fragments are useful for:
//   - Avoiding unnecessary wrapper divs in lists
//   - Keeping semantic HTML structure clean
//   - Avoiding CSS layout issues from wrapper elements
//   - Grouping related elements without a container
//
// Example with Fragment to group items without wrapper:
//
//	func ListItems(props dom.Attrs) *dom.Element {
//	    return dom.Fragment(nil,
//	        dom.Li(nil, dom.Text("Item 1")),
//	        dom.Li(nil, dom.Text("Item 2")),
//	        dom.Li(nil, dom.Text("Item 3")),
//	    )
//	}
//	// Renders as:
//	// <li>Item 1</li>
//	// <li>Item 2</li>
//	// <li>Item 3</li>
//	// NOT wrapped in an extra div or span
//
// Example in a list to avoid key warnings:
//
//	func TodoList(props dom.Attrs) *dom.Element {
//	    items := []string{"Buy milk", "Write code", "Review PR"}
//	    children := make([]interface{}, 0)
//
//	    for i, item := range items {
//	        children = append(children,
//	            dom.Fragment(nil,
//	                dom.H3(nil, dom.Text(fmt.Sprintf("Task %d", i+1))),
//	                dom.P(nil, dom.Text(item)),
//	            ),
//	        )
//	    }
//
//	    return dom.Ul(nil, children...)
//	}
//
// Note: Fragment props are currently ignored. In the future, only the "key" prop
// will be supported for list reconciliation purposes.
func Fragment(props map[string]interface{}, children ...interface{}) *runtime.Element {
	return runtime.CreateElement("FRAGMENT", props, children...)
}
