//go:build js && wasm
// +build js,wasm

package fiber

import (
	"strings"
	"sync/atomic"
	"syscall/js"
)

// createElement constructs an Element with optimized allocations
func createElement(typ interface{}, props map[string]interface{}, children ...interface{}) *Element {
	// Get element from pool with safe type assertion
	poolElem := elementPool.Get()
	elem, ok := poolElem.(*Element)
	if !ok {
		// This should never happen if pool is properly initialized, but handle gracefully
		debugf("DOM", "🚨 createElement: elementPool returned unexpected type %T, creating new Element\n", poolElem)
		elem = &Element{
			Props: make(map[string]interface{}),
		}
	} else {
		// Decrement pool size counter when retrieving from pool
		atomic.AddInt32(&poolSizes.element, -1)
	}

	// Reset the element
	elem.Type = typ

	// Process children to support both component references and return values
	processedChildren := make([]interface{}, 0, len(children))

	for _, child := range children {
		if child == nil {
			continue
		}

		// Check if child is a component function reference (map[string]interface{} signature)
		if componentFunc, ok := child.(func(map[string]interface{}) *Element); ok {
			// Call the component function with nil props
			result := componentFunc(nil)
			if result != nil {
				processedChildren = append(processedChildren, result)
			}
		} else if componentFunc, ok := child.(func(Attrs) *Element); ok {
			// Check if child is a component function reference (Attrs signature)
			// Call the component function with nil props
			result := componentFunc(nil)
			if result != nil {
				processedChildren = append(processedChildren, result)
			}
		} else {
			// Child is already processed (Element, Text, etc.)
			processedChildren = append(processedChildren, child)
		}
	}

	elem.Children = processedChildren

	// Handle props efficiently - optimize map clearing for large maps
	if props != nil {
		// Performance optimization: for large maps, allocating new is faster than clearing
		const clearThreshold = 8 // Threshold where new allocation becomes more efficient
		if len(elem.Props) > clearThreshold {
			// Replace with new map - faster for large maps
			elem.Props = make(map[string]interface{}, len(props)+1) // +1 for children
		} else {
			// Clear existing props map for small maps
			for k := range elem.Props {
				delete(elem.Props, k)
			}
		}
		// Copy props (map is allocated or cleared)
		for k, v := range props {
			elem.Props[k] = v
		}
	} else {
		// Clear props if none provided - same optimization applies
		if len(elem.Props) > clearThreshold {
			// Replace with new map for large props
			elem.Props = make(map[string]interface{}, 1) // Just for children
		} else {
			// Clear small maps
			for k := range elem.Props {
				delete(elem.Props, k)
			}
		}
	}

	// Set children in props
	if len(processedChildren) > 0 {
		elem.Props["children"] = processedChildren
	} else {
		// Use a shared empty slice to avoid allocations while maintaining type safety
		elem.Props["children"] = emptyChildren
	}

	return elem
}

// Text creates a text node
func Text(content string) *Element {
	return createElement("TEXT_ELEMENT", map[string]interface{}{
		"nodeValue": content,
	})
}

// createDom creates a DOM node from a fiber
func createDom(fiber *Fiber) js.Value {
	// fmt.Printf("createDom: Creating DOM for fiber type %v\n", fiber.typeOf)
	var dom js.Value
	switch t := fiber.typeOf.(type) {
	case string:
		if t == "TEXT_ELEMENT" {
			dom = js.Global().Get("document").Call("createTextNode", fiber.props["nodeValue"])
		} else {
			dom = js.Global().Get("document").Call("createElement", t)
		}
	default:
		// Function components do not create DOM nodes here
		// fmt.Println("createDom: Function component, no DOM node created")
		return js.Value{}
	}

	// Add event listeners and properties
	for name, value := range fiber.props {
		if name == "children" {
			continue
		}
		if name == "dangerouslySetInnerHTML" {
			// Set innerHTML directly
			if htmlMap, ok := value.(map[string]string); ok {
				if htmlContent, htmlOk := htmlMap["__html"]; htmlOk {
					// fmt.Println("createDom: Setting innerHTML")
					dom.Set("innerHTML", htmlContent)
				} else {
					debugf("DOM", "🚨 createDom: dangerouslySetInnerHTML missing __html key\n")
				}
			} else {
				debugf("DOM", "🚨 createDom: dangerouslySetInnerHTML is not map[string]string, got %T\n", value)
			}
			continue
		}
		if len(name) > 2 && name[:2] == "on" {
			// Event handlers
			eventType := strings.ToLower(name[2:]) // Convert event type to lowercase
			// fmt.Printf("createDom: Adding event listener for %s\n", eventType)

			// Ensure the value is of the correct function type
			eventHandler, ok := value.(js.Func)
			if !ok {
				// fmt.Printf("createDom: Event handler for %s is not a js.Func\n", eventType)
				continue
			}

			dom.Call("addEventListener", eventType, eventHandler)
			debugf("DOM", "🔗 createDom: added event listener %s to DOM element\n", eventType)
			continue
		}
		if name == "class" {
			// Handle 'class' attribute using setAttribute
			// fmt.Printf("createDom: Setting attribute 'class' to '%v'\n", value)
			dom.Call("setAttribute", "class", value)
			continue
		}
		if name == "style" {
			// Support both string and map styles
			switch v := value.(type) {
			case string:
				dom.Set("style", v)
			case map[string]string:
				// If previous style was a string, clear it completely first
				if _, wasString := fiber.props["style"].(string); wasString {
					dom.Set("style", "")
				}
				styleObj := dom.Get("style")
				// Remove styles that no longer exist
				if oldStyleMap, okOld := fiber.props["style"].(map[string]string); okOld {
					for k := range oldStyleMap {
						if _, exists := v[k]; !exists {
							styleObj.Call("removeProperty", k)
						}
					}
				}
				for k, val := range v {
					styleObj.Call("setProperty", k, val)
				}
			default:
				debugf("DOM", "🚨 createDom: style must be string or map[string]string, got %T\n", value)
			}
			continue
		}
		// Set other properties directly
		// fmt.Printf("createDom: Setting property '%s' to '%v'\n", name, value)
		dom.Set(name, value)
	}
	return dom
}

