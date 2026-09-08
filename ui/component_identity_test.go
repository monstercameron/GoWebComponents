// This file deliberately carries NO build tag so it runs on BOTH targets.
//
// The defect it guards (see ui/component_handle_shared.go, getComponentHandle)
// corrupts rendered props identically under SSR, the native reconciler and
// wasm — but it was reported from a browser and initially misdiagnosed as
// browser-only, because the reproduction that "passed natively" happened to be
// written in a shape the Go compiler inlines. A tagged native-only test would
// re-create exactly that blind spot, so these cases must build and run under
// GOOS=js GOARCH=wasm too (the WASM Tests workflow enumerates every package
// that ships a *_wasm_test.go and runs `go test` over the whole package, so an
// untagged file in ui/ is gated by both lanes).

package ui_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// identityField is the shape the framework recommends for fixing
// GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT: a helper that calls hooks, wrapped in
// ui.CreateElement so the runtime invokes it inside a fiber. Every call creates
// a NEW closure from the SAME func literal, each capturing a different name.
//
// //go:noinline is the whole point of this test, not a micro-optimization
// artifact. Inlining gives each CALL SITE its own closure symbol
// (identityField.func1/func2/func3), which accidentally hides a
// name-keyed-identity collision. Any real-world field helper has a body well
// over the inlining budget, so production sees the un-inlined shape: one
// symbol, one qualified name, N closures. Removing this pragma does not make
// the test stricter — it makes it stop testing anything.
//
//go:noinline
func identityField(parseRt *runtime.Runtime, parseName string) ui.Node {
	return ui.CreateElement(func() ui.Node {
		// A hook — the reason the ui.CreateElement wrapper exists at all.
		_ = ui.UseId()
		_, _ = runtime.GoUseState(parseRt, parseName)
		return html.Input(html.Props{Type: "text", Name: parseName})
	})
}

// identitySSRField is identityField without a runtime handle, for the SSR lane.
//
//go:noinline
func identitySSRField(parseName string) ui.Node {
	return ui.CreateElement(func() ui.Node {
		_ = ui.UseId()
		return html.Input(html.Props{Type: "text", Name: parseName})
	})
}

type identityHarness struct {
	adapter *mockdom.MockDOMAdapter
	rt      *runtime.Runtime
	root    runtime.DOMNode
}

// newIdentityHarness mounts through the real reconciler against a mock DOM.
// With no scheduler configured, state updates flush synchronously.
func newIdentityHarness() *identityHarness {
	parseAdapter := mockdom.NewMockDOMAdapter()
	return &identityHarness{
		adapter: parseAdapter,
		rt:      runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true}),
		root:    parseAdapter.CreateElement("div"),
	}
}

// names returns the `name` attribute of every committed element, in document
// order — the exact observable that was corrupted in the field report
// ([csrf_token, preferred_warehouse_id, preferred_warehouse_id]).
func (parseH *identityHarness) names() []string {
	var parseOut []string
	var parseWalk func(parseNode *mockdom.MockDOMNode)
	parseWalk = func(parseNode *mockdom.MockDOMNode) {
		if parseNode == nil {
			return
		}
		if parseName, parseOk := parseNode.Attrs["name"]; parseOk {
			parseOut = append(parseOut, parseName)
		}
		for _, parseChild := range parseNode.Children {
			parseWalk(parseChild)
		}
	}
	for _, parseKid := range parseH.adapter.GetChildren(parseH.root) {
		if parseNode, parseOk := parseKid.(*mockdom.MockDOMNode); parseOk {
			parseWalk(parseNode)
		}
	}
	return parseOut
}

// allText concatenates every committed text node, in document order.
func (parseH *identityHarness) allText() string {
	var parseWalk func(parseNode *mockdom.MockDOMNode) string
	parseWalk = func(parseNode *mockdom.MockDOMNode) string {
		if parseNode == nil {
			return ""
		}
		parseOut := parseNode.TextContent
		for _, parseChild := range parseNode.Children {
			parseOut += parseWalk(parseChild)
		}
		return parseOut
	}
	parseOut := ""
	for _, parseKid := range parseH.adapter.GetChildren(parseH.root) {
		if parseNode, parseOk := parseKid.(*mockdom.MockDOMNode); parseOk {
			parseOut += parseWalk(parseNode)
		}
	}
	return parseOut
}

func expectNames(parseT *testing.T, parseLabel string, parseGot []string, parseWant ...string) {
	parseT.Helper()
	if strings.Join(parseGot, ",") == strings.Join(parseWant, ",") {
		return
	}
	parseT.Errorf("%s: sibling components built from one func literal rendered each other's captured props\n got: %v\nwant: %v",
		parseLabel, parseGot, parseWant)
}

