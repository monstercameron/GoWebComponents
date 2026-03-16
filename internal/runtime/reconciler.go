package runtime

import (
	"reflect"
	"sync"
	"time"
)

// currentFiber tracks the fiber being processed (for hooks)
var (
	currentFiber  *Fiber
	emptyChildren = []interface{}{}
	propMetaCache = map[string]domPropMeta{
		"children":  {kind: propKindSkip},
		"style":     {kind: propKindStyle, attrName: "style"},
		"className": {kind: propKindClass, attrName: "class"},
		"class":     {kind: propKindClass, attrName: "class"},
		"value":     {kind: propKindSpecialProperty, attrName: "value", resetValue: "", shouldReset: true},
		"checked":   {kind: propKindSpecialProperty, attrName: "checked", resetValue: false, shouldReset: true},
		"selected":  {kind: propKindSpecialProperty, attrName: "selected", resetValue: false, shouldReset: true},
		"disabled":  {kind: propKindDefault, attrName: "disabled", resetValue: false, shouldReset: true},
		"required":  {kind: propKindDefault, attrName: "required", resetValue: false, shouldReset: true},
		"readOnly":  {kind: propKindDefault, attrName: "readOnly", resetValue: false, shouldReset: true},
		"hidden":    {kind: propKindDefault, attrName: "hidden", resetValue: false, shouldReset: true},
		"multiple":  {kind: propKindDefault, attrName: "multiple", resetValue: false, shouldReset: true},
		"autofocus": {kind: propKindDefault, attrName: "autofocus", resetValue: false, shouldReset: true},
		"onclick":   {kind: propKindDefault, attrName: "onclick", resetValue: nil, shouldReset: true},
		"oninput":   {kind: propKindDefault, attrName: "oninput", resetValue: nil, shouldReset: true},
		"onchange":  {kind: propKindDefault, attrName: "onchange", resetValue: nil, shouldReset: true},
		"onsubmit":  {kind: propKindDefault, attrName: "onsubmit", resetValue: nil, shouldReset: true},
		"onkeydown": {kind: propKindDefault, attrName: "onkeydown", resetValue: nil, shouldReset: true},
		"onkeyup":   {kind: propKindDefault, attrName: "onkeyup", resetValue: nil, shouldReset: true},
		"onfocus":   {kind: propKindDefault, attrName: "onfocus", resetValue: nil, shouldReset: true},
		"onblur":    {kind: propKindDefault, attrName: "onblur", resetValue: nil, shouldReset: true},
	}
	slicePool = sync.Pool{
		New: func() interface{} {
			// Initial capacity 16 seems reasonable for children
			return make([]interface{}, 0, 16)
		},
	}
	// Fiber pool to reduce allocations
	fiberPool = sync.Pool{
		New: func() interface{} {
			return &Fiber{}
		},
	}
	keyedFiberMapPool = sync.Pool{
		New: func() interface{} {
			return make(map[interface{}]*Fiber, 16)
		},
	}
	fiberScratchSlicePool = sync.Pool{
		New: func() interface{} {
			return make([]*Fiber, 0, 16)
		},
	}
)

type domPropKind uint8

const (
	propKindDefault domPropKind = iota
	propKindSkip
	propKindStyle
	propKindClass
	propKindSpecialProperty
)

type domPropMeta struct {
	kind        domPropKind
	attrName    string
	resetValue  interface{}
	shouldReset bool
}

func getPropMeta(name string) domPropMeta {
	if meta, ok := propMetaCache[name]; ok {
		return meta
	}
	return domPropMeta{kind: propKindDefault, attrName: name}
}

func acquireWorkInProgress(oldFiber *Fiber) *Fiber {
	if oldFiber != nil && oldFiber.alternate != nil {
		reused := oldFiber.alternate
		*reused = Fiber{}
		return reused
	}
	reused := fiberPool.Get().(*Fiber)
	*reused = Fiber{}
	return reused
}

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
	if len(children) == 0 {
		children = emptyChildren
	}

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

	propsLen := len(props)
	elem := &Element{
		Type:     typ,
		Props:    make(map[string]interface{}, propsLen+1),
		Children: children,
	}

	if propsLen > 0 {
		for k, v := range props {
			elem.Props[k] = v
		}
	}
	elem.Props["children"] = children

	return elem
}

