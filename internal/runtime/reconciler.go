package runtime

import (
	"reflect"
	"strings"
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
		"htmlFor":   {kind: propKindDefault, attrName: "for"},
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
	if strings.HasPrefix(name, "__gwc_prop__:") {
		return domPropMeta{
			kind:        propKindSpecialProperty,
			attrName:    strings.TrimPrefix(name, "__gwc_prop__:"),
			resetValue:  nil,
			shouldReset: true,
		}
	}
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

func ensureFineGrainedTwinLink(oldFiber *Fiber, newFiber *Fiber) {
	if oldFiber == nil || newFiber == nil {
		return
	}
	if oldFiber.fineGrained || newFiber.fineGrained {
		oldFiber.alternate = newFiber
	}
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
		effectTag := ""
		if oldFiber.dirty || oldFiber.needsUpdate {
			effectTag = "UPDATE"
		}
		newFiber := acquireWorkInProgress(oldFiber)
		*newFiber = Fiber{
			typeOf:            oldFiber.typeOf,
			props:             oldFiber.props,
			textContent:       oldFiber.textContent,
			dom:               oldFiber.dom,
			parent:            parent,
			alternate:         oldFiber,
			effectTag:         effectTag,
			dirty:             oldFiber.dirty,
			needsUpdate:       oldFiber.needsUpdate,
			hooks:             oldFiber.hooks, // Share hooks for non-updated components
			eventCallbacks:    oldFiber.eventCallbacks,
			contextValues:     oldFiber.contextValues,
			reactiveAtomID:    oldFiber.reactiveAtomID,
			reactiveSourceIDs: oldFiber.reactiveSourceIDs,
			fineGrained:       oldFiber.fineGrained,
			updateOrigin:      oldFiber.updateOrigin,
		}
		if newFiber.hooks != nil {
			newFiber.hooks.owner = newFiber
		}
		ensureFineGrainedTwinLink(oldFiber, newFiber)
		if rt.atomRegistry != nil && newFiber.fineGrained && len(newFiber.reactiveSourceIDs) > 0 {
			if rt.hydrating {
				for _, sourceID := range newFiber.reactiveSourceIDs {
					rt.queueHydrationSubscription(sourceID, oldFiber, false)
					rt.queueHydrationSubscription(sourceID, newFiber, true)
				}
			}
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
						typeOf:            oldFiber.typeOf,
						props:             elem.Props,
						textContent:       elem.TextContent,
						dom:               oldFiber.dom,
						parent:            wipFiber,
						alternate:         oldFiber,
						effectTag:         effectTag,
						dirty:             needsUpdate,
						needsUpdate:       oldFiber.needsUpdate,
						hooks:             oldFiber.hooks,
						eventCallbacks:    oldFiber.eventCallbacks,
						hydration:         wipFiber.childHydration,
						reactiveAtomID:    oldFiber.reactiveAtomID,
						reactiveSourceIDs: oldFiber.reactiveSourceIDs,
						fineGrained:       oldFiber.fineGrained,
						updateOrigin:      oldFiber.updateOrigin,
					}
					ensureFineGrainedTwinLink(oldFiber, newFiber)

					// Advance oldFiber
					oldFiber = oldFiber.sibling
				} else {
					// REPLACE logic (Placement + Deletion)
					newFiber = acquireWorkInProgress(oldFiber)
					*newFiber = Fiber{
						typeOf:       elem.Type,
						props:        elem.Props,
						textContent:  elem.TextContent,
						parent:       wipFiber,
						effectTag:    "PLACEMENT",
						dirty:        true,
						hydration:    wipFiber.childHydration,
						fineGrained:  isFineGrainedType(elem.Type),
						updateOrigin: oldFiberUpdateOrigin(oldFiber, elem.Type),
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
					typeOf:       elem.Type,
					props:        elem.Props,
					textContent:  elem.TextContent,
					parent:       wipFiber,
					effectTag:    "PLACEMENT",
					dirty:        true,
					hydration:    wipFiber.childHydration,
					fineGrained:  isFineGrainedType(elem.Type),
					updateOrigin: oldFiberUpdateOrigin(nil, elem.Type),
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
				typeOf:            matchedOld.typeOf,
				props:             elem.Props,
				textContent:       elem.TextContent,
				dom:               matchedOld.dom,
				parent:            wipFiber,
				alternate:         matchedOld,
				effectTag:         effectTag,
				dirty:             needsUpdate,
				needsUpdate:       matchedOld.needsUpdate,
				hooks:             matchedOld.hooks,
				eventCallbacks:    matchedOld.eventCallbacks,
				hydration:         wipFiber.childHydration,
				reactiveAtomID:    matchedOld.reactiveAtomID,
				reactiveSourceIDs: matchedOld.reactiveSourceIDs,
				fineGrained:       matchedOld.fineGrained,
				updateOrigin:      matchedOld.updateOrigin,
			}
			ensureFineGrainedTwinLink(matchedOld, newFiber)
		} else {
			if matchedOld != nil {
				matchedOld.effectTag = "DELETION"
				rt.deletions = append(rt.deletions, matchedOld)
			}

			newFiber = acquireWorkInProgress(nil)
			*newFiber = Fiber{
				typeOf:       elem.Type,
				props:        elem.Props,
				textContent:  elem.TextContent,
				parent:       wipFiber,
				effectTag:    "PLACEMENT",
				dirty:        true,
				hydration:    wipFiber.childHydration,
				fineGrained:  isFineGrainedType(elem.Type),
				updateOrigin: oldFiberUpdateOrigin(nil, elem.Type),
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

	if component1, ok1 := type1.(*ComponentType); ok1 {
		component2, ok2 := type2.(*ComponentType)
		if !ok2 {
			return false
		}
		return component1.IdentityKey() != "" && component1.IdentityKey() == component2.IdentityKey()
	}

	v1 := reflect.ValueOf(type1)
	v2 := reflect.ValueOf(type2)

	if !v1.IsValid() || !v2.IsValid() {
		return false
	}

	if v1.Kind() == reflect.Func && v2.Kind() == reflect.Func {
		return sameFunctionIdentity(type1, type2)
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
	if fiber.hydration == nil && fiber.parent != nil {
		fiber.hydration = fiber.parent.childHydration
	}

	if fiber.typeOf == nil || fiber.typeOf == "ROOT" {
		// Root fiber - reconcile children
		fiber.childHydration = fiber.hydration
		if children, ok := fiber.props["children"].([]interface{}); ok {
			rt.reconcileChildren(fiber, children)
		}
	} else {
		switch typed := fiber.typeOf.(type) {
		case string:
			// Host component (HTML element)
			if typed == "FRAGMENT" {
				fiber.childHydration = fiber.hydration
			} else if fiber.dom == nil || fiber.dom.IsNull() {
				if hydratedDOM, ok := rt.claimHydrationNode(fiber); ok {
					fiber.dom = hydratedDOM
					fiber.hydrated = true
					fiber.effectTag = "HYDRATE"
					fiber.childHydration = newHydrationBoundary(hydratedDOM, rt.domAdapter.GetFirstChild(hydratedDOM))
				} else {
					fiber.dom = rt.createDom(fiber)
					fiber.hydrated = false
					fiber.childHydration = nil
				}
			} else if typed != "TEXT_ELEMENT" {
				fiber.childHydration = nil
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
			fiber.childHydration = fiber.hydration

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

		case *ReactiveTextElementType:
			rt.syncReactiveTextSubscription(fiber)
			fiber.textContent = reactiveTextValue(fiber)
			if fiber.dom == nil || fiber.dom.IsNull() {
				if hydratedDOM, ok := rt.claimHydrationNode(fiber); ok {
					fiber.dom = hydratedDOM
					fiber.hydrated = true
					fiber.effectTag = "HYDRATE"
				} else {
					fiber.dom = rt.createDom(fiber)
					fiber.hydrated = false
				}
			}
			fiber.childHydration = nil

		case *ReactiveRegionElementType:
			rt.syncReactiveRegionSubscription(fiber)
			fiber.childHydration = fiber.hydration
			rendered := reactiveRegionValue(fiber)
			if rendered != nil {
				children := [1]interface{}{rendered}
				rt.reconcileChildren(fiber, children[:])
			} else {
				rt.reconcileChildren(fiber, emptyChildren)
			}

		case *ErrorBoundaryType:
			rt.renderBoundaryChildren(fiber)

		default:
			fiber.childHydration = fiber.hydration
			element, handledPanic, nextFromBoundary := rt.renderFunctionComponent(fiber)
			if handledPanic {
				return nextFromBoundary
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
			text := fiber.textContent
			if text == "" && fiber.props != nil {
				text, _ = fiber.props["nodeValue"].(string)
			}
			dom = rt.domAdapter.CreateTextNode(text)
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
	if _, ok := fiber.typeOf.(*ReactiveRegionElementType); ok {
		return nil
	}
	if _, ok := fiber.typeOf.(*ReactiveTextElementType); ok {
		return rt.domAdapter.CreateTextNode(reactiveTextValue(fiber))
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
		rt.applyInitialDomProps(dom, newProps, supportsBatching, batchAdapter, false)
		return
	}

	// Remove old properties
	for name := range oldProps {
		meta := getPropMeta(name)
		if meta.kind == propKindSkip {
			continue
		}
		targetName := meta.attrName
		if targetName == "" {
			targetName = name
		}
		if _, exists := newProps[name]; !exists {
			if meta.shouldReset {
				rt.domAdapter.SetProperty(dom, targetName, meta.resetValue)
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
		targetName := meta.attrName
		if targetName == "" {
			targetName = name
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
			rt.domAdapter.SetProperty(dom, targetName, value)
		default:
			if str, ok := value.(string); ok {
				rt.domAdapter.SetAttribute(dom, meta.attrName, str)
			} else {
				// Always update properties (especially event handlers which are closures)
				rt.domAdapter.SetProperty(dom, targetName, value)
			}
		}
	}
}

func (rt *Runtime) applyInitialDomProps(dom DOMNode, newProps map[string]interface{}, supportsBatching bool, batchAdapter interface {
	BatchSetAttributes(DOMNode, map[string]string)
}, preserveHydrationState bool) {
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
		targetName := meta.attrName
		if targetName == "" {
			targetName = name
		}
		if preserveHydrationState && shouldPreserveHydrationInitialProperty(targetName) {
			continue
		}

		switch meta.kind {
		case propKindStyle:
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
			flushAttrBatch()
			rt.domAdapter.SetProperty(dom, targetName, value)
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
				flushAttrBatch()
				rt.domAdapter.SetProperty(dom, targetName, value)
			}
		}
	}

	flushAttrBatch()
}

func shouldPreserveHydrationInitialProperty(name string) bool {
	switch name {
	case "value", "checked", "selected", "autofocus":
		return true
	default:
		return false
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
	committedRoot := rt.wipRoot
	if rt.wipRoot != nil && rt.wipRoot.child != nil {
		rt.finalizeHydrationBoundary(rt.wipRoot.childHydration, rt.wipRoot)
		// The root fiber's DOM node is the container
		rt.commitWork(rt.wipRoot.child, rt.wipRoot.dom)
	} else {
		// fmt.Printf("[COMMIT] WARNING: wipRoot.child is nil\n")
	}

	rt.currentRoot = committedRoot
	rt.wipRoot = nil
	wasHydrating := rt.hydrating
	if wasHydrating {
		rt.hydrating = false
		rt.strictHydration = false
		rt.flushHydrationSubscriptions()
	}
	rt.updateScheduled = false

	// Run effects after the committed tree is current and hydration gates are lifted.
	rt.runEffects(committedRoot)

	if wasHydrating {
		rt.finishHydrationMetrics(false, "")
		rt.flushDeferredHydrationUpdates()
	}
	if rt.pendingBoundaryRecovery {
		rt.pendingBoundaryRecovery = false
		rt.ScheduleUpdate()
	}
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
			if fiber.fineGrained {
				rt.profiling.fineGrainedCommits++
			} else {
				rt.recordFineGrainedDescendantCommit(fiber)
			}
		} else if fiber.effectTag == "HYDRATE" && fiber.dom != nil && !fiber.dom.IsNull() {
			if isTextLikeFiber(fiber) {
				newValue := fiber.textContent
				if _, ok := fiber.typeOf.(*ReactiveTextElementType); ok {
					newValue = reactiveTextValue(fiber)
				} else if newValue == "" && fiber.props != nil {
					newValue, _ = fiber.props["nodeValue"].(string)
				}
				start := time.Now()
				rt.domAdapter.SetTextContent(fiber.dom, newValue)
				fiber.commitDurationNs += time.Since(start).Nanoseconds()
				if fiber.fineGrained {
					rt.profiling.fineGrainedCommits++
				} else {
					rt.recordFineGrainedDescendantCommit(fiber)
				}
			} else {
				start := time.Now()
				batchAdapter, supportsBatching := rt.domAdapter.(interface {
					BatchSetAttributes(DOMNode, map[string]string)
				})
				rt.applyInitialDomProps(fiber.dom, fiber.props, supportsBatching, batchAdapter, fiber.hydrated)
				fiber.commitDurationNs += time.Since(start).Nanoseconds()
				rt.recordFineGrainedDescendantCommit(fiber)
			}
		} else if fiber.effectTag == "UPDATE" && fiber.dom != nil && !fiber.dom.IsNull() {
			if fiber.alternate != nil {
				if isTextLikeFiber(fiber) {
					// Update text content
					oldValue := textLikeFiberValue(fiber.alternate)
					newValue := textLikeFiberValue(fiber)

					if oldValue != newValue {
						start := time.Now()
						rt.domAdapter.SetTextContent(fiber.dom, newValue)
						fiber.commitDurationNs += time.Since(start).Nanoseconds()
						if fiber.fineGrained {
							rt.profiling.fineGrainedCommits++
						} else {
							rt.recordFineGrainedDescendantCommit(fiber)
						}
					}
				} else {
					// Regular element - update properties
					start := time.Now()
					rt.updateDomProperties(fiber.dom, fiber.alternate.props, fiber.props)
					fiber.commitDurationNs += time.Since(start).Nanoseconds()
					rt.recordFineGrainedDescendantCommit(fiber)
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
	rt.finalizeHydrationBoundary(fiber.childHydration, fiber)

	// Recursively commit children and siblings
	if fiber.child != nil {
		rt.commitWork(fiber.child, childDomParent)
	}
	if fiber.sibling != nil {
		rt.commitWork(fiber.sibling, domParent)
	}
}

func oldFiberUpdateOrigin(oldFiber *Fiber, typeOf interface{}) string {
	if oldFiber != nil && oldFiber.updateOrigin != "" {
		return oldFiber.updateOrigin
	}
	if isFineGrainedType(typeOf) {
		return "fine-grained"
	}
	return ""
}

func isReactiveTextType(typeOf interface{}) bool {
	_, ok := typeOf.(*ReactiveTextElementType)
	return ok
}

func isReactiveRegionType(typeOf interface{}) bool {
	_, ok := typeOf.(*ReactiveRegionElementType)
	return ok
}

func isFineGrainedType(typeOf interface{}) bool {
	return isReactiveTextType(typeOf) || isReactiveRegionType(typeOf)
}

func isHostFiber(fiber *Fiber) bool {
	if fiber == nil {
		return false
	}
	typ, ok := fiber.typeOf.(string)
	if !ok {
		return false
	}
	switch typ {
	case "ROOT", "FRAGMENT", "TEXT_ELEMENT":
		return false
	default:
		return true
	}
}

func hasFineGrainedAncestor(fiber *Fiber) bool {
	for parent := fiber.parent; parent != nil; parent = parent.parent {
		if parent.fineGrained {
			return true
		}
	}
	return false
}

func (rt *Runtime) recordFineGrainedDescendantCommit(fiber *Fiber) {
	if rt == nil || fiber == nil || fiber.fineGrained || !hasFineGrainedAncestor(fiber) {
		return
	}
	if isTextLikeFiber(fiber) {
		rt.profiling.fineGrainedDescendantTextCommits++
		return
	}
	if isHostFiber(fiber) {
		rt.profiling.fineGrainedDescendantHostCommits++
	}
}

func isTextLikeFiber(fiber *Fiber) bool {
	if fiber == nil {
		return false
	}
	if _, ok := fiber.typeOf.(*ReactiveTextElementType); ok {
		return true
	}
	typ, ok := fiber.typeOf.(string)
	return ok && typ == "TEXT_ELEMENT"
}

func textLikeFiberValue(fiber *Fiber) string {
	if fiber == nil {
		return ""
	}
	if _, ok := fiber.typeOf.(*ReactiveTextElementType); ok {
		return fiber.textContent
	}
	value := fiber.textContent
	if value == "" && fiber.props != nil {
		value, _ = fiber.props["nodeValue"].(string)
	}
	return value
}

const (
	reactiveTextAtomIDProp      = "__gwc_reactive_text_atom_id"
	reactiveTextGetterProp      = "__gwc_reactive_text_getter"
	reactiveRegionSourceIDsProp = "__gwc_reactive_region_source_ids"
	reactiveRegionRenderProp    = "__gwc_reactive_region_render"
)

func reactiveTextValue(fiber *Fiber) string {
	if fiber == nil || fiber.props == nil {
		return ""
	}
	getter, _ := fiber.props[reactiveTextGetterProp].(func() string)
	if getter == nil {
		return ""
	}
	return getter()
}

func reactiveRegionValue(fiber *Fiber) *Element {
	if fiber == nil || fiber.props == nil {
		return nil
	}
	render, _ := fiber.props[reactiveRegionRenderProp].(func() *Element)
	if render == nil {
		return nil
	}
	return render()
}

func reactiveRegionSourceIDs(fiber *Fiber) []string {
	if fiber == nil || fiber.props == nil {
		return nil
	}
	raw, _ := fiber.props[reactiveRegionSourceIDsProp].([]string)
	if len(raw) == 0 {
		return nil
	}
	if len(raw) == 1 {
		if strings.TrimSpace(raw[0]) == "" {
			return nil
		}
		return raw
	}
	ids := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, id := range raw {
		if strings.TrimSpace(id) == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
}

func sameReactiveSourceIDs(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func (rt *Runtime) syncFineGrainedSubscriptions(fiber *Fiber, sourceIDs []string) {
	if rt == nil || rt.atomRegistry == nil || fiber == nil {
		return
	}
	previous := fiber.reactiveSourceIDs
	if fiber.alternate != nil && fiber.alternate != fiber && len(fiber.alternate.reactiveSourceIDs) > 0 {
		previous = fiber.alternate.reactiveSourceIDs
	}
	if sameReactiveSourceIDs(previous, sourceIDs) {
		fiber.reactiveSourceIDs = previous
		fiber.reactiveAtomID = ""
		if len(previous) > 0 {
			fiber.reactiveAtomID = strings.Join(previous, ",")
		}
		fiber.fineGrained = len(previous) > 0
		return
	}
	previousSet := make(map[string]struct{}, len(previous))
	for _, id := range previous {
		previousSet[id] = struct{}{}
	}
	nextSet := make(map[string]struct{}, len(sourceIDs))
	for _, id := range sourceIDs {
		nextSet[id] = struct{}{}
	}
	for _, oldID := range previous {
		if _, keep := nextSet[oldID]; keep {
			continue
		}
		if rt.hydrating {
			rt.queueHydrationSubscription(oldID, fiber, false)
			if fiber.alternate != nil && fiber.alternate != fiber {
				rt.queueHydrationSubscription(oldID, fiber.alternate, false)
			}
		} else {
			rt.atomRegistry.Unsubscribe(oldID, fiber)
			if fiber.alternate != nil && fiber.alternate != fiber {
				rt.atomRegistry.Unsubscribe(oldID, fiber.alternate)
			}
		}
	}
	for _, newID := range sourceIDs {
		if _, already := previousSet[newID]; already {
			continue
		}
		if rt.hydrating {
			rt.queueHydrationSubscription(newID, fiber, true)
		} else {
			rt.atomRegistry.Subscribe(newID, fiber)
		}
	}
	fiber.reactiveSourceIDs = append([]string(nil), sourceIDs...)
	fiber.reactiveAtomID = ""
	if len(sourceIDs) > 0 {
		fiber.reactiveAtomID = strings.Join(sourceIDs, ",")
	}
	fiber.fineGrained = len(sourceIDs) > 0
}

func (rt *Runtime) syncReactiveTextSubscription(fiber *Fiber) {
	if rt == nil || fiber == nil || fiber.props == nil {
		return
	}
	atomID, _ := fiber.props[reactiveTextAtomIDProp].(string)
	sourceIDs := []string{}
	if atomID != "" {
		sourceIDs = append(sourceIDs, atomID)
	}
	rt.syncFineGrainedSubscriptions(fiber, sourceIDs)
	if atomID != "" {
		fiber.reactiveAtomID = atomID
	}
}

func (rt *Runtime) syncReactiveRegionSubscription(fiber *Fiber) {
	if rt == nil || fiber == nil {
		return
	}
	rt.syncFineGrainedSubscriptions(fiber, reactiveRegionSourceIDs(fiber))
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
	rt.cleanupAtomSubscriptionsSubtree(fiber)

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

func (rt *Runtime) cleanupAtomSubscriptionsSubtree(fiber *Fiber) {
	if fiber == nil {
		return
	}
	rt.CleanupAtomSubscriptions(fiber)
	for child := fiber.child; child != nil; child = child.sibling {
		rt.cleanupAtomSubscriptionsSubtree(child)
	}
	if fiber.alternate != nil && fiber.alternate != fiber {
		rt.CleanupAtomSubscriptions(fiber.alternate)
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
		for index, cleanup := range fiber.hooks.cleanups {
			if cleanup != nil {
				start := time.Now()
				var handled bool
				func() {
					defer func() {
						if recovered := recover(); recovered != nil {
							if panicPhaseMayRecoverWithBoundary(PanicPhaseCleanup) {
								_, handled = rt.recoverBoundaryError(fiber, recovered, boundaryPhaseCleanup)
							}
							if !handled {
								panic(markUnhandledPanic(fiber, boundaryPhaseCleanup, recovered))
							}
						}
					}()
					cleanup()
				}()
				durationNs := time.Since(start).Nanoseconds()
				fiber.cleanupDurationNs += durationNs
				rt.profiling.cleanupExecutions++
				rt.profiling.lastCleanupDurationNs = durationNs
				recordSlowOperationDiagnostic("cleanup", fiber, durationNs)
				fiber.hooks.cleanups[index] = nil
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

// RefreshEffectsForFiber forces a fiber subtree's effects to clean up and rerun on the next render.
func (rt *Runtime) RefreshEffectsForFiber(fiber *Fiber) {
	if fiber == nil {
		return
	}

	rt.runCleanups(fiber)
	bumpEffectEpochs(fiber)
}

func bumpEffectEpochs(fiber *Fiber) {
	if fiber == nil {
		return
	}

	if fiber.hooks != nil {
		fiber.hooks.effectEpoch++
	}
	if fiber.child != nil {
		bumpEffectEpochs(fiber.child)
	}
	if fiber.sibling != nil {
		bumpEffectEpochs(fiber.sibling)
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
		cleanup := func() func() {
			var cleanup func()
			var handled bool
			func() {
				defer func() {
					if recovered := recover(); recovered != nil {
						if panicPhaseMayRecoverWithBoundary(PanicPhaseEffect) {
							_, handled = rt.recoverBoundaryError(fiber, recovered, boundaryPhaseEffect)
						}
						if !handled {
							panic(markUnhandledPanic(fiber, boundaryPhaseEffect, recovered))
						}
					}
				}()
				cleanup = effects[0].Fn()
			}()
			return cleanup
		}()
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
			cleanup := func() func() {
				var cleanup func()
				var handled bool
				func() {
					defer func() {
						if recovered := recover(); recovered != nil {
							if panicPhaseMayRecoverWithBoundary(PanicPhaseEffect) {
								_, handled = rt.recoverBoundaryError(fiber, recovered, boundaryPhaseEffect)
							}
							if !handled {
								panic(markUnhandledPanic(fiber, boundaryPhaseEffect, recovered))
							}
						}
					}()
					cleanup = effect.Fn()
				}()
				return cleanup
			}()
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
