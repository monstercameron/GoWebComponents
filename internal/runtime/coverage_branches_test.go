package runtime

import (
	"reflect"
	"testing"
)

type batchTestAdapter struct {
	*testDOMAdapter
	batches []map[string]string
}

func (parseA *batchTestAdapter) BatchSetAttributes(parseNode DOMNode, parseAttrs map[string]string) {
	parseCopyAttrs := make(map[string]string, len(parseAttrs))
	for parseK, parseV := range parseAttrs {
		parseCopyAttrs[parseK] = parseV
	}
	parseA.batches = append(parseA.batches, parseCopyAttrs)
	for parseK2, parseV2 := range parseAttrs {
		parseA.SetAttribute(parseNode, parseK2, parseV2)
	}
}

func expectPanic(parseT *testing.T, parseFn func()) {
	parseT.Helper()
	defer func() {
		if recover() == nil {
			parseT.Fatal("expected panic")
		}
	}()
	parseFn()
}

func TestHooks_PanicCoverage(parseT *testing.T) {
	SetCurrentFiber(nil)
	resetGlobalRuntimeForTest()

	expectPanic(parseT, func() { _, _ = GoUseState(nil, 1) })
	expectPanic(parseT, func() { GoUseEffect(func() func() { return nil }) })
	expectPanic(parseT, func() { _ = GoUseMemo(func() interface{} { return 1 }) })
	expectPanic(parseT, func() { _ = GoUseCallback(func() {}, "x") })
	expectPanic(parseT, func() { _ = GoUseRef("x") })
}

func TestGoUseState_CoversSetterGrowthAndInvalidTypes(parseT *testing.T) {
	resetGlobalRuntimeForTest()
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseFiber := &Fiber{typeOf: "test", props: map[string]interface{}{}, hooks: &Hooks{}}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseState(parseRt, 1)
	if get() != 1 {
		parseT.Fatalf("expected initial state 1")
	}

	// Force the len-growth branch.
	parseFiber.hooks.states = parseFiber.hooks.states[:1]
	set(2)
	if get() != 2 {
		parseT.Fatalf("expected updated state after len growth")
	}

	// Force the cap-growth branch.
	parseFiber.hooks.states = parseFiber.hooks.states[:1:1]
	set(3)
	if get() != 3 {
		parseT.Fatalf("expected updated state after cap growth")
	}

	// Force the current-value type assertion miss and invalid setter no-op.
	parseFiber.hooks.states[0] = "wrong-type"
	set(struct{}{})
	if _, parseOk := parseFiber.hooks.states[0].(string); !parseOk {
		parseT.Fatalf("expected invalid setter input to leave state untouched")
	}

	// Cover functional updates with zero-value current after type mismatch.
	set(func(parseV int) int { return parseV + 5 })
	if get() != 5 {
		parseT.Fatalf("expected functional update from zero-value current, got %d", get())
	}
}

func TestHooks_ExistingCapacityBranches(parseT *testing.T) {
	parseFiber := &Fiber{
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
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	resetGlobalRuntimeForTest()
	InitGlobalRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})

	GoUseEffect(func() func() { return nil }, "dep")
	_ = GoUseMemo(func() interface{} { return 1 }, "memo")
	parseTestFn := func() {}
	if parseCb := GoUseCallback(parseTestFn, "cb"); reflect.ValueOf(parseCb).Pointer() != reflect.ValueOf(parseTestFn).Pointer() {
		parseT.Fatalf("expected callback passthrough on first call")
	}
	parseRef := GoUseRef("v")
	if parseRef.Current != "v" {
		parseT.Fatalf("expected ref current to match initial value")
	}
	if parseId := GoUseId(); parseId == "" {
		parseT.Fatal("expected non-empty id")
	}
	_ = GoUseFunc(func() {})
}

func TestGoUseFunc_PanicsWithoutDOMAdapterAndCoversCapacityReuse(parseT *testing.T) {
	resetGlobalRuntimeForTest()
	globalRuntime = &Runtime{}
	parseFiber := &Fiber{
		typeOf: "test",
		props:  map[string]interface{}{},
		hooks:  &Hooks{funcs: make([]funcHandlerValue, 0, 2)},
	}
	SetCurrentFiber(parseFiber)
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	expectPanic(parseT, func() { _ = GoUseFunc(func() {}) })

	InitGlobalRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseFiber.hooks.funcIndex = 0
	parseFiber.hooks.index = 0
	_ = GoUseFunc(func() {})
	if len(parseFiber.hooks.funcs) != 1 {
		parseT.Fatalf("expected funcs slice to grow within capacity")
	}
}

