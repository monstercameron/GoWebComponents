//go:build js && wasm
// +build js,wasm

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
	fiber := &Fiber{typeOf: "test", props: map[string]interface{}{}}
	SetCurrentFiber(fiber)
	return fiber
}

func TestText_UsesOptimizedTextContentLayout(t *testing.T) {
	elem := Text("hello")
	if elem.Type != "TEXT_ELEMENT" {
		t.Fatalf("expected TEXT_ELEMENT, got %#v", elem.Type)
	}
	if elem.TextContent != "hello" {
		t.Fatalf("expected optimized text content, got %q", elem.TextContent)
	}
	if elem.Props != nil {
		t.Fatalf("expected text shim to avoid props map allocation, got %#v", elem.Props)
	}
	if len(elem.Children) != 0 {
		t.Fatalf("expected empty children for text element, got %d", len(elem.Children))
	}
}

func TestGoUseStateGlobal_DelegatesToGlobalRuntime(t *testing.T) {
	_ = initWasmShimTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	get, set := GoUseStateGlobal(1)
	if got := get(); got != 1 {
		t.Fatalf("expected initial global state 1, got %d", got)
	}

	set(4)
	if got := get(); got != 4 {
		t.Fatalf("expected updated global state 4, got %d", got)
	}
}

func TestGoUseAtomGlobal_UsesTypedSetterAndSharedRegistry(t *testing.T) {
	_ = initWasmShimTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	get, set := GoUseAtomGlobal("shared", 2)
	if got := get(); got != 2 {
		t.Fatalf("expected initial atom value 2, got %d", got)
	}

	set(7)
	if got := get(); got != 7 {
		t.Fatalf("expected updated atom value 7, got %d", got)
	}

	otherFiber := &Fiber{typeOf: "other", props: map[string]interface{}{}}
	SetCurrentFiber(otherFiber)
	otherGet, _ := GoUseAtomGlobal("shared", 0)
	if got := otherGet(); got != 7 {
		t.Fatalf("expected shared atom value 7 in second fiber, got %d", got)
	}
}

func TestGoUseMemoGlobalAndCallbackGlobal_Delegate(t *testing.T) {
	fiber := initWasmShimTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	computes := 0
	value1 := GoUseMemoGlobal(func() interface{} {
		computes++
		return "memo"
	}, "dep")
	if value1 != "memo" {
		t.Fatalf("expected memoized value, got %#v", value1)
	}

	fiber.hooks.index = 0
	fiber.hooks.memoIndex = 0
	value2 := GoUseMemoGlobal(func() interface{} {
		computes++
		return "memo2"
	}, "dep")
	if value2 != "memo" {
		t.Fatalf("expected memo reuse on same deps, got %#v", value2)
	}
	if computes != 1 {
		t.Fatalf("expected one compute on same deps, got %d", computes)
	}

	testFn := func() {}
	callback1 := GoUseCallbackGlobal(testFn, "cb")
	fiber.hooks.index = 0
	fiber.hooks.callbackIndex = 0
	callback2 := GoUseCallbackGlobal(testFn, "cb")
	if reflect.ValueOf(callback1).Pointer() != reflect.ValueOf(callback2).Pointer() {
		t.Fatalf("expected callback identity to remain stable for same deps")
	}
}

func TestGoUseRefGlobalAndIdGlobal_Delegate(t *testing.T) {
	fiber := initWasmShimTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	ref := GoUseRefGlobal("initial")
	if ref.Current != "initial" {
		t.Fatalf("expected initial ref value, got %#v", ref.Current)
	}
	ref.Current = "changed"

	id1 := GoUseIdGlobal()
	if id1 == "" {
		t.Fatal("expected non-empty id")
	}

	fiber.hooks.index = 0
	fiber.hooks.refIndex = 0
	fiber.hooks.idIndex = 0
	refAgain := GoUseRefGlobal("ignored")
	id2 := GoUseIdGlobal()

	if refAgain != ref || refAgain.Current != "changed" {
		t.Fatalf("expected ref persistence across render, got %#v", refAgain)
	}
	if id2 != id1 {
		t.Fatalf("expected stable id across render, got %q and %q", id1, id2)
	}
}
