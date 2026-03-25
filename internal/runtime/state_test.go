package runtime

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestGoUseAtomRegistryPanicUsesUnifiedContract(parseT *testing.T) {
	defer func() {
		parseRecovered := recover()
		if parseRecovered == nil {
			parseT.Fatal("expected GoUseAtom to panic when registry is missing")
		}
		parseMessage := parseRecovered.(string)
		if !strings.Contains(parseMessage, "GWC-RUNTIME-ATOM-REGISTRY-NIL") || !strings.Contains(parseMessage, "where:") || !strings.Contains(parseMessage, "runtime:") || !strings.Contains(parseMessage, "docs: ACTIONABLE_ERRORS.md#gwc-runtime-atom-registry-nil") {
			parseT.Fatalf("expected unified atom registry panic output, got %q", parseMessage)
		}
	}()

	_, _ = GoUseAtom(&Runtime{}, "counter", 0)
}

// newTestFiber creates test fibers with hooks initialized.
func newTestFiber(parseTypeOf string) *Fiber {
	return &Fiber{
		typeOf: parseTypeOf,
		props:  make(map[string]interface{}),
		hooks:  &Hooks{},
	}
}

func TestAtomRegistry_InitAtom(parseT *testing.T) {
	parseRegistry := NewAtomRegistry()

	parseRegistry.InitAtom("counter", 0)

	parseValue, parseOk := parseRegistry.GetAtom("counter")
	if !parseOk {
		parseT.Fatal("Expected atom to exist")
	}

	if parseValue != 0 {
		parseT.Errorf("Expected value 0, got %v", parseValue)
	}
}

func TestAtomRegistry_SetAndGet(parseT *testing.T) {
	parseRegistry := NewAtomRegistry()

	parseRegistry.SetAtom("message", "hello")

	parseValue, parseOk := parseRegistry.GetAtom("message")
	if !parseOk {
		parseT.Fatal("Expected atom to exist")
	}

	if parseValue != "hello" {
		parseT.Errorf("Expected 'hello', got %v", parseValue)
	}
}

func TestAtomRegistry_Subscribe(parseT *testing.T) {
	parseRegistry := NewAtomRegistry()
	parseFiber := &Fiber{typeOf: "test"}

	parseRegistry.Subscribe("counter", parseFiber)

	parseCount := parseRegistry.GetSubscriberCount("counter")
	if parseCount != 1 {
		parseT.Errorf("Expected 1 subscriber, got %d", parseCount)
	}
}

func TestAtomRegistry_MultipleSubscribers(parseT *testing.T) {
	parseRegistry := NewAtomRegistry()
	parseFiber1 := &Fiber{typeOf: "test1"}
	parseFiber2 := &Fiber{typeOf: "test2"}

	parseRegistry.Subscribe("counter", parseFiber1)
	parseRegistry.Subscribe("counter", parseFiber2)

	parseCount := parseRegistry.GetSubscriberCount("counter")
	if parseCount != 2 {
		parseT.Errorf("Expected 2 subscribers, got %d", parseCount)
	}
}

func TestAtomRegistry_SetAtom_ReturnsSubscribers(parseT *testing.T) {
	parseRegistry := NewAtomRegistry()
	parseFiber1 := &Fiber{typeOf: "test1"}
	parseFiber2 := &Fiber{typeOf: "test2"}

	parseRegistry.Subscribe("counter", parseFiber1)
	parseRegistry.Subscribe("counter", parseFiber2)

	parseSubscribers := parseRegistry.SetAtom("counter", 42)

	if len(parseSubscribers) != 2 {
		parseT.Errorf("Expected 2 subscribers returned, got %d", len(parseSubscribers))
	}
}

func TestAtomRegistry_Unsubscribe(parseT *testing.T) {
	parseRegistry := NewAtomRegistry()
	parseFiber := &Fiber{typeOf: "test"}

	parseRegistry.Subscribe("counter", parseFiber)
	parseRegistry.Unsubscribe("counter", parseFiber)

	parseCount := parseRegistry.GetSubscriberCount("counter")
	if parseCount != 0 {
		parseT.Errorf("Expected 0 subscribers, got %d", parseCount)
	}
}

