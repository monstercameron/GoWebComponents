package runtime

import "testing"

// TestCloneChildFibersMovesReactiveSourceSubscriptions verifies cloned reactive-source subscriptions move from stale fibers to current fibers.
func TestCloneChildFibersMovesReactiveSourceSubscriptions(parseT *testing.T) {
	parseRt := &Runtime{
		atomRegistry: NewAtomRegistry(),
	}
	parseRt.atomRegistry.InitAtom("count", 0)
	parseOldParent := &Fiber{typeOf: "section"}
	parseOldFiber := &Fiber{
		typeOf:            ReactiveRegionNodeType,
		parent:            parseOldParent,
		fineGrained:       true,
		reactiveSourceIDs: []string{"count"},
	}
	parseOldParent.child = parseOldFiber
	parseParent := &Fiber{
		typeOf:    "section",
		alternate: parseOldParent,
	}
	parseRt.atomRegistry.Subscribe("count", parseOldFiber)

	parseRt.cloneChildFibers(parseParent)

	parseNewFiber := parseParent.child
	if parseNewFiber == nil {
		parseT.Fatal("expected cloneChildFibers to produce one child fiber")
	}
	if parseNewFiber == parseOldFiber {
		parseT.Fatal("expected cloned child fiber to differ from old child fiber")
	}
	if parseRt.atomRegistry.GetSubscriberCount("count") != 1 {
		parseT.Fatalf("expected exactly one count subscriber after clone, got %d", parseRt.atomRegistry.GetSubscriberCount("count"))
	}
	parseRt.atomRegistry.mu.RLock()
	_, hasOldSubscriber := parseRt.atomRegistry.subscriptions["count"][parseOldFiber]
	_, hasNewSubscriber := parseRt.atomRegistry.subscriptions["count"][parseNewFiber]
	parseRt.atomRegistry.mu.RUnlock()
	if hasOldSubscriber {
		parseT.Fatal("expected old cloned fiber subscriber to be removed")
	}
	if !hasNewSubscriber {
		parseT.Fatal("expected new cloned fiber subscriber to be registered")
	}
}

// TestCloneChildFibersQueuesReactiveSourceHydrationSubscriptionMoves verifies hydration mode queues unsubscribe/subscribe actions for cloned reactive sources.
func TestCloneChildFibersQueuesReactiveSourceHydrationSubscriptionMoves(parseT *testing.T) {
	parseRt := &Runtime{
		atomRegistry: NewAtomRegistry(),
		hydrating:    true,
	}
	parseRt.atomRegistry.InitAtom("count", 0)
	parseOldParent := &Fiber{typeOf: "section"}
	parseOldFiber := &Fiber{
		typeOf:            ReactiveRegionNodeType,
		parent:            parseOldParent,
		fineGrained:       true,
		reactiveSourceIDs: []string{"count"},
	}
	parseOldParent.child = parseOldFiber
	parseParent := &Fiber{
		typeOf:    "section",
		alternate: parseOldParent,
	}
	parseRt.atomRegistry.Subscribe("count", parseOldFiber)

	parseRt.cloneChildFibers(parseParent)

	parseNewFiber := parseParent.child
	if parseNewFiber == nil {
		parseT.Fatal("expected cloneChildFibers to produce one child fiber")
	}
	if len(parseRt.deferredHydrationSubscriptions) != 2 {
		parseT.Fatalf("expected two hydration subscription actions, got %d", len(parseRt.deferredHydrationSubscriptions))
	}
	parseFirstAction := parseRt.deferredHydrationSubscriptions[0]
	parseSecondAction := parseRt.deferredHydrationSubscriptions[1]
	if parseFirstAction.atomID != "count" || parseFirstAction.subscribe || parseFirstAction.fiber != parseOldFiber {
		parseT.Fatalf("unexpected first hydration action: %#v", parseFirstAction)
	}
	if parseSecondAction.atomID != "count" || !parseSecondAction.subscribe || parseSecondAction.fiber != parseNewFiber {
		parseT.Fatalf("unexpected second hydration action: %#v", parseSecondAction)
	}
}

