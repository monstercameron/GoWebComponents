package runtime

import "fmt"

// ErrorBoundaryType marks runtime-recognized error-boundary elements.
type ErrorBoundaryType struct{}

// NewErrorBoundaryType creates a marker type recognized by the runtime as an error boundary.
func NewErrorBoundaryType() *ErrorBoundaryType {
	return &ErrorBoundaryType{}
}

type boundaryPhase string

const (
	boundaryPhaseRender  boundaryPhase = "render"
	boundaryPhaseEffect  boundaryPhase = "effect"
	boundaryPhaseCleanup boundaryPhase = "cleanup"
	boundaryPhaseEvent   boundaryPhase = "event"
)

func boundaryCapturedError(fiber *Fiber) error {
	if fiber == nil {
		return nil
	}
	if fiber.boundaryError != nil {
		return fiber.boundaryError
	}
	if fiber.alternate != nil {
		return fiber.alternate.boundaryError
	}
	return nil
}

func (rt *Runtime) recoverBoundaryError(source *Fiber, recovered interface{}, phase boundaryPhase) (*Fiber, bool) {
	boundary := findNearestErrorBoundary(source)
	if boundary == nil {
		return nil, false
	}

	err := normalizeBoundaryError(recovered)
	rt.setBoundaryError(boundary, err, string(phase))
	rt.invokeBoundaryOnError(boundary, err)
	ReportDiagnostic("runtime", DiagnosticWarning, fmt.Sprintf("error boundary caught %s failure: %v", phase, err))

	if phase == boundaryPhaseRender {
		rt.renderBoundaryChildren(boundary)
		if boundary.child != nil {
			return boundary.child, true
		}
		return rt.getNextUnitOfWork(boundary), true
	}

	rt.requestBoundaryRecovery(boundary)
	return boundary, true
}

func normalizeBoundaryError(recovered interface{}) error {
	if err, ok := recovered.(error); ok {
		return err
	}
	return fmt.Errorf("%v", recovered)
}

func findNearestErrorBoundary(source *Fiber) *Fiber {
	for fiber := source; fiber != nil; fiber = fiber.parent {
		if _, ok := fiber.typeOf.(*ErrorBoundaryType); !ok {
			continue
		}
		if boundaryCapturedError(fiber) != nil {
			continue
		}
		return fiber
	}
	return nil
}

func (rt *Runtime) setBoundaryError(boundary *Fiber, err error, phase string) {
	if boundary == nil {
		return
	}
	boundary.boundaryError = err
	boundary.boundaryPhase = phase
	if boundary.alternate != nil {
		boundary.alternate.boundaryError = err
		boundary.alternate.boundaryPhase = phase
	}
}

func (rt *Runtime) clearBoundaryError(boundary *Fiber) {
	if boundary == nil {
		return
	}
	boundary.boundaryError = nil
	boundary.boundaryPhase = ""
	if boundary.alternate != nil {
		boundary.alternate.boundaryError = nil
		boundary.alternate.boundaryPhase = ""
	}
}

func (rt *Runtime) invokeBoundaryOnError(boundary *Fiber, err error) {
	if boundary == nil || boundary.props == nil || err == nil {
		return
	}
	onError, _ := boundary.props["onError"].(func(error))
	if onError == nil {
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			ReportDiagnostic("runtime", DiagnosticWarning, fmt.Sprintf("error boundary onError callback panicked: %v", recovered))
		}
	}()
	onError(err)
}

func boundaryResetKeys(props map[string]interface{}) []interface{} {
	if props == nil {
		return nil
	}
	keys, _ := props["resetKeys"].([]interface{})
	return keys
}

func boundaryResetKeysChanged(fiber *Fiber) bool {
	if fiber == nil || fiber.alternate == nil {
		return false
	}
	return !areDepsEqual(boundaryResetKeys(fiber.alternate.props), boundaryResetKeys(fiber.props))
}

func (rt *Runtime) resetBoundary(boundary *Fiber) {
	if boundary == nil {
		return
	}
	rt.clearBoundaryError(boundary)
	rt.requestBoundaryRecovery(boundary)
}

func (rt *Runtime) requestBoundaryRecovery(boundary *Fiber) {
	if boundary == nil {
		return
	}
	alreadyScheduled := rt.updateScheduled
	rt.ScheduleUpdateForFiber(boundary)
	if alreadyScheduled {
		rt.pendingBoundaryRecovery = true
	}
}

func (rt *Runtime) renderBoundaryChildren(boundary *Fiber) {
	if boundary == nil {
		return
	}

	if boundaryCapturedError(boundary) != nil && boundaryResetKeysChanged(boundary) {
		rt.clearBoundaryError(boundary)
	}

	if err := boundaryCapturedError(boundary); err != nil {
		boundary.boundaryError = err
		if boundary.boundaryPhase == "" && boundary.alternate != nil {
			boundary.boundaryPhase = boundary.alternate.boundaryPhase
		}
		fallback := rt.renderBoundaryFallback(boundary, err)
		if fallback != nil {
			rt.reconcileChildren(boundary, []interface{}{fallback})
			return
		}
		rt.reconcileChildren(boundary, emptyChildren)
		return
	}

	if propsChildren, ok := boundary.props["children"]; ok {
		if elements, elementsOk := propsChildren.([]interface{}); elementsOk {
			rt.reconcileChildren(boundary, elements)
			return
		}
	}
	rt.reconcileChildren(boundary, emptyChildren)
}

func (rt *Runtime) renderBoundaryFallback(boundary *Fiber, err error) (fallback *Element) {
	if boundary == nil || boundary.props == nil {
		return nil
	}

	if staticFallback, ok := boundary.props["fallback"].(*Element); ok && staticFallback != nil {
		return staticFallback
	}

	fallbackFn, _ := boundary.props["errorFallback"].(func(error, func()) *Element)
	if fallbackFn == nil {
		return nil
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			outer, handled := rt.recoverBoundaryError(boundary.parent, recovered, boundaryPhaseRender)
			if handled {
				if outer != nil && outer != boundary {
					fallback = nil
				}
				return
			}
			panic(recovered)
		}
	}()

	return fallbackFn(err, func() { rt.resetBoundary(boundary) })
}
