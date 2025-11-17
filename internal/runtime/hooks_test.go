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
	fiber.effects[0]()

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
	fiber.effects = make([]func(), 0)
	fiber.hooks.index = 0

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
	fiber.effects = make([]func(), 0)
	fiber.hooks.index = 0

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

	GoUseCallback(func1, "dep1") // Same dep
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

func TestValidateHookOrder_Success(t *testing.T) {
	hooks := &Hooks{
		callOrder: make([]HookCall, 0),
		prevOrder: []HookCall{
			{Type: HookTypeState, Position: 0},
			{Type: HookTypeEffect, Position: 1},
		},
	}

	err := validateHookOrder(hooks, HookTypeState, 0)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = validateHookOrder(hooks, HookTypeEffect, 1)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestValidateHookOrder_FailsOnMismatch(t *testing.T) {
	hooks := &Hooks{
		callOrder: make([]HookCall, 0),
		prevOrder: []HookCall{
			{Type: HookTypeState, Position: 0},
		},
	}

	err := validateHookOrder(hooks, HookTypeEffect, 0) // Wrong type
	if err == nil {
		t.Error("Expected validation error for type mismatch")
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

func TestFinalizeHookOrder(t *testing.T) {
	hooks := &Hooks{
		callOrder: []HookCall{
			{Type: HookTypeState, Position: 0},
			{Type: HookTypeEffect, Position: 1},
		},
		prevOrder: make([]HookCall, 0),
	}

	err := FinalizeHookOrder(hooks)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(hooks.prevOrder) != 2 {
		t.Errorf("Expected prevOrder length 2, got %d", len(hooks.prevOrder))
	}

	if len(hooks.callOrder) != 0 {
		t.Errorf("Expected callOrder to be reset, got length %d", len(hooks.callOrder))
	}

	if !hooks.orderChecked {
		t.Error("Expected orderChecked to be true")
	}
}
