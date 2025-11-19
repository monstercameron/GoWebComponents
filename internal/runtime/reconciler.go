package runtime

import (
	"reflect"
	"sync"
)

// currentFiber tracks the fiber being processed (for hooks)
var (
	currentFiber  *Fiber
	emptyChildren = []interface{}{}
	slicePool     = sync.Pool{
		New: func() interface{} {
			// Initial capacity 16 seems reasonable for children
			return make([]interface{}, 0, 16)
		},
	}
)

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
	// Process children: wrap strings in TEXT_ELEMENT
	// We modify the children slice in-place to avoid allocation since it's a varargs slice
	for i, child := range children {
		if str, ok := child.(string); ok {
			children[i] = &Element{
				Type:        "TEXT_ELEMENT",
				TextContent: str,
				// Props:    nil, // No props map needed!
				Children: emptyChildren,
			}
		}
	}

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
func flattenFragments(elements []interface{}) ([]interface{}, bool) {
	// Optimization: Check if we have any fragments before allocating
	hasFragment := false
	for _, elem := range elements {
		if elemPtr, ok := elem.(*Element); ok && elemPtr != nil && elemPtr.Type == "FRAGMENT" {
			hasFragment = true
			break
		}
	}

	if !hasFragment {
		return elements, false
	}

	// Use pool
	flattened := slicePool.Get().([]interface{})
	flattened = flattened[:0] // Reset length

	for _, elem := range elements {
		if elem == nil {
			continue
		}

		if elemPtr, ok := elem.(*Element); ok {
			if elemPtr == nil {
				continue
			}
			if elemPtr.Type == "FRAGMENT" {
				// Recursively flatten the Fragment's children
				if fragmentChildren, ok := elemPtr.Props["children"].([]interface{}); ok {
					res, allocated := flattenFragments(fragmentChildren)
					flattened = append(flattened, res...)
					if allocated {
						// Clear and put back
						for i := range res {
							res[i] = nil
						}
						slicePool.Put(res)
					}
				}
			} else {
				flattened = append(flattened, elem)
			}
		} else {
			flattened = append(flattened, elem)
		}
	}

	return flattened, true
}

// cloneChildFibers clones the child fibers from the alternate to the current fiber
// This is used when skipping reconciliation for non-dirty fibers
func (rt *Runtime) cloneChildFibers(parent *Fiber) {
	if parent.alternate == nil || parent.alternate.child == nil {
		return
	}

	var prevSibling *Fiber
	oldFiber := parent.alternate.child

	for oldFiber != nil {
		newFiber := &Fiber{
			typeOf:         oldFiber.typeOf,
			props:          oldFiber.props,
			textContent:    oldFiber.textContent,
			dom:            oldFiber.dom,
			parent:         parent,
			alternate:      oldFiber,
			effectTag:      "",    // No change
			dirty:          false, // Not dirty
			needsUpdate:    false,
			hooks:          oldFiber.hooks, // Share hooks for non-updated components
			eventCallbacks: oldFiber.eventCallbacks,
		}

		// Optimization: Break the alternate chain to prevent memory leaks and long traversals
		// We only need the immediate alternate for the next reconciliation
		oldFiber.alternate = nil

		if prevSibling == nil {
			parent.child = newFiber
		} else {
			prevSibling.sibling = newFiber
		}
		prevSibling = newFiber
		oldFiber = oldFiber.sibling
	}
}

