package runtime

import (
	"fmt"
	"reflect"
)

// currentFiber tracks the fiber being processed (for hooks)
var currentFiber *Fiber

// GetCurrentFiber returns the fiber currently being processed
func GetCurrentFiber() *Fiber {
	return currentFiber
}

// SetCurrentFiber sets the current fiber (used during component rendering)
func SetCurrentFiber(fiber *Fiber) {
	currentFiber = fiber
}

// CreateElement creates a new virtual DOM element
func CreateElement(typ interface{}, props map[string]interface{}, children ...interface{}) *Element {
	elem := &Element{
		Type:     typ,
		Props:    make(map[string]interface{}),
		Children: children,
	}

	for k, v := range props {
		elem.Props[k] = v
	}

	// Always set children in props, even if empty, so reconciliation can handle deletions
	elem.Props["children"] = children

	return elem
}

// flattenFragments flattens Fragment elements, returning a new slice without Fragment wrappers
// This allows Fragments to work as transparent containers that don't create DOM nodes
func flattenFragments(elements []interface{}) []interface{} {
	var flattened []interface{}

	for _, elem := range elements {
		if elemPtr, ok := elem.(*Element); ok && elemPtr.Type == "FRAGMENT" {
			// Recursively flatten the Fragment's children
			if fragmentChildren, ok := elemPtr.Props["children"].([]interface{}); ok {
				flattened = append(flattened, flattenFragments(fragmentChildren)...)
			}
		} else {
			flattened = append(flattened, elem)
		}
	}

	return flattened
}

// reconcileChildren reconciles the children of a fiber
func (rt *Runtime) reconcileChildren(wipFiber *Fiber, elements []interface{}) {
	// Flatten any Fragment elements before reconciliation
	elements = flattenFragments(elements)
	fmt.Printf("[RECONCILE] wipFiber.typeOf=%v, elements count=%d\n", wipFiber.typeOf, len(elements))

	index := 0
	var oldFiber *Fiber
	if wipFiber.alternate != nil {
		oldFiber = wipFiber.alternate.child
	}
	var prevSibling *Fiber
	firstChildSet := false

	for index < len(elements) || oldFiber != nil {
		var element interface{}
		if index < len(elements) {
			element = elements[index]
		}

		var newFiber *Fiber
		sameType := false

		if oldFiber != nil && element != nil {
			if elem, ok := element.(*Element); ok {
				sameType = isSameType(elem.Type, oldFiber.typeOf)
			}
		}

		if sameType {
			// Reuse the existing fiber
			if elem, ok := element.(*Element); ok {
				// Check if this fiber or its subtree needs update
				needsUpdate := oldFiber.dirty || oldFiber.needsUpdate || !reflect.DeepEqual(oldFiber.props, elem.Props)

				// Always mark as dirty if props contain event handlers (they're closures that may have changed)
				if !needsUpdate {
					for key := range elem.Props {
						if len(key) > 2 && key[:2] == "on" {
							needsUpdate = true
							break
						}
					}
				}

				newFiber = &Fiber{
					typeOf:    oldFiber.typeOf,
					props:     elem.Props,
					dom:       oldFiber.dom,
					parent:    wipFiber,
					alternate: oldFiber,
					effectTag: "UPDATE",
					dirty:     needsUpdate,
				}
			} else {
				fmt.Printf("  [UPDATE] ERROR: not Element type=%T\n", element)
			}
			// Link to parent
			if !firstChildSet {
				wipFiber.child = newFiber
				firstChildSet = true
				fmt.Printf("[RECONCILE] set wipFiber(%v).child=%p (type=%v)\n", wipFiber.typeOf, newFiber, newFiber.typeOf)
			} else if newFiber != nil && prevSibling != nil {
				prevSibling.sibling = newFiber
				fmt.Printf("[RECONCILE] set prevSibling.sibling=%p\n", newFiber)
			}
			if newFiber != nil {
				prevSibling = newFiber
			}
			// Advance oldFiber only when we reuse it
			if oldFiber != nil {
				oldFiber = oldFiber.sibling
			}
			index++
		} else if element != nil {
			// Create a new fiber for a different element type
			if elem, ok := element.(*Element); ok {
				newFiber = &Fiber{
					typeOf:    elem.Type,
					props:     elem.Props,
					parent:    wipFiber,
					effectTag: "PLACEMENT",
					dirty:     true,
				}
				fmt.Printf("[RECONCILE] PLACEMENT: idx=%d, type=%v, newFiber=%p\n", index, elem.Type, newFiber)
				// Log when placing element without old fiber (potential duplication)
				if oldFiber != nil {
					fmt.Printf("[DUP] PLACEMENT idx=%d type=%v when oldFiber=%p exists (sibling=%p)\n", index, elem.Type, oldFiber, oldFiber.sibling)
					// Mark old fiber for deletion on type mismatch
					oldFiber.effectTag = "DELETION"
					rt.deletions = append(rt.deletions, oldFiber)
					oldFiber = oldFiber.sibling
				}
			} else {
				fmt.Printf("  [PLACEMENT] ERROR: not Element\n")
			}
			// Link to parent
			if !firstChildSet {
				wipFiber.child = newFiber
				firstChildSet = true
				fmt.Printf("[RECONCILE] set wipFiber(%v).child=%p (type=%v)\n", wipFiber.typeOf, newFiber, newFiber.typeOf)
			} else if newFiber != nil && prevSibling != nil {
				prevSibling.sibling = newFiber
				fmt.Printf("[RECONCILE] set prevSibling.sibling=%p\n", newFiber)
			}
			if newFiber != nil {
				prevSibling = newFiber
			}
			index++
		} else if oldFiber != nil {
			// element is nil, oldFiber exists - mark for deletion and advance
			oldFiber.effectTag = "DELETION"
			rt.deletions = append(rt.deletions, oldFiber)
			oldFiber = oldFiber.sibling
		} else {
			// element is nil and oldFiber is nil - skip this index
			// This prevents infinite loop when nil elements exist without old fibers
			index++
		}
	}
}

