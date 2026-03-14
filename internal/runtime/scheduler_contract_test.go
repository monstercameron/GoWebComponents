package runtime

import "testing"

func TestScheduleUpdateForFiber_MarksCleanAncestorsEvenIfLeafAlreadyDirty(t *testing.T) {
	scheduler := newTestScheduler()
	rt := &Runtime{
		scheduler: scheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  make(map[string]interface{}),
		},
	}

	root := &Fiber{typeOf: "root", dirty: false, needsUpdate: false}
	parent := &Fiber{typeOf: "parent", parent: root, dirty: false, needsUpdate: false}
	child := &Fiber{typeOf: "child", parent: parent, dirty: true, needsUpdate: true}

	rt.ScheduleUpdateForFiber(child)

	if !parent.dirty || !parent.needsUpdate {
		t.Fatal("expected clean parent to be marked even when leaf is already dirty")
	}
	if !root.dirty || !root.needsUpdate {
		t.Fatal("expected clean root ancestor to be marked even when leaf is already dirty")
	}
}

func TestRender_SchedulesWorkAndResetsDeletions(t *testing.T) {
	scheduler := newTestScheduler()
	container := newTestDOMAdapter().CreateElement("div")
	currentRoot := &Fiber{
		typeOf:    "ROOT",
		dom:       container,
		props:     map[string]interface{}{"children": []interface{}{}},
		alternate: &Fiber{typeOf: "stale"},
	}
	rt := &Runtime{
		scheduler:   scheduler,
		currentRoot: currentRoot,
		deletions:   []*Fiber{{typeOf: "old"}},
	}
	element := &Element{Type: "div", Props: map[string]interface{}{"id": "app"}}

	rt.Render(element, container)

	if rt.wipRoot == nil {
		t.Fatal("expected render to create a work-in-progress root")
	}
	if rt.wipRoot.alternate != currentRoot {
		t.Fatal("expected render to preserve the current root as alternate")
	}
	if currentRoot.alternate != nil {
		t.Fatal("expected render to break the old alternate chain")
	}
	children, ok := rt.wipRoot.props["children"].([]interface{})
	if !ok || len(children) != 1 || children[0] != element {
		t.Fatal("expected render root props to contain the rendered element")
	}
	if len(rt.deletions) != 0 {
		t.Fatal("expected render to clear pending deletions")
	}
	if len(scheduler.timeouts) != 1 {
		t.Fatalf("expected render to schedule one timeout, got %d", len(scheduler.timeouts))
	}
	if !rt.updateScheduled {
		t.Fatal("expected render to mark the runtime as updateScheduled")
	}
}

func TestRender_ReusesPendingTimeoutWhenWorkAlreadyScheduled(t *testing.T) {
	scheduler := newTestScheduler()
	firstContainer := newTestDOMAdapter().CreateElement("div")
	secondContainer := newTestDOMAdapter().CreateElement("div")
	currentRoot := &Fiber{
		typeOf: "ROOT",
		dom:    firstContainer,
		props:  map[string]interface{}{"children": []interface{}{}},
	}
	rt := &Runtime{
		scheduler:       scheduler,
		currentRoot:     currentRoot,
		updateScheduled: true,
		deletions:       []*Fiber{{typeOf: "old"}},
	}

	secondElement := &Element{Type: "section", Props: map[string]interface{}{"id": "next"}}
	rt.Render(secondElement, secondContainer)

	if len(scheduler.timeouts) != 0 {
		t.Fatalf("expected render not to schedule an extra timeout when one is already pending, got %d", len(scheduler.timeouts))
	}
	if rt.wipRoot == nil {
		t.Fatal("expected render to replace the pending work-in-progress root")
	}
	if rt.wipRoot.dom != secondContainer {
		t.Fatal("expected render to replace the pending container with the latest one")
	}
	children, ok := rt.wipRoot.props["children"].([]interface{})
	if !ok || len(children) != 1 || children[0] != secondElement {
		t.Fatal("expected render to replace pending children with the latest rendered element")
	}
	if len(rt.deletions) != 0 {
		t.Fatal("expected render to clear stale deletions when replacing pending work")
	}
}
