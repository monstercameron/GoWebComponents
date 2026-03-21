package runtime

import (
	"fmt"
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

func TestAtomRegistry_MoveSubscriptions(t *testing.T) {
	registry := NewAtomRegistry()
	from := &Fiber{typeOf: "from"}
	to := &Fiber{typeOf: "to"}

	registry.Subscribe("counter", from)
	registry.Subscribe("message", from)
	registry.MoveSubscriptions([]string{"counter", "message"}, from, to)

	if registry.GetSubscriberCount("counter") != 1 {
		t.Fatalf("expected one counter subscriber after move, got %d", registry.GetSubscriberCount("counter"))
	}
	if registry.GetSubscriberCount("message") != 1 {
		t.Fatalf("expected one message subscriber after move, got %d", registry.GetSubscriberCount("message"))
	}

	registry.Unsubscribe("counter", to)
	registry.Unsubscribe("message", to)
	if registry.GetSubscriberCount("counter") != 0 || registry.GetSubscriberCount("message") != 0 {
		t.Fatal("expected moved subscriptions to be owned by the destination fiber")
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

func TestAtomRegistry_SnapshotReturnsCopy(t *testing.T) {
	registry := NewAtomRegistry()
	registry.SetAtom("theme", "dark")

	snapshot := registry.Snapshot()
	snapshot["theme"] = "light"

	value, _ := registry.GetAtom("theme")
	if value != "dark" {
		t.Fatalf("expected registry atom to remain dark, got %#v", value)
	}
}

func TestRuntimeRestoreAtomSnapshotSchedulesSubscribers(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})
	rt.currentRoot = &Fiber{}

	fiber := newTestFiber("subscriber")
	rt.atomRegistry.Subscribe("theme", fiber)

	if err := rt.RestoreAtomSnapshot(map[string]interface{}{"theme": "dark"}); err != nil {
		t.Fatalf("unexpected restore error: %v", err)
	}

	value, _ := rt.GetAtomValue("theme")
	if value != "dark" {
		t.Fatalf("expected restored atom value dark, got %#v", value)
	}
	if !fiber.needsUpdate {
		t.Fatal("expected subscribed fiber to be marked for update after restore")
	}
}

func TestRuntimeHelpers_SetAtomValueTransitionDefersUntilTimeout(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})
	rt.currentRoot = &Fiber{}

	fiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	rt.atomRegistry.Subscribe("test", fiber)

	rt.StartTransition(func() {
		if err := rt.SetAtomValue("test", 123); err != nil {
			t.Fatalf("unexpected transition set atom error: %v", err)
		}
	})

	if value, _ := rt.GetAtomValue("test"); value != nil {
		t.Fatalf("expected transition atom value to stay deferred before timeout, got %#v", value)
	}
	if fiber.needsUpdate {
		t.Fatal("expected subscribed fiber to stay clean before transition timeout")
	}
	if len(scheduler.timeouts) != 1 {
		t.Fatalf("expected one deferred timeout for direct atom write, got %d", len(scheduler.timeouts))
	}

	scheduler.timeouts[0]()

	if value, _ := rt.GetAtomValue("test"); value != 123 {
		t.Fatalf("expected deferred atom value 123 after timeout, got %#v", value)
	}
	if !fiber.needsUpdate {
		t.Fatal("expected subscribed fiber to be marked after deferred atom write")
	}
}

func TestRuntimeRestoreAtomSnapshotTransitionDefersUntilTimeout(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})
	rt.currentRoot = &Fiber{}

	fiber := newTestFiber("theme-subscriber")
	rt.atomRegistry.Subscribe("theme", fiber)

	rt.StartTransition(func() {
		if err := rt.RestoreAtomSnapshot(map[string]interface{}{"theme": "dark"}); err != nil {
			t.Fatalf("unexpected transition restore error: %v", err)
		}
	})

	if value, _ := rt.GetAtomValue("theme"); value != nil {
		t.Fatalf("expected restored atom value to stay deferred before timeout, got %#v", value)
	}
	if fiber.needsUpdate {
		t.Fatal("expected subscribed fiber to stay clean before deferred restore flushes")
	}
	if len(scheduler.timeouts) != 1 {
		t.Fatalf("expected one deferred timeout for snapshot restore, got %d", len(scheduler.timeouts))
	}

	scheduler.timeouts[0]()

	if value, _ := rt.GetAtomValue("theme"); value != "dark" {
		t.Fatalf("expected restored atom value dark after timeout, got %#v", value)
	}
	if !fiber.needsUpdate {
		t.Fatal("expected subscribed fiber to be marked after deferred restore")
	}
}

