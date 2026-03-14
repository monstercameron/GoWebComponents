package runtime

import (
	"sync"
	"testing"
)

// Helper to create test fibers with hooks initialized
func newTestFiber(typeOf string) *Fiber {
	return &Fiber{
		typeOf: typeOf,
		props:  make(map[string]interface{}),
		hooks:  &Hooks{},
	}
}

func TestAtomRegistry_InitAtom(t *testing.T) {
	registry := NewAtomRegistry()

	registry.InitAtom("counter", 0)

	value, ok := registry.GetAtom("counter")
	if !ok {
		t.Fatal("Expected atom to exist")
	}

	if value != 0 {
		t.Errorf("Expected value 0, got %v", value)
	}
}

func TestAtomRegistry_SetAndGet(t *testing.T) {
	registry := NewAtomRegistry()

	registry.SetAtom("message", "hello")

	value, ok := registry.GetAtom("message")
	if !ok {
		t.Fatal("Expected atom to exist")
	}

	if value != "hello" {
		t.Errorf("Expected 'hello', got %v", value)
	}
}

func TestAtomRegistry_Subscribe(t *testing.T) {
	registry := NewAtomRegistry()
	fiber := &Fiber{typeOf: "test"}

	registry.Subscribe("counter", fiber)

	count := registry.GetSubscriberCount("counter")
	if count != 1 {
		t.Errorf("Expected 1 subscriber, got %d", count)
	}
}

func TestAtomRegistry_MultipleSubscribers(t *testing.T) {
	registry := NewAtomRegistry()
	fiber1 := &Fiber{typeOf: "test1"}
	fiber2 := &Fiber{typeOf: "test2"}

	registry.Subscribe("counter", fiber1)
	registry.Subscribe("counter", fiber2)

	count := registry.GetSubscriberCount("counter")
	if count != 2 {
		t.Errorf("Expected 2 subscribers, got %d", count)
	}
}

func TestAtomRegistry_SetAtom_ReturnsSubscribers(t *testing.T) {
	registry := NewAtomRegistry()
	fiber1 := &Fiber{typeOf: "test1"}
	fiber2 := &Fiber{typeOf: "test2"}

	registry.Subscribe("counter", fiber1)
	registry.Subscribe("counter", fiber2)

	subscribers := registry.SetAtom("counter", 42)

	if len(subscribers) != 2 {
		t.Errorf("Expected 2 subscribers returned, got %d", len(subscribers))
	}
}

func TestAtomRegistry_Unsubscribe(t *testing.T) {
	registry := NewAtomRegistry()
	fiber := &Fiber{typeOf: "test"}

	registry.Subscribe("counter", fiber)
	registry.Unsubscribe("counter", fiber)

	count := registry.GetSubscriberCount("counter")
	if count != 0 {
		t.Errorf("Expected 0 subscribers, got %d", count)
	}
}

func TestAtomRegistry_UnsubscribeFiberFromAll(t *testing.T) {
	registry := NewAtomRegistry()
	fiber := &Fiber{typeOf: "test"}

	registry.Subscribe("counter", fiber)
	registry.Subscribe("message", fiber)

	registry.UnsubscribeFiberFromAll(fiber)

	count1 := registry.GetSubscriberCount("counter")
	count2 := registry.GetSubscriberCount("message")

	if count1 != 0 || count2 != 0 {
		t.Errorf("Expected 0 subscribers for all atoms, got %d and %d", count1, count2)
	}
}

func TestAtomRegistry_ThreadSafety(t *testing.T) {
	registry := NewAtomRegistry()
	var wg sync.WaitGroup

	// Spawn multiple goroutines reading/writing
	for i := 0; i < 10; i++ {
		id := i
		wg.Go(func() {
			fiber := &Fiber{typeOf: "test"}
			registry.Subscribe("counter", fiber)
			registry.SetAtom("counter", id)
			_, _ = registry.GetAtom("counter")
			registry.Unsubscribe("counter", fiber)
		})
	}

	wg.Wait()

	// Should not crash
}

func TestGoUseAtom_InitialValue(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})

	fiber := newTestFiber("test")
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	get, _ := GoUseAtom(rt, "counter", 100)

	if get() != 100 {
		t.Errorf("Expected initial value 100, got %d", get())
	}
}

