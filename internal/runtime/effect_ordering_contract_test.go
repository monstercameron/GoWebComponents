package runtime

import (
	"strings"
	"testing"
)

// Effect ordering contract (v5 P1.4).
//
// runEffects carries a comment calling its ordering "DELIBERATE and
// load-bearing," but before this file only the *cleanup* order was pinned
// (ui/effect_ordering_native_test.go). Setup order — the thing that comment is
// actually about — had no test at all, which means v5's P1.1 (move passive
// effects after paint) could have silently changed it.
//
// These tests pin CURRENT behavior exactly. P1.1 is expected to change
// TestEffectOrder_TreeWideInterleaving deliberately; the others must survive
// unchanged. A diff to any of them is the signal that P1.1 did more than it
// meant to.
//
// Native-testable: with no Scheduler configured, dispatchRuntimeWork runs the
// work loop synchronously, so a RenderInto call completes its commit and
// effects before returning.

// newOrderingRuntime builds a synchronous runtime plus a container node.
func newOrderingRuntime(parseT *testing.T) (*Runtime, DOMNode) {
	parseT.Helper()
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Reset: true})
	return parseRt, parseAdapter.CreateElement("div")
}

// recorder collects effect firings in execution order.
type recorder struct {
	entries []string
}

func (parseR *recorder) add(parseLabel string) { parseR.entries = append(parseR.entries, parseLabel) }
func (parseR *recorder) String() string        { return strings.Join(parseR.entries, ",") }

// passive registers a passive effect that records label on setup.
func passive(parseLog *recorder, parseLabel string) {
	GoUseEffect(func() func() {
		parseLog.add(parseLabel)
		return nil
	})
}

// layout registers a layout effect that records label on setup.
func layout(parseLog *recorder, parseLabel string) {
	GoUseLayoutEffect(func() func() {
		parseLog.add(parseLabel)
		return nil
	})
}

// TestEffectOrder_SetupIsTopDownParentBeforeChild pins the invariant the
// runEffects comment describes and nothing previously tested: setup runs
// parent -> child. React runs setup child -> parent; GWC deliberately does not,
// and that divergence must not drift by accident.
func TestEffectOrder_SetupIsTopDownParentBeforeChild(parseT *testing.T) {
	parseRt, parseRoot := newOrderingRuntime(parseT)
	parseLog := &recorder{}

	parseChild := func() *Element {
		passive(parseLog, "child")
		return CreateElement("span", map[string]any{})
	}
	parseParent := func() *Element {
		passive(parseLog, "parent")
		return CreateElement("div", map[string]any{}, CreateElement(parseChild, map[string]any{}))
	}

	if parseErr := parseRt.RenderInto(parseRoot, CreateElement(parseParent, map[string]any{})); parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}

	if getGot := parseLog.String(); getGot != "parent,child" {
		parseT.Errorf("setup order = %q, want %q (top-down is deliberate — see runEffects)", getGot, "parent,child")
	}
}

// TestEffectOrder_SiblingsInDocumentOrder pins left-to-right sibling setup.
func TestEffectOrder_SiblingsInDocumentOrder(parseT *testing.T) {
	parseRt, parseRoot := newOrderingRuntime(parseT)
	parseLog := &recorder{}

	makeLeaf := func(parseName string) func() *Element {
		return func() *Element {
			passive(parseLog, parseName)
			return CreateElement("span", map[string]any{})
		}
	}

	if parseErr := parseRt.RenderInto(parseRoot, CreateElement("section", map[string]any{},
		CreateElement(makeLeaf("a"), map[string]any{}),
		CreateElement(makeLeaf("b"), map[string]any{}),
		CreateElement(makeLeaf("c"), map[string]any{}),
	)); parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}

	if getGot := parseLog.String(); getGot != "a,b,c" {
		parseT.Errorf("sibling setup order = %q, want %q", getGot, "a,b,c")
	}
}

// TestEffectOrder_LayoutBeforePassiveWithinFiber pins the guarantee
// GoUseLayoutEffect documents: within one component, layout effects run before
// passive effects regardless of registration order.
//
// P1.1 must preserve this. It is the half of the layout/passive split that does
// not depend on paint timing.
func TestEffectOrder_LayoutBeforePassiveWithinFiber(parseT *testing.T) {
	parseRt, parseRoot := newOrderingRuntime(parseT)
	parseLog := &recorder{}

	parseComponent := func() *Element {
		// Registered passive-first on purpose: ordering must come from the
		// tier, not from call order.
		passive(parseLog, "passive")
		layout(parseLog, "layout")
		return CreateElement("div", map[string]any{})
	}

	if parseErr := parseRt.RenderInto(parseRoot, CreateElement(parseComponent, map[string]any{})); parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}

	if getGot := parseLog.String(); getGot != "layout,passive" {
		parseT.Errorf("within-fiber order = %q, want %q (layout tier wins over registration order)", getGot, "layout,passive")
	}
}

