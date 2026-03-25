package runtime

import (
	"strings"
	"testing"
)

type releasableWrapper struct {
	released *int
}

func (parseW *releasableWrapper) Release() {
	*parseW.released++
}

type funcWrapTestAdapter struct {
	*testDOMAdapter
	releasedCount *int
}

func (parseA *funcWrapTestAdapter) WrapFunction(parseFn interface{}) interface{} {
	return &releasableWrapper{released: parseA.releasedCount}
}

func TestIsNilableType(parseT *testing.T) {
	if isNilableType[int]() {
		parseT.Fatal("expected int to be non-nilable")
	}
	if !isNilableType[*int]() {
		parseT.Fatal("expected pointer type to be nilable")
	}
	if !isNilableType[[]string]() {
		parseT.Fatal("expected slice type to be nilable")
	}
}

func TestGoUseFunc_ReleasesOldWrapperOnRerender(parseT *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	parseReleased := 0
	parseAdapter := &funcWrapTestAdapter{
		testDOMAdapter: newTestDOMAdapter(),
		releasedCount:  &parseReleased,
	}
	InitGlobalRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})

	parseFiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseFirst := GoUseFunc(func() {})
	if parseFirst == nil {
		parseT.Fatal("expected wrapped function on first render")
	}

	parseFiber.hooks.index = 0
	parseFiber.hooks.funcIndex = 0

	parseSecond := GoUseFunc(func() {})
	if parseSecond == nil {
		parseT.Fatal("expected wrapped function on second render")
	}
	if parseReleased != 1 {
		parseT.Fatalf("expected old wrapper to be released once, got %d", parseReleased)
	}
}

func TestGoUseFunc_PanicsWithoutComponentContext(parseT *testing.T) {
	SetCurrentFiber(nil)
	defer func() {
		parseRecovered := recover()
		if parseRecovered == nil {
			parseT.Fatal("expected GoUseFunc to panic outside component context")
		}
		parseMessage := parseRecovered.(string)
		if !strings.Contains(parseMessage, "GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT") || !strings.Contains(parseMessage, "where:") || !strings.Contains(parseMessage, "runtime:") || !strings.Contains(parseMessage, "next:") {
			parseT.Fatalf("expected unified hook misuse panic output, got %q", parseMessage)
		}
	}()
	GoUseFunc(func() {})
}

func TestGoUseFunc_PanicsForNonFunction(parseT *testing.T) {
	parseFiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	defer func() {
		parseRecovered := recover()
		if parseRecovered == nil {
			parseT.Fatal("expected GoUseFunc to panic for non-function input")
		}
		parseMessage := parseRecovered.(string)
		if !strings.Contains(parseMessage, "GWC-RUNTIME-HOOK-FUNC-TYPE") || !strings.Contains(parseMessage, "path: GoUseFunc") || !strings.Contains(parseMessage, "docs: ACTIONABLE_ERRORS.md#gwc-runtime-hook-func-type") {
			parseT.Fatalf("expected unified GoUseFunc type panic output, got %q", parseMessage)
		}
	}()
	GoUseFunc(123)
}

func TestGoUseState_SchedulesUpdateForLatestHookOwner(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{
		scheduler: parseScheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  map[string]interface{}{},
		},
	}

	parseOldRoot := &Fiber{typeOf: "ROOT", props: map[string]interface{}{}}
	parseOldFiber := &Fiber{typeOf: "counter", parent: parseOldRoot, props: map[string]interface{}{}}
	SetCurrentFiber(parseOldFiber)
	_, setValue := GoUseState(parseRt, 0)
	SetCurrentFiber(nil)

	parseCurrentRoot := &Fiber{typeOf: "ROOT", props: map[string]interface{}{}}
	parseCurrentFiber := &Fiber{
		typeOf:    "counter",
		parent:    parseCurrentRoot,
		props:     map[string]interface{}{},
		hooks:     parseOldFiber.hooks,
		alternate: parseOldFiber,
	}
	parseCurrentFiber.hooks.owner = parseCurrentFiber
	parseRt.currentRoot = parseCurrentRoot

	setValue(1)

	if !parseCurrentFiber.dirty || !parseCurrentFiber.needsUpdate {
		parseT.Fatal("expected current hook owner to be marked dirty")
	}
	if !parseCurrentRoot.dirty || !parseCurrentRoot.needsUpdate {
		parseT.Fatal("expected current root path to be marked dirty")
	}
	if parseOldRoot.dirty || parseOldRoot.needsUpdate {
		parseT.Fatal("did not expect stale fiber ancestry to receive the update")
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected one scheduled timeout, got %d", len(parseScheduler.timeouts))
	}
}

func TestGoUseId_PanicsWithoutComponentContext(parseT *testing.T) {
	SetCurrentFiber(nil)
	defer func() {
		if recover() == nil {
			parseT.Fatal("expected GoUseId to panic outside component context")
		}
	}()
	GoUseId()
}

func TestAreDepsEqual_LongSlicesAndMismatch(parseT *testing.T) {
	parsePrev := []interface{}{1, 2, 3, 4, 5}
	parseNext := []interface{}{1, 2, 3, 4, 5}
	parseDiff := []interface{}{1, 2, 3, 4, 6}

	if !areDepsEqual(parsePrev, parseNext) {
		parseT.Fatal("expected long equal deps to compare true")
	}
	if areDepsEqual(parsePrev, parseDiff) {
		parseT.Fatal("expected long mismatched deps to compare false")
	}
}

func TestFastEqual_FunctionsSlicesMapsAndStructs(parseT *testing.T) {
	parseFn := func() {}
	if !fastEqual(parseFn, parseFn) {
		parseT.Fatal("expected identical functions to compare equal")
	}
	if fastEqual(func() {}, func() {}) {
		parseT.Fatal("expected distinct functions to compare different")
	}

	parseSlice := []int{1, 2, 3}
	if !fastEqual(parseSlice, parseSlice) {
		parseT.Fatal("expected identical slice reference to compare equal")
	}
	if fastEqual([]int{1, 2}, []int{1, 2}) {
		parseT.Fatal("expected different slice references to compare different")
	}

	parseM := map[string]int{"a": 1}
	if !fastEqual(parseM, parseM) {
		parseT.Fatal("expected identical map reference to compare equal")
	}

	type pair struct {
		A int
		B string
	}
	if !fastEqual(pair{A: 1, B: "x"}, pair{A: 1, B: "x"}) {
		parseT.Fatal("expected equal structs to compare equal")
	}
	if fastEqual(pair{A: 1, B: "x"}, pair{A: 2, B: "x"}) {
		parseT.Fatal("expected different structs to compare false")
	}
	if fastEqual(1, int64(1)) {
		parseT.Fatal("expected different numeric types to compare false")
	}

	type interfaceWrapper struct {
		Value interface{}
	}
	parseSharedSlice := []int{1, 2, 3}
	if !fastEqual(interfaceWrapper{Value: parseSharedSlice}, interfaceWrapper{Value: parseSharedSlice}) {
		parseT.Fatal("expected interface-wrapped shared slice to compare equal without panicking")
	}
	if !fastEqual(interfaceWrapper{Value: []int{1, 2}}, interfaceWrapper{Value: []int{1, 2}}) {
		parseT.Fatal("expected interface-wrapped equal slices to fall back to deep equality")
	}
	if fastEqual(interfaceWrapper{Value: []int{1, 2}}, interfaceWrapper{Value: []int{1, 3}}) {
		parseT.Fatal("expected interface-wrapped unequal slices to compare false")
	}
}