func TestFastEqual_AdditionalCoverage(parseT *testing.T) {
	parseFn := func() {}
	if !fastEqual(float64(1.5), float64(1.5)) || fastEqual(float64(1.5), float64(2.5)) {
		parseT.Fatal("float64 branch failed")
	}
	if !fastEqual(int64(1), int64(1)) || fastEqual(int64(1), int64(2)) {
		parseT.Fatal("int64 branch failed")
	}
	if !fastEqual(float32(1.5), float32(1.5)) || fastEqual(float32(1.5), float32(2.5)) {
		parseT.Fatal("float32 branch failed")
	}
	if !fastEqual(int32(7), int32(7)) || fastEqual(int32(7), int32(8)) {
		parseT.Fatal("int32 branch failed")
	}
	if !fastEqual(uint(2), uint(2)) || fastEqual(uint(2), uint(3)) {
		parseT.Fatal("uint branch failed")
	}
	if !fastEqual(uint64(3), uint64(3)) || fastEqual(uint64(3), uint64(4)) {
		parseT.Fatal("uint64 branch failed")
	}
	if !fastEqual(uint32(4), uint32(4)) || fastEqual(uint32(4), uint32(5)) {
		parseT.Fatal("uint32 branch failed")
	}
	if !fastEqual(uint8(5), uint8(5)) || fastEqual(uint8(5), uint8(6)) {
		parseT.Fatal("uint8 branch failed")
	}
	if !fastEqual(parseFn, parseFn) {
		parseT.Fatal("function pointer equality failed")
	}
	parseSlice := []int{1, 2}
	if !fastEqual(parseSlice, parseSlice) || fastEqual(parseSlice, []int{1, 2}) {
		parseT.Fatal("slice pointer branch failed")
	}
	parseM := map[string]int{"x": 1}
	if !fastEqual(parseM, parseM) || fastEqual(parseM, map[string]int{"x": 1}) {
		parseT.Fatal("map pointer branch failed")
	}
	type deep struct{ Items []int }
	if !fastEqual(deep{Items: []int{1}}, deep{Items: []int{1}}) {
		parseT.Fatal("deep equal fallback failed")
	}
	if fastEqual(1, "1") {
		parseT.Fatal("mismatched type equality should be false")
	}
}

func TestAreDepsEqual_FourAndLonger(parseT *testing.T) {
	if !areDepsEqual([]interface{}{1, 2, 3, 4}, []interface{}{1, 2, 3, 4}) {
		parseT.Fatal("expected four-dep equality")
	}
	if areDepsEqual([]interface{}{1, 2, 3, 4}, []interface{}{1, 2, 3, 5}) {
		parseT.Fatal("expected four-dep inequality")
	}
	if !areDepsEqual([]interface{}{1, 2, 3, 4, 5}, []interface{}{1, 2, 3, 4, 5}) {
		parseT.Fatal("expected longer dependency equality")
	}
	if areDepsEqual([]interface{}{1, 2, 3, 4, 5}, []interface{}{1, 2, 3, 4, 6}) {
		parseT.Fatal("expected longer dependency inequality")
	}
}