// TestComponentIdentitySiblingClosuresKeepTheirOwnPropsOnTheReconciler is the
// primary regression test. Before the fix this committed
// [quantity quantity quantity]: three elements sharing one name-keyed component
// handle whose implementation had been overwritten by the last closure created.
func TestComponentIdentitySiblingClosuresKeepTheirOwnPropsOnTheReconciler(parseT *testing.T) {
	parseH := newIdentityHarness()
	parseH.rt.RenderInto(parseH.root, html.Form(html.Props{},
		identityField(parseH.rt, "email"),
		identityField(parseH.rt, "warehouse"),
		identityField(parseH.rt, "quantity"),
	))
	expectNames(parseT, "reconciler mount", parseH.names(), "email", "warehouse", "quantity")
}

// TestComponentIdentitySiblingClosuresKeepTheirOwnPropsOnSSR proves the same
// corruption on the server path. It is here to kill the "browser-only" theory:
// the mechanism is a shared component handle, not a renderer, so both targets
// were affected and both are now covered.
func TestComponentIdentitySiblingClosuresKeepTheirOwnPropsOnSSR(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(html.Form(html.Props{},
		identitySSRField("email"),
		identitySSRField("warehouse"),
		identitySSRField("quantity"),
	))
	if parseErr != nil {
		parseT.Fatalf("RenderToString: %v", parseErr)
	}
	for _, parseWant := range []string{`name="email"`, `name="warehouse"`, `name="quantity"`} {
		if !strings.Contains(parseMarkup, parseWant) {
			parseT.Errorf("SSR markup lost %s (shared component handle): %s", parseWant, parseMarkup)
		}
	}
	if parseCount := strings.Count(parseMarkup, `name="quantity"`); parseCount != 1 {
		parseT.Errorf("SSR rendered the last sibling %d times instead of once: %s", parseCount, parseMarkup)
	}
}

// TestComponentIdentityLoopBuiltClosuresKeepTheirOwnProps covers the shape that
// is immune to the inlining accident: one call site in a loop is one closure
// symbol no matter what the compiler decides, so every iteration reported the
// same qualified name. This is the shape the corrupted Atlas forms had.
func TestComponentIdentityLoopBuiltClosuresKeepTheirOwnProps(parseT *testing.T) {
	parseWant := []string{"csrf_token", "email", "preferred_warehouse_id", "quantity", "subject"}
	parseH := newIdentityHarness()

	parseChildren := make([]ui.Node, 0, len(parseWant))
	for _, parseName := range parseWant {
		parseChildren = append(parseChildren, identityField(parseH.rt, parseName))
	}
	parseH.rt.RenderInto(parseH.root, html.Form(html.Props{}, parseChildren...))

	expectNames(parseT, "loop-built fields", parseH.names(), parseWant...)
}

// TestComponentIdentityDistinctClosuresShareOneLogicalIdentity pins the two
// halves of the invariant at once: distinct closures from one literal get
// distinct handles (so each renders its own code) that report the SAME
// IdentityKey (so the reconciler keeps treating them as one component and
// preserves fiber/hook state across renders).
//
// If someone "simplifies" identity to the func value alone, IdentityKey starts
// differing between renders and every re-created inline component remounts,
// losing its state. If someone collapses it back to the name alone, the
// handles become one object and the props corrupt again. Both halves must hold.
func TestComponentIdentityDistinctClosuresShareOneLogicalIdentity(parseT *testing.T) {
	parseH := newIdentityHarness()
	parseFirst := identityField(parseH.rt, "email")
	parseSecond := identityField(parseH.rt, "warehouse")

	parseFirstHandle, parseOk := parseFirst.Type.(*runtime.ComponentType)
	if !parseOk {
		parseT.Fatalf("expected a component handle, got %T", parseFirst.Type)
	}
	parseSecondHandle, parseOk2 := parseSecond.Type.(*runtime.ComponentType)
	if !parseOk2 {
		parseT.Fatalf("expected a component handle, got %T", parseSecond.Type)
	}

	if parseFirstHandle == parseSecondHandle {
		parseT.Error("two closures from one func literal share one component handle; whichever registered last wins and the other element renders its props")
	}
	if parseFirstHandle.IdentityKey() != parseSecondHandle.IdentityKey() {
		parseT.Errorf("closures from one literal must stay ONE logical component for reconciliation: %q vs %q",
			parseFirstHandle.IdentityKey(), parseSecondHandle.IdentityKey())
	}
}

// identityTopLevel is a package-level component function: one func value for
// the whole process, no captures.
func identityTopLevel(parseAttrs runtime.Attrs) ui.Node {
	return html.Span(html.Props{Text: "top-level"})
}

// TestComponentIdentitySameFunctionValueReusesOneCachedHandle protects the
// reason the handle cache exists. A top-level (or ui.Typed-registered)
// component is the same function value at every CreateElement, so it must keep
// resolving to the one cached handle: no allocation, no renderer rebuild, no
// lock on the hot path. Fixing the collision by always allocating a fresh
// handle would pass every correctness test above and quietly tax every render.
func TestComponentIdentitySameFunctionValueReusesOneCachedHandle(parseT *testing.T) {
	parseFirst := ui.CreateElement(identityTopLevel)
	parseSecond := ui.CreateElement(identityTopLevel)

	parseFirstHandle, _ := parseFirst.Type.(*runtime.ComponentType)
	parseSecondHandle, _ := parseSecond.Type.(*runtime.ComponentType)
	if parseFirstHandle == nil || parseSecondHandle == nil {
		parseT.Fatalf("expected component handles, got %T and %T", parseFirst.Type, parseSecond.Type)
	}
	if parseFirstHandle != parseSecondHandle {
		parseT.Error("the same function value must resolve to the one cached handle; allocating per CreateElement gives up the steady-state fast path")
	}
}

