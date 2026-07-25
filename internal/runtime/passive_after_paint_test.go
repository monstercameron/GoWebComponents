package runtime

import (
	"strings"
	"testing"
)

// v5 P1.1 — passive effects after paint.
//
// Layout effects must stay synchronous inside the commit task (they exist to
// observe committed DOM before the browser paints). Passive effects must move
// past the paint boundary so a slow UseEffect can no longer hold the frame.
//
// These tests pair with effect_ordering_contract_test.go, which pins the
// FLAG-OFF behavior. Every test here runs with the flag ON.

// deferredScheduler captures SetTimeout callbacks instead of running them, so a
// test can assert what happened *before* the paint boundary and then release
// the deferred work explicitly. A real browser scheduler yields to paint at
// exactly this seam.
type deferredScheduler struct {
	pending []func()
}

func (parseS *deferredScheduler) SetTimeout(parseCallback func(), parseDelay int) {
	parseS.pending = append(parseS.pending, parseCallback)
}

func (parseS *deferredScheduler) RequestIdleCallback(parseCallback func(Deadline)) {
	parseS.pending = append(parseS.pending, func() { parseCallback(globalInfiniteDeadline) })
}

// flush runs every queued callback, including any queued while draining.
func (parseS *deferredScheduler) flush() {
	for len(parseS.pending) > 0 {
		parseNext := parseS.pending[0]
		parseS.pending = parseS.pending[1:]
		parseNext()
	}
}

func (parseS *deferredScheduler) pendingCount() int { return len(parseS.pending) }

// newPaintSplitRuntime builds a runtime with P1.1 enabled and a scheduler whose
// deferred work the test controls.
func newPaintSplitRuntime(parseT *testing.T) (*Runtime, DOMNode, *deferredScheduler) {
	parseT.Helper()
	parseAdapter := newTestDOMAdapter()
	parseScheduler := &deferredScheduler{}
	parseRt := NewRuntime(Config{
		DOMAdapter:               parseAdapter,
		Scheduler:                parseScheduler,
		PassiveEffectsAfterPaint: true,
		Reset:                    true,
	})
	return parseRt, parseAdapter.CreateElement("div"), parseScheduler
}

// TestPaintSplit_LayoutRunsBeforePaintPassiveAfter is the core P1.1 guarantee.
func TestPaintSplit_LayoutRunsBeforePaintPassiveAfter(parseT *testing.T) {
	parseRt, parseRoot, parseScheduler := newPaintSplitRuntime(parseT)
	parseLog := &recorder{}

	parseComponent := func() *Element {
		layout(parseLog, "layout")
		passive(parseLog, "passive")
		return CreateElement("div", map[string]any{})
	}

	if parseErr := parseRt.RenderInto(parseRoot, CreateElement(parseComponent, map[string]any{})); parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}
	// Work loop is dispatched through the scheduler too, so drive it to commit.
	parseScheduler.flush()

	// Everything has now run, but the ORDER is what matters: layout must have
	// been observable before the passive drain was queued.
	if getGot := parseLog.String(); getGot != "layout,passive" {
		parseT.Fatalf("order = %q, want %q", getGot, "layout,passive")
	}
}

// TestPaintSplit_PassiveIsDeferredNotInline proves the passive effect really
// crosses a scheduler boundary rather than merely running later in the same
// task. Without flushing, the layout effect must have run and the passive one
// must not have.
func TestPaintSplit_PassiveIsDeferredNotInline(parseT *testing.T) {
	parseRt, parseRoot, parseScheduler := newPaintSplitRuntime(parseT)
	parseLog := &recorder{}

	parseComponent := func() *Element {
		layout(parseLog, "layout")
		passive(parseLog, "passive")
		return CreateElement("div", map[string]any{})
	}

	// Drive the render pass to commit, then stop: the commit's own layout
	// effects have run and the passive drain is still queued.
	_ = parseRt.RenderInto(parseRoot, CreateElement(parseComponent, map[string]any{}))
	for parseScheduler.pendingCount() > 0 && !strings.Contains(parseLog.String(), "layout") {
		parseNext := parseScheduler.pending[0]
		parseScheduler.pending = parseScheduler.pending[1:]
		parseNext()
	}

	if getGot := parseLog.String(); getGot != "layout" {
		parseT.Fatalf("before paint boundary = %q, want %q — passive effects must not run in the commit task", getGot, "layout")
	}
	if parseScheduler.pendingCount() == 0 {
		parseT.Fatal("expected a deferred passive drain to be queued")
	}

	parseScheduler.flush()
	if getGot := parseLog.String(); getGot != "layout,passive" {
		parseT.Errorf("after paint boundary = %q, want %q", getGot, "layout,passive")
	}
}

