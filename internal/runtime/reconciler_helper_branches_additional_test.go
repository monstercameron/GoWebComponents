package runtime

import "testing"

// portalTestDOMAdapter extends the test adapter with selector and raw-node resolution for portal helper coverage.
type portalTestDOMAdapter struct {
	*testDOMAdapter
	parseSelectors map[string]DOMNode
	parseResolved  map[string]DOMNode
}

// QuerySelector resolves a selector to a pre-registered DOM node for tests.
func (parseA *portalTestDOMAdapter) QuerySelector(parseSelector string) any {
	return parseA.parseSelectors[parseSelector]
}

// ResolveNode resolves a raw token to a pre-registered DOM node for tests.
func (parseA *portalTestDOMAdapter) ResolveNode(parseValue any) DOMNode {
	parseToken, parseOk := parseValue.(string)
	if !parseOk {
		return nil
	}
	return parseA.parseResolved[parseToken]
}

// TestReconcilerHelpersCoverHostAndReactiveValueBranches verifies host-fiber, reactive value, and source-id normalization helpers.
func TestReconcilerHelpersCoverHostAndReactiveValueBranches(parseT *testing.T) {
	if isHostFiber(nil) {
		parseT.Fatal("expected nil fiber not to be treated as host")
	}
	if isHostFiber(&Fiber{typeOf: func() {}}) {
		parseT.Fatal("expected non-string fiber type not to be treated as host")
	}
	for _, parseType := range []string{"ROOT", "FRAGMENT", "TEXT_ELEMENT"} {
		if isHostFiber(&Fiber{typeOf: parseType}) {
			parseT.Fatalf("expected %s not to be treated as host", parseType)
		}
	}
	if !isHostFiber(&Fiber{typeOf: "div"}) {
		parseT.Fatal("expected div fiber to be treated as host")
	}

	if parseGot := reactiveTextValue(nil); parseGot != "" {
		parseT.Fatalf("reactiveTextValue(nil) = %q, want empty", parseGot)
	}
	if parseGot := reactiveTextValue(&Fiber{props: map[string]any{}}); parseGot != "" {
		parseT.Fatalf("reactiveTextValue(missing getter) = %q, want empty", parseGot)
	}
	if parseGot := reactiveTextValue(&Fiber{props: map[string]any{reactiveTextGetterProp: func() string { return "count:1" }}}); parseGot != "count:1" {
		parseT.Fatalf("reactiveTextValue(getter) = %q, want count:1", parseGot)
	}

	if parseGot := reactiveRegionValue(nil); parseGot != nil {
		parseT.Fatalf("reactiveRegionValue(nil) = %#v, want nil", parseGot)
	}
	if parseGot := reactiveRegionValue(&Fiber{props: map[string]any{}}); parseGot != nil {
		parseT.Fatalf("reactiveRegionValue(missing render) = %#v, want nil", parseGot)
	}
	parseElement := CreateElement("section", nil, "portal")
	if parseGot := reactiveRegionValue(&Fiber{props: map[string]any{reactiveRegionRenderProp: func() *Element { return parseElement }}}); parseGot != parseElement {
		parseT.Fatalf("reactiveRegionValue(render) = %#v, want %#v", parseGot, parseElement)
	}

	if parseIDs := reactiveRegionSourceIDs(nil); parseIDs != nil {
		parseT.Fatalf("reactiveRegionSourceIDs(nil) = %#v, want nil", parseIDs)
	}
	if parseIDs := reactiveRegionSourceIDs(&Fiber{props: map[string]any{reactiveRegionSourceIDsProp: []string{""}}}); parseIDs != nil {
		parseT.Fatalf("reactiveRegionSourceIDs(blank single) = %#v, want nil", parseIDs)
	}
	parseSingleIDs := reactiveRegionSourceIDs(&Fiber{props: map[string]any{reactiveRegionSourceIDsProp: []string{"count"}}})
	if len(parseSingleIDs) != 1 || parseSingleIDs[0] != "count" {
		parseT.Fatalf("reactiveRegionSourceIDs(single) = %#v, want [count]", parseSingleIDs)
	}
	parseMultiIDs := reactiveRegionSourceIDs(&Fiber{props: map[string]any{reactiveRegionSourceIDsProp: []string{"count", "", "count", "theme"}}})
	if len(parseMultiIDs) != 2 || parseMultiIDs[0] != "count" || parseMultiIDs[1] != "theme" {
		parseT.Fatalf("reactiveRegionSourceIDs(multi) = %#v, want [count theme]", parseMultiIDs)
	}
}