func flattenFragments(elements []interface{}) ([]interface{}, bool) {
	needsFlatten := false
	for _, element := range elements {
		elem, ok := element.(*Element)
		if !ok || elem == nil {
			continue
		}
		if t, ok := elem.Type.(string); ok && t == "FRAGMENT" {
			needsFlatten = true
			break
		}
	}

	if !needsFlatten {
		return elements, false
	}

	flattened := slicePool.Get().([]interface{})
	flattened = flattened[:0]
	for _, element := range elements {
		elem, ok := element.(*Element)
		if !ok {
			if element != nil {
				flattened = append(flattened, element)
			}
			continue
		}
		if elem == nil {
			continue
		}
		if t, ok := elem.Type.(string); ok && t == "FRAGMENT" {
			if children, ok := elem.Props["children"].([]interface{}); ok {
				res, allocated := flattenFragments(children)
				flattened = append(flattened, res...)
				if allocated {
					for i := range res {
						res[i] = nil
					}
					slicePool.Put(res)
				}
			}
			continue
		}
		flattened = append(flattened, elem)
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
		newFiber := acquireWorkInProgress(oldFiber)
		*newFiber = Fiber{
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
		if newFiber.hooks != nil {
			newFiber.hooks.owner = newFiber
		}

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
	// Fast path: empty elements
	if len(elements) == 0 {
		if wipFiber.alternate != nil && wipFiber.alternate.child != nil {
			// Delete all old children
			oldFiber := wipFiber.alternate.child
			for oldFiber != nil {
				oldFiber.effectTag = "DELETION"
				rt.deletions = append(rt.deletions, oldFiber)
				oldFiber = oldFiber.sibling
			}
		}
		return
	}

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
	reportMissingKeys(wipFiber, elements)

	if shouldUseKeyedReconciliation(elements, wipFiber) {
		rt.reconcileKeyedChildren(wipFiber, elements)
		return
	}

	index := 0
	var oldFiber *Fiber
	if wipFiber.alternate != nil {
		oldFiber = wipFiber.alternate.child
	}
	var prevSibling *Fiber
	firstChildSet := false

	// Loop 1: Update/Replace (Both exist)
	// Pre-compute element count for better branch prediction
	elemCount := len(elements)
	for index < elemCount && oldFiber != nil {
		element := elements[index]

		var newFiber *Fiber

		if element != nil {
			if elem, ok := element.(*Element); ok && elem != nil {
				// Inline fast path for string type comparison (most common case)
				sameType := false
				if s1, ok1 := elem.Type.(string); ok1 {
					if s2, ok2 := oldFiber.typeOf.(string); ok2 {
						sameType = s1 == s2
					}
				} else {
					sameType = isSameType(elem.Type, oldFiber.typeOf)
				}

				if sameType {
					// UPDATE logic - optimized path
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

					// Get from pool and reset
					newFiber = acquireWorkInProgress(oldFiber)
					*newFiber = Fiber{
						typeOf:         oldFiber.typeOf,
						props:          elem.Props,
						textContent:    elem.TextContent,
						dom:            oldFiber.dom,
						parent:         wipFiber,
						alternate:      oldFiber,
						effectTag:      effectTag,
						dirty:          needsUpdate,
						hooks:          oldFiber.hooks,
						eventCallbacks: oldFiber.eventCallbacks,
					}

					// Advance oldFiber
					oldFiber = oldFiber.sibling
				} else {
					// REPLACE logic (Placement + Deletion)
					newFiber = acquireWorkInProgress(oldFiber)
					*newFiber = Fiber{
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
			oldFiber.effectTag = "DELETION"
			rt.deletions = append(rt.deletions, oldFiber)
			oldFiber = oldFiber.sibling
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
			if elem, ok := element.(*Element); ok && elem != nil {
				newFiber = acquireWorkInProgress(nil)
				*newFiber = Fiber{
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

func shouldUseKeyedReconciliation(elements []interface{}, wipFiber *Fiber) bool {
	for _, element := range elements {
		elem, ok := element.(*Element)
		if !ok || elem == nil {
			continue
		}
		if hasElementKey(elem) {
			return true
		}
	}

	if wipFiber == nil || wipFiber.alternate == nil {
		return false
	}
	for oldFiber := wipFiber.alternate.child; oldFiber != nil; oldFiber = oldFiber.sibling {
		if hasFiberKey(oldFiber) {
			return true
		}
	}

	return false
}

func (rt *Runtime) reconcileKeyedChildren(wipFiber *Fiber, elements []interface{}) {
	oldByKey := keyedFiberMapPool.Get().(map[interface{}]*Fiber)
	oldFallbackKeyed := fiberScratchSlicePool.Get().([]*Fiber)
	oldFallbackKeyed = oldFallbackKeyed[:0]
	oldUnkeyed := fiberScratchSlicePool.Get().([]*Fiber)
	oldUnkeyed = oldUnkeyed[:0]
	defer func() {
		clear(oldByKey)
		keyedFiberMapPool.Put(oldByKey)
		clear(oldFallbackKeyed)
		fiberScratchSlicePool.Put(oldFallbackKeyed[:0])
		clear(oldUnkeyed)
		fiberScratchSlicePool.Put(oldUnkeyed[:0])
	}()

	var oldFirst *Fiber
	if wipFiber.alternate != nil {
		oldFirst = wipFiber.alternate.child
	}

	for oldFiber := oldFirst; oldFiber != nil; oldFiber = oldFiber.sibling {
		if key, ok := fiberComparableKey(oldFiber); ok {
			oldByKey[key] = oldFiber
		} else if hasFiberKey(oldFiber) {
			oldFallbackKeyed = append(oldFallbackKeyed, oldFiber)
		} else {
			oldUnkeyed = append(oldUnkeyed, oldFiber)
		}
	}

	unkeyedIndex := 0
	var prevSibling *Fiber
	firstChildSet := false

	for _, element := range elements {
		elem, ok := element.(*Element)
		if !ok || elem == nil {
			continue
		}

		var matchedOld *Fiber
		if key, hasKey := elementComparableKey(elem); hasKey {
			matchedOld = oldByKey[key]
			if matchedOld != nil {
				delete(oldByKey, key)
			}
		} else if hasElementKey(elem) {
			matchedOld = takeMatchingFallbackKeyed(oldFallbackKeyed, elem)
		} else if unkeyedIndex < len(oldUnkeyed) {
			matchedOld = oldUnkeyed[unkeyedIndex]
			unkeyedIndex++
		}

		var newFiber *Fiber
		if matchedOld != nil && sameFiberType(elem, matchedOld) {
			isDirty := rt.isFiberDirty(matchedOld)
			needsUpdate := isDirty || matchedOld.needsUpdate

			if !needsUpdate {
				if t, ok := elem.Type.(string); ok && t == "TEXT_ELEMENT" {
					oldText := matchedOld.textContent
					if oldText == "" && matchedOld.props != nil {
						oldText, _ = matchedOld.props["nodeValue"].(string)
					}
					newText := elem.TextContent
					if newText == "" && elem.Props != nil {
						newText, _ = elem.Props["nodeValue"].(string)
					}
					needsUpdate = oldText != newText
				} else {
					needsUpdate = !propsEqual(matchedOld.props, elem.Props)
				}
			}

			effectTag := "UPDATE"
			if !needsUpdate {
				effectTag = ""
			}

			newFiber = acquireWorkInProgress(matchedOld)
			*newFiber = Fiber{
				typeOf:         matchedOld.typeOf,
				props:          elem.Props,
				textContent:    elem.TextContent,
				dom:            matchedOld.dom,
				parent:         wipFiber,
				alternate:      matchedOld,
				effectTag:      effectTag,
				dirty:          needsUpdate,
				hooks:          matchedOld.hooks,
				eventCallbacks: matchedOld.eventCallbacks,
			}
		} else {
			if matchedOld != nil {
				matchedOld.effectTag = "DELETION"
				rt.deletions = append(rt.deletions, matchedOld)
			}

			newFiber = acquireWorkInProgress(nil)
			*newFiber = Fiber{
				typeOf:      elem.Type,
				props:       elem.Props,
				textContent: elem.TextContent,
				parent:      wipFiber,
				effectTag:   "PLACEMENT",
				dirty:       true,
			}
		}

		if !firstChildSet {
			wipFiber.child = newFiber
			firstChildSet = true
		} else if prevSibling != nil {
			prevSibling.sibling = newFiber
		}
		prevSibling = newFiber
	}

	for _, oldFiber := range oldByKey {
		oldFiber.effectTag = "DELETION"
		rt.deletions = append(rt.deletions, oldFiber)
	}
	for _, oldFiber := range oldFallbackKeyed {
		if oldFiber == nil {
			continue
		}
		oldFiber.effectTag = "DELETION"
		rt.deletions = append(rt.deletions, oldFiber)
	}
	for ; unkeyedIndex < len(oldUnkeyed); unkeyedIndex++ {
		oldFiber := oldUnkeyed[unkeyedIndex]
		oldFiber.effectTag = "DELETION"
		rt.deletions = append(rt.deletions, oldFiber)
	}
}

func sameFiberType(elem *Element, oldFiber *Fiber) bool {
	if elem == nil || oldFiber == nil {
		return false
	}

	if s1, ok1 := elem.Type.(string); ok1 {
		if s2, ok2 := oldFiber.typeOf.(string); ok2 {
			return s1 == s2
		}
	}

	return isSameType(elem.Type, oldFiber.typeOf)
}

func hasElementKey(elem *Element) bool {
	if elem == nil || elem.Props == nil {
		return false
	}
	_, ok := elem.Props["key"]
	return ok
}

func hasFiberKey(fiber *Fiber) bool {
	if fiber == nil || fiber.props == nil {
		return false
	}
	_, ok := fiber.props["key"]
	return ok
}

func elementComparableKey(elem *Element) (interface{}, bool) {
	if elem == nil || elem.Props == nil {
		return nil, false
	}
	return propsComparableKey(elem.Props)
}

func fiberComparableKey(fiber *Fiber) (interface{}, bool) {
	if fiber == nil || fiber.props == nil {
		return nil, false
	}
	return propsComparableKey(fiber.props)
}

func propsComparableKey(props map[string]interface{}) (interface{}, bool) {
	key, ok := props["key"]
	if !ok || key == nil {
		return nil, false
	}
	switch typed := key.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, bool:
		return typed, true
	case *Element, *Fiber:
		return typed, true
	}
	t := reflect.TypeOf(key)
	if t == nil || !t.Comparable() {
		return nil, false
	}
	return key, true
}

func takeMatchingFallbackKeyed(oldFibers []*Fiber, elem *Element) *Fiber {
	if elem == nil || elem.Props == nil {
		return nil
	}
	key := elem.Props["key"]
	for index, oldFiber := range oldFibers {
		if oldFiber == nil || oldFiber.props == nil {
			continue
		}
		if fastEqual(oldFiber.props["key"], key) {
			oldFibers[index] = nil
			return oldFiber
		}
	}
	return nil
}

// propsEqual compares two property maps for equality
func propsEqual(a, b map[string]interface{}) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}

	// Fast path: different lengths
	aLen := len(a)
	if aLen != len(b) {
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

			// Fast path for the common children representation.
			if c1, ok1 := v1.([]interface{}); ok1 {
				if c2, ok2 := v2.([]interface{}); ok2 {
					if len(c1) == len(c2) {
						if len(c1) == 0 {
							continue
						}
						if &c1[0] == &c2[0] {
							continue
						}
					}
					return false
				}
			}

			// Fallback for non-standard slice types.
			t1 := reflect.TypeOf(v1)
			t2 := reflect.TypeOf(v2)
			if t1 != nil && t2 != nil && t1.Kind() == reflect.Slice && t2.Kind() == reflect.Slice {
				rv1 := reflect.ValueOf(v1)
				rv2 := reflect.ValueOf(v2)
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

	if v1.Type().Comparable() {
		return type1 == type2
	}

	return reflect.DeepEqual(type1, type2)
}

// isFiberDirty checks if a fiber or any of its alternates are dirty
func (rt *Runtime) isFiberDirty(fiber *Fiber) bool {
	if fiber == nil {
		return false
	}

	if fiber.dirty {
		return true
	}

	// Child fibers are reused as current/work-in-progress pairs.
	// Avoid walking an alternate cycle indefinitely.
	alternate := fiber.alternate
	return alternate != nil && alternate != fiber && alternate.dirty
}

// clearFiberDirty clears the dirty flag on a fiber and its alternate pair.
func (rt *Runtime) clearFiberDirty(fiber *Fiber) {
	if fiber == nil {
		return
	}

	fiber.dirty = false

	alternate := fiber.alternate
	if alternate != nil && alternate != fiber {
		alternate.dirty = false
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
		if fiber.hooks != nil {
			fiber.hooks.owner = fiber
		}
		rt.cloneChildFibers(fiber)
		return rt.getNextUnitOfWork(fiber)
	}

	// Clear dirty flags on fiber and alternates
	rt.clearFiberDirty(fiber)

	if fiber.contextValues == nil && fiber.parent != nil {
		fiber.contextValues = fiber.parent.contextValues
	}

	if fiber.typeOf == nil || fiber.typeOf == "ROOT" {
		// Root fiber - reconcile children
		if children, ok := fiber.props["children"].([]interface{}); ok {
			rt.reconcileChildren(fiber, children)
		}
	} else {
		switch typed := fiber.typeOf.(type) {
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

		case *ContextProviderType:
			value := typed.Descriptor.DefaultValue
			if fiber.props != nil {
				if provided, ok := fiber.props["value"]; ok {
					value = provided
				}
			}

			var parentContextValues map[int64]interface{}
			if fiber.parent != nil {
				parentContextValues = fiber.parent.contextValues
			}
			fiber.contextValues = deriveContextValues(parentContextValues, typed.Descriptor.ID, value)

			if fiber.alternate != nil && !fastEqual(resolveContextValue(fiber.alternate, typed.Descriptor), value) {
				markSubtreeNeedsUpdate(fiber.alternate.child)
			}

			if propsChildren, ok := fiber.props["children"]; ok {
				if elements, elementsOk := propsChildren.([]interface{}); elementsOk {
					rt.reconcileChildren(fiber, elements)
					break
				}
			}
			rt.reconcileChildren(fiber, emptyChildren)

		case *PortalElementType:
			if propsChildren, ok := fiber.props["children"]; ok {
				if elements, elementsOk := propsChildren.([]interface{}); elementsOk {
					rt.reconcileChildren(fiber, elements)
					break
				}
			}
			rt.reconcileChildren(fiber, emptyChildren)

		default:
			// Function component
			currentFiber = fiber
			// Preserve hooks from alternate fiber or initialize new hooks
			if fiber.alternate != nil && fiber.alternate.hooks != nil {
				// Reuse the hooks struct to avoid allocations and preserve closures
				fiber.hooks = fiber.alternate.hooks
				fiber.hooks.owner = fiber

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
			} else {
				// Initialize hooks for first render
				fiber.hooks = &Hooks{owner: fiber}
			}

			// Clear effects
			if fiber.effects != nil {
				fiber.effects = fiber.effects[:0]
			} else {
				fiber.effects = make([]Effect, 0)
			}
			var element *Element
			if fn, ok := fiber.typeOf.(func() *Element); ok {
				element = fn()
			} else if fn, ok := fiber.typeOf.(func(map[string]interface{}) *Element); ok {
				element = fn(fiber.props)
			} else if fn, ok := fiber.typeOf.(func(Attrs) *Element); ok {
				element = fn(Attrs(fiber.props))
			}

			if element != nil {
				children := [1]interface{}{element}
				rt.reconcileChildren(fiber, children[:])
			} else {
				// A component that now renders nothing must delete any previous subtree.
				rt.reconcileChildren(fiber, emptyChildren)
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
	if _, ok := fiber.typeOf.(*PortalElementType); ok {
		return nil
	}
	// Function components don't have DOM nodes - they render their children

	return dom
}

// updateDomProperties updates DOM properties with optimized batching when available
func (rt *Runtime) updateDomProperties(dom DOMNode, oldProps, newProps map[string]interface{}) {
	// Check if dom is nil (interface is nil) or if the concrete value is null
	if dom == nil || dom.IsNull() {
		return
	}

	// fmt.Printf("updateDomProperties: updating %d old props, %d new props\n", len(oldProps), len(newProps))

	// Check if adapter supports batching (only for WASM adapter)
	batchAdapter, supportsBatching := rt.domAdapter.(interface {
		BatchSetAttributes(DOMNode, map[string]string)
	})

	// Optimization: Fast path for initial render (no old props)
	if len(oldProps) == 0 && len(newProps) > 0 {
		// If batching is supported, collect attributes
		var attrBatch map[string]string
		flushAttrBatch := func() {
			if !supportsBatching || len(attrBatch) == 0 {
				return
			}
			batchAdapter.BatchSetAttributes(dom, attrBatch)
			clear(attrBatch)
		}

		for name, value := range newProps {
			meta := getPropMeta(name)
			if meta.kind == propKindSkip {
				continue
			}

			switch meta.kind {
			case propKindStyle:
				// Flush batch before style
				flushAttrBatch()
				if styles, ok := value.(map[string]string); ok {
					rt.domAdapter.SetStyles(dom, styles)
				} else if str, ok := value.(string); ok {
					if supportsBatching {
						if attrBatch == nil {
							attrBatch = make(map[string]string, len(newProps))
						}
						attrBatch[meta.attrName] = str
					} else {
						rt.domAdapter.SetAttribute(dom, meta.attrName, str)
					}
				}
			case propKindClass:
				if str, ok := value.(string); ok {
					if supportsBatching {
						if attrBatch == nil {
							attrBatch = make(map[string]string, len(newProps))
						}
						attrBatch[meta.attrName] = str
					} else {
						rt.domAdapter.SetAttribute(dom, meta.attrName, str)
					}
				}
			case propKindSpecialProperty:
				// Flush batch before property
				flushAttrBatch()
				rt.domAdapter.SetProperty(dom, name, value)
			default:
				if str, ok := value.(string); ok {
					if supportsBatching {
						if attrBatch == nil {
							attrBatch = make(map[string]string, len(newProps))
						}
						attrBatch[meta.attrName] = str
					} else {
						rt.domAdapter.SetAttribute(dom, meta.attrName, str)
					}
				} else {
					// Flush batch before property
					flushAttrBatch()
					rt.domAdapter.SetProperty(dom, name, value)
				}
			}
		}

		// Flush remaining batched attributes
		flushAttrBatch()
		return
	}

	// Remove old properties
	for name := range oldProps {
		meta := getPropMeta(name)
		if meta.kind == propKindSkip {
			continue
		}
		if _, exists := newProps[name]; !exists {
			if meta.shouldReset {
				rt.domAdapter.SetProperty(dom, name, meta.resetValue)
			} else {
				rt.domAdapter.RemoveAttribute(dom, meta.attrName)
			}
		}
	}

	// Set new properties
	for name, value := range newProps {
		meta := getPropMeta(name)
		if meta.kind == propKindSkip {
			continue
		}

		// Optimization: Skip if value hasn't changed
		if oldValue, exists := oldProps[name]; exists && fastEqual(oldValue, value) {
			continue
		}

		switch meta.kind {
		case propKindStyle:
			if styles, ok := value.(map[string]string); ok {
				rt.domAdapter.SetStyles(dom, styles)
			} else if str, ok := value.(string); ok {
				// Allow inline style strings as attribute values
				rt.domAdapter.SetAttribute(dom, meta.attrName, str)
			}
		case propKindClass:
			if str, ok := value.(string); ok {
				rt.domAdapter.SetAttribute(dom, meta.attrName, str)
			}
		case propKindSpecialProperty:
			// Always set these as properties to ensure UI updates correctly
			rt.domAdapter.SetProperty(dom, name, value)
		default:
			if str, ok := value.(string); ok {
				rt.domAdapter.SetAttribute(dom, meta.attrName, str)
			} else {
				// Always update properties (especially event handlers which are closures)
				rt.domAdapter.SetProperty(dom, name, value)
			}
		}
	}
}

// commitRoot commits all changes to the DOM
func (rt *Runtime) commitRoot() {
	start := time.Now()
	defer func() {
		rt.profiling.commitCount++
		rt.profiling.lastCommitDurationNs = time.Since(start).Nanoseconds()
	}()
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

func reportMissingKeys(parent *Fiber, elements []interface{}) {
	renderableCount := 0
	missingKeyCount := 0
	hasKeyedSibling := false
	for _, element := range elements {
		elem, ok := element.(*Element)
		if !ok || elem == nil {
			continue
		}
		renderableCount++
		if hasElementKey(elem) {
			hasKeyedSibling = true
			continue
		}
		missingKeyCount++
	}

	if renderableCount <= 1 || missingKeyCount == 0 {
		return
	}

	if !hasKeyedSibling {
		for oldFiber := parentChild(parent); oldFiber != nil; oldFiber = oldFiber.sibling {
			if hasFiberKey(oldFiber) {
				hasKeyedSibling = true
				break
			}
		}
	}

	if !hasKeyedSibling {
		return
	}

	_, parentName := describeFiber(parent)
	ReportDiagnostic("runtime", DiagnosticWarning, "missing key on one or more sibling elements under "+parentName)
}

func parentChild(parent *Fiber) *Fiber {
	if parent == nil || parent.alternate == nil {
		return nil
	}
	return parent.alternate.child
}

// commitWork commits a fiber's changes to the DOM with batch optimization
func (rt *Runtime) commitWork(fiber *Fiber, domParent DOMNode) {
	if fiber == nil {
		return
	}

	// Fast path: most calls have valid domParent
	if domParent == nil || domParent.IsNull() {
		var domParentFiber *Fiber = fiber.parent
		for domParentFiber != nil && (domParentFiber.dom == nil || domParentFiber.dom.IsNull()) {
			domParentFiber = domParentFiber.parent
		}
		if domParentFiber != nil {
			domParent = domParentFiber.dom
		}
	}

	isPortal := rt.isPortalFiber(fiber)
	portalParent := domParent
	portalTargetChanged := false
	if isPortal {
		portalParent = rt.resolvePortalParent(fiber)
		portalTargetChanged = rt.portalTargetChanged(fiber)
	}

	if !isPortal && domParent != nil && !domParent.IsNull() {
		if fiber.effectTag == "PLACEMENT" && fiber.dom != nil && !fiber.dom.IsNull() {
			start := time.Now()
			rt.domAdapter.AppendChild(domParent, fiber.dom)
			fiber.commitDurationNs += time.Since(start).Nanoseconds()
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
						start := time.Now()
						rt.domAdapter.SetTextContent(fiber.dom, newValue)
						fiber.commitDurationNs += time.Since(start).Nanoseconds()
					}
				} else {
					// Regular element - update properties
					start := time.Now()
					rt.updateDomProperties(fiber.dom, fiber.alternate.props, fiber.props)
					fiber.commitDurationNs += time.Since(start).Nanoseconds()
				}
			}
		} else if fiber.effectTag == "DELETION" {
			// fmt.Printf("DEBUG: Committing deletion for %v\n", fiber.typeOf)
			rt.commitDeletion(fiber, domParent)
			return
		}
	}

	if isPortal {
		if fiber.effectTag == "DELETION" {
			rt.commitDeletion(fiber, portalParent)
			return
		}
		if portalTargetChanged {
			rt.movePortalSubtree(fiber.child, rt.resolvePortalParent(fiber.alternate), portalParent)
		}
		if portalParent == nil || portalParent.IsNull() {
			if fiber.alternate != nil {
				rt.deleteFiberSubtree(fiber.child, rt.resolvePortalParent(fiber.alternate))
			}
			if fiber.sibling != nil {
				rt.commitWork(fiber.sibling, domParent)
			}
			return
		}
	}

	// Determine the parent DOM node for children
	// If this fiber has a DOM node, it becomes the parent for its children
	childDomParent := domParent
	if isPortal {
		childDomParent = portalParent
	}
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

	if rt.isPortalFiber(fiber) {
		rt.deleteFiberSubtree(fiber.child, rt.resolvePortalParent(fiber))
		return
	}

	if fiber.dom != nil && !fiber.dom.IsNull() {
		// This fiber has a DOM node, remove it
		rt.domAdapter.RemoveChild(domParent, fiber.dom)
	} else {
		// Function component without DOM node - recursively delete all descendants
		// We need to find and remove all actual DOM nodes in the subtree
		rt.deleteFiberSubtree(fiber.child, domParent)
	}
}

// deleteFiberSubtree recursively removes all DOM nodes in a fiber's subtree
func (rt *Runtime) deleteFiberSubtree(fiber *Fiber, domParent DOMNode) {
	if fiber == nil {
		return
	}

	// If this fiber has a DOM node, remove it (this stops the recursion down that branch)
	if fiber.dom != nil && !fiber.dom.IsNull() {
		rt.domAdapter.RemoveChild(domParent, fiber.dom)
	} else {
		// No DOM node on this fiber - recurse to children until we find DOM nodes.
		rt.deleteFiberSubtree(fiber.child, domParent)
	}

	// Continue across sibling branches so DOM-less parents remove their full subtree.
	rt.deleteFiberSubtree(fiber.sibling, domParent)
}

func (rt *Runtime) isPortalFiber(fiber *Fiber) bool {
	if fiber == nil {
		return false
	}
	_, ok := fiber.typeOf.(*PortalElementType)
	return ok
}

func (rt *Runtime) resolvePortalParent(fiber *Fiber) DOMNode {
	if fiber == nil || fiber.props == nil {
		return nil
	}

	if rawNode, ok := fiber.props["portalTargetNode"]; ok && rawNode != nil {
		if node, ok := rawNode.(DOMNode); ok {
			return node
		}
		if resolver, ok := rt.domAdapter.(interface{ ResolveNode(interface{}) DOMNode }); ok {
			return resolver.ResolveNode(rawNode)
		}
	}

	if selector, ok := fiber.props["portalTargetSelector"].(string); ok && selector != "" {
		return rt.queryContainer(selector)
	}

	return nil
}

func (rt *Runtime) portalTargetChanged(fiber *Fiber) bool {
	if fiber == nil || fiber.alternate == nil {
		return false
	}
	return !fastEqual(fiber.props["portalTargetSelector"], fiber.alternate.props["portalTargetSelector"]) || !fastEqual(fiber.props["portalTargetNode"], fiber.alternate.props["portalTargetNode"])
}

func (rt *Runtime) movePortalSubtree(fiber *Fiber, oldParent, newParent DOMNode) {
	if fiber == nil || oldParent == nil || oldParent.IsNull() || newParent == nil || newParent.IsNull() || oldParent.Equals(newParent) {
		return
	}

	if rt.isPortalFiber(fiber) {
		return
	}

	if fiber.dom != nil && !fiber.dom.IsNull() {
		rt.domAdapter.RemoveChild(oldParent, fiber.dom)
		rt.domAdapter.AppendChild(newParent, fiber.dom)
	} else {
		rt.movePortalSubtree(fiber.child, oldParent, newParent)
	}

	rt.movePortalSubtree(fiber.sibling, oldParent, newParent)
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
				start := time.Now()
				cleanup()
				durationNs := time.Since(start).Nanoseconds()
				fiber.cleanupDurationNs += durationNs
				rt.profiling.cleanupExecutions++
				rt.profiling.lastCleanupDurationNs = durationNs
				recordSlowOperationDiagnostic("cleanup", fiber, durationNs)
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

	// Run this fiber's effects in batch
	effects := fiber.effects
	effectCount := len(effects)
	fiber.effectDurationNs = 0

	// Unroll for common small effect counts
	if effectCount == 1 {
		start := time.Now()
		cleanup := effects[0].Fn()
		durationNs := time.Since(start).Nanoseconds()
		fiber.effectDurationNs += durationNs
		rt.profiling.effectExecutions++
		rt.profiling.lastEffectDurationNs = durationNs
		recordSlowOperationDiagnostic("effect", fiber, durationNs)
		if cleanup != nil {
			fiber.hooks.cleanups[effects[0].CleanupIndex] = cleanup
		}
	} else {
		for i := 0; i < effectCount; i++ {
			effect := &effects[i]
			start := time.Now()
			cleanup := effect.Fn()
			durationNs := time.Since(start).Nanoseconds()
			fiber.effectDurationNs += durationNs
			rt.profiling.effectExecutions++
			rt.profiling.lastEffectDurationNs = durationNs
			recordSlowOperationDiagnostic("effect", fiber, durationNs)
			if cleanup != nil {
				fiber.hooks.cleanups[effect.CleanupIndex] = cleanup
			}
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
