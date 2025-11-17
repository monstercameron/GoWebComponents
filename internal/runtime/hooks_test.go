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
	GoUseEffect(func() {
		executed = true
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
	GoUseEffect(func() {
		count++
	}, 1)
	
	if len(fiber.effects) != 1 {
		t.Fatalf("Expected 1 effect on first render, got %d", len(fiber.effects))
	}
	
	// Reset for "second render"
	fiber.effects = make([]func(), 0)
	fiber.hooks.index = 0
	
	// Second render with deps [2] (changed)
	GoUseEffect(func() {
		count++
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
	GoUseEffect(func() {}, 1, 2, 3)
	
	// Reset for second render
	fiber.effects = make([]func(), 0)
	fiber.hooks.index = 0
	
	// Second render with same deps
	GoUseEffect(func() {}, 1, 2, 3)
	
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
