package runtime

import (
	"testing"
)

func TestGoUseState_InitialValue(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{scheduler: parseScheduler}

	// Create a fiber context
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	get, _ := GoUseState(parseRt, 42)

	if get() != 42 {
		parseT.Errorf("Expected initial value 42, got %d", get())
	}
}

func TestGoUseState_UpdateValue(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{scheduler: parseScheduler}

	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseState(parseRt, 0)

	set(10)

	if get() != 10 {
		parseT.Errorf("Expected updated value 10, got %d", get())
	}
}

func TestGoUseState_NoUpdateOnSameValue(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{scheduler: parseScheduler, currentRoot: &Fiber{}}

	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseState(parseRt, 42)

	set(42) // Same value

	if len(parseScheduler.callbacks) > 0 {
		parseT.Error("Expected no update scheduled for same value")
	}

	if get() != 42 {
		parseT.Errorf("Expected value to remain 42, got %d", get())
	}
}

func TestGoUseState_MultipleStates(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{scheduler: parseScheduler}

	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseGet1, parseSet1 := GoUseState(parseRt, "first")
	parseGet2, parseSet2 := GoUseState(parseRt, "second")

	if parseGet1() != "first" {
		parseT.Errorf("Expected first state 'first', got %s", parseGet1())
	}

	if parseGet2() != "second" {
		parseT.Errorf("Expected second state 'second', got %s", parseGet2())
	}

	parseSet1("updated-first")
	parseSet2("updated-second")

	if parseGet1() != "updated-first" {
		parseT.Errorf("Expected updated first state, got %s", parseGet1())
	}

	if parseGet2() != "updated-second" {
		parseT.Errorf("Expected updated second state, got %s", parseGet2())
	}
}

func TestGoUseEffect_RunsOnMount(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	isParseExecuted := false
	GoUseEffect(func() func() {
		isParseExecuted = true
		return func() {}
	})

	if len(parseFiber.effects) != 1 {
		parseT.Errorf("Expected 1 effect, got %d", len(parseFiber.effects))
	}

	// Execute effect
	parseFiber.effects[0].Fn()

	if !isParseExecuted {
		parseT.Error("Expected effect to be executed")
	}
}

func TestGoUseEffect_RunsOnDepsChange(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseCount := 0

	// First render with deps [1]
	GoUseEffect(func() func() {
		parseCount++
		return func() {}
	}, 1)

	if len(parseFiber.effects) != 1 {
		parseT.Fatalf("Expected 1 effect on first render, got %d", len(parseFiber.effects))
	}

	// Reset for "second render"
	parseFiber.effects = make([]Effect, 0)
	parseFiber.hooks.index = 0
	parseFiber.hooks.depIndex = 0
	parseFiber.hooks.cleanupIndex = 0

	// Second render with deps [2] (changed)
	GoUseEffect(func() func() {
		parseCount++
		return func() {}
	}, 2)

	if len(parseFiber.effects) != 1 {
		parseT.Errorf("Expected effect to run on deps change, got %d effects", len(parseFiber.effects))
	}
}

func TestGoUseEffect_SkipsOnSameDeps(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	// First render
	GoUseEffect(func() func() { return func() {} }, 1, 2, 3)

	// Reset for second render
	parseFiber.effects = make([]Effect, 0)
	parseFiber.hooks.index = 0
	parseFiber.hooks.depIndex = 0
	parseFiber.hooks.cleanupIndex = 0

	// Second render with same deps
	GoUseEffect(func() func() { return func() {} }, 1, 2, 3)

	if len(parseFiber.effects) != 0 {
		parseT.Errorf("Expected effect to be skipped with same deps, got %d effects", len(parseFiber.effects))
	}
}

