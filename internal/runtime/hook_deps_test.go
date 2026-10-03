package runtime

import (
	"strings"
	"testing"
)

// getHookDiagnosticsByCode filters the recorded diagnostics by registered code.
func getHookDiagnosticsByCode(parseCode string) []Diagnostic {
	var parseOut []Diagnostic
	for _, parseDiagnostic := range GetDiagnostics() {
		if parseDiagnostic.Code == parseCode {
			parseOut = append(parseOut, parseDiagnostic)
		}
	}
	return parseOut
}

// renderHookDepsPasses renders a component the way renderHookOrderPasses does,
// on a runtime that has a DOM adapter so handler hooks can allocate.
func renderHookDepsPasses(parseT *testing.T, parseComponent func() *Element, parsePasses int) {
	parseT.Helper()
	parseRt := &Runtime{domAdapter: newTestDOMAdapter()}
	parseRoot := &Fiber{typeOf: "ROOT"}
	var parsePrevious *Fiber
	for parsePass := 0; parsePass < parsePasses; parsePass++ {
		parseFiber := &Fiber{typeOf: parseComponent, parent: parseRoot, alternate: parsePrevious}
		if _, parseHandled, _ := parseRt.renderFunctionComponent(parseFiber); parseHandled {
			parseT.Fatalf("pass %d: render was handled as a panic", parsePass)
		}
		parsePrevious = parseFiber
	}
}

var hookDepsStableMap = map[string]int{"a": 1}

func hookDepsFreshFuncEffect() *Element {
	parseCounter := 0
	GoUseEffect(func() func() { return nil }, func() { parseCounter++ })
	return CreateElement("div", nil, "ok")
}

func hookDepsFreshMapMemo() *Element {
	GoUseMemo(func() any { return 1 }, map[string]int{"a": 1})
	return CreateElement("div", nil, "ok")
}

func hookDepsFreshSliceCallback() *Element {
	GoUseCallback(func() {}, []string{"a", "b"})
	return CreateElement("div", nil, "ok")
}

func hookDepsStableDeps() *Element {
	GoUseEffect(func() func() { return nil }, hookDepsStableMap, 1, "x")
	GoUseMemo(func() any { return 1 }, hookDepsStableMap)
	return CreateElement("div", nil, "ok")
}

func hookDepsChangingSlice() *Element {
	GoUseMemoFor(func() int { return 1 }, []int{hookDepsRenderCount})
	return CreateElement("div", nil, "ok")
}

var hookDepsRenderCount int

func TestHookDepUnstableReportsFuncMapSliceOncePerSlot(parseT *testing.T) {
	if !hookSignatureRecordingEnabled {
		parseT.Skip("hook signature recording is compiled out")
	}
	for _, parseCase := range []struct {
		name      string
		component func() *Element
		want      string
	}{
		{"func effect", hookDepsFreshFuncEffect, "UseEffect dependency 0: a func dependency"},
		{"map memo", hookDepsFreshMapMemo, "UseMemo dependency 0: a map dependency"},
		{"slice callback", hookDepsFreshSliceCallback, "UseCallback dependency 0: a slice dependency"},
	} {
		ClearDiagnostics()
		renderHookOrderPasses(parseT, parseCase.component, 5, nil)
		parseDiagnostics := getHookDiagnosticsByCode("GWC-RUNTIME-HOOK-DEP-UNSTABLE")
		ClearDiagnostics()
		if len(parseDiagnostics) != 1 {
			parseT.Fatalf("%s: expected exactly one diagnostic over 5 renders, got %d: %+v", parseCase.name, len(parseDiagnostics), parseDiagnostics)
		}
		if !strings.Contains(parseDiagnostics[0].Message, parseCase.want) {
			parseT.Fatalf("%s: expected %q in %q", parseCase.name, parseCase.want, parseDiagnostics[0].Message)
		}
		if parseDiagnostics[0].Severity != DiagnosticWarning {
			parseT.Fatalf("%s: expected warning severity, got %v", parseCase.name, parseDiagnostics[0].Severity)
		}
	}
}

func TestHookDepUnstableIgnoresStableAndGenuinelyChangingDeps(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()
	renderHookOrderPasses(parseT, hookDepsStableDeps, 5, nil)
	defer func() { hookDepsRenderCount = 0 }()
	renderHookOrderPasses(parseT, hookDepsChangingSlice, 5, func(parsePass int) { hookDepsRenderCount = parsePass })
	if parseDiagnostics := getHookDiagnosticsByCode("GWC-RUNTIME-HOOK-DEP-UNSTABLE"); len(parseDiagnostics) != 0 {
		parseT.Fatalf("stable deps and genuinely changing slices must not report, got %+v", parseDiagnostics)
	}
}

