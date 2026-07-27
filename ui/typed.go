package ui

import (
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
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
//
// PASS A STABLE FUNCTION VALUE — a declared func, resolved once at package
// level, as shown above. Typed owns its handle for the process lifetime and
// installs a renderer on it, which is the deliberate form of the swap that
// getComponentHandle refuses to do accidentally (see the identity discussion in
// component_handle_shared.go). Calling Typed with a freshly built closure — or
// calling it per render — registers a separate handle each time: correct output,
// but it re-does the registration work Typed exists to do once, and the
// no-reflection renderer stops being shared.
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
