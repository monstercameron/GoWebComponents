package runtime

import (
	"reflect"
	"testing"
)

type batchTestAdapter struct {
	*testDOMAdapter
	batches []map[string]string
}

func (a *batchTestAdapter) BatchSetAttributes(node DOMNode, attrs map[string]string) {
	copyAttrs := make(map[string]string, len(attrs))
	for k, v := range attrs {
		copyAttrs[k] = v
	}
	a.batches = append(a.batches, copyAttrs)
	for k, v := range attrs {
		a.SetAttribute(node, k, v)
	}
}

func expectPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}

func TestHooks_PanicCoverage(t *testing.T) {
	SetCurrentFiber(nil)
	resetGlobalRuntimeForTest()

	expectPanic(t, func() { _, _ = GoUseState(nil, 1) })
	expectPanic(t, func() { GoUseEffect(func() func() { return nil }) })
	expectPanic(t, func() { _ = GoUseMemo(func() interface{} { return 1 }) })
	expectPanic(t, func() { _ = GoUseCallback(func() {}, "x") })
	expectPanic(t, func() { _ = GoUseRef("x") })
}

func TestGoUseState_CoversSetterGrowthAndInvalidTypes(t *testing.T) {
	resetGlobalRuntimeForTest()
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	fiber := &Fiber{typeOf: "test", props: map[string]interface{}{}, hooks: &Hooks{}}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseState(rt, 1)
	if get() != 1 {
		t.Fatalf("expected initial state 1")
	}

	// Force the len-growth branch.
	fiber.hooks.states = fiber.hooks.states[:1]
	set(2)
	if get() != 2 {
		t.Fatalf("expected updated state after len growth")
	}

	// Force the cap-growth branch.
	fiber.hooks.states = fiber.hooks.states[:1:1]
	set(3)
	if get() != 3 {
		t.Fatalf("expected updated state after cap growth")
	}

	// Force the current-value type assertion miss and invalid setter no-op.
	fiber.hooks.states[0] = "wrong-type"
	set(struct{}{})
	if _, ok := fiber.hooks.states[0].(string); !ok {
		t.Fatalf("expected invalid setter input to leave state untouched")
	}

	// Cover functional updates with zero-value current after type mismatch.
	set(func(v int) int { return v + 5 })
	if get() != 5 {
		t.Fatalf("expected functional update from zero-value current, got %d", get())
	}
}

func TestHooks_ExistingCapacityBranches(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  map[string]interface{}{},
		hooks: &Hooks{
			deps:      make([][]interface{}, 0, 2),
			cleanups:  make([]func(), 0, 2),
			memos:     make([]memoizedValue, 0, 2),
			callbacks: make([]callbackValue, 0, 2),
			refs:      make([]*RefValue, 0, 2),
			ids:       make([]string, 0, 2),
			funcs:     make([]funcHandlerValue, 0, 2),
		},
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	resetGlobalRuntimeForTest()
	InitGlobalRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})

	GoUseEffect(func() func() { return nil }, "dep")
	_ = GoUseMemo(func() interface{} { return 1 }, "memo")
	testFn := func() {}
	if cb := GoUseCallback(testFn, "cb"); reflect.ValueOf(cb).Pointer() != reflect.ValueOf(testFn).Pointer() {
		t.Fatalf("expected callback passthrough on first call")
	}
	ref := GoUseRef("v")
	if ref.Current != "v" {
		t.Fatalf("expected ref current to match initial value")
	}
	if id := GoUseId(); id == "" {
		t.Fatal("expected non-empty id")
	}
	_ = GoUseFunc(func() {})
}

func TestGoUseFunc_PanicsWithoutDOMAdapterAndCoversCapacityReuse(t *testing.T) {
	resetGlobalRuntimeForTest()
	globalRuntime = &Runtime{}
	fiber := &Fiber{
		typeOf: "test",
		props:  map[string]interface{}{},
		hooks:  &Hooks{funcs: make([]funcHandlerValue, 0, 2)},
	}
	SetCurrentFiber(fiber)
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	expectPanic(t, func() { _ = GoUseFunc(func() {}) })

	InitGlobalRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	fiber.hooks.funcIndex = 0
	fiber.hooks.index = 0
	_ = GoUseFunc(func() {})
	if len(fiber.hooks.funcs) != 1 {
		t.Fatalf("expected funcs slice to grow within capacity")
	}
}

