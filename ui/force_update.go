package ui

// UseForceUpdate returns a function that triggers a re-render of the calling
// component (G5). Use it after a mutation that does not change a subscribed
// value — an in-place edit or an external store write — instead of bumping a
// dummy "version" state by hand.
//
//	rerender := ui.UseForceUpdate()
//	... onExternalChange: rerender()
func UseForceUpdate() func() {
	parseTick := UseState(0)
	return func() { parseTick.Update(func(parseN int) int { return parseN + 1 }) }
}