func TestRegisterDerivedAtomRecomputesWhenDependencyChanges(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})
	rt.currentRoot = &Fiber{}

	if err := rt.SetAtomValue("count", 2); err != nil {
		t.Fatalf("unexpected set atom error: %v", err)
	}
	if err := rt.RegisterDerivedAtom("double", []string{"count"}, func() interface{} {
		value, _ := rt.GetAtomValue("count")
		return value.(int) * 2
	}); err != nil {
		t.Fatalf("unexpected register derived atom error: %v", err)
	}

	derivedFiber := newTestFiber("derived-subscriber")
	rt.atomRegistry.Subscribe("double", derivedFiber)

	value, _ := rt.GetAtomValue("double")
	if value != 4 {
		t.Fatalf("expected initial derived value 4, got %#v", value)
	}

	if err := rt.SetAtomValue("count", 5); err != nil {
		t.Fatalf("unexpected update atom error: %v", err)
	}
	value, _ = rt.GetAtomValue("double")
	if value != 10 {
		t.Fatalf("expected derived value 10 after source update, got %#v", value)
	}
	if !derivedFiber.needsUpdate {
		t.Fatal("expected derived subscribers to be scheduled after source update")
	}
}

func TestRegisterDerivedAtomSupportsChainedDependencies(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})

	_ = rt.SetAtomValue("base", 3)
	if err := rt.RegisterDerivedAtom("double", []string{"base"}, func() interface{} {
		value, _ := rt.GetAtomValue("base")
		return value.(int) * 2
	}); err != nil {
		t.Fatalf("unexpected register double error: %v", err)
	}
	if err := rt.RegisterDerivedAtom("label", []string{"double"}, func() interface{} {
		value, _ := rt.GetAtomValue("double")
		return fmt.Sprintf("value:%d", value.(int))
	}); err != nil {
		t.Fatalf("unexpected register label error: %v", err)
	}

	_ = rt.SetAtomValue("base", 4)
	value, _ := rt.GetAtomValue("label")
	if value != "value:8" {
		t.Fatalf("expected chained derived value value:8, got %#v", value)
	}
}

func TestRegisterDerivedAtomSkipsSubscriberNotifyWhenValueUnchanged(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})
	rt.currentRoot = &Fiber{}

	_ = rt.SetAtomValue("count", 1)
	if err := rt.RegisterDerivedAtom("parity", []string{"count"}, func() interface{} {
		value, _ := rt.GetAtomValue("count")
		return value.(int) % 2
	}); err != nil {
		t.Fatalf("unexpected register parity error: %v", err)
	}

	derivedFiber := newTestFiber("parity-subscriber")
	rt.atomRegistry.Subscribe("parity", derivedFiber)

	if err := rt.SetAtomValue("count", 3); err != nil {
		t.Fatalf("unexpected count update error: %v", err)
	}
	if derivedFiber.needsUpdate || derivedFiber.dirty {
		t.Fatal("expected unchanged derived value to avoid notifying subscribers")
	}

	if err := rt.SetAtomValue("count", 4); err != nil {
		t.Fatalf("unexpected second count update error: %v", err)
	}
	if !derivedFiber.needsUpdate {
		t.Fatal("expected changed derived value to notify subscribers")
	}
}

func TestRegisterDerivedAtomRejectsSimpleCycles(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})

	err := rt.RegisterDerivedAtom("loop", []string{"loop"}, func() interface{} { return 1 })
	if err == nil {
		t.Fatal("expected self-referential derived atom registration to fail")
	}
}

func TestRegisterDerivedAtomRejectsIndirectCycles(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})

	if err := rt.RegisterDerivedAtom("derived-a", []string{"derived-b"}, func() interface{} { return 1 }); err != nil {
		t.Fatalf("unexpected register derived-a error: %v", err)
	}
	err := rt.RegisterDerivedAtom("derived-b", []string{"derived-a"}, func() interface{} { return 2 })
	if err == nil {
		t.Fatal("expected indirect derived cycle registration to fail")
	}
}