// TestResolvePortalParentCoversDirectResolvedAndSelectorTargets verifies direct DOM node, resolver, selector, and nil branches.
func TestResolvePortalParentCoversDirectResolvedAndSelectorTargets(parseT *testing.T) {
	parseSelectorNode := &testDOMNode{nodeType: "element", tag: "section", attributes: map[string]string{}, properties: map[string]any{}, styles: map[string]string{}}
	parseResolvedNode := &testDOMNode{nodeType: "element", tag: "aside", attributes: map[string]string{}, properties: map[string]any{}, styles: map[string]string{}}
	parseDirectNode := &testDOMNode{nodeType: "element", tag: "div", attributes: map[string]string{}, properties: map[string]any{}, styles: map[string]string{}}
	parseRt := &Runtime{
		domAdapter: &portalTestDOMAdapter{
			testDOMAdapter: newTestDOMAdapter(),
			parseSelectors: map[string]DOMNode{"#portal": parseSelectorNode},
			parseResolved:  map[string]DOMNode{"raw-portal": parseResolvedNode},
		},
	}

	if parseGot := parseRt.resolvePortalParent(&Fiber{props: map[string]any{"portalTargetNode": parseDirectNode}}); parseGot != parseDirectNode {
		parseT.Fatalf("resolvePortalParent(direct) = %#v, want %#v", parseGot, parseDirectNode)
	}
	if parseGot := parseRt.resolvePortalParent(&Fiber{props: map[string]any{"portalTargetNode": "raw-portal"}}); parseGot != parseResolvedNode {
		parseT.Fatalf("resolvePortalParent(resolved) = %#v, want %#v", parseGot, parseResolvedNode)
	}
	if parseGot := parseRt.resolvePortalParent(&Fiber{props: map[string]any{"portalTargetSelector": "#portal"}}); parseGot != parseSelectorNode {
		parseT.Fatalf("resolvePortalParent(selector) = %#v, want %#v", parseGot, parseSelectorNode)
	}
	if parseGot := parseRt.resolvePortalParent(&Fiber{props: map[string]any{}}); parseGot != nil {
		parseT.Fatalf("resolvePortalParent(missing target) = %#v, want nil", parseGot)
	}
}

// TestRefreshEffectsForFiberCoversCleanupAndEpochSubtree verifies cleanup recursion and effect-epoch bumps across a subtree.
func TestRefreshEffectsForFiberCoversCleanupAndEpochSubtree(parseT *testing.T) {
	parseCleanupCount := 0
	parseParent := &Fiber{
		hooks: &Hooks{
			effectEpoch: 1,
			cleanups: []func(){
				func() { parseCleanupCount++ },
			},
		},
	}
	parseChild := &Fiber{
		hooks: &Hooks{
			effectEpoch: 3,
			cleanups: []func(){
				func() { parseCleanupCount++ },
			},
		},
		parent: parseParent,
	}
	parseSibling := &Fiber{
		hooks: &Hooks{
			effectEpoch: 5,
			cleanups: []func(){
				func() { parseCleanupCount++ },
			},
		},
		parent: parseParent,
	}
	parseParent.child = parseChild
	parseChild.sibling = parseSibling

	parseRt := &Runtime{}
	parseRt.RefreshEffectsForFiber(parseParent)

	if parseCleanupCount != 3 {
		parseT.Fatalf("expected three cleanup calls, got %d", parseCleanupCount)
	}
	if parseParent.hooks.cleanups[0] != nil || parseChild.hooks.cleanups[0] != nil || parseSibling.hooks.cleanups[0] != nil {
		parseT.Fatal("expected subtree cleanups to be cleared after refresh")
	}
	if parseParent.hooks.effectEpoch != 2 || parseChild.hooks.effectEpoch != 4 || parseSibling.hooks.effectEpoch != 6 {
		parseT.Fatalf("expected effect epochs to bump across subtree, got parent=%d child=%d sibling=%d", parseParent.hooks.effectEpoch, parseChild.hooks.effectEpoch, parseSibling.hooks.effectEpoch)
	}
	if parseRt.profiling.cleanupExecutions != 3 {
		parseT.Fatalf("expected cleanup execution profiling to increment, got %d", parseRt.profiling.cleanupExecutions)
	}
}