func TestGoUseAtom_UpdateValue(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})

	fiber := newTestFiber("test")
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseAtom(rt, "counter", 0)

	set(42)

	if get() != 42 {
		t.Errorf("Expected updated value 42, got %d", get())
	}
}

func TestGoUseAtom_SharedAcrossComponents(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})

	// First component
	fiber1 := newTestFiber("comp1")
	SetCurrentFiber(fiber1)
	get1, set1 := GoUseAtom(rt, "shared", "initial")
	SetCurrentFiber(nil)

	// Second component
	fiber2 := newTestFiber("comp2")
	SetCurrentFiber(fiber2)
	get2, _ := GoUseAtom(rt, "shared", "initial")
	SetCurrentFiber(nil)

	// Update from first component
	set1("updated")

	// Both should see the same value
	if get1() != "updated" {
		t.Errorf("Component 1 expected 'updated', got %s", get1())
	}

	if get2() != "updated" {
		t.Errorf("Component 2 expected 'updated', got %s", get2())
	}
}

func TestGoUseAtom_SchedulesUpdatesForSubscribers(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})
	rt.currentRoot = &Fiber{}

	fiber1 := newTestFiber("comp1")
	SetCurrentFiber(fiber1)
	_, set := GoUseAtom(rt, "counter", 0)
	SetCurrentFiber(nil)

	fiber2 := newTestFiber("comp2")
	SetCurrentFiber(fiber2)
	GoUseAtom(rt, "counter", 0)
	SetCurrentFiber(nil)

	set(42)

	// Both fibers should have pending updates
	if !fiber1.needsUpdate {
		t.Error("Expected fiber1 to need update")
	}

	if !fiber2.needsUpdate {
		t.Error("Expected fiber2 to need update")
	}
}

func TestGoUseAtom_NoUpdateOnSameValue(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})
	rt.currentRoot = &Fiber{}

	fiber := newTestFiber("test")
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseAtom(rt, "counter", 42)

	set(42) // Same value

	if fiber.needsUpdate {
		t.Error("Expected no update scheduled for same value")
	}

	if get() != 42 {
		t.Errorf("Expected value to remain 42, got %d", get())
	}
}

func TestCleanupAtomSubscriptions(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})

	fiber := newTestFiber("test")
	SetCurrentFiber(fiber)
	GoUseAtom(rt, "counter", 0)
	SetCurrentFiber(nil)

	// Verify subscription exists
	count := rt.atomRegistry.GetSubscriberCount("counter")
	if count != 1 {
		t.Errorf("Expected 1 subscriber before cleanup, got %d", count)
	}

	// Cleanup
	rt.CleanupAtomSubscriptions(fiber)

	// Verify subscription removed
	count = rt.atomRegistry.GetSubscriberCount("counter")
	if count != 0 {
		t.Errorf("Expected 0 subscribers after cleanup, got %d", count)
	}
}

func TestRuntimeHelpers_GetAtomValue(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})

	rt.atomRegistry.SetAtom("test", "value")

	value, ok := rt.GetAtomValue("test")
	if !ok {
		t.Fatal("Expected atom to exist")
	}

	if value != "value" {
		t.Errorf("Expected 'value', got %v", value)
	}
}

func TestRuntimeHelpers_SetAtomValue(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})
	rt.currentRoot = &Fiber{}

	fiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	rt.atomRegistry.Subscribe("test", fiber)

	err := rt.SetAtomValue("test", 123)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	value, _ := rt.GetAtomValue("test")
	if value != 123 {
		t.Errorf("Expected 123, got %v", value)
	}

	if !fiber.needsUpdate {
		t.Error("Expected fiber to need update")
	}
}

func TestAtomRegistry_GetAtomCount(t *testing.T) {
	registry := NewAtomRegistry()

	registry.SetAtom("atom1", 1)
	registry.SetAtom("atom2", 2)
	registry.SetAtom("atom3", 3)

	count := registry.GetAtomCount()
	if count != 3 {
		t.Errorf("Expected 3 atoms, got %d", count)
	}
}