func TestReconciler_InternalCoverage(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})

	// cloneChildFibers nil fast path
	parseRt.cloneChildFibers(&Fiber{})

	parseParent := &Fiber{
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
	parseRt.cloneChildFibers(parseParent)
	if parseParent.child == nil || parseParent.child.sibling == nil {
		parseT.Fatalf("expected sibling chain to be cloned")
	}

	parseChildren := []interface{}{"a", "b"}
	if !propsEqual(map[string]interface{}{}, map[string]interface{}{}) {
		parseT.Fatal("expected empty props to be equal")
	}
	if propsEqual(map[string]interface{}{"a": 1}, map[string]interface{}{}) {
		parseT.Fatal("expected len mismatch to be unequal")
	}
	if propsEqual(map[string]interface{}{"a": 1}, map[string]interface{}{"b": 1}) {
		parseT.Fatal("expected missing key to be unequal")
	}
	if !propsEqual(map[string]interface{}{"children": nil}, map[string]interface{}{"children": nil}) {
		parseT.Fatal("expected nil children to be equal")
	}
	if propsEqual(map[string]interface{}{"children": nil}, map[string]interface{}{"children": parseChildren}) {
		parseT.Fatal("expected nil vs non-nil children to differ")
	}
	if !propsEqual(map[string]interface{}{"children": parseChildren}, map[string]interface{}{"children": parseChildren}) {
		parseT.Fatal("expected identical children slice to be equal")
	}
	parseNonstr1 := []string{"a"}
	parseNonstr2 := parseNonstr1
	if !propsEqual(map[string]interface{}{"children": parseNonstr1}, map[string]interface{}{"children": parseNonstr2}) {
		parseT.Fatal("expected non-standard slice with same pointer to be equal")
	}
	if propsEqual(map[string]interface{}{"children": []interface{}{"a"}}, map[string]interface{}{"children": []interface{}{"a"}}) {
		parseT.Fatal("expected distinct children slices to be treated as different")
	}
	if !propsEqual(map[string]interface{}{"children": []interface{}{}}, map[string]interface{}{"children": []interface{}{}}) {
		parseT.Fatal("expected empty children slices to be equal")
	}
	if propsEqual(map[string]interface{}{"children": 1}, map[string]interface{}{"children": 1}) {
		parseT.Fatal("expected non-slice children fallback to be false")
	}

	parseFragment := &Element{Type: "FRAGMENT", Props: map[string]interface{}{"children": "bad"}}
	var parseTypedNil *Element
	parseOut, parseAllocated := flattenFragments([]interface{}{nil, parseTypedNil, parseFragment, "tail"})
	if !parseAllocated || len(parseOut) != 1 || parseOut[0] != "tail" {
		parseT.Fatalf("expected flattenFragments to skip invalid fragment children, got %#v", parseOut)
	}
	for parseI := range parseOut {
		parseOut[parseI] = nil
	}
	slicePool.clear(parseOut)
}

func TestReconciler_BatchedDomAndEffectCoverage(parseT *testing.T) {
	parseAdapter := &batchTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})
	parseDom := parseAdapter.CreateElement("div")

	for _, parseKey := range []string{"id", "class", "data-a", "title", "name", "role", "aria-label"} {
		parseRt.updateDomProperties(parseDom, nil, map[string]interface{}{
			parseKey:   "root",
			"count":    3,
			"children": []interface{}{},
		})
	}
	for _, parseKey2 := range []string{"id", "class", "data-a", "title", "name", "role", "aria-label"} {
		parseRt.updateDomProperties(parseDom, nil, map[string]interface{}{
			parseKey2:  "root",
			"value":    "x",
			"children": []interface{}{},
		})
	}
	parseRt.applyInitialDomProps(parseDom, map[string]interface{}{
		"id":       "root",
		"class":    "panel",
		"style":    "display:block",
		"children": []interface{}{},
	}, true, parseAdapter, false)
	if len(parseAdapter.batches) == 0 {
		parseT.Fatal("expected batched attribute adapter path to run")
	}

	parseComponent := func(parseProps map[string]interface{}) *Element { return Div(nil, "x") }
	parseFiber := &Fiber{
		typeOf:  parseComponent,
		props:   map[string]interface{}{},
		hooks:   &Hooks{},
		dirty:   true,
		effects: []Effect{{Fn: func() func() { return nil }, CleanupIndex: 0}},
	}
	parseRt.performUnitOfWork(parseFiber)
	if parseFiber.effects == nil {
		parseT.Fatal("expected effects slice to be retained")
	}

	parseSingle := &Fiber{
		hooks:   &Hooks{cleanups: make([]func(), 1)},
		effects: []Effect{{Fn: func() func() { return func() {} }, CleanupIndex: 0}},
	}
	parseRt.runEffects(parseSingle)
	if parseSingle.hooks.cleanups[0] == nil {
		parseT.Fatal("expected single effect cleanup to be stored")
	}

	parseMulti := &Fiber{
		hooks: &Hooks{cleanups: make([]func(), 2)},
		effects: []Effect{
			{Fn: func() func() { return func() {} }, CleanupIndex: 0},
			{Fn: func() func() { return func() {} }, CleanupIndex: 1},
		},
	}
	parseRt.runEffects(parseMulti)
	if parseMulti.hooks.cleanups[0] == nil || parseMulti.hooks.cleanups[1] == nil {
		parseT.Fatal("expected multi-effect cleanups to be stored")
	}
}

