package runtime

import (
	"strings"
	"testing"
)

// Regressions found reviewing the v5 renderer. Each test names the defect it
// pins rather than the function it calls, because the function is not the point.

// A handler whose prop name is not in propMetaCache must still come off the node.
//
// The removal path asked getPropMeta — a NAME-keyed table — how to unset a prop,
// and for anything outside that hand-written table it answered "remove the
// attribute" while the value had been installed as a property. Every event below
// survived its own removal before domPropUnsetsAsProperty; the three in the
// table did not, which is why the bug stayed invisible.
func TestUnlistedEventHandlerIsRemovedFromTheNode(parseT *testing.T) {
	parseUnlisted := []string{
		"onmouseover", "onmouseout", "onpointerenter", "onpointerleave",
		"onpointercancel", "onkeypress", "onpaste", "oncopy", "ontoggle",
		"onfocusin", "onfocusout", "onauxclick", "onbeforeinput",
		"ondragenter", "ondragleave", "ontouchcancel", "onanimationstart",
		"ontransitionstart", "onselect", "oninvalid",
	}
	parseListed := []string{"onclick", "oninput", "onblur"}

	isStillAttached := func(parseName string) bool {
		parseT.Helper()
		parseAdapter := newTestDOMAdapter()
		parseScheduler := newTestScheduler()
		parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
		parseApp := parseAdapter.CreateElement("div")

		parseHandler := func() {}
		renderAndDrain(parseT, parseRt, parseScheduler, parseApp,
			CreateElement("section", nil,
				CreateElement("button", map[string]any{"id": "b", parseName: parseHandler}, "x"),
			))
		parseButton := findNodeByID(parseApp, "b")
		if parseButton == nil || parseButton.properties[parseName] == nil {
			parseT.Fatalf("setup: %s was not installed as a property", parseName)
		}

		renderAndDrain(parseT, parseRt, parseScheduler, parseApp,
			CreateElement("section", nil,
				CreateElement("button", map[string]any{"id": "b"}, "x"),
			))
		return parseButton.properties[parseName] != nil
	}

	for _, parseName := range parseListed {
		if isStillAttached(parseName) {
			parseT.Errorf("%s (in propMetaCache) survived removal", parseName)
		}
	}
	parseSurvived := []string{}
	for _, parseName := range parseUnlisted {
		if isStillAttached(parseName) {
			parseSurvived = append(parseSurvived, parseName)
		}
	}
	if len(parseSurvived) > 0 {
		parseT.Errorf("%d/%d handlers outside propMetaCache survived removal: %v",
			len(parseSurvived), len(parseUnlisted), parseSurvived)
	}
}

// A string-valued attribute must still be removed as an ATTRIBUTE. The fix above
// keys off the value's type, so this is the other half of the same contract:
// getting it wrong the other way would leave stale attributes in the DOM.
func TestStringAttributeIsStillRemovedAsAnAttribute(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseApp := parseAdapter.CreateElement("div")

	renderAndDrain(parseT, parseRt, parseScheduler, parseApp,
		CreateElement("section", nil,
			CreateElement("div", map[string]any{"id": "d", "data-tag": "keep", "title": "hello"}, "x"),
		))
	parseNode := findNodeByID(parseApp, "d")
	if parseNode == nil || parseNode.attributes["data-tag"] != "keep" {
		parseT.Fatal("setup: expected data-tag to be applied as an attribute")
	}

	renderAndDrain(parseT, parseRt, parseScheduler, parseApp,
		CreateElement("section", nil,
			CreateElement("div", map[string]any{"id": "d"}, "x"),
		))

	if parseValue, hasValue := parseNode.attributes["data-tag"]; hasValue && parseValue != "" {
		parseT.Errorf("data-tag=%q survived removal; string props must still be removed as attributes", parseValue)
	}
	if parseValue, hasValue := parseNode.attributes["title"]; hasValue && parseValue != "" {
		parseT.Errorf("title=%q survived removal", parseValue)
	}
}

