package runtime

import "fmt"

// ErrorBoundaryType marks runtime-recognized error-boundary elements.
type ErrorBoundaryType struct{}

// NewErrorBoundaryType creates a marker type recognized by the runtime as an error boundary.
func NewErrorBoundaryType() *ErrorBoundaryType {
	return &ErrorBoundaryType{}
}

type PanicPhase string

type boundaryPhase = PanicPhase

const (
	PanicPhaseRender    PanicPhase = "render"
	PanicPhaseEffect    PanicPhase = "effect"
	PanicPhaseCleanup   PanicPhase = "cleanup"
	PanicPhaseEvent     PanicPhase = "event"
	PanicPhaseLoader    PanicPhase = "loader"
	PanicPhaseHydration PanicPhase = "hydration"
	PanicPhaseStartup   PanicPhase = "startup"
	PanicPhaseDeferred  PanicPhase = "deferred"
	PanicPhaseSSR       PanicPhase = "ssr"

	boundaryPhaseRender  boundaryPhase = PanicPhaseRender
	boundaryPhaseEffect  boundaryPhase = PanicPhaseEffect
	boundaryPhaseCleanup boundaryPhase = PanicPhaseCleanup
	boundaryPhaseEvent   boundaryPhase = PanicPhaseEvent
)

// boundaryCapturedError is a core package helper.
func boundaryCapturedError(parseFiber *Fiber) error {
	if parseFiber == nil {
		return nil
	}
	if parseFiber.boundaryError != nil {
		return parseFiber.boundaryError
	}
	if parseFiber.alternate != nil {
		return parseFiber.alternate.boundaryError
	}
	return nil
}

// recoverBoundaryError is a core package helper.
func (parseRt *Runtime) recoverBoundaryError(parseSource *Fiber, parseRecovered interface{}, parsePhase boundaryPhase) (*Fiber, bool) {
	parseBoundary := findNearestErrorBoundary(parseSource)
	if parseBoundary == nil {
		return nil, false
	}

	parseErr := normalizeBoundaryError(parseRecovered)
	parseRt.setBoundaryError(parseBoundary, parseErr, string(parsePhase))
	parseRt.invokeBoundaryOnError(parseBoundary, parseErr)
	ReportDiagnosticWithContext(
		"runtime",
		DiagnosticWarning,
		fmt.Sprintf("error boundary caught %s failure: %v", parsePhase, parseErr),
		diagnosticPathForFiber(parseSource),
		diagnosticComponentStack(parseSource),
	)

	if parsePhase == boundaryPhaseRender {
		parseRt.renderBoundaryChildren(parseBoundary)
		if parseBoundary.child != nil {
			return parseBoundary.child, true
		}
		return parseRt.getNextUnitOfWork(parseBoundary), true
	}

	parseRt.requestBoundaryRecovery(parseBoundary)
	return parseBoundary, true
}

// normalizeBoundaryError is a core package helper.
func normalizeBoundaryError(parseRecovered interface{}) error {
	if parseErr, parseOk := parseRecovered.(error); parseOk {
		return parseErr
	}
	return fmt.Errorf("%v", parseRecovered)
}

// findNearestErrorBoundary is a core package helper.
func findNearestErrorBoundary(parseSource *Fiber) *Fiber {
	for parseFiber := parseSource; parseFiber != nil; parseFiber = parseFiber.parent {
		if _, parseOk := parseFiber.typeOf.(*ErrorBoundaryType); !parseOk {
			continue
		}
		if boundaryCapturedError(parseFiber) != nil {
			continue
		}
		return parseFiber
	}
	return nil
}

// setBoundaryError is a core package helper.
func (parseRt *Runtime) setBoundaryError(parseBoundary *Fiber, parseErr error, parsePhase string) {
	if parseBoundary == nil {
		return
	}
	parseBoundary.boundaryError = parseErr
	parseBoundary.boundaryPhase = parsePhase
	if parseBoundary.alternate != nil {
		parseBoundary.alternate.boundaryError = parseErr
		parseBoundary.alternate.boundaryPhase = parsePhase
	}
}

// clearBoundaryError is a core package helper.
func (parseRt *Runtime) clearBoundaryError(parseBoundary *Fiber) {
	if parseBoundary == nil {
		return
	}
	parseBoundary.boundaryError = nil
	parseBoundary.boundaryPhase = ""
	if parseBoundary.alternate != nil {
		parseBoundary.alternate.boundaryError = nil
		parseBoundary.alternate.boundaryPhase = ""
	}
}

