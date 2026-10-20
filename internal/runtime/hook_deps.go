package runtime

import (
	"reflect"
	"strconv"
	"sync"

	"github.com/monstercameron/GoWebComponents/v7/internal/errcode"
)

// alwaysDependency is the type behind the exported AlwaysDependency sentinel. It
// is a distinct unexported type so no user value can collide with it.
type alwaysDependency struct{}

// AlwaysDependency is the sentinel passed as the only dependency to make a
// hook's intent explicit: "run on every render". UseEffect(fn, ui.Always) is
// exactly UseEffect(fn) (no dependency list), UseMemo(fn, ui.Always) recomputes
// on every render, and UseCallback(fn, ui.Always) stores the latest function on
// every render. It is only recognized as the sole dependency; mixed into a
// longer list it is an ordinary constant value and changes nothing.
var AlwaysDependency any = alwaysDependency{}

// isAlwaysDeps reports whether a dependency list is exactly the Always sentinel.
func isAlwaysDeps(parseDeps []any) bool {
	if len(parseDeps) != 1 {
		return false
	}
	_, parseOk := parseDeps[0].(alwaysDependency)
	return parseOk
}

// describeUnstableDep explains why one dependency pair changes identity every
// render, or returns "" when the pair is fine. Funcs, maps and slices are
// compared by identity, never by contents: a func that changed identity is a
// fresh closure, and a map or slice with a new identity but deeply equal
// contents is a literal rebuilt on every render.
func describeUnstableDep(parsePrev, parseNext any) string {
	if parsePrev == nil || parseNext == nil || fastEqual(parsePrev, parseNext) {
		return ""
	}
	parseTypePrev := reflect.TypeOf(parsePrev)
	if parseTypePrev != reflect.TypeOf(parseNext) {
		return ""
	}
	switch parseTypePrev.Kind() {
	case reflect.Func:
		return "a func dependency (" + parseTypePrev.String() + ") is a new closure on every render"
	case reflect.Map, reflect.Slice:
		if reflect.DeepEqual(parsePrev, parseNext) {
			return "a " + parseTypePrev.Kind().String() + " dependency (" + parseTypePrev.String() + ") is rebuilt with equal contents on every render"
		}
	}
	return ""
}

// checkHookDepStability emits one GWC-RUNTIME-HOOK-DEP-UNSTABLE diagnostic per
// hook slot when a changed dependency list contains a func, map or slice whose
// identity changes while its meaning does not. Development builds only: the
// const gate erases the call in production wasm.
func checkHookDepStability(parseHooks *Hooks, parseKind string, parseSlot int, parsePrev, parseNext []any) {
	if !hookSignatureRecordingEnabled || parseHooks == nil || len(parsePrev) == 0 || isAlwaysDeps(parseNext) {
		return
	}
	for parseIndex := range parseNext {
		if parseIndex >= len(parsePrev) {
			break
		}
		parseDetail := describeUnstableDep(parsePrev[parseIndex], parseNext[parseIndex])
		if parseDetail == "" {
			continue
		}
		parseKey := parseKind + ":" + strconv.Itoa(parseSlot)
		if parseHooks.order == nil {
			parseHooks.order = &hookOrderTracker{}
		}
		if _, parseSeen := parseHooks.order.reportedDeps[parseKey]; parseSeen {
			return
		}
		if parseHooks.order.reportedDeps == nil {
			parseHooks.order.reportedDeps = map[string]struct{}{}
		}
		parseHooks.order.reportedDeps[parseKey] = struct{}{}
		parseFiber := parseHooks.owner
		ReportDiagnosticWithContext(
			"runtime",
			DiagnosticWarning,
			errcode.New(errcode.RuntimeHookDepUnstable, getHookKindLabel(parseKind)+" dependency "+strconv.Itoa(parseIndex)+": "+parseDetail).Error(),
			diagnosticPathWithSource(parseFiber),
			diagnosticComponentStackWithSource(parseFiber),
		)
		return
	}
}

// checkCurrentMemoDepStability runs the dependency check for the memo slot the
// prologue just claimed on the current fiber.
func checkCurrentMemoDepStability(parsePrev, parseNext []any) {
	if !hookSignatureRecordingEnabled {
		return
	}
	if parseFiber := GetCurrentFiber(); parseFiber != nil && parseFiber.hooks != nil {
		checkHookDepStability(parseFiber.hooks, "memo", parseFiber.hooks.memoIndex-1, parsePrev, parseNext)
	}
}

// listRenderState tracks list-render depth so handler hooks allocated while a
// list is being mapped can be flagged. Development builds only.
var listRenderState struct {
	mu    sync.Mutex
	depth int
	owner *Fiber
}

// noopListRenderRestore is the shared restore func returned when tracking is
// compiled out, so production callers allocate nothing.
var noopListRenderRestore = func() {}

// BeginListRender marks the start of a list render (html.Map / html.MapKeyed
// calling its per-item callback in a loop) and returns the func that ends it;
// call it as `defer runtime.BeginListRender()()`. While a list render is open on
// the current component, allocating a handler hook (UseEvent / UseFunc, which
// html.OnClick(func...) does implicitly) reports GWC-RUNTIME-HOOK-IN-LIST once
// per component. It is a no-op in production wasm.
func BeginListRender() func() {
	if !hookSignatureRecordingEnabled {
		return noopListRenderRestore
	}
	listRenderState.mu.Lock()
	if listRenderState.depth == 0 {
		listRenderState.owner = GetCurrentFiber()
	}
	listRenderState.depth++
	listRenderState.mu.Unlock()
	return func() {
		listRenderState.mu.Lock()
		if listRenderState.depth > 0 {
			listRenderState.depth--
		}
		if listRenderState.depth == 0 {
			listRenderState.owner = nil
		}
		listRenderState.mu.Unlock()
	}
}

// isListRenderActive reports whether a list render is open for the fiber.
func isListRenderActive(parseFiber *Fiber) bool {
	listRenderState.mu.Lock()
	defer listRenderState.mu.Unlock()
	return listRenderState.depth > 0 && (listRenderState.owner == nil || listRenderState.owner == parseFiber)
}

// checkHookInList reports GWC-RUNTIME-HOOK-IN-LIST once per component when a
// handler hook slot is allocated during a list render.
func checkHookInList(parseHooks *Hooks) {
	if !hookSignatureRecordingEnabled || parseHooks == nil || !isListRenderActive(parseHooks.owner) {
		return
	}
	if parseHooks.order == nil {
		parseHooks.order = &hookOrderTracker{}
	}
	if parseHooks.order.isListReported {
		return
	}
	parseHooks.order.isListReported = true
	parseFiber := parseHooks.owner
	ReportDiagnosticWithContext(
		"runtime",
		DiagnosticWarning,
		errcode.New(errcode.RuntimeHookInList, "a handler hook was allocated while mapping a list; per-item inline handlers take positional hook slots, so use MapKeyedComponent for rows that own handlers").Error(),
		diagnosticPathWithSource(parseFiber),
		diagnosticComponentStackWithSource(parseFiber),
	)
}