func TestAtomRegistry_MoveSubscriptions(parseT *testing.T) {
	parseRegistry := NewAtomRegistry()
	parseFrom := &Fiber{typeOf: "from"}
	parseTo := &Fiber{typeOf: "to"}

	parseRegistry.Subscribe("counter", parseFrom)
	parseRegistry.Subscribe("message", parseFrom)
	parseRegistry.MoveSubscriptions([]string{"counter", "message"}, parseFrom, parseTo)

	if parseRegistry.GetSubscriberCount("counter") != 1 {
		parseT.Fatalf("expected one counter subscriber after move, got %d", parseRegistry.GetSubscriberCount("counter"))
	}
	if parseRegistry.GetSubscriberCount("message") != 1 {
		parseT.Fatalf("expected one message subscriber after move, got %d", parseRegistry.GetSubscriberCount("message"))
	}

	parseRegistry.Unsubscribe("counter", parseTo)
	parseRegistry.Unsubscribe("message", parseTo)
	if parseRegistry.GetSubscriberCount("counter") != 0 || parseRegistry.GetSubscriberCount("message") != 0 {
		parseT.Fatal("expected moved subscriptions to be owned by the destination fiber")
	}
}

func TestAtomRegistry_UnsubscribeFiberFromAll(parseT *testing.T) {
	parseRegistry := NewAtomRegistry()
	parseFiber := &Fiber{typeOf: "test"}

	parseRegistry.Subscribe("counter", parseFiber)
	parseRegistry.Subscribe("message", parseFiber)

	parseRegistry.UnsubscribeFiberFromAll(parseFiber)

	parseCount1 := parseRegistry.GetSubscriberCount("counter")
	parseCount2 := parseRegistry.GetSubscriberCount("message")

	if parseCount1 != 0 || parseCount2 != 0 {
		parseT.Errorf("Expected 0 subscribers for all atoms, got %d and %d", parseCount1, parseCount2)
	}
}

func TestAtomRegistry_ThreadSafety(parseT *testing.T) {
	parseRegistry := NewAtomRegistry()
	var parseWg sync.WaitGroup

	// Spawn multiple goroutines reading/writing
	for parseI := 0; parseI < 10; parseI++ {
		parseId := parseI
		parseWg.Go(func() {
			parseFiber := &Fiber{typeOf: "test"}
			parseRegistry.Subscribe("counter", parseFiber)
			parseRegistry.SetAtom("counter", parseId)
			_, _ = parseRegistry.GetAtom("counter")
			parseRegistry.Unsubscribe("counter", parseFiber)
		})
	}

	parseWg.Wait()

	// Should not crash
}

func TestGoUseAtom_InitialValue(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})

	parseFiber := newTestFiber("test")
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	get, _ := GoUseAtom(parseRt, "counter", 100)

	if get() != 100 {
		parseT.Errorf("Expected initial value 100, got %d", get())
	}
}

func TestGoUseAtom_UpdateValue(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})

	parseFiber := newTestFiber("test")
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseAtom(parseRt, "counter", 0)

	set(42)

	if get() != 42 {
		parseT.Errorf("Expected updated value 42, got %d", get())
	}
}

func TestGoUseAtom_SharedAcrossComponents(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})

	// First component
	parseFiber1 := newTestFiber("comp1")
	SetCurrentFiber(parseFiber1)
	parseGet1, parseSet1 := GoUseAtom(parseRt, "shared", "initial")
	SetCurrentFiber(nil)

	// Second component
	parseFiber2 := newTestFiber("comp2")
	SetCurrentFiber(parseFiber2)
	parseGet2, _ := GoUseAtom(parseRt, "shared", "initial")
	SetCurrentFiber(nil)

	// Update from first component
	parseSet1("updated")

	// Both should see the same value
	if parseGet1() != "updated" {
		parseT.Errorf("Component 1 expected 'updated', got %s", parseGet1())
	}

	if parseGet2() != "updated" {
		parseT.Errorf("Component 2 expected 'updated', got %s", parseGet2())
	}
}

