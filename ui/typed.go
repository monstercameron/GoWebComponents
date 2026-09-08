package ui

import (
	"reflect"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
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
	parseHandle.SetImplementationTypedRenderer(parseComponent, func(parseImplementation any, parseRawProps any) *runtime.Element {
		parseTypedImplementation, parseOk := parseImplementation.(func(P) Node)
		if !parseOk {
			return renderComponent(parseImplementation, map[string]any{propsKey: parseRawProps})
		}
		parseProps, _ := parseRawProps.(P)
		return parseTypedImplementation(parseProps)
	})
	parseHandle.SetTypedPropsEqual(buildTypedPropsEqual[P]())
	return func(parseProps P) Node {
		return runtime.CreateTypedComponentElement(parseHandle, parseProps)
	}
}

// buildTypedPropsEqual constructs one comparer per Typed registration. Values
// with ordinary comparable types retain exact Go equality. Structs/arrays that
// contain slices, maps, pointers, or functions use conservative shallow
// identity for those reference fields and exact equality for scalar fields.
// A false result merely rerenders; a true result always means the observable
// typed payload is unchanged under the framework's shallow dependency model.
func buildTypedPropsEqual[P any]() func(any, any) bool {
	parseType := reflect.TypeFor[P]()
	return func(parseLeft, parseRight any) bool {
		parseLeftTyped, parseLeftOK := parseLeft.(P)
		parseRightTyped, parseRightOK := parseRight.(P)
		if !parseLeftOK || !parseRightOK {
			return false
		}
		parseLeftValue := reflect.ValueOf(parseLeftTyped)
		parseRightValue := reflect.ValueOf(parseRightTyped)
		if parseType == nil {
			return !parseLeftValue.IsValid() && !parseRightValue.IsValid()
		}
		if parseLeftValue.Comparable() && parseRightValue.Comparable() {
			return parseLeft == parseRight
		}
		return shallowTypedPropsEqualValue(parseLeftValue, parseRightValue)
	}
}

func shallowTypedPropsEqualValue(parseLeft, parseRight reflect.Value) bool {
	if !parseLeft.IsValid() || !parseRight.IsValid() {
		return parseLeft.IsValid() == parseRight.IsValid()
	}
	if parseLeft.Type() != parseRight.Type() {
		return false
	}
	switch parseLeft.Kind() {
	case reflect.Interface:
		if parseLeft.IsNil() || parseRight.IsNil() {
			return parseLeft.IsNil() && parseRight.IsNil()
		}
		return shallowTypedPropsEqualValue(parseLeft.Elem(), parseRight.Elem())
	case reflect.Struct:
		for parseIndex := 0; parseIndex < parseLeft.NumField(); parseIndex++ {
			if !shallowTypedPropsEqualValue(parseLeft.Field(parseIndex), parseRight.Field(parseIndex)) {
				return false
			}
		}
		return true
	case reflect.Array:
		for parseIndex := 0; parseIndex < parseLeft.Len(); parseIndex++ {
			if !shallowTypedPropsEqualValue(parseLeft.Index(parseIndex), parseRight.Index(parseIndex)) {
				return false
			}
		}
		return true
	case reflect.Slice:
		return parseLeft.IsNil() == parseRight.IsNil() && parseLeft.Len() == parseRight.Len() && parseLeft.Pointer() == parseRight.Pointer()
	case reflect.Map, reflect.Pointer, reflect.Chan, reflect.UnsafePointer:
		return parseLeft.IsNil() == parseRight.IsNil() && parseLeft.Pointer() == parseRight.Pointer()
	case reflect.Func:
		return parseLeft.IsNil() && parseRight.IsNil()
	case reflect.String:
		return parseLeft.String() == parseRight.String()
	case reflect.Bool:
		return parseLeft.Bool() == parseRight.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return parseLeft.Int() == parseRight.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return parseLeft.Uint() == parseRight.Uint()
	case reflect.Float32, reflect.Float64:
		return parseLeft.Float() == parseRight.Float()
	case reflect.Complex64, reflect.Complex128:
		return parseLeft.Complex() == parseRight.Complex()
	default:
		return false
	}
}
