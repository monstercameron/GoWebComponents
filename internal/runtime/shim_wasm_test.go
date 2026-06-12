//go:build js && wasm

package runtime

import (
	"reflect"
	"testing"
)

func initWasmShimTestRuntime() *Fiber {
	resetGlobalRuntimeForTest()
	InitGlobalRuntime(Config{
		DOMAdapter: newTestDOMAdapter(),
		Scheduler:  newTestScheduler(),
	})
	parseFiber := &Fiber{typeOf: "test", props: map[string]interface{}{}}
	SetCurrentFiber(parseFiber)
	return parseFiber
}

func TestText_UsesOptimizedTextContentLayout(parseT *testing.T) {
	parseElem := Text("hello")
	if parseElem.Type != "TEXT_ELEMENT" {
		parseT.Fatalf("expected TEXT_ELEMENT, got %#v", parseElem.Type)
	}
	if parseElem.TextContent != "hello" {
		parseT.Fatalf("expected optimized text content, got %q", parseElem.TextContent)
	}
	if parseElem.Props != nil {
		parseT.Fatalf("expected text shim to avoid props map allocation, got %#v", parseElem.Props)
	}
	if len(parseElem.Children) != 0 {
		parseT.Fatalf("expected empty children for text element, got %d", len(parseElem.Children))
	}
}

func TestGoUseStateGlobal_DelegatesToGlobalRuntime(parseT *testing.T) {
	_ = initWasmShimTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	get, set := GoUseStateGlobal(1)
	if parseGot := get(); parseGot != 1 {
		parseT.Fatalf("expected initial global state 1, got %d", parseGot)
	}

	set(4)
	if parseGot2 := get(); parseGot2 != 4 {
		parseT.Fatalf("expected updated global state 4, got %d", parseGot2)
	}
}

func TestGoUseAtomGlobal_UsesTypedSetterAndSharedRegistry(parseT *testing.T) {
	_ = initWasmShimTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	get, set := GoUseAtomGlobal("shared", 2)
	if parseGot := get(); parseGot != 2 {
		parseT.Fatalf("expected initial atom value 2, got %d", parseGot)
	}

	set(7)
	if parseGot2 := get(); parseGot2 != 7 {
		parseT.Fatalf("expected updated atom value 7, got %d", parseGot2)
	}

	parseOtherFiber := &Fiber{typeOf: "other", props: map[string]interface{}{}}
	SetCurrentFiber(parseOtherFiber)
	parseOtherGet, _ := GoUseAtomGlobal("shared", 0)
	if parseGot3 := parseOtherGet(); parseGot3 != 7 {
		parseT.Fatalf("expected shared atom value 7 in second fiber, got %d", parseGot3)
	}
}

func TestGoUseMemoGlobalAndCallbackGlobal_Delegate(parseT *testing.T) {
	parseFiber := initWasmShimTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	parseComputes := 0
	parseValue1 := GoUseMemoGlobal(func() interface{} {
		parseComputes++
		return "memo"
	}, "dep")
	if parseValue1 != "memo" {
		parseT.Fatalf("expected memoized value, got %#v", parseValue1)
	}

	parseFiber.hooks.index = 0
	parseFiber.hooks.memoIndex = 0
	parseValue2 := GoUseMemoGlobal(func() interface{} {
		parseComputes++
		return "memo2"
	}, "dep")
	if parseValue2 != "memo" {
		parseT.Fatalf("expected memo reuse on same deps, got %#v", parseValue2)
	}
	if parseComputes != 1 {
		parseT.Fatalf("expected one compute on same deps, got %d", parseComputes)
	}

	parseTestFn := func() {}
	parseCallback1 := GoUseCallbackGlobal(parseTestFn, "cb")
	parseFiber.hooks.index = 0
	parseFiber.hooks.callbackIndex = 0
	parseCallback2 := GoUseCallbackGlobal(parseTestFn, "cb")
	if reflect.ValueOf(parseCallback1).Pointer() != reflect.ValueOf(parseCallback2).Pointer() {
		parseT.Fatalf("expected callback identity to remain stable for same deps")
	}
}

func TestGoUseRefGlobalAndIdGlobal_Delegate(parseT *testing.T) {
	parseFiber := initWasmShimTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	parseRef := GoUseRefGlobal("initial")
	if parseRef.Current != "initial" {
		parseT.Fatalf("expected initial ref value, got %#v", parseRef.Current)
	}
	parseRef.Current = "changed"

	parseId1 := GoUseIdGlobal()
	if parseId1 == "" {
		parseT.Fatal("expected non-empty id")
	}

	parseFiber.hooks.index = 0
	parseFiber.hooks.refIndex = 0
	parseFiber.hooks.idIndex = 0
	parseRefAgain := GoUseRefGlobal("ignored")
	parseId2 := GoUseIdGlobal()

	if parseRefAgain != parseRef || parseRefAgain.Current != "changed" {
		parseT.Fatalf("expected ref persistence across render, got %#v", parseRefAgain)
	}
	if parseId2 != parseId1 {
		parseT.Fatalf("expected stable id across render, got %q and %q", parseId1, parseId2)
	}
}

func TestStartTransitionGlobalAndPendingAtom(parseT *testing.T) {
	_ = initWasmShimTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	parseStateGet, parseStateSet := GoUseStateGlobal(1)
	parsePendingGet, _ := GoUseTransitionPendingGlobal()
	StartTransitionGlobal(func() {
		parseStateSet(9)
	})

	if parseGot := parseStateGet(); parseGot != 1 {
		parseT.Fatalf("expected transition update to remain deferred before scheduler flush, got %d", parseGot)
	}
	if !parsePendingGet() {
		parseT.Fatal("expected pending transition atom to report true before scheduler flush")
	}

	parseRt := GetGlobalRuntime()
	parseScheduler, parseOk := parseRt.scheduler.(*testScheduler)
	if !parseOk || len(parseScheduler.timeouts) == 0 {
		parseT.Fatal("expected global test scheduler to capture transition timeout")
	}
	parseScheduler.timeouts[0]()

	if parseGot2 := parseStateGet(); parseGot2 != 9 {
		parseT.Fatalf("expected deferred global state update after scheduler flush, got %d", parseGot2)
	}
	if parsePendingGet() {
		parseT.Fatal("expected pending transition atom to clear after scheduler flush")
	}
}
