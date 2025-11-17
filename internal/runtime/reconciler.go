package runtime

import "reflect"

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

	if props != nil {
		for k, v := range props {
			elem.Props[k] = v
		}
	}

	if len(children) > 0 {
		elem.Props["children"] = children
	}

	return elem
}

// reconcileChildren reconciles the children of a fiber
func (rt *Runtime) reconcileChildren(wipFiber *Fiber, elements []interface{}) {
	index := 0
	var oldFiber *Fiber
	if wipFiber.alternate != nil {
		oldFiber = wipFiber.alternate.child
	}
	var prevSibling *Fiber

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
				newFiber = &Fiber{
					typeOf:    oldFiber.typeOf,
					props:     elem.Props,
					dom:       oldFiber.dom,
					parent:    wipFiber,
					alternate: oldFiber,
					effectTag: "UPDATE",
					// TODO: avoid reflect.DeepEqual on props every render; use a cheaper diff
					dirty: !reflect.DeepEqual(oldFiber.props, elem.Props),
				}
			}
		} else if element != nil {
			// Create a new fiber
			if elem, ok := element.(*Element); ok {
				newFiber = &Fiber{
					typeOf:    elem.Type,
					props:     elem.Props,
					parent:    wipFiber,
					effectTag: "PLACEMENT",
					dirty:     true,
				}
			}
		}

		if oldFiber != nil && !sameType {
			// Mark the old fiber for deletion
			oldFiber.effectTag = "DELETION"
			rt.deletions = append(rt.deletions, oldFiber)
		}

		if oldFiber != nil {
			oldFiber = oldFiber.sibling
		}

		if index == 0 {
			wipFiber.child = newFiber
		} else if element != nil && prevSibling != nil {
			prevSibling.sibling = newFiber
		}

		prevSibling = newFiber
		index++
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
				if fiber.hooks == nil {
					fiber.hooks = &Hooks{
						state:     make([]interface{}, 0),
						deps:      make([][]interface{}, 0),
						memos:     make([]memoizedValue, 0),
						callOrder: make([]HookCall, 0),
						prevOrder: make([]HookCall, 0),
					}
				}
				// Copy state from previous render
				fiber.hooks.state = append([]interface{}(nil), fiber.alternate.hooks.state...)
				fiber.hooks.deps = append([][]interface{}(nil), fiber.alternate.hooks.deps...)
				fiber.hooks.memos = append([]memoizedValue(nil), fiber.alternate.hooks.memos...)
				fiber.hooks.prevOrder = append([]HookCall(nil), fiber.alternate.hooks.prevOrder...)
				fiber.hooks.index = 0
				fiber.hooks.callOrder = make([]HookCall, 0)
			} else if fiber.hooks == nil {
				fiber.hooks = &Hooks{
					state:     make([]interface{}, 0),
					deps:      make([][]interface{}, 0),
					memos:     make([]memoizedValue, 0),
					callOrder: make([]HookCall, 0),
					prevOrder: make([]HookCall, 0),
					index:     0,
				}
			}

			// Clear effects
			fiber.effects = make([]func(), 0)

			// Call component function
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
		if t == "TEXT_ELEMENT" {
			if nodeValue, ok := fiber.props["nodeValue"].(string); ok {
				dom = rt.domAdapter.CreateTextNode(nodeValue)
			}
		} else {
			dom = rt.domAdapter.CreateElement(t)
		}
	}

	// Apply properties
	rt.updateDomProperties(dom, make(map[string]interface{}), fiber.props)

	return dom
}

// updateDomProperties updates DOM properties
func (rt *Runtime) updateDomProperties(dom DOMNode, oldProps, newProps map[string]interface{}) {
	if dom == nil || dom.IsNull() {
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
			}
		case "className", "class":
			if str, ok := value.(string); ok {
				rt.domAdapter.SetAttribute(dom, "class", str)
			}
		default:
			if str, ok := value.(string); ok {
				rt.domAdapter.SetAttribute(dom, name, str)
			} else {
				rt.domAdapter.SetProperty(dom, name, value)
			}
		}
	}
}

// commitRoot commits all changes to the DOM
func (rt *Runtime) commitRoot() {
	// Process deletions first
	for _, fiber := range rt.deletions {
		rt.commitWork(fiber)
	}
	rt.deletions = make([]*Fiber, 0)

	// Commit the work
	if rt.wipRoot != nil && rt.wipRoot.child != nil {
		rt.commitWork(rt.wipRoot.child)
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
				rt.updateDomProperties(fiber.dom, fiber.alternate.props, fiber.props)
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

// commitDeletion removes a fiber from the DOM
func (rt *Runtime) commitDeletion(fiber *Fiber, domParent DOMNode) {
	if fiber.dom != nil && !fiber.dom.IsNull() {
		rt.domAdapter.RemoveChild(domParent, fiber.dom)
	} else if fiber.child != nil {
		rt.commitDeletion(fiber.child, domParent)
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
