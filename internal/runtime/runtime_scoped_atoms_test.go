package runtime

import "testing"

// v5 P2.6 — atoms resolve to the runtime that owns the rendering component.
//
// Hard dependency of P3.7: projections are published into the atom registry.
// If that registry is reached through GetGlobalRuntime, every projection lands
// in whichever runtime happens to be global, and the whole projection
// architecture silently inherits the single-runtime assumption P2.3 removes —
// after P2.3's own tests have gone green, which is the worst time to find out.

// TestResolveRuntime_FallsBackToGlobalOutsideRender pins the non-render path.
func TestResolveRuntime_FallsBackToGlobalOutsideRender(parseT *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	SetCurrentFiber(nil)
	if ResolveRuntime() != GetGlobalRuntime() {
		parseT.Error("outside a render, atom access must resolve to the global runtime")
	}
}

// TestResolveRuntime_UsesOwningRuntimeDuringRender is the property P3.7 needs.
func TestResolveRuntime_UsesOwningRuntimeDuringRender(parseT *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()
	defer SetCurrentFiber(nil)

	parseOwned := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Reset: false})
	parseFiber := &Fiber{typeOf: "component", ownerRuntime: parseOwned}
	SetCurrentFiber(parseFiber)

	if ResolveRuntime() != parseOwned {
		parseT.Error("during a render, atom access must resolve to the runtime owning the tree")
	}
	if ResolveRuntime() == GetGlobalRuntime() {
		parseT.Error("an owned fiber must not resolve to the global runtime")
	}
}

// TestResolveRuntime_WalksToAnOwningAncestor covers fibers that carry no owner
// of their own, which is the common case below the root.
func TestResolveRuntime_WalksToAnOwningAncestor(parseT *testing.T) {
	defer SetCurrentFiber(nil)

	parseOwned := NewRuntime(Config{DOMAdapter: newTestDOMAdapter()})
	parseRoot := &Fiber{typeOf: "ROOT", ownerRuntime: parseOwned}
	parseChild := &Fiber{typeOf: "component", parent: parseRoot}
	parseLeaf := &Fiber{typeOf: "leaf", parent: parseChild}
	SetCurrentFiber(parseLeaf)

	if ResolveRuntime() != parseOwned {
		parseT.Error("resolution must walk up to the nearest owning ancestor")
	}
}

// TestRuntimeScopedAtoms_TwoRuntimesStayIsolated is the accept criterion:
// a value published in one runtime is invisible to the other.
func TestRuntimeScopedAtoms_TwoRuntimesStayIsolated(parseT *testing.T) {
	defer SetCurrentFiber(nil)

	parseFirst := NewRuntime(Config{DOMAdapter: newTestDOMAdapter()})
	parseSecond := NewRuntime(Config{DOMAdapter: newTestDOMAdapter()})

	parseFirstFiber := &Fiber{typeOf: "component", ownerRuntime: parseFirst}
	parseSecondFiber := &Fiber{typeOf: "component", ownerRuntime: parseSecond}

	SetCurrentFiber(parseFirstFiber)
	if parseErr := ResolveRuntime().SetAtomValue("shared", "from-first"); parseErr != nil {
		parseT.Fatalf("set on first runtime: %v", parseErr)
	}

	SetCurrentFiber(parseSecondFiber)
	if parseErr := ResolveRuntime().SetAtomValue("shared", "from-second"); parseErr != nil {
		parseT.Fatalf("set on second runtime: %v", parseErr)
	}

	SetCurrentFiber(parseFirstFiber)
	if parseValue, _ := ResolveRuntime().GetAtomValue("shared"); parseValue != "from-first" {
		parseT.Errorf("first runtime saw %#v; the second runtime's write must not leak across", parseValue)
	}

	SetCurrentFiber(parseSecondFiber)
	if parseValue, _ := ResolveRuntime().GetAtomValue("shared"); parseValue != "from-second" {
		parseT.Errorf("second runtime saw %#v, want its own value", parseValue)
	}
}
