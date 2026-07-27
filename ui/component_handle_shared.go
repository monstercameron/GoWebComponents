package ui

import (
	"fmt"
	"reflect"
	goRuntime "runtime"
	"strings"
	"sync"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

var componentHandleCache sync.Map
var getComponentIdentityCache sync.Map
var getComponentRenderCache sync.Map

type componentIdentityCacheKey struct {
	getType    reflect.Type
	getPointer uintptr
}

type componentIdentityCacheValue struct {
	getPrettyName    string
	getQualifiedName string
}

// getComponentHandle resolves the runtime handle that will render one component
// value.
//
// WHAT IDENTIFIES A COMPONENT — READ THIS BEFORE "SIMPLIFYING" THE CACHE
//
// There are two different questions here and they have two different answers.
// Collapsing them is the bug this function exists to prevent.
//
//  1. "Is this the same component as last render?" — a LOGICAL question, asked
//     by the reconciler (isSameType -> ComponentType.IdentityKey) to decide
//     whether to reuse a fiber and keep its hook state. The answer is the
//     qualified function name: MyForm re-created every render is still MyForm,
//     and its state must survive.
//
//  2. "Which code do I call to render THIS element?" — a PHYSICAL question,
//     asked at render time. The answer is the exact function VALUE the caller
//     passed, captured variables included. A name cannot answer it, and neither
//     can reflect.ValueOf(fn).Pointer(): every closure created from one `func`
//     literal shares one code pointer and one qualified name. Three closures
//     from the same literal are three different programs that report identical
//     names to `runtime.FuncForPC`.
//
// This function used to answer (2) with (1): it kept ONE handle per qualified
// name and, whenever a lookup arrived carrying a different function value, it
// overwrote that shared handle's implementation
// (handle.SetImplementationRenderer). Every element already built from that
// handle silently started rendering the newest closure's captures. N siblings
// built from one literal all rendered the LAST one.
//
// The corruption this caused was real and shipped: a form helper of the shape
//
//	func field(name string) ui.Node {
//	    return ui.CreateElement(func() ui.Node {   // one literal, N closures
//	        _ = ui.UseId()                         // a hook — hence the wrapper
//	        return html.Input(html.Props{Name: name})
//	    })
//	}
//
// rendered four inputs all named "subject" on one page, and
// [csrf_token, preferred_warehouse_id, preferred_warehouse_id] on another —
// the "email" field vanished because its element rendered a later sibling's
// captures. That wrapper shape is not exotic: it is the framework's own
// recommended fix for GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT, so the collision was
// reachable from following the framework's advice.
//
// WHY THIS WAS INVISIBLE IN TESTS. The collision is gated on Go INLINING, not
// on the render target (it reproduces identically under SSR, the native
// reconciler, and wasm). When the compiler inlines the wrapper at each call
// site, each inlined copy gets its OWN closure symbol — field.func1,
// field.func2, field.func3 — so the names differ and nothing collides. The
// moment the wrapper stops being inlinable (a body over the budget, which any
// real field helper is) or is called from a loop, every instance reports the
// same name and the last writer wins. That is why a three-sibling reproduction
// written inline in a test file passes while the production form corrupts:
// same framework code, different inlining decision.
//
// THE INVARIANT: a handle's implementation is never mutated on behalf of a
// different function value. Distinct function values get distinct handles that
// share a logical IdentityKey, so the reconciler still sees "same component"
// (fiber reuse and hook state are preserved — it compares IdentityKey strings,
// not handle pointers) while each element renders its own code. Deliberate
// swaps (ui.Typed, hot reload) still go through SetImplementationRenderer on a
// handle the caller owns; what is gone is the ACCIDENTAL swap between unrelated
// siblings.
func getComponentHandle(parseComponent any) *runtime.ComponentType {
	parsePrettyName, parseQualifiedName := describeComponentIdentity(parseComponent)
	parseIdentity := parseQualifiedName
	if strings.TrimSpace(parseIdentity) == "" {
		parseIdentity = parsePrettyName
	}
	if strings.TrimSpace(parseIdentity) == "" {
		parseIdentity = reflect.TypeOf(parseComponent).String()
	}

	if parseCached, parseOk := componentHandleCache.Load(parseIdentity); parseOk {
		handle := parseCached.(*runtime.ComponentType)
		// Steady-state fast path, and the reason the cache still exists: a
		// top-level component function (or a ui.Typed-registered one) is the
		// SAME function value at every CreateElement, so the cached handle
		// already holds the right implementation and the right renderer. Return
		// it untouched — no allocation, no renderer rebuild, no lock. This is
		// the overwhelmingly common case and it costs one map load.
		if handle.ImplementationMatches(parseComponent) {
			return handle
		}
		// A different function value under the same name: a fresh closure from
		// the same literal (a sibling, a loop iteration, a re-created inline
		// component) or a hot-reloaded body. It gets its OWN handle carrying
		// its OWN implementation, tagged with the same logical identity so the
		// reconciler keeps treating it as the same component.
		//
		// Do NOT "optimize" this into mutating `handle` — that is precisely the
		// cross-sibling corruption described above. Do NOT cache this handle
		// under a per-closure key either: closures are re-created per render,
		// so a per-closure cache is an unbounded leak. The handle lives and
		// dies with the element that owns it.
		return runtime.NewComponentType(parseIdentity, parsePrettyName, parseQualifiedName, parseComponent, buildComponentRenderer(parseComponent))
	}

	handle := runtime.NewComponentType(parseIdentity, parsePrettyName, parseQualifiedName, parseComponent, buildComponentRenderer(parseComponent))
	parseStored, isParseLoaded := componentHandleCache.LoadOrStore(parseIdentity, handle)
	if !isParseLoaded {
		return handle
	}
	// Lost the store race. Reuse the winner only when it holds this very
	// function value; otherwise keep our own handle rather than overwriting
	// theirs — the same non-mutation rule as the hit path, applied to a race
	// that used to silently clobber another goroutine's implementation.
	parseResolved := parseStored.(*runtime.ComponentType)
	if parseResolved.ImplementationMatches(parseComponent) {
		return parseResolved
	}
	return handle
}

// buildComponentRenderer prepares one reusable renderer closure for a component implementation signature.
func buildComponentRenderer(parseComponent any) func(any, map[string]any) *runtime.Element {
	if parseComponent == nil {
		return nil
	}

	switch parseComponent.(type) {
	case func() Node:
		return func(parseImplementation any, parseRawProps map[string]any) *runtime.Element {
			parseTypedImplementation, parseOk := parseImplementation.(func() Node)
			if !parseOk {
				return renderComponent(parseImplementation, parseRawProps)
			}
			return parseTypedImplementation()
		}
	case func(map[string]any) Node:
		return func(parseImplementation any, parseRawProps map[string]any) *runtime.Element {
			parseTypedImplementation, parseOk := parseImplementation.(func(map[string]any) Node)
			if !parseOk {
				return renderComponent(parseImplementation, parseRawProps)
			}
			return parseTypedImplementation(getComponentMapProps(parseRawProps))
		}
	case func(runtime.Attrs) Node:
		return func(parseImplementation any, parseRawProps map[string]any) *runtime.Element {
			parseTypedImplementation, parseOk := parseImplementation.(func(runtime.Attrs) Node)
			if !parseOk {
				return renderComponent(parseImplementation, parseRawProps)
			}
			return parseTypedImplementation(getComponentAttrsProps(parseRawProps))
		}
	}

	parseComponentType := reflect.TypeOf(parseComponent)
	if parseComponentType == nil || parseComponentType.Kind() != reflect.Func {
		return func(parseImplementation any, parseRawProps map[string]any) *runtime.Element {
			return renderComponent(parseImplementation, parseRawProps)
		}
	}
	if parseCached, parseOk := getComponentRenderCache.Load(parseComponentType); parseOk {
		return parseCached.(func(any, map[string]any) *runtime.Element)
	}

	parseMeta := getComponentMeta(parseComponentType)
	parseRenderer := func(parseImplementation any, parseRawProps map[string]any) *runtime.Element {
		parseImplementationValue := reflect.ValueOf(parseImplementation)
		if !parseImplementationValue.IsValid() || parseImplementationValue.Kind() != reflect.Func {
			panic(actionableCreateElementPanic("ui.CreateElement requires a component function or ui.Node"))
		}

		var parseResults []reflect.Value
		if parseMeta.hasArg {
			var parseArgBuf [1]reflect.Value
			parseArgBuf[0] = parseMeta.getArgValue(parseRawProps)
			parseResults = parseImplementationValue.Call(parseArgBuf[:])
		} else {
			parseResults = parseImplementationValue.Call(nil)
		}

		if len(parseResults) == 0 || !parseResults[0].IsValid() || parseResults[0].IsNil() {
			return nil
		}

		parseElement, _ := parseResults[0].Interface().(*runtime.Element)
		return parseElement
	}

	parseStored, _ := getComponentRenderCache.LoadOrStore(parseComponentType, parseRenderer)
	return parseStored.(func(any, map[string]any) *runtime.Element)
}

// describeComponentIdentity is a core package helper.
func describeComponentIdentity(parseComponent any) (string, string) {
	if parseComponent == nil {
		return "", ""
	}

	parseValue := reflect.ValueOf(parseComponent)
	if parseValue.IsValid() && parseValue.Kind() == reflect.Func {
		getCacheKey := componentIdentityCacheKey{
			getType:    parseValue.Type(),
			getPointer: parseValue.Pointer(),
		}
		if parseCached, hasParseCached := getComponentIdentityCache.Load(getCacheKey); hasParseCached {
			getCached := parseCached.(componentIdentityCacheValue)
			return getCached.getPrettyName, getCached.getQualifiedName
		}
		if parseFn := goRuntime.FuncForPC(parseValue.Pointer()); parseFn != nil {
			parseQualified := parseFn.Name()
			getIdentity := componentIdentityCacheValue{
				getPrettyName:    trimComponentName(parseQualified),
				getQualifiedName: parseQualified,
			}
			getComponentIdentityCache.Store(getCacheKey, getIdentity)
			return getIdentity.getPrettyName, getIdentity.getQualifiedName
		}
		parseQualified := fmt.Sprintf("%s@%x", reflect.TypeOf(parseComponent).String(), parseValue.Pointer())
		getIdentity := componentIdentityCacheValue{
			getPrettyName:    reflect.TypeOf(parseComponent).String(),
			getQualifiedName: parseQualified,
		}
		getComponentIdentityCache.Store(getCacheKey, getIdentity)
		return getIdentity.getPrettyName, getIdentity.getQualifiedName
	}

	parseRendered := reflect.TypeOf(parseComponent).String()
	return parseRendered, parseRendered
}

// trimComponentName is a core package helper.
func trimComponentName(parseName string) string {
	if parseIndex := strings.LastIndex(parseName, "/"); parseIndex >= 0 {
		parseName = parseName[parseIndex+1:]
	}
	if parseIndex2 := strings.LastIndex(parseName, "."); parseIndex2 >= 0 {
		parseName = parseName[parseIndex2+1:]
	}
	return parseName
}