func TestFastEqual_AdditionalCoverage(t *testing.T) {
	fn := func() {}
	if !fastEqual(float64(1.5), float64(1.5)) || fastEqual(float64(1.5), float64(2.5)) {
		t.Fatal("float64 branch failed")
	}
	if !fastEqual(int64(1), int64(1)) || fastEqual(int64(1), int64(2)) {
		t.Fatal("int64 branch failed")
	}
	if !fastEqual(float32(1.5), float32(1.5)) || fastEqual(float32(1.5), float32(2.5)) {
		t.Fatal("float32 branch failed")
	}
	if !fastEqual(int32(7), int32(7)) || fastEqual(int32(7), int32(8)) {
		t.Fatal("int32 branch failed")
	}
	if !fastEqual(uint(2), uint(2)) || fastEqual(uint(2), uint(3)) {
		t.Fatal("uint branch failed")
	}
	if !fastEqual(uint64(3), uint64(3)) || fastEqual(uint64(3), uint64(4)) {
		t.Fatal("uint64 branch failed")
	}
	if !fastEqual(uint32(4), uint32(4)) || fastEqual(uint32(4), uint32(5)) {
		t.Fatal("uint32 branch failed")
	}
	if !fastEqual(uint8(5), uint8(5)) || fastEqual(uint8(5), uint8(6)) {
		t.Fatal("uint8 branch failed")
	}
	if !fastEqual(fn, fn) {
		t.Fatal("function pointer equality failed")
	}
	slice := []int{1, 2}
	if !fastEqual(slice, slice) || fastEqual(slice, []int{1, 2}) {
		t.Fatal("slice pointer branch failed")
	}
	m := map[string]int{"x": 1}
	if !fastEqual(m, m) || fastEqual(m, map[string]int{"x": 1}) {
		t.Fatal("map pointer branch failed")
	}
	type deep struct{ Items []int }
	if !fastEqual(deep{Items: []int{1}}, deep{Items: []int{1}}) {
		t.Fatal("deep equal fallback failed")
	}
	if fastEqual(1, "1") {
		t.Fatal("mismatched type equality should be false")
	}
}

func TestAreDepsEqual_FourAndLonger(t *testing.T) {
	if !areDepsEqual([]interface{}{1, 2, 3, 4}, []interface{}{1, 2, 3, 4}) {
		t.Fatal("expected four-dep equality")
	}
	if areDepsEqual([]interface{}{1, 2, 3, 4}, []interface{}{1, 2, 3, 5}) {
		t.Fatal("expected four-dep inequality")
	}
	if !areDepsEqual([]interface{}{1, 2, 3, 4, 5}, []interface{}{1, 2, 3, 4, 5}) {
		t.Fatal("expected longer dependency equality")
	}
	if areDepsEqual([]interface{}{1, 2, 3, 4, 5}, []interface{}{1, 2, 3, 4, 6}) {
		t.Fatal("expected longer dependency inequality")
	}
}