func TestGoUseAtom_SchedulesUpdatesForSubscribers(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = &Fiber{}

	parseFiber1 := newTestFiber("comp1")
	SetCurrentFiber(parseFiber1)
	_, set := GoUseAtom(parseRt, "counter", 0)
	SetCurrentFiber(nil)

	parseFiber2 := newTestFiber("comp2")
	SetCurrentFiber(parseFiber2)
	GoUseAtom(parseRt, "counter", 0)
	SetCurrentFiber(nil)

	set(42)

	// Both fibers should have pending updates
	if !parseFiber1.needsUpdate {
		parseT.Error("Expected fiber1 to need update")
	}

	if !parseFiber2.needsUpdate {
		parseT.Error("Expected fiber2 to need update")
	}
}

func TestGoUseAtom_NoUpdateOnSameValue(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = &Fiber{}

	parseFiber := newTestFiber("test")
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseAtom(parseRt, "counter", 42)

	set(42) // Same value

	if parseFiber.needsUpdate {
		parseT.Error("Expected no update scheduled for same value")
	}

	if get() != 42 {
		parseT.Errorf("Expected value to remain 42, got %d", get())
	}
}

func TestCleanupAtomSubscriptions(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})

	parseFiber := newTestFiber("test")
	SetCurrentFiber(parseFiber)
	GoUseAtom(parseRt, "counter", 0)
	SetCurrentFiber(nil)

	// Verify subscription exists
	parseCount := parseRt.atomRegistry.GetSubscriberCount("counter")
	if parseCount != 1 {
		parseT.Errorf("Expected 1 subscriber before cleanup, got %d", parseCount)
	}

	// Cleanup
	parseRt.CleanupAtomSubscriptions(parseFiber)

	// Verify subscription removed
	parseCount = parseRt.atomRegistry.GetSubscriberCount("counter")
	if parseCount != 0 {
		parseT.Errorf("Expected 0 subscribers after cleanup, got %d", parseCount)
	}
}

func TestRuntimeHelpers_GetAtomValue(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})

	parseRt.atomRegistry.SetAtom("test", "value")

	parseValue, parseOk := parseRt.GetAtomValue("test")
	if !parseOk {
		parseT.Fatal("Expected atom to exist")
	}

	if parseValue != "value" {
		parseT.Errorf("Expected 'value', got %v", parseValue)
	}
}

func TestRuntimeHelpers_SetAtomValue(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = &Fiber{}

	parseFiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	parseRt.atomRegistry.Subscribe("test", parseFiber)

	parseErr := parseRt.SetAtomValue("test", 123)
	if parseErr != nil {
		parseT.Errorf("Unexpected error: %v", parseErr)
	}

	parseValue, _ := parseRt.GetAtomValue("test")
	if parseValue != 123 {
		parseT.Errorf("Expected 123, got %v", parseValue)
	}

	if !parseFiber.needsUpdate {
		parseT.Error("Expected fiber to need update")
	}
}

func TestAtomRegistry_GetAtomCount(parseT *testing.T) {
	parseRegistry := NewAtomRegistry()

	parseRegistry.SetAtom("atom1", 1)
	parseRegistry.SetAtom("atom2", 2)
	parseRegistry.SetAtom("atom3", 3)

	parseCount := parseRegistry.GetAtomCount()
	if parseCount != 3 {
		parseT.Errorf("Expected 3 atoms, got %d", parseCount)
	}
}

func TestAtomRegistry_SnapshotReturnsCopy(parseT *testing.T) {
	parseRegistry := NewAtomRegistry()
	parseRegistry.SetAtom("theme", "dark")

	parseSnapshot := parseRegistry.Snapshot()
	parseSnapshot["theme"] = "light"

	parseValue, _ := parseRegistry.GetAtom("theme")
	if parseValue != "dark" {
		parseT.Fatalf("expected registry atom to remain dark, got %#v", parseValue)
	}
}

