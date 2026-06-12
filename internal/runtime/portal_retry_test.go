package runtime

import "testing"

// TestPortalRetriesUnresolvedTargetOnNextCommit is the regression test for the
// permanently-empty-portal defect: when a portal's target selector does not
// resolve at first commit (target appears later in the tree or in the page),
// the portal must mount its subtree on the next commit once the target exists,
// instead of staying empty forever.
func TestPortalRetriesUnresolvedTargetOnNextCommit(parseT *testing.T) {
	parseAdapter := newQueryTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseApp := parseAdapter.CreateElement("div")

	buildTree := func(parseLabel string) *Element {
		return CreateElement("section", map[string]any{"id": "shell"},
			CreateElement(PortalNodeType, map[string]any{"portalTargetSelector": "#late-root"},
				CreateElement("div", map[string]any{"id": "portaled"}, parseLabel),
			),
			CreateElement("p", map[string]any{"id": "inline"}, "inline"),
		)
	}

	// First commit: the target does not exist anywhere.
	renderAndDrain(parseT, parseRt, parseScheduler, parseApp, buildTree("first"))
	if findNodeByID(parseApp, "portaled") != nil {
		parseT.Fatal("portal child must not fall back into the app container")
	}

	// The target element appears (as the hosting page would provide it later).
	parseLate := parseAdapter.CreateElement("div")
	parseAdapter.selectorResults["#late-root"] = parseLate

	// Next commit: the portal must now mount into the late target.
	renderAndDrain(parseT, parseRt, parseScheduler, parseApp, buildTree("second"))
	parsePortaled := findNodeByID(parseLate, "portaled")
	if parsePortaled == nil {
		parseT.Fatal("portal did not retry its unresolved target on the next commit")
	}

	// And it must keep updating normally afterwards.
	renderAndDrain(parseT, parseRt, parseScheduler, parseApp, buildTree("third"))
	if findNodeByID(parseLate, "portaled") == nil {
		parseT.Fatal("portal subtree lost after retry")
	}
}
