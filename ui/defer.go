package ui

// deferLatched is the pure latch rule for a deferred view: once shown, it stays shown.
func deferLatched(parsePreviouslyShown, parseTriggered bool) bool {
	return parsePreviouslyShown || parseTriggered
}

// UseDefer latches a deferred view: it returns false until triggered first becomes true,
// then stays true for the rest of the component's lifetime (mount-once). Feed it a trigger
// derived from component state — a viewport-visibility, browser-idle, or interaction hook —
// so that when the trigger flips, the component re-renders and the deferred subtree mounts
// for good (it will not flicker back to the placeholder if the trigger later goes false).
//
//	visible := UseInViewport(ref)            // a state-backed browser trigger
//	if ui.UseDefer(visible) { /* render heavy view */ } else { /* render placeholder */ }
func UseDefer(parseTriggered bool) bool {
	parseLatch := UseRef(false)
	parseShown := deferLatched(parseLatch.Get(), parseTriggered)
	if parseShown != parseLatch.Get() {
		parseLatch.Set(parseShown)
	}
	return parseShown
}
