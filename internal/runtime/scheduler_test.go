package runtime

import (
	"testing"
)

// Mock deadline for testing
type testDeadline struct {
	remaining float64
	timeout   bool
}

func (d *testDeadline) TimeRemaining() float64 {
	return d.remaining
}

func (d *testDeadline) DidTimeout() bool {
	return d.timeout
}

// Mock scheduler for testing
type testScheduler struct {
	callbacks []func(Deadline)
	timeouts  []func()
}

func newTestScheduler() *testScheduler {
	return &testScheduler{
		callbacks: make([]func(Deadline), 0),
		timeouts:  make([]func(), 0),
	}
}

func (s *testScheduler) RequestIdleCallback(callback func(deadline Deadline)) {
	s.callbacks = append(s.callbacks, callback)
}

func (s *testScheduler) SetTimeout(callback func(), delay int) {
	s.timeouts = append(s.timeouts, callback)
}

func TestScheduleUpdate_CreatesWipRoot(t *testing.T) {
	scheduler := newTestScheduler()
	rt := &Runtime{
		scheduler: scheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  make(map[string]interface{}),
		},
	}

	rt.ScheduleUpdate()

	if rt.wipRoot == nil {
		t.Fatal("Expected wipRoot to be created")
	}

	if rt.wipRoot.typeOf != "ROOT" {
		t.Errorf("Expected wipRoot typeOf to be ROOT, got %v", rt.wipRoot.typeOf)
	}

	if !rt.updateScheduled {
		t.Error("Expected updateScheduled to be true")
	}

	if len(scheduler.timeouts) != 1 {
		t.Errorf("Expected 1 timeout scheduled, got %d", len(scheduler.timeouts))
	}
}

func TestScheduleUpdate_PreventsDuplicates(t *testing.T) {
	scheduler := newTestScheduler()
	rt := &Runtime{
		scheduler: scheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  make(map[string]interface{}),
		},
	}

	rt.ScheduleUpdate()
	rt.ScheduleUpdate()
	rt.ScheduleUpdate()

	if len(scheduler.timeouts) != 1 {
		t.Errorf("Expected only 1 timeout despite multiple calls, got %d", len(scheduler.timeouts))
	}
}

func TestWorkLoop_ProcessesWork(t *testing.T) {
	scheduler := newTestScheduler()
	rt := &Runtime{
		scheduler: scheduler,
	}

	// Create a simple fiber tree
	root := &Fiber{
		typeOf: "ROOT",
		props:  map[string]interface{}{"children": []interface{}{}},
		dirty:  true,
	}

	rt.wipRoot = root
	rt.nextUnitOfWork = root

	// Execute work loop
	deadline := &testDeadline{remaining: 16.0, timeout: false}
	rt.workLoop(deadline)

	if rt.nextUnitOfWork != nil {
		t.Error("Expected all work to be completed")
	}

	if rt.currentRoot != root {
		t.Error("Expected currentRoot to be updated")
	}

	if rt.updateScheduled {
		t.Error("Expected updateScheduled to be false after completion")
	}
}

func TestWorkLoop_RespectsDeadline(t *testing.T) {
	scheduler := newTestScheduler()
	rt := &Runtime{
		scheduler: scheduler,
	}

	// Create a fiber with many children
	root := &Fiber{
		typeOf: "ROOT",
		props:  map[string]interface{}{"children": []interface{}{}},
		dirty:  true,
	}

	// Add many children to simulate heavy work
	child := root
	for i := 0; i < 400; i++ {
		nextChild := &Fiber{
			typeOf: "div",
			props:  make(map[string]interface{}),
			parent: root,
			dirty:  true,
		}
		child.child = nextChild
		child = nextChild
	}

	rt.wipRoot = root
	rt.nextUnitOfWork = root

	// Execute with tight deadline (should yield)
	deadline := &testDeadline{remaining: 0.5, timeout: false}
	rt.workLoop(deadline)

	// Should have scheduled another timeout due to yielding
	if len(scheduler.timeouts) != 1 {
		t.Errorf("Expected work loop to schedule continuation, got %d timeouts", len(scheduler.timeouts))
	}
}

