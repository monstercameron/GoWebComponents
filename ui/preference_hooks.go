package ui

import "github.com/monstercameron/GoWebComponents/interop"

// ColorScheme is the user's preferred color scheme as reported by the
// prefers-color-scheme media feature.
type ColorScheme string

const (
	// ColorSchemeLight is the default when no dark preference is expressed or
	// when media queries are unavailable (for example native/SSR builds).
	ColorSchemeLight ColorScheme = "light"
	// ColorSchemeDark indicates the user prefers a dark color scheme.
	ColorSchemeDark ColorScheme = "dark"
)

const (
	prefersReducedMotionQuery = "(prefers-reduced-motion: reduce)"
	prefersDarkSchemeQuery    = "(prefers-color-scheme: dark)"
)

// UsePrefersReducedMotion reports whether the user has requested reduced motion.
// It reflects the media query at mount and updates live when the emulated or OS
// preference flips, unsubscribing on unmount (the underlying js.Func handle is
// released). On builds without media-query support it returns false.
func UsePrefersReducedMotion() bool {
	return useMediaQueryMatch(prefersReducedMotionQuery)
}

// UsePrefersColorScheme reports the user's preferred color scheme, defaulting to
// ColorSchemeLight. Like UsePrefersReducedMotion it reflects the value at mount,
// updates live on preference changes, and cleans up its listener on unmount.
func UsePrefersColorScheme() ColorScheme {
	if useMediaQueryMatch(prefersDarkSchemeQuery) {
		return ColorSchemeDark
	}
	return ColorSchemeLight
}

// useMediaQueryMatch binds component state to a CSS media query's match value.
func useMediaQueryMatch(parseQuery string) bool {
	parseState := UseState(currentMediaMatch(parseQuery))
	UseEffect(func() func() {
		parseList, parseErr := interop.GetMediaQuery(parseQuery)
		if parseErr != nil {
			return func() {}
		}
		// Re-read after mount in case the preference changed between the initial
		// synchronous read and the effect running.
		parseState.Set(parseList.Matches())
		parseSubscription, parseSubErr := parseList.Subscribe(func(parseEvent interop.MediaQueryEvent) {
			parseState.Set(parseEvent.Matches)
		})
		if parseSubErr != nil {
			return func() {}
		}
		return func() { parseSubscription.Cancel() }
	}, parseQuery)
	return parseState.Get()
}

// currentMediaMatch reads a media query's current match, returning false when
// media queries are unavailable (native/SSR) so callers get a stable default.
func currentMediaMatch(parseQuery string) bool {
	parseList, parseErr := interop.GetMediaQuery(parseQuery)
	if parseErr != nil {
		return false
	}
	return parseList.Matches()
}
