//go:build js && wasm
// +build js,wasm

package fiber

import (
	"strings"
	"sync/atomic"
	"syscall/js"
)

// createElement constructs an Element with optimized allocations and utilization tracking
func createElement(typ interface{}, props map[string]interface{}, children ...interface{}) *Element {
	// Track allocation metrics
	atomic.AddInt64(&poolUtilization.totalAllocations, 1)
	
	// Get element from pool with safe type assertion
	poolElem := elementPool.Get()
	elem, ok := poolElem.(*Element)
	if !ok {
		// Pool miss - track metrics and create new
		atomic.AddInt64(&poolUtilization.elementMisses, 1)
		debugf("DOM", "🚨 createElement: elementPool returned unexpected type %T, creating new Element (miss)\n", poolElem)
		elem = &Element{
			Props: make(map[string]interface{}),
		}
		// Trigger pool optimization periodically
		optimizePoolSizes()
	} else {
		// Pool hit - track metrics
		atomic.AddInt64(&poolUtilization.elementHits, 1)
		atomic.AddInt64(&poolUtilization.totalPoolHits, 1)
		atomic.AddInt32(&poolSizes.element, -1)
		debugf("DOM", "✅ createElement: reused element from pool (hit)\n")
	}

	// Reset the element
	elem.Type = typ

	// Process children to support both component references and return values
	// Optimized with type switch for better performance than chained type assertions
	processedChildren := make([]interface{}, 0, len(children))

	for _, child := range children {
		if child == nil {
			continue
		}

		// Optimized type dispatch using type switch instead of chained assertions
		// This is faster than multiple type assertions as it only evaluates the type once
		switch componentFunc := child.(type) {
		case func(map[string]interface{}) *Element:
			// Component function with map[string]interface{} signature
			result := componentFunc(nil)
			if result != nil {
				processedChildren = append(processedChildren, result)
			}
		case func(Attrs) *Element:
			// Component function with Attrs signature (same as map[string]interface{})
			result := componentFunc(nil)
			if result != nil {
				processedChildren = append(processedChildren, result)
			}
		default:
			// Child is already processed (Element, Text, etc.) - most common case
			processedChildren = append(processedChildren, child)
		}
	}

	elem.Children = processedChildren

	// Handle props efficiently - optimize map clearing for large maps
	// Performance optimization: for large maps, allocating new is faster than clearing
	const clearThreshold = 8 // Threshold where new allocation becomes more efficient
	
	if props != nil {
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

	// Add event listeners and properties - optimized with batching and reduced DOM calls
	//
	// PERFORMANCE OPTIMIZATION EXPLANATION:
	// Instead of calling dom.Get("style") multiple times for each style property,
	// we cache the styleObj and reuse it. This reduces expensive DOM API calls.
	// We also use a switch statement to handle common properties in fast paths,
	// avoiding repeated string comparisons and method calls.
	//
	var styleObj js.Value
	var styleObjInitialized bool
	
	// Batch properties by type to reduce DOM interaction overhead
	for name, value := range fiber.props {
		if name == "children" {
			continue
		}
		
		// Fast path: handle most common properties with optimized branches
		switch name {
		case "dangerouslySetInnerHTML":
			if htmlMap, ok := value.(map[string]string); ok {
				if htmlContent, htmlOk := htmlMap["__html"]; htmlOk {
					dom.Set("innerHTML", htmlContent)
				} else {
					debugf("DOM", "🚨 createDom: dangerouslySetInnerHTML missing __html key\n")
				}
			} else {
				debugf("DOM", "🚨 createDom: dangerouslySetInnerHTML is not map[string]string, got %T\n", value)
			}
		case "class":
			// Handle 'class' attribute using setAttribute
			dom.Call("setAttribute", "class", value)
		case "style":
			// Support both string and map styles - cache styleObj to reduce DOM calls
			//
			// OPTIMIZATION: Instead of calling dom.Get("style") for each style property,
			// we get it once and reuse it. This turns O(n) DOM calls into O(1) + n property sets.
			//
			switch v := value.(type) {
			case string:
				dom.Set("style", v)
			case map[string]string:
				// Initialize styleObj only when needed (lazy initialization)
				if !styleObjInitialized {
					styleObj = dom.Get("style")  // Expensive DOM call - do once
					styleObjInitialized = true
				}
				// Batch style operations - all use the cached styleObj
				for k, val := range v {
					styleObj.Call("setProperty", k, val)  // Fast: reuse cached object
				}
			default:
				debugf("DOM", "🚨 createDom: style must be string or map[string]string, got %T\n", value)
			}
		case "id", "value", "type", "placeholder", "disabled", "checked", "selected":
			// Common properties - set directly for better performance
			// OPTIMIZATION: These are the most frequently used properties, so we handle
			// them in a fast path to avoid the string comparisons in the default case
			dom.Set(name, value)
		default:
			// Check for event handlers
			if len(name) > 2 && name[0] == 'o' && name[1] == 'n' {
				// Event handlers - use cached lowercase conversion
				eventType := strings.ToLower(name[2:])
				if eventHandler, ok := value.(js.Func); ok {
					dom.Call("addEventListener", eventType, eventHandler)
					debugf("DOM", "🔗 createDom: added event listener %s to DOM element\n", eventType)
				} else {
					debugf("DOM", "🚨 createDom: event handler %s is not js.Func, got %T\n", name, value)
				}
			} else {
				// Other properties - set directly
				dom.Set(name, value)
			}
		}
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

	// 2. Add new or changed properties and event listeners (optimized with batching)
	//
	// PERFORMANCE OPTIMIZATION EXPLANATION:
	// Same optimization as createDom - cache the style object to avoid repeated
	// expensive DOM.Get("style") calls. For elements with many style properties,
	// this reduces DOM API calls from O(n) to O(1) + n property operations.
	//
	var styleObj js.Value
	var styleObjInitialized bool
	
	// Batch DOM operations to reduce overhead
	for name, value := range newProps {
		// Skip common exclusions first (most frequent check)
		if name == "children" {
			continue
		}

		// Skip if value hasn't changed - use fast equality check
		if oldValue, exists := oldProps[name]; exists && fastEqual(oldValue, value) {
			continue
		}

		// Optimized switch for common properties - reduces string comparisons
		switch name {
		case "class":
			dom.Call("setAttribute", "class", value)
		case "style":
			// Handle string vs map[string]string styles with cached styleObj
			// Optimized with type switch instead of chained type assertions
			//
			// OPTIMIZATION: For map-based styles, we cache the DOM style object
			// and batch all operations. This is especially beneficial when updating
			// many CSS properties, as it avoids repeated DOM.Get("style") calls.
			//
			switch styleValue := value.(type) {
			case string:
				// String style - most common case
				dom.Set("style", styleValue)
			case map[string]string:
				// Map-based styles - initialize styleObj only when needed
				if !styleObjInitialized {
					styleObj = dom.Get("style")  // Expensive DOM call - do once
					styleObjInitialized = true
				}
				
				// If previous style was a string, clear it completely first
				if _, wasString := oldProps["style"].(string); wasString {
					dom.Set("style", "")
				}
				
				// Batch style removals - all use cached styleObj
				if oldStyleMap, okOld := oldProps["style"].(map[string]string); okOld {
					for k := range oldStyleMap {
						if _, exists := styleValue[k]; !exists {
							styleObj.Call("removeProperty", k)  // Fast: reuse cached object
						}
					}
				}
				
				// Batch style additions/updates - all use cached styleObj
				for k, val := range styleValue {
					styleObj.Call("setProperty", k, val)  // Fast: reuse cached object
				}
			default:
				debugf("DOM", "🚨 updateDom: style must be string or map[string]string, got %T\n", value)
			}
		case "id", "value", "type", "placeholder", "disabled", "checked", "selected":
			// Common properties - batch these for better performance
			dom.Set(name, value)
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
			// Check for event handlers (less common path)
			if len(name) > 2 && name[0] == 'o' && name[1] == 'n' {
				eventType := strings.ToLower(name[2:])
				if eventHandler, ok := value.(js.Func); ok {
					dom.Call("addEventListener", eventType, eventHandler)
					debugf("DOM", "🔗 updateDom: added new event listener %s to DOM element\n", eventType)
				} else {
					debugf("DOM", "🚨 updateDom: event handler %s is not js.Func, got %T\n", name, value)
				}
			} else {
				// Other properties - set directly
				dom.Set(name, value)
			}
		}
	}
}