func TestRuntimeRestoreAtomSnapshotSchedulesSubscribers(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = &Fiber{}

	parseFiber := newTestFiber("subscriber")
	parseRt.atomRegistry.Subscribe("theme", parseFiber)

	if parseErr := parseRt.RestoreAtomSnapshot(map[string]interface{}{"theme": "dark"}); parseErr != nil {
		parseT.Fatalf("unexpected restore error: %v", parseErr)
	}

	parseValue, _ := parseRt.GetAtomValue("theme")
	if parseValue != "dark" {
		parseT.Fatalf("expected restored atom value dark, got %#v", parseValue)
	}
	if !parseFiber.needsUpdate {
		parseT.Fatal("expected subscribed fiber to be marked for update after restore")
	}
}

func TestRuntimeHelpers_SetAtomValueTransitionDefersUntilTimeout(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = &Fiber{}

	parseFiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	parseRt.atomRegistry.Subscribe("test", parseFiber)

	parseRt.StartTransition(func() {
		if parseErr := parseRt.SetAtomValue("test", 123); parseErr != nil {
			parseT.Fatalf("unexpected transition set atom error: %v", parseErr)
		}
	})

	if parseValue, _ := parseRt.GetAtomValue("test"); parseValue != nil {
		parseT.Fatalf("expected transition atom value to stay deferred before timeout, got %#v", parseValue)
	}
	if parseFiber.needsUpdate {
		parseT.Fatal("expected subscribed fiber to stay clean before transition timeout")
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected one deferred timeout for direct atom write, got %d", len(parseScheduler.timeouts))
	}

	parseScheduler.timeouts[0]()

	if parseValue2, _ := parseRt.GetAtomValue("test"); parseValue2 != 123 {
		parseT.Fatalf("expected deferred atom value 123 after timeout, got %#v", parseValue2)
	}
	if !parseFiber.needsUpdate {
		parseT.Fatal("expected subscribed fiber to be marked after deferred atom write")
	}
}

func TestRuntimeRestoreAtomSnapshotTransitionDefersUntilTimeout(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = &Fiber{}

	parseFiber := newTestFiber("theme-subscriber")
	parseRt.atomRegistry.Subscribe("theme", parseFiber)

	parseRt.StartTransition(func() {
		if parseErr := parseRt.RestoreAtomSnapshot(map[string]interface{}{"theme": "dark"}); parseErr != nil {
			parseT.Fatalf("unexpected transition restore error: %v", parseErr)
		}
	})

	if parseValue, _ := parseRt.GetAtomValue("theme"); parseValue != nil {
		parseT.Fatalf("expected restored atom value to stay deferred before timeout, got %#v", parseValue)
	}
	if parseFiber.needsUpdate {
		parseT.Fatal("expected subscribed fiber to stay clean before deferred restore flushes")
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected one deferred timeout for snapshot restore, got %d", len(parseScheduler.timeouts))
	}

	parseScheduler.timeouts[0]()

	if parseValue2, _ := parseRt.GetAtomValue("theme"); parseValue2 != "dark" {
		parseT.Fatalf("expected restored atom value dark after timeout, got %#v", parseValue2)
	}
	if !parseFiber.needsUpdate {
		parseT.Fatal("expected subscribed fiber to be marked after deferred restore")
	}
}

