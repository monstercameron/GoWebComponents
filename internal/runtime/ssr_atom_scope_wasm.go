//go:build js && wasm

package runtime

// ssrAtomScopingEnabled reports that the browser keeps SSR renders on the
// process-global atom registry.
//
// In the browser there is exactly one "request": the page. RenderToString here is
// a snapshot of the LIVE app (markup export, tests, devtools), so it should see
// the live atoms — and the atom's whole contract in the browser is that it is
// global and outlives every render. Scoping it per render would break that and
// would fix nothing: a page cannot leak state into another visitor's page.
//
// Do NOT "unify" this with the native file to remove a build-tag split. The split
// IS the fix: it is what keeps a server-only correctness rule out of the browser
// semantics. See ssr_atom_scope.go.
func ssrAtomScopingEnabled() bool { return false }
