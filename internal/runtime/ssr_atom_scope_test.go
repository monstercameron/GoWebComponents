//go:build !js || !wasm

package runtime

import (
	"fmt"
	"testing"
)

// These sit inside the package because the properties worth pinning are about
// the REGISTRY, not the markup: which registry a server render writes into, and
// what it leaves behind. ui/ssr_atom_scope_test.go covers the observable markup
// contract. See ssr_atom_scope.go for the rationale both files defend.

// TestSSRRenderDoesNotTouchTheGlobalAtomRegistry is the leak test stated in terms
// of the defect's mechanism.
//
// Before the per-render scope, each RenderToString call (1) seeded the global
// registry, so the NEXT render's initial value was ignored, and (2) subscribed its
// transient SSR fiber and never unsubscribed, because a server render has no
// unmount to clean up after it. Property (2) is an unbounded memory leak in any
// long-running SSR process: the subscription map grew by one dead fiber per render
// per atom, forever.
//
// Both are the same assertion now: after N server renders the process-global
// registry must be exactly as empty as it was before them.
func TestSSRRenderDoesNotTouchTheGlobalAtomRegistry(parseT *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()
	defer SetCurrentFiber(nil)

	parseGlobal := GetGlobalRuntime()
	parseComponent := func(parseInitial string) *Element {
		return CreateElement(func() *Element {
			parseGet, _ := GoUseAtomGlobal("ssr-registry:probe", parseInitial)
			return CreateElement("span", map[string]any{}, parseGet())
		}, map[string]any{})
	}

	const parseRenders = 25
	for parseIndex := 0; parseIndex < parseRenders; parseIndex++ {
		parseWant := fmt.Sprintf("value-%d", parseIndex)
		parseMarkup, parseErr := RenderToString(parseComponent(parseWant))
		if parseErr != nil {
			parseT.Fatalf("render %d: %v", parseIndex, parseErr)
		}
		if parseMarkup != "<span>"+parseWant+"</span>" {
			parseT.Fatalf("render %d = %q, want <span>%s</span>", parseIndex, parseMarkup, parseWant)
		}
	}

	parseGlobal.atomRegistry.mu.RLock()
	parseAtomCount := len(parseGlobal.atomRegistry.atoms)
	parseSubscriptionCount := 0
	for _, parseSet := range parseGlobal.atomRegistry.subscriptions {
		parseSubscriptionCount += len(parseSet)
	}
	parseGlobal.atomRegistry.mu.RUnlock()

	if parseAtomCount != 0 {
		parseT.Errorf("global registry holds %d atom(s) after %d server renders; request state must not outlive its render", parseAtomCount, parseRenders)
	}
	if parseSubscriptionCount != 0 {
		parseT.Errorf("global registry holds %d subscription(s) after %d server renders; SSR fibers are never unmounted, so they must never be registered globally", parseSubscriptionCount, parseRenders)
	}
}

// TestSSRRenderScopesAreDistinctAndSelfContained proves the subscription leak is
// fixed by construction rather than by a cleanup pass that could be forgotten:
// each render gets its OWN registry, which holds only that render's subscriptions
// and becomes garbage when the render returns.
func TestSSRRenderScopesAreDistinctAndSelfContained(parseT *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()
	defer SetCurrentFiber(nil)

	var parseScopes []*Runtime
	parseComponent := CreateElement(func() *Element {
		parseScope := ssrRequestAtomScope()
		if parseScope == nil {
			parseT.Error("a component rendering under RenderToString saw no SSR atom scope")
			return CreateElement("span", map[string]any{}, "no-scope")
		}
		parseScopes = append(parseScopes, parseScope)
		parseGet, _ := GoUseAtomGlobal("ssr-scope:identity", "x")
		return CreateElement("span", map[string]any{}, parseGet())
	}, map[string]any{})

	for parseIndex := 0; parseIndex < 3; parseIndex++ {
		if _, parseErr := RenderToString(parseComponent); parseErr != nil {
			parseT.Fatalf("render %d: %v", parseIndex, parseErr)
		}
	}
	if len(parseScopes) != 3 {
		parseT.Fatalf("captured %d scopes, want 3", len(parseScopes))
	}

	for parseIndex, parseScope := range parseScopes {
		if parseScope == GetGlobalRuntime() {
			parseT.Fatalf("scope %d IS the global runtime; the SSR render was not scoped", parseIndex)
		}
		if !parseScope.ssrRequestScope {
			parseT.Fatalf("scope %d is not marked as an SSR request scope", parseIndex)
		}
		parseScope.atomRegistry.mu.RLock()
		parseSubs := 0
		for _, parseSet := range parseScope.atomRegistry.subscriptions {
			parseSubs += len(parseSet)
		}
		parseScope.atomRegistry.mu.RUnlock()
		// Exactly one: the single component of this render, and nothing carried
		// over from the renders before it.
		if parseSubs != 1 {
			parseT.Errorf("scope %d holds %d subscriptions, want exactly 1 (its own render's component)", parseIndex, parseSubs)
		}
	}
	if parseScopes[0] == parseScopes[1] || parseScopes[1] == parseScopes[2] || parseScopes[0] == parseScopes[2] {
		parseT.Error("two server renders shared one atom scope; scopes must be per render")
	}
}

