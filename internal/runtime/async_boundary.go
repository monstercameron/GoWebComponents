package runtime

import "fmt"

// Suspension describes render-time async work that should be retried after Done closes.
type Suspension struct {
	Reason string
	Done   <-chan struct{}
}

// Error returns the suspension reason.
func (parseS *Suspension) Error() string {
	if parseS == nil || parseS.Reason == "" {
		return "render suspended"
	}
	return parseS.Reason
}

// NewSuspension creates a render suspension payload.
func NewSuspension(parseDone <-chan struct{}, parseReason string) *Suspension {
	if parseReason == "" {
		parseReason = "render suspended"
	}
	return &Suspension{Reason: parseReason, Done: parseDone}
}

// SuspendUntil interrupts the current render until done closes.
func SuspendUntil(parseDone <-chan struct{}, parseReason string) {
	if doneChannelClosed(parseDone) {
		return
	}
	panic(NewSuspension(parseDone, parseReason))
}

// AsSuspension returns a suspension payload when recovered carries one.
func AsSuspension(parseRecovered interface{}) (*Suspension, bool) {
	switch parseTyped := parseRecovered.(type) {
	case *Suspension:
		return parseTyped, parseTyped != nil
	case Suspension:
		parseCopy := parseTyped
		return &parseCopy, true
	default:
		return nil, false
	}
}

// doneChannelClosed reports whether a completion channel has already resolved.
func doneChannelClosed(parseDone <-chan struct{}) bool {
	if parseDone == nil {
		return false
	}
	select {
	case <-parseDone:
		return true
	default:
		return false
	}
}

// suspensionResolved reports whether a suspension can be retried immediately.
func suspensionResolved(parseSuspension *Suspension) bool {
	if parseSuspension == nil {
		return false
	}
	return doneChannelClosed(parseSuspension.Done)
}

// findNearestAsyncBoundary is a core package helper.
func findNearestAsyncBoundary(parseSource *Fiber) *Fiber {
	for parseFiber := parseSource; parseFiber != nil; parseFiber = parseFiber.parent {
		if _, parseOk := parseFiber.typeOf.(*AsyncBoundaryElementType); parseOk {
			return parseFiber
		}
	}
	return nil
}

// recoverAsyncBoundarySuspension records a suspension on the nearest async boundary.
func (parseRt *Runtime) recoverAsyncBoundarySuspension(parseSource *Fiber, parseSuspension *Suspension) (*Fiber, bool) {
	parseBoundary := findNearestAsyncBoundary(parseSource)
	if parseBoundary == nil || parseSuspension == nil {
		return nil, false
	}

	parseRt.setAsyncBoundarySuspension(parseBoundary, parseSuspension)
	parseRt.subscribeAsyncBoundary(parseBoundary, parseSuspension)
	parseRt.renderAsyncBoundaryChildren(parseBoundary)
	if parseBoundary.child != nil {
		return parseBoundary.child, true
	}
	return parseRt.getNextUnitOfWork(parseBoundary), true
}

// setAsyncBoundarySuspension stores pending suspension state on both fiber generations.
func (parseRt *Runtime) setAsyncBoundarySuspension(parseBoundary *Fiber, parseSuspension *Suspension) {
	if parseBoundary == nil {
		return
	}
	parseBoundary.asyncSuspension = parseSuspension
	parseBoundary.asyncWait = nil
	if parseBoundary.alternate != nil {
		parseBoundary.alternate.asyncSuspension = parseSuspension
		parseBoundary.alternate.asyncWait = nil
	}
}

// clearAsyncBoundarySuspension clears pending suspension state on both fiber generations.
func (parseRt *Runtime) clearAsyncBoundarySuspension(parseBoundary *Fiber) {
	if parseBoundary == nil {
		return
	}
	parseBoundary.asyncSuspension = nil
	parseBoundary.asyncWait = nil
	if parseBoundary.alternate != nil {
		parseBoundary.alternate.asyncSuspension = nil
		parseBoundary.alternate.asyncWait = nil
	}
}