// TestReconcileChildrenMovesReactiveSourceSubscriptions verifies non-keyed reconciliation moves reactive-source subscriptions to the current fiber.
func TestReconcileChildrenMovesReactiveSourceSubscriptions(parseT *testing.T) {
	parseRt := &Runtime{
		atomRegistry: NewAtomRegistry(),
	}
	parseRt.atomRegistry.InitAtom("count", 0)
	parseOldParent := &Fiber{typeOf: "section"}
	parseOldFiber := &Fiber{
		typeOf:            "div",
		parent:            parseOldParent,
		fineGrained:       true,
		reactiveSourceIDs: []string{"count"},
	}
	parseOldParent.child = parseOldFiber
	parseWipParent := &Fiber{
		typeOf:    "section",
		alternate: parseOldParent,
	}
	parseRt.atomRegistry.Subscribe("count", parseOldFiber)

	parseRt.reconcileChildren(parseWipParent, []any{CreateElement("div", nil)})

	parseNewFiber := parseWipParent.child
	if parseNewFiber == nil {
		parseT.Fatal("expected reconcileChildren to produce one child fiber")
	}
	assertReactiveSourceSubscriptionMoved(parseT, parseRt, "count", parseOldFiber, parseNewFiber)
}

// TestReconcileKeyedChildrenMovesReactiveSourceSubscriptions verifies keyed reconciliation moves reactive-source subscriptions to the current fiber.
func TestReconcileKeyedChildrenMovesReactiveSourceSubscriptions(parseT *testing.T) {
	parseRt := &Runtime{
		atomRegistry: NewAtomRegistry(),
	}
	parseRt.atomRegistry.InitAtom("count", 0)
	parseOldParent := &Fiber{typeOf: "section"}
	parseOldFiber := &Fiber{
		typeOf:            "div",
		props:             map[string]any{"key": "slot-1"},
		parent:            parseOldParent,
		fineGrained:       true,
		reactiveSourceIDs: []string{"count"},
	}
	parseOldParent.child = parseOldFiber
	parseWipParent := &Fiber{
		typeOf:    "section",
		alternate: parseOldParent,
	}
	parseRt.atomRegistry.Subscribe("count", parseOldFiber)

	parseRt.reconcileChildren(parseWipParent, []any{CreateElement("div", map[string]any{"key": "slot-1"})})

	parseNewFiber := parseWipParent.child
	if parseNewFiber == nil {
		parseT.Fatal("expected keyed reconcileChildren to produce one child fiber")
	}
	assertReactiveSourceSubscriptionMoved(parseT, parseRt, "count", parseOldFiber, parseNewFiber)
}

// assertReactiveSourceSubscriptionMoved validates that one source subscription moved from one fiber to another.
func assertReactiveSourceSubscriptionMoved(parseT *testing.T, parseRt *Runtime, parseSourceID string, parseOldFiber *Fiber, parseNewFiber *Fiber) {
	parseT.Helper()
	if parseNewFiber == parseOldFiber {
		parseT.Fatal("expected new fiber to differ from old fiber")
	}
	if parseRt.atomRegistry.GetSubscriberCount(parseSourceID) != 1 {
		parseT.Fatalf(
			"expected exactly one %s subscriber after move, got %d",
			parseSourceID,
			parseRt.atomRegistry.GetSubscriberCount(parseSourceID),
		)
	}
	parseRt.atomRegistry.mu.RLock()
	_, hasOldSubscriber := parseRt.atomRegistry.subscriptions[parseSourceID][parseOldFiber]
	_, hasNewSubscriber := parseRt.atomRegistry.subscriptions[parseSourceID][parseNewFiber]
	parseRt.atomRegistry.mu.RUnlock()
	if hasOldSubscriber {
		parseT.Fatalf("expected old subscriber fiber to be removed for source %s", parseSourceID)
	}
	if !hasNewSubscriber {
		parseT.Fatalf("expected new subscriber fiber for source %s", parseSourceID)
	}
}