func TestReconciler_InternalCoverage(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})

	// cloneChildFibers nil fast path
	rt.cloneChildFibers(&Fiber{})

	parent := &Fiber{
		alternate: &Fiber{
			child: &Fiber{
				typeOf:         "div",
				props:          map[string]interface{}{"id": "a"},
				textContent:    "x",
				dom:            &testDOMNode{tag: "div"},
				hooks:          &Hooks{},
				eventCallbacks: []EventHandler{nil},
				sibling: &Fiber{
					typeOf: "span",
					props:  map[string]interface{}{"id": "b"},
				},
			},
		},
	}
	rt.cloneChildFibers(parent)
	if parent.child == nil || parent.child.sibling == nil {
		t.Fatalf("expected sibling chain to be cloned")
	}

	children := []interface{}{"a", "b"}
	if !propsEqual(map[string]interface{}{}, map[string]interface{}{}) {
		t.Fatal("expected empty props to be equal")
	}
	if propsEqual(map[string]interface{}{"a": 1}, map[string]interface{}{}) {
		t.Fatal("expected len mismatch to be unequal")
	}
	if propsEqual(map[string]interface{}{"a": 1}, map[string]interface{}{"b": 1}) {
		t.Fatal("expected missing key to be unequal")
	}
	if !propsEqual(map[string]interface{}{"children": nil}, map[string]interface{}{"children": nil}) {
		t.Fatal("expected nil children to be equal")
	}
	if propsEqual(map[string]interface{}{"children": nil}, map[string]interface{}{"children": children}) {
		t.Fatal("expected nil vs non-nil children to differ")
	}
	if !propsEqual(map[string]interface{}{"children": children}, map[string]interface{}{"children": children}) {
		t.Fatal("expected identical children slice to be equal")
	}
	nonstr1 := []string{"a"}
	nonstr2 := nonstr1
	if !propsEqual(map[string]interface{}{"children": nonstr1}, map[string]interface{}{"children": nonstr2}) {
		t.Fatal("expected non-standard slice with same pointer to be equal")
	}
	if propsEqual(map[string]interface{}{"children": []interface{}{"a"}}, map[string]interface{}{"children": []interface{}{"a"}}) {
		t.Fatal("expected distinct children slices to be treated as different")
	}
	if !propsEqual(map[string]interface{}{"children": []interface{}{}}, map[string]interface{}{"children": []interface{}{}}) {
		t.Fatal("expected empty children slices to be equal")
	}
	if propsEqual(map[string]interface{}{"children": 1}, map[string]interface{}{"children": 1}) {
		t.Fatal("expected non-slice children fallback to be false")
	}

	fragment := &Element{Type: "FRAGMENT", Props: map[string]interface{}{"children": "bad"}}
	var typedNil *Element
	out, allocated := flattenFragments([]interface{}{nil, typedNil, fragment, "tail"})
	if !allocated || len(out) != 1 || out[0] != "tail" {
		t.Fatalf("expected flattenFragments to skip invalid fragment children, got %#v", out)
	}
	for i := range out {
		out[i] = nil
	}
	slicePool.Put(out)
}

func TestReconciler_BatchedDomAndEffectCoverage(t *testing.T) {
	adapter := &batchTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: newTestScheduler()})
	dom := adapter.CreateElement("div")

	for _, key := range []string{"id", "class", "data-a", "title", "name", "role", "aria-label"} {
		rt.updateDomProperties(dom, nil, map[string]interface{}{
			key:        "root",
			"count":    3,
			"children": []interface{}{},
		})
	}
	for _, key := range []string{"id", "class", "data-a", "title", "name", "role", "aria-label"} {
		rt.updateDomProperties(dom, nil, map[string]interface{}{
			key:        "root",
			"value":    "x",
			"children": []interface{}{},
		})
	}
	rt.updateDomProperties(dom, nil, map[string]interface{}{
		"id":       "root",
		"class":    "panel",
		"style":    "display:block",
		"children": []interface{}{},
	})
	if len(adapter.batches) == 0 {
		t.Fatal("expected batched attribute adapter path to run")
	}

	component := func(props map[string]interface{}) *Element { return Div(nil, "x") }
	fiber := &Fiber{
		typeOf:  component,
		props:   map[string]interface{}{},
		hooks:   &Hooks{},
		dirty:   true,
		effects: []Effect{{Fn: func() func() { return nil }, CleanupIndex: 0}},
	}
	rt.performUnitOfWork(fiber)
	if fiber.effects == nil {
		t.Fatal("expected effects slice to be retained")
	}

	single := &Fiber{
		hooks:   &Hooks{cleanups: make([]func(), 1)},
		effects: []Effect{{Fn: func() func() { return func() {} }, CleanupIndex: 0}},
	}
	rt.runEffects(single)
	if single.hooks.cleanups[0] == nil {
		t.Fatal("expected single effect cleanup to be stored")
	}

	multi := &Fiber{
		hooks: &Hooks{cleanups: make([]func(), 2)},
		effects: []Effect{
			{Fn: func() func() { return func() {} }, CleanupIndex: 0},
			{Fn: func() func() { return func() {} }, CleanupIndex: 1},
		},
	}
	rt.runEffects(multi)
	if multi.hooks.cleanups[0] == nil || multi.hooks.cleanups[1] == nil {
		t.Fatal("expected multi-effect cleanups to be stored")
	}
}

