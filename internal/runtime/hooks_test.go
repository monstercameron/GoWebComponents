package runtime

import (
	"testing"
)

func TestGoUseState_InitialValue(t *testing.T) {
	scheduler := newTestScheduler()
	rt := &Runtime{scheduler: scheduler}

	// Create a fiber context
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	get, _ := GoUseState(rt, 42)

	if get() != 42 {
		t.Errorf("Expected initial value 42, got %d", get())
	}
}

func TestGoUseState_UpdateValue(t *testing.T) {
	scheduler := newTestScheduler()
	rt := &Runtime{scheduler: scheduler}

	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseState(rt, 0)

	set(10)

	if get() != 10 {
		t.Errorf("Expected updated value 10, got %d", get())
	}
}

func TestGoUseState_NoUpdateOnSameValue(t *testing.T) {
	scheduler := newTestScheduler()
	rt := &Runtime{scheduler: scheduler, currentRoot: &Fiber{}}

	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseState(rt, 42)

	set(42) // Same value

	if len(scheduler.callbacks) > 0 {
		t.Error("Expected no update scheduled for same value")
	}

	if get() != 42 {
		t.Errorf("Expected value to remain 42, got %d", get())
	}
}

func TestGoUseState_MultipleStates(t *testing.T) {
	scheduler := newTestScheduler()
	rt := &Runtime{scheduler: scheduler}

	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	get1, set1 := GoUseState(rt, "first")
	get2, set2 := GoUseState(rt, "second")

	if get1() != "first" {
		t.Errorf("Expected first state 'first', got %s", get1())
	}

	if get2() != "second" {
		t.Errorf("Expected second state 'second', got %s", get2())
	}

	set1("updated-first")
	set2("updated-second")

	if get1() != "updated-first" {
		t.Errorf("Expected updated first state, got %s", get1())
	}

	if get2() != "updated-second" {
		t.Errorf("Expected updated second state, got %s", get2())
	}
}

func TestGoUseEffect_RunsOnMount(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	executed := false
	GoUseEffect(func() func() {
		executed = true
		return func() {}
	})

	if len(fiber.effects) != 1 {
		t.Errorf("Expected 1 effect, got %d", len(fiber.effects))
	}

	// Execute effect
	fiber.effects[0].Fn()

	if !executed {
		t.Error("Expected effect to be executed")
	}
}

func TestGoUseEffect_RunsOnDepsChange(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	count := 0

	// First render with deps [1]
	GoUseEffect(func() func() {
		count++
		return func() {}
	}, 1)

	if len(fiber.effects) != 1 {
		t.Fatalf("Expected 1 effect on first render, got %d", len(fiber.effects))
	}

	// Reset for "second render"
	fiber.effects = make([]Effect, 0)
	fiber.hooks.index = 0
	fiber.hooks.depIndex = 0
	fiber.hooks.cleanupIndex = 0

	// Second render with deps [2] (changed)
	GoUseEffect(func() func() {
		count++
		return func() {}
	}, 2)

	if len(fiber.effects) != 1 {
		t.Errorf("Expected effect to run on deps change, got %d effects", len(fiber.effects))
	}
}

func TestGoUseEffect_SkipsOnSameDeps(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	// First render
	GoUseEffect(func() func() { return func() {} }, 1, 2, 3)

	// Reset for second render
	fiber.effects = make([]Effect, 0)
	fiber.hooks.index = 0
	fiber.hooks.depIndex = 0
	fiber.hooks.cleanupIndex = 0

	// Second render with same deps
	GoUseEffect(func() func() { return func() {} }, 1, 2, 3)

	if len(fiber.effects) != 0 {
		t.Errorf("Expected effect to be skipped with same deps, got %d effects", len(fiber.effects))
	}
}

func TestGoUseMemo_ComputesOnce(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	computeCount := 0
	result := GoUseMemo(func() interface{} {
		computeCount++
		return 42
	}, "dep1")

	if result != 42 {
		t.Errorf("Expected memoized value 42, got %v", result)
	}

	if computeCount != 1 {
		t.Errorf("Expected compute to be called once, got %d times", computeCount)
	}
}