func TestScheduleUpdateForFiber_MarksParentsDirty(t *testing.T) {
	scheduler := newTestScheduler()
	rt := &Runtime{
		scheduler: scheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  make(map[string]interface{}),
		},
	}

	grandparent := &Fiber{typeOf: "grandparent", dirty: false}
	parent := &Fiber{typeOf: "parent", parent: grandparent, dirty: false}
	child := &Fiber{typeOf: "child", parent: parent, dirty: false}

	rt.ScheduleUpdateForFiber(child)

	if !child.dirty {
		t.Error("Expected child to be marked dirty")
	}

	if !parent.dirty {
		t.Error("Expected parent to be marked dirty")
	}

	if !grandparent.dirty {
		t.Error("Expected grandparent to be marked dirty")
	}
	if child.updateOrigin != "hook" {
		t.Fatalf("expected child update origin hook, got %q", child.updateOrigin)
	}
	if parent.updateOrigin != "ancestor" || grandparent.updateOrigin != "ancestor" {
		t.Fatalf("expected ancestors to be marked as ancestor origin, got parent=%q grandparent=%q", parent.updateOrigin, grandparent.updateOrigin)
	}
}

func TestScheduleUpdateForFiberWithOrigin_UsesSpecificTrigger(t *testing.T) {
	scheduler := newTestScheduler()
	rt := &Runtime{
		scheduler: scheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  make(map[string]interface{}),
		},
	}

	parent := &Fiber{typeOf: "parent"}
	child := &Fiber{typeOf: "child", parent: parent}

	rt.ScheduleUpdateForFiberWithOrigin(child, "local-state")

	if child.updateOrigin != "local-state" {
		t.Fatalf("expected child update origin local-state, got %q", child.updateOrigin)
	}
	if parent.updateOrigin != "ancestor" {
		t.Fatalf("expected parent update origin ancestor, got %q", parent.updateOrigin)
	}
}

func TestScheduleSubscribedFiberUpdateWithOrigin_UsesGranularOrigin(t *testing.T) {
	scheduler := newTestScheduler()
	rt := &Runtime{
		scheduler: scheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  make(map[string]interface{}),
		},
	}

	fiber := &Fiber{typeOf: ReactiveTextNodeType, fineGrained: true}

	rt.ScheduleSubscribedFiberUpdateWithOrigin(fiber, "atom")

	if !fiber.dirty || !fiber.needsUpdate {
		t.Fatal("expected fine-grained subscribed fiber to be marked for update")
	}
	if fiber.updateOrigin != "atom" {
		t.Fatalf("expected fine-grained update origin atom, got %q", fiber.updateOrigin)
	}
	if rt.profiling.scheduledGranularMarks != 1 {
		t.Fatalf("expected one granular scheduling mark, got %d", rt.profiling.scheduledGranularMarks)
	}
}

func TestEnqueueUI(t *testing.T) {
	executed := false

	EnqueueUI(func() {
		executed = true
	})

	ProcessUIQueue()

	if !executed {
		t.Error("Expected UI function to be executed")
	}
}

func TestUIQueue_MultipleItems(t *testing.T) {
	count := 0

	for i := 0; i < 10; i++ {
		EnqueueUI(func() {
			count++
		})
	}

	ProcessUIQueue()

	if count != 10 {
		t.Errorf("Expected 10 executions, got %d", count)
	}
}

func TestGetUIQueueSize(t *testing.T) {
	// Clear queue first
	ProcessUIQueue()

	EnqueueUI(func() {})
	EnqueueUI(func() {})

	size := GetUIQueueSize()
	if size != 2 {
		t.Errorf("Expected queue size 2, got %d", size)
	}

	ProcessUIQueue()

	size = GetUIQueueSize()
	if size != 0 {
		t.Errorf("Expected queue size 0 after processing, got %d", size)
	}
}