// TestNestedSSRRenderInheritsTheRequestScope pins the other half of the scoping
// rule. A component that stringifies a subtree mid-render is still serving the
// SAME request, so the inner render must join the outer scope instead of minting
// a second one — otherwise the nested markup would silently lose the request's
// atom state, and a value it published would be invisible to its caller.
func TestNestedSSRRenderInheritsTheRequestScope(parseT *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()
	defer SetCurrentFiber(nil)

	parseInner := CreateElement(func() *Element {
		parseGet, _ := GoUseAtomGlobal("ssr-nested:flag", "inner-default")
		return CreateElement("em", map[string]any{}, parseGet())
	}, map[string]any{})

	parseOuter := CreateElement(func() *Element {
		// The outer component publishes request state, then renders a subtree to
		// string. The nested render must observe the publication.
		_, parseSet := GoUseAtomGlobal("ssr-nested:flag", "outer-value")
		parseSet("published-by-outer")
		parseNestedMarkup, parseErr := RenderToString(parseInner)
		if parseErr != nil {
			parseT.Fatalf("nested RenderToString: %v", parseErr)
		}
		return CreateElement("span", map[string]any{}, parseNestedMarkup)
	}, map[string]any{})

	parseMarkup, parseErr := RenderToString(parseOuter)
	if parseErr != nil {
		parseT.Fatalf("outer RenderToString: %v", parseErr)
	}
	if parseMarkup != "<span>&lt;em&gt;published-by-outer&lt;/em&gt;</span>" {
		parseT.Errorf("nested render = %q; a nested server render must inherit the enclosing request's atom scope", parseMarkup)
	}

	// And the inherited scope still dies with the request: a later top-level
	// render starts from its own initial value.
	parseFresh, parseErr2 := RenderToString(parseInner)
	if parseErr2 != nil {
		parseT.Fatalf("follow-up RenderToString: %v", parseErr2)
	}
	if parseFresh != "<em>inner-default</em>" {
		parseT.Errorf("follow-up render = %q, want <em>inner-default</em> — the previous request's scope outlived it", parseFresh)
	}
}

// TestSSRRequestAtomScopeIsNilOutsideServerRendering pins the boundary that keeps
// this fix out of the browser/reconciler path: only a fiber owned by an SSR scope
// resolves to one. A fiber owned by a real runtime, or owned by nothing, keeps
// using the process-global registry — which is what makes an atom an atom.
func TestSSRRequestAtomScopeIsNilOutsideServerRendering(parseT *testing.T) {
	defer SetCurrentFiber(nil)

	SetCurrentFiber(nil)
	if ssrRequestAtomScope() != nil {
		parseT.Error("no current fiber must not resolve to an SSR scope")
	}

	SetCurrentFiber(&Fiber{typeOf: "component"})
	if ssrRequestAtomScope() != nil {
		parseT.Error("an unowned fiber (unit test, ad-hoc render) must not resolve to an SSR scope")
	}

	parseLive := NewRuntime(Config{DOMAdapter: newTestDOMAdapter()})
	SetCurrentFiber(&Fiber{typeOf: "component", ownerRuntime: parseLive})
	if ssrRequestAtomScope() != nil {
		parseT.Error("a fiber owned by a live runtime must keep using the global atom registry")
	}
}