func TestGoUseEffect_RerunsAfterHotRefresh(parseT *testing.T) {
	parseRt := &Runtime{}
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseRunCount := 0
	parseCleanupCount := 0

	GoUseEffect(func() func() {
		parseRunCount++
		return func() {
			parseCleanupCount++
		}
	}, "dep")

	if len(parseFiber.effects) != 1 {
		parseT.Fatalf("expected 1 queued effect, got %d", len(parseFiber.effects))
	}

	parseFirstCleanup := parseFiber.effects[0].Fn()
	if parseFirstCleanup == nil {
		parseT.Fatal("expected first effect to return a cleanup")
	}
	if parseRunCount != 1 {
		parseT.Fatalf("expected first effect to run once, got %d", parseRunCount)
	}
	parseFiber.hooks.cleanups[0] = parseFirstCleanup
	parseFiber.effects = nil
	parseFiber.hooks.index = 0
	parseFiber.hooks.depIndex = 0
	parseFiber.hooks.cleanupIndex = 0

	parseRt.RefreshEffectsForFiber(parseFiber)
	if parseCleanupCount != 1 {
		parseT.Fatalf("expected hot refresh to run cleanup once, got %d", parseCleanupCount)
	}

	GoUseEffect(func() func() {
		parseRunCount++
		return func() {
			parseCleanupCount++
		}
	}, "dep")

	if len(parseFiber.effects) != 1 {
		parseT.Fatalf("expected effect to rerun after hot refresh, got %d queued effects", len(parseFiber.effects))
	}
	if parseRunCount != 1 {
		parseT.Fatalf("expected effect body to remain deferred until commit, got runCount=%d", parseRunCount)
	}

	parseSecondCleanup := parseFiber.effects[0].Fn()
	if parseSecondCleanup == nil {
		parseT.Fatal("expected rerun effect to return a cleanup")
	}
	if parseRunCount != 2 {
		parseT.Fatalf("expected effect body to run again after refresh, got %d", parseRunCount)
	}
}

func TestGoUseMemo_ComputesOnce(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseComputeCount := 0
	parseResult := GoUseMemo(func() any {
		parseComputeCount++
		return 42
	}, "dep1")

	if parseResult != 42 {
		parseT.Errorf("Expected memoized value 42, got %v", parseResult)
	}

	if parseComputeCount != 1 {
		parseT.Errorf("Expected compute to be called once, got %d times", parseComputeCount)
	}
}

func TestGoUseMemo_RecomputesOnDepsChange(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseValue := 10
	parseResult1 := GoUseMemo(func() any {
		return parseValue * 2
	}, parseValue)

	// Reset for second render
	parseFiber.hooks.index = 0
	parseFiber.hooks.memoIndex = 0
	parseValue = 20

	parseResult2 := GoUseMemo(func() any {
		return parseValue * 2
	}, parseValue)

	if parseResult1 != 20 {
		parseT.Errorf("Expected first result 20, got %v", parseResult1)
	}

	if parseResult2 != 40 {
		parseT.Errorf("Expected second result 40, got %v", parseResult2)
	}
}

func TestGoUseMemo_SkipRecomputeOnSameDeps(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseComputeCount := 0

	// First render
	parseResult1 := GoUseMemo(func() any {
		parseComputeCount++
		return 42
	}, "dep")

	// Reset for second render with same deps
	parseFiber.hooks.index = 0
	parseFiber.hooks.memoIndex = 0

	parseResult2 := GoUseMemo(func() any {
		parseComputeCount++
		return 42
	}, "dep")

	if parseResult1 != 42 {
		parseT.Errorf("Expected first result 42, got %v", parseResult1)
	}

	if parseResult2 != 42 {
		parseT.Errorf("Expected second result 42, got %v", parseResult2)
	}

	if parseComputeCount != 1 {
		parseT.Errorf("Expected compute to be called once with same deps, got %d times", parseComputeCount)
	}
}

func TestGoUseMemo_NilDepsInitialization(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseComputeCount := 0

	// First render - memo.deps starts as nil
	parseResult := GoUseMemo(func() any {
		parseComputeCount++
		return "computed"
	}, "dep1")

	if parseResult != "computed" {
		parseT.Errorf("Expected computed value, got %v", parseResult)
	}

	if parseComputeCount != 1 {
		parseT.Errorf("Expected compute once on first render, got %d", parseComputeCount)
	}

	// Check that memo.deps was actually set
	if parseFiber.hooks.memos[0].deps == nil {
		parseT.Error("Expected memo.deps to be set after first render")
	}

	if len(parseFiber.hooks.memos[0].deps) != 1 {
		parseT.Errorf("Expected 1 dependency, got %d", len(parseFiber.hooks.memos[0].deps))
	}
}