// isSameType checks if two component types are the same
func isSameType(type1, type2 interface{}) bool {
	// String types (HTML tags)
	if s1, ok1 := type1.(string); ok1 {
		if s2, ok2 := type2.(string); ok2 {
			return s1 == s2
		}
		return false
	}

	// Function components - compare function pointers
	t1 := reflect.TypeOf(type1)
	t2 := reflect.TypeOf(type2)

	if t1 == nil || t2 == nil {
		return false
	}

	if t1.Kind() == reflect.Func && t2.Kind() == reflect.Func {
		v1 := reflect.ValueOf(type1)
		v2 := reflect.ValueOf(type2)
		return v1.Pointer() == v2.Pointer()
	}

	return reflect.DeepEqual(type1, type2)
}

// performUnitOfWork processes a single fiber
func (rt *Runtime) performUnitOfWork(fiber *Fiber) *Fiber {
	if fiber == nil {
		return nil
	}

	fmt.Printf("[PERFORM] fiber.typeOf=%v, dirty=%v, child=%p\n", fiber.typeOf, fiber.dirty, fiber.child)

	// Skip non-dirty fibers (optimization)
	if !fiber.dirty {
		return rt.getNextUnitOfWork(fiber)
	}

	fiber.dirty = false

	if fiber.typeOf == nil || fiber.typeOf == "ROOT" {
		// Root fiber - reconcile children
		if children, ok := fiber.props["children"].([]interface{}); ok {
			rt.reconcileChildren(fiber, children)
		}
	} else {
		switch fiber.typeOf.(type) {
		case string:
			// Host component (HTML element)
			if fiber.dom == nil || fiber.dom.IsNull() {
				fiber.dom = rt.createDom(fiber)
			}

			if propsChildren, ok := fiber.props["children"]; ok {
				if elements, elementsOk := propsChildren.([]interface{}); elementsOk {
					rt.reconcileChildren(fiber, elements)
				}
			}

		default:
			// Function component
			currentFiber = fiber
			// TODO: ensure FinalizeHookOrder is called after function component render; currently only validateHookOrder runs

			// Preserve hooks from alternate fiber
			if fiber.alternate != nil && fiber.alternate.hooks != nil {
				// REUSE the same hooks object so that old event handler closures
				// still reference the correct state!
				fiber.hooks = fiber.alternate.hooks
				// Reset index for new render
				fiber.hooks.index = 0
				fiber.hooks.callOrder = make([]HookCall, 0)
			} else if fiber.hooks == nil {
				fiber.hooks = &Hooks{
					state:        make([]interface{}, 0),
					pendingState: make([]interface{}, 0),
					deps:         make([][]interface{}, 0),
					memos:        make([]memoizedValue, 0),
					callbacks:    make([]callbackValue, 0),
					refs:         make([]*RefValue, 0),
					cleanups:     make([]func(), 0),
					callOrder:    make([]HookCall, 0),
					prevOrder:    make([]HookCall, 0),
					index:        0,
				}
			}

			// Clear effects
			fiber.effects = make([]func(), 0) // Call component function
			var element *Element
			if fn, ok := fiber.typeOf.(func(map[string]interface{}) *Element); ok {
				element = fn(fiber.props)
			} else if fn, ok := fiber.typeOf.(func(Attrs) *Element); ok {
				element = fn(Attrs(fiber.props))
			}

			if element != nil {
				rt.reconcileChildren(fiber, []interface{}{element})
			}
		}
	}

	return rt.getNextUnitOfWork(fiber)
}