// P2.2 must defer through the REAL scheduling path, not only when a test writes
// fiber.updateLane by hand.
//
// The existing accept test set the lane on a current-tree fiber itself and then
// asserted only that the work eventually rendered — true whether or not anything
// was deferred. Driving it through ScheduleUpdateForFiberWithOrigin exposed that
// the WIP clone dropped updateLane, so laneAdmitsFiber saw "no recorded lane"
// and admitted every fiber: nothing was ever deferred on any application path.
func TestLanes_BackgroundWorkIsDeferredThroughRealScheduling(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, Reset: true})
	parseContainer := parseAdapter.CreateElement("div")

	parseBackgroundRenders := 0
	parseBackgroundChild := func() *Element {
		parseBackgroundRenders++
		return CreateElement("span", map[string]any{}, "bg")
	}
	parseInputRenders := 0
	parseInputChild := func() *Element {
		parseInputRenders++
		return CreateElement("em", map[string]any{}, "in")
	}
	parseApp := func() *Element {
		return CreateElement("section", map[string]any{},
			CreateElement(parseBackgroundChild, map[string]any{}),
			CreateElement(parseInputChild, map[string]any{}),
		)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)
	parseAfterMount := parseBackgroundRenders
	parseInputAfterMount := parseInputRenders

	parseSection := parseRt.currentRoot.child.child
	parseBackgroundFiber := parseSection.child
	parseInputFiber := parseBackgroundFiber.sibling
	if parseBackgroundFiber == nil || parseInputFiber == nil {
		parseT.Fatal("expected two child fibers")
	}

	// Background work arrives first, input work second, so the coalesced pass
	// runs at input priority with a background-marked fiber inside it.
	parseRt.ScheduleUpdateForFiberWithOrigin(parseBackgroundFiber, "background")
	parseRt.ScheduleUpdateForFiberWithOrigin(parseInputFiber, "input")

	if len(parseScheduler.timeouts) == 0 {
		parseT.Fatal("expected a scheduled pass")
	}
	parseFirstPass := parseScheduler.timeouts[0]
	parseScheduler.timeouts = parseScheduler.timeouts[1:]
	parseFirstPass()

	if parseInputRenders <= parseInputAfterMount {
		parseT.Fatal("setup: the input-lane child did not render in the input pass")
	}
	if parseBackgroundRenders > parseAfterMount {
		parseT.Errorf("background-lane child rendered inside an input-lane pass (%d -> %d): deferral did not fire",
			parseAfterMount, parseBackgroundRenders)
	}
	if !parseRt.schedulerState.lanes.pending[UpdateLaneBackground] {
		parseT.Error("the deferred lane was not recorded as pending, so no follow-up pass is owed")
	}

	// Deferred is not dropped: the follow-up pass must land it.
	runScheduledTimeouts(parseScheduler)
	if parseBackgroundRenders <= parseAfterMount {
		parseT.Error("deferred background work never rendered; deferral must not strand it")
	}
}

// The lane must survive BOTH clone paths — the update clone and the bailout
// clone — since either can be how a marked fiber reaches the work loop.
func TestLanes_CloneCarriesTheUpdateLane(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Reset: true})

	parseParent := &Fiber{typeOf: "div"}
	parseOld := &Fiber{typeOf: "span", updateLane: UpdateLaneBackground, dirty: true}
	parseParent.alternate = &Fiber{typeOf: "div", child: parseOld}

	parseUpdated := parseRt.buildUpdatedFiber(parseParent, parseOld, CreateElement("span", nil))
	if parseUpdated.updateLane != UpdateLaneBackground {
		parseT.Errorf("buildUpdatedFiber dropped the lane: got %v", parseUpdated.updateLane)
	}

	parseBailoutParent := &Fiber{typeOf: "div", alternate: &Fiber{typeOf: "div", child: parseOld}}
	parseRt.cloneChildFibers(parseBailoutParent)
	if parseBailoutParent.child == nil || parseBailoutParent.child.updateLane != UpdateLaneBackground {
		parseT.Errorf("cloneChildFibers dropped the lane: got %v", parseBailoutParent.child.updateLane)
	}
}

// Duplicate sibling keys are reported. The in-order fast path cannot see them
// (trailing appends are validated for castability only), and the slow path
// tolerates rather than resolves them, so the only real fix is upstream — which
// means the framework has to say so.
func TestDuplicateSiblingKeysAreReported(parseT *testing.T) {
	if !hookThreadingGuardEnabled {
		parseT.Skip("dev-only diagnostic; stripped from production builds")
	}
	duplicateKeyWarned.Store(false)
	defer duplicateKeyWarned.Store(false)
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseRt := &Runtime{}
	parseParent := &Fiber{typeOf: "ul"}
	parseRt.reconcileChildren(parseParent, []any{
		CreateElement("li", map[string]any{"key": "a"}),
		CreateElement("li", map[string]any{"key": "b"}),
		CreateElement("li", map[string]any{"key": "a"}),
	})

	parseFound := false
	for _, parseDiagnostic := range GetDiagnostics() {
		if strings.Contains(parseDiagnostic.Message, "share the key") {
			parseFound = true
		}
	}
	if !parseFound {
		parseT.Error("expected a duplicate-key diagnostic")
	}
}

// ...and a correctly keyed list must stay silent, or the warning is noise.
func TestUniqueSiblingKeysAreNotReported(parseT *testing.T) {
	if !hookThreadingGuardEnabled {
		parseT.Skip("dev-only diagnostic; stripped from production builds")
	}
	duplicateKeyWarned.Store(false)
	defer duplicateKeyWarned.Store(false)
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseRt := &Runtime{}
	parseParent := &Fiber{typeOf: "ul"}
	parseRt.reconcileChildren(parseParent, []any{
		CreateElement("li", map[string]any{"key": "a"}),
		CreateElement("li", map[string]any{"key": "b"}),
	})

	for _, parseDiagnostic := range GetDiagnostics() {
		if strings.Contains(parseDiagnostic.Message, "share the key") {
			parseT.Error("a uniquely keyed list must not produce a duplicate-key diagnostic")
		}
	}
}