// reconcileChildren reconciles the children of a fiber
func (rt *Runtime) reconcileChildren(wipFiber *Fiber, elements []interface{}) {
	// Flatten any Fragment elements before reconciliation
	flatElements, wasAllocated := flattenFragments(elements)
	if wasAllocated {
		defer func() {
			for i := range flatElements {
				flatElements[i] = nil
			}
			slicePool.Put(flatElements)
		}()
	}
	elements = flatElements

	index := 0
	var oldFiber *Fiber
	if wipFiber.alternate != nil {
		oldFiber = wipFiber.alternate.child
	}
	var prevSibling *Fiber
	firstChildSet := false

	// Loop 1: Update/Replace (Both exist)
	for index < len(elements) && oldFiber != nil {
		element := elements[index]

		var newFiber *Fiber
		sameType := false

		if element != nil {
			if elem, ok := element.(*Element); ok {
				sameType = isSameType(elem.Type, oldFiber.typeOf)

				if sameType {
					// UPDATE logic
					// Check if this fiber or its subtree needs update
					isDirty := rt.isFiberDirty(oldFiber)
					needsUpdate := isDirty || oldFiber.needsUpdate

					if !needsUpdate {
						if t, ok := elem.Type.(string); ok && t == "TEXT_ELEMENT" {
							oldText := oldFiber.textContent
							if oldText == "" && oldFiber.props != nil {
								oldText, _ = oldFiber.props["nodeValue"].(string)
							}
							newText := elem.TextContent
							if newText == "" && elem.Props != nil {
								newText, _ = elem.Props["nodeValue"].(string)
							}
							needsUpdate = oldText != newText
						} else {
							needsUpdate = !propsEqual(oldFiber.props, elem.Props)
						}
					}

					effectTag := "UPDATE"
					if !needsUpdate {
						effectTag = ""
					}

					newFiber = &Fiber{
						typeOf:      oldFiber.typeOf,
						props:       elem.Props,
						textContent: elem.TextContent,
						dom:         oldFiber.dom,
						parent:      wipFiber,
						alternate:   oldFiber,
						effectTag:   effectTag,
						dirty:       needsUpdate,
					}

					// Optimization: Break the alternate chain
					oldFiber.alternate = nil

					// Advance oldFiber
					oldFiber = oldFiber.sibling
				} else {
					// REPLACE logic (Placement + Deletion)
					newFiber = &Fiber{
						typeOf:      elem.Type,
						props:       elem.Props,
						textContent: elem.TextContent,
						parent:      wipFiber,
						effectTag:   "PLACEMENT",
						dirty:       true,
					}

					// Mark old fiber for deletion
					oldFiber.effectTag = "DELETION"
					rt.deletions = append(rt.deletions, oldFiber)
					oldFiber = oldFiber.sibling
				}
			}
		} else {
			// element is nil, but oldFiber exists
			// Mark old fiber for deletion
			oldFiber.effectTag = "DELETION"
			rt.deletions = append(rt.deletions, oldFiber)
			oldFiber = oldFiber.sibling
			// Don't increment index here, we just consumed oldFiber
			// Wait, if element is nil, it means there is a hole in the array?
			// Or it means we should skip this index?
			// Original logic: "element is nil, oldFiber exists - mark for deletion and advance"
			// So we consume both index and oldFiber?
			// Original logic:
			// } else if oldFiber != nil { ... oldFiber = oldFiber.sibling }
			// It didn't increment index in that branch!
			// Wait, the original logic had:
			// if sameType { ... index++ }
			// else if element != nil { ... index++ }
			// else if oldFiber != nil { ... oldFiber = oldFiber.sibling } (NO index++)
			// else { index++ }

			// So if element is nil, we delete oldFiber and stay at same index?
			// That implies elements[index] is NOT consumed if it is nil?
			// But elements[index] IS nil. So we should consume it?
			// If elements[index] is nil, it means "nothing here".
			// If we have oldFiber, we delete it.
			// If we don't consume index, we will loop forever if elements[index] is nil.
			// Ah, the original logic:
			// if oldFiber != nil { ... } else { index++ }
			// So if oldFiber != nil, it deletes oldFiber and DOES NOT increment index.
			// This means it tries to match elements[index] (which is nil) against the NEXT oldFiber.
			// This effectively deletes all oldFibers until one matches nil? Or until oldFiber is nil?
			// If elements[index] is nil, it will keep deleting oldFibers until oldFiber is nil.
			// Then it hits the `else { index++ }` block.
			// So it deletes all remaining oldFibers?
			// That seems wrong if elements has more items after nil.

			// Let's assume standard behavior: index corresponds to position.
			// If elements[index] is nil, it's a hole. We should probably skip it.
			// But if there was an oldFiber at this position, it should be deleted.
			// So: Delete oldFiber, Increment index.

			// Let's stick to the behavior:
			// If element is nil, we treat it as "nothing to render".
			// If there was something (oldFiber), delete it.
			// And move to next element.
		}

		// Link to parent
		if newFiber != nil {
			if !firstChildSet {
				wipFiber.child = newFiber
				firstChildSet = true
			} else if prevSibling != nil {
				prevSibling.sibling = newFiber
			}
			prevSibling = newFiber
		}

		index++
	}

	// Loop 2: Placement (Remaining elements)
	for index < len(elements) {
		element := elements[index]
		var newFiber *Fiber

		if element != nil {
			if elem, ok := element.(*Element); ok {
				newFiber = &Fiber{
					typeOf:      elem.Type,
					props:       elem.Props,
					textContent: elem.TextContent,
					parent:      wipFiber,
					effectTag:   "PLACEMENT",
					dirty:       true,
				}
			}
		}

		if newFiber != nil {
			if !firstChildSet {
				wipFiber.child = newFiber
				firstChildSet = true
			} else if prevSibling != nil {
				prevSibling.sibling = newFiber
			}
			prevSibling = newFiber
		}
		index++
	}

	// Loop 3: Deletion (Remaining old fibers)
	for oldFiber != nil {
		oldFiber.effectTag = "DELETION"
		rt.deletions = append(rt.deletions, oldFiber)
		oldFiber = oldFiber.sibling
	}
}