func TestGoUseMemo_RecomputesOnDepsChange(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	value := 10
	result1 := GoUseMemo(func() interface{} {
		return value * 2
	}, value)

	// Reset for second render
	fiber.hooks.index = 0
	fiber.hooks.memoIndex = 0
	value = 20

	result2 := GoUseMemo(func() interface{} {
		return value * 2
	}, value)

	if result1 != 20 {
		t.Errorf("Expected first result 20, got %v", result1)
	}

	if result2 != 40 {
		t.Errorf("Expected second result 40, got %v", result2)
	}
}

func TestGoUseMemo_SkipRecomputeOnSameDeps(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	computeCount := 0

	// First render
	result1 := GoUseMemo(func() interface{} {
		computeCount++
		return 42
	}, "dep")

	// Reset for second render with same deps
	fiber.hooks.index = 0
	fiber.hooks.memoIndex = 0

	result2 := GoUseMemo(func() interface{} {
		computeCount++
		return 42
	}, "dep")

	if result1 != 42 {
		t.Errorf("Expected first result 42, got %v", result1)
	}

	if result2 != 42 {
		t.Errorf("Expected second result 42, got %v", result2)
	}

	if computeCount != 1 {
		t.Errorf("Expected compute to be called once with same deps, got %d times", computeCount)
	}
}

func TestGoUseMemo_NilDepsInitialization(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	computeCount := 0

	// First render - memo.deps starts as nil
	result := GoUseMemo(func() interface{} {
		computeCount++
		return "computed"
	}, "dep1")

	if result != "computed" {
		t.Errorf("Expected computed value, got %v", result)
	}

	if computeCount != 1 {
		t.Errorf("Expected compute once on first render, got %d", computeCount)
	}

	// Check that memo.deps was actually set
	if fiber.hooks.memos[0].deps == nil {
		t.Error("Expected memo.deps to be set after first render")
	}

	if len(fiber.hooks.memos[0].deps) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(fiber.hooks.memos[0].deps))
	}
}

func TestGoUseMemo_MultipleMemosIndependent(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	count1 := 0
	count2 := 0

	// First render - create two memos
	GoUseMemo(func() interface{} {
		count1++
		return "memo1"
	}, "dep1")

	GoUseMemo(func() interface{} {
		count2++
		return "memo2"
	}, "dep2")

	if count1 != 1 || count2 != 1 {
		t.Errorf("Expected each memo to compute once, got count1=%d count2=%d", count1, count2)
	}

	// Reset for second render - change dep2 only
	fiber.hooks.index = 0
	fiber.hooks.memoIndex = 0

	GoUseMemo(func() interface{} {
		count1++
		return "memo1"
	}, "dep1") // Same dep

	GoUseMemo(func() interface{} {
		count2++
		return "memo2"
	}, "dep2_changed") // Different dep

	if count1 != 1 {
		t.Errorf("Expected memo1 to not recompute (count1 still 1), got %d", count1)
	}

	if count2 != 2 {
		t.Errorf("Expected memo2 to recompute (count2=2), got %d", count2)
	}
}

func TestGoUseCallback_StableReference(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	testFunc := func(x int) int {
		return x * 2
	}

	// First render
	result1 := GoUseCallback(testFunc, "dep1")

	// Verify it returns something
	if result1 == nil {
		t.Error("Expected callback to return non-nil function")
	}

	// Reset for second render with same deps
	fiber.hooks.index = 0
	fiber.hooks.callbackIndex = 0

	result2 := GoUseCallback(testFunc, "dep1")

	// Verify both calls return non-nil
	if result2 == nil {
		t.Error("Expected second callback to return non-nil function")
	}

	// Verify callback was memoized - deps should still be "dep1"
	if len(fiber.hooks.callbacks) != 1 {
		t.Errorf("Expected 1 callback stored, got %d", len(fiber.hooks.callbacks))
	}

	if len(fiber.hooks.callbacks[0].deps) != 1 || fiber.hooks.callbacks[0].deps[0] != "dep1" {
		t.Error("Expected callback deps to be set to [dep1]")
	}
}