func TestGoUseMemo_MultipleMemosIndependent(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseCount1 := 0
	parseCount2 := 0

	// First render - create two memos
	GoUseMemo(func() any {
		parseCount1++
		return "memo1"
	}, "dep1")

	GoUseMemo(func() any {
		parseCount2++
		return "memo2"
	}, "dep2")

	if parseCount1 != 1 || parseCount2 != 1 {
		parseT.Errorf("Expected each memo to compute once, got count1=%d count2=%d", parseCount1, parseCount2)
	}

	// Reset for second render - change dep2 only
	parseFiber.hooks.index = 0
	parseFiber.hooks.memoIndex = 0

	GoUseMemo(func() any {
		parseCount1++
		return "memo1"
	}, "dep1") // Same dep

	GoUseMemo(func() any {
		parseCount2++
		return "memo2"
	}, "dep2_changed") // Different dep

	if parseCount1 != 1 {
		parseT.Errorf("Expected memo1 to not recompute (count1 still 1), got %d", parseCount1)
	}

	if parseCount2 != 2 {
		parseT.Errorf("Expected memo2 to recompute (count2=2), got %d", parseCount2)
	}
}

func TestGoUseCallback_StableReference(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseTestFunc := func(parseX int) int {
		return parseX * 2
	}

	// First render
	parseResult1 := GoUseCallback(parseTestFunc, "dep1")

	// Verify it returns something
	if parseResult1 == nil {
		parseT.Error("Expected callback to return non-nil function")
	}

	// Reset for second render with same deps
	parseFiber.hooks.index = 0
	parseFiber.hooks.callbackIndex = 0

	parseResult2 := GoUseCallback(parseTestFunc, "dep1")

	// Verify both calls return non-nil
	if parseResult2 == nil {
		parseT.Error("Expected second callback to return non-nil function")
	}

	// Verify callback was memoized - deps should still be "dep1"
	if len(parseFiber.hooks.callbacks) != 1 {
		parseT.Errorf("Expected 1 callback stored, got %d", len(parseFiber.hooks.callbacks))
	}

	if len(parseFiber.hooks.callbacks[0].deps) != 1 || parseFiber.hooks.callbacks[0].deps[0] != "dep1" {
		parseT.Error("Expected callback deps to be set to [dep1]")
	}
}

func TestGoUseCallback_UpdatesOnDepsChange(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseFunc1 := func(parseX int) int { return parseX * 2 }
	parseFunc2 := func(parseX2 int) int { return parseX2 * 3 }

	// First render with func1
	GoUseCallback(parseFunc1, "dep1")

	parseFirstDeps := parseFiber.hooks.callbacks[0].deps

	// Reset for second render - change deps
	parseFiber.hooks.index = 0
	parseFiber.hooks.callbackIndex = 0

	GoUseCallback(parseFunc2, "dep1_changed")

	// Deps should have changed
	if len(parseFiber.hooks.callbacks[0].deps) != 1 || parseFiber.hooks.callbacks[0].deps[0] == parseFirstDeps[0] {
		parseT.Error("Expected callback deps to update when deps changed")
	}
}

func TestGoUseCallback_MultipleCallbacksIndependent(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseFunc1 := func() {}
	parseFunc2 := func() {}

	// First render - create two callbacks
	GoUseCallback(parseFunc1, "dep1")
	GoUseCallback(parseFunc2, "dep2")

	if len(parseFiber.hooks.callbacks) != 2 {
		parseT.Errorf("Expected 2 callbacks, got %d", len(parseFiber.hooks.callbacks))
	}

	parseDeps1 := parseFiber.hooks.callbacks[0].deps
	parseDeps2 := parseFiber.hooks.callbacks[1].deps

	// Reset for second render - change dep2 only
	parseFiber.hooks.index = 0
	parseFiber.hooks.callbackIndex = 0

	GoUseCallback(parseFunc1, "dep1")         // Same dep
	GoUseCallback(parseFunc2, "dep2_changed") // Different dep

	// First callback deps should remain unchanged
	if len(parseFiber.hooks.callbacks[0].deps) != 1 || parseFiber.hooks.callbacks[0].deps[0] != parseDeps1[0] {
		parseT.Error("Expected first callback deps to remain stable")
	}

	// Second callback deps should have changed
	if len(parseFiber.hooks.callbacks[1].deps) == 1 && parseFiber.hooks.callbacks[1].deps[0] == parseDeps2[0] {
		parseT.Error("Expected second callback deps to update")
	}
}

