package runtime

import "testing"

type releasableWrapper struct {
	released *int
}

func (w *releasableWrapper) Release() {
	*w.released++
}

type funcWrapTestAdapter struct {
	*testDOMAdapter
	releasedCount *int
}

func (a *funcWrapTestAdapter) WrapFunction(fn interface{}) interface{} {
	return &releasableWrapper{released: a.releasedCount}
}

func TestIsNilableType(t *testing.T) {
	if isNilableType[int]() {
		t.Fatal("expected int to be non-nilable")
	}
	if !isNilableType[*int]() {
		t.Fatal("expected pointer type to be nilable")
	}
	if !isNilableType[[]string]() {
		t.Fatal("expected slice type to be nilable")
	}
}

func TestGoUseFunc_ReleasesOldWrapperOnRerender(t *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	released := 0
	adapter := &funcWrapTestAdapter{
		testDOMAdapter: newTestDOMAdapter(),
		releasedCount:  &released,
	}
	InitGlobalRuntime(Config{DOMAdapter: adapter, Scheduler: newTestScheduler()})

	fiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	first := GoUseFunc(func() {})
	if first == nil {
		t.Fatal("expected wrapped function on first render")
	}

	fiber.hooks.index = 0
	fiber.hooks.funcIndex = 0

	second := GoUseFunc(func() {})
	if second == nil {
		t.Fatal("expected wrapped function on second render")
	}
	if released != 1 {
		t.Fatalf("expected old wrapper to be released once, got %d", released)
	}
}

func TestGoUseFunc_PanicsWithoutComponentContext(t *testing.T) {
	SetCurrentFiber(nil)
	defer func() {
		if recover() == nil {
			t.Fatal("expected GoUseFunc to panic outside component context")
		}
	}()
	GoUseFunc(func() {})
}

func TestGoUseFunc_PanicsForNonFunction(t *testing.T) {
	fiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	defer func() {
		if recover() == nil {
			t.Fatal("expected GoUseFunc to panic for non-function input")
		}
	}()
	GoUseFunc(123)
}

func TestGoUseState_SchedulesUpdateForLatestHookOwner(t *testing.T) {
	scheduler := newTestScheduler()
	rt := &Runtime{
		scheduler: scheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  map[string]interface{}{},
		},
	}

	oldRoot := &Fiber{typeOf: "ROOT", props: map[string]interface{}{}}
	oldFiber := &Fiber{typeOf: "counter", parent: oldRoot, props: map[string]interface{}{}}
	SetCurrentFiber(oldFiber)
	_, setValue := GoUseState[int](rt, 0)
	SetCurrentFiber(nil)

	currentRoot := &Fiber{typeOf: "ROOT", props: map[string]interface{}{}}
	currentFiber := &Fiber{
		typeOf:    "counter",
		parent:    currentRoot,
		props:     map[string]interface{}{},
		hooks:     oldFiber.hooks,
		alternate: oldFiber,
	}
	currentFiber.hooks.owner = currentFiber
	rt.currentRoot = currentRoot

	setValue(1)

	if !currentFiber.dirty || !currentFiber.needsUpdate {
		t.Fatal("expected current hook owner to be marked dirty")
	}
	if !currentRoot.dirty || !currentRoot.needsUpdate {
		t.Fatal("expected current root path to be marked dirty")
	}
	if oldRoot.dirty || oldRoot.needsUpdate {
		t.Fatal("did not expect stale fiber ancestry to receive the update")
	}
	if len(scheduler.timeouts) != 1 {
		t.Fatalf("expected one scheduled timeout, got %d", len(scheduler.timeouts))
	}
}

func TestGoUseId_PanicsWithoutComponentContext(t *testing.T) {
	SetCurrentFiber(nil)
	defer func() {
		if recover() == nil {
			t.Fatal("expected GoUseId to panic outside component context")
		}
	}()
	GoUseId()
}

func TestAreDepsEqual_LongSlicesAndMismatch(t *testing.T) {
	prev := []interface{}{1, 2, 3, 4, 5}
	next := []interface{}{1, 2, 3, 4, 5}
	diff := []interface{}{1, 2, 3, 4, 6}

	if !areDepsEqual(prev, next) {
		t.Fatal("expected long equal deps to compare true")
	}
	if areDepsEqual(prev, diff) {
		t.Fatal("expected long mismatched deps to compare false")
	}
}

func TestFastEqual_FunctionsSlicesMapsAndStructs(t *testing.T) {
	fn := func() {}
	if !fastEqual(fn, fn) {
		t.Fatal("expected identical functions to compare equal")
	}
	if fastEqual(func() {}, func() {}) {
		t.Fatal("expected distinct functions to compare different")
	}

	slice := []int{1, 2, 3}
	if !fastEqual(slice, slice) {
		t.Fatal("expected identical slice reference to compare equal")
	}
	if fastEqual([]int{1, 2}, []int{1, 2}) {
		t.Fatal("expected different slice references to compare different")
	}

	m := map[string]int{"a": 1}
	if !fastEqual(m, m) {
		t.Fatal("expected identical map reference to compare equal")
	}

	type pair struct {
		A int
		B string
	}
	if !fastEqual(pair{A: 1, B: "x"}, pair{A: 1, B: "x"}) {
		t.Fatal("expected equal structs to compare equal")
	}
	if fastEqual(pair{A: 1, B: "x"}, pair{A: 2, B: "x"}) {
		t.Fatal("expected different structs to compare false")
	}
	if fastEqual(1, int64(1)) {
		t.Fatal("expected different numeric types to compare false")
	}
}
