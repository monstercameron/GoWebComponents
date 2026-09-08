// Package html provides typed HTML element builders plus additive authoring
// sugar for GoWebComponents.
//
// The package is designed to pair with the ui package:
//
//	func Counter() ui.Node {
//	    count := ui.UseState(0)
//	    increment := ui.UseEvent(func() {
//	        count.Update(func(v int) int { return v + 1 })
//	    })
//
//	    return html.Div(html.Props{},
//	        html.H1(html.Props{}, html.Text("Counter")),
//	        html.Button(html.Props{OnClick: increment}, html.Text("Increment")),
//	    )
//	}
//
// html.Props keeps common DOM metadata explicit while still exposing Raw for
// escape-hatch attributes that do not need first-class fields yet. Use
// CustomElement when a browser-defined custom element needs explicit
// attribute-versus-property mapping instead of a single Raw map. Small
// convenience builders such as HiddenInput help with repetitive form markup
// without changing the underlying explicit props model. The package also
// exposes explicit markdown rendering helpers through RenderMarkdown(...)
// when applications want semantic ui.Node output without adding a second
// templating runtime.
//
// The additive sugar layer also lives here rather than in ui. The current
// supported first pass is intentionally conservative:
//
//   - existing typed builders such as Div, Button, P, Ul, and Input remain the
//     stable host-element entrypoints
//   - the companion html/shorthand package exposes mixed-argument host-tag
//     wrappers such as shorthand.Div(...) without changing the typed html.Div(...)
//     signatures
//   - Children(...) handles mixed child normalization for shorthand composition
//   - PropsOf(...), WithProps(...), plus Option helpers such as Class(...),
//     ID(...), OnClick(...), Attr(...), and Data(...) reduce repetitive literal
//     Props construction
//   - Text(...), Textf(...), TextIf(...), When(...), ClassNames(...), If(...),
//     IfElse(...), Unless(...), Map(...), MapKeyed(...), FlatMap(...),
//     FilterMap(...), Join(...), Maybe(...), OrElse(...), Coalesce(...),
//     Switch(...), Case(...), and Default(...) provide additive authoring sugar
//     without introducing JSX, hidden reactivity, or a second runtime model
//
// Children(...) currently accepts ui.Node, string, fmt.Stringer,
// func() string, nested []ui.Node, nested []string, nested []interface{}, and
// other slice or array values that recursively contain those forms. Nil values
// are skipped. Scalar fallbacks are stringified with fmt.Sprint(...).
//
// False branches from If(...), IfElse(...), Unless(...), and TextIf(...) return
// nil rather than an empty fragment. ClassNames(...) recursively flattens nested
// class-part slices and drops empty fragments. Keyed list sugar is explicit:
// MapKeyed(...) writes the computed key onto each realized node rather than
// hiding reconciliation behavior behind cosmetic naming. Optional helpers stay
// pointer-based so zero values are not overloaded as absence.
//
// The preferred happy-path import is an ordinary package import such as
// `html "github.com/monstercameron/GoWebComponents/v6/html"`. Dot-importing html
// is acceptable only in tiny example packages where collisions are tightly
// controlled. When one-call mixed props plus children is preferable, use the
// narrower companion package `github.com/monstercameron/GoWebComponents/html/shorthand`.
// Primitive DOM composition may use option-style helpers, but business
// components should continue to use explicit typed props structs.
package html
