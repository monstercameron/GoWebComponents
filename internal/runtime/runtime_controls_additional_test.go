package runtime

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestUpdateLaneLabelsAndSliceBudgets(t *testing.T) {
	parseCases := []struct {
		lane UpdateLane
		name string
		max  int
	}{
		{UpdateLaneSync, "sync", 8},
		{UpdateLaneInput, "input", 4},
		{UpdateLaneDefault, "default", 4},
		{UpdateLaneTransition, "transition", 2},
		{UpdateLaneBackground, "background", 1},
		{UpdateLane(99), "default", 4},
	}
	for _, parseCase := range parseCases {
		if parseGot := parseCase.lane.String(); parseGot != parseCase.name {
			t.Fatalf("%v.String() = %q, want %q", uint8(parseCase.lane), parseGot, parseCase.name)
		}
		if parseGot := parseCase.lane.maxUnitsPerSlice(4); parseGot != parseCase.max {
			t.Fatalf("%s.maxUnitsPerSlice(4) = %d, want %d", parseCase.name, parseGot, parseCase.max)
		}
	}
	if parseGot := UpdateLaneTransition.maxUnitsPerSlice(1); parseGot != 1 {
		t.Fatalf("transition minimum slice = %d, want 1", parseGot)
	}
	if parseGot := UpdateLaneBackground.maxUnitsPerSlice(0); parseGot != 1 {
		t.Fatalf("background zero-base slice = %d, want 1", parseGot)
	}
}

func TestRuntimeSchedulerNilAndNormalizationBranches(t *testing.T) {
	var parseNilState *runtimeSchedulerState
	parseNilState.ensureDefaults(RuntimeLimits{})
	parseNilState.beginScheduledLocked(UpdateLaneSync)
	parseNilState.finishScheduledLocked()
	var parseNilRuntime *Runtime
	parseNilRuntime.coalesceScheduledUpdateLocked(UpdateLaneSync)

	parseState := &runtimeSchedulerState{}
	parseState.ensureDefaults(RuntimeLimits{MaxQueuedUpdates: 7})
	if parseState.maxQueuedUpdates != 7 {
		t.Fatalf("maxQueuedUpdates = %d, want explicit limit", parseState.maxQueuedUpdates)
	}
	parseState.beginScheduledLocked(UpdateLane(99))
	if parseState.pendingLane != UpdateLaneDefault || parseState.currentLane != UpdateLaneDefault || parseState.enqueuedUpdates != 1 {
		t.Fatalf("beginScheduledLocked normalized incorrectly: %#v", parseState)
	}
	parseState.finishScheduledLocked()
	if parseState.pendingLane != 0 || parseState.currentLane != 0 || parseState.coalescedUpdates != 0 {
		t.Fatalf("finishScheduledLocked did not clear transient scheduler state: %#v", parseState)
	}
	if parseGot := normalizeUpdateLane(UpdateLane(42)); parseGot != UpdateLaneDefault {
		t.Fatalf("normalizeUpdateLane(42) = %v, want default", parseGot)
	}
}

