package runtime

import "testing"

func renderAndDrain(t *testing.T, rt *Runtime, scheduler *testScheduler, container DOMNode, element *Element) {
	t.Helper()
	rt.Render(element, container)
	drainScheduledTimeouts(t, scheduler, 64)
}

func TestPortalCommitsChildrenToSelectorTarget(t *testing.T) {
	adapter := newQueryTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	app := adapter.CreateElement("div")
	overlay := adapter.CreateElement("div")
	adapter.selectorResults["#overlay-root"] = overlay

	element := CreateElement("section", map[string]interface{}{"id": "shell"},
		CreateElement("p", map[string]interface{}{"id": "inline"}, "inline"),
		CreateElement(PortalNodeType, map[string]interface{}{"portalTargetSelector": "#overlay-root"},
			CreateElement("div", map[string]interface{}{"id": "portaled"}, "overlay"),
		),
	)

	renderAndDrain(t, rt, scheduler, app, element)

	if got := findNodeByID(app, "portaled"); got != nil {
		t.Fatal("expected portal child to be absent from logical app container")
	}
	if got := findNodeByID(app, "inline"); got == nil {
		t.Fatal("expected inline child to remain in logical app container")
	}
	if got := findNodeByID(overlay, "portaled"); got == nil {
		t.Fatal("expected portal child to mount under selector target")
	}
}

func TestPortalCommitsChildrenToExplicitNodeTarget(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	app := adapter.CreateElement("div")
	overlay := adapter.CreateElement("div")

	element := CreateElement("section", nil,
		CreateElement(PortalNodeType, map[string]interface{}{"portalTargetNode": overlay},
			CreateElement("button", map[string]interface{}{"id": "portaled-button"}, "click"),
		),
	)

	renderAndDrain(t, rt, scheduler, app, element)

	if got := findNodeByID(overlay, "portaled-button"); got == nil {
		t.Fatal("expected portal child to mount under explicit node target")
	}
	if got := findNodeByID(app, "portaled-button"); got != nil {
		t.Fatal("expected explicit-node portal child to stay out of app container")
	}
}

func TestPortalRetargetMovesExistingSubtree(t *testing.T) {
	adapter := newQueryTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	app := adapter.CreateElement("div")
	left := adapter.CreateElement("div")
	right := adapter.CreateElement("div")
	adapter.selectorResults["#left"] = left
	adapter.selectorResults["#right"] = right

	first := CreateElement("section", nil,
		CreateElement(PortalNodeType, map[string]interface{}{"portalTargetSelector": "#left"},
			CreateElement("div", map[string]interface{}{"id": "moving"}, "one"),
		),
	)
	second := CreateElement("section", nil,
		CreateElement(PortalNodeType, map[string]interface{}{"portalTargetSelector": "#right"},
			CreateElement("div", map[string]interface{}{"id": "moving"}, "one"),
		),
	)

	renderAndDrain(t, rt, scheduler, app, first)
	if got := findNodeByID(left, "moving"); got == nil {
		t.Fatal("expected portal child under initial target")
	}

	renderAndDrain(t, rt, scheduler, app, second)
	if got := findNodeByID(left, "moving"); got != nil {
		t.Fatal("expected portal child to move away from old target")
	}
	if got := findNodeByID(right, "moving"); got == nil {
		t.Fatal("expected portal child to move to new target")
	}
}

func TestPortalDeletionCleansTargetSubtree(t *testing.T) {
	adapter := newQueryTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	app := adapter.CreateElement("div")
	overlay := adapter.CreateElement("div")
	adapter.selectorResults["#overlay-root"] = overlay

	withPortal := CreateElement("section", nil,
		CreateElement(PortalNodeType, map[string]interface{}{"portalTargetSelector": "#overlay-root"},
			CreateElement("div", map[string]interface{}{"id": "portaled"}, "overlay"),
		),
	)
	withoutPortal := CreateElement("section", nil,
		CreateElement("p", map[string]interface{}{"id": "inline"}, "plain"),
	)

	renderAndDrain(t, rt, scheduler, app, withPortal)
	if got := findNodeByID(overlay, "portaled"); got == nil {
		t.Fatal("expected portal child to mount before deletion")
	}

	renderAndDrain(t, rt, scheduler, app, withoutPortal)
	if got := findNodeByID(overlay, "portaled"); got != nil {
		t.Fatal("expected portal child to be removed from target on unmount")
	}
}