// TestEffectOrder_TreeWideInterleaving pins the ordering P1.1 IS EXPECTED TO
// CHANGE, so the change is deliberate and reviewed rather than incidental.
//
//	current (pre-P1.1): parent-layout, parent-passive, child-layout, child-passive
//	after P1.1:         parent-layout, child-layout, <paint>, parent-passive, child-passive
//
// Today runEffects walks fiber by fiber, running both tiers per fiber, so a
// parent's PASSIVE effect runs before a child's LAYOUT effect. Once passive
// effects move after paint, every layout effect in the tree must precede every
// passive effect.
//
// When P1.1 lands, update the want string here and record the rationale — do
// not delete this test.
func TestEffectOrder_TreeWideInterleaving(parseT *testing.T) {
	parseRt, parseRoot := newOrderingRuntime(parseT)
	parseLog := &recorder{}

	parseChild := func() *Element {
		layout(parseLog, "child-layout")
		passive(parseLog, "child-passive")
		return CreateElement("span", map[string]any{})
	}
	parseParent := func() *Element {
		layout(parseLog, "parent-layout")
		passive(parseLog, "parent-passive")
		return CreateElement("div", map[string]any{}, CreateElement(parseChild, map[string]any{}))
	}

	if parseErr := parseRt.RenderInto(parseRoot, CreateElement(parseParent, map[string]any{})); parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}

	getWantPreP11 := "parent-layout,parent-passive,child-layout,child-passive"
	if getGot := parseLog.String(); getGot != getWantPreP11 {
		parseT.Errorf("tree-wide order = %q, want %q\n"+
			"If P1.1 (passive-after-paint) has landed, the expected value is\n"+
			"  parent-layout,child-layout,parent-passive,child-passive\n"+
			"— update this test WITH a rationale rather than deleting it.", getGot, getWantPreP11)
	}
}

// TestEffectOrder_EffectsRunExactlyOncePerCommit pins the queue-consumption
// invariant. runFiberEffects truncates the queue after running specifically so
// a bailout-reused fiber cannot re-fire stale mount effects on an unrelated
// later commit; P1.1 moves where that draining happens, so the invariant needs
// to be checked independently of when it runs.
func TestEffectOrder_EffectsRunExactlyOncePerCommit(parseT *testing.T) {
	parseRt, parseRoot := newOrderingRuntime(parseT)
	parseLog := &recorder{}

	parseStatic := func() *Element {
		// Empty deps: setup must fire on mount and never again.
		GoUseEffect(func() func() {
			parseLog.add("mount")
			return nil
		}, []any{})
		return CreateElement("div", map[string]any{})
	}

	if parseErr := parseRt.RenderInto(parseRoot, CreateElement(parseStatic, map[string]any{})); parseErr != nil {
		parseT.Fatalf("first render: %v", parseErr)
	}
	if getGot := parseLog.String(); getGot != "mount" {
		parseT.Fatalf("after mount = %q, want %q", getGot, "mount")
	}

	// Re-render the same tree several times. A mount effect with empty deps
	// must not fire again.
	for parseI := range 3 {
		if parseErr := parseRt.RenderInto(parseRoot, CreateElement(parseStatic, map[string]any{})); parseErr != nil {
			parseT.Fatalf("re-render %d: %v", parseI, parseErr)
		}
	}

	if getGot := parseLog.String(); getGot != "mount" {
		parseT.Errorf("after 3 re-renders = %q, want %q (empty-deps effect must fire once)", getGot, "mount")
	}
}

// TestEffectOrder_LayoutEffectsObserveCommittedDOM pins the reason layout
// effects exist: the DOM they read must already be committed when they run.
// P1.1 keeps layout effects synchronous precisely so this stays true, and a
// regression here would be silent in every test that only checks call order.
func TestEffectOrder_LayoutEffectsObserveCommittedDOM(parseT *testing.T) {
	parseRt, parseRoot := newOrderingRuntime(parseT)

	var isDOMPresentAtLayout bool
	parseComponent := func() *Element {
		GoUseLayoutEffect(func() func() {
			// The container must already hold the committed child by the time
			// a layout effect runs.
			isDOMPresentAtLayout = len(parseRt.domAdapter.GetChildren(parseRoot)) > 0
			return nil
		})
		return CreateElement("div", map[string]any{})
	}

	if parseErr := parseRt.RenderInto(parseRoot, CreateElement(parseComponent, map[string]any{})); parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}

	if !isDOMPresentAtLayout {
		parseT.Error("layout effect ran before its DOM was committed; layout effects must observe committed DOM")
	}
}
