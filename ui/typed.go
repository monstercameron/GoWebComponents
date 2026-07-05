package ui

import (
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// Typed registers a props-taking component once and returns a constructor
// that creates its elements without any reflection on the render path.
//
// A component passed to CreateElement with a custom props struct renders
// through a reflect.Value.Call trampoline on every render — measurable
// overhead in wasm, where reflection is expensive. Typed builds a statically
// dispatched renderer instead:
//
//	var HookCell = ui.Typed(renderHookCell) // package level, once
//	...
//	HookCell(HookCellProps{Index: i})       // in render bodies
//
// The returned constructor is equivalent to
// ui.CreateElement(component, props): same component identity (handle
// cache), same reconciliation, hooks work unchanged. Mixing is safe — plain
// CreateElement calls for the same function value keep the static renderer
// (the implementation-identity guard skips re-registration).
func Typed[P any](parseComponent func(P) Node) func(P) Node {
	parseHandle := getComponentHandle(parseComponent)
	parseHandle.SetImplementationRenderer(parseComponent, func(parseImplementation any, parseRawProps map[string]any) *runtime.Element {
		parseTypedImplementation, parseOk := parseImplementation.(func(P) Node)
		if !parseOk {
			// Hot reload swapped in a different implementation shape; fall
			// back to the generic renderer rather than dropping the render.
			return renderComponent(parseImplementation, parseRawProps)
		}
		var parseProps P
		if parseProvided, parseOk2 := parseRawProps[propsKey].(P); parseOk2 {
			parseProps = parseProvided
		}
		return parseTypedImplementation(parseProps)
	})
	return func(parseProps P) Node {
		return runtime.CreateElementOwned(parseHandle, map[string]any{propsKey: parseProps})
	}
}