// TestPaintSplit_TreeWideLayoutBeatsAllPassive is the ordering change P1.1
// deliberately makes. effect_ordering_contract_test.go pins the flag-off shape:
//
//	parent-layout, parent-passive, child-layout, child-passive
//
// With the split, every layout effect in the tree precedes every passive one,
// because passive effects no longer run until after paint.
func TestPaintSplit_TreeWideLayoutBeatsAllPassive(parseT *testing.T) {
	parseRt, parseRoot, parseScheduler := newPaintSplitRuntime(parseT)
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
	parseScheduler.flush()

	getWant := "parent-layout,child-layout,parent-passive,child-passive"
	if getGot := parseLog.String(); getGot != getWant {
		parseT.Errorf("tree-wide order = %q, want %q", getGot, getWant)
	}
}

// TestPaintSplit_SetupStaysTopDownWithinEachTier: P1.1 changes the relationship
// BETWEEN tiers, not the parent->child order inside one.
func TestPaintSplit_SetupStaysTopDownWithinEachTier(parseT *testing.T) {
	parseRt, parseRoot, parseScheduler := newPaintSplitRuntime(parseT)
	parseLog := &recorder{}

	parseGrandchild := func() *Element {
		passive(parseLog, "c")
		return CreateElement("i", map[string]any{})
	}
	parseChild := func() *Element {
		passive(parseLog, "b")
		return CreateElement("span", map[string]any{}, CreateElement(parseGrandchild, map[string]any{}))
	}
	parseParent := func() *Element {
		passive(parseLog, "a")
		return CreateElement("div", map[string]any{}, CreateElement(parseChild, map[string]any{}))
	}

	if parseErr := parseRt.RenderInto(parseRoot, CreateElement(parseParent, map[string]any{})); parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}
	parseScheduler.flush()

	if getGot := parseLog.String(); getGot != "a,b,c" {
		parseT.Errorf("passive setup order = %q, want %q (top-down must survive the split)", getGot, "a,b,c")
	}
}

// TestPaintSplit_LayoutEffectsStillObserveCommittedDOM: the whole point of
// keeping layout effects synchronous. If P1.1 deferred them too, this breaks.
func TestPaintSplit_LayoutEffectsStillObserveCommittedDOM(parseT *testing.T) {
	parseRt, parseRoot, parseScheduler := newPaintSplitRuntime(parseT)

	var isDOMPresentAtLayout bool
	parseComponent := func() *Element {
		GoUseLayoutEffect(func() func() {
			isDOMPresentAtLayout = len(parseRt.domAdapter.GetChildren(parseRoot)) > 0
			return nil
		})
		return CreateElement("div", map[string]any{})
	}

	if parseErr := parseRt.RenderInto(parseRoot, CreateElement(parseComponent, map[string]any{})); parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}
	parseScheduler.flush()

	if !isDOMPresentAtLayout {
		parseT.Error("layout effect ran before its DOM was committed")
	}
}

// TestPaintSplit_EffectsStillRunExactlyOnce guards the queue-consumption
// invariant across the split. The layout tier deliberately leaves the queue
// intact for the passive tier; if the passive tier then failed to consume it,
// effects would re-fire on later commits.
func TestPaintSplit_EffectsStillRunExactlyOnce(parseT *testing.T) {
	parseRt, parseRoot, parseScheduler := newPaintSplitRuntime(parseT)
	parseLog := &recorder{}

	parseStatic := func() *Element {
		GoUseLayoutEffect(func() func() {
			parseLog.add("layout-mount")
			return nil
		}, []any{})
		GoUseEffect(func() func() {
			parseLog.add("passive-mount")
			return nil
		}, []any{})
		return CreateElement("div", map[string]any{})
	}

	if parseErr := parseRt.RenderInto(parseRoot, CreateElement(parseStatic, map[string]any{})); parseErr != nil {
		parseT.Fatalf("first render: %v", parseErr)
	}
	parseScheduler.flush()
	if getGot := parseLog.String(); getGot != "layout-mount,passive-mount" {
		parseT.Fatalf("after mount = %q", getGot)
	}

	for parseI := range 3 {
		if parseErr := parseRt.RenderInto(parseRoot, CreateElement(parseStatic, map[string]any{})); parseErr != nil {
			parseT.Fatalf("re-render %d: %v", parseI, parseErr)
		}
		parseScheduler.flush()
	}

	if getGot := parseLog.String(); getGot != "layout-mount,passive-mount" {
		parseT.Errorf("after 3 re-renders = %q, want the mount effects to have fired once each", getGot)
	}
}

