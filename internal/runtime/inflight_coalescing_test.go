package runtime

import "testing"

// TestDirtyOwnerUpdatesRetainInflightReplay prevents dirty-fiber fast paths from dropping late writes.
func TestDirtyOwnerUpdatesRetainInflightReplay(parseT *testing.T) {
	for _, parseKind := range []string{"state", "granular", "fiber"} {
		for _, isParseInFlight := range []bool{false, true} {
			parseName := parseKind + "/scheduled"
			if isParseInFlight {
				parseName = parseKind + "/in-flight"
			}
			parseT.Run(parseName, func(parseT *testing.T) {
				parseRt := NewRuntime(Config{})
				parseLane := laneForUpdateOrigin("local-state")
				parseRoot := &Fiber{typeOf: "ROOT"}
				parseOwner := &Fiber{parent: parseRoot, dirty: true, needsUpdate: true, updateLane: parseLane}
				parseRoot.child = parseOwner
				parseHooks := &Hooks{owner: parseOwner, states: []any{0, 0}}
				parseOwner.hooks = parseHooks
				parseRt.currentRoot = parseRoot
				parseRt.wipRoot = &Fiber{typeOf: "ROOT", alternate: parseRoot}
				parseRt.nextUnitOfWork = parseRt.wipRoot
				if isParseInFlight {
					parseRt.nextUnitOfWork = &Fiber{parent: parseRt.wipRoot}
				}
				parseRt.updateScheduled = true
				parseRt.schedulerState.currentLane = parseLane
				switch parseKind {
				case "state":
					parseSlot := StateSlot[int]{runtime: parseRt, hooks: parseHooks, fiber: parseOwner, stateIndex: 0, pendingIndex: 1}
					parseSlot.Set(40)
					if parseSlot.Get() != 40 {
						parseT.Fatal("state write was not published")
					}
				case "granular":
					parseRt.ScheduleGranularUpdateForFiberWithOrigin(parseOwner, "local-state")
				case "fiber":
					parseRt.ScheduleUpdateForFiberWithOrigin(parseOwner, "local-state")
				}
				if parseRt.schedulerState.updateArrivedInFlight != isParseInFlight {
					parseT.Fatalf("replay owed=%t, want %t", parseRt.schedulerState.updateArrivedInFlight, isParseInFlight)
				}
			})
		}
	}
}

// TestUpdateAfterCommitTreePublicationSchedulesFollowUp covers a write between tree publication and clearing updateScheduled.
func TestUpdateAfterCommitTreePublicationSchedulesFollowUp(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{
		scheduler:       parseScheduler,
		currentRoot:     &Fiber{typeOf: "ROOT", props: map[string]any{}},
		updateScheduled: true,
	}
	// commitRoot has published currentRoot and cleared wipRoot, but has not yet
	// cleared updateScheduled. The next state write must receive its own pass.
	schedulerMu.Lock()
	parseRt.coalesceScheduledUpdateLocked(UpdateLaneDefault)
	schedulerMu.Unlock()
	parseRt.updateScheduled = false
	parseRt.scheduleFollowUpForInFlightUpdate()
	if !parseRt.updateScheduled || parseRt.wipRoot == nil || parseScheduler.getPendingTimeoutCount() != 1 {
		parseT.Fatalf("late update lost: scheduled=%t wip=%t callbacks=%d", parseRt.updateScheduled, parseRt.wipRoot != nil, parseScheduler.getPendingTimeoutCount())
	}
}

// TestUpdateDuringCommitSchedulesFollowUp covers a write after traversal ends but before the committed root is published.
func TestUpdateDuringCommitSchedulesFollowUp(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{
		scheduler:       parseScheduler,
		currentRoot:     &Fiber{typeOf: "ROOT", props: map[string]any{}},
		wipRoot:         &Fiber{typeOf: "ROOT"},
		updateScheduled: true,
	}
	// The work loop clears nextUnitOfWork before commitRoot mutates the DOM.
	// A write during that mutation phase cannot be consumed by this pass.
	schedulerMu.Lock()
	parseRt.coalesceScheduledUpdateLocked(UpdateLaneDefault)
	schedulerMu.Unlock()
	parseRt.wipRoot = nil
	parseRt.updateScheduled = false
	parseRt.scheduleFollowUpForInFlightUpdate()
	if !parseRt.updateScheduled || parseRt.wipRoot == nil || parseScheduler.getPendingTimeoutCount() != 1 {
		parseT.Fatalf("commit-phase update lost: scheduled=%t wip=%t callbacks=%d", parseRt.updateScheduled, parseRt.wipRoot != nil, parseScheduler.getPendingTimeoutCount())
	}
}

// TestOwnedUpdateDuringCommitTargetsWIP prevents a late hook update from
// marking the old current tree after the WIP traversal has finished.
func TestOwnedUpdateDuringCommitTargetsWIP(parseT *testing.T) {
	parseRt := NewRuntime(Config{})
	parseOldRoot := &Fiber{typeOf: "ROOT"}
	parseOldOwner := &Fiber{parent: parseOldRoot}
	parseOldRoot.child = parseOldOwner
	parseWipRoot := &Fiber{typeOf: "ROOT", alternate: parseOldRoot}
	parseWipOwner := &Fiber{parent: parseWipRoot, alternate: parseOldOwner}
	parseWipRoot.child = parseWipOwner
	parseOldOwner.alternate = parseWipOwner
	parseRt.currentRoot = parseOldRoot
	parseRt.wipRoot = parseWipRoot
	parseRt.updateScheduled = true
	// nextUnitOfWork == nil models the commit mutation window.
	parseRt.nextUnitOfWork = nil

	// Updated component hooks can still point at the old owner until the WIP
	// tree is published; resolve that alternate onto the in-flight tree.
	parseRt.ScheduleOwnedFiberUpdateWithOrigin(parseOldOwner, "local-state")

	if !parseWipOwner.dirty || !parseWipOwner.needsUpdate {
		parseT.Fatal("late owned update did not mark the in-flight WIP owner")
	}
	if parseOldOwner.dirty || parseOldOwner.needsUpdate {
		parseT.Fatal("late owned update marked the obsolete current owner")
	}
}