// propsEqual compares two property maps for equality
func propsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v1 := range a {
		v2, ok := b[k]
		if !ok {
			return false
		}

		if k == "children" {
			// Avoid deep comparison for children
			// Check if they are the same slice reference
			// If not, assume they are different to avoid O(Subtree) traversal
			if v1 == nil && v2 == nil {
				continue
			}
			if v1 == nil || v2 == nil {
				return false
			}

			// Use reflect to check pointer equality
			rv1 := reflect.ValueOf(v1)
			rv2 := reflect.ValueOf(v2)
			if rv1.Kind() == reflect.Slice && rv2.Kind() == reflect.Slice {
				if rv1.Pointer() == rv2.Pointer() && rv1.Len() == rv2.Len() {
					continue
				}
			}

			// If pointers differ, assume different.
			// This skips DeepEqual.
			return false
		}

		if !fastEqual(v1, v2) {
			return false
		}
	}

	return true
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

	v1 := reflect.ValueOf(type1)
	v2 := reflect.ValueOf(type2)

	if !v1.IsValid() || !v2.IsValid() {
		return false
	}

	if v1.Kind() == reflect.Func && v2.Kind() == reflect.Func {
		return v1.Pointer() == v2.Pointer()
	}

	return reflect.DeepEqual(type1, type2)
}

// isFiberDirty checks if a fiber or any of its alternates are dirty
func (rt *Runtime) isFiberDirty(fiber *Fiber) bool {
	f := fiber
	for f != nil {
		if f.dirty {
			return true
		}
		f = f.alternate
	}
	return false
}

// clearFiberDirty clears the dirty flag on a fiber and its alternates
func (rt *Runtime) clearFiberDirty(fiber *Fiber) {
	f := fiber
	for f != nil {
		f.dirty = false
		f = f.alternate
	}
}

// performUnitOfWork processes a single fiber
func (rt *Runtime) performUnitOfWork(fiber *Fiber) *Fiber {
	if fiber == nil {
		return nil
	}

	// Check if fiber or any alternate is dirty
	isDirty := rt.isFiberDirty(fiber)

	// Skip non-dirty fibers (optimization)
	if !isDirty {
		rt.cloneChildFibers(fiber)
		return rt.getNextUnitOfWork(fiber)
	}

	// Clear dirty flags on fiber and alternates
	rt.clearFiberDirty(fiber)

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
			// Preserve hooks from alternate fiber
			if fiber.alternate != nil && fiber.alternate.hooks != nil {
				// Reuse the hooks struct to avoid allocations and preserve closures
				fiber.hooks = fiber.alternate.hooks

				// Prepare for new render
				fiber.hooks.index = 0
				fiber.hooks.stateIndex = 0
				fiber.hooks.depIndex = 0
				fiber.hooks.memoIndex = 0
				fiber.hooks.callbackIndex = 0
				fiber.hooks.refIndex = 0
				fiber.hooks.idIndex = 0
				fiber.hooks.fetchIndex = 0
				fiber.hooks.funcIndex = 0
				fiber.hooks.atomIndex = 0
				fiber.hooks.cleanupIndex = 0
			}

			// Clear effects
			if fiber.effects != nil {
				fiber.effects = fiber.effects[:0]
			} else {
				fiber.effects = make([]Effect, 0)
			}
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
			if fiber.textContent != "" {
				dom = rt.domAdapter.CreateTextNode(fiber.textContent)
			} else if nodeValue, ok := fiber.props["nodeValue"].(string); ok {
				dom = rt.domAdapter.CreateTextNode(nodeValue)
			}
		case "FRAGMENT":
			// Fragments don't create DOM nodes - children are rendered directly
			return nil
		default:
			// Regular element (not TEXT_ELEMENT or FRAGMENT)
			dom = rt.domAdapter.CreateElement(t)
			// Apply properties only for non-text elements
			rt.updateDomProperties(dom, nil, fiber.props)
		}
	}
	// Function components don't have DOM nodes - they render their children

	return dom
}