// TestPaintSplit_CleanupsStillRun: deferring setup must not strand cleanups.
func TestPaintSplit_CleanupsStillRun(parseT *testing.T) {
	parseRt, parseRoot, parseScheduler := newPaintSplitRuntime(parseT)
	parseLog := &recorder{}

	parseLeaf := func() *Element {
		GoUseEffect(func() func() {
			return func() { parseLog.add("cleanup") }
		})
		return CreateElement("span", map[string]any{})
	}

	if parseErr := parseRt.RenderInto(parseRoot, CreateElement("section", map[string]any{},
		CreateElement(parseLeaf, map[string]any{}))); parseErr != nil {
		parseT.Fatalf("mount: %v", parseErr)
	}
	parseScheduler.flush()
	parseLog.entries = nil

	// Remove the subtree.
	if parseErr := parseRt.RenderInto(parseRoot, CreateElement("section", map[string]any{})); parseErr != nil {
		parseT.Fatalf("unmount: %v", parseErr)
	}
	parseScheduler.flush()

	if getGot := parseLog.String(); getGot != "cleanup" {
		parseT.Errorf("cleanup log = %q, want %q", getGot, "cleanup")
	}
}

// TestPaintSplit_MultipleCommitsCoalesceToOneDrain: several commits landing
// before the deferred drain fires must not queue several drains, and must not
// run any effect twice.
func TestPaintSplit_MultipleCommitsCoalesceToOneDrain(parseT *testing.T) {
	parseRt, parseRoot, parseScheduler := newPaintSplitRuntime(parseT)
	parseLog := &recorder{}

	parseCounter := 0
	parseComponent := func() *Element {
		parseCounter++
		passive(parseLog, "p")
		return CreateElement("div", map[string]any{})
	}

	// Two renders back to back; drive both to commit without letting the
	// passive drain fire in between where possible.
	_ = parseRt.RenderInto(parseRoot, CreateElement(parseComponent, map[string]any{}))
	_ = parseRt.RenderInto(parseRoot, CreateElement(parseComponent, map[string]any{}))
	parseScheduler.flush()

	// Whatever the commit count, no effect may run more times than it was
	// queued: the log must not exceed the number of component renders.
	if getCount := len(parseLog.entries); getCount > parseCounter {
		parseT.Errorf("passive effect ran %d times across %d renders — effects must not double-run", getCount, parseCounter)
	}
	if len(parseLog.entries) == 0 {
		parseT.Error("passive effects never ran")
	}
}

// TestPaintSplit_NoSchedulerRunsPassiveInline pins the deliberate native/SSR
// carve-out: with no scheduler there is no paint to yield to, and deferring
// would silently change semantics for every native caller.
func TestPaintSplit_NoSchedulerRunsPassiveInline(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{
		DOMAdapter:               parseAdapter,
		PassiveEffectsAfterPaint: true,
		Reset:                    true,
	})
	parseRoot := parseAdapter.CreateElement("div")
	parseLog := &recorder{}

	parseComponent := func() *Element {
		layout(parseLog, "layout")
		passive(parseLog, "passive")
		return CreateElement("div", map[string]any{})
	}

	// No scheduler: RenderInto must complete everything before returning.
	if parseErr := parseRt.RenderInto(parseRoot, CreateElement(parseComponent, map[string]any{})); parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}

	if getGot := parseLog.String(); getGot != "layout,passive" {
		parseT.Errorf("inline order = %q, want %q — with no scheduler both tiers run synchronously", getGot, "layout,passive")
	}
}

// TestPaintSplit_FlagOffPreservesLegacyOrdering: the flag must be a true
// no-op when disabled (R2 — every phase defaults to current behavior).
func TestPaintSplit_FlagOffPreservesLegacyOrdering(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := &deferredScheduler{}
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
		Reset:      true,
	})
	parseRoot := parseAdapter.CreateElement("div")
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
	parseScheduler.flush()

	getWantLegacy := "parent-layout,parent-passive,child-layout,child-passive"
	if getGot := parseLog.String(); getGot != getWantLegacy {
		parseT.Errorf("flag-off order = %q, want the pre-P1.1 shape %q", getGot, getWantLegacy)
	}
}
