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

		// Use the cached reflect.Value so reflect.ValueOf is not called on every event dispatch.
		parseCurrentFnValue := parseCell.fnVal
		if !parseCurrentFnValue.IsValid() || parseCurrentFnValue.Kind() != reflect.Func {
			return buildEventResultValues(parseFnType)
		}

		return parseCurrentFnValue.Call(parseArgs)
	})

	return parseWrapper.Interface()
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