// updateDomProperties updates DOM properties
func (rt *Runtime) updateDomProperties(dom DOMNode, oldProps, newProps map[string]interface{}) {
	// Check if dom is nil (interface is nil) or if the concrete value is null
	if dom == nil || dom.IsNull() {
		return
	}

	// Optimization: Fast path for initial render (no old props)
	if len(oldProps) == 0 {
		for name, value := range newProps {
			if name == "children" {
				continue
			}

			switch name {
			case "style":
				if styles, ok := value.(map[string]string); ok {
					rt.domAdapter.SetStyles(dom, styles)
				} else if str, ok := value.(string); ok {
					rt.domAdapter.SetAttribute(dom, "style", str)
				}
			case "className", "class":
				if str, ok := value.(string); ok {
					rt.domAdapter.SetAttribute(dom, "class", str)
				}
			case "value", "checked", "selected":
				rt.domAdapter.SetProperty(dom, name, value)
			default:
				if str, ok := value.(string); ok {
					rt.domAdapter.SetAttribute(dom, name, str)
				} else {
					rt.domAdapter.SetProperty(dom, name, value)
				}
			}
		}
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

		// Optimization: Skip if value hasn't changed
		if oldValue, exists := oldProps[name]; exists && fastEqual(oldValue, value) {
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
		case "value", "checked", "selected":
			// Always set these as properties to ensure UI updates correctly
			rt.domAdapter.SetProperty(dom, name, value)
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
	// Process deletions first
	for _, fiber := range rt.deletions {
		// Deletions need to find their parent DOM node
		// We can't pass a cached parent here easily because deletions can be anywhere
		rt.commitWork(fiber, nil)
	}
	// Clear deletions but keep capacity
	rt.deletions = rt.deletions[:0]

	// Commit the work
	if rt.wipRoot != nil && rt.wipRoot.child != nil {
		// The root fiber's DOM node is the container
		rt.commitWork(rt.wipRoot.child, rt.wipRoot.dom)
	} else {
		// fmt.Printf("[COMMIT] WARNING: wipRoot.child is nil\n")
	}

	// Run effects
	rt.runEffects(rt.wipRoot)

	rt.currentRoot = rt.wipRoot
	rt.wipRoot = nil
}

// commitWork commits a fiber's changes to the DOM
func (rt *Runtime) commitWork(fiber *Fiber, domParent DOMNode) {
	if fiber == nil {
		return
	}

	// If domParent is nil (e.g. from deletions loop), we must find it
	if domParent == nil || domParent.IsNull() {
		var domParentFiber *Fiber = fiber.parent
		for domParentFiber != nil && (domParentFiber.dom == nil || domParentFiber.dom.IsNull()) {
			domParentFiber = domParentFiber.parent
		}
		if domParentFiber != nil {
			domParent = domParentFiber.dom
		}
	}

	if domParent != nil && !domParent.IsNull() {
		if fiber.effectTag == "PLACEMENT" && fiber.dom != nil && !fiber.dom.IsNull() {
			rt.domAdapter.AppendChild(domParent, fiber.dom)
		} else if fiber.effectTag == "UPDATE" && fiber.dom != nil && !fiber.dom.IsNull() {
			if fiber.alternate != nil {
				// Check if this is a text node
				if t, ok := fiber.typeOf.(string); ok && t == "TEXT_ELEMENT" {
					// Update text content
					oldValue := fiber.alternate.textContent
					if oldValue == "" && fiber.alternate.props != nil {
						oldValue, _ = fiber.alternate.props["nodeValue"].(string)
					}

					newValue := fiber.textContent
					if newValue == "" && fiber.props != nil {
						newValue, _ = fiber.props["nodeValue"].(string)
					}

					if oldValue != newValue {
						rt.domAdapter.SetTextContent(fiber.dom, newValue)
					}
				} else {
					// Regular element - update properties
					rt.updateDomProperties(fiber.dom, fiber.alternate.props, fiber.props)
				}
			}
		} else if fiber.effectTag == "DELETION" {
			// fmt.Printf("DEBUG: Committing deletion for %v\n", fiber.typeOf)
			rt.commitDeletion(fiber, domParent)
			return
		}
	}

	// Determine the parent DOM node for children
	// If this fiber has a DOM node, it becomes the parent for its children
	childDomParent := domParent
	if fiber.dom != nil && !fiber.dom.IsNull() {
		childDomParent = fiber.dom
	}

	// Recursively commit children and siblings
	if fiber.child != nil {
		rt.commitWork(fiber.child, childDomParent)
	}
	if fiber.sibling != nil {
		rt.commitWork(fiber.sibling, domParent)
	}
}

// commitDeletion removes a fiber from the DOM and runs cleanup functions
func (rt *Runtime) commitDeletion(fiber *Fiber, domParent DOMNode) {
	if fiber == nil {
		return
	}
	// fmt.Printf("DEBUG: commitDeletion for %v\n", fiber.typeOf)
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

	// If this fiber has a DOM node, remove it (this stops the recursion down that branch)
	if fiber.dom != nil && !fiber.dom.IsNull() {
		// fmt.Printf("DEBUG: Removing child from DOM\n")
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
		cleanup := effect.Fn()
		if cleanup != nil {
			fiber.hooks.cleanups[effect.CleanupIndex] = cleanup
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