// getNextUnitOfWork determines the next fiber to process
func (rt *Runtime) getNextUnitOfWork(fiber *Fiber) *Fiber {
	// Child first
	if fiber.child != nil {
		return fiber.child
	}

	// Then sibling
	nextFiber := fiber
	for nextFiber != nil {
		if nextFiber.sibling != nil {
			return nextFiber.sibling
		}
		nextFiber = nextFiber.parent
	}

	return nil
}

// createDom creates a DOM node from a fiber
func (rt *Runtime) createDom(fiber *Fiber) DOMNode {
	var dom DOMNode

	if t, ok := fiber.typeOf.(string); ok {
		switch t {
		case "TEXT_ELEMENT":
			if nodeValue, ok := fiber.props["nodeValue"].(string); ok {
				dom = rt.domAdapter.CreateTextNode(nodeValue)
			}
		case "FRAGMENT":
			// Fragments don't create DOM nodes - children are rendered directly
			return nil
		default:
			// Regular element (not TEXT_ELEMENT or FRAGMENT)
			dom = rt.domAdapter.CreateElement(t)
			// Apply properties only for non-text elements
			rt.updateDomProperties(dom, make(map[string]interface{}), fiber.props)
		}
	}
	// Function components don't have DOM nodes - they render their children

	return dom
}

// updateDomProperties updates DOM properties
func (rt *Runtime) updateDomProperties(dom DOMNode, oldProps, newProps map[string]interface{}) {
	// Check if dom is nil (interface is nil) or if the concrete value is null
	if dom == nil {
		return
	}
	if dom.IsNull() {
		return
	}

	// Remove old properties
	for name := range oldProps {
		if name == "children" {
			continue
		}
		if _, exists := newProps[name]; !exists {
			rt.domAdapter.RemoveAttribute(dom, name)
		}
	}

	// Set new properties
	for name, value := range newProps {
		if name == "children" {
			continue
		}

		switch name {
		case "style":
			if styles, ok := value.(map[string]string); ok {
				rt.domAdapter.SetStyles(dom, styles)
			} else if str, ok := value.(string); ok {
				// Allow inline style strings as attribute values
				rt.domAdapter.SetAttribute(dom, "style", str)
			}
		case "className", "class":
			if str, ok := value.(string); ok {
				rt.domAdapter.SetAttribute(dom, "class", str)
			}
		default:
			if str, ok := value.(string); ok {
				rt.domAdapter.SetAttribute(dom, name, str)
			} else {
				// Always update properties (especially event handlers which are closures)
				rt.domAdapter.SetProperty(dom, name, value)
			}
		}
	}
}

// commitRoot commits all changes to the DOM
func (rt *Runtime) commitRoot() {
	fmt.Printf("[COMMIT] commitRoot starting, deletions=%d\n", len(rt.deletions))
	// Process deletions first
	for _, fiber := range rt.deletions {
		rt.commitWork(fiber)
	}
	rt.deletions = make([]*Fiber, 0)

	// Commit the work
	if rt.wipRoot != nil && rt.wipRoot.child != nil {
		fmt.Printf("[COMMIT] committing wipRoot.child, typeOf=%v\n", rt.wipRoot.child.typeOf)
		rt.commitWork(rt.wipRoot.child)
	} else {
		fmt.Printf("[COMMIT] WARNING: wipRoot.child is nil\n")
	}

	// Run effects
	rt.runEffects(rt.wipRoot)

	rt.currentRoot = rt.wipRoot
	rt.wipRoot = nil
}