func TestRuntimeSchedulerState_ExtraCoverage(t *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	// InitGlobalRuntime branch that repairs nil internals on an existing global runtime.
	globalRuntime = &Runtime{}
	InitGlobalRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	if globalRuntime.atomRegistry == nil || globalRuntime.deletions == nil || globalRuntime.uiQueue == nil {
		t.Fatal("expected InitGlobalRuntime to repair runtime internals")
	}

	// continueWorkLoop direct coverage.
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	rt.continueWorkLoop()

	// Render branch with nil deletions slice.
	rt.deletions = nil
	rt.Render(Div(nil, "x"), newTestDOMAdapter().CreateElement("div"))
	if rt.deletions == nil {
		t.Fatal("expected Render to initialize deletions slice")
	}

	// ScheduleUpdateForFiber nil fast path and already-dirty parent break.
	rt.ScheduleUpdateForFiber(nil)
	root := &Fiber{dirty: true, needsUpdate: true}
	child := &Fiber{parent: root}
	rt.currentRoot = &Fiber{typeOf: "ROOT", props: map[string]interface{}{}, dom: newTestDOMAdapter().CreateElement("div")}
	rt.updateScheduled = false
	rt.ScheduleUpdateForFiber(child)
	if !child.dirty || !child.needsUpdate {
		t.Fatal("expected child to be marked dirty")
	}

	// EnqueueUI fallback path when queue is full.
	oldQueue := uiQueue
	defer func() { uiQueue = oldQueue }()
	uiQueue = make(chan uiQueueItem, 1)
	uiQueue <- uiQueueItem{fn: func() {}}
	fallbackCalled := false
	EnqueueUI(func() { fallbackCalled = true })
	if !fallbackCalled {
		t.Fatal("expected EnqueueUI fallback to execute synchronously when queue is full")
	}

	// State helper and fallback paths.
	rt.atomRegistry = nil
	if _, ok := rt.GetAtomValue("missing"); ok {
		t.Fatal("expected GetAtomValue to fail without registry")
	}
	if err := rt.SetAtomValue("missing", 1); err == nil {
		t.Fatal("expected SetAtomValue to error without registry")
	}
	rt.atomRegistry = NewAtomRegistry()
	if _, ok := rt.atomRegistry.GetAtom("missing"); ok {
		t.Fatal("expected GetAtom miss")
	}

	fiber := &Fiber{hooks: &Hooks{}}
	rt.CleanupAtomSubscriptions(fiber)

	SetCurrentFiber(nil)
	expectPanic(t, func() { _, _ = GoUseAtom(rt, "id", 1) })
	expectPanic(t, func() { _, _ = GoUseAtom(&Runtime{}, "id", 1) })

	SetCurrentFiber(&Fiber{typeOf: "test", props: map[string]interface{}{}})
	defer SetCurrentFiber(nil)
	get, set := GoUseAtom(rt, "value", 3)
	rt.atomRegistry.SetAtom("value", "wrong")
	if get() != 3 {
		t.Fatal("expected type-mismatch fallback to initial value")
	}
	delete(rt.atomRegistry.atoms, "value")
	if get() != 3 {
		t.Fatal("expected missing atom to fall back to initial value")
	}
	set(struct{}{})
	if _, ok := rt.atomRegistry.GetAtom("value"); ok {
		t.Fatal("expected invalid setter input to leave missing atom untouched")
	}

	// Cover atom hook bookkeeping branches.
	atomFiber := &Fiber{
		typeOf: "test",
		props:  map[string]interface{}{},
		hooks:  &Hooks{atoms: make([]string, 0, 2)},
	}
	SetCurrentFiber(atomFiber)
	_, _ = GoUseAtom(rt, "first", 1)
	if len(atomFiber.hooks.atoms) != 1 {
		t.Fatal("expected atom slot to grow within capacity")
	}
	atomFiber.hooks.atomIndex = 0
	_, _ = GoUseAtom(rt, "first", 1)
	if atomFiber.hooks.atoms[0] != "first" {
		t.Fatal("expected existing atom slot to be updated")
	}

	rt.atomRegistry = nil
	rt.CleanupAtomSubscriptions(&Fiber{})
}