func TestAreDepsEqual_Primitives(parseT *testing.T) {
	parseTests := []struct {
		name     string
		prev     []any
		new      []any
		expected bool
	}{
		{"Same primitives", []any{1, 2, 3}, []any{1, 2, 3}, true},
		{"Different primitives", []any{1, 2}, []any{1, 3}, false},
		{"Different lengths", []any{1, 2}, []any{1}, false},
		{"Empty arrays", []any{}, []any{}, true},
		{"Strings equal", []any{"a", "b"}, []any{"a", "b"}, true},
		{"Strings different", []any{"a"}, []any{"b"}, false},
	}

	for _, parseTt := range parseTests {
		parseT.Run(parseTt.name, func(parseT2 *testing.T) {
			parseResult := areDepsEqual(parseTt.prev, parseTt.new)
			if parseResult != parseTt.expected {
				parseT2.Errorf("Expected %v, got %v", parseTt.expected, parseResult)
			}
		})
	}
}

func TestFastEqual(parseT *testing.T) {
	parseTests := []struct {
		name     string
		a        any
		b        any
		expected bool
	}{
		{"Both nil", nil, nil, true},
		{"One nil", nil, 5, false},
		{"Same ints", 42, 42, true},
		{"Different ints", 42, 43, false},
		{"Same strings", "hello", "hello", true},
		{"Different strings", "hello", "world", false},
		{"Same bool", true, true, true},
		{"Different bool", true, false, false},
	}

	for _, parseTt := range parseTests {
		parseT.Run(parseTt.name, func(parseT2 *testing.T) {
			parseResult := fastEqual(parseTt.a, parseTt.b)
			if parseResult != parseTt.expected {
				parseT2.Errorf("Expected %v, got %v", parseTt.expected, parseResult)
			}
		})
	}
}

func TestFastEqual_FunctionClosuresUseInstanceIdentity(parseT *testing.T) {
	parseFactory := func(parseValue int) func() int {
		return func() int { return parseValue }
	}

	parseFirst := parseFactory(1)
	parseSecond := parseFactory(2)
	parseAlias := parseFirst

	if !fastEqual(parseFirst, parseAlias) {
		parseT.Fatal("expected the same closure instance to compare equal")
	}
	if fastEqual(parseFirst, parseSecond) {
		parseT.Fatal("expected distinct closure instances to compare different")
	}
}

func TestGoUseRef_InitialValue(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	// Create ref with initial value
	parseRef := GoUseRef(42)

	if parseRef == nil {
		parseT.Fatal("Expected ref to be non-nil")
	}

	if parseRef.Current != 42 {
		parseT.Errorf("Expected ref.Current to be 42, got %v", parseRef.Current)
	}
}

func TestGoUseRef_PersistsAcrossRenders(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	// First render - create ref
	parseRef1 := GoUseRef("initial")

	if parseRef1.Current != "initial" {
		parseT.Errorf("Expected initial value 'initial', got %v", parseRef1.Current)
	}

	// Simulate mutation
	parseRef1.Current = "modified"

	// Reset hook index for second render
	parseFiber.hooks.index = 0
	parseFiber.hooks.refIndex = 0

	// Second render - get same ref
	parseRef2 := GoUseRef("initial") // Note: initial value is ignored on subsequent renders

	if parseRef2.Current != "modified" {
		parseT.Errorf("Expected persisted value 'modified', got %v", parseRef2.Current)
	}

	// Verify it's the same ref object
	if parseRef1 != parseRef2 {
		parseT.Error("Expected ref to be the same object across renders")
	}
}

func TestGoUseRef_MultipleRefsIndependent(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	// Create multiple refs
	parseRef1 := GoUseRef("ref1")
	parseRef2 := GoUseRef("ref2")
	parseRef3 := GoUseRef("ref3")

	if len(parseFiber.hooks.refs) != 3 {
		parseT.Errorf("Expected 3 refs stored, got %d", len(parseFiber.hooks.refs))
	}

	// Modify refs
	parseRef1.Current = "modified1"
	parseRef2.Current = "modified2"
	parseRef3.Current = "modified3"

	// Verify each ref maintains its own value
	if parseRef1.Current != "modified1" {
		parseT.Errorf("Expected ref1.Current to be 'modified1', got %v", parseRef1.Current)
	}

	if parseRef2.Current != "modified2" {
		parseT.Errorf("Expected ref2.Current to be 'modified2', got %v", parseRef2.Current)
	}

	if parseRef3.Current != "modified3" {
		parseT.Errorf("Expected ref3.Current to be 'modified3', got %v", parseRef3.Current)
	}

	// Reset for second render and verify independence
	parseFiber.hooks.index = 0
	parseFiber.hooks.refIndex = 0

	parseRef1Again := GoUseRef("ignored1")
	parseRef2Again := GoUseRef("ignored2")
	parseRef3Again := GoUseRef("ignored3")

	// All should retain their modified values
	if parseRef1Again.Current != "modified1" || parseRef2Again.Current != "modified2" || parseRef3Again.Current != "modified3" {
		parseT.Error("Expected all refs to retain their modified values independently")
	}
}