func TestRenderFunctionComponentElementVariants(t *testing.T) {
	parseElement := &Element{Type: "span"}
	parseAttrsSeen := false
	parseCases := []struct {
		name  string
		fiber *Fiber
	}{
		{"nil", nil},
		{"func-no-props", &Fiber{typeOf: func() *Element { return parseElement }}},
		{"func-map-props", &Fiber{typeOf: func(parseProps map[string]any) *Element {
			if parseProps["answer"] != 42 {
				t.Fatalf("map props = %#v, want answer", parseProps)
			}
			return parseElement
		}, props: map[string]any{"answer": 42}}},
		{"func-attrs", &Fiber{typeOf: func(parseAttrs Attrs) *Element {
			parseAttrsSeen = parseAttrs["label"] == "ok"
			return parseElement
		}, props: map[string]any{"label": "ok"}}},
		{"component-type", &Fiber{typeOf: NewComponentType("id", "Probe", "pkg.Probe", "impl", func(parseImpl any, parseProps map[string]any) *Element {
			if parseImpl != "impl" || parseProps["x"] != true {
				t.Fatalf("component render args impl=%#v props=%#v", parseImpl, parseProps)
			}
			return parseElement
		}), props: map[string]any{"x": true}}},
		{"unknown", &Fiber{typeOf: 123}},
	}
	for _, parseCase := range parseCases {
		t.Run(parseCase.name, func(t *testing.T) {
			parseGot, parseOK := renderFunctionComponentElement(parseCase.fiber)
			parseWantOK := parseCase.name != "nil" && parseCase.name != "unknown"
			if parseOK != parseWantOK {
				t.Fatalf("ok = %v, want %v", parseOK, parseWantOK)
			}
			if parseWantOK && parseGot != parseElement {
				t.Fatalf("element = %#v, want shared element", parseGot)
			}
		})
	}
	if !parseAttrsSeen {
		t.Fatal("Attrs component did not receive typed attrs")
	}
}

func TestReplayHelpersCopyNormalizeAndResolvePaths(t *testing.T) {
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseA := &Fiber{typeOf: "a", parent: parseRoot}
	parseB := &Fiber{typeOf: "b", parent: parseRoot}
	parseC := &Fiber{typeOf: "c", parent: parseB}
	parseRoot.child = parseA
	parseA.sibling = parseB
	parseB.child = parseC
	parseRt := &Runtime{currentRoot: parseRoot, limits: RuntimeLimits{MaxReplayEvents: 2}}

	if parseGot := parseRt.ReplayEvents(); parseGot != nil {
		t.Fatalf("ReplayEvents before recording = %#v, want nil", parseGot)
	}
	parseRt.StartReplayRecording()
	parseRt.recordReplayUpdateLocked(replayUpdateKindRoot, []int{0}, "render", UpdateLaneSync)
	parseRt.recordReplayUpdateLocked(replayUpdateKindFiber, []int{1, 0}, "event", UpdateLaneInput)
	parseRt.recordReplayUpdateLocked(replayUpdateKindGranular, []int{9}, "idle", UpdateLaneBackground)
	parseEvents := parseRt.ReplayEvents()
	if len(parseEvents) != 2 {
		t.Fatalf("events len = %d, want bounded buffer of 2: %#v", len(parseEvents), parseEvents)
	}
	if !parseEvents[1].Dropped {
		t.Fatalf("last event should carry dropped marker after overflow: %#v", parseEvents)
	}
	parseEvents[0].Path[0] = 99
	parseCopy := parseRt.StopReplayRecording()
	if parseCopy[0].Path[0] == 99 {
		t.Fatalf("ReplayEvents returned aliased path slice: %#v", parseCopy)
	}

	if parseGot := parseReplayLane("sync"); parseGot != UpdateLaneSync {
		t.Fatalf("parseReplayLane(sync) = %v", parseGot)
	}
	if parseGot := parseReplayLane("input"); parseGot != UpdateLaneInput {
		t.Fatalf("parseReplayLane(input) = %v", parseGot)
	}
	if parseGot := parseReplayLane("transition"); parseGot != UpdateLaneTransition {
		t.Fatalf("parseReplayLane(transition) = %v", parseGot)
	}
	if parseGot := parseReplayLane("background"); parseGot != UpdateLaneBackground {
		t.Fatalf("parseReplayLane(background) = %v", parseGot)
	}
	if parseGot := parseReplayLane("other"); parseGot != UpdateLaneDefault {
		t.Fatalf("parseReplayLane(other) = %v", parseGot)
	}
	if parseGot := parseRt.fiberAtPath([]int{1, 0}); parseGot != parseC {
		t.Fatalf("fiberAtPath([1 0]) = %#v, want c", parseGot)
	}
	if parseGot := parseRt.fiberAtPath([]int{2}); parseGot != nil {
		t.Fatalf("fiberAtPath(out of range) = %#v, want nil", parseGot)
	}
	var parseNilRuntime *Runtime
	if parseGot := parseNilRuntime.fiberAtPath([]int{0}); parseGot != nil {
		t.Fatalf("nil runtime fiberAtPath = %#v, want nil", parseGot)
	}
}

