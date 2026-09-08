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
