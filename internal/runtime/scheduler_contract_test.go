package runtime

import "testing"

func TestScheduleUpdateForFiber_MarksCleanAncestorsEvenIfLeafAlreadyDirty(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{
		scheduler: parseScheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  make(map[string]any),
		},
	}

	parseRoot := &Fiber{typeOf: "root", dirty: false, needsUpdate: false}
	parseParent := &Fiber{typeOf: "parent", parent: parseRoot, dirty: false, needsUpdate: false}
	parseChild := &Fiber{typeOf: "child", parent: parseParent, dirty: true, needsUpdate: true}

	parseRt.ScheduleUpdateForFiber(parseChild)

	if !parseParent.dirty || !parseParent.needsUpdate {
		parseT.Fatal("expected clean parent to be marked even when leaf is already dirty")
	}
	if !parseRoot.dirty || !parseRoot.needsUpdate {
		parseT.Fatal("expected clean root ancestor to be marked even when leaf is already dirty")
	}
}

func TestRender_SchedulesWorkAndResetsDeletions(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseAdapter := newTestDOMAdapter()
	parseContainer := parseAdapter.CreateElement("div")
	parseCurrentRoot := &Fiber{
		typeOf:    "ROOT",
		dom:       parseContainer,
		props:     map[string]any{"children": []any{}},
		alternate: &Fiber{typeOf: "stale"},
	}
	parseRt := &Runtime{
		domAdapter:  parseAdapter,
		scheduler:   parseScheduler,
		currentRoot: parseCurrentRoot,
		deletions:   []*Fiber{{typeOf: "old"}},
	}
	parseElement := &Element{Type: "div", Props: map[string]any{"id": "app"}}

	parseRt.Render(parseElement, parseContainer)

	if parseRt.wipRoot == nil {
		parseT.Fatal("expected render to create a work-in-progress root")
	}
	if parseRt.wipRoot.alternate != parseCurrentRoot {
		parseT.Fatal("expected render to preserve the current root as alternate")
	}
	if parseCurrentRoot.alternate != nil {
		parseT.Fatal("expected render to break the old alternate chain")
	}
	parseChildren, parseOk := parseRt.wipRoot.props["children"].([]any)
	if !parseOk || len(parseChildren) != 1 || parseChildren[0] != parseElement {
		parseT.Fatal("expected render root props to contain the rendered element")
	}
	if len(parseRt.deletions) != 0 {
		parseT.Fatal("expected render to clear pending deletions")
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected render to schedule one timeout, got %d", len(parseScheduler.timeouts))
	}
	if !parseRt.updateScheduled {
		parseT.Fatal("expected render to mark the runtime as updateScheduled")
	}
}

func TestRender_ReusesPendingTimeoutWhenWorkAlreadyScheduled(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseAdapter := newTestDOMAdapter()
	parseFirstContainer := parseAdapter.CreateElement("div")
	parseSecondContainer := parseAdapter.CreateElement("div")
	parseCurrentRoot := &Fiber{
		typeOf: "ROOT",
		dom:    parseFirstContainer,
		props:  map[string]any{"children": []any{}},
	}
	parseRt := &Runtime{
		domAdapter:      parseAdapter,
		scheduler:       parseScheduler,
		currentRoot:     parseCurrentRoot,
		updateScheduled: true,
		deletions:       []*Fiber{{typeOf: "old"}},
	}

	parseSecondElement := &Element{Type: "section", Props: map[string]any{"id": "next"}}
	parseRt.Render(parseSecondElement, parseSecondContainer)

	if len(parseScheduler.timeouts) != 0 {
		parseT.Fatalf("expected render not to schedule an extra timeout when one is already pending, got %d", len(parseScheduler.timeouts))
	}
	if parseRt.wipRoot == nil {
		parseT.Fatal("expected render to replace the pending work-in-progress root")
	}
	if parseRt.wipRoot.dom != parseSecondContainer {
		parseT.Fatal("expected render to replace the pending container with the latest one")
	}
	parseChildren, parseOk := parseRt.wipRoot.props["children"].([]any)
	if !parseOk || len(parseChildren) != 1 || parseChildren[0] != parseSecondElement {
		parseT.Fatal("expected render to replace pending children with the latest rendered element")
	}
	if len(parseRt.deletions) != 0 {
		parseT.Fatal("expected render to clear stale deletions when replacing pending work")
	}
}