func TestAgentWriteAdditionalErrorBranches(t *testing.T) {
	if parseErr := SetAgentState(nil, "", 0, nil); !errors.Is(parseErr, ErrAgentRefStale) {
		t.Fatalf("SetAgentState nil runtime = %v, want ErrAgentRefStale", parseErr)
	}
	parseRt, parseButton := buildAgentWriteFixtureTree(func() {})
	parseRef := AgentRefForFiber(parseButton)
	if parseErr := SetAgentState(parseRt, parseRef, -1, nil); !errors.Is(parseErr, ErrAgentStateSlotInvalid) {
		t.Fatalf("SetAgentState negative slot = %v, want invalid slot", parseErr)
	}

	parseRoot := &Fiber{typeOf: "ROOT"}
	parseComponent := &Fiber{
		typeOf: NewComponentType("example/Probe", "Probe", "example/Probe", nil, nil),
		parent: parseRoot,
		hooks:  &Hooks{signature: []string{"state"}, states: []any{1, 1}},
	}
	parseComponent.hooks.owner = parseComponent
	parseRoot.child = parseComponent
	parseRt = &Runtime{currentRoot: parseRoot, atomRegistry: NewAtomRegistry()}
	parseRef = AgentRefForFiber(parseComponent)
	if parseErr := SetAgentState(parseRt, parseRef, 0, "wrong-type"); !errors.Is(parseErr, ErrAgentStateSlotInvalid) {
		t.Fatalf("SetAgentState wrong type = %v, want invalid slot", parseErr)
	}

	if parseResult, parseErr := DeleteAgentAtom(&Runtime{}, "", false); parseErr == nil || parseResult.Deleted {
		t.Fatalf("DeleteAgentAtom empty registry = (%#v, %v), want error", parseResult, parseErr)
	}
	parseRegistry := NewAtomRegistry()
	parseRt = &Runtime{atomRegistry: parseRegistry}
	if parseResult, parseErr := DeleteAgentAtom(parseRt, "missing", false); parseErr == nil || parseResult.Deleted {
		t.Fatalf("DeleteAgentAtom missing = (%#v, %v), want error", parseResult, parseErr)
	}

	parseRefs := (&Runtime{}).agentSubscriberRefs([]*Fiber{nil, parseComponent, parseComponent})
	if len(parseRefs) != 1 || strings.TrimSpace(parseRefs[0]) == "" {
		t.Fatalf("agentSubscriberRefs de-dupe = %#v, want one non-empty ref", parseRefs)
	}

	parseCalled := false
	if parseErr := writeInvokeHandler(func() { parseCalled = true }, "onclick", "ref"); parseErr != nil || !parseCalled {
		t.Fatalf("writeInvokeHandler func() = called %v err %v", parseCalled, parseErr)
	}
	if parseErr := writeInvokeHandler(123, "onclick", "ref"); parseErr == nil {
		t.Fatal("writeInvokeHandler unknown native handler returned nil error")
	}
}

func TestRuntimeControlsPureHelpersRemainStable(t *testing.T) {
	parseFiber := &Fiber{}
	parseRt := &Runtime{}
	parseRt.reportStrictSetStateDuringRender(parseFiber, "state")
	parseRt.checkStrictEffectCleanupSymmetry(nil, 0, true)
	parseRt.strictPreviewRender(nil)

	parseOriginal := []ReplayEvent{{Kind: "root", Path: []int{1}, Lane: "sync"}}
	parseRt.ReplayUpdates(parseOriginal)
	if !reflect.DeepEqual(parseOriginal[0].Path, []int{1}) {
		t.Fatalf("ReplayUpdates mutated input events: %#v", parseOriginal)
	}
}
