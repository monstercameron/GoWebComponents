//go:build js && wasm
// +build js,wasm

package fiber

import "testing"

func TestResolveContextValueUsesDefaultWhenMissing(t *testing.T) {
	context := CreateContext("fallback")

	value := resolveContextValue(&Fiber{}, context)
	if value != "fallback" {
		t.Fatalf("expected default value fallback, got %v", value)
	}
}

func TestDeriveContextValuesCopiesAndOverrides(t *testing.T) {
	first := CreateContext("a")
	second := CreateContext("b")

	parent := map[int64]interface{}{
		first.id: "outer",
	}

	derived := deriveContextValues(parent, second.id, "inner")
	derivedOverride := deriveContextValues(derived, first.id, "override")

	if parent[first.id] != "outer" {
		t.Fatalf("expected parent value to remain outer, got %v", parent[first.id])
	}

	if derived[second.id] != "inner" {
		t.Fatalf("expected derived second context value inner, got %v", derived[second.id])
	}

	if derivedOverride[first.id] != "override" {
		t.Fatalf("expected override value, got %v", derivedOverride[first.id])
	}
}

func TestGoUseContextReadsCurrentFiberValue(t *testing.T) {
	context := CreateContext("fallback")
	previousFiber := wipFiber
	defer func() {
		wipFiber = previousFiber
	}()

	wipFiber = &Fiber{
		hooks: &Hooks{},
		contextValues: map[int64]interface{}{
			context.id: "provided",
		},
	}

	value := GoUseContext[string](context)
	if value != "provided" {
		t.Fatalf("expected provided value, got %s", value)
	}

	if wipFiber.hooks.index != 1 {
		t.Fatalf("expected hook index 1, got %d", wipFiber.hooks.index)
	}

	if len(wipFiber.hooks.callOrder) != 1 {
		t.Fatalf("expected one hook call recorded, got %d", len(wipFiber.hooks.callOrder))
	}

	if wipFiber.hooks.callOrder[0].Type != HookTypeContext {
		t.Fatalf("expected context hook type, got %v", wipFiber.hooks.callOrder[0].Type)
	}
}