func TestGoUseCallback_UpdatesOnDepsChange(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	func1 := func(x int) int { return x * 2 }
	func2 := func(x int) int { return x * 3 }

	// First render with func1
	GoUseCallback(func1, "dep1")

	firstDeps := fiber.hooks.callbacks[0].deps

	// Reset for second render - change deps
	fiber.hooks.index = 0
	fiber.hooks.callbackIndex = 0

	GoUseCallback(func2, "dep1_changed")

	// Deps should have changed
	if len(fiber.hooks.callbacks[0].deps) != 1 || fiber.hooks.callbacks[0].deps[0] == firstDeps[0] {
		t.Error("Expected callback deps to update when deps changed")
	}
}

func TestGoUseCallback_MultipleCallbacksIndependent(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	func1 := func() {}
	func2 := func() {}

	// First render - create two callbacks
	GoUseCallback(func1, "dep1")
	GoUseCallback(func2, "dep2")

	if len(fiber.hooks.callbacks) != 2 {
		t.Errorf("Expected 2 callbacks, got %d", len(fiber.hooks.callbacks))
	}

	deps1 := fiber.hooks.callbacks[0].deps
	deps2 := fiber.hooks.callbacks[1].deps

	// Reset for second render - change dep2 only
	fiber.hooks.index = 0
	fiber.hooks.callbackIndex = 0

	GoUseCallback(func1, "dep1")         // Same dep
	GoUseCallback(func2, "dep2_changed") // Different dep

	// First callback deps should remain unchanged
	if len(fiber.hooks.callbacks[0].deps) != 1 || fiber.hooks.callbacks[0].deps[0] != deps1[0] {
		t.Error("Expected first callback deps to remain stable")
	}

	// Second callback deps should have changed
	if len(fiber.hooks.callbacks[1].deps) == 1 && fiber.hooks.callbacks[1].deps[0] == deps2[0] {
		t.Error("Expected second callback deps to update")
	}
}



