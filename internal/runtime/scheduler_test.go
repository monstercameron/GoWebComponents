package runtime

import (
	"testing"
)

// Mock deadline for testing
type testDeadline struct {
	remaining float64
	timeout   bool
}

func (parseD *testDeadline) TimeRemaining() float64 {
	return parseD.remaining
}

func (parseD *testDeadline) DidTimeout() bool {
	return parseD.timeout
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

func (parseS *testScheduler) RequestIdleCallback(parseCallback func(deadline Deadline)) {
	parseS.callbacks = append(parseS.callbacks, parseCallback)
}

func (parseS *testScheduler) SetTimeout(parseCallback func(), parseDelay int) {
	parseS.timeouts = append(parseS.timeouts, parseCallback)
}

func TestScheduleUpdate_CreatesWipRoot(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{
		scheduler: parseScheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  make(map[string]interface{}),
		},
	}

	parseRt.ScheduleUpdate()

	if parseRt.wipRoot == nil {
		parseT.Fatal("Expected wipRoot to be created")
	}

	if parseRt.wipRoot.typeOf != "ROOT" {
		parseT.Errorf("Expected wipRoot typeOf to be ROOT, got %v", parseRt.wipRoot.typeOf)
	}

	if !parseRt.updateScheduled {
		parseT.Error("Expected updateScheduled to be true")
	}

	if len(parseScheduler.timeouts) != 1 {
		parseT.Errorf("Expected 1 timeout scheduled, got %d", len(parseScheduler.timeouts))
	}
}

func TestScheduleUpdate_PreventsDuplicates(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{
		scheduler: parseScheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  make(map[string]interface{}),
		},
	}

	parseRt.ScheduleUpdate()
	parseRt.ScheduleUpdate()
	parseRt.ScheduleUpdate()

	if len(parseScheduler.timeouts) != 1 {
		parseT.Errorf("Expected only 1 timeout despite multiple calls, got %d", len(parseScheduler.timeouts))
	}
}

func TestWorkLoop_ProcessesWork(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{
		scheduler: parseScheduler,
	}

	// Create a simple fiber tree
	parseRoot := &Fiber{
		typeOf: "ROOT",
		props:  map[string]interface{}{"children": []interface{}{}},
		dirty:  true,
	}

	parseRt.wipRoot = parseRoot
	parseRt.nextUnitOfWork = parseRoot

	// Execute work loop
	parseDeadline := &testDeadline{remaining: 16.0, timeout: false}
	parseRt.workLoop(parseDeadline)

	if parseRt.nextUnitOfWork != nil {
		parseT.Error("Expected all work to be completed")
	}

	if parseRt.currentRoot != parseRoot {
		parseT.Error("Expected currentRoot to be updated")
	}

	if parseRt.updateScheduled {
		parseT.Error("Expected updateScheduled to be false after completion")
	}
}

func TestWorkLoop_RespectsDeadline(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{
		scheduler: parseScheduler,
	}

	// Create a fiber with many children
	parseRoot := &Fiber{
		typeOf: "ROOT",
		props:  map[string]interface{}{"children": []interface{}{}},
		dirty:  true,
	}

	// Add many children to simulate heavy work
	parseChild := parseRoot
	for parseI := 0; parseI < 400; parseI++ {
		parseNextChild := &Fiber{
			typeOf: "div",
			props:  make(map[string]interface{}),
			parent: parseRoot,
			dirty:  true,
		}
		parseChild.child = parseNextChild
		parseChild = parseNextChild
	}

	parseRt.wipRoot = parseRoot
	parseRt.nextUnitOfWork = parseRoot

	// Execute with tight deadline (should yield)
	parseDeadline := &testDeadline{remaining: 0.5, timeout: false}
	parseRt.workLoop(parseDeadline)

	// Should have scheduled another timeout due to yielding
	if len(parseScheduler.timeouts) != 1 {
		parseT.Errorf("Expected work loop to schedule continuation, got %d timeouts", len(parseScheduler.timeouts))
	}
}