// updateDom updates DOM properties between old and new props
func updateDom(dom js.Value, oldProps, newProps map[string]interface{}) {
	// Fast path: check if maps are equal first
	if len(oldProps) == 0 && len(newProps) == 0 {
		return
	}

	// Ultra-fast path: pointer equality check
	// if unsafe.Pointer(&oldProps) == unsafe.Pointer(&newProps) {
	//	return
	// }

	// 1. Remove old or changed event listeners (optimized)
	for name, oldValue := range oldProps {
		// Branch optimization: check first character before string operations
		if len(name) > 2 && name[0] == 'o' && name[1] == 'n' {
			// Only remove if not in new props or value changed
			if newValue, exists := newProps[name]; !exists || !fastEqual(oldValue, newValue) {
				eventType := strings.ToLower(name[2:])
				if oldHandler, ok := oldValue.(js.Func); ok {
					// Use defer for cleaner resource management
					func() {
						defer oldHandler.Release() // CRITICAL: Always release js.Func to prevent memory leak
						dom.Call("removeEventListener", eventType, oldHandler)
						debugf("DOM", "🧹 updateDom: removed and released old event listener %s from DOM element\n", eventType)
					}()
				} else {
					debugf("DOM", "🚨 updateDom: old event handler %s is not js.Func, got %T\n", name, oldValue)
				}
			}
		} else if newProps[name] == nil && name != "children" {
			// Remove properties that no longer exist, excluding event listeners and children
			dom.Set(name, js.Undefined())
		}
	}

	// 2. Add new or changed properties and event listeners (optimized)
	for name, value := range newProps {
		// Skip common exclusions first (most frequent check)
		if name == "children" {
			continue
		}

		// Skip if value hasn't changed
		if oldValue, exists := oldProps[name]; exists && fastEqual(oldValue, value) {
			continue
		}

		// Branch optimization: inline checks for most common patterns
		switch name {
		case "class":
			dom.Call("setAttribute", "class", value)
		case "style":
			// Handle string vs map[string]string styles
			if styleStr, ok := value.(string); ok {
				dom.Set("style", styleStr)
			} else if styleMap, ok := value.(map[string]string); ok {
				// If previous style was a string, clear it completely first
				if _, wasString := oldProps["style"].(string); wasString {
					dom.Set("style", "")
				}
				styleObj := dom.Get("style")
				// Remove styles that no longer exist
				if oldStyleMap, okOld := oldProps["style"].(map[string]string); okOld {
					for k := range oldStyleMap {
						if _, exists := styleMap[k]; !exists {
							styleObj.Call("removeProperty", k)
						}
					}
				}
				for k, val := range styleMap {
					styleObj.Call("setProperty", k, val)
				}
			} else {
				debugf("DOM", "🚨 updateDom: style must be string or map[string]string, got %T\n", value)
			}
		case "id":
			dom.Set("id", value)
		case "value":
			dom.Set("value", value)
		case "dangerouslySetInnerHTML":
			if htmlMap, ok := value.(map[string]string); ok {
				if htmlContent, htmlOk := htmlMap["__html"]; htmlOk {
					dom.Set("innerHTML", htmlContent)
				} else {
					debugf("DOM", "🚨 updateDom: dangerouslySetInnerHTML missing __html key\n")
				}
			} else {
				debugf("DOM", "🚨 updateDom: dangerouslySetInnerHTML is not map[string]string, got %T\n", value)
			}
		default:
			// Check for event handlers (less common)
			if len(name) > 2 && name[0] == 'o' && name[1] == 'n' {
				eventType := strings.ToLower(name[2:])
				if eventHandler, ok := value.(js.Func); ok {
					dom.Call("addEventListener", eventType, eventHandler)
					debugf("DOM", "🔗 updateDom: added new event listener %s to DOM element\n", eventType)
				} else {
					debugf("DOM", "🚨 updateDom: event handler %s is not js.Func, got %T\n", name, value)
				}
			} else {
				dom.Set(name, value)
			}
		}
	}
}
