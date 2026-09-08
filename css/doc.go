// Package css authoring guide.
//
// # Layers
//
// Layer 1 — typed raw CSS (the foundation). Typed properties and values fold into
// a hashed class:
//
//	class := css.New(
//	    css.Display.Flex,
//	    css.Gap(css.Px(8)),
//	    css.Bg(css.Slate900),
//	    css.Hover(css.Bg(css.Slate800)),          // pseudo variant
//	    css.Media(css.MinW(768), css.FlexDir.Row), // at-rule variant
//	)
//	// class is "c-…"; the compiled CSS is emitted once through the active Sink.
//
// Layer 2 — Tailwind-shaped utilities (subpackage css/u), built on Layer 1. Every
// utility resolves against the active css.Theme and returns Layer-1 rules:
//
//	import "github.com/monstercameron/GoWebComponents/v6/css/u"
//	css.New(u.Flex, u.Gap(3), u.Bg(u.Slate900), u.Md(u.Hover(u.FlexRow)...)...)
//
// # Crossing into html
//
// Two entry points cross into the html package with zero edits to it:
//
//	// css.Class is an html.PropOption — drop it into any shorthand call site.
//	shorthand.Div(css.Class(css.Display.Flex, css.Gap(css.Px(8))), …)
//
//	// css.New returns a Sheet (fmt.Stringer) that flows through html.ClassNames,
//	// ClassMap keys, SSR, and attribute strings.
//	sheet := css.New(css.Display.Grid)
//
// # Dynamic values
//
// Runtime values would mint a new class per distinct value. Use Dynamic to keep a
// stable class that references a custom property and set the live value inline:
//
//	d := css.DynamicLength("--gap", "gap", state.Gap())
//	shorthand.Div(d.Class(), d.Style(), …)
//
// # SSR & hydration
//
// On native (SSR) the buffer sink collects every emitted rule. Serialize it into
// the head with css.StyleBlock(), which renders <style data-gwc-css="…">. On wasm
// hydration, css.SeedFromDocument() pre-seeds the registry from that block so
// already-present rules are recognized as hits and not re-injected.
//
// # Extension points
//
//   - css.DefineUtility(name, rules…) — named reusable bundles (the plugin path).
//   - css.UseTheme(theme) — swap/extend the token scales the utility engine reads.
//   - css.DefineVariant(selectorTemplate) — new variant selectors via an "&" template.
//   - css.SetSink(sink) — custom emission targets without touching authoring.
package css