// TestComponentIdentityDeliberateImplementationSwapStillApplies keeps hot
// reload working. The bug was that SetImplementationRenderer fired
// ACCIDENTALLY, between unrelated siblings; swapping an implementation on
// purpose — which is how hot reload and ui.Typed install their renderers — must
// still take effect on elements already built from that handle.
func TestComponentIdentityDeliberateImplementationSwapStillApplies(parseT *testing.T) {
	parseH := newIdentityHarness()
	parseElement := ui.CreateElement(identityTopLevel)
	parseHandle, parseOk := parseElement.Type.(*runtime.ComponentType)
	if !parseOk {
		parseT.Fatalf("expected a component handle, got %T", parseElement.Type)
	}

	parseHandle.SetImplementationRenderer(
		func(parseAttrs runtime.Attrs) ui.Node { return html.Span(html.Props{Text: "hot-reloaded"}) },
		func(parseImplementation any, parseRawProps map[string]any) *runtime.Element {
			return parseImplementation.(func(runtime.Attrs) ui.Node)(runtime.Attrs(parseRawProps))
		},
	)

	parseH.rt.RenderInto(parseH.root, parseElement)
	if parseGot := parseH.allText(); !strings.Contains(parseGot, "hot-reloaded") {
		parseT.Errorf("a deliberate SetImplementationRenderer swap must still render the new body; got %q", parseGot)
	}
}

// identityLabel wraps a captured label in a stateful inline component, so a
// test can re-render the CHILD without re-rendering the parent.
//
//go:noinline
func identityLabel(parseRt *runtime.Runtime, parseLabel string, parseBumpChild *func(any)) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseGet, parseSet := runtime.GoUseState(parseRt, "a")
		*parseBumpChild = parseSet
		return html.Span(html.Props{Text: parseLabel + "/" + parseGet()})
	})
}

// TestComponentIdentityReRenderPicksUpTheCurrentElementsClosure pins the update
// half of the fix, in internal/runtime buildUpdatedFiber: a reused fiber must
// adopt the CURRENT element's handle, not the one it mounted with.
//
// Per-closure handles alone would freeze an inline component's captures at
// mount, because the fiber would keep rendering the closure that first
// occupied its slot. This is the pre-existing behaviour being preserved: after
// the parent re-renders with a new label, the child's own next render shows the
// new label — without a shared, mutated handle making that happen.
func TestComponentIdentityReRenderPicksUpTheCurrentElementsClosure(parseT *testing.T) {
	parseH := newIdentityHarness()
	var parseBumpParent func(any)
	var parseBumpChild func(any)

	parseH.rt.RenderInto(parseH.root, ui.CreateElement(func() ui.Node {
		parseGet, parseSet := runtime.GoUseState(parseH.rt, "first")
		parseBumpParent = parseSet
		return html.Div(html.Props{}, identityLabel(parseH.rt, parseGet(), &parseBumpChild))
	}))
	if parseGot := parseH.allText(); parseGot != "first/a" {
		parseT.Fatalf("mount: got %q, want %q", parseGot, "first/a")
	}

	parseBumpParent("second")
	parseBumpChild("b")
	if parseGot := parseH.allText(); parseGot != "second/b" {
		parseT.Errorf("after the parent supplied a new closure, the child's own re-render must use it: got %q, want %q", parseGot, "second/b")
	}
}

// BenchmarkCreateElementCachedComponentHandle measures the path the handle cache
// exists for: a top-level component function, the same value at every call, must
// resolve through one map load with no allocation and no renderer rebuild.
func BenchmarkCreateElementCachedComponentHandle(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = ui.CreateElement(identityTopLevel)
	}
}

// benchClosureComponent returns a fresh closure from one literal on every call —
// the shape that used to collide, and the only shape that pays for the fix.
//
//go:noinline
func benchClosureComponent(parseName string) ui.Node {
	return ui.CreateElement(func() ui.Node {
		return html.Input(html.Props{Type: "text", Name: parseName})
	})
}

// BenchmarkCreateElementFreshClosureComponentHandle measures the changed path.
// Before the fix this rebuilt a renderer and took a mutex to overwrite one
// shared handle (and corrupted its siblings); now it allocates a handle of its
// own. Keep both numbers in view when touching getComponentHandle.
func BenchmarkCreateElementFreshClosureComponentHandle(parseB *testing.B) {
	parseB.ReportAllocs()
	parseIndex := 0
	for parseB.Loop() {
		parseIndex++
		_ = benchClosureComponent("field")
	}
}
