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
			props:  make(map[string]any),
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
			props:  make(map[string]any),
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
		props:  map[string]any{"children": []any{}},
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
		props:  map[string]any{"children": []any{}},
		dirty:  true,
	}

	// Add many children to simulate heavy work
	parseChild := parseRoot
	for range 400 {
		parseNextChild := &Fiber{
			typeOf: "div",
			props:  make(map[string]any),
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
			props:  make(map[string]any),
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
			props:  make(map[string]any),
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

func TestScheduleOwnedFiberUpdateWithOrigin_UsesLiveAlternateGranularPath(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRoot := &Fiber{
		typeOf: "ROOT",
		props:  make(map[string]any),
	}
	parseLiveOwner := &Fiber{typeOf: "owner", parent: parseRoot}
	parseRoot.child = parseLiveOwner

	parseStaleOwner := &Fiber{typeOf: "owner", alternate: parseLiveOwner}
	parseLiveOwner.alternate = parseStaleOwner

	parseRt := &Runtime{
		scheduler:   parseScheduler,
		currentRoot: parseRoot,
	}

	parseRt.ScheduleOwnedFiberUpdateWithOrigin(parseStaleOwner, "local-state")

	if !parseLiveOwner.dirty || !parseLiveOwner.needsUpdate {
		parseT.Fatal("expected live owner alternate to carry the scheduled update")
	}
	if parseLiveOwner.updateOrigin != "local-state" {
		parseT.Fatalf("expected live owner update origin local-state, got %q", parseLiveOwner.updateOrigin)
	}
	if parseRoot.dirty || parseRoot.needsUpdate {
		parseT.Fatal("expected root to remain clean for owned granular scheduling")
	}
	if parseStaleOwner.dirty || parseStaleOwner.needsUpdate {
		parseT.Fatal("expected stale owner to remain unscheduled")
	}
	if parseRt.profiling.scheduledGranularMarks != 1 {
		parseT.Fatalf("expected one granular scheduling mark, got %d", parseRt.profiling.scheduledGranularMarks)
	}
}

func TestScheduleOwnedFiberUpdateWithOrigin_UsesDetachedOwnerFallbackBeforeMount(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRoot := &Fiber{
		typeOf: "ROOT",
		props:  make(map[string]any),
	}
	parseOwner := &Fiber{typeOf: "owner"}
	parseRt := &Runtime{
		scheduler:   parseScheduler,
		currentRoot: parseRoot,
	}

	parseRt.ScheduleOwnedFiberUpdateWithOrigin(parseOwner, "local-state")

	if !parseOwner.dirty || !parseOwner.needsUpdate {
		parseT.Fatal("expected detached owner fallback to mark owner for update before mount")
	}
	if parseOwner.updateOrigin != "local-state" {
		parseT.Fatalf("expected detached owner fallback origin local-state, got %q", parseOwner.updateOrigin)
	}
	if parseRt.profiling.scheduledGranularMarks != 0 {
		parseT.Fatalf("expected detached owner fallback to avoid granular scheduling, got %d", parseRt.profiling.scheduledGranularMarks)
	}
}

func TestScheduleOwnedFiberUpdateWithOrigin_IgnoresDetachedOwnerWhenTreeIsMounted(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRoot := &Fiber{
		typeOf: "ROOT",
		props:  make(map[string]any),
	}
	parseRoot.child = &Fiber{typeOf: "app", parent: parseRoot}
	parseOwner := &Fiber{typeOf: "owner"}
	parseRt := &Runtime{
		scheduler:   parseScheduler,
		currentRoot: parseRoot,
	}

	parseRt.ScheduleOwnedFiberUpdateWithOrigin(parseOwner, "local-state")

	if parseOwner.dirty || parseOwner.needsUpdate {
		parseT.Fatal("expected detached owner to stay ignored once a live tree is mounted")
	}
	if parseRt.profiling.scheduledFiberMarks != 0 {
		parseT.Fatalf("expected no scheduled marks for ignored detached owner, got %d", parseRt.profiling.scheduledFiberMarks)
	}
}

func TestScheduleSubscribedFiberUpdateWithOrigin_UsesGranularOrigin(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{
		scheduler: parseScheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  make(map[string]any),
		},
	}

	parseFiber := &Fiber{typeOf: ReactiveTextNodeType, fineGrained: true, parent: parseRt.currentRoot}
	parseRt.currentRoot.child = parseFiber

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

func TestScheduleSubscribedFiberUpdateWithOrigin_UsesFineGrainedAncestor(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRoot := &Fiber{
		typeOf: "ROOT",
		props:  make(map[string]any),
	}
	parseApp := &Fiber{typeOf: "app", parent: parseRoot}
	parseRegion := &Fiber{typeOf: ReactiveRegionNodeType, parent: parseApp, fineGrained: true}
	parseSubscriber := &Fiber{typeOf: "child", parent: parseRegion}
	parseRoot.child = parseApp
	parseApp.child = parseRegion
	parseRegion.child = parseSubscriber
	parseRt := &Runtime{
		scheduler:   parseScheduler,
		currentRoot: parseRoot,
	}

	parseRt.ScheduleSubscribedFiberUpdateWithOrigin(parseSubscriber, "atom")

	if !parseSubscriber.dirty || !parseSubscriber.needsUpdate {
		parseT.Fatal("expected subscriber fiber to be scheduled granularly inside fine-grained boundary")
	}
	if parseSubscriber.updateOrigin != "atom" {
		parseT.Fatalf("expected subscriber origin atom, got %q", parseSubscriber.updateOrigin)
	}
	if parseRegion.dirty || parseRegion.needsUpdate {
		parseT.Fatal("expected fine-grained ancestor to stay clean while subscriber carries granular update")
	}
	if parseApp.dirty || parseRoot.dirty {
		parseT.Fatal("expected app and root ancestors to remain clean for fine-grained scheduling")
	}
	if parseRt.profiling.scheduledGranularMarks != 1 {
		parseT.Fatalf("expected one granular scheduling mark, got %d", parseRt.profiling.scheduledGranularMarks)
	}
}

func TestScheduleSubscribedFiberUpdateWithOrigin_UsesLiveFineGrainedAlternate(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRoot := &Fiber{
		typeOf: "ROOT",
		props:  make(map[string]any),
	}
	parseApp := &Fiber{typeOf: "app", parent: parseRoot}
	parseLiveRegion := &Fiber{typeOf: ReactiveRegionNodeType, parent: parseApp, fineGrained: true}
	parseLiveSubscriber := &Fiber{typeOf: "child", parent: parseLiveRegion}
	parseRoot.child = parseApp
	parseApp.child = parseLiveRegion
	parseLiveRegion.child = parseLiveSubscriber

	parseStaleRegion := &Fiber{typeOf: ReactiveRegionNodeType, fineGrained: true, alternate: parseLiveRegion}
	parseStaleSubscriber := &Fiber{typeOf: "child", parent: parseStaleRegion, alternate: parseLiveSubscriber}
	parseLiveRegion.alternate = parseStaleRegion
	parseLiveSubscriber.alternate = parseStaleSubscriber

	parseRt := &Runtime{
		scheduler:   parseScheduler,
		currentRoot: parseRoot,
	}

	parseRt.ScheduleSubscribedFiberUpdateWithOrigin(parseStaleSubscriber, "atom")

	if !parseLiveSubscriber.dirty || !parseLiveSubscriber.needsUpdate {
		parseT.Fatal("expected live subscriber alternate to be scheduled")
	}
	if parseLiveSubscriber.updateOrigin != "atom" {
		parseT.Fatalf("expected live subscriber origin atom, got %q", parseLiveSubscriber.updateOrigin)
	}
	if parseLiveRegion.dirty || parseLiveRegion.needsUpdate {
		parseT.Fatal("expected live fine-grained region to remain clean while subscriber carries granular update")
	}
	if parseStaleRegion.dirty || parseStaleRegion.needsUpdate {
		parseT.Fatal("expected stale fine-grained region to remain unscheduled")
	}
	if parseRt.profiling.scheduledGranularMarks != 1 {
		parseT.Fatalf("expected one granular scheduling mark, got %d", parseRt.profiling.scheduledGranularMarks)
	}
}

func TestScheduleSubscribedFiberUpdateWithOrigin_UsesDetachedSubscriberFallback(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{
		scheduler: parseScheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  make(map[string]any),
		},
	}

	parseSubscriber := &Fiber{typeOf: "subscriber"}
	parseRt.ScheduleSubscribedFiberUpdateWithOrigin(parseSubscriber, "atom")

	if !parseSubscriber.dirty || !parseSubscriber.needsUpdate {
		parseT.Fatal("expected detached subscriber fallback to mark subscriber for update")
	}
	if parseSubscriber.updateOrigin != "atom" {
		parseT.Fatalf("expected detached subscriber origin atom, got %q", parseSubscriber.updateOrigin)
	}
	if parseRt.profiling.scheduledGranularMarks != 0 {
		parseT.Fatalf("expected no granular marks for detached subscriber fallback, got %d", parseRt.profiling.scheduledGranularMarks)
	}
}

func TestScheduleSubscribedFiberUpdateWithOrigin_IgnoresDetachedSubscriberWhenTreeIsMounted(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRoot := &Fiber{
		typeOf: "ROOT",
		props:  make(map[string]any),
	}
	parseRoot.child = &Fiber{typeOf: "app", parent: parseRoot}
	parseRt := &Runtime{
		scheduler:   parseScheduler,
		currentRoot: parseRoot,
	}

	parseSubscriber := &Fiber{typeOf: "subscriber"}
	parseRt.ScheduleSubscribedFiberUpdateWithOrigin(parseSubscriber, "atom")

	if parseSubscriber.dirty || parseSubscriber.needsUpdate {
		parseT.Fatal("expected detached subscriber to stay ignored once a live tree is mounted")
	}
	if parseRt.profiling.scheduledFiberMarks != 0 {
		parseT.Fatalf("expected no scheduled marks for ignored detached subscriber, got %d", parseRt.profiling.scheduledFiberMarks)
	}
}

// TestScheduleUpdateForFiberWithOrigin_PreservesTransitionOriginAcrossMixedMarks verifies later non-transition marks do not erase a queued transition rerender.
func TestScheduleUpdateForFiberWithOrigin_PreservesTransitionOriginAcrossMixedMarks(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRoot := &Fiber{
		typeOf: "ROOT",
		props:  make(map[string]any),
	}
	parseOwner := &Fiber{typeOf: "owner", parent: parseRoot}
	parseRoot.child = parseOwner
	parseRt := &Runtime{
		scheduler:   parseScheduler,
		currentRoot: parseRoot,
	}

	parseRt.ScheduleUpdateForFiberWithOrigin(parseOwner, "atom")
	if parseOwner.updateOrigin != "atom" {
		parseT.Fatalf("expected first origin atom, got %q", parseOwner.updateOrigin)
	}

	parseRt.ScheduleUpdateForFiberWithOrigin(parseOwner, "transition")
	if parseOwner.updateOrigin != "transition" {
		parseT.Fatalf("expected transition mark to override pending atom mark, got %q", parseOwner.updateOrigin)
	}

	parseRt.ScheduleUpdateForFiberWithOrigin(parseOwner, "atom")
	if parseOwner.updateOrigin != "transition" {
		parseT.Fatalf("expected later atom mark to preserve queued transition origin, got %q", parseOwner.updateOrigin)
	}

	parseOwner.dirty = false
	parseOwner.needsUpdate = false
	parseRt.ScheduleUpdateForFiberWithOrigin(parseOwner, "atom")
	if parseOwner.updateOrigin != "atom" {
		parseT.Fatalf("expected fresh scheduling cycle to replace stale transition origin, got %q", parseOwner.updateOrigin)
	}
}

// Ported from TestUIQueue_MultipleItems (v5 P2.4). The package-global UI queue
// it exercised is gone; the per-runtime async inbox replaced it, so the same
// property — many queued items all run — is asserted against the new mechanism.
func TestAsyncInbox_MultipleItems(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler(), Reset: true})
	parseCount := 0

	for range 10 {
		parseRt.PostAsync(func() {
			parseCount++
		})
	}

	parseRt.DrainAsyncInbox()

	if parseCount != 10 {
		parseT.Errorf("Expected 10 executions, got %d", parseCount)
	}
}

