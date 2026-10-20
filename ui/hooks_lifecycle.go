package ui

import "github.com/monstercameron/GoWebComponents/v7/internal/runtime"

// Always is the dependency sentinel that says "run on every render". Passing it
// as the only dependency documents intent that omitting the dependency list
// leaves ambiguous:
//
//	ui.UseEffect(syncTitle, ui.Always) // re-runs after every render
//
// UseEffect(fn, ui.Always) behaves exactly like UseEffect(fn); UseMemo(fn,
// ui.Always) recomputes on every render; UseCallback(fn, ui.Always) stores the
// newest function on every render. It is recognized only as the single
// dependency: in a longer list it is an ordinary constant and changes nothing.
// Dev builds never raise GWC-RUNTIME-HOOK-DEP-UNSTABLE for an Always hook.
var Always = runtime.AlwaysDependency

// UseMount runs fn once after the component first mounts and runs the returned
// cleanup (if any) on unmount (G38). It is the self-documenting form of the
// "run once" effect — clearer and safer than the UseEffect(fn, true) constant-dep
// idiom, which silently changes behavior if dep-equality is ever refactored.
//
//	ui.UseMount(func() func() {
//	    sub := subscribe()
//	    return func() { sub.Close() }   // cleanup on unmount
//	})
//
// On the native/SSR build effects do not run, so UseMount is a no-op there
// (matching UseEffect).
func UseMount(parseFn func() func()) {
	UseEffect(parseFn, useMountSentinel)
}

// useMountSentinel is a stable, non-empty dependency so the effect runs exactly
// once on mount. (A zero-length dep list runs every render.)
const useMountSentinel = "gwc-use-mount"

// UseMediaQuery reports whether the given CSS media query currently matches, and
// re-renders the component when it changes (G20). It reflects the query at mount,
// updates live via the MediaQueryList change event, and releases its listener on
// unmount. On builds without media-query support it returns false.
//
//	wide := ui.UseMediaQuery("(min-width: 768px)")
func UseMediaQuery(parseQuery string) bool {
	return useMediaQueryMatch(parseQuery)
}
