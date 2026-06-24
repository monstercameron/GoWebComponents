//go:build !js || !wasm

package router

// UseRoute is a non-reactive empty snapshot outside the browser runtime, where
// there is no live navigation to subscribe to. The reactive implementation lives
// in the js/wasm build.
func UseRoute() Location {
	return Location{}
}

// UseLocation is an alias for UseRoute (native stub).
func UseLocation() Location {
	return Location{}
}

// OnNavigate registers a navigation callback. On non-browser builds there is no
// live navigation, so it stores nothing and returns a no-op unsubscribe.
func OnNavigate(parseFn func(Location)) func() {
	return func() {}
}

// Href returns the path unchanged on non-browser builds (no active router mode to
// resolve against). The mode-aware implementation lives in the js/wasm build.
func Href(parsePath string) string {
	return parsePath
}

// FragmentHref returns "#<fragment>" on non-browser builds.
func FragmentHref(parseFragment string) string {
	if len(parseFragment) > 0 && parseFragment[0] == '#' {
		return parseFragment
	}
	return "#" + parseFragment
}
