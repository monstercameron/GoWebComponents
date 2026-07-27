//go:build !js || !wasm

package runtime

// ssrAtomScopingEnabled reports that server renders on this target get their own
// per-render atom scope.
//
// True off-browser, where RenderToString/RenderToStream ARE the request path: a
// process serves many visitors, so sharing one atom registry across renders leaks
// one visitor's state into another's HTML. See ssr_atom_scope.go for the full
// rationale, and ssr_atom_scope_wasm.go for why the browser answers false.
func ssrAtomScopingEnabled() bool { return true }

// RenderToStringWithAtoms renders parseElement with parseInitialAtoms seeded into
// this render's atom scope before the walk begins.
//
// This is the explicit form of the SSR contract: a request's state is INPUT to
// the render, not something left over from the last one. Because seeding happens
// first and InitAtom is init-if-absent, precedence is
//
//	seeded request value  >  the component's UseAtom initial  >  nothing
//
// It is the supported replacement for "write the global registry, then render",
// which cannot work once each render has its own scope (and was never safe under
// concurrency anyway). Native-only on purpose: seeding a request has no meaning
// in a browser, where there is one page and one set of live atoms.
func RenderToStringWithAtoms(parseElement *Element, parseInitialAtoms map[string]any) (string, error) {
	return renderToStringScoped(parseElement, parseInitialAtoms)
}
