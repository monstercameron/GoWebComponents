package runtime

import "testing"

// Deleted fibers are stripped of their references at the end of commitDeletion
// (releaseDeletedSubtree), which is what stops a slab slot from keeping an
// unmounted subtree's props, hooks and DOM handles reachable. These tests pin
// both halves of that: the references really are dropped, and dropping them
// does not break the accessors that outlive the fiber.

// TestDeletedSubtreeReleasesItsReferences fails if a deleted fiber still points
// at its subtree, its DOM node, or its hooks — the state that made a mostly
// unmounted slab retain everything it had ever held.
func TestDeletedSubtreeReleasesItsReferences(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("section")

	parseHost := func(isIncludeRow bool) *Element {
		if !isIncludeRow {
			// A different type forces the old subtree to be tagged DELETION.
			return CreateElement("div", map[string]any{"id": "host"},
				CreateElement("span", map[string]any{"id": "placeholder"}, "x"))
		}
		return CreateElement("div", map[string]any{"id": "host"},
			CreateElement("div", map[string]any{"id": "row"},
				CreateElement("em", map[string]any{"id": "cell"}, "text")))
	}

	parseRt.Render(parseHost(true), parseContainer)

	parseRow := findFiberByProp(parseRt.currentRoot, "id", "row")
	if parseRow == nil {
		parseT.Fatal("setup: the row fiber was never mounted")
	}
	parseCell := parseRow.child
	if parseCell == nil || parseRow.props == nil || IsDOMNodeNull(parseRow.dom) {
		parseT.Fatal("setup: the mounted row should carry a child, props, and a DOM node")
	}

	parseRt.Render(parseHost(false), parseContainer)

	if parseRow.child != nil {
		parseT.Error("deleted fiber still references its child subtree")
	}
	if parseRow.parent != nil {
		parseT.Error("deleted fiber still references its parent")
	}
	if parseRow.alternate != nil {
		parseT.Error("deleted fiber still references its alternate")
	}
	if !IsDOMNodeNull(parseRow.dom) {
		parseT.Error("deleted fiber still references its DOM node")
	}
	if parseRow.props != nil {
		parseT.Error("deleted fiber still references its props map")
	}
	if parseRow.hooks != nil {
		parseT.Error("deleted fiber still references its hooks")
	}
	// The release walks the whole deleted subtree, not just its root.
	if parseCell.props != nil || !IsDOMNodeNull(parseCell.dom) {
		parseT.Error("a deleted DESCENDANT kept its props or DOM node — the release did not walk the subtree")
	}
}

// TestSetterAfterUnmountIsASafeNoOp is the counterweight: hook accessors capture
// the *Hooks object rather than the fiber, so an async callback that resolves
// after its component unmounted still holds a live setter. Releasing the fiber
// must leave that setter safe to call — no panic, and no resurrection of the
// unmounted component.
func TestSetterAfterUnmountIsASafeNoOp(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("section")

	var parseSet func(any)
	parseChild := func(parseProps map[string]any) *Element {
		parseValue, parseSetter := GoUseState(parseRt, "before")
		parseSet = parseSetter
		return CreateElement("div", map[string]any{"id": "child"}, parseValue())
	}
	parseHost := func(isIncludeChild bool) *Element {
		if !isIncludeChild {
			return CreateElement("div", map[string]any{"id": "host"},
				CreateElement("span", map[string]any{"id": "placeholder"}, "x"))
		}
		return CreateElement("div", map[string]any{"id": "host"},
			CreateElement(parseChild, nil))
	}

	parseRt.Render(parseHost(true), parseContainer)
	if parseSet == nil {
		parseT.Fatal("setup: the child never rendered")
	}

	parseRt.Render(parseHost(false), parseContainer)
	if findFiberByProp(parseRt.currentRoot, "id", "child") != nil {
		parseT.Fatal("setup: the child should be unmounted")
	}

	// The late write. This is the shape of every async callback that outlives
	// its component: a worker reply, a resolved fetch, a form validator.
	parseSet("after")

	if findFiberByProp(parseRt.currentRoot, "id", "child") != nil {
		parseT.Error("a setter called after unmount remounted the component")
	}
}

// findFiberByProp walks the committed tree for the first fiber carrying a prop.
func findFiberByProp(parseFiber *Fiber, parseName string, parseValue string) *Fiber {
	if parseFiber == nil {
		return nil
	}
	if parseFiber.props != nil {
		if parseFound, parseOk := parseFiber.props[parseName].(string); parseOk && parseFound == parseValue {
			return parseFiber
		}
	}
	if parseHit := findFiberByProp(parseFiber.child, parseName, parseValue); parseHit != nil {
		return parseHit
	}
	return findFiberByProp(parseFiber.sibling, parseName, parseValue)
}