func TestRuntimeSchedulerState_ExtraCoverage(parseT *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	// InitGlobalRuntime branch that repairs nil internals on an existing global runtime.
	globalRuntime = &Runtime{}
	InitGlobalRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	if globalRuntime.atomRegistry == nil || globalRuntime.deletions == nil || globalRuntime.uiQueue == nil {
		parseT.Fatal("expected InitGlobalRuntime to repair runtime internals")
	}

	// continueWorkLoop direct coverage.
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseRt.continueWorkLoop()

	// Render branch with nil deletions slice.
	parseRt.deletions = nil
	parseRt.Render(Div(nil, "x"), newTestDOMAdapter().CreateElement("div"))
	if parseRt.deletions == nil {
		parseT.Fatal("expected Render to initialize deletions slice")
	}

	// ScheduleUpdateForFiber nil fast path and already-dirty parent break.
	parseRt.ScheduleUpdateForFiber(nil)
	parseRoot := &Fiber{dirty: true, needsUpdate: true}
	parseChild := &Fiber{parent: parseRoot}
	parseRt.currentRoot = &Fiber{typeOf: "ROOT", props: map[string]interface{}{}, dom: newTestDOMAdapter().CreateElement("div")}
	parseRt.updateScheduled = false
	parseRt.ScheduleUpdateForFiber(parseChild)
	if !parseChild.dirty || !parseChild.needsUpdate {
		parseT.Fatal("expected child to be marked dirty")
	}

	// EnqueueUI fallback path when queue is full.
	parseOldQueue := uiQueue
	defer func() { uiQueue = parseOldQueue }()
	uiQueue = make(chan uiQueueItem, 1)
	uiQueue <- uiQueueItem{fn: func() {}}
	isParseFallbackCalled := false
	EnqueueUI(func() { isParseFallbackCalled = true })
	if !isParseFallbackCalled {
		parseT.Fatal("expected EnqueueUI fallback to execute synchronously when queue is full")
	}

	// State helper and fallback paths.
	parseRt.atomRegistry = nil
	if _, parseOk := parseRt.GetAtomValue("missing"); parseOk {
		parseT.Fatal("expected GetAtomValue to fail without registry")
	}
	if parseErr := parseRt.SetAtomValue("missing", 1); parseErr == nil {
		parseT.Fatal("expected SetAtomValue to error without registry")
	}
	parseRt.atomRegistry = NewAtomRegistry()
	if _, parseOk2 := parseRt.atomRegistry.GetAtom("missing"); parseOk2 {
		parseT.Fatal("expected GetAtom miss")
	}

	parseFiber := &Fiber{hooks: &Hooks{}}
	parseRt.CleanupAtomSubscriptions(parseFiber)

	SetCurrentFiber(nil)
	expectPanic(parseT, func() { _, _ = GoUseAtom(parseRt, "id", 1) })
	expectPanic(parseT, func() { _, _ = GoUseAtom(&Runtime{}, "id", 1) })

	SetCurrentFiber(&Fiber{typeOf: "test", props: map[string]interface{}{}})
	defer SetCurrentFiber(nil)
	get, set := GoUseAtom(parseRt, "value", 3)
	parseRt.atomRegistry.SetAtom("value", "wrong")
	if get() != 3 {
		parseT.Fatal("expected type-mismatch fallback to initial value")
	}
	delete(parseRt.atomRegistry.atoms, "value")
	if get() != 3 {
		parseT.Fatal("expected missing atom to fall back to initial value")
	}
	set(struct{}{})
	if _, parseOk3 := parseRt.atomRegistry.GetAtom("value"); parseOk3 {
		parseT.Fatal("expected invalid setter input to leave missing atom untouched")
	}

	// Cover atom hook bookkeeping branches.
	parseAtomFiber := &Fiber{
		typeOf: "test",
		props:  map[string]interface{}{},
		hooks:  &Hooks{atoms: make([]string, 0, 2)},
	}
	SetCurrentFiber(parseAtomFiber)
	_, _ = GoUseAtom(parseRt, "first", 1)
	if len(parseAtomFiber.hooks.atoms) != 1 {
		parseT.Fatal("expected atom slot to grow within capacity")
	}
	parseAtomFiber.hooks.atomIndex = 0
	_, _ = GoUseAtom(parseRt, "first", 1)
	if parseAtomFiber.hooks.atoms[0] != "first" {
		parseT.Fatal("expected existing atom slot to be updated")
	}

	parseRt.atomRegistry = nil
	parseRt.CleanupAtomSubscriptions(&Fiber{})
}
