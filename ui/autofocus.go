package ui

// UseAutoFocus moves keyboard focus to a referenced element after it mounts (G22).
//
// The HTML autofocus attribute (html.AutoFocus) only fires on the initial page
// load, not when an element is mounted later in a single-page app — so inline
// editors, dialogs, and revealed fields never receive focus from it. UseAutoFocus
// closes that gap without the focusByID/getElementById workaround: pair it with a
// DOM ref and the element is focused once it is live.
//
//	r := ui.UseDOMRef()
//	ui.UseAutoFocus(r)                    // focus once, on mount
//	return shorthand.Input(shorthand.Ref(r))
//
// Pass deps to re-focus when they change (focus-follows-reveal):
//
//	ui.UseAutoFocus(r, isEditing)         // focus each time isEditing flips true-with-mount
//
// With no deps it runs exactly once on mount (it must not run every render, or it
// would steal focus continuously). Focusing is a no-op before mount, after
// unmount, and on the native/SSR build.
func UseAutoFocus(parseRef DOMRef, parseDeps ...any) {
	if len(parseDeps) == 0 {
		// A stable, non-empty dep → run once on mount. An empty variadic would
		// mean "every render" in UseEffect, which is wrong for focus.
		parseDeps = autoFocusOnceDeps
	}
	UseEffect(func() func() {
		parseRef.Focus()
		return nil
	}, parseDeps...)
}

// autoFocusOnceDeps is the shared stable sentinel dep for the once-on-mount case.
var autoFocusOnceDeps = []any{"gwc-autofocus-once"}
