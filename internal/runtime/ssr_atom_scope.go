package runtime

// ---------------------------------------------------------------------------
// Per-render atom scope for server-side rendering.
//
// WHY THIS EXISTS (read before "simplifying" any of it away)
//
// An atom is deliberately process-global in the BROWSER: state.UseAtom("theme",
// "light") in two unrelated components must address ONE cell that survives every
// re-render for the life of the page. AtomRegistry.InitAtom is therefore
// init-if-absent — the first mount seeds the cell, later renders read whatever is
// there. That is the whole point of an atom and must not change.
//
// On the SERVER the same two facts combine into a correctness AND privacy defect.
// A server render:
//
//   - reaches the registry through a process-wide singleton (GetGlobalRuntime),
//   - never unmounts, so nothing ever clears what it left behind.
//
// So request 1 seeds "tenant" with ACME-CORP, and request 2 — which asked for
// HOOLI-INC — renders ACME-CORP into HOOLI-INC's HTML. One visitor's state in
// another visitor's page. The same mechanism made the registry's subscription
// map grow forever: a transient SSR fiber subscribes and has no unmount to
// unsubscribe it.
//
// THE FIX is a per-render scope, not a between-renders reset. A reset would be a
// race the first time a server renders two requests concurrently, and "correct
// because renders happen to be serialized today" is not correct. Each SSR entry
// point mints a throwaway *Runtime (atom registry included), threads it down the
// walk in the context-values map, and hangs it on every transient SSR hook fiber
// as that fiber's ownerRuntime. Atom reads and writes made during that render
// resolve to it; when the render returns, the whole scope — values AND
// subscriptions — is garbage.
//
// THE CONTRACT that follows, and that the tests pin:
//
//  1. Each SSR render is a fresh world. UseAtom's initial value wins on EVERY
//     request; it can never lose to a value another request left behind.
//  2. Writes during a render are visible to the rest of THAT render (a parent may
//     publish state a later sibling reads) and to nothing else, ever.
//  3. Explicit request state is seeded into the scope BEFORE the walk
//     (RenderToStringWithAtoms / SSRStreamOptions.InitialAtoms). A seeded id is
//     present, so init-if-absent makes the seed beat the component's initial —
//     request payload > component default > nothing.
//  4. Concurrent renders are independent by construction: no shared mutable
//     registry, so no cross-talk and nothing to reset.
//  5. Browser/wasm is untouched. newSSRAtomScope returns nil there (see
//     ssr_atom_scope_wasm.go), so a wasm RenderToString still reads the live
//     page's atoms — in a browser there is exactly one "request", and a
//     string-render is a snapshot OF that live app, not a foreign one.
//
// A boundary chunk in streaming SSR resolves on its own goroutine but inherits
// the shell's context values, hence the same scope — deliberately, since the
// boundary is part of the same request. AtomRegistry is mutex-guarded, so the
// sharing is safe.
// ---------------------------------------------------------------------------

// ssrAtomScopeContextKey is a reserved context-values key carrying the current
// SSR render's atom scope down the walk. Like ssrHookOwnerContextKey it is
// negative, so it can never collide with a context descriptor ID (those come
// from a positive counter). Keep the two constants distinct.
const ssrAtomScopeContextKey int64 = -1<<62 + 1

// newSSRAtomScope mints the per-render atom scope, seeded with parseInitialAtoms,
// or returns nil on targets where SSR must keep using the process-global registry
// (the browser). The shape mirrors GetGlobalRuntime's lazy construction: an atom
// registry and a deletions slice are all the SSR path can reach for. Everything
// else stays nil on purpose — a scope has no scheduler and no DOM adapter, so a
// setter called during a server render finds no live tree to schedule against and
// degrades to a plain registry write, which is exactly what SSR wants.
func newSSRAtomScope(parseInitialAtoms map[string]any) *Runtime {
	if !ssrAtomScopingEnabled() {
		return nil
	}
	parseScope := &Runtime{
		atomRegistry:    NewAtomRegistry(),
		deletions:       make([]*Fiber, 0),
		ssrRequestScope: true,
	}
	for parseID, parseValue := range parseInitialAtoms {
		if parseID == "" {
			continue
		}
		parseScope.atomRegistry.InitAtom(parseID, parseValue)
	}
	return parseScope
}

// ssrAtomScopeForRender returns the scope one SSR entry point should render under.
//
// A render that starts while another server render is already on this call stack
// (a component that stringifies a subtree, a helper that renders a fragment for an
// attribute) is part of the SAME request, so it INHERITS the enclosing scope
// instead of minting a second one. Minting would be the wrong answer twice: the
// nested render would lose the request's atom state, and a write it made would be
// invisible to the parent that asked for it. Seeds still apply, init-if-absent, so
// a nested call cannot clobber a value the request already holds.
//
// A top-level render finds no ambient scope and gets a fresh one — which is the
// property the whole file exists for.
func ssrAtomScopeForRender(parseInitialAtoms map[string]any) *Runtime {
	parseInherited := ssrRequestAtomScope()
	if parseInherited == nil {
		return newSSRAtomScope(parseInitialAtoms)
	}
	for parseID, parseValue := range parseInitialAtoms {
		if parseID == "" {
			continue
		}
		parseInherited.atomRegistry.InitAtom(parseID, parseValue)
	}
	return parseInherited
}

// withSSRAtomScope returns parseParent extended with parseScope. A nil scope
// returns the parent untouched so the browser path allocates nothing.
func withSSRAtomScope(parseParent map[int64]any, parseScope *Runtime) map[int64]any {
	if parseScope == nil {
		return parseParent
	}
	parseCtx := make(map[int64]any, len(parseParent)+1)
	for parseKey, parseValue := range parseParent {
		parseCtx[parseKey] = parseValue
	}
	parseCtx[ssrAtomScopeContextKey] = parseScope
	return parseCtx
}

// ssrAtomScopeFromContext reads the atom scope a walk is carrying, if any.
func ssrAtomScopeFromContext(parseCtx map[int64]any) *Runtime {
	if len(parseCtx) == 0 {
		return nil
	}
	parseScope, _ := parseCtx[ssrAtomScopeContextKey].(*Runtime)
	return parseScope
}

// ssrRequestAtomScope returns the SSR atom scope owning the render in progress,
// or nil when the caller is not inside a server render.
//
// It reads the calling render's fiber: each streaming boundary installs its
// inherited request scope on its own goroutine. withSSRHookFiber stamps it onto every
// transient SSR fiber, so the lookup is a field read in the common case; the
// parent walk is defensive against future SSR fibers that get linked to a parent.
//
// The walk STOPS at the first fiber that declares an ownerRuntime: a fiber owned
// by a real (non-SSR) runtime is a browser/reconciler render, which must keep
// using the process-global registry. Returning nil there is what keeps this fix
// scoped to SSR.
func ssrRequestAtomScope() *Runtime {
	for parseCursor := GetCurrentFiber(); parseCursor != nil; parseCursor = parseCursor.parent {
		if parseCursor.ownerRuntime == nil {
			continue
		}
		if parseCursor.ownerRuntime.ssrRequestScope {
			return parseCursor.ownerRuntime
		}
		return nil
	}
	return nil
}