func TestScheduleUpdateForFiber_MarksParentsDirty(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{
		scheduler: parseScheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  make(map[string]interface{}),
		},
	}

	parseGrandparent := &Fiber{typeOf: "grandparent", dirty: false}
	parseParent := &Fiber{typeOf: "parent", parent: parseGrandparent, dirty: false}
	parseChild := &Fiber{typeOf: "child", parent: parseParent, dirty: false}

	parseRt.ScheduleUpdateForFiber(parseChild)

	if !parseChild.dirty {
		parseT.Error("Expected child to be marked dirty")
	}

	if !parseParent.dirty {
		parseT.Error("Expected parent to be marked dirty")
	}

	if !parseGrandparent.dirty {
		parseT.Error("Expected grandparent to be marked dirty")
	}
	if parseChild.updateOrigin != "hook" {
		parseT.Fatalf("expected child update origin hook, got %q", parseChild.updateOrigin)
	}
	if parseParent.updateOrigin != "ancestor" || parseGrandparent.updateOrigin != "ancestor" {
		parseT.Fatalf("expected ancestors to be marked as ancestor origin, got parent=%q grandparent=%q", parseParent.updateOrigin, parseGrandparent.updateOrigin)
	}
}

func TestScheduleUpdateForFiberWithOrigin_UsesSpecificTrigger(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{
		scheduler: parseScheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  make(map[string]interface{}),
		},
	}

	parseParent := &Fiber{typeOf: "parent"}
	parseChild := &Fiber{typeOf: "child", parent: parseParent}

	parseRt.ScheduleUpdateForFiberWithOrigin(parseChild, "local-state")

	if parseChild.updateOrigin != "local-state" {
		parseT.Fatalf("expected child update origin local-state, got %q", parseChild.updateOrigin)
	}
	if parseParent.updateOrigin != "ancestor" {
		parseT.Fatalf("expected parent update origin ancestor, got %q", parseParent.updateOrigin)
	}
}

func TestScheduleSubscribedFiberUpdateWithOrigin_UsesGranularOrigin(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{
		scheduler: parseScheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  make(map[string]interface{}),
		},
	}

	parseFiber := &Fiber{typeOf: ReactiveTextNodeType, fineGrained: true}

	parseRt.ScheduleSubscribedFiberUpdateWithOrigin(parseFiber, "atom")

	if !parseFiber.dirty || !parseFiber.needsUpdate {
		parseT.Fatal("expected fine-grained subscribed fiber to be marked for update")
	}
	if parseFiber.updateOrigin != "atom" {
		parseT.Fatalf("expected fine-grained update origin atom, got %q", parseFiber.updateOrigin)
	}
	if parseRt.profiling.scheduledGranularMarks != 1 {
		parseT.Fatalf("expected one granular scheduling mark, got %d", parseRt.profiling.scheduledGranularMarks)
	}
}

func TestEnqueueUI(parseT *testing.T) {
	isParseExecuted := false

	EnqueueUI(func() {
		isParseExecuted = true
	})

	ProcessUIQueue()

	if !isParseExecuted {
		parseT.Error("Expected UI function to be executed")
	}
}

func TestUIQueue_MultipleItems(parseT *testing.T) {
	parseCount := 0

	for parseI := 0; parseI < 10; parseI++ {
		EnqueueUI(func() {
			parseCount++
		})
	}

	ProcessUIQueue()

	if parseCount != 10 {
		parseT.Errorf("Expected 10 executions, got %d", parseCount)
	}
}

func TestGetUIQueueSize(parseT *testing.T) {
	// Clear queue first
	ProcessUIQueue()

	EnqueueUI(func() {})
	EnqueueUI(func() {})

	parseSize := GetUIQueueSize()
	if parseSize != 2 {
		parseT.Errorf("Expected queue size 2, got %d", parseSize)
	}

	ProcessUIQueue()

	parseSize = GetUIQueueSize()
	if parseSize != 0 {
		parseT.Errorf("Expected queue size 0 after processing, got %d", parseSize)
	}
}
