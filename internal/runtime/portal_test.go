package runtime

import "testing"

func renderAndDrain(parseT *testing.T, parseRt *Runtime, parseScheduler *testScheduler, parseContainer DOMNode, parseElement *Element) {
	parseT.Helper()
	parseRt.Render(parseElement, parseContainer)
	drainScheduledTimeouts(parseT, parseScheduler, 64)
}

func TestPortalCommitsChildrenToSelectorTarget(parseT *testing.T) {
	parseAdapter := newQueryTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseApp := parseAdapter.CreateElement("div")
	parseOverlay := parseAdapter.CreateElement("div")
	parseAdapter.selectorResults["#overlay-root"] = parseOverlay

	parseElement := CreateElement("section", map[string]any{"id": "shell"},
		CreateElement("p", map[string]any{"id": "inline"}, "inline"),
		CreateElement(PortalNodeType, map[string]any{"portalTargetSelector": "#overlay-root"},
			CreateElement("div", map[string]any{"id": "portaled"}, "overlay"),
		),
	)

	renderAndDrain(parseT, parseRt, parseScheduler, parseApp, parseElement)

	if parseGot := findNodeByID(parseApp, "portaled"); parseGot != nil {
		parseT.Fatal("expected portal child to be absent from logical app container")
	}
	if parseGot2 := findNodeByID(parseApp, "inline"); parseGot2 == nil {
		parseT.Fatal("expected inline child to remain in logical app container")
	}
	if parseGot3 := findNodeByID(parseOverlay, "portaled"); parseGot3 == nil {
		parseT.Fatal("expected portal child to mount under selector target")
	}
}

func TestPortalCommitsChildrenToExplicitNodeTarget(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseApp := parseAdapter.CreateElement("div")
	parseOverlay := parseAdapter.CreateElement("div")

	parseElement := CreateElement("section", nil,
		CreateElement(PortalNodeType, map[string]any{"portalTargetNode": parseOverlay},
			CreateElement("button", map[string]any{"id": "portaled-button"}, "click"),
		),
	)

	renderAndDrain(parseT, parseRt, parseScheduler, parseApp, parseElement)

	if parseGot := findNodeByID(parseOverlay, "portaled-button"); parseGot == nil {
		parseT.Fatal("expected portal child to mount under explicit node target")
	}
	if parseGot2 := findNodeByID(parseApp, "portaled-button"); parseGot2 != nil {
		parseT.Fatal("expected explicit-node portal child to stay out of app container")
	}
}

func TestPortalRetargetMovesExistingSubtree(parseT *testing.T) {
	parseAdapter := newQueryTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseApp := parseAdapter.CreateElement("div")
	parseLeft := parseAdapter.CreateElement("div")
	parseRight := parseAdapter.CreateElement("div")
	parseAdapter.selectorResults["#left"] = parseLeft
	parseAdapter.selectorResults["#right"] = parseRight

	parseFirst := CreateElement("section", nil,
		CreateElement(PortalNodeType, map[string]any{"portalTargetSelector": "#left"},
			CreateElement("div", map[string]any{"id": "moving"}, "one"),
		),
	)
	parseSecond := CreateElement("section", nil,
		CreateElement(PortalNodeType, map[string]any{"portalTargetSelector": "#right"},
			CreateElement("div", map[string]any{"id": "moving"}, "one"),
		),
	)

	renderAndDrain(parseT, parseRt, parseScheduler, parseApp, parseFirst)
	if parseGot := findNodeByID(parseLeft, "moving"); parseGot == nil {
		parseT.Fatal("expected portal child under initial target")
	}

	renderAndDrain(parseT, parseRt, parseScheduler, parseApp, parseSecond)
	if parseGot2 := findNodeByID(parseLeft, "moving"); parseGot2 != nil {
		parseT.Fatal("expected portal child to move away from old target")
	}
	if parseGot3 := findNodeByID(parseRight, "moving"); parseGot3 == nil {
		parseT.Fatal("expected portal child to move to new target")
	}
}

func TestPortalDeletionCleansTargetSubtree(parseT *testing.T) {
	parseAdapter := newQueryTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseApp := parseAdapter.CreateElement("div")
	parseOverlay := parseAdapter.CreateElement("div")
	parseAdapter.selectorResults["#overlay-root"] = parseOverlay

	parseWithPortal := CreateElement("section", nil,
		CreateElement(PortalNodeType, map[string]any{"portalTargetSelector": "#overlay-root"},
			CreateElement("div", map[string]any{"id": "portaled"}, "overlay"),
		),
	)
	parseWithoutPortal := CreateElement("section", nil,
		CreateElement("p", map[string]any{"id": "inline"}, "plain"),
	)

	renderAndDrain(parseT, parseRt, parseScheduler, parseApp, parseWithPortal)
	if parseGot := findNodeByID(parseOverlay, "portaled"); parseGot == nil {
		parseT.Fatal("expected portal child to mount before deletion")
	}

	renderAndDrain(parseT, parseRt, parseScheduler, parseApp, parseWithoutPortal)
	if parseGot2 := findNodeByID(parseOverlay, "portaled"); parseGot2 != nil {
		parseT.Fatal("expected portal child to be removed from target on unmount")
	}
}