// invokeBoundaryOnError is a core package helper.
func (parseRt *Runtime) invokeBoundaryOnError(parseBoundary *Fiber, parseErr error) {
	if parseBoundary == nil || parseBoundary.props == nil || parseErr == nil {
		return
	}
	parseOnError, _ := parseBoundary.props["onError"].(func(error))
	if parseOnError == nil {
		return
	}
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			ReportDiagnosticWithContext(
				"runtime",
				DiagnosticWarning,
				fmt.Sprintf("error boundary onError callback panicked: %v", parseRecovered),
				diagnosticPathForFiber(parseBoundary),
				diagnosticComponentStack(parseBoundary),
			)
		}
	}()
	parseOnError(parseErr)
}

// boundaryResetKeys is a core package helper.
func boundaryResetKeys(parseProps map[string]interface{}) []interface{} {
	if parseProps == nil {
		return nil
	}
	parseKeys, _ := parseProps["resetKeys"].([]interface{})
	return parseKeys
}

// boundaryResetKeysChanged is a core package helper.
func boundaryResetKeysChanged(parseFiber *Fiber) bool {
	if parseFiber == nil || parseFiber.alternate == nil {
		return false
	}
	return !areDepsEqual(boundaryResetKeys(parseFiber.alternate.props), boundaryResetKeys(parseFiber.props))
}

// resetBoundary is a core package helper.
func (parseRt *Runtime) resetBoundary(parseBoundary *Fiber) {
	if parseBoundary == nil {
		return
	}
	parseRt.clearBoundaryError(parseBoundary)
	parseRt.requestBoundaryRecovery(parseBoundary)
}

// requestBoundaryRecovery is a core package helper.
func (parseRt *Runtime) requestBoundaryRecovery(parseBoundary *Fiber) {
	if parseBoundary == nil {
		return
	}
	parseAlreadyScheduled := parseRt.updateScheduled
	parseRt.ScheduleUpdateForFiberWithOrigin(parseBoundary, "error-boundary")
	if parseAlreadyScheduled {
		parseRt.pendingBoundaryRecovery = true
	}
}

// renderBoundaryChildren is a core package helper.
func (parseRt *Runtime) renderBoundaryChildren(parseBoundary *Fiber) {
	if parseBoundary == nil {
		return
	}

	if boundaryCapturedError(parseBoundary) != nil && boundaryResetKeysChanged(parseBoundary) {
		parseRt.clearBoundaryError(parseBoundary)
	}

	if parseErr := boundaryCapturedError(parseBoundary); parseErr != nil {
		parseBoundary.boundaryError = parseErr
		if parseBoundary.boundaryPhase == "" && parseBoundary.alternate != nil {
			parseBoundary.boundaryPhase = parseBoundary.alternate.boundaryPhase
		}
		parseFallback := parseRt.renderBoundaryFallback(parseBoundary, parseErr)
		if parseFallback != nil {
			parseRt.reconcileChildren(parseBoundary, []interface{}{parseFallback})
			return
		}
		parseRt.reconcileChildren(parseBoundary, emptyChildren)
		return
	}

	if parsePropsChildren, parseOk := parseBoundary.props["children"]; parseOk {
		if parseElements, parseElementsOk := parsePropsChildren.([]interface{}); parseElementsOk {
			parseRt.reconcileChildren(parseBoundary, parseElements)
			return
		}
	}
	parseRt.reconcileChildren(parseBoundary, emptyChildren)
}

// renderBoundaryFallback is a core package helper.
func (parseRt *Runtime) renderBoundaryFallback(parseBoundary *Fiber, parseErr error) (parseFallback *Element) {
	if parseBoundary == nil || parseBoundary.props == nil {
		return nil
	}

	if parseStaticFallback, parseOk := parseBoundary.props["fallback"].(*Element); parseOk && parseStaticFallback != nil {
		return parseStaticFallback
	}

	parseFallbackFn, _ := parseBoundary.props["errorFallback"].(func(error, func()) *Element)
	if parseFallbackFn == nil {
		return nil
	}

	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseOuter, parseHandled := parseRt.recoverBoundaryError(parseBoundary.parent, parseRecovered, boundaryPhaseRender)
			if parseHandled {
				if parseOuter != nil && parseOuter != parseBoundary {
					parseFallback = nil
				}
				return
			}
			panic(markUnhandledPanic(parseBoundary, boundaryPhaseRender, parseRecovered))
		}
	}()

	return parseFallbackFn(parseErr, func() { parseRt.resetBoundary(parseBoundary) })
}
