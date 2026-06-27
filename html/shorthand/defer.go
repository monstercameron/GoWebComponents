package shorthand

import "github.com/monstercameron/GoWebComponents/ui"

// Defer renders placeholder until shown is true, then builds and renders content. content is
// a thunk invoked ONLY when shown — so the deferred (often expensive) subtree is never
// constructed until it is actually needed. This is the rendering half of deferrable views
// (Angular `@defer` / React lazy): pair shown with a viewport/idle/interaction trigger
// latched via ui.UseDefer to keep heavy off-screen views out of the render path until they
// matter.
//
//	shorthand.Defer(ui.UseDefer(visible), Spinner(), func() ui.Node { return HeavyChart(data) })
func Defer(shown bool, placeholder ui.Node, content func() ui.Node) ui.Node {
	if shown && content != nil {
		return content()
	}
	return placeholder
}