// commitWork commits a fiber's changes to the DOM
func (rt *Runtime) commitWork(fiber *Fiber) {
	if fiber == nil {
		return
	}

	fmt.Printf("[COMMIT] commitWork fiber typeOf=%v effectTag=%v\n", fiber.typeOf, fiber.effectTag)

	// Find the parent DOM node
	var domParentFiber *Fiber = fiber.parent
	for domParentFiber != nil && (domParentFiber.dom == nil || domParentFiber.dom.IsNull()) {
		domParentFiber = domParentFiber.parent
	}

	if domParentFiber != nil && domParentFiber.dom != nil && !domParentFiber.dom.IsNull() {
		domParent := domParentFiber.dom

		if fiber.effectTag == "PLACEMENT" && fiber.dom != nil && !fiber.dom.IsNull() {
			rt.domAdapter.AppendChild(domParent, fiber.dom)
		} else if fiber.effectTag == "UPDATE" && fiber.dom != nil && !fiber.dom.IsNull() {
			if fiber.alternate != nil {
				// Check if this is a text node
				if t, ok := fiber.typeOf.(string); ok && t == "TEXT_ELEMENT" {
					// Update text content
					oldValue, _ := fiber.alternate.props["nodeValue"].(string)
					newValue, _ := fiber.props["nodeValue"].(string)
					if oldValue != newValue {
						rt.domAdapter.SetTextContent(fiber.dom, newValue)
					}
				} else {
					// Regular element - update properties
					rt.updateDomProperties(fiber.dom, fiber.alternate.props, fiber.props)
				}
			}
		} else if fiber.effectTag == "DELETION" {
			rt.commitDeletion(fiber, domParent)
			return
		}
	}

	// Recursively commit children and siblings
	if fiber.child != nil {
		rt.commitWork(fiber.child)
	}
	if fiber.sibling != nil {
		rt.commitWork(fiber.sibling)
	}
}

// commitDeletion removes a fiber from the DOM and runs cleanup functions
func (rt *Runtime) commitDeletion(fiber *Fiber, domParent DOMNode) {
	if fiber == nil {
		return
	}
	// Run all cleanup functions before removing from DOM
	rt.runCleanups(fiber)

	// Cleanup atom subscriptions for this fiber and subtree
	rt.CleanupAtomSubscriptions(fiber)

	if fiber.dom != nil && !fiber.dom.IsNull() {
		// This fiber has a DOM node, remove it
		rt.domAdapter.RemoveChild(domParent, fiber.dom)
	} else {
		// Function component without DOM node - recursively delete all descendants
		// We need to find and remove all actual DOM nodes in the subtree
		rt.deleteFiberSubtree(fiber, domParent)
	}
}

// deleteFiberSubtree recursively removes all DOM nodes in a fiber's subtree
func (rt *Runtime) deleteFiberSubtree(fiber *Fiber, domParent DOMNode) {
	if fiber == nil {
		return
	}

	fmt.Printf("  deleteFiberSubtree: fiber=%p type=%v dom=%p\n", fiber, fiber.typeOf, fiber.dom)

	// If this fiber has a DOM node, remove it (this stops the recursion down that branch)
	if fiber.dom != nil && !fiber.dom.IsNull() {
		fmt.Printf("    deleteFiberSubtree: removing DOM node %p from parent %p\n", fiber.dom, domParent)
		rt.domAdapter.RemoveChild(domParent, fiber.dom)
		return
	}

	// No DOM node on this fiber - recurse to children
	rt.deleteFiberSubtree(fiber.child, domParent)

	// After processing the child, we need to handle siblings
	// But siblings should be at the same level, and we're being called from commitDeletion
	// which means the parent will handle siblings. We should NOT recurse to sibling here.
	// Instead, the caller should handle the sibling chain.
}

// runCleanups runs all cleanup functions for a fiber and its children
func (rt *Runtime) runCleanups(fiber *Fiber) {
	if fiber == nil {
		return
	}

	// Run this fiber's cleanups
	if fiber.hooks != nil {
		for _, cleanup := range fiber.hooks.cleanups {
			if cleanup != nil {
				cleanup()
			}
		}
	}

	// Recursively run cleanups for children and siblings
	if fiber.child != nil {
		rt.runCleanups(fiber.child)
	}
	if fiber.sibling != nil {
		rt.runCleanups(fiber.sibling)
	}
}

// runEffects runs all effects for a fiber tree
func (rt *Runtime) runEffects(fiber *Fiber) {
	if fiber == nil {
		return
	}

	// Run this fiber's effects
	for _, effect := range fiber.effects {
		if effect != nil {
			effect()
		}
	}

	// Recursively run effects for children and siblings
	if fiber.child != nil {
		rt.runEffects(fiber.child)
	}
	if fiber.sibling != nil {
		rt.runEffects(fiber.sibling)
	}
}