var hookDepsAlwaysEffectRuns, hookDepsAlwaysMemoRuns int

func hookDepsAlwaysComponent() *Element {
	GoUseEffect(func() func() { hookDepsAlwaysEffectRuns++; return nil }, AlwaysDependency)
	GoUseMemo(func() any { hookDepsAlwaysMemoRuns++; return 1 }, AlwaysDependency)
	return CreateElement("div", nil, "ok")
}

func TestAlwaysDependencyRunsEveryRenderWithoutDiagnostics(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()
	hookDepsAlwaysMemoRuns = 0
	renderHookOrderPasses(parseT, hookDepsAlwaysComponent, 4, nil)
	if hookDepsAlwaysMemoRuns != 4 {
		parseT.Fatalf("Always memo must recompute every render, ran %d times", hookDepsAlwaysMemoRuns)
	}
	if parseDiagnostics := getHookDiagnosticsByCode("GWC-RUNTIME-HOOK-DEP-UNSTABLE"); len(parseDiagnostics) != 0 {
		parseT.Fatalf("Always must not report, got %+v", parseDiagnostics)
	}
	if !isAlwaysDeps([]any{AlwaysDependency}) || isAlwaysDeps([]any{AlwaysDependency, 1}) || isAlwaysDeps(nil) || isAlwaysDeps([]any{"always"}) {
		parseT.Fatal("Always is only recognized as the sole dependency")
	}
}

func TestAlwaysDependencyQueuesEffectEveryRender(parseT *testing.T) {
	parseHooks := &Hooks{}
	parseFiber := &Fiber{typeOf: "test", props: map[string]any{}, hooks: parseHooks}
	parseHooks.owner = parseFiber
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)
	for parsePass := 0; parsePass < 3; parsePass++ {
		parseHooks.depIndex, parseHooks.cleanupIndex, parseHooks.index = 0, 0, 0
		parseFiber.effects = nil
		GoUseEffect(func() func() { return nil }, AlwaysDependency)
		if len(parseFiber.effects) != 1 {
			parseT.Fatalf("pass %d: Always effect must be queued, got %d effects", parsePass, len(parseFiber.effects))
		}
	}
}

func TestListRenderFlagsHandlerHookOncePerComponent(parseT *testing.T) {
	if !hookSignatureRecordingEnabled {
		parseT.Skip("hook signature recording is compiled out")
	}
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseHandlers := func() *Element {
		GoUseFunc(func() {}) // outside a list: fine
		parseRestore := BeginListRender()
		for parseI := 0; parseI < 3; parseI++ {
			GoUseFunc(func() {})
		}
		parseRestore()
		return CreateElement("div", nil, "ok")
	}
	renderHookDepsPasses(parseT, parseHandlers, 1)

	parseDiagnostics := getHookDiagnosticsByCode("GWC-RUNTIME-HOOK-IN-LIST")
	if len(parseDiagnostics) != 1 {
		parseT.Fatalf("expected exactly one in-list diagnostic, got %d: %+v", len(parseDiagnostics), parseDiagnostics)
	}
	if !strings.Contains(parseDiagnostics[0].Message, "MapKeyedComponent") {
		parseT.Fatalf("remediation must name MapKeyedComponent, got %q", parseDiagnostics[0].Message)
	}
}

func TestListRenderRestoreEndsTrackingAndNests(parseT *testing.T) {
	parseOuter := BeginListRender()
	parseInner := BeginListRender()
	parseInner()
	if !isListRenderActive(nil) {
		parseT.Fatal("outer list render must stay active after the inner one ends")
	}
	parseOuter()
	if isListRenderActive(nil) {
		parseT.Fatal("list render must end after the outermost restore")
	}
}

func TestHandlerHookOutsideListDoesNotReport(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()
	renderHookDepsPasses(parseT, func() *Element {
		GoUseFunc(func() {})
		return CreateElement("div", nil, "ok")
	}, 2)
	if parseDiagnostics := getHookDiagnosticsByCode("GWC-RUNTIME-HOOK-IN-LIST"); len(parseDiagnostics) != 0 {
		parseT.Fatalf("handler outside a list must not report, got %+v", parseDiagnostics)
	}
}