// subscribeAsyncBoundary schedules a boundary retry when a suspension resolves.
func (parseRt *Runtime) subscribeAsyncBoundary(parseBoundary *Fiber, parseSuspension *Suspension) {
	if parseRt == nil || parseBoundary == nil || parseSuspension == nil || parseSuspension.Done == nil {
		return
	}
	if parseBoundary.asyncWait == parseSuspension.Done {
		return
	}
	parseBoundary.asyncWait = parseSuspension.Done
	if parseBoundary.alternate != nil {
		parseBoundary.alternate.asyncWait = parseSuspension.Done
	}
	go func(parseDone <-chan struct{}, parseTarget *Fiber) {
		<-parseDone
		parseRt.ScheduleUpdateForFiberWithOrigin(parseTarget, "async-suspense")
	}(parseSuspension.Done, parseBoundary)
}

// asyncBoundaryCapturedSuspension returns the active suspension for a boundary.
func asyncBoundaryCapturedSuspension(parseFiber *Fiber) *Suspension {
	if parseFiber == nil {
		return nil
	}
	if parseFiber.asyncSuspension != nil {
		return parseFiber.asyncSuspension
	}
	if parseFiber.alternate != nil {
		return parseFiber.alternate.asyncSuspension
	}
	return nil
}

// renderAsyncBoundaryChildren renders explicit async states or normal content.
func (parseRt *Runtime) renderAsyncBoundaryChildren(parseBoundary *Fiber) {
	if parseBoundary == nil {
		return
	}

	if parseErr, _ := parseBoundary.props["error"].(error); parseErr != nil {
		parseRt.reconcileChildren(parseBoundary, asyncBoundaryFallbackChildren(parseBoundary, parseErr))
		return
	}

	if parsePending, _ := parseBoundary.props["pending"].(bool); parsePending {
		parseRt.reconcileChildren(parseBoundary, asyncBoundaryFallbackChildren(parseBoundary, nil))
		return
	}

	if parseSuspension := asyncBoundaryCapturedSuspension(parseBoundary); parseSuspension != nil {
		if suspensionResolved(parseSuspension) {
			parseRt.clearAsyncBoundarySuspension(parseBoundary)
		} else {
			parseRt.subscribeAsyncBoundary(parseBoundary, parseSuspension)
			parseRt.reconcileChildren(parseBoundary, asyncBoundaryFallbackChildren(parseBoundary, nil))
			return
		}
	}

	if parseContent := asyncBoundaryContent(parseBoundary); parseContent != nil {
		parseChildren := [1]interface{}{parseContent}
		parseRt.reconcileChildren(parseBoundary, parseChildren[:])
		return
	}

	parseRt.clearAsyncBoundarySuspension(parseBoundary)
	parseRt.reconcileChildren(parseBoundary, emptyChildren)
}

// asyncBoundaryFallbackChildren returns the fallback node list for a boundary.
func asyncBoundaryFallbackChildren(parseBoundary *Fiber, parseErr error) []interface{} {
	if parseBoundary == nil || parseBoundary.props == nil {
		return emptyChildren
	}
	if parseErr != nil {
		if parseFallbackFn, _ := parseBoundary.props["errorFallback"].(func(error) *Element); parseFallbackFn != nil {
			return singleElementChild(safeAsyncBoundaryFallback(parseFallbackFn, parseErr))
		}
	}
	if parseFallback, _ := parseBoundary.props["fallback"].(*Element); parseFallback != nil {
		return singleElementChild(parseFallback)
	}
	return emptyChildren
}

// safeAsyncBoundaryFallback isolates fallback callback panics from suspension handling.
func safeAsyncBoundaryFallback(parseFallback func(error) *Element, parseErr error) (parseElement *Element) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			panic(fmt.Errorf("async boundary fallback panic: %v", parseRecovered))
		}
	}()
	return parseFallback(parseErr)
}

// singleElementChild wraps one element for reconciliation.
func singleElementChild(parseElement *Element) []interface{} {
	if parseElement == nil {
		return emptyChildren
	}
	parseChildren := [1]interface{}{parseElement}
	return parseChildren[:]
}

// asyncBoundaryContent returns the configured content node.
func asyncBoundaryContent(parseBoundary *Fiber) *Element {
	if parseBoundary == nil {
		return nil
	}
	if parseContent, _ := parseBoundary.props["content"].(*Element); parseContent != nil {
		return parseContent
	}
	parseChildren := getFiberChildren(parseBoundary)
	if len(parseChildren) == 0 {
		return nil
	}
	parseContent, _ := parseChildren[0].(*Element)
	return parseContent
}
