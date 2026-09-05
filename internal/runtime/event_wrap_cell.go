package runtime

import "reflect"

// wrapEventHandlerCell builds one stable wrapper that dispatches to the cell's latest handler and owner.
func (parseRuntime *Runtime) wrapEventHandlerCell(parseCell *funcHandlerCell) any {
	if parseCell == nil || parseCell.fn == nil {
		return nil
	}

	parseFnType := reflect.TypeOf(parseCell.fn)
	if parseFnType == nil || parseFnType.Kind() != reflect.Func {
		return parseCell.fn
	}

	// DOM handlers overwhelmingly use one of these signatures. Keeping them as
	// ordinary Go closures avoids reflect.MakeFunc at registration and
	// reflect.Value.Call on every interaction (particularly costly in wasm).
	// The cell is still dereferenced at dispatch, so the wrapper keeps UseEvent's
	// latest-closure semantics.
	switch parseCell.fn.(type) {
	case func():
		return func() { parseRuntime.invokeEventCellNoArgs(parseCell) }
	case func(string):
		return func(parseValue string) { parseRuntime.invokeEventCellString(parseCell, parseValue) }
	}

	parseWrapper := reflect.MakeFunc(parseFnType, func(parseArgs []reflect.Value) (parseResults []reflect.Value) {
		parseRuntime.recordFirstInteraction("event")
		defer func() {
			if parseRecovered := recover(); parseRecovered != nil {
				if panicPhaseMayRecoverWithBoundary(PanicPhaseEvent) {
					if _, parseHandled := parseRuntime.recoverBoundaryError(parseCell.owner, parseRecovered, boundaryPhaseEvent); parseHandled {
						parseResults = buildEventResultValues(parseFnType)
						return
					}
				}
				panicFinalUnhandledPanic(parseCell.owner, boundaryPhaseEvent, parseRecovered)
				parseResults = buildEventResultValues(parseFnType)
			}
		}()

		// A handler runs ON the frame loop, so setters it calls apply directly.
		// Without this mark every click would post instead, costing one extra
		// task before the render was even scheduled.
		parseRuntime.enterFrameLoop()
		defer parseRuntime.exitFrameLoop()

		// Use the cached reflect.Value so reflect.ValueOf is not called on every event dispatch.
		parseCurrentFnValue := parseCell.fnVal
		if !parseCurrentFnValue.IsValid() || parseCurrentFnValue.Kind() != reflect.Func {
			return buildEventResultValues(parseFnType)
		}

		return parseCurrentFnValue.Call(parseArgs)
	})

	return parseWrapper.Interface()
}

func (parseRuntime *Runtime) invokeEventCellNoArgs(parseCell *funcHandlerCell) {
	parseRuntime.invokeEventCell(parseCell, func() {
		if parseFn, parseOk := parseCell.fn.(func()); parseOk {
			parseFn()
		}
	})
}

func (parseRuntime *Runtime) invokeEventCellString(parseCell *funcHandlerCell, parseValue string) {
	parseRuntime.invokeEventCell(parseCell, func() {
		if parseFn, parseOk := parseCell.fn.(func(string)); parseOk {
			parseFn(parseValue)
		}
	})
}

func (parseRuntime *Runtime) invokeEventCell(parseCell *funcHandlerCell, parseInvoke func()) {
	parseRuntime.recordFirstInteraction("event")
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			if panicPhaseMayRecoverWithBoundary(PanicPhaseEvent) {
				if _, parseHandled := parseRuntime.recoverBoundaryError(parseCell.owner, parseRecovered, boundaryPhaseEvent); parseHandled {
					return
				}
			}
			panicFinalUnhandledPanic(parseCell.owner, boundaryPhaseEvent, parseRecovered)
		}
	}()
	parseRuntime.enterFrameLoop()
	defer parseRuntime.exitFrameLoop()
	parseInvoke()
}

// buildEventResultValues allocates zero-valued results for one event wrapper signature.
func buildEventResultValues(parseFnType reflect.Type) []reflect.Value {
	parseResultCount := parseFnType.NumOut()
	if parseResultCount == 0 {
		return nil
	}

	parseResults := make([]reflect.Value, parseResultCount)
	for parseIndex := range parseResultCount {
		parseResults[parseIndex] = reflect.Zero(parseFnType.Out(parseIndex))
	}
	return parseResults
}

// markFrameLoopHandler wraps one plain callback so a dispatch of it counts as
// frame-loop work.
//
// wrapEventHandlerCell marks hook-created handlers inline, but callbacks wrapped
// through BuildDOMWrappedFunction* have no cell to mark. An unmarked handler
// looks async to the setter, so every state write inside it would post and cost
// an extra task before the render was even scheduled — on every interaction.
//
// Costs one reflect.MakeFunc per WRAP, not per dispatch, and only when async
// ingress is on; otherwise the callback is returned untouched.
func (parseRuntime *Runtime) markFrameLoopHandler(parseFn any) any {
	if parseRuntime == nil || parseFn == nil || !parseRuntime.asyncIngress {
		return parseFn
	}
	parseFnType := reflect.TypeOf(parseFn)
	if parseFnType == nil || parseFnType.Kind() != reflect.Func {
		return parseFn
	}
	// Variadic signatures need CallSlice rather than Call, and a wrong choice
	// here corrupts arguments rather than failing loudly. No event handler is
	// variadic, so the case is declined instead of guessed at.
	if parseFnType.IsVariadic() {
		return parseFn
	}
	parseFnValue := reflect.ValueOf(parseFn)
	return reflect.MakeFunc(parseFnType, func(parseArgs []reflect.Value) []reflect.Value {
		parseRuntime.enterFrameLoop()
		defer parseRuntime.exitFrameLoop()
		return parseFnValue.Call(parseArgs)
	}).Interface()
}