func TestRegisterDerivedAtomRecomputesWhenDependencyChanges(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = &Fiber{}

	if parseErr := parseRt.SetAtomValue("count", 2); parseErr != nil {
		parseT.Fatalf("unexpected set atom error: %v", parseErr)
	}
	if parseErr2 := parseRt.RegisterDerivedAtom("double", []string{"count"}, func() interface{} {
		parseValue, _ := parseRt.GetAtomValue("count")
		return parseValue.(int) * 2
	}); parseErr2 != nil {
		parseT.Fatalf("unexpected register derived atom error: %v", parseErr2)
	}

	parseDerivedFiber := newTestFiber("derived-subscriber")
	parseRt.atomRegistry.Subscribe("double", parseDerivedFiber)

	parseValue2, _ := parseRt.GetAtomValue("double")
	if parseValue2 != 4 {
		parseT.Fatalf("expected initial derived value 4, got %#v", parseValue2)
	}

	if parseErr3 := parseRt.SetAtomValue("count", 5); parseErr3 != nil {
		parseT.Fatalf("unexpected update atom error: %v", parseErr3)
	}
	parseValue2, _ = parseRt.GetAtomValue("double")
	if parseValue2 != 10 {
		parseT.Fatalf("expected derived value 10 after source update, got %#v", parseValue2)
	}
	if !parseDerivedFiber.needsUpdate {
		parseT.Fatal("expected derived subscribers to be scheduled after source update")
	}
}

func TestRegisterDerivedAtomSupportsChainedDependencies(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})

	_ = parseRt.SetAtomValue("base", 3)
	if parseErr := parseRt.RegisterDerivedAtom("double", []string{"base"}, func() interface{} {
		parseValue, _ := parseRt.GetAtomValue("base")
		return parseValue.(int) * 2
	}); parseErr != nil {
		parseT.Fatalf("unexpected register double error: %v", parseErr)
	}
	if parseErr2 := parseRt.RegisterDerivedAtom("label", []string{"double"}, func() interface{} {
		parseValue2, _ := parseRt.GetAtomValue("double")
		return fmt.Sprintf("value:%d", parseValue2.(int))
	}); parseErr2 != nil {
		parseT.Fatalf("unexpected register label error: %v", parseErr2)
	}

	_ = parseRt.SetAtomValue("base", 4)
	parseValue3, _ := parseRt.GetAtomValue("label")
	if parseValue3 != "value:8" {
		parseT.Fatalf("expected chained derived value value:8, got %#v", parseValue3)
	}
}

func TestRegisterDerivedAtomSkipsSubscriberNotifyWhenValueUnchanged(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = &Fiber{}

	_ = parseRt.SetAtomValue("count", 1)
	if parseErr := parseRt.RegisterDerivedAtom("parity", []string{"count"}, func() interface{} {
		parseValue, _ := parseRt.GetAtomValue("count")
		return parseValue.(int) % 2
	}); parseErr != nil {
		parseT.Fatalf("unexpected register parity error: %v", parseErr)
	}

	parseDerivedFiber := newTestFiber("parity-subscriber")
	parseRt.atomRegistry.Subscribe("parity", parseDerivedFiber)

	if parseErr2 := parseRt.SetAtomValue("count", 3); parseErr2 != nil {
		parseT.Fatalf("unexpected count update error: %v", parseErr2)
	}
	if parseDerivedFiber.needsUpdate || parseDerivedFiber.dirty {
		parseT.Fatal("expected unchanged derived value to avoid notifying subscribers")
	}

	if parseErr3 := parseRt.SetAtomValue("count", 4); parseErr3 != nil {
		parseT.Fatalf("unexpected second count update error: %v", parseErr3)
	}
	if !parseDerivedFiber.needsUpdate {
		parseT.Fatal("expected changed derived value to notify subscribers")
	}
}

func TestRegisterDerivedAtomRejectsSimpleCycles(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})

	parseErr := parseRt.RegisterDerivedAtom("loop", []string{"loop"}, func() interface{} { return 1 })
	if parseErr == nil {
		parseT.Fatal("expected self-referential derived atom registration to fail")
	}
}

func TestRegisterDerivedAtomRejectsIndirectCycles(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})

	if parseErr := parseRt.RegisterDerivedAtom("derived-a", []string{"derived-b"}, func() interface{} { return 1 }); parseErr != nil {
		parseT.Fatalf("unexpected register derived-a error: %v", parseErr)
	}
	parseErr2 := parseRt.RegisterDerivedAtom("derived-b", []string{"derived-a"}, func() interface{} { return 2 })
	if parseErr2 == nil {
		parseT.Fatal("expected indirect derived cycle registration to fail")
	}
}