func TestGoUseRef_WithNilInitialValue(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	// Create ref with nil initial value
	parseRef := GoUseRef(nil)

	if parseRef == nil {
		parseT.Fatal("Expected ref object to be non-nil")
	}

	if parseRef.Current != nil {
		parseT.Errorf("Expected ref.Current to be nil, got %v", parseRef.Current)
	}

	// Assign a value later
	parseRef.Current = "assigned later"

	if parseRef.Current != "assigned later" {
		parseT.Errorf("Expected ref.Current to be 'assigned later', got %v", parseRef.Current)
	}
}

func TestGoUseId_GeneratesUniqueId(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	// Generate ID
	parseId := GoUseId()

	if parseId == "" {
		parseT.Error("Expected non-empty ID")
	}

	// Check format: "gwc:<number>:<position>"
	// Should start with "gwc:"
	if len(parseId) < 4 || parseId[:4] != "gwc:" {
		parseT.Errorf("Expected ID to start with 'gwc:', got %s", parseId)
	}

	// Should contain at least one colon after "gwc:"
	if !contains(parseId, ":") || len(parseId) <= 4 {
		parseT.Errorf("Expected ID to have format 'gwc:<number>:<position>', got %s", parseId)
	}
}

func TestGoUseId_PersistsAcrossRenders(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	// First render - generate ID
	parseId1 := GoUseId()

	if parseId1 == "" {
		parseT.Fatal("Expected non-empty ID on first render")
	}

	// Reset hook index for second render
	parseFiber.hooks.index = 0
	parseFiber.hooks.idIndex = 0

	// Second render - get same ID
	parseId2 := GoUseId()

	if parseId1 != parseId2 {
		parseT.Errorf("Expected ID to persist, got %s then %s", parseId1, parseId2)
	}
}

func TestGoUseId_MultipleIdsIndependent(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	// Generate multiple IDs
	parseId1 := GoUseId()
	parseId2 := GoUseId()
	parseId3 := GoUseId()

	if len(parseFiber.hooks.ids) != 3 {
		parseT.Errorf("Expected 3 IDs stored, got %d", len(parseFiber.hooks.ids))
	}

	// All IDs should be different
	if parseId1 == parseId2 {
		parseT.Error("Expected different IDs for first and second call")
	}

	if parseId2 == parseId3 {
		parseT.Error("Expected different IDs for second and third call")
	}

	if parseId1 == parseId3 {
		parseT.Error("Expected different IDs for first and third call")
	}

	// Reset for second render and verify independence
	parseFiber.hooks.index = 0
	parseFiber.hooks.idIndex = 0

	parseId1Again := GoUseId()
	parseId2Again := GoUseId()
	parseId3Again := GoUseId()

	// All should retain their IDs
	if parseId1Again != parseId1 || parseId2Again != parseId2 || parseId3Again != parseId3 {
		parseT.Error("Expected all IDs to persist across renders")
	}
}

func TestGoUseId_ContainsHookPosition(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]any),
	}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseId1 := GoUseId() // Position 0
	parseId2 := GoUseId() // Position 1
	parseId3 := GoUseId() // Position 2

	// IDs should contain their positions in the format
	if !contains(parseId1, ":0") {
		parseT.Errorf("Expected first ID to contain ':0', got %s", parseId1)
	}

	if !contains(parseId2, ":1") {
		parseT.Errorf("Expected second ID to contain ':1', got %s", parseId2)
	}

	if !contains(parseId3, ":2") {
		parseT.Errorf("Expected third ID to contain ':2', got %s", parseId3)
	}
}

func contains(parseS, parseSubstr string) bool {
	for parseI := 0; parseI <= len(parseS)-len(parseSubstr); parseI++ {
		if parseS[parseI:parseI+len(parseSubstr)] == parseSubstr {
			return true
		}
	}
	return false
}