func TestAreDepsEqual_Primitives(t *testing.T) {
	tests := []struct {
		name     string
		prev     []interface{}
		new      []interface{}
		expected bool
	}{
		{"Same primitives", []interface{}{1, 2, 3}, []interface{}{1, 2, 3}, true},
		{"Different primitives", []interface{}{1, 2}, []interface{}{1, 3}, false},
		{"Different lengths", []interface{}{1, 2}, []interface{}{1}, false},
		{"Empty arrays", []interface{}{}, []interface{}{}, true},
		{"Strings equal", []interface{}{"a", "b"}, []interface{}{"a", "b"}, true},
		{"Strings different", []interface{}{"a"}, []interface{}{"b"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := areDepsEqual(tt.prev, tt.new)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFastEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        interface{}
		b        interface{}
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fastEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}



func TestGoUseRef_InitialValue(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	// Create ref with initial value
	ref := GoUseRef(42)

	if ref == nil {
		t.Fatal("Expected ref to be non-nil")
	}

	if ref.Current != 42 {
		t.Errorf("Expected ref.Current to be 42, got %v", ref.Current)
	}
}

func TestGoUseRef_PersistsAcrossRenders(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	// First render - create ref
	ref1 := GoUseRef("initial")

	if ref1.Current != "initial" {
		t.Errorf("Expected initial value 'initial', got %v", ref1.Current)
	}

	// Simulate mutation
	ref1.Current = "modified"

	// Reset hook index for second render
	fiber.hooks.index = 0
	fiber.hooks.refIndex = 0

	// Second render - get same ref
	ref2 := GoUseRef("initial") // Note: initial value is ignored on subsequent renders

	if ref2.Current != "modified" {
		t.Errorf("Expected persisted value 'modified', got %v", ref2.Current)
	}

	// Verify it's the same ref object
	if ref1 != ref2 {
		t.Error("Expected ref to be the same object across renders")
	}
}

func TestGoUseRef_MultipleRefsIndependent(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	// Create multiple refs
	ref1 := GoUseRef("ref1")
	ref2 := GoUseRef("ref2")
	ref3 := GoUseRef("ref3")

	if len(fiber.hooks.refs) != 3 {
		t.Errorf("Expected 3 refs stored, got %d", len(fiber.hooks.refs))
	}

	// Modify refs
	ref1.Current = "modified1"
	ref2.Current = "modified2"
	ref3.Current = "modified3"

	// Verify each ref maintains its own value
	if ref1.Current != "modified1" {
		t.Errorf("Expected ref1.Current to be 'modified1', got %v", ref1.Current)
	}

	if ref2.Current != "modified2" {
		t.Errorf("Expected ref2.Current to be 'modified2', got %v", ref2.Current)
	}

	if ref3.Current != "modified3" {
		t.Errorf("Expected ref3.Current to be 'modified3', got %v", ref3.Current)
	}

	// Reset for second render and verify independence
	fiber.hooks.index = 0
	fiber.hooks.refIndex = 0

	ref1Again := GoUseRef("ignored1")
	ref2Again := GoUseRef("ignored2")
	ref3Again := GoUseRef("ignored3")

	// All should retain their modified values
	if ref1Again.Current != "modified1" || ref2Again.Current != "modified2" || ref3Again.Current != "modified3" {
		t.Error("Expected all refs to retain their modified values independently")
	}
}

func TestGoUseRef_WithNilInitialValue(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	// Create ref with nil initial value
	ref := GoUseRef(nil)

	if ref == nil {
		t.Fatal("Expected ref object to be non-nil")
	}

	if ref.Current != nil {
		t.Errorf("Expected ref.Current to be nil, got %v", ref.Current)
	}

	// Assign a value later
	ref.Current = "assigned later"

	if ref.Current != "assigned later" {
		t.Errorf("Expected ref.Current to be 'assigned later', got %v", ref.Current)
	}
}

func TestGoUseId_GeneratesUniqueId(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	// Generate ID
	id := GoUseId()

	if id == "" {
		t.Error("Expected non-empty ID")
	}

	// Check format: "gwc:<number>:<position>"
	// Should start with "gwc:"
	if len(id) < 4 || id[:4] != "gwc:" {
		t.Errorf("Expected ID to start with 'gwc:', got %s", id)
	}

	// Should contain at least one colon after "gwc:"
	if !contains(id, ":") || len(id) <= 4 {
		t.Errorf("Expected ID to have format 'gwc:<number>:<position>', got %s", id)
	}
}

func TestGoUseId_PersistsAcrossRenders(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	// First render - generate ID
	id1 := GoUseId()

	if id1 == "" {
		t.Fatal("Expected non-empty ID on first render")
	}

	// Reset hook index for second render
	fiber.hooks.index = 0
	fiber.hooks.idIndex = 0

	// Second render - get same ID
	id2 := GoUseId()

	if id1 != id2 {
		t.Errorf("Expected ID to persist, got %s then %s", id1, id2)
	}
}

func TestGoUseId_MultipleIdsIndependent(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	// Generate multiple IDs
	id1 := GoUseId()
	id2 := GoUseId()
	id3 := GoUseId()

	if len(fiber.hooks.ids) != 3 {
		t.Errorf("Expected 3 IDs stored, got %d", len(fiber.hooks.ids))
	}

	// All IDs should be different
	if id1 == id2 {
		t.Error("Expected different IDs for first and second call")
	}

	if id2 == id3 {
		t.Error("Expected different IDs for second and third call")
	}

	if id1 == id3 {
		t.Error("Expected different IDs for first and third call")
	}

	// Reset for second render and verify independence
	fiber.hooks.index = 0
	fiber.hooks.idIndex = 0

	id1Again := GoUseId()
	id2Again := GoUseId()
	id3Again := GoUseId()

	// All should retain their IDs
	if id1Again != id1 || id2Again != id2 || id3Again != id3 {
		t.Error("Expected all IDs to persist across renders")
	}
}

func TestGoUseId_ContainsHookPosition(t *testing.T) {
	fiber := &Fiber{
		typeOf: "test",
		props:  make(map[string]interface{}),
	}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	id1 := GoUseId() // Position 0
	id2 := GoUseId() // Position 1
	id3 := GoUseId() // Position 2

	// IDs should contain their positions in the format
	if !contains(id1, ":0") {
		t.Errorf("Expected first ID to contain ':0', got %s", id1)
	}

	if !contains(id2, ":1") {
		t.Errorf("Expected second ID to contain ':1', got %s", id2)
	}

	if !contains(id3, ":2") {
		t.Errorf("Expected third ID to contain ':2', got %s", id3)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
